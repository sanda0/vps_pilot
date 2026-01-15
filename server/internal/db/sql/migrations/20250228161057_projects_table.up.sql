CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL,
    description TEXT,
    node_id INTEGER NOT NULL,
    
    -- GitHub/Repository info
    repo_url TEXT,
    branch TEXT DEFAULT 'main',
    
    -- Deployment
    deploy_path TEXT NOT NULL,
    
    -- Metadata
    status TEXT DEFAULT 'inactive' CHECK(status IN ('inactive', 'cloning', 'active', 'error')),
    last_deployed_at INTEGER,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_projects_node_id ON projects(node_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
