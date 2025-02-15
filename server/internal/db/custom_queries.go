package db

import (
	"context"
	"fmt"
)

type GetCPUStatsParams struct {
	NodeID    int32  `json:"node_id"`
	TimeRange string `json:"time_range"`
	CpuCount  int32  `json:"cpu_count"`
}

func (q *Queries) GetCPUStats(ctx context.Context, arg GetCPUStatsParams) ([]map[string]interface{}, error) {

	customColSelect := ""
	for i := 1; i <= int(arg.CpuCount); i++ {
		if i == int(arg.CpuCount) {
			customColSelect += fmt.Sprintf("MAX(COALESCE(cpu_%d, 0)) AS cpu_%d ", i, i)
			break
		}
		customColSelect += fmt.Sprintf("MAX(COALESCE(cpu_%d, 0)) AS cpu_%d, ", i, i)
	}

	customColCT := ""
	for i := 1; i <= int(arg.CpuCount); i++ {
		if i == int(arg.CpuCount) {
			customColCT += fmt.Sprintf("cpu_%d FLOAT ", i)
			break
		}
		customColCT += fmt.Sprintf("cpu_%d FLOAT, ", i)
	}

	query := `
	
	SELECT
    to_char(time, 'YYYY-MM-DD HH24:MI:SS') AS formatted_time,
		%s
FROM crosstab(
    $$SELECT time, cpu_id::text, value
      FROM system_stats
      WHERE node_id = %d AND stat_type = 'cpu'
        AND time >= now() - INTERVAL '%s'
      ORDER BY 1, 2$$,
    $$SELECT DISTINCT cpu_id::text
      FROM system_stats
      WHERE node_id = %d AND stat_type = 'cpu'
      ORDER BY 1$$
) AS ct(
    time TIMESTAMP,
    %s
)
GROUP BY formatted_time
ORDER BY formatted_time;
	`
	query = fmt.Sprintf(query, customColSelect, arg.NodeID, arg.TimeRange, arg.NodeID, customColCT)

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}

	for rows.Next() {
		var formattedTime string
		cpuData := make([]interface{}, arg.CpuCount)
		scanArgs := make([]interface{}, arg.CpuCount+1)
		scanArgs[0] = &formattedTime
		for i := range cpuData {
			scanArgs[i+1] = &cpuData[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		// Create a map and insert time & CPU values dynamically
		record := make(map[string]interface{})
		record["time"] = formattedTime
		for i := 1; i <= int(arg.CpuCount); i++ {
			record[fmt.Sprintf("cpu_%d", i)] = cpuData[i-1]
		}

		result = append(result, record)
	}

	return result, nil
}
