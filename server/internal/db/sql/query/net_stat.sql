-- name: InsertNetStats :exec
INSERT INTO net_stat (timestamp, node_id, sent, recv) VALUES (?, ?, ?, ?);

-- name: GetNetStats :many
select timestamp, sent, recv from net_stat ns
where node_id = ?
and timestamp >= strftime('%s', 'now') - ?;
-- name: GetNodeLastSeen :one
-- Unix time of the node's latest sys_stat (one net_stat row each), 0 if none.
select cast(coalesce(max(timestamp), 0) as integer) as last_seen from net_stat
where node_id = ?;
