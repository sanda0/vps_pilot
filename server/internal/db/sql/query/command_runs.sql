-- name: CreateCommandRun :one
INSERT INTO command_runs (id, node_id, project_path, project_name, command, command_line, user_id, username)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: FinishCommandRun :execrows
-- Only a running run can finish, so a late or repeated command_exit is ignored.
UPDATE command_runs
SET status = ?, exit_code = ?, run_as = ?, error = ?, truncated = ?, duration_ms = ?,
    finished_at = strftime('%s', 'now')
WHERE id = ? AND status = 'running';

-- name: MarkAllRunningCommandRunsLost :execrows
-- At hub start: no agent connection survived the restart.
UPDATE command_runs
SET status = 'lost', error = 'the hub restarted during the run', finished_at = strftime('%s', 'now')
WHERE status = 'running';

-- name: AddCommandRunOutput :exec
INSERT OR IGNORE INTO command_run_output (run_id, seq, stream, data)
VALUES (?, ?, ?, ?);

-- name: GetCommandRun :one
SELECT * FROM command_runs WHERE id = ?;

-- name: ListCommandRunOutput :many
SELECT seq, stream, data FROM command_run_output
WHERE run_id = ?
ORDER BY seq;

-- name: ListCommandRunsByNode :many
SELECT * FROM command_runs
WHERE node_id = ?
ORDER BY started_at DESC, rowid DESC
LIMIT ? OFFSET ?;

-- name: ListCommandRunsByProject :many
SELECT * FROM command_runs
WHERE node_id = ? AND project_path = ?
ORDER BY started_at DESC, rowid DESC
LIMIT ? OFFSET ?;
