-- Audit log of project commands started from the dashboard, one row per run.
-- The hub sends only the command name; command_line is what the project's
-- config said when the run started, kept for the record.
CREATE TABLE command_runs (
  id TEXT PRIMARY KEY,                       -- run_id sent to the agent (UUID)
  node_id INTEGER NOT NULL,
  project_path TEXT NOT NULL,
  project_name TEXT NOT NULL,
  command TEXT NOT NULL,                     -- command name
  command_line TEXT NOT NULL DEFAULT '',
  user_id INTEGER,                           -- dashboard user who started it
  username TEXT NOT NULL,                    -- kept if the user is deleted
  status TEXT NOT NULL DEFAULT 'running',    -- running, exited, timeout, cancelled, rejected, failed, lost
  exit_code INTEGER,
  run_as TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  truncated INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  started_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  finished_at INTEGER,
  FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
);

CREATE INDEX idx_command_runs_node ON command_runs (node_id, started_at DESC);
CREATE INDEX idx_command_runs_project ON command_runs (node_id, project_path, started_at DESC);

-- Output chunks of a run, in the order the agent numbered them (seq).
CREATE TABLE command_run_output (
  run_id TEXT NOT NULL,
  seq INTEGER NOT NULL,
  stream TEXT NOT NULL,                      -- stdout or stderr
  data TEXT NOT NULL,
  PRIMARY KEY (run_id, seq),
  FOREIGN KEY (run_id) REFERENCES command_runs (id) ON DELETE CASCADE
);
