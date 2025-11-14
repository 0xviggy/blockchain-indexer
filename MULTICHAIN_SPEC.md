# Multi-Chain Support Specification

## Overview

This indexer provides **native multi-chain support** similar to Blockscan, allowing you to index and query data across multiple blockchain networks from a single deployment. Each chain is treated as a first-class citizen with independent ingestion, processing, and storage, while providing unified APIs for cross-chain queries.

---

## Supported Chains

### Tier 1: EVM Chains (Day 1 Support)
| Chain | Chain ID | RPC Provider | Block Time | Finality |
|-------|----------|--------------|------------|----------|
| **Ethereum Mainnet** | 1 | Alchemy, Infura, QuickNode | ~12s | 2 epochs (~13min) |
| **Polygon** | 137 | Alchemy, Infura, Polygon RPC | ~2s | 256 blocks (~8.5min) |
| **Arbitrum One** | 42161 | Alchemy, Infura, Arbitrum RPC | ~0.25s | ~15min |
| **Optimism** | 10 | Alchemy, Infura, Optimism RPC | ~2s | ~10min |
| **Base** | 8453 | Alchemy, Base RPC | ~2s | ~10min |

### Tier 2: EVM Chains (Easy to Add)
| Chain | Chain ID | Notes |
|-------|----------|-------|
| **BSC** | 56 | Binance Smart Chain |
| **Avalanche C-Chain** | 43114 | Fast finality |
| **Fantom** | 250 | High throughput |
| **Gnosis** | 100 | xDai stable chain |
| **zkSync Era** | 324 | zkEVM L2 |
| **Polygon zkEVM** | 1101 | zkEVM L2 |

### Tier 3: Non-EVM (Future)
- Solana
- Cosmos chains
- Polkadot parachains

---

## Architecture Design

### Multi-Chain Ingestion Strategy

**Option 1: Dedicated Ingester per Chain** (Recommended for Production)
```
┌─────────────────────────────────────────────────────────────┐
│                    Ingester Services                         │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Ethereum    │  │  Polygon     │  │  Arbitrum    │      │
│  │  Ingester    │  │  Ingester    │  │  Ingester    │      │
│  │  (chain_id=1)│  │ (chain_id=137)│ │(chain_id=42161)│    │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                 │                 │                │
│         └─────────────────┼─────────────────┘                │
│                           │                                   │
└───────────────────────────┼───────────────────────────────────┘
                            ▼
                   ┌────────────────┐
                   │  Kafka Topics  │
                   │  - eth.blocks  │
                   │  - poly.blocks │
                   │  - arb.blocks  │
                   └────────────────┘
```

**Benefits**:
- Independent scaling per chain
- Isolated failures (one chain down doesn't affect others)
- Chain-specific optimizations
- Easy to add/remove chains

**Option 2: Unified Ingester with Chain Router** (Simpler for Small Deployments)
```
┌─────────────────────────────────────────┐
│      Multi-Chain Ingester               │
│  ┌───────────────────────────────────┐  │
│  │  Chain Router                     │  │
│  │  - Load chain configs             │  │
│  │  - Route to correct RPC           │  │
│  │  - Tag with chain_id              │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

---

## Database Schema (Multi-Chain)

### Chain Metadata Table
```sql
CREATE TABLE chains (
    chain_id INT PRIMARY KEY,
    chain_name VARCHAR(50) NOT NULL,
    rpc_url VARCHAR(255) NOT NULL,
    ws_url VARCHAR(255),
    block_time_seconds INT NOT NULL,
    finality_blocks INT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    last_indexed_block BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Seed data
INSERT INTO chains (chain_id, chain_name, rpc_url, block_time_seconds, finality_blocks) VALUES
(1, 'Ethereum', 'https://eth-mainnet.g.alchemy.com/v2/KEY', 12, 64),
(137, 'Polygon', 'https://polygon-mainnet.g.alchemy.com/v2/KEY', 2, 256),
(42161, 'Arbitrum', 'https://arb-mainnet.g.alchemy.com/v2/KEY', 1, 900),
(10, 'Optimism', 'https://opt-mainnet.g.alchemy.com/v2/KEY', 2, 300),
(8453, 'Base', 'https://base-mainnet.g.alchemy.com/v2/KEY', 2, 300);
```

### Partitioned Tables by Chain

```sql
-- Blocks table (partitioned by chain_id)
CREATE TABLE blocks (
    chain_id INT NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    parent_hash VARCHAR(66) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    miner VARCHAR(42),
    gas_used BIGINT,
    gas_limit BIGINT,
    transaction_count INT,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, block_number),
    UNIQUE (chain_id, block_hash)
) PARTITION BY LIST (chain_id);

-- Create partition per chain
CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks FOR VALUES IN (137);
CREATE TABLE blocks_arbitrum PARTITION OF blocks FOR VALUES IN (42161);
CREATE TABLE blocks_optimism PARTITION OF blocks FOR VALUES IN (10);
CREATE TABLE blocks_base PARTITION OF blocks FOR VALUES IN (8453);

-- Transactions table (partitioned by chain_id)
CREATE TABLE transactions (
    chain_id INT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    tx_index INT NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    value NUMERIC(78, 0),
    gas_price BIGINT,
    gas_used BIGINT,
    input BYTEA,
    status BOOLEAN,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, tx_hash)
) PARTITION BY LIST (chain_id);

