package dto

import "github.com/sanda0/vps_pilot/internal/db"

// CommandRunDto is one entry of the command run audit log.
type CommandRunDto struct {
	ID          string `json:"id"`
	NodeID      int64  `json:"node_id"`
	ProjectPath string `json:"project_path"`
	ProjectName string `json:"project_name"`
	Command     string `json:"command"`
	CommandLine string `json:"command_line"`
	Username    string `json:"username"`
	Status      string `json:"status"` // running, exited, timeout, cancelled, rejected, failed, lost
	ExitCode    *int64 `json:"exit_code"`
	RunAs       string `json:"run_as"`
	Error       string `json:"error"`
	Truncated   bool   `json:"truncated"`
	DurationMs  int64  `json:"duration_ms"`
	StartedAt   int64  `json:"started_at"`
	FinishedAt  *int64 `json:"finished_at"`
}

// CommandOutputDto is one chunk of a run's output.
type CommandOutputDto struct {
	Seq    int64  `json:"seq"`
	Stream string `json:"stream"` // stdout or stderr
	Data   string `json:"data"`
}

// StartCommandRunRequest names a command from the project's config.
type StartCommandRunRequest struct {
	Command string `json:"command" binding:"required"`
}

func NewCommandRunDto(r db.CommandRun) CommandRunDto {
	out := CommandRunDto{
		ID: r.ID, NodeID: r.NodeID, ProjectPath: r.ProjectPath, ProjectName: r.ProjectName,
		Command: r.Command, CommandLine: r.CommandLine, Username: r.Username, Status: r.Status,
		RunAs: r.RunAs, Error: r.Error, Truncated: r.Truncated != 0, DurationMs: r.DurationMs,
		StartedAt: r.StartedAt,
	}
	if r.ExitCode.Valid {
		out.ExitCode = &r.ExitCode.Int64
	}
	if r.FinishedAt.Valid {
		out.FinishedAt = &r.FinishedAt.Int64
	}
	return out
}

func NewCommandOutputDtos(rows []db.ListCommandRunOutputRow) []CommandOutputDto {
	out := make([]CommandOutputDto, len(rows))
	for i, r := range rows {
		out[i] = CommandOutputDto{Seq: r.Seq, Stream: r.Stream, Data: r.Data}
	}
	return out
}
