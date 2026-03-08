-- Drop old projects table and its indexes
DROP INDEX IF EXISTS idx_projects_status;
DROP INDEX IF EXISTS idx_projects_node_id;
DROP TABLE IF EXISTS projects;

-- Recreate projects table for agent-discovered model
CREATE TABLE IF NOT EXISTS projects (
    id          TEXT    PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    node_id     INTEGER NOT NULL,
    name        TEXT    NOT NULL,
    path        TEXT    NOT NULL,       -- absolute path on disk where config.vpspilot.json was found
    tech        TEXT    NOT NULL DEFAULT '[]',    -- JSON array  e.g. ["laravel","react","mysql"]
    commands    TEXT    NOT NULL DEFAULT '[]',    -- JSON array  e.g. [{"name":"build","command":"npm run build"}]
    logs        TEXT    NOT NULL DEFAULT '[]',    -- JSON array  e.g. ["storage/logs"]
    backups     TEXT    NOT NULL DEFAULT '{}',    -- JSON object (backup config)
    discovered_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at   INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    UNIQUE (node_id, path)   -- a node cannot have two projects at the same path
);

CREATE INDEX IF NOT EXISTS idx_projects_node_id ON projects(node_id);
