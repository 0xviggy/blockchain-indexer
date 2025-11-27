-- Migration: 003_add_skipped_blocks.sql
-- Description: Track blocks that were skipped due to errors or unsupported features
-- Date: 2025-11-26

BEGIN;

-- ============================================================================
-- SKIPPED BLOCKS TABLE
-- ============================================================================
CREATE TABLE skipped_blocks (
    chain_id INT NOT NULL,
    block_number BIGINT NOT NULL,
    skip_reason VARCHAR(255) NOT NULL,
    error_message TEXT,
    skipped_at TIMESTAMP DEFAULT NOW(),
    retry_count INT DEFAULT 0,
    last_retry_at TIMESTAMP,
    PRIMARY KEY (chain_id, block_number),
    FOREIGN KEY (chain_id) REFERENCES chains(chain_id)
);

CREATE INDEX idx_skipped_blocks_reason ON skipped_blocks(skip_reason);
CREATE INDEX idx_skipped_blocks_chain_time ON skipped_blocks(chain_id, skipped_at DESC);

COMMIT;
