CREATE TABLE IF NOT EXISTS node_disk_info (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_id INTEGER NOT NULL,
  device TEXT,
  mount_point TEXT,
  fstype TEXT,
  total REAL,
  used REAL,
  created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE
);