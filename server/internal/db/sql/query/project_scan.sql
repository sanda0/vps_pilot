-- name: GetNodeProjectScan :one
-- Defaults apply when the node has no row yet.
SELECT
  n.id AS node_id,
  CAST(COALESCE(s.auto_scan, 1) AS INTEGER) AS auto_scan,
  CAST(COALESCE(s.scan_interval, 900) AS INTEGER) AS scan_interval,
  s.last_scanned_at,
  CAST(COALESCE(s.errors, '[]') AS TEXT) AS errors
FROM nodes n
LEFT JOIN node_project_scan s ON s.node_id = n.id
WHERE n.id = ?;

-- name: UpdateNodeProjectScanSettings :exec
INSERT INTO node_project_scan (node_id, auto_scan, scan_interval)
VALUES (?, ?, ?)
ON CONFLICT (node_id) DO UPDATE SET
  auto_scan = excluded.auto_scan,
  scan_interval = excluded.scan_interval,
  updated_at = strftime('%s', 'now');

-- name: RecordNodeProjectScan :exec
INSERT INTO node_project_scan (node_id, last_scanned_at, errors)
VALUES (?, strftime('%s', 'now'), ?)
ON CONFLICT (node_id) DO UPDATE SET
  last_scanned_at = excluded.last_scanned_at,
  errors = excluded.errors;

-- name: ListNodeProjectPaths :many
SELECT id, node_id, path, created_at
FROM node_project_paths
WHERE node_id = ?
ORDER BY path;

-- name: AddNodeProjectPath :one
INSERT INTO node_project_paths (node_id, path)
VALUES (?, ?)
RETURNING id, node_id, path, created_at;

-- name: DeleteNodeProjectPath :execrows
DELETE FROM node_project_paths WHERE id = ? AND node_id = ?;
