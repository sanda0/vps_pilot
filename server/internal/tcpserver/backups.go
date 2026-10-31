package tcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

// BackupLostAfter is how long a backup may go without a result before it is
// marked lost. Agents keep running backups and queue their results while
// disconnected, so a late result still replaces "lost".
var BackupLostAfter = 24 * time.Hour

// Backups records the backups agents report in backup_runs. See "Back up"
// in docs/PROTOCOL.md.
type Backups struct {
	repo *db.Repo
}

// NewBackups marks backups that have had no result for BackupLostAfter as
// lost, now and every hour until ctx ends.
func NewBackups(ctx context.Context, repo *db.Repo) *Backups {
	b := &Backups{repo: repo}
	b.markLost(ctx)
	go func() {
		tick := time.NewTicker(time.Hour)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				b.markLost(ctx)
			}
		}
	}()
	return b
}

func (b *Backups) markLost(ctx context.Context) {
	before := time.Now().Add(-BackupLostAfter).Unix()
	if n, err := b.repo.Queries.MarkStaleBackupRunsLost(ctx, before); err != nil {
		fmt.Printf("mark backup runs lost: %v\n", err)
	} else if n > 0 {
		fmt.Printf("%d backup run(s) had no result for %v; marked lost\n", n, BackupLostAfter)
	}
}

var (
	backupTriggers = map[string]bool{"manual": true, "schedule": true}
	backupStatuses = map[string]bool{"ok": true, "failed": true, "rejected": true}
)

// HandleStarted records a backup_started.
func (b *Backups) HandleStarted(ctx context.Context, c *ConnContext, msg Msg) {
	var s BackupStarted
	if err := json.Unmarshal(msg.Data, &s); err != nil || s.RunID == "" {
		fmt.Printf("HandleBackupStarted: bad backup_started from node %d: %v\n", c.NodeID, err)
		return
	}
	if !backupTriggers[s.Trigger] {
		s.Trigger = "schedule"
	}
	err := b.repo.Queries.StartBackupRun(ctx, db.StartBackupRunParams{
		ID: s.RunID, NodeID: c.NodeID, ProjectPath: s.ProjectPath,
		TriggerKind: s.Trigger, RunAs: s.RunAs, StartedAt: s.StartedAt,
	})
	if err != nil {
		fmt.Printf("HandleBackupStarted: node %d run %s: %v\n", c.NodeID, s.RunID, err)
	}
}

// HandleResult records a backup_result.
func (b *Backups) HandleResult(ctx context.Context, c *ConnContext, msg Msg) {
	var r BackupResult
	if err := json.Unmarshal(msg.Data, &r); err != nil || r.RunID == "" {
		fmt.Printf("HandleBackupResult: bad backup_result from node %d: %v\n", c.NodeID, err)
		return
	}
	if !backupTriggers[r.Trigger] {
		r.Trigger = "schedule"
	}
	if !backupStatuses[r.Status] {
		r.Error = fmt.Sprintf("agent sent unknown status %q", r.Status)
		r.Status = "failed"
	}
	_, err := b.repo.Queries.FinishBackupRun(ctx, db.FinishBackupRunParams{
		ID: r.RunID, NodeID: c.NodeID, ProjectPath: r.ProjectPath, TriggerKind: r.Trigger,
		Status: r.Status, File: r.File, LocalPath: r.LocalPath, Size: r.Size,
		DatabaseType: r.Database, UploadStatus: r.Upload.Status, UploadKey: r.Upload.Key,
		UploadError: r.Upload.Error, Error: r.Error, DurationMs: r.DurationMS, StartedAt: r.StartedAt,
	})
	if err != nil {
		fmt.Printf("HandleBackupResult: node %d run %s: %v\n", c.NodeID, r.RunID, err)
	}
}
