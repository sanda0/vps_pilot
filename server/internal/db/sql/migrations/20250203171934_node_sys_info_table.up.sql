CREATE TABLE IF NOT EXISTS node_sys_info (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_id INTEGER NOT NULL,
  os TEXT,
  platform TEXT,
  platform_version TEXT,
  kernel_version TEXT,
  cpus INTEGER,
  total_memory REAL,
  created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE
);