-- Percent used of the node's root filesystem, one row per sys_stat.
CREATE TABLE IF NOT EXISTS disk_stat (
    timestamp INTEGER NOT NULL,
    node_id INTEGER NOT NULL,
    value REAL NOT NULL,
    PRIMARY KEY (timestamp, node_id)
);

CREATE INDEX IF NOT EXISTS idx_disk_stat_timestamp ON disk_stat(timestamp);
CREATE INDEX IF NOT EXISTS idx_disk_stat_node_time ON disk_stat(node_id, timestamp);
