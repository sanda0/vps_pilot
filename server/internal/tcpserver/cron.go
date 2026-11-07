package tcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/sanda0/vps_pilot/internal/db"
)

// CronRunsKept is how many runs of each cron job the hub keeps.
const CronRunsKept = 50

// maxCronOutput is the most output the agent sends with a run; more from a
// misbehaving agent is cut the same way, keeping the end.
const maxCronOutput = 64 << 10

// Cron records the cron job runs agents report in cron_runs. The agent runs
// the jobs itself; the hub can't start, add or change one. See "Cron jobs"
// in docs/PROTOCOL.md.
type Cron struct {
	repo *db.Repo
}

func NewCron(repo *db.Repo) *Cron {
	return &Cron{repo: repo}
}

var cronStatuses = map[string]bool{
	"exited": true, "timeout": true, "skipped": true, "rejected": true, "failed": true,
}

// HandleResult records a cron_result and drops the job's oldest runs.
func (cr *Cron) HandleResult(ctx context.Context, c *ConnContext, msg Msg) {
	var r CronResult
	if err := json.Unmarshal(msg.Data, &r); err != nil || r.RunID == "" || r.ProjectPath == "" || r.Command == "" {
		fmt.Printf("HandleCronResult: bad cron_result from node %d: %v\n", c.NodeID, err)
		return
	}
	if !cronStatuses[r.Status] {
		r.Error = fmt.Sprintf("agent sent unknown status %q", r.Status)
		r.Status, r.ExitCode = "failed", nil
	}
	if len(r.Output) > maxCronOutput {
		start := len(r.Output) - maxCronOutput
		for start < len(r.Output) && !utf8.RuneStart(r.Output[start]) {
			start++
		}
		r.Output, r.Truncated = r.Output[start:], true
	}
	params := db.InsertCronRunParams{
		ID: r.RunID, NodeID: c.NodeID, ProjectPath: r.ProjectPath, Command: r.Command,
		Schedule: r.Schedule, ScheduledAt: r.ScheduledAt, Status: r.Status, RunAs: r.RunAs,
		DurationMs: r.DurationMS, Output: r.Output, Error: r.Error,
	}
	if r.ExitCode != nil {
		params.ExitCode.Int64, params.ExitCode.Valid = *r.ExitCode, true
	}
	if r.Truncated {
		params.Truncated = 1
	}
	n, err := cr.repo.Queries.InsertCronRun(ctx, params)
	if err != nil {
		fmt.Printf("HandleCronResult: node %d run %s: %v\n", c.NodeID, r.RunID, err)
		return
	}
	if n == 0 {
		return // already recorded
	}
	_, err = cr.repo.Queries.PruneCronRuns(ctx, db.PruneCronRunsParams{
		NodeID: c.NodeID, ProjectPath: r.ProjectPath, Command: r.Command, Keep: CronRunsKept,
	})
	if err != nil {
		fmt.Printf("HandleCronResult: prune node %d %s %q: %v\n", c.NodeID, r.ProjectPath, r.Command, err)
	}
}
