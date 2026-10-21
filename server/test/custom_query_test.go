package test

import (
	"context"
	"testing"

	"github.com/sanda0/vps_pilot/internal/db"
)

func TestGetCPUStats(t *testing.T) {
	operationalDB, timeseriesDB, err := db.InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer operationalDB.Close()
	defer timeseriesDB.Close()

	q := db.New(timeseriesDB)
	if _, err := q.GetCPUStats(context.Background(), db.GetCPUStatsParams{
		NodeID:    1,
		TimeRange: 60 * 24,
		CpuCount:  8,
	}); err != nil {
		t.Fatal(err)
	}
}
