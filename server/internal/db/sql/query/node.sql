-- name: CreateEnrolledNode :one
INSERT INTO nodes (name, ip, agent_id, key_hash, last_seen_at)
VALUES (?, ?, ?, ?, strftime('%s', 'now'))
RETURNING *;
-- name: SetNodeCredentials :execrows
UPDATE nodes
SET agent_id = ?,
  key_hash = ?,
  updated_at = strftime('%s', 'now')
WHERE id = ?;
-- name: ClearNodeKey :execrows
UPDATE nodes
SET key_hash = NULL,
  updated_at = strftime('%s', 'now')
WHERE id = ?;
-- name: GetNodeByAgentID :one
SELECT *
FROM nodes
WHERE agent_id = ?;
-- name: TouchNode :exec
UPDATE nodes
SET ip = ?,
  last_seen_at = strftime('%s', 'now')
WHERE id = ?;
-- name: GetNode :one
SELECT *
FROM nodes
WHERE id = ?;
-- name: GetNodes :many
SELECT *
FROM nodes
LIMIT ? OFFSET ?;
-- name: UpdateNode :one
UPDATE nodes
SET name = ?,
  ip = ?,
  updated_at = strftime('%s', 'now')
WHERE id = ?
RETURNING *;
-- name: DeleteNode :execrows
DELETE FROM nodes
WHERE id = ?;
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
WHERE n.name LIKE '%' || ? || '%'
  OR n.ip LIKE '%' || ? || '%'
  OR nsi.os LIKE '%' || ? || '%'
  OR nsi.platform LIKE '%' || ? || '%'
  OR nsi.platform_version LIKE '%' || ? || '%'
  OR nsi.kernel_version LIKE '%' || ? || '%'
LIMIT ? OFFSET ?;

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
WHERE n.id = ?;
-- name: UpdateNodeName :exec
UPDATE nodes
SET name = ?
WHERE id = ?;
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
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;
-- name: GetNodeSysInfoByNodeID :one
SELECT *
FROM node_sys_info
WHERE node_id = ?;
-- name: UpdateNodeSysInfo :one
UPDATE node_sys_info
SET os = ?,
  platform = ?,
  platform_version = ?,
  kernel_version = ?,
  cpus = ?,
  total_memory = ?,
  updated_at = strftime('%s', 'now')
WHERE node_id = ?
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
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;
-- name: GetNodeDiskInfoByNodeID :many
SELECT *
FROM node_disk_info
WHERE node_id = ?;
-- name: UpdateNodeDiskInfo :one
UPDATE node_disk_info
SET device = ?,
  mount_point = ?,
  fstype = ?,
  total = ?,
  used = ?,
  updated_at = strftime('%s', 'now')
WHERE node_id = ?
RETURNING *;