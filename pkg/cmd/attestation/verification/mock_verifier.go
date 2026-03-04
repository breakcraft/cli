package verification

import (
	"fmt"
	"testing"

	"github.com/cli/cli/v2/pkg/cmd/attestation/api"
	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"

	in_toto "github.com/in-toto/attestation/go/v1"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

// MockSigstoreVerifier is a test double for the SigstoreVerifier interface.
type MockSigstoreVerifier struct {
	t           *testing.T
	mockResults []*AttestationProcessingResult
}

// Verify returns preconfigured mock results or a default successful result.
func (v *MockSigstoreVerifier) Verify([]*api.Attestation, verify.PolicyBuilder) ([]*AttestationProcessingResult, error) {
	if v.mockResults != nil {
		return v.mockResults, nil
	}

	statement := &in_toto.Statement{}
	statement.PredicateType = SLSAPredicateV1

	result := AttestationProcessingResult{
		Attestation: &api.Attestation{
			Bundle: data.SigstoreBundle(v.t),
		},
		VerificationResult: &verify.VerificationResult{
			Statement: statement,
			Signature: &verify.SignatureVerificationResult{
				Certificate: &certificate.Summary{
					Extensions: certificate.Extensions{
						BuildSignerURI:           "https://github.com/github/example/.github/workflows/release.yml@refs/heads/main",
						SourceRepositoryOwnerURI: "https://github.com/sigstore",
						SourceRepositoryURI:      "https://github.com/sigstore/sigstore-js",
						Issuer:                   "https://token.actions.githubusercontent.com",
					},
				},
			},
		},
	}

	results := []*AttestationProcessingResult{&result}

	return results, nil
}

// NewMockSigstoreVerifier creates a MockSigstoreVerifier with default test results.
func NewMockSigstoreVerifier(t *testing.T) *MockSigstoreVerifier {
	result := BuildSigstoreJsMockResult(t)
	results := []*AttestationProcessingResult{&result}

	return &MockSigstoreVerifier{t, results}
}

// NewMockSigstoreVerifierWithMockResults creates a MockSigstoreVerifier with the given results.
func NewMockSigstoreVerifierWithMockResults(t *testing.T, mockResults []*AttestationProcessingResult) *MockSigstoreVerifier {
	return &MockSigstoreVerifier{t, mockResults}
}

// FailSigstoreVerifier is a test double that always returns a verification error.
type FailSigstoreVerifier struct{}

// Verify always returns an error for the FailSigstoreVerifier.
func (v *FailSigstoreVerifier) Verify([]*api.Attestation, verify.PolicyBuilder) ([]*AttestationProcessingResult, error) {
	return nil, fmt.Errorf("failed to verify attestations")
}

// BuildMockResult creates an AttestationProcessingResult with the given certificate extensions.
func BuildMockResult(b *bundle.Bundle, buildConfigURI, buildSignerURI, sourceRepoOwnerURI, sourceRepoURI, issuer string) AttestationProcessingResult {
	statement := &in_toto.Statement{}
	statement.PredicateType = SLSAPredicateV1

	return AttestationProcessingResult{
		Attestation: &api.Attestation{
			Bundle: b,
		},
		VerificationResult: &verify.VerificationResult{
			Statement: statement,
			Signature: &verify.SignatureVerificationResult{
				Certificate: &certificate.Summary{
					Extensions: certificate.Extensions{
						BuildConfigURI:           buildConfigURI,
						BuildSignerURI:           buildSignerURI,
						Issuer:                   issuer,
						SourceRepositoryOwnerURI: sourceRepoOwnerURI,
						SourceRepositoryURI:      sourceRepoURI,
					},
				},
			},
		},
	}
}

// BuildSigstoreJsMockResult creates a mock result using the sigstore-js test bundle.
func BuildSigstoreJsMockResult(t *testing.T) AttestationProcessingResult {
	bundle := data.SigstoreBundle(t)
	buildConfigURI := "https://github.com/sigstore/sigstore-js/.github/workflows/build.yml@refs/heads/main"
	buildSignerURI := "https://github.com/github/example/.github/workflows/release.yml@refs/heads/main"
	sourceRepoOwnerURI := "https://github.com/sigstore"
	sourceRepoURI := "https://github.com/sigstore/sigstore-js"
	issuer := "https://token.actions.githubusercontent.com"
	return BuildMockResult(bundle, buildConfigURI, buildSignerURI, sourceRepoOwnerURI, sourceRepoURI, issuer)
}
