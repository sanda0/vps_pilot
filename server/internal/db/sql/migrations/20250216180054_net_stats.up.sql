CREATE TABLE IF NOT EXISTS net_stat (
    time TIMESTAMPTZ NOT NULL,
    node_id INT NOT NULL,
    sent BIGINT NOT NULL,  -- Bytes sent
    recv BIGINT NOT NULL,  -- Bytes received
    PRIMARY KEY (time, node_id)
);

DO $$ 
BEGIN 
    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.hypertables 
        WHERE hypertable_name = 'net_stat'
    ) THEN 
        PERFORM create_hypertable('net_stat', 'time'); 
    END IF;
END $$;
