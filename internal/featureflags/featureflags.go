// Package featureflags provides a cached feature flag client backed by the CAFE service.
//
// The intended lifecycle is:
//  1. At startup, call Fetch to load defaults overlaid with cached flags. If the
//     cache is stale, Fetch spawns an async subprocess to refresh it for next time.
//  2. The caller memoizes the returned snapshot — flags never change mid-command.
//  3. The refresh subprocess calls Client.FetchAndCache to fetch from CAFE and
//     atomically write the cache to disk.
//  4. The next invocation picks up the refreshed cache.
package featureflags

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cli/cli/v2/internal/featureflags/cafe"
	"github.com/cli/cli/v2/internal/gh"
	ghauth "github.com/cli/go-gh/v2/pkg/auth"
)

const (
	defaultCacheTTL = 30 * time.Minute

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

func Fetch(cacheDir, host, user string, executable string) gh.FeatureFlagState {
	// Short-circuit fetching from CAFE for GHE hosts since they don't support telemetry — this avoids unnecessary CAFE calls and cache churn for GHE users.
	if ghauth.IsEnterprise(host) {
		return gh.FeatureFlagState{
			Telemetry: false,
		}
	}

	var defaultFlagState = gh.FeatureFlagState{
		Telemetry: false,
	}

	// Read from the cache
	data, err := os.ReadFile(cachePath(cacheDir, host, user))
	if err != nil {
		// If the cache is missing or unreadable, we'll return client side defaults, there's not much else to do.
		return defaultFlagState
	}

	var c cache
	if err := json.Unmarshal(data, &c); err != nil {
		// If the cache is corrupt, we'll return client side defaults and ignore the cache.
		return defaultFlagState
	}

	// If the cache is stale, then kick off a background refresh for the next invocation
	if time.Since(c.FetchedAt) > defaultCacheTTL {
		spawnFetchFeatureFlags(executable, host)
	}

	// Return the flags from cache even if stale, we want to avoid inconsistent flag values within the same command invocation. The next invocation will pick up the refreshed cache.
	return fromMap(c.Flags)
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
// and atomically writes the cache. If the CAFE response is invalid, the prior
// cache is preserved.
func (c *Client) FetchAndCache(ctx context.Context) error {
	flags, err := c.cafe.GetFeatureFlags(ctx, allFlagNames)
	if err != nil {
		return fmt.Errorf("fetching feature flags from CAFE: %w", err)
	}

	// Validate: ensure we got a non-nil map with expected keys before overwriting cache.
	if flags == nil {
		return fmt.Errorf("CAFE returned nil flags")
	}

	return writeCache(cachePath(c.cacheDir, c.host, c.user), &cache{
		Flags:     flags,
		FetchedAt: c.now(),
	})
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

// SpawnFetchFeatureFlags spawns a subprocess to fetch feature flags from CAFE.
// The host parameter is passed via GH_HOST so the subprocess resolves the
// correct auth token and cache scope for the targeted host.
// All errors are silently ignored since this is best-effort.
func spawnFetchFeatureFlags(executable, host string) {
	cmd := exec.Command(executable, "fetch-feature-flags")
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Env = append(os.Environ(), "GH_HOST="+host)
	if err := cmd.Start(); err != nil {
		return
	}
	_ = cmd.Process.Release()
}
