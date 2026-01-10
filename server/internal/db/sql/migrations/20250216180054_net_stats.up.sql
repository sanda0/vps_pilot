CREATE TABLE IF NOT EXISTS net_stat (
    timestamp INTEGER NOT NULL,
    node_id INTEGER NOT NULL,
    sent INTEGER NOT NULL,  -- Bytes sent
    recv INTEGER NOT NULL,  -- Bytes received
    PRIMARY KEY (timestamp, node_id)
);

-- Create index for time-based queries
CREATE INDEX IF NOT EXISTS idx_net_stat_timestamp ON net_stat(timestamp);
CREATE INDEX IF NOT EXISTS idx_net_stat_node_time ON net_stat(node_id, timestamp);
