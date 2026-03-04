package data

import (
	_ "embed"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/bundle"
)

// SigstoreBundleRaw contains the embedded raw bytes of a test Sigstore bundle.
//
//go:embed sigstore-js-2.1.0-bundle.json
var SigstoreBundleRaw []byte

// GitHubReleaseBundleRaw contains the embedded raw bytes of a test GitHub release bundle.
//
//go:embed github_release_bundle.json
var GitHubReleaseBundleRaw []byte

// SigstoreBundle returns a test sigstore-go bundle.Bundle
func SigstoreBundle(t *testing.T) *bundle.Bundle {
	b := &bundle.Bundle{}
	err := b.UnmarshalJSON(SigstoreBundleRaw)
	if err != nil {
		t.Fatalf("failed to unmarshal sigstore bundle: %v", err)
	}
	return b
}

// GitHubReleaseBundle returns a test GitHub release bundle.Bundle.
func GitHubReleaseBundle(t *testing.T) *bundle.Bundle {
	b := &bundle.Bundle{}
	err := b.UnmarshalJSON(GitHubReleaseBundleRaw)
	if err != nil {
		t.Fatalf("failed to unmarshal GitHub release bundle: %v", err)
	}
	return b
}
