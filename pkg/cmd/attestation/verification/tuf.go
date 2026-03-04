package verification

import (
	_ "embed"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cenkalti/backoff/v5"
	o "github.com/cli/cli/v2/pkg/option"
	"github.com/cli/go-gh/v2/pkg/config"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/theupdateframework/go-tuf/v2/metadata/fetcher"
)

//go:embed embed/tuf-repo.github.com/root.json
var githubRoot []byte

// GitHubTUFMirror is the URL of GitHub's TUF repository mirror.
const GitHubTUFMirror = "https://tuf-repo.github.com"

// DefaultOptionsWithCacheSetting returns TUF options with sensible cache defaults for the current environment.
func DefaultOptionsWithCacheSetting(tufMetadataDir o.Option[string], hc *http.Client) *tuf.Options {
	opts := tuf.DefaultOptions()

	// The CODESPACES environment variable will be set to true in a Codespaces workspace
	if os.Getenv("CODESPACES") == "true" {
		// if the tool is being used in a Codespace, disable the local cache
		// because there is a permissions issue preventing the tuf library
		// from writing the Sigstore cache to the home directory
		opts.DisableLocalCache = true
	}

	// Set the cache path to the provided dir, or a directory owned by the CLI
	opts.CachePath = tufMetadataDir.UnwrapOr(filepath.Join(config.CacheDir(), ".sigstore", "root"))

	// Allow TUF cache for 1 day
	opts.CacheValidity = 1

	// configure fetcher timeout and retry
	f := fetcher.NewDefaultFetcher()
	f.SetHTTPClient(hc)
	retryOptions := []backoff.RetryOption{backoff.WithMaxTries(3)}
	f.SetRetryOptions(retryOptions...)
	opts.WithFetcher(f)

	return opts
}

// GitHubTUFOptions returns TUF options configured for GitHub's TUF repository.
func GitHubTUFOptions(tufMetadataDir o.Option[string], hc *http.Client) *tuf.Options {
	opts := DefaultOptionsWithCacheSetting(tufMetadataDir, hc)

	opts.Root = githubRoot
	opts.RepositoryBaseURL = GitHubTUFMirror

	return opts
}
