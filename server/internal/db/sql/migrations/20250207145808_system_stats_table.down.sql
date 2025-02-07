DO $$
BEGIN
    -- Check if the hypertable exists before attempting to drop it
    IF EXISTS (
        SELECT 1
        FROM timescaledb_information.hypertables
        WHERE hypertable_name = 'system_stats'
    ) THEN
        -- Dropping the hypertable (equivalent to dropping the table)
        EXECUTE 'DROP TABLE IF EXISTS system_stats CASCADE';
    END IF;
END $$;

-- Optional: Remove the TimescaleDB extension if it's no longer needed
DROP EXTENSION IF EXISTS timescaledb CASCADE;
