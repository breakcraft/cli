package api

// MetaPath is the API path for fetching server metadata including trust domain.
const MetaPath = "meta"

// ArtifactAttestations holds the trust domain for artifact attestations.
type ArtifactAttestations struct {
	TrustDomain string `json:"trust_domain"`
}

// Domain contains the domain-specific configuration from the API meta response.
type Domain struct {
	ArtifactAttestations ArtifactAttestations `json:"artifact_attestations"`
}

// MetaResponse represents the response from the GitHub API meta endpoint.
type MetaResponse struct {
	Domains Domain `json:"domains"`
}
