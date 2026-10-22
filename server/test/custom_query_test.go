package test

import (
	"context"
	"testing"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

func TestGetCPUStats(t *testing.T) {
	operationalDB, timeseriesDB, err := db.InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer operationalDB.Close()
	defer timeseriesDB.Close()

	// Insert out of order, two CPUs per timestamp.
	now := time.Now().Unix()
	for _, ts := range []int64{now - 10, now - 30, now - 20, now - 40} {
		for cpu := 1; cpu <= 2; cpu++ {
			if _, err := timeseriesDB.Exec(
				`INSERT INTO system_stats (timestamp, node_id, stat_type, cpu_id, value) VALUES (?, 1, 'cpu', ?, 5)`,
				ts, cpu,
			); err != nil {
				t.Fatal(err)
			}
		}
	}

	q := db.New(timeseriesDB)
	rows, err := q.GetCPUStats(context.Background(), db.GetCPUStatsParams{
		NodeID:    1,
		TimeRange: 60 * 24,
		CpuCount:  8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	for i := 1; i < len(rows); i++ {
		if rows[i-1]["timestamp"].(int64) >= rows[i]["timestamp"].(int64) {
			t.Fatalf("rows not in time order: %v then %v", rows[i-1]["timestamp"], rows[i]["timestamp"])
		}
	}
}
