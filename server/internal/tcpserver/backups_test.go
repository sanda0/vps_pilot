package tcpserver

import (
	"context"
	"encoding/json"
	"testing"

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
