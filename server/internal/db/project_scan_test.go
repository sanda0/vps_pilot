package db

import (
	"context"
	"testing"
)

func TestNodeProjectScan(t *testing.T) {
	ctx := context.Background()
	operationalDB, timeseriesDB, err := InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer operationalDB.Close()
	defer timeseriesDB.Close()
	q := New(operationalDB)
	mustExec(t, operationalDB, `INSERT INTO nodes (id, name) VALUES (1, 'web'), (2, 'db')`)

	// No row yet: defaults.
	got, err := q.GetNodeProjectScan(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.AutoScan != 1 || got.ScanInterval != 900 || got.LastScannedAt.Valid || got.Errors != "[]" {
		t.Fatalf("defaults: got %+v", got)
	}

	// Settings and scan results are written separately and don't clobber each other.
	if err := q.UpdateNodeProjectScanSettings(ctx, UpdateNodeProjectScanSettingsParams{NodeID: 1, AutoScan: 0, ScanInterval: 120}); err != nil {
		t.Fatal(err)
	}
	errs := `[{"path":"/opt/x","error":"outside"}]`
	if err := q.RecordNodeProjectScan(ctx, RecordNodeProjectScanParams{NodeID: 1, Errors: errs}); err != nil {
		t.Fatal(err)
	}
	got, _ = q.GetNodeProjectScan(ctx, 1)
	if got.AutoScan != 0 || got.ScanInterval != 120 || !got.LastScannedAt.Valid || got.Errors != errs {
		t.Fatalf("after update and record: got %+v", got)
	}
	if err := q.RecordNodeProjectScan(ctx, RecordNodeProjectScanParams{NodeID: 2, Errors: "[]"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := q.GetNodeProjectScan(ctx, 2); got.AutoScan != 1 || got.ScanInterval != 900 {
		t.Fatalf("record on a new row keeps default settings: got %+v", got)
	}

	// The interval stays within 1 minute .. 1 day.
	if err := q.UpdateNodeProjectScanSettings(ctx, UpdateNodeProjectScanSettingsParams{NodeID: 1, AutoScan: 1, ScanInterval: 30}); err == nil {
		t.Fatal("interval below 60s accepted")
	}

	// Manual paths: unique per node, deleted only through their own node.
	p, err := q.AddNodeProjectPath(ctx, AddNodeProjectPathParams{NodeID: 1, Path: "/srv/app"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.AddNodeProjectPath(ctx, AddNodeProjectPathParams{NodeID: 1, Path: "/srv/app"}); err == nil {
		t.Fatal("duplicate path accepted")
	}
	if n, _ := q.DeleteNodeProjectPath(ctx, DeleteNodeProjectPathParams{ID: p.ID, NodeID: 2}); n != 0 {
		t.Fatal("path deleted through another node")
	}
	if paths, _ := q.ListNodeProjectPaths(ctx, 1); len(paths) != 1 || paths[0].Path != "/srv/app" {
		t.Fatalf("paths: got %+v", paths)
	}

	// Deleting the node removes its settings and paths.
	mustExec(t, operationalDB, `DELETE FROM nodes WHERE id = 1`)
	var rows int
	operationalDB.QueryRow(`SELECT (SELECT COUNT(*) FROM node_project_scan WHERE node_id = 1) + (SELECT COUNT(*) FROM node_project_paths WHERE node_id = 1)`).Scan(&rows)
	if rows != 0 {
		t.Fatalf("%d rows left after deleting the node", rows)
	}
}
