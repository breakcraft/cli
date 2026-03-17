package featureflags

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cli/cli/v2/internal/featureflags/cafe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeViewerAPI implements cafe.ViewerAPI for testing.
type fakeViewerAPI struct {
	flags      []*cafe.FeatureFlag
	stubbedErr error
}

func (f *fakeViewerAPI) GetDetails(context.Context, *cafe.GetDetailsRequest) (*cafe.GetDetailsResponse, error) {
	return nil, nil
}

func (f *fakeViewerAPI) GetFeatureFlags(_ context.Context, _ *cafe.GetFeatureFlagsRequest) (*cafe.GetFeatureFlagsResponse, error) {
	if f.stubbedErr != nil {
		return nil, f.stubbedErr
	}
	return &cafe.GetFeatureFlagsResponse{FeatureFlags: f.flags}, nil
}

func newTestServer(t *testing.T, flags map[string]bool) *httptest.Server {
	t.Helper()
	var cafeFlags []*cafe.FeatureFlag
	for name, enabled := range flags {
		cafeFlags = append(cafeFlags, &cafe.FeatureFlag{Name: name, IsEnabled: enabled})
	}
	handler := cafe.NewViewerAPIServer(&fakeViewerAPI{flags: cafeFlags})
	mux := http.NewServeMux()
	mux.Handle(handler.PathPrefix(), handler)
	return httptest.NewServer(mux)
}

func writeTestCache(t *testing.T, cacheDir, host, user string, c *cache) {
	t.Helper()
	data, err := json.Marshal(c)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, host+"-"+user+"-feature-flags.json"), data, 0o600))
}

// TODO: ensure we have test coverage of when background fetch occurs or not on .Fetch.

func TestFetch_freshCache(t *testing.T) {
	// Given a cache with telemetry enabled that was fetched recently
	cacheDir := t.TempDir()
	writeTestCache(t, cacheDir, "github.com", "testuser", &cache{
		Flags:     map[string]bool{"gh_cli_telemetry": true},
		FetchedAt: time.Now().Add(-10 * time.Minute),
	})

	// When I fetch feature flags
	flags := Fetch(cacheDir, "github.com", "testuser", "gh")

	// Then telemetry should be enabled from the cached value
	assert.True(t, flags.Telemetry)
}

func TestFetch_noCache(t *testing.T) {
	// Given no cache file exists
	cacheDir := t.TempDir()

	// When I fetch feature flags
	flags := Fetch(cacheDir, "github.com", "testuser", "gh")

	// Then it should return defaults (all flags disabled)
	assert.False(t, flags.Telemetry)
}

