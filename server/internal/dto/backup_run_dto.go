package dto

import "github.com/sanda0/vps_pilot/internal/db"

// BackupRunDto is one backup of a project, scheduled or started from the
// dashboard.
type BackupRunDto struct {
	ID           string `json:"id"`
	NodeID       int64  `json:"node_id"`
	ProjectPath  string `json:"project_path"`
	Trigger      string `json:"trigger"`  // manual or schedule
	Username     string `json:"username"` // who pressed Back up now
	Status       string `json:"status"`   // running, ok, failed, rejected, lost
	RunAs        string `json:"run_as"`
	File         string `json:"file"`
	LocalPath    string `json:"local_path"`
	Size         int64  `json:"size"`
	Database     string `json:"database"`
	UploadStatus string `json:"upload_status"` // ok, failed or skipped
	UploadKey    string `json:"upload_key"`
	UploadError  string `json:"upload_error"`
	Error        string `json:"error"`
	DurationMs   int64  `json:"duration_ms"`
	StartedAt    int64  `json:"started_at"`
	FinishedAt   *int64 `json:"finished_at"`
}

func NewBackupRunDto(r db.BackupRun) BackupRunDto {
	out := BackupRunDto{
		ID: r.ID, NodeID: r.NodeID, ProjectPath: r.ProjectPath, Trigger: r.TriggerKind,
		Username: r.Username, Status: r.Status, RunAs: r.RunAs, File: r.File, LocalPath: r.LocalPath,
		Size: r.Size, Database: r.DatabaseType, UploadStatus: r.UploadStatus, UploadKey: r.UploadKey,
		UploadError: r.UploadError, Error: r.Error, DurationMs: r.DurationMs, StartedAt: r.StartedAt,
	}
	if r.FinishedAt.Valid {
		out.FinishedAt = &r.FinishedAt.Int64
	}
	return out
}
