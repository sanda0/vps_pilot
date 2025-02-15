package test

import (
	"context"
	"testing"

	"github.com/sanda0/vps_pilot/internal/db"
)

func TestGetCPUStats(t *testing.T) {
	q := db.Queries{}

	// Test case 1
	_, err := q.GetCPUStats(context.Background(), db.GetCPUStatsParams{
		NodeID:    1,
		TimeRange: "1 day",
		CpuCount:  8,
	})
	if err != nil {
		panic(err)
	}

	t.Log("GetCPUStats test case 1")

}
