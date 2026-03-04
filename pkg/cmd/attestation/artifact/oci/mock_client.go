package oci

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmd/attestation/api"
	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func makeTestAttestation() api.Attestation {
	return api.Attestation{Bundle: data.SigstoreBundle(nil)}
}

// MockClient is a test double that returns successful OCI responses.
type MockClient struct{}

// GetImageDigest returns a fixed test digest.
func (c MockClient) GetImageDigest(imgName string) (*v1.Hash, name.Reference, error) {
	return &v1.Hash{
		Hex:       "1234567890abcdef",
		Algorithm: "sha256",
	}, nil, nil
}

// GetAttestations returns test attestations.
func (c MockClient) GetAttestations(name name.Reference, digest string) ([]*api.Attestation, error) {
	att1 := makeTestAttestation()
	att2 := makeTestAttestation()
	return []*api.Attestation{&att1, &att2}, nil
}

// ReferenceFailClient is a test double that fails to parse OCI references.
type ReferenceFailClient struct{}

// GetImageDigest always returns a reference parse error.
func (c ReferenceFailClient) GetImageDigest(imgName string) (*v1.Hash, name.Reference, error) {
	return nil, nil, fmt.Errorf("failed to parse reference")
}

// GetAttestations returns nil for the ReferenceFailClient.
func (c ReferenceFailClient) GetAttestations(name name.Reference, digest string) ([]*api.Attestation, error) {
	return nil, nil
}

// AuthFailClient is a test double that returns authentication errors.
type AuthFailClient struct{}

// GetImageDigest always returns an authentication error.
func (c AuthFailClient) GetImageDigest(imgName string) (*v1.Hash, name.Reference, error) {
	return nil, nil, ErrRegistryAuthz
}

// GetAttestations returns nil for the AuthFailClient.
func (c AuthFailClient) GetAttestations(name name.Reference, digest string) ([]*api.Attestation, error) {
	return nil, nil
}

// DeniedClient is a test double that returns access denied errors.
type DeniedClient struct{}

// GetImageDigest always returns an access denied error.
func (c DeniedClient) GetImageDigest(imgName string) (*v1.Hash, name.Reference, error) {
	return nil, nil, ErrDenied
}

// GetAttestations returns nil for the DeniedClient.
func (c DeniedClient) GetAttestations(name name.Reference, digest string) ([]*api.Attestation, error) {
	return nil, nil
}

// NoAttestationsClient is a test double that returns no attestations.
type NoAttestationsClient struct{}

// GetImageDigest returns a fixed test digest for the NoAttestationsClient.
func (c NoAttestationsClient) GetImageDigest(imgName string) (*v1.Hash, name.Reference, error) {
	return &v1.Hash{
		Hex:       "1234567890abcdef",
		Algorithm: "sha256",
	}, nil, nil
}

// GetAttestations returns an empty result for the NoAttestationsClient.
func (c NoAttestationsClient) GetAttestations(name name.Reference, digest string) ([]*api.Attestation, error) {
	return nil, nil
}

// FailedToFetchAttestationsClient is a test double that fails when fetching attestations.
type FailedToFetchAttestationsClient struct{}

// GetImageDigest returns a fixed test digest for the FailedToFetchAttestationsClient.
func (c FailedToFetchAttestationsClient) GetImageDigest(imgName string) (*v1.Hash, name.Reference, error) {
	return &v1.Hash{
		Hex:       "1234567890abcdef",
		Algorithm: "sha256",
	}, nil, nil
}

// GetAttestations always returns an error for the FailedToFetchAttestationsClient.
func (c FailedToFetchAttestationsClient) GetAttestations(name name.Reference, digest string) ([]*api.Attestation, error) {
	return nil, fmt.Errorf("failed to fetch attestations")
}
