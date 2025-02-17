DO $$
BEGIN
    -- Check if the hypertable exists before attempting to drop it
    IF EXISTS (
        SELECT 1
        FROM timescaledb_information.hypertables
        WHERE hypertable_name = 'net_stat'
    ) THEN
        -- Dropping the hypertable (equivalent to dropping the table)
        EXECUTE 'DROP TABLE IF EXISTS net_stat CASCADE';
    END IF;
END $$;