CREATE TABLE transactions_eth PARTITION OF transactions FOR VALUES IN (1);
CREATE TABLE transactions_polygon PARTITION OF transactions FOR VALUES IN (137);
CREATE TABLE transactions_arbitrum PARTITION OF transactions FOR VALUES IN (42161);
CREATE TABLE transactions_optimism PARTITION OF transactions FOR VALUES IN (10);
CREATE TABLE transactions_base PARTITION OF transactions FOR VALUES IN (8453);

-- Events table (partitioned by chain_id)
CREATE TABLE events (
    chain_id INT NOT NULL,
    id BIGSERIAL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    log_index INT NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    event_signature VARCHAR(66),
    topic1 VARCHAR(66),
    topic2 VARCHAR(66),
    topic3 VARCHAR(66),
    data BYTEA,
    decoded_data JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, id),
    UNIQUE (chain_id, tx_hash, log_index)
) PARTITION BY LIST (chain_id);

CREATE TABLE events_eth PARTITION OF events FOR VALUES IN (1);
CREATE TABLE events_polygon PARTITION OF events FOR VALUES IN (137);
CREATE TABLE events_arbitrum PARTITION OF events FOR VALUES IN (42161);
CREATE TABLE events_optimism PARTITION OF events FOR VALUES IN (10);
CREATE TABLE events_base PARTITION OF events FOR VALUES IN (8453);

-- Indexes for cross-chain queries
CREATE INDEX idx_transactions_from_chain ON transactions(from_address, chain_id, block_number DESC);
CREATE INDEX idx_transactions_to_chain ON transactions(to_address, chain_id, block_number DESC);
CREATE INDEX idx_events_contract_chain ON events(contract_address, chain_id, block_number DESC);
```

---

## API Design (Multi-Chain)

### Chain-Specific Endpoints

```bash
# Get block on specific chain
GET /api/v1/eth/blocks/18234567
GET /api/v1/polygon/blocks/50000000
GET /api/v1/arbitrum/blocks/150000000

# Alternative: chain_id as query param
GET /api/v1/blocks/18234567?chain_id=1

# Get transaction
GET /api/v1/eth/transactions/0x123...
GET /api/v1/polygon/transactions/0x456...

# Query events on specific chain
GET /api/v1/arbitrum/events?contract_address=0xABC&from_block=100000&to_block=200000

# Get address activity on specific chain
GET /api/v1/eth/addresses/0x123.../transactions
GET /api/v1/polygon/addresses/0x123.../balance
```

### Cross-Chain Endpoints

```bash
# Get all transactions for an address across ALL chains
GET /api/v1/addresses/0x123.../transactions
Response:
{
  "data": [
    {
      "chain_id": 1,
      "chain_name": "Ethereum",
      "tx_hash": "0xaaa...",
      "block_number": 18234567,
      ...
    },
    {
      "chain_id": 137,
      "chain_name": "Polygon",
      "tx_hash": "0xbbb...",
      "block_number": 50000000,
      ...
    }
  ]
}

# Get portfolio across all chains
GET /api/v1/addresses/0x123.../portfolio
Response:
{
  "address": "0x123...",
  "chains": {
    "1": {
      "chain_name": "Ethereum",
      "native_balance": "1.5 ETH",
      "token_balances": [...]
    },
    "137": {
      "chain_name": "Polygon",
      "native_balance": "100 MATIC",
      "token_balances": [...]
    }
  }
}

# List supported chains
GET /api/v1/chains
Response:
{
  "chains": [
    {
      "chain_id": 1,
      "name": "Ethereum",
      "enabled": true,
      "current_block": 18234567,
      "lag_seconds": 15
    },
    {
      "chain_id": 137,
      "name": "Polygon",
      "enabled": true,
      "current_block": 50000123,
      "lag_seconds": 8
    }
  ]
}

