package tcpserver

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

func TestProjectsKeepTheBackupSchedule(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureBackups)
	// As a v0.5 agent sends it.
	var payload ProjectSyncPayload
	json.Unmarshal([]byte(`{"projects": [{"name": "shop", "path": "/srv/shop", "backups": {
		"env_file": ".env", "dir": ["storage/app"], "schedule": "weekly", "at": "02:30",
		"local_dir": "../backups/shop", "retention_days": 14,
		"s3": {"access_key": "S3_KEY", "secret_key": "S3_SECRET", "bucket": "b", "region": "r",
		       "endpoint": "https://e", "prefix": "shop/", "path_style": true}}}]}`), &payload)
	agent.send(MsgProjects, "", payload)

	var got *ProjectBackup
	eventually(t, "project stored", func() bool {
		rows, err := h.repo.Queries.ListProjectsByNode(context.Background(), db.ListProjectsByNodeParams{NodeID: nodeID, Limit: 10})
		if err != nil || len(rows) != 1 {
			return false
		}
		return json.Unmarshal([]byte(rows[0].Backups), &got) == nil
	})
	if got == nil || got.Schedule != "weekly" || got.At != "02:30" || got.LocalDir != "../backups/shop" ||
		*got.RetentionDays != 14 || got.S3 == nil || got.S3.SecretKey != "S3_SECRET" || !got.S3.PathStyle ||
		got.S3.RetentionDays != nil {
		t.Fatalf("backups %+v, s3 %+v", got, got.S3)
	}
}

func TestBackupReportsAreRecorded(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureBackups)
	other, _ := h.agentWith(FeatureBackups)
	q, ctx := h.repo.Queries, context.Background()
	run := func(id string) db.BackupRun {
		r, _ := q.GetBackupRun(ctx, id)
		return r
	}

	agent.send(MsgBackupStarted, "", BackupStarted{RunID: "s1", ProjectPath: "/srv/shop", Trigger: "schedule", StartedAt: 1793415600, RunAs: "deploy"})
	h.synced(agent)
	if r := run("s1"); r.Status != "running" || r.RunAs != "deploy" || r.NodeID != nodeID || r.TriggerKind != "schedule" {
		t.Fatalf("started %+v", r)
	}
	ok := BackupResult{
		RunID: "s1", ProjectPath: "/srv/shop", Trigger: "schedule", Status: "ok", StartedAt: 1793415600,
		DurationMS: 84210, File: "shop-20261031T030000Z.zip", LocalPath: "/srv/backups/shop-20261031T030000Z.zip",
		Size: 52428800, Database: "mysql", Upload: BackupUpload{Status: "failed", Key: "shop/x.zip", Error: "AccessDenied"},
	}
	// Another node can't finish this node's run.
	other.send(MsgBackupResult, "", BackupResult{RunID: "s1", Status: "failed", Error: "not yours"})
	h.synced(other)
	agent.send(MsgBackupResult, "", ok)
	// A repeated result is ignored.
	agent.send(MsgBackupResult, "", BackupResult{RunID: "s1", Status: "failed", Error: "again"})
	h.synced(agent)
	if r := run("s1"); r.Status != "ok" || r.Size != 52428800 || r.DatabaseType != "mysql" || r.UploadError != "AccessDenied" ||
		r.RunAs != "deploy" || !r.FinishedAt.Valid || r.Error != "" {
		t.Fatalf("finished %+v", r)
	}

	// A rejected backup has no backup_started; an unknown status is a failure.
	agent.send(MsgBackupResult, "", BackupResult{RunID: "r1", ProjectPath: "/srv/blog", Trigger: "manual", Status: "rejected", Error: "config.vpspilot.json declares no backups"})
	agent.send(MsgBackupResult, "", BackupResult{RunID: "u1", ProjectPath: "/srv/blog", Trigger: "odd", Status: "bored"})
	h.synced(agent)
	if r := run("r1"); r.Status != "rejected" || r.TriggerKind != "manual" || r.Error == "" {
		t.Fatalf("rejected %+v", r)
	}
	if r := run("u1"); r.Status != "failed" || r.TriggerKind != "schedule" || r.Error != `agent sent unknown status "bored"` {
		t.Fatalf("unknown status %+v", r)
	}
}

func TestLateBackupResultReplacesLost(t *testing.T) {
	h := newTestHub(t)
	agent, _ := h.agentWith(FeatureBackups)
	q, ctx := h.repo.Queries, context.Background()
	old := time.Now().Add(-BackupLostAfter - time.Minute).Unix()
	agent.send(MsgBackupStarted, "", BackupStarted{RunID: "a", ProjectPath: "/srv/shop", Trigger: "schedule", StartedAt: old})
	agent.send(MsgBackupStarted, "", BackupStarted{RunID: "b", ProjectPath: "/srv/shop", Trigger: "schedule", StartedAt: time.Now().Unix()})
	h.synced(agent)

	h.backups.markLost(ctx)
	if r, _ := q.GetBackupRun(ctx, "a"); r.Status != "lost" || r.Error != "no result from the agent" {
		t.Fatalf("stale run %+v", r)
	}
	if r, _ := q.GetBackupRun(ctx, "b"); r.Status != "running" {
		t.Fatalf("recent run %+v", r)
	}
	agent.send(MsgBackupResult, "", BackupResult{RunID: "a", ProjectPath: "/srv/shop", Trigger: "schedule", Status: "ok", StartedAt: old})
	h.synced(agent)
	if r, _ := q.GetBackupRun(ctx, "a"); r.Status != "ok" || r.Error != "" {
		t.Fatalf("late result %+v", r)
	}
}
