-- name: InsertDiskStat :exec
INSERT INTO disk_stat (timestamp, node_id, value) VALUES (?, ?, ?);

-- name: GetDiskStats :many
select timestamp, value from disk_stat
where node_id = ?
and timestamp >= strftime('%s', 'now') - ?;
