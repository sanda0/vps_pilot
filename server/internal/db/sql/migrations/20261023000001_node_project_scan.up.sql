-- How and when each node's projects are searched (set in the dashboard), and
-- the last result the agent reported. A node without a row uses the defaults.
CREATE TABLE node_project_scan (
  node_id INTEGER PRIMARY KEY,
  auto_scan INTEGER NOT NULL DEFAULT 1,
  scan_interval INTEGER NOT NULL DEFAULT 900 CHECK (scan_interval BETWEEN 60 AND 86400),
  last_scanned_at INTEGER,
  errors TEXT NOT NULL DEFAULT '[]',  -- JSON [{"path": "...", "error": "..."}] from the last report
  updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE
);

-- Project directories added by hand in the dashboard. The agent only accepts
-- them inside its --projects-dir unless started with --allow-any-project-path.
CREATE TABLE node_project_paths (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_id INTEGER NOT NULL,
  path TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
  UNIQUE (node_id, path)
);
