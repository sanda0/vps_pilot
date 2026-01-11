package db

import (
	"context"
	"fmt"
	"time"
)

type GetCPUStatsParams struct {
	NodeID    int32  `json:"node_id"`
	TimeRange string `json:"time_range"` // in seconds
	CpuCount  int32  `json:"cpu_count"`
}

func (q *Queries) GetCPUStats(ctx context.Context, arg GetCPUStatsParams) ([]map[string]interface{}, error) {
	// Calculate cutoff timestamp
	cutoffTime := time.Now().Unix() - int64(mustParseInt(arg.TimeRange))

	// Query to get all CPU stats within time range
	// We'll need to pivot the data in Go since SQLite doesn't have crosstab
	query := `
		SELECT timestamp, cpu_id, value
		FROM system_stats
		WHERE node_id = ? 
		  AND stat_type = 'cpu'
		  AND timestamp >= ?
		ORDER BY timestamp, cpu_id
	`

	rows, err := q.db.QueryContext(ctx, query, arg.NodeID, cutoffTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Map to group by timestamp
	timeMap := make(map[int64]map[string]interface{})

	for rows.Next() {
		var timestamp int64
		var cpuID int32
		var value float64

		if err := rows.Scan(&timestamp, &cpuID, &value); err != nil {
			return nil, err
		}

		// Initialize map for this timestamp if it doesn't exist
		if _, exists := timeMap[timestamp]; !exists {
			timeMap[timestamp] = make(map[string]interface{})
			timeMap[timestamp]["timestamp"] = timestamp
			timeMap[timestamp]["time"] = time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
		}

		// Add CPU value
		timeMap[timestamp][fmt.Sprintf("cpu_%d", cpuID)] = value
	}

	// Convert map to slice
	result := make([]map[string]interface{}, 0, len(timeMap))
	for _, record := range timeMap {
		result = append(result, record)
	}

	return result, nil
}

// Helper function to parse time range string to int
func mustParseInt(s string) int {
	var val int
	fmt.Sscanf(s, "%d", &val)
	return val
}
