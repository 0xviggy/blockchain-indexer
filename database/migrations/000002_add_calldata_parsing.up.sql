-- Migration: 002_add_calldata_parsing.sql
-- Description: Add tables for calldata parsing, internal transactions, and revert reasons
-- Date: 2025-11-14

BEGIN;

-- ============================================================================
-- PARSED_CALLDATA TABLE
-- Stores decoded function calls from transaction input data
-- ============================================================================
CREATE TABLE parsed_calldata (
    chain_id INT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    function_signature VARCHAR(10) NOT NULL, -- 4-byte selector (0x12345678)
    function_name VARCHAR(100),
    protocol VARCHAR(50), -- 'uniswap-v2', 'uniswap-v3', 'layerzero', etc.
    decoded_params JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, tx_hash),
    FOREIGN KEY (chain_id, tx_hash) REFERENCES transactions(chain_id, tx_hash) ON DELETE CASCADE
) PARTITION BY LIST (chain_id);

-- Create partitions
CREATE TABLE parsed_calldata_eth PARTITION OF parsed_calldata FOR VALUES IN (1);
CREATE TABLE parsed_calldata_polygon PARTITION OF parsed_calldata FOR VALUES IN (137);
CREATE TABLE parsed_calldata_arbitrum PARTITION OF parsed_calldata FOR VALUES IN (42161);
CREATE TABLE parsed_calldata_optimism PARTITION OF parsed_calldata FOR VALUES IN (10);
CREATE TABLE parsed_calldata_base PARTITION OF parsed_calldata FOR VALUES IN (8453);

-- Indexes for calldata
CREATE INDEX idx_calldata_eth_signature ON parsed_calldata_eth(function_signature);
CREATE INDEX idx_calldata_eth_protocol ON parsed_calldata_eth(protocol);
CREATE INDEX idx_calldata_eth_params ON parsed_calldata_eth USING GIN(decoded_params);

CREATE INDEX idx_calldata_polygon_signature ON parsed_calldata_polygon(function_signature);
CREATE INDEX idx_calldata_polygon_protocol ON parsed_calldata_polygon(protocol);

CREATE INDEX idx_calldata_arbitrum_signature ON parsed_calldata_arbitrum(function_signature);
CREATE INDEX idx_calldata_arbitrum_protocol ON parsed_calldata_arbitrum(protocol);

CREATE INDEX idx_calldata_optimism_signature ON parsed_calldata_optimism(function_signature);
CREATE INDEX idx_calldata_optimism_protocol ON parsed_calldata_optimism(protocol);

CREATE INDEX idx_calldata_base_signature ON parsed_calldata_base(function_signature);
CREATE INDEX idx_calldata_base_protocol ON parsed_calldata_base(protocol);

-- ============================================================================
-- INTERNAL_TRANSACTIONS TABLE
-- Tracks contract-to-contract calls (internal transactions)
-- ============================================================================
CREATE TABLE internal_transactions (
    chain_id INT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    internal_tx_index INT NOT NULL, -- Order within the parent transaction
    call_type VARCHAR(20) NOT NULL, -- 'call', 'delegatecall', 'staticcall', 'create', 'create2'
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    value NUMERIC(78, 0),
    gas BIGINT,
    gas_used BIGINT,
    input BYTEA,
    output BYTEA,
    success BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, tx_hash, internal_tx_index),
    FOREIGN KEY (chain_id, tx_hash) REFERENCES transactions(chain_id, tx_hash) ON DELETE CASCADE
) PARTITION BY LIST (chain_id);

-- Create partitions
CREATE TABLE internal_transactions_eth PARTITION OF internal_transactions FOR VALUES IN (1);
CREATE TABLE internal_transactions_polygon PARTITION OF internal_transactions FOR VALUES IN (137);
CREATE TABLE internal_transactions_arbitrum PARTITION OF internal_transactions FOR VALUES IN (42161);
CREATE TABLE internal_transactions_optimism PARTITION OF internal_transactions FOR VALUES IN (10);
CREATE TABLE internal_transactions_base PARTITION OF internal_transactions FOR VALUES IN (8453);

-- Indexes for internal transactions
CREATE INDEX idx_internal_tx_eth_from ON internal_transactions_eth(from_address);
CREATE INDEX idx_internal_tx_eth_to ON internal_transactions_eth(to_address);
CREATE INDEX idx_internal_tx_eth_type ON internal_transactions_eth(call_type);

CREATE INDEX idx_internal_tx_polygon_from ON internal_transactions_polygon(from_address);
CREATE INDEX idx_internal_tx_polygon_to ON internal_transactions_polygon(to_address);

