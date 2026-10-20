DROP TABLE agent_join_tokens;

CREATE TABLE nodes_old (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  ip TEXT NOT NULL UNIQUE,
  created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

INSERT OR IGNORE INTO nodes_old (id, name, ip, created_at, updated_at)
SELECT id, name, ip, created_at, updated_at FROM nodes;

DROP TABLE nodes;

ALTER TABLE nodes_old RENAME TO nodes;
