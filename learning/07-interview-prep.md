# Interview Preparation Guide

Common blockchain infrastructure interview questions with detailed answers based on this project.

## Table of Contents
- [System Design Questions](#system-design-questions)
- [Blockchain-Specific Questions](#blockchain-specific-questions)
- [Database & Performance](#database--performance)
- [API Design & Scaling](#api-design--scaling)
- [Quick Reference](#quick-reference)

---

## System Design Questions

### Q1: How do you handle blockchain reorganizations?

**Answer:**

"We detect reorgs by comparing parent hashes. When ingesting block N, we check if our stored block N-1's hash matches the new block's parent hash. If not, we've detected a reorg.

We then:
1. Find the common ancestor by walking back through parent hashes
2. Use a database transaction to delete all blocks after the common ancestor
3. Resume ingestion from the common ancestor + 1
4. Kafka's replay capability allows the processor to re-process affected events

For production, we only mark blocks as 'finalized' after the chain-specific finality threshold (e.g., 13 minutes for Ethereum, 256 blocks for Polygon)."

**Follow-up: What if the reorg is 1000 blocks deep?**

"Deep reorgs are extremely rare on major chains (would require massive 51% attack). For defense:
1. Alert on-call engineers for manual review
2. Pause ingestion temporarily
3. Verify multiple RPC providers agree on the chain state
4. Consider the chain compromised and wait for community resolution

For probabilistic finality chains (Ethereum), we typically consider blocks final after 32 slots (~6.4 minutes). For deterministic finality (Cosmos, Polkadot), reorgs are impossible after finalization."

---

### Q2: How do you scale the ingester for multiple chains?

**Answer:**

"We deploy separate ingester instances per chain, each with:
- Dedicated checkpoint tracking (last_indexed_block per chain)
- Chain-specific configuration (RPC URLs, finality rules, block times)
- Independent failure domains (one chain's RPC issues don't affect others)

Each ingester publishes to a chain-specific Kafka topic (e.g., 'eth.blocks', 'poly.blocks'). This enables:
- Parallel processing across chains
- Independent scaling per chain (Ethereum might need 3 ingesters, Polygon 1)
- Chain-specific monitoring and alerting

For efficiency, we could run multiple chains in one process with goroutines, but separate deployments provide better isolation."

**Follow-up: How do you handle rate limits from RPC providers?**

"Multi-layered approach:
1. **Token bucket rate limiter** per RPC provider (respects their limits)
2. **Multiple providers** with automatic failover (Infura + Alchemy + Quicknode)
3. **Connection pooling** to reuse TCP connections
4. **Exponential backoff** on 429 responses
5. **Caching** of immutable data (old blocks never change)
6. **WebSocket subscriptions** for new blocks (more efficient than polling)"

---

### Q3: How do you ensure data consistency during high load?

**Answer:**

"We use several strategies:

1. **Database transactions**: All reorg rollbacks and event processing use ACID transactions

2. **Idempotency**: All operations are idempotent using `ON CONFLICT DO NOTHING` or upserts
   ```sql
   INSERT INTO events (...) VALUES (...) 
   ON CONFLICT (tx_hash, log_index) DO NOTHING;
   ```

3. **Exactly-once semantics**: Kafka consumers commit offsets only after successful database writes

4. **Checkpointing**: We persist last_processed_block and resume from there on restart

5. **Backpressure handling**: Kafka acts as a buffer between ingestion and processing, preventing overload

6. **Connection pooling**: Limit concurrent database connections to prevent saturation"

**Follow-up: What happens if the processor crashes mid-batch?**

"Because we use Kafka's offset commit AFTER successful database write, the crash scenario is safe:

1. Processor crashes after writing 7/10 messages to database
2. Kafka offset was NOT committed (we only commit after full batch success)
3. On restart, processor sees last committed offset
4. Re-processes those 7 messages
5. Database INSERT uses `ON CONFLICT DO NOTHING` → No duplicates
6. Continues with remaining 3 messages

This guarantees at-least-once delivery with idempotent writes = effectively exactly-once."

---

## Blockchain-Specific Questions

### Q4: How would you optimize query performance for address transaction history?

**Answer:**

"Multiple approaches:

1. **Composite indexes**:
   ```sql
   CREATE INDEX idx_txs_from_addr ON transactions (from_address, block_number DESC);
   CREATE INDEX idx_txs_to_addr ON transactions (to_address, block_number DESC);
   ```

2. **Materialized view** for frequently accessed patterns:
   ```sql
   CREATE MATERIALIZED VIEW address_activity AS
   SELECT address, COUNT(*) as tx_count, MAX(block_number) as last_active
   FROM (
       SELECT from_address as address FROM transactions
       UNION ALL
       SELECT to_address FROM transactions WHERE to_address IS NOT NULL
   ) GROUP BY address;
   ```

3. **Redis caching**: Cache recent 100 transactions per address (TTL 5 minutes)

4. **Pagination**: Always use limit/offset or cursor-based pagination

5. **Partitioning**: Partition by block_number to limit scan range

6. **Denormalization**: For high-value addresses, maintain a separate summary table"

**Follow-up: Cursor-based pagination vs offset/limit?**

"Cursor-based is superior for large datasets:

**Offset/Limit Issues:**
```sql
SELECT * FROM transactions WHERE from_address = '0x123' 
ORDER BY block_number DESC 
OFFSET 1000000 LIMIT 20;  -- Scans 1M rows to skip them!
```

**Cursor-Based Solution:**
```sql
SELECT * FROM transactions 
WHERE from_address = '0x123' AND block_number < 18500000 
ORDER BY block_number DESC 
LIMIT 20;  -- Index seek, not scan
```

Client passes `last_block_number` from previous page as cursor. Much faster for deep pagination."

---

### Q5: How do you handle API rate limiting at scale?

**Answer:**

"We implement multi-tier rate limiting:

1. **Per-IP limiting**: Token bucket with 100 req/sec, burst 200 (prevents DDoS)
2. **Per-API-key limiting**: Higher limits for authenticated users (1000 req/sec)
3. **Distributed rate limiting**: Use Redis with sliding window counters
   ```go
   key := fmt.Sprintf("ratelimit:%s:%d", apiKey, time.Now().Unix()/60)
   count := redis.Incr(key)
   redis.Expire(key, 120) // 2 minute window
   if count > limit { return 429 }
   ```

4. **Circuit breaker**: Temporarily block IPs that consistently hit limits
5. **Response headers**: Return `X-RateLimit-Remaining` and `X-RateLimit-Reset`
6. **Tiered pricing**: Free tier 100 req/sec, paid tiers 1000-10000 req/sec

We use Redis for distributed state, so any API server can enforce limits consistently."

**Follow-up: Why Redis over in-memory rate limiting?**

"In-memory rate limiting doesn't work with multiple API servers:

**Problem**: User makes 100 req/sec to each of 5 API servers = 500 req/sec total (bypasses 100 req/sec limit)

**Solution**: Redis as single source of truth
- All API servers increment the same Redis key
- Atomic operations (INCR) prevent race conditions
- TTL (EXPIRE) handles window expiration automatically
- Trade-off: Network latency to Redis (~1ms), but necessary for correctness"

---

## Database & Performance

### Q6: Explain your partitioning strategy

**Answer:**

"We partition tables by `chain_id` (LIST partitioning) so each blockchain gets its own partition:

```sql
CREATE TABLE blocks (...) PARTITION BY LIST (chain_id);
CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks FOR VALUES IN (137);
```

**Benefits:**
1. **Query performance**: Queries with `chain_id` filter only scan relevant partition (partition pruning)
2. **Maintenance**: Can archive old Ethereum data without affecting Polygon
3. **Scalability**: Move hot chains to SSD, cold chains to HDD
4. **Isolation**: One chain's high volume doesn't slow down others

**Critical**: Always include `chain_id` in WHERE clause:
```sql
-- Fast (partition pruning)
SELECT * FROM blocks WHERE chain_id = 1 AND block_number > 18000000;

-- Slow (scans ALL partitions)
SELECT * FROM blocks WHERE block_number > 18000000;
```

**Why LIST not RANGE?**: Chain IDs aren't sequential (Ethereum=1, Polygon=137, Arbitrum=42161). LIST partitioning maps specific values to partitions."

---

### Q7: How would you handle 1 billion transactions?

**Answer:**

"Current design supports ~100M transactions. For 1B+:

1. **Partition by time + chain**:
   ```sql
   -- Sub-partition Ethereum by month
   CREATE TABLE blocks_eth_2024_01 PARTITION OF blocks_eth 
   FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
   ```

2. **Time-series database**: Migrate to TimescaleDB (PostgreSQL extension)
   - Automatic partitioning by time
   - Compression (10x space savings)
   - Continuous aggregates for analytics

3. **Archival strategy**: 
   - Hot data: Last 3 months on SSD
   - Warm data: 3-12 months on HDD
   - Cold data: >12 months on S3 (query via Athena)

4. **Horizontal sharding**: Shard by address hash for address-centric queries
   ```
   Shard 1: addresses starting with 0x0-0x7
   Shard 2: addresses starting with 0x8-0xf
   ```

5. **Read replicas**: 5 read replicas for queries, 1 primary for writes"

---

## API Design & Scaling

### Q8: Design an API endpoint for real-time block streaming

**Answer:**

"Two approaches depending on client capabilities:

**Approach 1: WebSocket** (best for browsers)
```
GET /ws/blocks?chain_id=1
Upgrade: websocket

// Server pushes new blocks
{
  "type": "block",
  "chain_id": 1,
  "block_number": 18500000,
  "hash": "0x123...",
  "timestamp": 1699564800,
  "transaction_count": 156
}
```

**Approach 2: Server-Sent Events (SSE)** (simpler than WebSocket)
```
GET /v1/chains/1/blocks/stream
Accept: text/event-stream

data: {"block_number": 18500000, ...}

data: {"block_number": 18500001, ...}
```

**Approach 3: Polling with long-timeout** (fallback for restricted networks)
```
GET /v1/chains/1/blocks?since=18500000&timeout=30s

// Server holds connection for 30s until new block arrives
```

**Production considerations:**
- Use Redis Pub/Sub to fan out block events to all API servers
- Implement heartbeat (ping every 30s) to detect dead connections
- Limit concurrent WebSocket connections per IP (prevent DoS)
- Buffer last 100 blocks in memory for reconnecting clients
- Return 429 if client can't keep up with block rate"

---

### Q9: How would you design the API for a block explorer?

**Answer:**

"**Core endpoints:**

```
GET /v1/chains                        # List supported chains
GET /v1/chains/{id}/status            # Current block, indexing lag
GET /v1/chains/{id}/blocks            # Recent blocks (paginated)
GET /v1/chains/{id}/blocks/{number}   # Block detail
GET /v1/chains/{id}/transactions/{hash}  # Transaction detail
GET /v1/addresses/{address}/transactions # Address activity
GET /v1/addresses/{address}/balance   # Current balance (call node RPC)
```

**Design principles:**

1. **Versioning**: `/v1/` in path (allows breaking changes in `/v2/`)

2. **Pagination**: Cursor-based for performance
   ```json
   {
     "data": [...],
     "pagination": {
       "next_cursor": "18500000",
       "has_more": true
     }
   }
   ```

3. **Caching headers**:
   ```
   Cache-Control: public, max-age=300  // 5 min for recent blocks
   Cache-Control: public, max-age=31536000, immutable  // 1 year for old blocks
   ```

4. **Field selection**: `?fields=number,hash,timestamp` (reduce payload size)

5. **Filtering**: `?status=success&from_block=18000000&to_block=18500000`

6. **Error responses**:
   ```json
   {
     "error": {
       "code": "INVALID_CHAIN_ID",
       "message": "Chain ID 999 is not supported",
       "supported_chains": [1, 137, 42161]
     }
   }
   ```

7. **Rate limiting**: Return headers
   ```
   X-RateLimit-Limit: 100
   X-RateLimit-Remaining: 73
   X-RateLimit-Reset: 1699564800
   ```"

---

## Quick Reference

### Architecture Patterns Used

| Pattern | Usage | Benefit |
|---------|-------|---------|
| **Event-Driven** | Kafka between services | Decoupling, replay capability |
| **CQRS** | Separate read/write paths | Optimized queries |
| **Microservices** | Ingester, Processor, API | Independent scaling |
| **Database Partitioning** | Partition by chain_id | Query performance |
| **Idempotency** | All operations | Crash safety |
| **Token Bucket** | Rate limiting | Burst handling |

### Technology Choices

| Choice | Alternative | Why Chosen |
|--------|-------------|------------|
| **Go** | Rust | Faster development, I/O-bound workload |
| **Kafka** | RabbitMQ | Replay capability, throughput |
| **PostgreSQL** | Cassandra | ACID transactions, query flexibility |
| **Redis** | Memcached | Pub/Sub, data structures |
| **Monorepo** | Separate repos | Shared models, atomic changes |

### Performance Numbers to Remember

- Ethereum: ~12s block time, ~150 transactions/block
- Polygon: ~2s block time, ~50 transactions/block
- PostgreSQL: ~10K writes/sec (with proper indexes)
- Kafka: ~100K messages/sec per partition
- Redis: ~50K ops/sec (single-threaded)
- Go goroutines: Can spawn 100K+ (2KB each)

### Common Pitfalls & Solutions

| Pitfall | Solution |
|---------|----------|
| N+1 query problem | Use JOINs or batch queries |
| Missing partition key | Always filter by `chain_id` |
| No index on foreign keys | Index all JOIN columns |
| Unbounded array growth | Use pagination everywhere |
| Blocking I/O in loop | Use goroutines + sync.WaitGroup |
| Missing context timeout | Always use `context.WithTimeout` |

---

## Related Documentation

- [Go Concepts - Interview Guide](./08-go-concepts-interview.md)
- [Implementation Concepts](./06-implementation-concepts.md)
- [Technology Stack](./01-technology-stack.md)
- [Troubleshooting](./09-troubleshooting.md)
