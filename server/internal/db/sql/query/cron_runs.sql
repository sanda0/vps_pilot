-- name: InsertCronRun :execrows
-- From cron_result. Agents resend results they couldn't deliver, so a
-- repeated run_id is ignored.
INSERT INTO cron_runs (
  id, node_id, project_path, command, schedule, scheduled_at, status, exit_code,
  run_as, duration_ms, output, truncated, error
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO NOTHING;

-- name: PruneCronRuns :execrows
-- Keeps the newest runs of one job.
DELETE FROM cron_runs
WHERE cron_runs.node_id = sqlc.arg(node_id) AND cron_runs.project_path = sqlc.arg(project_path)
  AND cron_runs.command = sqlc.arg(command)
  AND cron_runs.id NOT IN (
    SELECT newest.id FROM cron_runs AS newest
    WHERE newest.node_id = sqlc.arg(node_id) AND newest.project_path = sqlc.arg(project_path)
      AND newest.command = sqlc.arg(command)
    ORDER BY newest.scheduled_at DESC, newest.received_at DESC, newest.rowid DESC
    LIMIT sqlc.arg(keep)
  );

-- name: ListCronRunsByJob :many
SELECT * FROM cron_runs
WHERE node_id = ? AND project_path = ? AND command = ?
ORDER BY scheduled_at DESC, received_at DESC, rowid DESC
LIMIT ? OFFSET ?;
