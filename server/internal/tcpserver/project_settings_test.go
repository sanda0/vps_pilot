package tcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

// eventually retries cond until it holds or 2s pass.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); !cond(); {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestProjectSettingsRescanAndScanErrors(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	q := h.repo.Queries

	agent := h.dial()
	agent.send(MsgEnroll, h.joinSecret(nil, time.Hour), SystemInfo{Hostname: "web-1"})
	var creds EnrolledPayload
	json.Unmarshal(agent.expect(MsgEnrolled).Data, &creds)
	agent.expect(MsgSysStat)

	// Defaults right after the handshake. paths must be [] (not null) for the agent.
	if got := string(agent.expect(MsgProjectSettings).Data); got != `{"auto_scan":true,"scan_interval":900,"paths":[]}` {
		t.Fatalf("settings after handshake: %s", got)
	}
	eventually(t, "registration", func() bool { _, ok := h.registry.Get(creds.NodeID); return ok })

	// Changed settings are pushed to the live agent.
	q.UpdateNodeProjectScanSettings(ctx, db.UpdateNodeProjectScanSettingsParams{NodeID: creds.NodeID, AutoScan: 0, ScanInterval: 3600})
	q.AddNodeProjectPath(ctx, db.AddNodeProjectPathParams{NodeID: creds.NodeID, Path: "/srv/legacy"})
	go PushProjectSettings(ctx, h.registry, creds.NodeID)
	if got := string(agent.expect(MsgProjectSettings).Data); got != `{"auto_scan":false,"scan_interval":3600,"paths":["/srv/legacy"]}` {
		t.Fatalf("pushed settings: %s", got)
	}

	go RequestProjectRescan(h.registry, creds.NodeID)
	agent.expect(MsgRescanProjects)
	if ok, _ := RequestProjectRescan(h.registry, creds.NodeID+1); ok {
		t.Fatal("rescan reported sent to a node that isn't connected")
	}

	// A report stores its errors and the scan time.
	agent.send(MsgProjects, "", ProjectSyncPayload{
		Projects: []ProjectConfig{{Name: "app", Path: "/srv/app"}},
		Errors:   []ProjectError{{Path: "/opt/x", Error: "outside the agent's project directories"}},
	})
	eventually(t, "scan record", func() bool {
		scan, err := q.GetNodeProjectScan(ctx, creds.NodeID)
		return err == nil && scan.LastScannedAt.Valid && strings.Contains(scan.Errors, `"/opt/x"`)
	})
	if n, _ := q.CountProjectsByNode(ctx, creds.NodeID); n != 1 {
		t.Fatalf("%d projects stored, want 1", n)
	}
}