# Get chain sync status
GET /api/v1/chains/1/status
Response:
{
  "chain_id": 1,
  "chain_name": "Ethereum",
  "current_block": 18234567,
  "chain_head": 18234580,
  "lag_blocks": 13,
  "lag_seconds": 156,
  "sync_rate_blocks_per_sec": 5.2,
  "last_updated": "2025-11-14T12:34:56Z"
}
```

### WebSocket (Multi-Chain)

```javascript
// Subscribe to specific chain
const ws = new WebSocket('wss://api.indexer.io/v1/eth/stream');
ws.send(JSON.stringify({
  type: 'subscribe',
  filters: {
    contract_address: '0xA0b...',
    event_signature: '0xddf...'
  }
}));

// Subscribe to multiple chains
const ws = new WebSocket('wss://api.indexer.io/v1/stream');
ws.send(JSON.stringify({
  type: 'subscribe',
  chains: [1, 137, 42161],  // Ethereum, Polygon, Arbitrum
  filters: {
    address: '0x123...'
  }
}));

// Receive events
{
  "type": "event",
  "chain_id": 137,
  "chain_name": "Polygon",
  "data": {
    "tx_hash": "0x456...",
    "block_number": 50000123,
    ...
  }
}
```

---

## Configuration (Multi-Chain)

### Ingester Config

```yaml
chains:
  - chain_id: 1
    name: Ethereum
    enabled: true
    rpc_urls:
      - https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
      - https://mainnet.infura.io/v3/YOUR_KEY
    ws_url: wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
    start_block: 18000000
    batch_size: 100
    block_time_seconds: 12
    finality_blocks: 64
    
  - chain_id: 137
    name: Polygon
    enabled: true
    rpc_urls:
      - https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY
    ws_url: wss://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY
    start_block: 45000000
    batch_size: 500  # Faster blocks, larger batches
    block_time_seconds: 2
    finality_blocks: 256
    
  - chain_id: 42161
    name: Arbitrum
    enabled: true
    rpc_urls:
      - https://arb-mainnet.g.alchemy.com/v2/YOUR_KEY
    ws_url: wss://arb-mainnet.g.alchemy.com/v2/YOUR_KEY
    start_block: 100000000
    batch_size: 1000  # Very fast blocks
    block_time_seconds: 1
    finality_blocks: 900

kafka:
  topics:
    pattern: "{chain_name}.{event_type}"  # e.g., eth.blocks, polygon.events
```

---

## Implementation Guide

### Step 1: Add New Chain (5 minutes)

```sql
-- 1. Add chain metadata
INSERT INTO chains (chain_id, chain_name, rpc_url, block_time_seconds, finality_blocks)
VALUES (56, 'BSC', 'https://bsc-dataseed.binance.org', 3, 15);

-- 2. Create partitions
CREATE TABLE blocks_bsc PARTITION OF blocks FOR VALUES IN (56);
CREATE TABLE transactions_bsc PARTITION OF transactions FOR VALUES IN (56);
CREATE TABLE events_bsc PARTITION OF events FOR VALUES IN (56);
```

```yaml
# 3. Update ingester config
chains:
  - chain_id: 56
    name: BSC
    enabled: true
    rpc_urls:
      - https://bsc-dataseed.binance.org
    start_block: 30000000
    batch_size: 200
    block_time_seconds: 3
    finality_blocks: 15
```

```bash
# 4. Restart ingester
docker restart ingester-bsc
```

### Step 2: Query New Chain

```bash
# Chain is now available
GET /api/v1/bsc/blocks/latest
GET /api/v1/chains/56/status
```

---

## Monitoring (Multi-Chain)

### Prometheus Metrics

```prometheus
# Per-chain ingestion rate
indexer_blocks_ingested_total{chain="ethereum"} 18234567
indexer_blocks_ingested_total{chain="polygon"} 50000123
indexer_blocks_ingested_total{chain="arbitrum"} 150000456

# Per-chain lag
indexer_block_lag_seconds{chain="ethereum"} 15
indexer_block_lag_seconds{chain="polygon"} 8
indexer_block_lag_seconds{chain="arbitrum"} 5

# Per-chain error rate
indexer_rpc_errors_total{chain="ethereum"} 12
indexer_rpc_errors_total{chain="polygon"} 3

