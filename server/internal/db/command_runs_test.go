package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestCommandRuns(t *testing.T) {
	ctx := context.Background()
	operationalDB, timeseriesDB, err := InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer operationalDB.Close()
	defer timeseriesDB.Close()
	q := New(operationalDB)
	mustExec(t, operationalDB, `INSERT INTO nodes (id, name) VALUES (1, 'web')`)

	for _, id := range []string{"r1", "r2"} {
		if _, err := q.CreateCommandRun(ctx, CreateCommandRunParams{
			ID: id, NodeID: 1, ProjectPath: "/var/www/shop", ProjectName: "shop",
			Command: "build", CommandLine: "npm run build", Username: "admin",
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Output is kept in seq order; a repeated seq is ignored.
	for _, o := range []AddCommandRunOutputParams{
		{RunID: "r1", Seq: 1, Stream: "stderr", Data: "b"},
		{RunID: "r1", Seq: 0, Stream: "stdout", Data: "a"},
		{RunID: "r1", Seq: 0, Stream: "stdout", Data: "dup"},
	} {
		if err := q.AddCommandRunOutput(ctx, o); err != nil {
			t.Fatal(err)
		}
	}
	out, err := q.ListCommandRunOutput(ctx, "r1")
	if err != nil || len(out) != 2 || out[0].Data != "a" || out[1].Stream != "stderr" {
		t.Fatalf("output: %+v, %v", out, err)
	}

	// A run finishes once.
	finish := FinishCommandRunParams{Status: "exited", ExitCode: sql.NullInt64{Int64: 0, Valid: true}, RunAs: "deploy", ID: "r1"}
	for want := int64(1); want >= 0; want-- {
		if n, err := q.FinishCommandRun(ctx, finish); err != nil || n != want {
			t.Fatalf("finish: %d rows, %v; want %d", n, err, want)
		}
	}

	// At hub start, runs still marked running are lost.
	if n, err := q.MarkAllRunningCommandRunsLost(ctx); err != nil || n != 1 {
		t.Fatalf("mark lost: %d, %v", n, err)
	}
	runs, err := q.ListCommandRunsByProject(ctx, ListCommandRunsByProjectParams{NodeID: 1, ProjectPath: "/var/www/shop", Limit: 10})
	if err != nil || len(runs) != 2 {
		t.Fatalf("list: %+v, %v", runs, err)
	}
	status := map[string]string{runs[0].ID: runs[0].Status, runs[1].ID: runs[1].Status}
	if status["r1"] != "exited" || status["r2"] != "lost" {
		t.Fatalf("statuses: %v", status)
	}

	// Deleting the node removes its runs and their output.
	mustExec(t, operationalDB, `DELETE FROM nodes WHERE id = 1`)
	if out, _ := q.ListCommandRunOutput(ctx, "r1"); len(out) != 0 {
		t.Fatalf("output left after node delete: %+v", out)
	}
}
