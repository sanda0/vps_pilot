-- name: InsertNetStats :exec
INSERT INTO net_stat (timestamp, node_id, sent, recv) VALUES (?, ?, ?, ?);

-- name: GetNetStats :many
select timestamp, sent, recv from net_stat ns
where node_id = ?
and timestamp >= strftime('%s', 'now') - ?;