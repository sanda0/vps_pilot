// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: net_stat.sql

package db

import (
	"context"
	"database/sql"
	"time"
)

const getNetStats = `-- name: GetNetStats :many
select time,sent,recv from net_stat ns
where node_id = $1
and time >= now() -  ($2||'')::interval
`

type GetNetStatsParams struct {
	NodeID  int32          `json:"node_id"`
	Column2 sql.NullString `json:"column_2"`
}

type GetNetStatsRow struct {
	Time time.Time `json:"time"`
	Sent int64     `json:"sent"`
	Recv int64     `json:"recv"`
}

func (q *Queries) GetNetStats(ctx context.Context, arg GetNetStatsParams) ([]GetNetStatsRow, error) {
	rows, err := q.query(ctx, q.getNetStatsStmt, getNetStats, arg.NodeID, arg.Column2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetNetStatsRow
	for rows.Next() {
		var i GetNetStatsRow
		if err := rows.Scan(&i.Time, &i.Sent, &i.Recv); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertNetStats = `-- name: InsertNetStats :exec
INSERT INTO net_stat (time, node_id, sent, recv) VALUES ($1, $2, $3, $4)
`

type InsertNetStatsParams struct {
	Time   time.Time `json:"time"`
	NodeID int32     `json:"node_id"`
	Sent   int64     `json:"sent"`
	Recv   int64     `json:"recv"`
}

func (q *Queries) InsertNetStats(ctx context.Context, arg InsertNetStatsParams) error {
	_, err := q.exec(ctx, q.insertNetStatsStmt, insertNetStats,
		arg.Time,
		arg.NodeID,
		arg.Sent,
		arg.Recv,
	)
	return err
}
