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
-- name: GetNodesWithSysInfo :many
SELECT n.id,
  n.name,
  n.ip,
  nsi.os,
  nsi.platform,
  nsi.platform_version,
  nsi.kernel_version,
  nsi.cpus,
  nsi.total_memory
FROM nodes as n
  JOIN node_sys_info as nsi ON n.id = nsi.node_id
WHERE n.name LIKE '%' || $1 || '%'
  OR n.ip LIKE '%' || $1 || '%'
  OR nsi.os LIKE '%' || $1 || '%'
  OR nsi.platform LIKE '%' || $1 || '%'
  OR nsi.platform_version LIKE '%' || $1 || '%'
  OR nsi.kernel_version LIKE '%' || $1 || '%'
LIMIT $2 OFFSET $3;

-- name: GetNodeWithSysInfo :one
SELECT n.id,
  n.name,
  n.ip,
  nsi.os,
  nsi.platform,
  nsi.platform_version,
  nsi.kernel_version,
  nsi.cpus,
  nsi.total_memory
FROM nodes as n
  JOIN node_sys_info as nsi ON n.id = nsi.node_id
WHERE n.id = $1;
-- name: UpdateNodeName :exec
UPDATE nodes
SET name = $2
WHERE id = $1;
--######################################################################################
------------------------------------sys info-------------------------------------------
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
--######################################################################################
------------------------------------disk info-------------------------------------------
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