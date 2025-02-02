-- name: CreateNode :one
INSERT INTO nodes (name, ip, cpu_cores, cpu_ghz, memory, disk)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
-- name: GetNode :one
SELECT *
FROM nodes
WHERE id = $1;
-- name: GetNodes :many
SELECT *
FROM nodes
LIMIT $1 OFFSET $2;
-- name: UpdateNode :one
UPDATE nodes
SET name = $1,
  ip = $2,
  cpu_cores = $3,
  cpu_ghz = $4,
  memory = $5,
  disk = $6,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $7
RETURNING *;
-- name: DeleteNode :execrows
DELETE FROM nodes
WHERE id = $1;