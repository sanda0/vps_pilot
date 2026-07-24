ALTER TABLE users ADD COLUMN github_token TEXT;

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY NOT NULL,
    node_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    tech TEXT NOT NULL DEFAULT '[]',
    commands TEXT NOT NULL DEFAULT '[]',
    logs TEXT NOT NULL DEFAULT '[]',
    backups TEXT NOT NULL DEFAULT '{}',
    discovered_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    UNIQUE(node_id, path)
);