# Cross-chain query performance
indexer_api_cross_chain_query_duration_seconds{quantile="0.95"} 0.45
```

### Grafana Dashboard

```
┌─────────────────────────────────────────────────────────────┐
│               Multi-Chain Indexer Dashboard                  │
├─────────────────────────────────────────────────────────────┤
│ Chain Status                                                 │
│ ┌─────────────┬─────────┬──────────┬──────┬───────────┐    │
│ │ Chain       │ Block   │ Lag (s)  │ Rate │ Status    │    │
│ ├─────────────┼─────────┼──────────┼──────┼───────────┤    │
│ │ Ethereum    │ 18.2M   │ 15s      │ 5/s  │ 🟢 Healthy│    │
│ │ Polygon     │ 50.0M   │ 8s       │ 25/s │ 🟢 Healthy│    │
│ │ Arbitrum    │ 150.0M  │ 5s       │ 50/s │ 🟢 Healthy│    │
│ │ Optimism    │ 110.5M  │ 12s      │ 10/s │ 🟡 Warning│    │
│ │ Base        │ 5.2M    │ 180s     │ 1/s  │ 🔴 Behind │    │
│ └─────────────┴─────────┴──────────┴──────┴───────────┘    │
├─────────────────────────────────────────────────────────────┤
│ Ingestion Rate (blocks/sec)                                  │
│ [Chart showing all chains over time]                         │
├─────────────────────────────────────────────────────────────┤
│ API Request Distribution by Chain                            │
│ Ethereum: 45% │ Polygon: 30% │ Arbitrum: 20% │ Others: 5%   │
└─────────────────────────────────────────────────────────────┘
```

---

## Performance Considerations

### Chain-Specific Optimizations

1. **Ethereum**: 
   - Slower blocks (12s), smaller batches
   - Higher RPC costs, use caching aggressively
   
2. **Polygon**:
   - Fast blocks (2s), larger batches (500-1000)
   - More frequent reorgs, shorter finality wait
   
3. **Arbitrum**:
   - Very fast blocks (0.25s), very large batches (1000+)
   - Lower RPC costs, can query more frequently

### Resource Allocation

```yaml
# Example Kubernetes resource allocation
ingester-ethereum:
  replicas: 2
  resources:
    cpu: 1000m
    memory: 2Gi

ingester-polygon:
  replicas: 3
  resources:
    cpu: 1500m
    memory: 3Gi

ingester-arbitrum:
  replicas: 5
  resources:
    cpu: 2000m
    memory: 4Gi
```

---

## Cross-Chain Features

### 1. Address Portfolio Aggregation

Aggregate balances and activity across all chains for a given address.

### 2. Cross-Chain Event Correlation

Detect related events across chains (e.g., bridge transactions).

```sql
-- Find bridge events (lock on Ethereum, mint on Polygon)
SELECT 
    e1.chain_id as source_chain,
    e2.chain_id as dest_chain,
    e1.decoded_data->>'from' as user,
    e1.decoded_data->>'amount' as amount,
    e1.block_number as lock_block,
    e2.block_number as mint_block
FROM events e1
JOIN events e2 
    ON e1.decoded_data->>'from' = e2.decoded_data->>'to'
    AND e1.decoded_data->>'amount' = e2.decoded_data->>'amount'
WHERE e1.chain_id = 1  -- Ethereum
    AND e2.chain_id = 137  -- Polygon
    AND e1.event_signature = '0xLockEventSig'
    AND e2.event_signature = '0xMintEventSig'
    AND e2.timestamp BETWEEN e1.timestamp AND e1.timestamp + INTERVAL '1 hour';
```

### 3. Multi-Chain Analytics

- Compare activity across chains
- Identify arbitrage opportunities
- Track token migrations

---

## Comparison with Blockscan

| Feature | Our Indexer | Blockscan |
|---------|-------------|-----------|
| **Supported Chains** | Unlimited (self-add) | 90+ chains |
| **Self-hosted** | ✅ Yes | ❌ No |
| **API Access** | ✅ Full control | ⚠️ Rate limited |
| **Cross-chain queries** | ✅ Yes | ⚠️ Limited |
| **Real-time** | ✅ <30s lag | ✅ Real-time |
| **Custom parsers** | ✅ Full control | ❌ No |
| **Cost** | $500-1000/mo | Free (limited) |
| **Privacy** | ✅ On-premise | ❌ Public API |

---

## Conclusion

This multi-chain architecture provides:

✅ **Scalability**: Independent ingestion per chain  
✅ **Flexibility**: Add new chains in minutes  
✅ **Performance**: Chain-specific optimizations  
✅ **Unified API**: Single endpoint for cross-chain queries  
✅ **Production-ready**: Like Blockscan but self-hosted

**Next Steps**:
1. Implement dedicated ingester per chain
2. Add chain metadata management
3. Build cross-chain query layer
4. Create multi-chain monitoring dashboards
