-- name: InsertNetStats :exec
INSERT INTO net_stat (time, node_id, sent, recv) VALUES ($1, $2, $3, $4);

-- name: GetNetStats :many
select time,sent,recv from net_stat ns
where node_id = $1
and time >= now() -  ($2||'')::interval;