CREATE INDEX idx_internal_tx_arbitrum_from ON internal_transactions_arbitrum(from_address);
CREATE INDEX idx_internal_tx_arbitrum_to ON internal_transactions_arbitrum(to_address);

CREATE INDEX idx_internal_tx_optimism_from ON internal_transactions_optimism(from_address);
CREATE INDEX idx_internal_tx_optimism_to ON internal_transactions_optimism(to_address);

CREATE INDEX idx_internal_tx_base_from ON internal_transactions_base(from_address);
CREATE INDEX idx_internal_tx_base_to ON internal_transactions_base(to_address);

-- ============================================================================
-- REVERT_REASONS TABLE
-- Stores error messages from failed transactions
-- ============================================================================
CREATE TABLE revert_reasons (
    chain_id INT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    revert_reason TEXT,
    error_signature VARCHAR(10), -- 4-byte error selector (for custom errors)
    error_name VARCHAR(100),
    error_params JSONB,
    extracted_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, tx_hash),
    FOREIGN KEY (chain_id, tx_hash) REFERENCES transactions(chain_id, tx_hash) ON DELETE CASCADE
) PARTITION BY LIST (chain_id);

-- Create partitions
CREATE TABLE revert_reasons_eth PARTITION OF revert_reasons FOR VALUES IN (1);
CREATE TABLE revert_reasons_polygon PARTITION OF revert_reasons FOR VALUES IN (137);
CREATE TABLE revert_reasons_arbitrum PARTITION OF revert_reasons FOR VALUES IN (42161);
CREATE TABLE revert_reasons_optimism PARTITION OF revert_reasons FOR VALUES IN (10);
CREATE TABLE revert_reasons_base PARTITION OF revert_reasons FOR VALUES IN (8453);

-- Indexes for revert reasons
CREATE INDEX idx_revert_eth_reason ON revert_reasons_eth(revert_reason);
CREATE INDEX idx_revert_eth_error_sig ON revert_reasons_eth(error_signature);
CREATE INDEX idx_revert_polygon_reason ON revert_reasons_polygon(revert_reason);
CREATE INDEX idx_revert_arbitrum_reason ON revert_reasons_arbitrum(revert_reason);
CREATE INDEX idx_revert_optimism_reason ON revert_reasons_optimism(revert_reason);
CREATE INDEX idx_revert_base_reason ON revert_reasons_base(revert_reason);

