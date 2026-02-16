package example

import (
	fd "featuredetection_stub"
)

// --- Cases that should trigger diagnostics (missing TODO) ---

func missingTODO_IssueFeatures(features fd.IssueFeatures) {
	if features.StateReason { // want `if-statement references featuredetection field "StateReason" but is missing a required TODO comment`
		_ = "has state reason"
	}
}

func missingTODO_PullRequestFeatures(features fd.PullRequestFeatures) {
	if features.MergeQueue { // want `if-statement references featuredetection field "MergeQueue" but is missing a required TODO comment`
		_ = "merge queue enabled"
	}
}

func missingTODO_RepositoryFeatures(features fd.RepositoryFeatures) {
	if features.VisibilityField { // want `if-statement references featuredetection field "VisibilityField" but is missing a required TODO comment`
		_ = "has visibility"
	}
}

func missingTODO_SearchFeatures(features fd.SearchFeatures) {
	if features.AdvancedIssueSearchAPI { // want `if-statement references featuredetection field "AdvancedIssueSearchAPI" but is missing a required TODO comment`
		_ = "advanced search"
	}
}

func missingTODO_ReleaseFeatures(features fd.ReleaseFeatures) {
	if !features.ImmutableReleases { // want `if-statement references featuredetection field "ImmutableReleases" but is missing a required TODO comment`
		_ = "no immutable releases"
	}
}

func missingTODO_pointer(features *fd.PullRequestFeatures) {
	if features.CheckRunEvent { // want `if-statement references featuredetection field "CheckRunEvent" but is missing a required TODO comment`
		_ = "check run event"
	}
}

// --- Cases that should NOT trigger diagnostics (TODO present) ---

func hasTODO_simple(features fd.IssueFeatures) {
	// TODO stateReasonCleanup
	if features.StateReason {
		_ = "has state reason"
	}
}

func hasTODO_colon(features fd.PullRequestFeatures) {
	// TODO: mergeQueueCleanup
	if features.MergeQueue {
		_ = "merge queue enabled"
	}
}

func hasTODO_withDescription(features fd.RepositoryFeatures) {
	// TODO visibilityCleanup
	// Once all GHES versions support visibility, remove this.
	if features.VisibilityField {
		_ = "has visibility"
	}
}

func hasTODO_negated(features fd.ReleaseFeatures) {
	// TODO: immutableReleaseFullSupport
	if !features.ImmutableReleases {
		_ = "no immutable releases"
	}
}

// --- Cases that should NOT trigger diagnostics (not featuredetection) ---

type unrelatedStruct struct {
	StateReason bool
}

func notFeatureDetection(s unrelatedStruct) {
	if s.StateReason {
		_ = "no diagnostic"
	}
}

// --- Edge case: TODO too far away (11 lines above) ---

func todoTooFarAway(features fd.IssueFeatures) {
	// TODO stateReasonCleanup
	_ = "line 1"
	_ = "line 2"
	_ = "line 3"
	_ = "line 4"
	_ = "line 5"
	_ = "line 6"
	_ = "line 7"
	_ = "line 8"
	_ = "line 9"
	_ = "line 10"
	if features.StateReason { // want `if-statement references featuredetection field "StateReason" but is missing a required TODO comment`
		_ = "too far"
	}
}

// --- Edge case: TODO exactly 10 lines above ---

func todoExactly10LinesAbove(features fd.IssueFeatures) {
	// TODO stateReasonCleanup
	_ = "line 1"
	_ = "line 2"
	_ = "line 3"
	_ = "line 4"
	_ = "line 5"
	_ = "line 6"
	_ = "line 7"
	_ = "line 8"
	_ = "line 9"
	if features.StateReason {
		_ = "just right"
	}
}
