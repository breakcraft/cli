package featuredetection_stub

// Stub types for testing the linter. These mirror the real types.

type IssueFeatures struct {
	StateReason       bool
	ActorIsAssignable bool
}

type PullRequestFeatures struct {
	MergeQueue                     bool
	CheckRunAndStatusContextCounts bool
	CheckRunEvent                  bool
}

type RepositoryFeatures struct {
	PullRequestTemplateQuery bool
	VisibilityField          bool
	AutoMerge                bool
}

type SearchFeatures struct {
	AdvancedIssueSearchAPI      bool
	AdvancedIssueSearchAPIOptIn bool
}

type ReleaseFeatures struct {
	ImmutableReleases bool
}
