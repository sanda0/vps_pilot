-- Backups the agents report, one row per run: scheduled ones and those
-- started from the dashboard. The backup itself stays on the server and in
-- the project's bucket; this is the record.
CREATE TABLE backup_runs (
  id TEXT PRIMARY KEY,                       -- run_id (UUID, from the hub or the agent)
  node_id INTEGER NOT NULL,
  project_path TEXT NOT NULL,
  trigger_kind TEXT NOT NULL,                -- manual or schedule
  user_id INTEGER,                           -- who pressed Back up now
  username TEXT NOT NULL DEFAULT '',         -- kept if the user is deleted
  status TEXT NOT NULL DEFAULT 'running',    -- running, ok, failed, rejected, lost
  run_as TEXT NOT NULL DEFAULT '',
  file TEXT NOT NULL DEFAULT '',
  local_path TEXT NOT NULL DEFAULT '',
  size INTEGER NOT NULL DEFAULT 0,
  database_type TEXT NOT NULL DEFAULT '',    -- mysql, postgres, sqlite or empty
  upload_status TEXT NOT NULL DEFAULT '',    -- ok, failed or skipped
  upload_key TEXT NOT NULL DEFAULT '',
  upload_error TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  duration_ms INTEGER NOT NULL DEFAULT 0,
  started_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  finished_at INTEGER,
  FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
);

CREATE INDEX idx_backup_runs_project ON backup_runs (node_id, project_path, started_at DESC);
CREATE INDEX idx_backup_runs_running ON backup_runs (status) WHERE status = 'running';
