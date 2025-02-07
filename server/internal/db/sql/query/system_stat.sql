-- name: InsertSystemStats :exec
INSERT INTO system_stats (time, node_id, stat_type, cpu_id, value)
SELECT 
    unnest($1::timestamptz[]),
    unnest($2::int[]),
    unnest($3::text[]),
    unnest($4::int[]),
    unnest($5::double precision[]);
