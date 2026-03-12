// Package cafe provides a client for the CAFE (Client Apps Front End) feature flag service.
//
// The viewer.pb.go and viewer.twirp.go files are vendored from
// github.com/github/clientappsfe/pkg/api/twirp/identity/v1 to keep the
// request/response types in sync with the canonical protobuf schema.
package cafe

import (
	"context"
	"fmt"
	"net/http"
)

const defaultBaseURL = "https://clientapps.github.com"

// Client talks to the CAFE service to fetch feature flags.
type Client struct {
	viewer ViewerAPI
}

// NewClient creates a CAFE client. If baseURL is empty, the default is used.
// The httpClient must already have authentication configured (e.g. Bearer token transport).
func NewClient(httpClient *http.Client, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		viewer: NewViewerAPIProtobufClient(baseURL, httpClient),
	}
}

// GetFeatureFlags fetches the given flag names from CAFE and returns a map of flag name to enabled state.
func (c *Client) GetFeatureFlags(ctx context.Context, flagNames []string) (map[string]bool, error) {
	resp, err := c.viewer.GetFeatureFlags(ctx, &GetFeatureFlagsRequest{
		FlagNames: flagNames,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching feature flags from CAFE: %w", err)
	}

	flags := make(map[string]bool, len(resp.GetFeatureFlags()))
	for _, f := range resp.GetFeatureFlags() {
		flags[f.GetName()] = f.GetIsEnabled()
	}
	return flags, nil
}
