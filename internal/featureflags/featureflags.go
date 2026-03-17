// Package featureflags provides a cached feature flag client backed by the CAFE service.
//
// The intended lifecycle is:
//  1. At startup, call Fetch to load defaults overlaid with cached flags. If the
//     cache is missing, stale, or broken, Fetch spawns a subprocess to refresh it
//     and blocks for up to fetchTimeout waiting for the result.
//  2. The caller memoizes the returned snapshot — flags never change mid-command.
//  3. The refresh subprocess calls Client.FetchAndCache to fetch from CAFE,
//     atomically write the cache to disk, and print the flags to stdout.
//  4. If the subprocess completes within the timeout, Fetch uses the stdout result
//     directly. Otherwise it falls back to stale cached values or defaults.
//
// A lock file prevents concurrent fetch subprocesses when gh is invoked in a loop.
package featureflags

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cli/cli/v2/internal/featureflags/cafe"
	"github.com/cli/cli/v2/internal/gh"
	ghauth "github.com/cli/go-gh/v2/pkg/auth"
)

const (
	defaultCacheTTL = 30 * time.Minute
	softRefreshTTL  = defaultCacheTTL - 5*time.Minute // start background refresh before cache expires
	fetchTimeout    = 2 * time.Second
	lockMaxAge      = 30 * time.Second

	flagTelemetry = "gh_cli_telemetry"
)

// allFlagNames is the list of all flag names we request from CAFE.
var allFlagNames = []string{flagTelemetry}

// cache represents the on-disk feature flag cache.
type cache struct {
	Flags     map[string]bool `json:"flags"`
	FetchedAt time.Time       `json:"fetched_at"`
}

func cachePath(cacheDir, host, user string) string {
	return filepath.Join(cacheDir, host+"-"+user+"-feature-flags.json")
}

func fromMap(flags map[string]bool) gh.FeatureFlagState {
	return gh.FeatureFlagState{
		Telemetry: flags[flagTelemetry],
	}
}

// FromCache reads the feature flag cache for the given scope and returns the
// flag state. Returns defaults if the cache is missing or unreadable.
func FromCache(cacheDir, host, user string) gh.FeatureFlagState {
	c, err := readCache(cachePath(cacheDir, host, user))
	if err != nil {
		return gh.FeatureFlagState{}
	}
	return fromMap(c.Flags)
}

func Fetch(cacheDir, host, user string, executable string) gh.FeatureFlagState {
	// Short-circuit fetching from CAFE for GHE hosts since they don't support telemetry,
	// this avoids unnecessary CAFE calls and cache churn for GHE users.
	if ghauth.IsEnterprise(host) {
		return gh.FeatureFlagState{
			Telemetry: false,
		}
	}

	var fallback gh.FeatureFlagState

	c, err := readCache(cachePath(cacheDir, host, user))
	// If there was no error and we have a cache value
	if err == nil {
		age := time.Since(c.FetchedAt)
		switch {
		case age <= softRefreshTTL:
			// Cache is fresh, return it immediately.
			return fromMap(c.Flags)
		case age <= defaultCacheTTL:
			// Cache is nearing expiry — return it immediately but kick off a
			// background (non-blocking) refresh so the next invocation gets
			// fresh values without having to block.
			if !IsLocked(cacheDir, host, user) {
				go runFetchSubprocess(executable, host)
			}
			return fromMap(c.Flags)
		default:
			// Cache is stale, use it as fallback while we do a blocking fetch.
			fallback = fromMap(c.Flags)
		}
	} else {
		// Cache is missing or unreadable, use defaults as fallback while we fetch new values
		fallback = gh.FeatureFlagState{
			Telemetry: false,
		}
	}

	// If another process is already fetching, don't spawn a second one.
	if IsLocked(cacheDir, host, user) {
		return fallback
	}

	// Spawn the fetch subprocess and block up to fetchTimeout for the result.
	// The subprocess creates and removes its own lock file.
	type result struct {
		flags gh.FeatureFlagState
		ok    bool
	}
	done := make(chan result, 1)
	go func() {
		flags, ok := runFetchSubprocess(executable, host)
		done <- result{flags, ok}
	}()

	select {
	case r := <-done:
		if r.ok {
			return r.flags
		}
	case <-time.After(fetchTimeout):
		// Timeout: the subprocess continues in the background and will
		// write the cache for the next invocation.
	}

	return fallback
}

