CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE TABLE IF NOT EXISTS system_stats(
    time TIMESTAMPTZ NOT NULL,
    node_id INT NOT NULL,
    stat_type TEXT NOT NULL CHECK (stat_type IN ('cpu', 'mem')),
    cpu_id INT,
    value DOUBLE PRECISION,
    PRIMARY KEY (time, node_id, stat_type, cpu_id)
);
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM timescaledb_information.hypertables
    WHERE hypertable_name = 'system_stats'
) THEN PERFORM create_hypertable('system_stats', 'time');
END IF;
END $$;