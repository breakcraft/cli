package cafe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeViewerAPI implements the ViewerAPI interface for testing.
type fakeViewerAPI struct {
	flags      []*FeatureFlag
	stubbedErr error
}

func (f *fakeViewerAPI) GetDetails(context.Context, *GetDetailsRequest) (*GetDetailsResponse, error) {
	return nil, nil
}

func (f *fakeViewerAPI) GetFeatureFlags(_ context.Context, req *GetFeatureFlagsRequest) (*GetFeatureFlagsResponse, error) {
	if f.stubbedErr != nil {
		return nil, f.stubbedErr
	}
	return &GetFeatureFlagsResponse{FeatureFlags: f.flags}, nil
}

func newTestServer(t *testing.T, api ViewerAPI) *httptest.Server {
	t.Helper()
	handler := NewViewerAPIServer(api)
	mux := http.NewServeMux()
	mux.Handle(handler.PathPrefix(), handler)
	return httptest.NewServer(mux)
}

func TestGetFeatureFlags(t *testing.T) {
	tests := []struct {
		name      string
		flagNames []string
		flags     []*FeatureFlag
		wantFlags map[string]bool
	}{
		{
			name:      "returns enabled flags",
			flagNames: []string{"gh_cli_telemetry"},
			flags: []*FeatureFlag{
				{Name: "gh_cli_telemetry", IsEnabled: true},
			},
			wantFlags: map[string]bool{"gh_cli_telemetry": true},
		},
		{
			name:      "returns disabled flags",
			flagNames: []string{"gh_cli_telemetry"},
			flags: []*FeatureFlag{
				{Name: "gh_cli_telemetry", IsEnabled: false},
			},
			wantFlags: map[string]bool{"gh_cli_telemetry": false},
		},
		{
			name:      "returns multiple flags",
			flagNames: []string{"flag_a", "flag_b"},
			flags: []*FeatureFlag{
				{Name: "flag_a", IsEnabled: true},
				{Name: "flag_b", IsEnabled: false},
			},
			wantFlags: map[string]bool{"flag_a": true, "flag_b": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t, &fakeViewerAPI{flags: tt.flags})
			defer server.Close()

			client := NewClient(server.Client(), server.URL)
			flags, err := client.GetFeatureFlags(context.Background(), tt.flagNames)

			require.NoError(t, err)
			assert.Equal(t, tt.wantFlags, flags)
		})
	}
}

func TestGetFeatureFlags_serverError(t *testing.T) {
	server := newTestServer(t, &fakeViewerAPI{stubbedErr: assert.AnError})
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	_, err := client.GetFeatureFlags(context.Background(), []string{"flag"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching feature flags from CAFE")
}

func TestGetFeatureFlags_connectionError(t *testing.T) {
	client := NewClient(http.DefaultClient, "http://localhost:1")
	_, err := client.GetFeatureFlags(context.Background(), []string{"flag"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching feature flags from CAFE")
}

func TestNewClient_defaultBaseURL(t *testing.T) {
	// Verify it doesn't panic and creates a valid client with the default URL
	client := NewClient(http.DefaultClient, "")
	assert.NotNil(t, client.viewer)
}
