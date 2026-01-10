-- name: InsertSystemStats :exec
INSERT INTO system_stats (timestamp, node_id, stat_type, cpu_id, value)
VALUES (?, ?, ?, ?, ?);

-- name: GetSystemStats :many
select timestamp, value from system_stats ss 
where node_id = ? and stat_type = ?
and cpu_id = ?
and timestamp >= strftime('%s', 'now') - ?;