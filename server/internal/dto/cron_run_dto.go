package dto

import "github.com/sanda0/vps_pilot/internal/db"

// CronRunDto is one scheduled run of a project's cron job, as its agent
// reported it.
type CronRunDto struct {
	ID          string `json:"id"`
	Command     string `json:"command"`
	Schedule    string `json:"schedule"`
	ScheduledAt int64  `json:"scheduled_at"` // the minute it was due, Unix seconds
	Status      string `json:"status"`       // exited, timeout, skipped, rejected, failed
	ExitCode    *int64 `json:"exit_code"`
	RunAs       string `json:"run_as"`
	DurationMs  int64  `json:"duration_ms"`
	Output      string `json:"output"` // last 64 KiB of stdout and stderr
	Truncated   bool   `json:"truncated"`
	Error       string `json:"error"`
	ReceivedAt  int64  `json:"received_at"`
}

func NewCronRunDto(r db.CronRun) CronRunDto {
	out := CronRunDto{
		ID: r.ID, Command: r.Command, Schedule: r.Schedule, ScheduledAt: r.ScheduledAt,
		Status: r.Status, RunAs: r.RunAs, DurationMs: r.DurationMs, Output: r.Output,
		Truncated: r.Truncated != 0, Error: r.Error, ReceivedAt: r.ReceivedAt,
	}
	if r.ExitCode.Valid {
		out.ExitCode = &r.ExitCode.Int64
	}
	return out
}

// CronJobDto is a cron job a project declares, with its latest run.
type CronJobDto struct {
	Command  string      `json:"command"`
	Schedule string      `json:"schedule"`
	LastRun  *CronRunDto `json:"last_run"` // null until the agent reports one
}
