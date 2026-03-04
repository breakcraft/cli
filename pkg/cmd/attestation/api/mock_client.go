package api

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"
)

func makeTestReleaseAttestation() Attestation {
	return Attestation{
		Bundle:    data.GitHubReleaseBundle(nil),
		BundleURL: "https://example.com",
		Initiator: "github",
	}
}

func makeTestAttestation() Attestation {
	return Attestation{
		Bundle:    data.SigstoreBundle(nil),
		BundleURL: "https://example.com",
		Initiator: "user",
	}
}

// MockClient is a test double for the Client interface.
type MockClient struct {
	OnGetByDigest    func(params FetchParams) ([]*Attestation, error)
	OnGetTrustDomain func() (string, error)
}

// GetByDigest delegates to the OnGetByDigest callback.
func (m MockClient) GetByDigest(params FetchParams) ([]*Attestation, error) {
	return m.OnGetByDigest(params)
}

// GetTrustDomain delegates to the OnGetTrustDomain callback.
func (m MockClient) GetTrustDomain() (string, error) {
	return m.OnGetTrustDomain()
}

// OnGetByDigestSuccess is a mock callback that returns test attestations successfully.
func OnGetByDigestSuccess(params FetchParams) ([]*Attestation, error) {
	att1 := makeTestAttestation()
	att2 := makeTestAttestation()
	att3 := makeTestReleaseAttestation()
	attestations := []*Attestation{&att1, &att2}
	if params.PredicateType != "" {
		// "release" is a sentinel value that returns all release attestations (v0.1, v0.2, etc.)
		// This mimics the GitHub API behavior which handles this server-side
		if params.PredicateType == "release" {
			return []*Attestation{&att3}, nil
		}
		return FilterAttestations(params.PredicateType, attestations)
	}

	return attestations, nil
}

// OnGetByDigestFailure is a mock callback that always returns an error.
func OnGetByDigestFailure(params FetchParams) ([]*Attestation, error) {
	if params.Repo != "" {
		return nil, fmt.Errorf("failed to fetch attestations from %s", params.Repo)
	}
	return nil, fmt.Errorf("failed to fetch attestations from %s", params.Owner)
}

// NewTestClient returns a MockClient that succeeds on GetByDigest.
func NewTestClient() *MockClient {
	return &MockClient{
		OnGetByDigest: OnGetByDigestSuccess,
	}
}

// NewFailTestClient returns a MockClient that fails on GetByDigest.
func NewFailTestClient() *MockClient {
	return &MockClient{
		OnGetByDigest: OnGetByDigestFailure,
	}
}