func readCache(path string) (cache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cache{}, err
	}

	var c cache
	if err := json.Unmarshal(data, &c); err != nil {
		return cache{}, err
	}

	return c, nil
}

// Client fetches feature flags from the CAFE service and writes them to the disk cache.
// Used by the fetch-feature-flags subprocess.
type Client struct {
	cafe     *cafe.Client
	cacheDir string
	host     string
	user     string
	now      func() time.Time
}

// NewClient creates a feature flag client for fetching and caching flags.
func NewClient(cafeClient *cafe.Client, cacheDir, host, user string) *Client {
	return &Client{
		cafe:     cafeClient,
		cacheDir: cacheDir,
		host:     host,
		user:     user,
		now:      time.Now,
	}
}

// FetchAndCache fetches all feature flags from CAFE, validates the response,
// atomically writes the cache, and returns the flag state. If the CAFE response
// is invalid, the prior cache is preserved.
func (c *Client) FetchAndCache(ctx context.Context) (gh.FeatureFlagState, error) {
	flags, err := c.cafe.GetFeatureFlags(ctx, allFlagNames)
	if err != nil {
		return gh.FeatureFlagState{}, fmt.Errorf("fetching feature flags from CAFE: %w", err)
	}

	// Validate: ensure we got a non-nil map with all expected keys before overwriting cache.
	if flags == nil {
		return gh.FeatureFlagState{}, fmt.Errorf("CAFE returned nil flags")
	}
	for _, name := range allFlagNames {
		if _, ok := flags[name]; !ok {
			return gh.FeatureFlagState{}, fmt.Errorf("CAFE response missing expected flag: %s", name)
		}
	}

	if err := writeCache(cachePath(c.cacheDir, c.host, c.user), &cache{
		Flags:     flags,
		FetchedAt: c.now(),
	}); err != nil {
		return gh.FeatureFlagState{}, err
	}

	return fromMap(flags), nil
}

// writeCache atomically writes the cache to disk using a temp file + rename.
func writeCache(path string, c *cache) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".feature-flags-*.json.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// runFetchSubprocess runs a fetch-feature-flags subprocess synchronously,
// waits for it to complete, and returns the parsed flags from its stdout.
// Returns (flags, true) on success or (zero, false) on any failure.
func runFetchSubprocess(executable, host string) (gh.FeatureFlagState, bool) {
	var stdout bytes.Buffer
	cmd := exec.Command(executable, "fetch-feature-flags", "--hostname", host)
	cmd.Stdin = nil
	cmd.Stdout = &stdout
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return gh.FeatureFlagState{}, false
	}
	var flags gh.FeatureFlagState
	if err := json.Unmarshal(stdout.Bytes(), &flags); err != nil {
		return gh.FeatureFlagState{}, false
	}
	return flags, true
}

// Lock file helpers — prevent concurrent fetch subprocesses when gh is invoked in a loop.

func lockPath(cacheDir, host, user string) string {
	return filepath.Join(cacheDir, host+"-"+user+"-feature-flags.lock")
}

// IsLocked reports whether a non-stale lock file exists, indicating another
// process is currently fetching feature flags.
func IsLocked(cacheDir, host, user string) bool {
	info, err := os.Stat(lockPath(cacheDir, host, user))
	if err != nil {
		return false
	}
	// Consider the lock stale if it is older than lockMaxAge (handles crashed processes).
	return time.Since(info.ModTime()) < lockMaxAge
}

// CreateLockFile creates a lock file to signal that a fetch is in progress.
// The caller should defer RemoveLockFile.
func CreateLockFile(cacheDir, host, user string) error {
	p := lockPath(cacheDir, host, user)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	// Write our PID so the lock can be inspected for debugging.
	return os.WriteFile(p, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

// RemoveLockFile removes the lock file created by CreateLockFile.
func RemoveLockFile(cacheDir, host, user string) {
	_ = os.Remove(lockPath(cacheDir, host, user))
}
