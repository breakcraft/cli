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

// --- Fetch tests ---

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

	// Then it should return the stale cached value (not defaults)
	// The background refresh is fire-and-forget and won't affect this invocation
	assert.True(t, flags.Telemetry)
}

// --- FetchAndCache tests ---

func TestFetchAndCache_success(t *testing.T) {
	// Given a CAFE server returning the telemetry flag as enabled
	cacheDir := t.TempDir()
	server := newTestServer(t, map[string]bool{"gh_cli_telemetry": true})
	t.Cleanup(server.Close)

	cafeClient := cafe.NewClient(server.Client(), cafe.WithBaseURL(server.URL))
	client := NewClient(cafeClient, cacheDir, "github.com", "testuser")

	// When I fetch and cache
	err := client.FetchAndCache(context.Background())

	// Then it should succeed
	require.NoError(t, err)

	// And the cache should contain the enabled flag
	flags := Fetch(cacheDir, "github.com", "testuser", "gh")
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
	err := client.FetchAndCache(context.Background())

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
	err := client.FetchAndCache(context.Background())

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
	err := client.FetchAndCache(context.Background())

	// Then it should succeed
	require.NoError(t, err)

	// And the cache should be updated with the new value
	flags := Fetch(cacheDir, "github.com", "testuser", "gh")
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
	err := client.FetchAndCache(context.Background())

	// Then it should return an error about the missing flag
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing expected flag")

	// And the prior cache should be preserved
	flags := Fetch(cacheDir, "github.com", "testuser", "gh")
	assert.True(t, flags.Telemetry)
}
