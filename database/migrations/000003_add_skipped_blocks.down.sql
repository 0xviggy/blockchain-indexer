-- Rollback: Skipped Blocks
-- This removes the skipped_blocks table

BEGIN;

DROP TABLE IF EXISTS skipped_blocks CASCADE;

COMMIT;