-- ============================================================================
-- PROTOCOL_SIGNATURES TABLE
-- Registry of known function signatures for various protocols
-- ============================================================================
CREATE TABLE protocol_signatures (
    signature VARCHAR(10) PRIMARY KEY, -- 4-byte selector
    function_name VARCHAR(100) NOT NULL,
    protocol VARCHAR(50) NOT NULL,
    abi TEXT NOT NULL, -- Full function ABI for decoding
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert common DEX signatures
INSERT INTO protocol_signatures (signature, function_name, protocol, abi, description) VALUES
    -- Uniswap V2
    ('0x38ed1739', 'swapExactTokensForTokens', 'uniswap-v2', 
     'swapExactTokensForTokens(uint256,uint256,address[],address,uint256)',
     'Swap exact amount of tokens for tokens'),
    ('0x7ff36ab5', 'swapExactETHForTokens', 'uniswap-v2',
     'swapExactETHForTokens(uint256,address[],address,uint256)',
     'Swap exact ETH for tokens'),
    ('0x18cbafe5', 'swapExactTokensForETH', 'uniswap-v2',
     'swapExactTokensForETH(uint256,uint256,address[],address,uint256)',
     'Swap exact tokens for ETH'),
    
    -- Uniswap V3
    ('0x414bf389', 'exactInputSingle', 'uniswap-v3',
     'exactInputSingle((address,address,uint24,address,uint256,uint256,uint256,uint160))',
     'Swap with single pool'),
    ('0xc04b8d59', 'exactInput', 'uniswap-v3',
     'exactInput((bytes,address,uint256,uint256,uint256))',
     'Swap with multiple pools'),
    
    -- LayerZero Bridge
    ('0x1a0a6e', 'send', 'layerzero',
     'send(uint16,bytes,bytes,address,address,bytes)',
     'Send cross-chain message'),
    
    -- OpenSea Seaport
    ('0xfb0f3ee1', 'fulfillOrder', 'seaport',
     'fulfillOrder((address,address,(uint8,address,uint256,uint256,uint256)[],(uint8,address,uint256,uint256,uint256,address)[],uint8,uint256,uint256,bytes32,uint256,bytes32,uint256),bytes32)',
     'Fulfill NFT order'),
    
    -- Across Bridge
    ('0x3687011a', 'deposit', 'across',
     'deposit(address,address,uint256,uint256,uint64,uint32,uint256)',
     'Deposit for cross-chain transfer'),
    
    -- Stargate Bridge
    ('0x0f5287b0', 'swap', 'stargate',
     'swap(uint16,uint256,uint256,address,uint256,uint256,tuple,bytes,bytes)',
     'Cross-chain token swap'),
    
    -- 1inch Aggregator
    ('0x7c025200', 'swap', '1inch',
     'swap(address,tuple,bytes,bytes)',
     '1inch swap aggregator'),
    
    -- Curve Finance
    ('0x3df02124', 'exchange', 'curve',
     'exchange(int128,int128,uint256,uint256)',
     'Exchange tokens in pool'),
    
    -- Aave V3
    ('0xe8eda9df', 'deposit', 'aave-v3',
     'supply(address,uint256,address,uint16)',
     'Supply assets to Aave'),
    ('0x69328dec', 'withdraw', 'aave-v3',
     'withdraw(address,uint256,address)',
     'Withdraw from Aave'),
    
    -- Uniswap Universal Router (V3+)
    ('0x3593564c', 'execute', 'uniswap-universal',
     'execute(bytes,bytes[],uint256)',
     'Execute commands on Universal Router'),
    ('0x24856bc3', 'execute', 'uniswap-universal',
     'execute(bytes,bytes[])',
     'Execute commands without deadline'),
    
    -- Permit2 (Token approvals)
    ('0x30f28b7a', 'permit', 'permit2',
     'permit(address,((address,uint160,uint48,uint48),address,uint256),bytes)',
     'Permit token spending'),
    
    -- CoW Protocol
    ('0x13d79a0b', 'settle', 'cowswap',
     'settle(bytes,bytes[],bytes[])',
     'Settle CoW swap order'),
    
    -- Metamask Swap Router
    ('0xa08edebc', 'swap', 'metamask-router',
     'swap(address,address,uint256,uint256,address,bytes)',
     'Metamask aggregator swap'),
    
    -- Additional High-Volume Signatures (discovered via multi-block scan)
    ('0x78e111f6', 'executeFFsYo', 'forwarder',
     'executeFFsYo(address,bytes)',
     'Meta-transaction forwarder (43 calls in sample)'),
    ('0x122067ed', 'unknown_swap', 'aggregator',
     'unknown_function()',
     'High-volume aggregator function (17 calls in sample)'),
    ('0x88ffe867', 'pledge', 'staking',
     'pledge()',
     'Staking pledge function (12 calls in sample)'),
    ('0x6fadcf72', 'forward', 'forwarder',
     'forward(address,bytes)',
     'Generic meta-transaction forwarder (7 calls in sample)'),
    ('0x791ac947', 'swapExactTokensForETHSupportingFeeOnTransferTokens', 'uniswap-v2',
     'swapExactTokensForETHSupportingFeeOnTransferTokens(uint256,uint256,address[],address,uint256)',
     'Uniswap V2 swap with fee-on-transfer token support (6 calls)'),
    ('0x3d0e3ec5', 'swapExactTokensForETHSupportingFeeOnTransferTokens', 'custom-dex',
     'swapExactTokensForETHSupportingFeeOnTransferTokens(uint256,uint256,address[],address,uint256,address)',
     'Custom DEX swap with fee support and extra parameter (6 calls)');

-- ============================================================================
-- VIEWS
-- ============================================================================

-- View for transaction analytics with parsed data
CREATE VIEW transaction_analytics AS
SELECT 
    t.chain_id,
    t.tx_hash,
    t.block_number,
    t.from_address,
    t.to_address,
    t.value,
    t.status,
    pc.protocol,
    pc.function_name,
    pc.decoded_params,
    rr.revert_reason,
    COUNT(it.internal_tx_index) as internal_tx_count
FROM transactions t
LEFT JOIN parsed_calldata pc ON t.chain_id = pc.chain_id AND t.tx_hash = pc.tx_hash
LEFT JOIN revert_reasons rr ON t.chain_id = rr.chain_id AND t.tx_hash = rr.tx_hash
LEFT JOIN internal_transactions it ON t.chain_id = it.chain_id AND t.tx_hash = it.tx_hash
GROUP BY t.chain_id, t.tx_hash, t.block_number, t.from_address, t.to_address, 
         t.value, t.status, pc.protocol, pc.function_name, pc.decoded_params, rr.revert_reason;

-- View for protocol usage statistics
CREATE VIEW protocol_usage_stats AS
SELECT 
    chain_id,
    protocol,
    function_name,
    COUNT(*) as call_count,
    COUNT(DISTINCT DATE(created_at)) as active_days
FROM parsed_calldata
WHERE protocol IS NOT NULL
GROUP BY chain_id, protocol, function_name
ORDER BY call_count DESC;

COMMIT;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO indexer;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO indexer;
