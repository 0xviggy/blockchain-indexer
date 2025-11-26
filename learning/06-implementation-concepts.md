# Implementation Concepts & Design Decisions

Deep dive into the key technical concepts, design decisions, and implementation details of the blockchain indexer.

## Table of Contents
- [Message Parsing](#message-parsing)
- [Blockchain Reorg Handling](#blockchain-reorg-handling)
- [Event Parsing](#event-parsing)
- [Database Partitioning Strategy](#database-partitioning-strategy)
- [Rate Limiting](#rate-limiting)
- [Kafka Message Ordering](#kafka-message-ordering)
- [Design Decisions & Trade-offs](#design-decisions--trade-offs)

---

## Message Parsing

### 1. Message Parsing Overview

Advanced message parsing enables deep transaction analysis:
- Function call decoding
- Internal transaction tracking
- Revert reason extraction

### Internal Transactions

**What are they?** Contract-to-contract calls that happen within a transaction.

**Why track them?**
- See the full flow of funds through complex DeFi transactions
- Understand how a swap goes through multiple routers
- Track delegatecall patterns and contract interactions

**Example**: Uniswap V2 swap
```
1. User calls Router.swapExactTokensForTokens()
2. Router transfers tokens from user (CALL to ERC20)
3. Router calls Pair.swap() (CALL)
4. Pair transfers tokens to user (CALL to ERC20)
```

**Database Schema:**
```sql
CREATE TABLE internal_transactions (
    chain_id BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    internal_tx_index INT NOT NULL,
    call_type TEXT NOT NULL,  -- 'call', 'delegatecall', 'staticcall', 'create'
    from_address TEXT NOT NULL,
    to_address TEXT,
    value NUMERIC(78, 0),
    input TEXT,
    output TEXT,
    success BOOLEAN NOT NULL,
    PRIMARY KEY (chain_id, tx_hash, internal_tx_index)
) PARTITION BY LIST (chain_id);
```

### Calldata Parsing

**What is calldata?** The input data to a transaction, containing:
- Function signature (first 4 bytes)
- Encoded parameters

**Why parse it?**
- Identify which function was called
- Extract parameters (swap amounts, token addresses, etc.)
- Categorize by protocol (Uniswap, Aave, LayerZero)

**Database Schema:**
```sql
CREATE TABLE parsed_calldata (
    chain_id BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    function_signature TEXT NOT NULL,  -- e.g., "0x38ed1739"
    function_name TEXT,                -- e.g., "swapExactTokensForTokens"
    protocol TEXT,                     -- e.g., "uniswap-v2"
    decoded_params JSONB,              -- Flexible JSON storage
    PRIMARY KEY (chain_id, tx_hash)
) PARTITION BY LIST (chain_id);
```

**Example Query - Track DEX Swaps:**
```sql
SELECT 
    t.tx_hash,
    t.from_address,
    pc.function_name,
    pc.decoded_params->>'amountIn' as amount_in,
    pc.decoded_params->>'path' as swap_path
FROM transactions t
JOIN parsed_calldata pc ON t.tx_hash = pc.tx_hash
WHERE pc.protocol = 'uniswap-v2'
  AND pc.function_name = 'swapExactTokensForTokens';
```

### Revert Reason Extraction

**What is it?** Error messages from failed transactions.

**Why important?**
- Debug why transactions failed
- Identify common failure patterns
- Improve protocol UX

**Database Schema:**
```sql
CREATE TABLE revert_reasons (
    chain_id BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    revert_reason TEXT,
    error_signature TEXT,
    error_name TEXT,
    error_params JSONB,
    PRIMARY KEY (chain_id, tx_hash)
) PARTITION BY LIST (chain_id);
```

**Example Query - Most Common Failures:**
```sql
SELECT 
    revert_reason,
    COUNT(*) as failure_count
FROM revert_reasons
WHERE chain_id = 1
GROUP BY revert_reason
ORDER BY failure_count DESC
LIMIT 10;
```

### Protocol Signatures Registry

Pre-configured function signatures for major protocols:

**DEX Protocols:**
- Uniswap V2: `swapExactTokensForTokens`, `swapExactETHForTokens`
- Uniswap V3: `exactInputSingle`, `exactInput`
- Curve: `exchange`
- 1inch: `swap`

**Bridge Protocols:**
- LayerZero: `send`
- Across: `deposit`
- Stargate: `swap`

**DeFi Lending:**
- Aave V3: `supply`, `withdraw`

**NFT Marketplaces:**
- OpenSea Seaport: `fulfillOrder`

---

## Blockchain Reorg Handling

### What is a Reorg?

**Definition**: When the canonical chain changes, invalidating previously indexed blocks.

**Why it happens:**
- Two miners produce blocks simultaneously
- Network latency causes temporary chain splits
- Chain resolves to the longest valid chain

### Detection Strategy

```go
// Pseudocode
func detectReorg(newBlock Block) bool {
    storedParentBlock := db.GetBlock(newBlock.ParentHash)
    if storedParentBlock == nil {
        // Parent doesn't exist - reorg detected
        return true
    }
    if storedParentBlock.BlockNumber != newBlock.BlockNumber - 1 {
        // Block numbers don't align - reorg detected
        return true
    }
    return false
}
```

### Handling Process

1. **Detect**: Compare parent hashes when ingesting new blocks
2. **Find common ancestor**: Walk backwards through parent hashes
3. **Rollback**: Delete all blocks after common ancestor in a transaction
4. **Resume**: Continue ingestion from common ancestor + 1
5. **Replay**: Kafka allows processor to re-process affected events

### Database Implementation

```sql
-- Track reorgs
CREATE TABLE reorg_events (
    chain_id BIGINT NOT NULL,
    detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    rollback_from_block BIGINT NOT NULL,
    rollback_to_block BIGINT NOT NULL,
    blocks_removed INT NOT NULL,
    handled BOOLEAN DEFAULT FALSE
);

-- Delete blocks in transaction
BEGIN;
DELETE FROM events WHERE chain_id = 1 AND block_number > 12345;
DELETE FROM transactions WHERE chain_id = 1 AND block_number > 12345;
DELETE FROM blocks WHERE chain_id = 1 AND block_number > 12345;
INSERT INTO reorg_events (...) VALUES (...);
COMMIT;
```

### Interview Answer

**Q: How do you handle blockchain reorganizations?**

**A:** "We detect reorgs by comparing parent hashes. When ingesting block N, we check if our stored block N-1's hash matches the new block's parent hash. If not, we've detected a reorg.

We then:
1. Find the common ancestor by walking back through parent hashes
2. Use a database transaction to delete all blocks after the common ancestor
3. Resume ingestion from the common ancestor + 1
4. Kafka's replay capability allows the processor to re-process affected events

For production, we only mark blocks as 'finalized' after the chain-specific finality threshold (e.g., 13 minutes for Ethereum, 256 blocks for Polygon)."

---

## Event Parsing

### ERC20 Transfer Example

**Event signature:**
```solidity
event Transfer(address indexed from, address indexed to, uint256 value);
```

**Ethereum log:**
```json
{
  "topics": [
    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",  // keccak256("Transfer(address,address,uint256)")
    "0x000000000000000000000000742d35cc6634c0532925a3b844bc454e4438f44e",      // from (padded address)
    "0x000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"       // to (padded address)
  ],
  "data": "0x0000000000000000000000000000000000000000000000056bc75e2d63100000"  // value (100 tokens with 18 decimals)
}
```

**Decoded:**
```json
{
  "event": "Transfer",
  "from": "0x742d35cc6634c0532925a3b844bc454e4438f44e",
  "to": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
  "value": "100000000000000000000"
}
```

### Database Storage

```sql
INSERT INTO events (
    chain_id,
    tx_hash,
    log_index,
    contract_address,
    event_signature,
    decoded_data
) VALUES (
    1,
    '0x123...',
    42,
    '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48',  -- USDC
    'Transfer',
    '{"from": "0x742...", "to": "0xc02...", "value": "100000000000000000000"}'::jsonb
);
```

**Query Example:**
```sql
-- Find all transfers to/from an address
SELECT * FROM events
WHERE event_signature = 'Transfer'
  AND (
    decoded_data->>'from' = '0x742d35cc...'
    OR decoded_data->>'to' = '0x742d35cc...'
  );
```

---

## Database Partitioning Strategy

### Why Partition?

**Benefits:**
- **Query performance**: Only scan relevant partitions (partition pruning)
- **Maintenance**: Easier to archive/delete old data per chain
- **Scalability**: Different chains can have different storage tiers
- **Isolation**: One chain's issues don't affect others

### Partitioning Scheme

**Partition by `chain_id` (LIST partitioning):**

```sql
-- Parent table
CREATE TABLE blocks (
    chain_id BIGINT NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash TEXT NOT NULL,
    -- ... other fields
    PRIMARY KEY (chain_id, block_number)
) PARTITION BY LIST (chain_id);

-- Child partitions
CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks FOR VALUES IN (137);
CREATE TABLE blocks_arbitrum PARTITION OF blocks FOR VALUES IN (42161);
```

### How Partition Pruning Works

**Query with chain_id filter:**
```sql
SELECT * FROM blocks WHERE chain_id = 1 AND block_number > 18000000;
-- PostgreSQL only scans blocks_eth, not other partitions
```

**Query without chain_id:**
```sql
SELECT * FROM blocks WHERE block_number > 18000000;
-- PostgreSQL scans ALL partitions (slower)
```

**Lesson**: Always include `chain_id` in WHERE clause for best performance.

### Interview Answer

**Q: Why use database partitioning?**

**A:** "We partition tables by `chain_id` so each blockchain gets its own partition. This provides:

1. **Performance**: Queries filtered by chain_id only scan the relevant partition (partition pruning)
2. **Maintenance**: Easy to archive old Ethereum data without affecting Polygon
3. **Scalability**: Can move hot chains to SSD, cold chains to HDD
4. **Isolation**: One chain's high volume doesn't slow down others

For example, `blocks_eth` and `blocks_polygon` are separate partitions of the `blocks` table. A query for Ethereum blocks never touches Polygon data."

---

## Rate Limiting

### Token Bucket Algorithm

**Concept**: Fill a bucket with tokens at a constant rate. Each request consumes a token. If bucket is empty, request is denied.

**Implementation:**
```go
type TokenBucket struct {
    capacity    int64
    tokens      int64
    refillRate  int64  // tokens per second
    lastRefill  time.Time
    mu          sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    // Refill tokens based on time elapsed
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()
    tokensToAdd := int64(elapsed * float64(tb.refillRate))
    
    tb.tokens = min(tb.capacity, tb.tokens + tokensToAdd)
    tb.lastRefill = now
    
    // Consume token if available
    if tb.tokens > 0 {
        tb.tokens--
        return true
    }
    return false
}
```

**Parameters:**
- `capacity`: Maximum tokens (burst size)
- `refillRate`: Tokens added per second
- Example: 100 capacity, 10 refill rate = 10 req/sec sustained, 100 req burst

### Distributed Rate Limiting with Redis

```go
func rateLimitRedis(apiKey string, limit int) bool {
    key := fmt.Sprintf("ratelimit:%s:%d", apiKey, time.Now().Unix()/60)
    
    count := redis.Incr(key)
    redis.Expire(key, 120) // 2 minute window
    
    return count <= limit
}
```

**Interview Answer:**

**Q: How do you handle API rate limiting at scale?**

**A:** "We implement multi-tier rate limiting:

1. **Per-IP limiting**: Token bucket with 100 req/sec, burst 200 (prevents DDoS)
2. **Per-API-key limiting**: Higher limits for authenticated users (1000 req/sec)
3. **Distributed rate limiting**: Use Redis with sliding window counters for consistency across API servers
4. **Circuit breaker**: Temporarily block IPs that consistently hit limits
5. **Response headers**: Return `X-RateLimit-Remaining` and `X-RateLimit-Reset`

Redis ensures any API server can enforce limits consistently in a distributed system."

---

## Kafka Message Ordering

### Why Ordering Matters

Blockchain data has strict ordering requirements:
- Block N must be processed before Block N+1
- Reorgs require processing blocks in reverse order
- Events within a block must maintain log order

### Kafka Guarantees

**Within a partition:**
- ✅ Messages are ordered
- ✅ Consumers see messages in order
- ✅ Offsets are sequential

**Across partitions:**
- ❌ No ordering guarantee
- Messages can be processed in any order

### Design Decision

**Use chain_id as partition key:**
```go
kafkaMessage := kafka.Message{
    Topic: "blocks",
    Key:   []byte(fmt.Sprintf("chain-%d", chainID)),  // Same chain always goes to same partition
    Value: blockJSON,
}
```

**Result:**
- All Ethereum blocks go to same partition → ordered processing
- Different chains go to different partitions → parallel processing
- Processors can scale per-chain

---

## Design Decisions & Trade-offs

### 1. Language Choice: Go vs Rust

**Decision: Go** ✅

**Why Go:**
- ✅ Faster development (simpler syntax, less ceremony)
- ✅ Better ecosystem for our stack (go-ethereum, Kafka, PostgreSQL drivers)
- ✅ Easier hiring (more Go developers)
- ✅ Good enough performance (not CPU-bound workload)
- ✅ Great concurrency primitives (goroutines, channels)

**Why Not Rust:**
- ❌ Steeper learning curve (borrow checker, lifetimes)
- ❌ Slower development velocity
- ❌ Would be beneficial for: CPU-intensive parsing, zero-copy deserialization
- ❌ Overkill for I/O-bound blockchain indexing

**Interview Answer:** "We chose Go because blockchain indexing is I/O-bound (waiting on RPC calls, database writes), not CPU-bound. Go's goroutines make it trivial to parallelize RPC calls across multiple chains. The go-ethereum library is the canonical Ethereum implementation. Development velocity matters more than squeezing out 20% more performance for this use case. If we were building a high-frequency trading engine with microsecond latency requirements, Rust would be the choice."

### 2. Message Broker: Kafka vs RabbitMQ

**Decision: Kafka** ✅

**Why Kafka:**
- ✅ High throughput (100K+ msg/sec per partition)
- ✅ Replay capability (critical for reorg handling)
- ✅ Persistent storage (messages kept for days/weeks)
- ✅ Consumer groups (multiple processors can share load)
- ✅ Ordering guarantees within partition

**Why Not RabbitMQ:**
- ❌ Lower throughput (~20K msg/sec)
- ❌ Message deletion after consumption (can't replay)
- ❌ Better for: Job queues, RPC-style messaging

**Interview Answer:** "Kafka's replay capability is critical for blockchain reorgs. When we detect a reorg, we can reset the consumer offset and re-process affected blocks. RabbitMQ deletes messages after consumption, making replay impossible. Kafka's high throughput handles mainnet's 15M transactions per day easily. The trade-off is operational complexity (managing partitions, retention policies), but the benefits outweigh the cost."

### 3. Database: PostgreSQL vs TimescaleDB vs Cassandra

**Decision: PostgreSQL** ✅

**Why PostgreSQL:**
- ✅ ACID transactions (critical for reorg rollbacks)
- ✅ Rich query capabilities (joins, aggregations, JSONB)
- ✅ Excellent partitioning support
- ✅ Mature ecosystem and tooling
- ✅ Good enough performance with proper indexing

**Why Not TimescaleDB:**
- ❌ Optimized for time-series, but we need relational queries
- ❌ Would help: If we added high-frequency price data (ticks every second)
- ✅ Could migrate later if needed

**Why Not Cassandra:**
- ❌ Eventual consistency (not acceptable for financial data)
- ❌ No joins (would need denormalization everywhere)
- ❌ Better for: Massive scale (billions of rows), geo-distribution
- ❌ Overkill for our scale (millions of rows)

**Interview Answer:** "PostgreSQL's ACID transactions are essential for reorg handling. When we rollback a reorg, we need all deletes (blocks, transactions, events) to succeed or fail atomically. Cassandra's eventual consistency would allow partial state, corrupting our index. PostgreSQL's JSONB gives us flexibility for event data without sacrificing query power. If we scale to billions of rows, we'd consider TimescaleDB (drop-in PostgreSQL replacement) or shard by chain_id."

### 4. Monorepo vs Separate Repos

**Decision: Monorepo** ✅

**Why Monorepo:**
- ✅ Shared types (`models.go`) used by all services
- ✅ Shared config (`config.go`)
- ✅ Easier refactoring across services
- ✅ Single CI/CD pipeline
- ✅ Easier local development (clone once)

**Trade-offs:**
- ❌ All services share Go module versions (less flexibility)
- ❌ Larger repository size
- ✅ Acceptable for small team, would revisit at 10+ services

**Interview Answer:** "Monorepo makes sense for our architecture because services share common models (Block, Transaction, Event structs). Changes to the schema require updates across all services - monorepo makes this atomic. If services had truly independent domains (e.g., payment processing + analytics + auth), separate repos would be better. For our tightly coupled blockchain services, monorepo reduces friction."

---

## Related Documentation

- [Technology Stack](./01-technology-stack.md)
- [Go Programming Concepts](./04-go-programming.md)
- [Interview Preparation](./07-interview-prep.md)
- [Technical Specification](../docs/TECHNICAL_SPEC.md)
