-- Migration: 001_initial_schema.sql
-- Description: Create initial tables for multi-chain blockchain indexer
-- Date: 2025-11-14

BEGIN;

-- ============================================================================
-- CHAINS TABLE
-- ============================================================================
CREATE TABLE chains (
    chain_id INT PRIMARY KEY,
    chain_name VARCHAR(50) NOT NULL UNIQUE,
    rpc_url VARCHAR(255) NOT NULL,
    ws_url VARCHAR(255),
    block_time_seconds INT NOT NULL,
    finality_blocks INT NOT NULL DEFAULT 64,
    enabled BOOLEAN DEFAULT TRUE,
    last_indexed_block BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert supported chains
INSERT INTO chains (chain_id, chain_name, rpc_url, ws_url, block_time_seconds, finality_blocks) VALUES
    (1, 'Ethereum', 'https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY', 'wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY', 12, 64),
    (137, 'Polygon', 'https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY', 'wss://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY', 2, 256),
    (42161, 'Arbitrum', 'https://arb-mainnet.g.alchemy.com/v2/YOUR_KEY', 'wss://arb-mainnet.g.alchemy.com/v2/YOUR_KEY', 1, 64),
    (10, 'Optimism', 'https://opt-mainnet.g.alchemy.com/v2/YOUR_KEY', 'wss://opt-mainnet.g.alchemy.com/v2/YOUR_KEY', 2, 64),
    (8453, 'Base', 'https://base-mainnet.g.alchemy.com/v2/YOUR_KEY', 'wss://base-mainnet.g.alchemy.com/v2/YOUR_KEY', 2, 64);

CREATE INDEX idx_chains_enabled ON chains(enabled);

-- ============================================================================
-- BLOCKS TABLE (Partitioned by chain_id)
-- ============================================================================
CREATE TABLE blocks (
    chain_id INT NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    parent_hash VARCHAR(66) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    gas_used BIGINT,
    gas_limit BIGINT,
    base_fee_per_gas NUMERIC(78, 0), -- EIP-1559 base fee (wei)
    transaction_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, block_number),
    FOREIGN KEY (chain_id) REFERENCES chains(chain_id)
) PARTITION BY LIST (chain_id);

-- Create partitions for each chain
CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks FOR VALUES IN (137);
CREATE TABLE blocks_arbitrum PARTITION OF blocks FOR VALUES IN (42161);
CREATE TABLE blocks_optimism PARTITION OF blocks FOR VALUES IN (10);
CREATE TABLE blocks_base PARTITION OF blocks FOR VALUES IN (8453);

-- Indexes for blocks
CREATE INDEX idx_blocks_eth_number ON blocks_eth(block_number DESC);
CREATE INDEX idx_blocks_eth_hash ON blocks_eth(block_hash);
CREATE INDEX idx_blocks_eth_timestamp ON blocks_eth(timestamp DESC);

CREATE INDEX idx_blocks_polygon_number ON blocks_polygon(block_number DESC);
CREATE INDEX idx_blocks_polygon_hash ON blocks_polygon(block_hash);

CREATE INDEX idx_blocks_arbitrum_number ON blocks_arbitrum(block_number DESC);
CREATE INDEX idx_blocks_arbitrum_hash ON blocks_arbitrum(block_hash);

CREATE INDEX idx_blocks_optimism_number ON blocks_optimism(block_number DESC);
CREATE INDEX idx_blocks_optimism_hash ON blocks_optimism(block_hash);

CREATE INDEX idx_blocks_base_number ON blocks_base(block_number DESC);
CREATE INDEX idx_blocks_base_hash ON blocks_base(block_hash);

-- ============================================================================
-- TRANSACTIONS TABLE (Partitioned by chain_id)
-- ============================================================================
CREATE TABLE transactions (
    chain_id INT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    tx_index INT NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    value NUMERIC(78, 0), -- Support up to 2^256
    gas_limit BIGINT, -- Gas limit set by sender (from tx.Gas())
    gas_price BIGINT,
    gas_used BIGINT, -- Actual gas consumed (from receipt.GasUsed)
    input_data BYTEA,
    nonce BIGINT, -- Transaction nonce (from tx.Nonce())
    status INTEGER DEFAULT 1, -- 0=failed, 1=success (from receipt.Status)
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, tx_hash),
    FOREIGN KEY (chain_id, block_number) REFERENCES blocks(chain_id, block_number) ON DELETE CASCADE
) PARTITION BY LIST (chain_id);

-- Create partitions
CREATE TABLE transactions_eth PARTITION OF transactions FOR VALUES IN (1);
CREATE TABLE transactions_polygon PARTITION OF transactions FOR VALUES IN (137);
CREATE TABLE transactions_arbitrum PARTITION OF transactions FOR VALUES IN (42161);
CREATE TABLE transactions_optimism PARTITION OF transactions FOR VALUES IN (10);
CREATE TABLE transactions_base PARTITION OF transactions FOR VALUES IN (8453);

-- Indexes for transactions
CREATE INDEX idx_tx_eth_block ON transactions_eth(block_number DESC);
CREATE INDEX idx_tx_eth_from ON transactions_eth(from_address);
CREATE INDEX idx_tx_eth_to ON transactions_eth(to_address);

CREATE INDEX idx_tx_polygon_block ON transactions_polygon(block_number DESC);
CREATE INDEX idx_tx_polygon_from ON transactions_polygon(from_address);
CREATE INDEX idx_tx_polygon_to ON transactions_polygon(to_address);

