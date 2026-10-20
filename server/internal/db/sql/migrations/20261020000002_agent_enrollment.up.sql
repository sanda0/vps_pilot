-- Nodes are identified by agent_id (issued at enrollment), not by IP.
-- SQLite can't drop the UNIQUE constraint on ip in place, so rebuild the table.
-- The migration runner executes this with foreign_keys OFF, so DROP TABLE does
-- not cascade into node_sys_info / node_disk_info / projects.
CREATE TABLE nodes_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  ip TEXT NOT NULL DEFAULT '',
  agent_id TEXT UNIQUE,
  key_hash TEXT,
  last_seen_at INTEGER,
  created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

INSERT INTO nodes_new (id, name, ip, created_at, updated_at)
SELECT id, name, ip, created_at, updated_at FROM nodes;

DROP TABLE nodes;

ALTER TABLE nodes_new RENAME TO nodes;

-- One-time join tokens. Only the SHA-256 of the secret is stored.
-- node_id set = re-enroll that existing node instead of creating a new one.
CREATE TABLE agent_join_tokens (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  token_hash TEXT NOT NULL UNIQUE,
  node_id INTEGER,
  created_by INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  used_at INTEGER,
  used_by_node_id INTEGER,
  created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE
);
