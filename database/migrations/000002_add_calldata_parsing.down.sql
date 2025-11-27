-- Rollback: Calldata Parsing
-- This removes tables created in 000002_add_calldata_parsing.up.sql

BEGIN;

DROP TABLE IF EXISTS revert_reasons_eth CASCADE;
DROP TABLE IF EXISTS revert_reasons_polygon CASCADE;
DROP TABLE IF EXISTS revert_reasons_arbitrum CASCADE;
DROP TABLE IF EXISTS revert_reasons_optimism CASCADE;
DROP TABLE IF EXISTS revert_reasons_base CASCADE;
DROP TABLE IF EXISTS revert_reasons CASCADE;

DROP TABLE IF EXISTS parsed_calldata_eth CASCADE;
DROP TABLE IF EXISTS parsed_calldata_polygon CASCADE;
DROP TABLE IF EXISTS parsed_calldata_arbitrum CASCADE;
DROP TABLE IF EXISTS parsed_calldata_optimism CASCADE;
DROP TABLE IF EXISTS parsed_calldata_base CASCADE;
DROP TABLE IF EXISTS parsed_calldata CASCADE;

DROP TABLE IF EXISTS internal_transactions_eth CASCADE;
DROP TABLE IF EXISTS internal_transactions_polygon CASCADE;
DROP TABLE IF EXISTS internal_transactions_arbitrum CASCADE;
DROP TABLE IF EXISTS internal_transactions_optimism CASCADE;
DROP TABLE IF EXISTS internal_transactions_base CASCADE;
DROP TABLE IF EXISTS internal_transactions CASCADE;

COMMIT;
