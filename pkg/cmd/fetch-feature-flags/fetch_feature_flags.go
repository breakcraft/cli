package fetchfeatureflags

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cli/cli/v2/internal/featureflags"
	"github.com/cli/cli/v2/internal/featureflags/cafe"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

const defaultFeatureFlagServerURL = "https://clientapps.github.com"

func NewCmdFetchFeatureFlags(f *cmdutil.Factory) *cobra.Command {
	return newCmdFetchFeatureFlags(f, nil)
}

type FetchFeatureFlagsOptions struct {
	IO                     *iostreams.IOStreams
	FeatureFlagEndpointURL string
	AuthToken              string
	CacheDir               string
	Host                   string
	User                   string
	HTTPUnixSocket         string
	FromCache              bool
}

func newCmdFetchFeatureFlags(f *cmdutil.Factory, runF func(*FetchFeatureFlagsOptions) error) *cobra.Command {
	opts := &FetchFeatureFlagsOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:    "fetch-feature-flags",
		Short:  "Fetch feature flags from CAFE and update the local cache",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			authCfg := cfg.Authentication()
			token, _ := authCfg.ActiveToken(opts.Host)
			if token == "" {
				return errors.New("expected to have a token")
			}

			user, err := authCfg.ActiveUser(opts.Host)
			if err != nil {
				return err
			}

			opts.FeatureFlagEndpointURL = cmp.Or(os.Getenv("FEATURE_FLAG_ENDPOINT_URL"), defaultFeatureFlagServerURL)
			opts.AuthToken = token
			opts.CacheDir = cfg.CacheDir()
			opts.User = user
			opts.HTTPUnixSocket = cfg.HTTPUnixSocket(opts.Host).Value

			if runF != nil {
				return runF(opts)
			}
			return runFetchFeatureFlags(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.FromCache, "from-cache", false, "Print cached feature flags instead of fetching from remote")
	cmd.Flags().StringVar(&opts.Host, "hostname", "", "GitHub hostname to fetch feature flags for")
	_ = cmd.MarkFlagRequired("hostname")

	return cmd
}

func runFetchFeatureFlags(opts *FetchFeatureFlagsOptions) error {
	if opts.FromCache {
		flags := featureflags.FromCache(opts.CacheDir, opts.Host, opts.User)
		flagStr, err := json.MarshalIndent(flags, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.IO.Out, "%s\n", flagStr)
		return nil
	}

	// Acquire a lock file so concurrent gh invocations (e.g. in a loop) don't
	// all try to fetch at the same time.
	if err := featureflags.CreateLockFile(opts.CacheDir, opts.Host, opts.User); err != nil {
		return fmt.Errorf("creating lock file: %w", err)
	}
	defer featureflags.RemoveLockFile(opts.CacheDir, opts.Host, opts.User)

	// TODO: This looks very similar to the send-telemtry http client.
	httpClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &bearerTokenTransport{token: opts.AuthToken, base: handleUnixDomainSocket(opts.HTTPUnixSocket)},
	}

	var cafeOpts []cafe.Option
	if opts.FeatureFlagEndpointURL != "" {
		cafeOpts = append(cafeOpts, cafe.WithBaseURL(opts.FeatureFlagEndpointURL))
	}
	cafeClient := cafe.NewClient(httpClient, cafeOpts...)
	ffClient := featureflags.NewClient(cafeClient, opts.CacheDir, opts.Host, opts.User)

	flags, err := ffClient.FetchAndCache(context.Background())
	if err != nil {
		return err
	}

	// Output the flags on stdout so the parent process can consume them
	// directly without re-reading the cache file.
	data, err := json.Marshal(flags)
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.IO.Out, "%s\n", data)
	return nil
}

type bearerTokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.token))
	return t.base.RoundTrip(req)
}

func handleUnixDomainSocket(socketPath string) http.RoundTripper {
	if socketPath == "" {
		return http.DefaultTransport
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}

	return &http.Transport{
		DialContext:       dialContext,
		DisableKeepAlives: true,
	}
}
