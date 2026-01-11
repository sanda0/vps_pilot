CREATE TABLE IF NOT EXISTS alerts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_id INTEGER NOT NULL,
  metric TEXT NOT NULL,
  duration INTEGER NOT NULL,
  threshold REAL DEFAULT 0,
  net_rece_threshold REAL DEFAULT 0,
  net_send_threshold REAL DEFAULT 0,
  email TEXT,
  discord_webhook TEXT,
  slack_webhook TEXT,
  is_active INTEGER DEFAULT 1,
  created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);