func TestFetch_corruptCache(t *testing.T) {
	// Given a corrupt cache file
	cacheDir := t.TempDir()
	path := filepath.Join(cacheDir, "github.com-testuser-feature-flags.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	// When I fetch feature flags
	flags := Fetch(cacheDir, "github.com", "testuser", "gh")

	// Then it should return defaults (all flags disabled)
	assert.False(t, flags.Telemetry)
}

func TestFetch_enterpriseHost(t *testing.T) {
	// Given any state (even a cache with telemetry enabled)
	cacheDir := t.TempDir()
	writeTestCache(t, cacheDir, "ghes.example.com", "testuser", &cache{
		Flags:     map[string]bool{"gh_cli_telemetry": true},
		FetchedAt: time.Now(),
	})

	// When I fetch feature flags for an enterprise host
	flags := Fetch(cacheDir, "ghes.example.com", "testuser", "gh")

	// Then telemetry should be disabled (enterprise hosts short-circuit)
	assert.False(t, flags.Telemetry)
}

func TestFetch_staleCacheReturnsExistingFlags(t *testing.T) {
	// Given a stale cache with telemetry enabled
	cacheDir := t.TempDir()
	writeTestCache(t, cacheDir, "github.com", "testuser", &cache{
		Flags:     map[string]bool{"gh_cli_telemetry": true},
		FetchedAt: time.Now().Add(-31 * time.Minute),
	})

	// When I fetch feature flags
	flags := Fetch(cacheDir, "github.com", "testuser", "gh")

	// Then it should return the stale cached value as fallback
	// (the blocking fetch fails because "gh" isn't a real executable here)
	assert.True(t, flags.Telemetry)
}

func TestFetch_softRefreshWindowReturnsCacheImmediately(t *testing.T) {
	// Given a cache in the soft refresh window (between softRefreshTTL and defaultCacheTTL)
	cacheDir := t.TempDir()
	writeTestCache(t, cacheDir, "github.com", "testuser", &cache{
		Flags:     map[string]bool{"gh_cli_telemetry": true},
		FetchedAt: time.Now().Add(-27 * time.Minute), // between 25 and 30 min
	})

	// When I fetch feature flags (using a non-existent executable to prove it doesn't block)
	start := time.Now()
	flags := Fetch(cacheDir, "github.com", "testuser", "this-executable-does-not-exist")
	elapsed := time.Since(start)

	// Then it should return cached flags immediately without blocking
	assert.True(t, flags.Telemetry)
	assert.Less(t, elapsed, 500*time.Millisecond, "soft refresh should not block")
}

func TestFetchAndCache_success(t *testing.T) {
	// Given a CAFE server returning the telemetry flag as enabled
	cacheDir := t.TempDir()
	server := newTestServer(t, map[string]bool{"gh_cli_telemetry": true})
	t.Cleanup(server.Close)

	cafeClient := cafe.NewClient(server.Client(), cafe.WithBaseURL(server.URL))
	client := NewClient(cafeClient, cacheDir, "github.com", "testuser")

	// When I fetch and cache
	flags, err := client.FetchAndCache(context.Background())

	// Then it should succeed
	require.NoError(t, err)

	// And the returned flags should have telemetry enabled
	assert.True(t, flags.Telemetry)
}

func TestFetchAndCache_cafeError(t *testing.T) {
	// Given a CAFE server that returns an error
	cacheDir := t.TempDir()
	handler := cafe.NewViewerAPIServer(&fakeViewerAPI{stubbedErr: assert.AnError})
	mux := http.NewServeMux()
	mux.Handle(handler.PathPrefix(), handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cafeClient := cafe.NewClient(server.Client(), cafe.WithBaseURL(server.URL))
	client := NewClient(cafeClient, cacheDir, "github.com", "testuser")

	// When I fetch and cache
	_, err := client.FetchAndCache(context.Background())

	// Then it should return an error
	require.Error(t, err)

	// And no cache file should be written
	_, statErr := os.Stat(cachePath(cacheDir, "github.com", "testuser"))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestFetchAndCache_preservesPriorCacheOnError(t *testing.T) {
	// Given a valid existing cache with telemetry enabled
	cacheDir := t.TempDir()
	writeTestCache(t, cacheDir, "github.com", "testuser", &cache{
		Flags:     map[string]bool{"gh_cli_telemetry": true},
		FetchedAt: time.Now().Add(-31 * time.Minute),
	})

	// And a CAFE server that returns an error
	handler := cafe.NewViewerAPIServer(&fakeViewerAPI{stubbedErr: assert.AnError})
	mux := http.NewServeMux()
	mux.Handle(handler.PathPrefix(), handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cafeClient := cafe.NewClient(server.Client(), cafe.WithBaseURL(server.URL))
	client := NewClient(cafeClient, cacheDir, "github.com", "testuser")

	// When I fetch and cache
	_, err := client.FetchAndCache(context.Background())

	// Then it should return an error
	require.Error(t, err)

	// And the prior cache should be preserved
	flags := Fetch(cacheDir, "github.com", "testuser", "gh")
	assert.True(t, flags.Telemetry)
}

func TestFetchAndCache_updatesCache(t *testing.T) {
	// Given a stale cache with telemetry disabled
	cacheDir := t.TempDir()
	writeTestCache(t, cacheDir, "github.com", "testuser", &cache{
		Flags:     map[string]bool{"gh_cli_telemetry": false},
		FetchedAt: time.Now().Add(-31 * time.Minute),
	})

	// And a CAFE server returning telemetry enabled
	server := newTestServer(t, map[string]bool{"gh_cli_telemetry": true})
	t.Cleanup(server.Close)

	cafeClient := cafe.NewClient(server.Client(), cafe.WithBaseURL(server.URL))
	client := NewClient(cafeClient, cacheDir, "github.com", "testuser")

	// When I fetch and cache
	flags, err := client.FetchAndCache(context.Background())

	// Then it should succeed
	require.NoError(t, err)

	// And the returned flags should reflect the updated value
	assert.True(t, flags.Telemetry)
}

func TestFetchAndCache_emptyResponsePreservesCache(t *testing.T) {
	// Given a valid existing cache with telemetry enabled
	cacheDir := t.TempDir()
	writeTestCache(t, cacheDir, "github.com", "testuser", &cache{
		Flags:     map[string]bool{"gh_cli_telemetry": true},
		FetchedAt: time.Now().Add(-31 * time.Minute),
	})

	// And a CAFE server returning an empty flag set
	server := newTestServer(t, map[string]bool{})
	t.Cleanup(server.Close)

	cafeClient := cafe.NewClient(server.Client(), cafe.WithBaseURL(server.URL))
	client := NewClient(cafeClient, cacheDir, "github.com", "testuser")

	// When I fetch and cache
	_, err := client.FetchAndCache(context.Background())

	// Then it should return an error about the missing flag
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing expected flag")

	// And the prior cache should be preserved
	flags := Fetch(cacheDir, "github.com", "testuser", "gh")
	assert.True(t, flags.Telemetry)
}

// Lock file tests

func TestIsLocked_noLockFile(t *testing.T) {
	// Given no lock file exists
	cacheDir := t.TempDir()

	// Then IsLocked should return false
	assert.False(t, IsLocked(cacheDir, "github.com", "testuser"))
}

func TestIsLocked_freshLock(t *testing.T) {
	// Given a recently created lock file
	cacheDir := t.TempDir()
	require.NoError(t, CreateLockFile(cacheDir, "github.com", "testuser"))
	t.Cleanup(func() { RemoveLockFile(cacheDir, "github.com", "testuser") })

	// Then IsLocked should return true
	assert.True(t, IsLocked(cacheDir, "github.com", "testuser"))
}

func TestIsLocked_staleLock(t *testing.T) {
	// Given a lock file that is older than lockMaxAge
	cacheDir := t.TempDir()
	p := lockPath(cacheDir, "github.com", "testuser")
	require.NoError(t, os.WriteFile(p, []byte("12345"), 0o644))
	staleTime := time.Now().Add(-(lockMaxAge + time.Second))
	require.NoError(t, os.Chtimes(p, staleTime, staleTime))

	// Then IsLocked should return false (stale lock is ignored)
	assert.False(t, IsLocked(cacheDir, "github.com", "testuser"))
}

func TestCreateAndRemoveLockFile(t *testing.T) {
	// Given no lock file exists
	cacheDir := t.TempDir()

	// When I create a lock file
	require.NoError(t, CreateLockFile(cacheDir, "github.com", "testuser"))

	// Then it should be locked
	assert.True(t, IsLocked(cacheDir, "github.com", "testuser"))

	// When I remove the lock file
	RemoveLockFile(cacheDir, "github.com", "testuser")

	// Then it should no longer be locked
	assert.False(t, IsLocked(cacheDir, "github.com", "testuser"))
}

func TestFetch_skipsSpawnWhenLocked(t *testing.T) {
	// Given a missing cache but a fresh lock file (another process is fetching)
	cacheDir := t.TempDir()
	require.NoError(t, CreateLockFile(cacheDir, "github.com", "testuser"))
	t.Cleanup(func() { RemoveLockFile(cacheDir, "github.com", "testuser") })

	// When I fetch feature flags
	flags := Fetch(cacheDir, "github.com", "testuser", "this-executable-does-not-exist")

	// Then it should return defaults without trying to spawn (which would fail with the fake executable)
	assert.False(t, flags.Telemetry)
}

func TestFetch_staleCacheWhenLocked(t *testing.T) {
	// Given a stale cache with telemetry enabled and a lock file held by another process
	cacheDir := t.TempDir()
	writeTestCache(t, cacheDir, "github.com", "testuser", &cache{
		Flags:     map[string]bool{"gh_cli_telemetry": true},
		FetchedAt: time.Now().Add(-31 * time.Minute),
	})
	require.NoError(t, CreateLockFile(cacheDir, "github.com", "testuser"))
	t.Cleanup(func() { RemoveLockFile(cacheDir, "github.com", "testuser") })

	// When I fetch feature flags
	flags := Fetch(cacheDir, "github.com", "testuser", "this-executable-does-not-exist")

	// Then it should return the stale cached value as fallback (not defaults)
	assert.True(t, flags.Telemetry)
}

// FromCache tests

func TestFromCache_freshCache(t *testing.T) {
	// Given a cache with telemetry enabled
	cacheDir := t.TempDir()
	writeTestCache(t, cacheDir, "github.com", "testuser", &cache{
		Flags:     map[string]bool{"gh_cli_telemetry": true},
		FetchedAt: time.Now(),
	})

	// When I read from cache
	flags := FromCache(cacheDir, "github.com", "testuser")

	// Then telemetry should be enabled
	assert.True(t, flags.Telemetry)
}

func TestFromCache_missingCache(t *testing.T) {
	// Given no cache file exists
	cacheDir := t.TempDir()

	// When I read from cache
	flags := FromCache(cacheDir, "github.com", "testuser")

	// Then it should return defaults (all flags disabled)
	assert.False(t, flags.Telemetry)
}

func TestFromCache_corruptCache(t *testing.T) {
	// Given a corrupt cache file
	cacheDir := t.TempDir()
	path := filepath.Join(cacheDir, "github.com-testuser-feature-flags.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	// When I read from cache
	flags := FromCache(cacheDir, "github.com", "testuser")

	// Then it should return defaults (all flags disabled)
	assert.False(t, flags.Telemetry)
}

// runFetchSubprocess tests

func TestRunFetchSubprocess_invalidExecutable(t *testing.T) {
	// Given an executable that does not exist

	// When I run the fetch subprocess
	flags, ok := runFetchSubprocess("this-executable-does-not-exist", "github.com")

	// Then it should indicate failure
	assert.False(t, ok)
	assert.False(t, flags.Telemetry)
}
