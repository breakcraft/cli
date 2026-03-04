package api

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/bundle"
)

const (
	// GetAttestationByRepoAndSubjectDigestPath is the API path for fetching attestations by repo and digest.
	GetAttestationByRepoAndSubjectDigestPath = "repos/%s/attestations/%s"
	// GetAttestationByOwnerAndSubjectDigestPath is the API path for fetching attestations by owner and digest.
	GetAttestationByOwnerAndSubjectDigestPath = "orgs/%s/attestations/%s"
)

// ErrNoAttestationsFound is returned when no attestations are found for a given digest.
var ErrNoAttestationsFound = errors.New("no attestations found")

// Attestation represents a single attestation bundle with its metadata.
type Attestation struct {
	Bundle    *bundle.Bundle `json:"bundle"`
	BundleURL string         `json:"bundle_url"`
	Initiator string         `json:"initiator"`
}

// AttestationsResponse wraps a list of attestations returned by the API.
type AttestationsResponse struct {
	Attestations []*Attestation `json:"attestations"`
}

// IntotoStatement represents a minimal in-toto statement with a predicate type.
type IntotoStatement struct {
	PredicateType string `json:"predicateType"`
}

// FilterAttestations returns only attestations matching the given predicate type.
func FilterAttestations(predicateType string, attestations []*Attestation) ([]*Attestation, error) {
	filteredAttestations := []*Attestation{}

	for _, each := range attestations {
		dsseEnvelope := each.Bundle.GetDsseEnvelope()
		if dsseEnvelope != nil {
			if dsseEnvelope.PayloadType != "application/vnd.in-toto+json" {
				// Don't fail just because an entry isn't intoto
				continue
			}
			var intotoStatement IntotoStatement
			if err := json.Unmarshal([]byte(dsseEnvelope.Payload), &intotoStatement); err != nil {
				// Don't fail just because a single entry can't be unmarshalled
				continue
			}
			if intotoStatement.PredicateType == predicateType {
				filteredAttestations = append(filteredAttestations, each)
			}
		}
	}

	if len(filteredAttestations) == 0 {
		return nil, fmt.Errorf("no attestations found with predicate type: %s", predicateType)
	}

	return filteredAttestations, nil
}
