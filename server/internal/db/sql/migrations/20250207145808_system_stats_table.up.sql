CREATE TABLE IF NOT EXISTS system_stats(
    timestamp INTEGER NOT NULL,
    node_id INTEGER NOT NULL,
    stat_type TEXT NOT NULL CHECK (stat_type IN ('cpu', 'mem')),
    cpu_id INTEGER,
    value REAL NOT NULL,
    PRIMARY KEY (timestamp, node_id, stat_type, cpu_id)
);

-- Create index for time-based queries
CREATE INDEX IF NOT EXISTS idx_system_stats_timestamp ON system_stats(timestamp);
CREATE INDEX IF NOT EXISTS idx_system_stats_node_time ON system_stats(node_id, timestamp);