CREATE INDEX idx_tx_arbitrum_block ON transactions_arbitrum(block_number DESC);
CREATE INDEX idx_tx_arbitrum_from ON transactions_arbitrum(from_address);

CREATE INDEX idx_tx_optimism_block ON transactions_optimism(block_number DESC);
CREATE INDEX idx_tx_optimism_from ON transactions_optimism(from_address);

CREATE INDEX idx_tx_base_block ON transactions_base(block_number DESC);
CREATE INDEX idx_tx_base_from ON transactions_base(from_address);

-- ============================================================================
-- EVENTS TABLE (Partitioned by chain_id)
-- ============================================================================
CREATE TABLE events (
    chain_id INT NOT NULL,
    id BIGSERIAL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    log_index INT NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    event_signature VARCHAR(66) NOT NULL,
    topic1 VARCHAR(66),
    topic2 VARCHAR(66),
    topic3 VARCHAR(66),
    data BYTEA,
    decoded_data JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, id),
    FOREIGN KEY (chain_id, tx_hash) REFERENCES transactions(chain_id, tx_hash) ON DELETE CASCADE
) PARTITION BY LIST (chain_id);

-- Create partitions
CREATE TABLE events_eth PARTITION OF events FOR VALUES IN (1);
CREATE TABLE events_polygon PARTITION OF events FOR VALUES IN (137);
CREATE TABLE events_arbitrum PARTITION OF events FOR VALUES IN (42161);
CREATE TABLE events_optimism PARTITION OF events FOR VALUES IN (10);
CREATE TABLE events_base PARTITION OF events FOR VALUES IN (8453);

-- Indexes for events
CREATE INDEX idx_events_eth_contract ON events_eth(contract_address);
CREATE INDEX idx_events_eth_signature ON events_eth(event_signature);
CREATE INDEX idx_events_eth_block ON events_eth(block_number DESC);
CREATE INDEX idx_events_eth_decoded ON events_eth USING GIN(decoded_data);

CREATE INDEX idx_events_polygon_contract ON events_polygon(contract_address);
CREATE INDEX idx_events_polygon_signature ON events_polygon(event_signature);
CREATE INDEX idx_events_polygon_block ON events_polygon(block_number DESC);

CREATE INDEX idx_events_arbitrum_contract ON events_arbitrum(contract_address);
CREATE INDEX idx_events_arbitrum_signature ON events_arbitrum(event_signature);

CREATE INDEX idx_events_optimism_contract ON events_optimism(contract_address);
CREATE INDEX idx_events_optimism_signature ON events_optimism(event_signature);

CREATE INDEX idx_events_base_contract ON events_base(contract_address);
CREATE INDEX idx_events_base_signature ON events_base(event_signature);

-- ============================================================================
-- CHECKPOINTS TABLE (For tracking ingestion progress)
-- ============================================================================
CREATE TABLE checkpoints (
    service_name VARCHAR(50) NOT NULL,
    chain_id INT NOT NULL,
    last_processed_block BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (service_name, chain_id),
    FOREIGN KEY (chain_id) REFERENCES chains(chain_id)
);

-- Initialize checkpoints for ingester service
INSERT INTO checkpoints (service_name, chain_id, last_processed_block) 
SELECT 'ingester', chain_id, 0 FROM chains WHERE enabled = TRUE;

CREATE INDEX idx_checkpoints_service ON checkpoints(service_name);

-- ============================================================================
-- REORG_EVENTS TABLE (Track blockchain reorganizations)
-- ============================================================================
CREATE TABLE reorg_events (
    id SERIAL PRIMARY KEY,
    chain_id INT NOT NULL,
    detected_at TIMESTAMP DEFAULT NOW(),
    rollback_from_block BIGINT NOT NULL,
    rollback_to_block BIGINT NOT NULL,
    blocks_affected INT NOT NULL,
    handled BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (chain_id) REFERENCES chains(chain_id)
);

CREATE INDEX idx_reorg_chain ON reorg_events(chain_id);
CREATE INDEX idx_reorg_handled ON reorg_events(handled);

-- ============================================================================
-- FUNCTIONS & TRIGGERS
-- ============================================================================

-- Update updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_chains_updated_at
    BEFORE UPDATE ON chains
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_checkpoints_updated_at
    BEFORE UPDATE ON checkpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- VIEWS
-- ============================================================================

-- View for latest block per chain
CREATE VIEW latest_blocks AS
SELECT 
    c.chain_id,
    c.chain_name,
    MAX(b.block_number) as latest_block,
    MAX(b.timestamp) as latest_timestamp
FROM chains c
LEFT JOIN blocks b ON c.chain_id = b.chain_id
WHERE c.enabled = TRUE
GROUP BY c.chain_id, c.chain_name;

-- View for indexing status
CREATE VIEW indexing_status AS
SELECT 
    c.chain_id,
    c.chain_name,
    c.enabled,
    COALESCE(MAX(b.block_number), 0) as latest_indexed_block,
    ch.last_processed_block as checkpoint_block,
    COALESCE(MAX(b.block_number), 0) - ch.last_processed_block as lag_blocks
FROM chains c
LEFT JOIN blocks b ON c.chain_id = b.chain_id
LEFT JOIN checkpoints ch ON c.chain_id = ch.chain_id AND ch.service_name = 'ingester'
WHERE c.enabled = TRUE
GROUP BY c.chain_id, c.chain_name, c.enabled, ch.last_processed_block;

COMMIT;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO indexer;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO indexer;
