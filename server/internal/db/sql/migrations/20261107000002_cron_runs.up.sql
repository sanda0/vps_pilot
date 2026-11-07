-- Cron job runs the agents report, one row per scheduled run. The agent
-- runs the jobs itself; the hub only keeps the newest runs of each job.
CREATE TABLE cron_runs (
  id TEXT PRIMARY KEY,                       -- run_id (UUID, from the agent)
  node_id INTEGER NOT NULL,
  project_path TEXT NOT NULL,
  command TEXT NOT NULL,                     -- command name
  schedule TEXT NOT NULL DEFAULT '',         -- as written in the config
  scheduled_at INTEGER NOT NULL,             -- the minute it was due, Unix seconds
  status TEXT NOT NULL,                      -- exited, timeout, skipped, rejected, failed
  exit_code INTEGER,
  run_as TEXT NOT NULL DEFAULT '',
  duration_ms INTEGER NOT NULL DEFAULT 0,
  output TEXT NOT NULL DEFAULT '',           -- last 64 KiB of stdout and stderr
  truncated INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  received_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE
);

CREATE INDEX idx_cron_runs_job ON cron_runs (node_id, project_path, command, scheduled_at DESC);
