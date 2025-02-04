-- name: CreateNode :one
INSERT INTO nodes (name, ip)
VALUES ($1, $2)
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
SET name = $2,
  ip = $3,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;
-- name: DeleteNode :execrows
DELETE FROM nodes
WHERE id = $1;
-- name: GetNodeByIP :one
SELECT *
FROM nodes
WHERE ip = $1;
-- name: AddNodeSysInfo :one
INSERT INTO node_sys_info (
    node_id,
    os,
    platform,
    platform_version,
    kernel_version,
    cpus,
    total_memory
  )
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;
-- name: GetNodeSysInfoByNodeID :one
SELECT *
FROM node_sys_info
WHERE node_id = $1;
-- name: UpdateNodeSysInfo :one
UPDATE node_sys_info
SET os = $2,
  platform = $3,
  platform_version = $4,
  kernel_version = $5,
  cpus = $6,
  total_memory = $7,
  updated_at = CURRENT_TIMESTAMP
WHERE node_id = $1
RETURNING *;
-- name: AddNodeDiskInfo :one
INSERT INTO node_disk_info (
    node_id,
    device,
    mount_point,
    fstype,
    total,
    used
  )
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
-- name: GetNodeDiskInfoByNodeID :many
SELECT *
FROM node_disk_info
WHERE node_id = $1;
-- name: UpdateNodeDiskInfo :one
UPDATE node_disk_info
SET device = $2,
  mount_point = $3,
  fstype = $4,
  total = $5,
  used = $6,
  updated_at = CURRENT_TIMESTAMP
WHERE node_id = $1
RETURNING *;
