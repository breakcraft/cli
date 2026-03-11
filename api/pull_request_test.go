package api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChecksStatus_NoCheckRunsOrStatusContexts(t *testing.T) {
	t.Parallel()

	payload := `
	{ "statusCheckRollup": { "nodes": [] } }
	`
	var pr PullRequest
	require.NoError(t, json.Unmarshal([]byte(payload), &pr))

	expectedChecksStatus := PullRequestChecksStatus{
		Pending: 0,
		Failing: 0,
		Passing: 0,
		Total:   0,
	}
	require.Equal(t, expectedChecksStatus, pr.ChecksStatus())
}

func TestChecksStatus_SummarisingCheckRuns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		payload              string
		expectedChecksStatus PullRequestChecksStatus
	}{
		{
			name:                 "QUEUED is treated as Pending",
			payload:              singleCheckRunWithStatus("QUEUED"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "IN_PROGRESS is treated as Pending",
			payload:              singleCheckRunWithStatus("IN_PROGRESS"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "WAITING is treated as Pending",
			payload:              singleCheckRunWithStatus("WAITING"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "PENDING is treated as Pending",
			payload:              singleCheckRunWithStatus("PENDING"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "REQUESTED is treated as Pending",
			payload:              singleCheckRunWithStatus("REQUESTED"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "COMPLETED with no conclusion is treated as Pending",
			payload:              singleCheckRunWithStatus("COMPLETED"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "COMPLETED / STARTUP_FAILURE is treated as Pending",
			payload:              singleCompletedCheckRunWithConclusion("STARTUP_FAILURE"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "COMPLETED / STALE is treated as Pending",
			payload:              singleCompletedCheckRunWithConclusion("STALE"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "COMPLETED / SUCCESS is treated as Passing",
			payload:              singleCompletedCheckRunWithConclusion("SUCCESS"),
			expectedChecksStatus: PullRequestChecksStatus{Passing: 1, Total: 1},
		},
		{
			name:                 "COMPLETED / NEUTRAL is treated as Passing",
			payload:              singleCompletedCheckRunWithConclusion("NEUTRAL"),
			expectedChecksStatus: PullRequestChecksStatus{Passing: 1, Total: 1},
		},
		{
			name:                 "COMPLETED / SKIPPED is treated as Passing",
			payload:              singleCompletedCheckRunWithConclusion("SKIPPED"),
			expectedChecksStatus: PullRequestChecksStatus{Passing: 1, Total: 1},
		},
		{
			name:                 "COMPLETED / ACTION_REQUIRED is treated as Failing",
			payload:              singleCompletedCheckRunWithConclusion("ACTION_REQUIRED"),
			expectedChecksStatus: PullRequestChecksStatus{Failing: 1, Total: 1},
		},
		{
			name:                 "COMPLETED / TIMED_OUT is treated as Failing",
			payload:              singleCompletedCheckRunWithConclusion("TIMED_OUT"),
			expectedChecksStatus: PullRequestChecksStatus{Failing: 1, Total: 1},
		},
		{
			name:                 "COMPLETED / CANCELLED is excluded from counts",
			payload:              singleCompletedCheckRunWithConclusion("CANCELLED"),
			expectedChecksStatus: PullRequestChecksStatus{},
		},
		{
			name:                 "COMPLETED / CANCELLED is excluded from counts (duplicate)",
			payload:              singleCompletedCheckRunWithConclusion("CANCELLED"),
			expectedChecksStatus: PullRequestChecksStatus{},
		},
		{
			name:                 "COMPLETED / FAILURE is treated as Failing",
			payload:              singleCompletedCheckRunWithConclusion("FAILURE"),
			expectedChecksStatus: PullRequestChecksStatus{Failing: 1, Total: 1},
		},
		{
			name:                 "Unrecognized Status are treated as Pending",
			payload:              singleCheckRunWithStatus("AnUnrecognizedStatusJustForThisTest"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "Unrecognized Conclusions are treated as Pending",
			payload:              singleCompletedCheckRunWithConclusion("AnUnrecognizedConclusionJustForThisTest"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var pr PullRequest
			require.NoError(t, json.Unmarshal([]byte(tt.payload), &pr))

			require.Equal(t, tt.expectedChecksStatus, pr.ChecksStatus())
		})
	}
}

func TestChecksStatus_SummarisingStatusContexts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		payload              string
		expectedChecksStatus PullRequestChecksStatus
	}{
		{
			name:                 "EXPECTED is treated as Pending",
			payload:              singleStatusContextWithState("EXPECTED"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "PENDING is treated as Pending",
			payload:              singleStatusContextWithState("PENDING"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
		{
			name:                 "SUCCESS is treated as Passing",
			payload:              singleStatusContextWithState("SUCCESS"),
			expectedChecksStatus: PullRequestChecksStatus{Passing: 1, Total: 1},
		},
		{
			name:                 "ERROR is treated as Failing",
			payload:              singleStatusContextWithState("ERROR"),
			expectedChecksStatus: PullRequestChecksStatus{Failing: 1, Total: 1},
		},
		{
			name:                 "FAILURE is treated as Failing",
			payload:              singleStatusContextWithState("FAILURE"),
			expectedChecksStatus: PullRequestChecksStatus{Failing: 1, Total: 1},
		},
		{
			name:                 "Unrecognized States are treated as Pending",
			payload:              singleStatusContextWithState("AnUnrecognizedStateJustForThisTest"),
			expectedChecksStatus: PullRequestChecksStatus{Pending: 1, Total: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var pr PullRequest
			require.NoError(t, json.Unmarshal([]byte(tt.payload), &pr))

			require.Equal(t, tt.expectedChecksStatus, pr.ChecksStatus())
		})
	}
}

func TestChecksStatus_SummarisingCheckRunsAndStatusContexts(t *testing.T) {
	t.Parallel()

	// This might look a bit intimidating, but we're just inserting three nodes
	// into the rollup, two completed check run nodes and one status context node.
	payload := fmt.Sprintf(`
	{ "statusCheckRollup": { "nodes": [{ "commit": {
		"statusCheckRollup": {
			"contexts": {
				"nodes": [
					%s,
					%s,
					%s
				]
			}
		}
	} }] } }
	`,
		completedCheckRunNodeWithName("build", "SUCCESS"),
		statusContextNodeWithName("ci/deploy", "PENDING"),
		completedCheckRunNodeWithName("lint", "FAILURE"),
	)

	var pr PullRequest
	require.NoError(t, json.Unmarshal([]byte(payload), &pr))

	expectedChecksStatus := PullRequestChecksStatus{
		Pending: 1,
		Failing: 1,
		Passing: 1,
		Total:   3,
	}
	require.Equal(t, expectedChecksStatus, pr.ChecksStatus())
}

func TestChecksStatus_SummarisingCheckRunAndStatusContextCountsByState(t *testing.T) {
	t.Parallel()

	payload := `
	{ "statusCheckRollup": { "nodes": [{ "commit": {
		"statusCheckRollup": {
			"contexts": {
				"checkRunCount": 14,
				"checkRunCountsByState": [
					{
						"state": "ACTION_REQUIRED",
						"count": 1
					},
					{
						"state": "CANCELLED",
						"count": 1
					},
					{
						"state": "COMPLETED",
						"count": 1
					},
					{
						"state": "FAILURE",
						"count": 1
					},
					{
						"state": "IN_PROGRESS",
						"count": 1
					},
					{
						"state": "NEUTRAL",
						"count": 1
					},
					{
						"state": "PENDING",
						"count": 1
					},
					{
						"state": "QUEUED",
						"count": 1
					},
					{
						"state": "SKIPPED",
						"count": 1
					},
					{
						"state": "STALE",
						"count": 1
					},
					{
						"state": "STARTUP_FAILURE",
						"count": 1
					},
					{
						"state": "SUCCESS",
						"count": 1
					},
					{
						"state": "TIMED_OUT",
						"count": 1
					},
					{
						"state": "WAITING",
						"count": 1
					},
					{
						"state": "AnUnrecognizedStateJustForThisTest",
						"count": 1
					}
				],
				"statusContextCount": 6,
				"statusContextCountsByState": [
					{
						"state": "EXPECTED",
						"count": 1
					},
					{
						"state": "ERROR",
						"count": 1
					},
					{
						"state": "FAILURE",
						"count": 1
					},
					{
						"state": "PENDING",
						"count": 1
					},
					{
						"state": "SUCCESS",
						"count": 1
					},
					{
						"state": "AnUnrecognizedStateJustForThisTest",
						"count": 1
					}
				]
			}
		}
	} }] } }
	`

	var pr PullRequest
	require.NoError(t, json.Unmarshal([]byte(payload), &pr))

	expectedChecksStatus := PullRequestChecksStatus{
		Pending: 11,
		Failing: 5,
		Passing: 4,
		Total:   19,
	}
	require.Equal(t, expectedChecksStatus, pr.ChecksStatus())
}

// Note that it would be incorrect to provide a status of COMPLETED here
// as the conclusion is always set to null. If you want a COMPLETED status,
// use `singleCompletedCheckRunWithConclusion`.
func singleCheckRunWithStatus(status string) string {
	return fmt.Sprintf(`
	{ "statusCheckRollup": { "nodes": [{ "commit": {
		"statusCheckRollup": {
			"contexts": {
				"nodes": [
					{
						"__typename": "CheckRun",
						"status": "%s",
						"conclusion": null
					}
				]
			}
		}
	} }] } }
	`, status)
}

func singleCompletedCheckRunWithConclusion(conclusion string) string {
	return fmt.Sprintf(`
	{ "statusCheckRollup": { "nodes": [{ "commit": {
		"statusCheckRollup": {
			"contexts": {
				"nodes": [
					{
						"__typename": "CheckRun",
						"status": "COMPLETED",
						"conclusion": "%s"
					}
				]
			}
		}
	} }] } }
	`, conclusion)
}

func singleStatusContextWithState(state string) string {
	return fmt.Sprintf(`
	{ "statusCheckRollup": { "nodes": [{ "commit": {
		"statusCheckRollup": {
			"contexts": {
				"nodes": [
					{
						"__typename": "StatusContext",
						"state": "%s"
					}
				]
			}
		}
	} }] } }
	`, state)
}

func completedCheckRunNodeWithName(name, conclusion string) string {
	return fmt.Sprintf(`
	{
		"__typename": "CheckRun",
		"name": "%s",
		"status": "COMPLETED",
		"conclusion": "%s"
	}`, name, conclusion)
}

func statusContextNodeWithName(context, state string) string {
	return fmt.Sprintf(`
	{
		"__typename": "StatusContext",
		"context": "%s",
		"state": "%s"
	}`, context, state)
}

func TestChecksStatus_DuplicateCheckRunsAreDeduplicated(t *testing.T) {
	t.Parallel()

	// Simulate cancel-in-progress: a cancelled run followed by a newer successful run
	// for the same check name. Only the newer (successful) run should be counted.
	pr := PullRequest{}
	pr.StatusCheckRollup.Nodes = []StatusCheckRollupNode{
		{
			Commit: StatusCheckRollupCommit{
				StatusCheckRollup: CommitStatusCheckRollup{
					Contexts: CheckContexts{
						Nodes: []CheckContext{
							{
								TypeName:   "CheckRun",
								Name:       "Prevent Merging",
								Status:     "COMPLETED",
								Conclusion: CheckConclusionStateCancelled,
								StartedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
							},
							{
								TypeName:   "CheckRun",
								Name:       "Prevent Merging",
								Status:     "COMPLETED",
								Conclusion: CheckConclusionStateSuccess,
								StartedAt:  time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
							},
							{
								TypeName:   "CheckRun",
								Name:       "Build",
								Status:     "COMPLETED",
								Conclusion: CheckConclusionStateSuccess,
								StartedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
							},
						},
					},
				},
			},
		},
	}

	status := pr.ChecksStatus()
	require.Equal(t, PullRequestChecksStatus{
		Passing: 2,
		Failing: 0,
		Pending: 0,
		Total:   2,
	}, status)
}

func TestEliminateDuplicateChecks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		checkContexts []CheckContext
		want          []CheckContext
	}{
		{
			name: "duplicate CheckRun keeps most recent",
			checkContexts: []CheckContext{
				{
					TypeName:   "CheckRun",
					Name:       "lint",
					Status:     "COMPLETED",
					Conclusion: "FAILURE",
					StartedAt:  time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
				},
				{
					TypeName:   "CheckRun",
					Name:       "lint",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 2, 2, 2, 2, 2, 2, time.UTC),
				},
				{
					TypeName:   "CheckRun",
					Name:       "build",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
				},
			},
			want: []CheckContext{
				{
					TypeName:   "CheckRun",
					Name:       "lint",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 2, 2, 2, 2, 2, 2, time.UTC),
				},
				{
					TypeName:   "CheckRun",
					Name:       "build",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
				},
			},
		},
		{
			name: "duplicate StatusContext keeps most recent",
			checkContexts: []CheckContext{
				{
					TypeName:  "StatusContext",
					Context:   "ci/test",
					State:     "FAILURE",
					StartedAt: time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
				},
				{
					TypeName:  "StatusContext",
					Context:   "ci/test",
					State:     "SUCCESS",
					StartedAt: time.Date(2022, 2, 2, 2, 2, 2, 2, time.UTC),
				},
			},
			want: []CheckContext{
				{
					TypeName:  "StatusContext",
					Context:   "ci/test",
					State:     "SUCCESS",
					StartedAt: time.Date(2022, 2, 2, 2, 2, 2, 2, time.UTC),
				},
			},
		},
		{
			name: "unique checks are preserved",
			checkContexts: []CheckContext{
				{
					TypeName:   "CheckRun",
					Name:       "build",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
				},
				{
					TypeName:  "StatusContext",
					Context:   "ci/test",
					State:     "SUCCESS",
					StartedAt: time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
				},
			},
			want: []CheckContext{
				{
					TypeName:   "CheckRun",
					Name:       "build",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
				},
				{
					TypeName:  "StatusContext",
					Context:   "ci/test",
					State:     "SUCCESS",
					StartedAt: time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
				},
			},
		},
		{
			name: "different workflow names are not deduplicated",
			checkContexts: []CheckContext{
				{
					TypeName:   "CheckRun",
					Name:       "build",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
					CheckSuite: CheckSuite{WorkflowRun: WorkflowRun{Event: "push", Workflow: Workflow{Name: "CI"}}},
				},
				{
					TypeName:   "CheckRun",
					Name:       "build",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
					CheckSuite: CheckSuite{WorkflowRun: WorkflowRun{Event: "push", Workflow: Workflow{Name: "Release"}}},
				},
			},
			want: []CheckContext{
				{
					TypeName:   "CheckRun",
					Name:       "build",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
					CheckSuite: CheckSuite{WorkflowRun: WorkflowRun{Event: "push", Workflow: Workflow{Name: "CI"}}},
				},
				{
					TypeName:   "CheckRun",
					Name:       "build",
					Status:     "COMPLETED",
					Conclusion: "SUCCESS",
					StartedAt:  time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
					CheckSuite: CheckSuite{WorkflowRun: WorkflowRun{Event: "push", Workflow: Workflow{Name: "Release"}}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EliminateDuplicateChecks(tt.checkContexts)
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("got EliminateDuplicateChecks %+v, want %+v", got, tt.want)
			}
		})
	}
}
