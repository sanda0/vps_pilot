-- name: InsertSystemStats :exec
INSERT INTO system_stats (time, node_id, stat_type, cpu_id, value)
SELECT 
    unnest($1::timestamptz[]),
    unnest($2::int[]),
    unnest($3::text[]),
    unnest($4::int[]),
    unnest($5::double precision[]);

-- name: GetSystemStats :many
select time,value from system_stats ss 
where node_id = $1 and stat_type = $2
and cpu_id = $3
and time >= now() - interval '' || $4 || '';