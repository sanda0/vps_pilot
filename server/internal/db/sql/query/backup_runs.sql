-- name: CreateBackupRun :one
-- Back up now, from the dashboard, before backup_run is sent.
INSERT INTO backup_runs (id, node_id, project_path, trigger_kind, user_id, username)
VALUES (?, ?, ?, 'manual', ?, ?)
RETURNING *;

-- name: StartBackupRun :exec
-- From backup_started. A run started from the dashboard already has a row;
-- the agent's report adds who it runs as. Only the node that owns a run
-- can change it, and a finished run is left alone.
INSERT INTO backup_runs (id, node_id, project_path, trigger_kind, run_as, started_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET run_as = excluded.run_as, started_at = excluded.started_at
WHERE backup_runs.node_id = excluded.node_id AND backup_runs.status = 'running';

-- name: FinishBackupRun :execrows
-- From backup_result. Agents queue results while disconnected, so a result
-- can come after the run was marked lost; it replaces that. A repeated
-- result is ignored.
INSERT INTO backup_runs (
  id, node_id, project_path, trigger_kind, status, file, local_path, size, database_type,
  upload_status, upload_key, upload_error, error, duration_ms, started_at, finished_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%s', 'now'))
ON CONFLICT (id) DO UPDATE SET
  status = excluded.status, file = excluded.file, local_path = excluded.local_path,
  size = excluded.size, database_type = excluded.database_type,
  upload_status = excluded.upload_status, upload_key = excluded.upload_key,
  upload_error = excluded.upload_error, error = excluded.error,
  duration_ms = excluded.duration_ms, finished_at = excluded.finished_at
WHERE backup_runs.node_id = excluded.node_id AND backup_runs.status IN ('running', 'lost');

-- name: MarkStaleBackupRunsLost :execrows
UPDATE backup_runs
SET status = 'lost', error = 'no result from the agent', finished_at = strftime('%s', 'now')
WHERE status = 'running' AND started_at < ?;

-- name: GetBackupRun :one
SELECT * FROM backup_runs WHERE id = ?;

-- name: ListBackupRunsByProject :many
SELECT * FROM backup_runs
WHERE node_id = ? AND project_path = ?
ORDER BY started_at DESC, rowid DESC
LIMIT ? OFFSET ?;
