package checks

import (
	"time"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/pkg/cmdutil"
)

type check struct {
	Name        string    `json:"name"`
	State       string    `json:"state"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
	Link        string    `json:"link"`
	Bucket      string    `json:"bucket"`
	Event       string    `json:"event"`
	Workflow    string    `json:"workflow"`
	Description string    `json:"description"`
}

type checkCounts struct {
	Failed   int
	Passed   int
	Pending  int
	Skipping int
	Canceled int
}

func (ch *check) ExportData(fields []string) map[string]interface{} {
	return cmdutil.StructExportData(ch, fields)
}

func aggregateChecks(checkContexts []api.CheckContext, requiredChecks bool) (checks []check, counts checkCounts) {
	for _, c := range api.EliminateDuplicateChecks(checkContexts) {
		if requiredChecks && !c.IsRequired {
			continue
		}

		state := string(c.State)
		if state == "" {
			if c.Status == "COMPLETED" {
				state = string(c.Conclusion)
			} else {
				state = c.Status
			}
		}

		link := c.DetailsURL
		if link == "" {
			link = c.TargetURL
		}

		name := c.Name
		if name == "" {
			name = c.Context
		}

		item := check{
			Name:        name,
			State:       state,
			StartedAt:   c.StartedAt,
			CompletedAt: c.CompletedAt,
			Link:        link,
			Event:       c.CheckSuite.WorkflowRun.Event,
			Workflow:    c.CheckSuite.WorkflowRun.Workflow.Name,
			Description: c.Description,
		}

		switch state {
		case "SUCCESS":
			item.Bucket = "pass"
			counts.Passed++
		case "SKIPPED", "NEUTRAL":
			item.Bucket = "skipping"
			counts.Skipping++
		case "ERROR", "FAILURE", "TIMED_OUT", "ACTION_REQUIRED":
			item.Bucket = "fail"
			counts.Failed++
		case "CANCELLED":
			item.Bucket = "cancel"
			counts.Canceled++
		default: // "EXPECTED", "REQUESTED", "WAITING", "QUEUED", "PENDING", "IN_PROGRESS", "STALE"
			item.Bucket = "pending"
			counts.Pending++
		}

		checks = append(checks, item)
	}
	return
}
