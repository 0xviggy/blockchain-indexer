-- Rollback: Initial Schema
-- This removes all tables created in 000001_initial_schema.up.sql

BEGIN;

-- Drop views
DROP VIEW IF EXISTS indexing_status CASCADE;
DROP VIEW IF EXISTS latest_blocks CASCADE;

-- Drop tables (reverse order of creation due to foreign keys)
DROP TABLE IF EXISTS checkpoints CASCADE;

-- Drop partitioned tables and their partitions
DROP TABLE IF EXISTS events_eth CASCADE;
DROP TABLE IF EXISTS events_polygon CASCADE;
DROP TABLE IF EXISTS events_arbitrum CASCADE;
DROP TABLE IF EXISTS events_optimism CASCADE;
DROP TABLE IF EXISTS events_base CASCADE;
DROP TABLE IF EXISTS events CASCADE;

DROP TABLE IF EXISTS transactions_eth CASCADE;
DROP TABLE IF EXISTS transactions_polygon CASCADE;
DROP TABLE IF EXISTS transactions_arbitrum CASCADE;
DROP TABLE IF EXISTS transactions_optimism CASCADE;
DROP TABLE IF EXISTS transactions_base CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;

DROP TABLE IF EXISTS blocks_eth CASCADE;
DROP TABLE IF EXISTS blocks_polygon CASCADE;
DROP TABLE IF EXISTS blocks_arbitrum CASCADE;
DROP TABLE IF EXISTS blocks_optimism CASCADE;
DROP TABLE IF EXISTS blocks_base CASCADE;
DROP TABLE IF EXISTS blocks CASCADE;

DROP TABLE IF EXISTS protocol_signatures CASCADE;
DROP TABLE IF EXISTS chains CASCADE;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column CASCADE;

COMMIT;
