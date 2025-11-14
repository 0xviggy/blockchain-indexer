# Learning Guide & Interview Preparation

This document captures all setup steps, learning points, architectural decisions, and technical concepts for interview preparation.

---

## Table of Contents
1. [System Architecture Overview](#system-architecture-overview)
2. [Setup Steps & Commands](#setup-steps--commands)
3. [Key Technical Concepts](#key-technical-concepts)
4. [Design Decisions & Trade-offs](#design-decisions--trade-offs)
5. [Common Interview Questions](#common-interview-questions)
6. [Troubleshooting Guide](#troubleshooting-guide)
7. [Performance Optimization](#performance-optimization)
8. [Production Readiness](#production-readiness)

---

## System Architecture Overview

### High-Level Flow
```
Blockchain → Ingester → Kafka → Processor → PostgreSQL → API → Users
                                              ↓
                                           Redis Cache
```

### Component Responsibilities
- **Ingester**: Fetches blocks from blockchain RPC nodes, detects reorgs, publishes to Kafka
- **Processor**: Consumes Kafka messages, parses events (ERC20/ERC721), writes to PostgreSQL
- **API**: Serves data via REST/WebSocket with caching and rate limiting

### Why This Architecture?
- **Event-driven**: Decouples components, enables independent scaling
- **Message broker**: Handles backpressure, provides replay capability
- **Caching layer**: Reduces database load for hot data
- **Partitioned storage**: Enables efficient queries on large datasets

---

## Setup Steps & Commands

### Prerequisites
```bash
# Check versions
go version        # Should be 1.21+
docker --version  # Should be 20.10+
psql --version    # Should be 15+

# Install dependencies
brew install golang postgresql docker
```

### Initial Project Setup
```bash
# Clone and navigate
cd /home/dev/dev/sdapp/indexer

# Initialize Go modules for each service
cd services/ingester && go mod init github.com/yourorg/indexer/ingester
cd ../processor && go mod init github.com/yourorg/indexer/processor
cd ../api && go mod init github.com/yourorg/indexer/api

# Install shared dependencies
cd ../../shared && go mod tidy
```

### Docker Infrastructure
```bash
# Start all dependencies
docker-compose up -d postgres redis kafka

# Verify services are running
docker ps

# Check logs
docker logs indexer-postgres
docker logs indexer-kafka
```

### Database Setup
```bash
# Create database
psql -U postgres -c "CREATE DATABASE indexer;"
psql -U postgres -c "CREATE USER indexer WITH PASSWORD 'password';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE indexer TO indexer;"

# Run migrations
make migrate

# Verify tables
psql -U indexer -d indexer -c "\dt"
```

### Running Services
```bash
# Terminal 1: Ingester
make run-ingester

# Terminal 2: Processor
make run-processor

# Terminal 3: API
make run-api

# Or run all with Docker
make docker-up
```

---

## Key Technical Concepts

### 1. Blockchain Reorg Handling

**What is a Reorg?**
A blockchain reorganization occurs when a competing chain becomes longer than the current canonical chain, causing blocks to be replaced.

**Detection Algorithm**:
```go
// Compare parent hash of new block with stored block hash
if storedBlock.Hash != newBlock.ParentHash {
    // Reorg detected! Find common ancestor
    rollbackToBlock := findCommonAncestor(db, rpc)
    
    // Delete conflicting data
    db.DeleteBlocksAfter(rollbackToBlock)
    
    // Re-index from common ancestor
    resumeIngestion(rollbackToBlock + 1)
}
```

**Interview Points**:
- Ethereum finality: ~13 minutes (2 epochs), so reorgs typically affect last 15-20 blocks
- Polygon has shorter finality: ~256 blocks
- Always track parent hashes to detect reorgs
- Use database transactions for atomic rollback

### 2. Event Parsing (ERC20 Transfer Example)

**Event Signature**:
```solidity
// ERC20 Transfer event
event Transfer(address indexed from, address indexed to, uint256 value);

// Keccak256 hash of signature
0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
```

**Parsing Logic**:
```go
// topics[0] = event signature
// topics[1] = from address (indexed)
// topics[2] = to address (indexed)
// data = value (non-indexed, 32 bytes)

func parseTransfer(log *types.Log) (*TokenTransfer, error) {
    if len(log.Topics) < 3 {
        return nil, errors.New("invalid transfer event")
    }
    
    return &TokenTransfer{
        From:   common.BytesToAddress(log.Topics[1].Bytes()),
        To:     common.BytesToAddress(log.Topics[2].Bytes()),
        Amount: new(big.Int).SetBytes(log.Data),
    }, nil
}
```

**Interview Points**:
- Indexed parameters go in topics (max 3 indexed params)
- Non-indexed parameters go in data field
- ABI encoding/decoding is crucial for custom events
- Use go-ethereum's `abigen` for type-safe parsing

### 3. Database Partitioning Strategy

**Why Partition?**
- Faster queries on recent data
- Easier archival of old data
- Better maintenance (vacuum, reindex)

**Implementation**:
```sql
-- Partition by chain_id and block_number range
CREATE TABLE blocks (
    chain_id INT NOT NULL,
    block_number BIGINT NOT NULL,
    -- other columns
) PARTITION BY RANGE (chain_id, block_number);

-- Create partitions for each chain
CREATE TABLE blocks_eth_0_to_1m PARTITION OF blocks
    FOR VALUES FROM (1, 0) TO (1, 1000000);

CREATE TABLE blocks_eth_1m_to_2m PARTITION OF blocks
    FOR VALUES FROM (1, 1000000) TO (1, 2000000);
```

**Interview Points**:
- Query planner automatically selects correct partition
- Can drop old partitions for archival
- Composite key (chain_id, block_number) enables multi-chain partitioning

### 4. Rate Limiting with Token Bucket

**Algorithm**:
- Bucket has capacity N tokens
- Tokens refill at rate R per second
- Each request consumes 1 token
- Request denied if bucket is empty

**Implementation**:
```go
type TokenBucket struct {
    tokens       float64
    capacity     float64
    refillRate   float64 // tokens per second
    lastRefill   time.Time
    mu           sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()
    
    // Refill tokens
    tb.tokens = math.Min(tb.capacity, tb.tokens + elapsed * tb.refillRate)
    tb.lastRefill = now
    
    if tb.tokens >= 1.0 {
        tb.tokens -= 1.0
        return true
    }
    return false
}
```

**Interview Points**:
- More flexible than fixed window (allows bursts)
- Can implement per-IP and per-API-key limiting
- Use Redis for distributed rate limiting

### 5. Kafka Message Ordering

**Challenge**: Maintain block ordering across consumers

**Solution**: Partition by chain_id
```go
// Producer
producer.SendMessage(&sarama.ProducerMessage{
    Topic: "raw.blocks",
    Key:   sarama.StringEncoder(fmt.Sprintf("%d", chainID)),
    Value: sarama.ByteEncoder(blockData),
})

// Same chain_id always goes to same partition
// Ensures ordering within a chain
```

**Interview Points**:
- Kafka guarantees ordering within a partition
- Use chain_id as message key for consistent partitioning
- Consumer group ensures each partition is consumed by one consumer
- Enables parallel processing across chains

---

## Design Decisions & Trade-offs

### 1. Language Choice: Go vs Rust

**Decision**: Go

**Rationale**:
- ✅ `go-ethereum` is the official Ethereum client library
- ✅ Faster development velocity (simpler syntax, faster compile)
- ✅ Better hiring pool for blockchain teams
- ✅ Sufficient performance (not CPU-bound, mostly I/O-bound)
- ❌ Rust has lower memory footprint (~30% less)
- ❌ Rust has slightly faster execution (~20-30%)

**Interview Answer**: "We chose Go because the blockchain indexer is I/O-bound (network calls, database writes), not CPU-bound. Go's goroutines handle concurrency well, and go-ethereum is the canonical implementation. The ~20% performance difference with Rust doesn't justify the slower development and hiring challenges."

### 2. Message Broker: Kafka vs RabbitMQ

**Decision**: Kafka

**Rationale**:
- ✅ Better for high-throughput streaming data
- ✅ Built-in partitioning for parallel processing
- ✅ Message replay capability (important for reorgs)
- ✅ Better log retention for debugging
- ❌ More complex setup than RabbitMQ
- ❌ Higher resource usage

**Interview Answer**: "Kafka is better suited for append-only event streams like blockchain data. The replay capability is crucial for handling reorgs, and partitioning enables parallel processing across chains. RabbitMQ excels at task queues with complex routing, which isn't our use case."

### 3. Database: PostgreSQL vs TimescaleDB vs Cassandra

**Decision**: PostgreSQL with partitioning

**Rationale**:
- ✅ Strong consistency (important for financial data)
- ✅ Complex query support (joins, aggregations)
- ✅ Mature ecosystem and tooling
- ✅ Partitioning handles time-series nature
- ❌ TimescaleDB has better time-series optimizations
- ❌ Cassandra has better horizontal scaling

**Interview Answer**: "PostgreSQL offers the right balance of consistency, query flexibility, and operational maturity. The data has relational aspects (blocks → transactions → events), and users need complex queries. Native partitioning handles the time-series aspect adequately. We can shard by chain_id if we need horizontal scaling later."

### 4. Monorepo vs Separate Repos

**Decision**: Monorepo

**Rationale**:
- ✅ Shared models/config across services
- ✅ Atomic changes across services
- ✅ Simpler dependency management
- ✅ Easier local development
- ❌ Larger repo size
- ❌ All services version together

**Interview Answer**: "A monorepo makes sense for tightly coupled microservices. All three services share data models and evolve together. Separate repos would require versioning shared packages and coordinating deployments. As the team grows, we could split out services."

---

## Common Interview Questions

### Q1: How do you handle blockchain reorganizations?

**Answer**:
"We detect reorgs by comparing parent hashes. When ingesting block N, we check if our stored block N-1's hash matches the new block's parent hash. If not, we've detected a reorg.

We then:
1. Find the common ancestor by walking back through parent hashes
2. Use a database transaction to delete all blocks after the common ancestor
3. Resume ingestion from the common ancestor + 1
4. Kafka's replay capability allows the processor to re-process affected events

For production, we only mark blocks as 'finalized' after the chain-specific finality threshold (e.g., 13 minutes for Ethereum, 256 blocks for Polygon)."

### Q2: How do you scale the ingester for multiple chains?

**Answer**:
"We deploy separate ingester instances per chain, each with:
- Dedicated checkpoint tracking (last_indexed_block per chain)
- Chain-specific configuration (RPC URLs, finality rules, block times)
- Independent failure domains (one chain's RPC issues don't affect others)

Each ingester publishes to a chain-specific Kafka topic (e.g., 'eth.blocks', 'poly.blocks'). This enables:
- Parallel processing across chains
- Independent scaling per chain (Ethereum might need 3 ingesters, Polygon 1)
- Chain-specific monitoring and alerting

For efficiency, we could run multiple chains in one process with goroutines, but separate deployments provide better isolation."

### Q3: How do you ensure data consistency during high load?

**Answer**:
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

### Q4: How would you optimize query performance for address transaction history?

**Answer**:
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

### Q5: How do you handle API rate limiting at scale?

**Answer**:
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

---

## Troubleshooting Guide

### Issue: Ingester falling behind chain head

**Symptoms**: `last_indexed_block` is 1000+ blocks behind current block

**Diagnosis**:
```bash
# Check ingestion rate
curl http://localhost:9090/metrics | grep blocks_ingested_total

# Check RPC latency
curl http://localhost:9090/metrics | grep rpc_request_duration
```

**Solutions**:
1. Add more RPC providers (load balance)
2. Increase batch size in config
3. Scale horizontally (multiple ingesters with block range sharding)
4. Check network connectivity to RPC provider

### Issue: Processor consumer lag

**Symptoms**: Kafka consumer lag increasing

**Diagnosis**:
```bash
# Check consumer group lag
kafka-consumer-groups --bootstrap-server localhost:9092 \
    --describe --group block_processor
```

**Solutions**:
1. Increase processor concurrency in config
2. Optimize database batch inserts (increase batch size)
3. Add database indexes for faster writes
4. Scale processor horizontally (Kafka will rebalance)
5. Check database connection pool settings

### Issue: API slow response times

**Symptoms**: P95 latency >1 second

**Diagnosis**:
```bash
# Check slow queries
psql -U indexer -d indexer -c "
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC LIMIT 10;"

# Check cache hit rate
redis-cli INFO stats | grep hit_rate
```

**Solutions**:
1. Add missing database indexes
2. Increase Redis cache TTL
3. Enable query result caching
4. Use connection pooling
5. Add read replicas for queries

---

## Performance Optimization

### Database Optimizations

```sql
-- 1. Analyze tables regularly
ANALYZE blocks;
ANALYZE transactions;
ANALYZE events;

-- 2. Vacuum to reclaim space
VACUUM FULL events;

-- 3. Reindex periodically
REINDEX TABLE events;

-- 4. Check index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0;  -- Unused indexes

-- 5. Connection pooling
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
```

### Go Performance Tips

```go
// 1. Use sync.Pool for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

// 2. Pre-allocate slices
events := make([]Event, 0, 1000)

// 3. Use buffered channels
ch := make(chan Block, 100)

// 4. Limit goroutines with worker pool
sem := make(chan struct{}, 10) // Max 10 concurrent workers

// 5. Profile in production
import _ "net/http/pprof"
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

### Kafka Optimizations

```yaml
# Producer settings
acks: 1  # Leader acknowledgment (balance between speed and durability)
compression.type: snappy  # Reduce network bandwidth
batch.size: 65536  # Larger batches
linger.ms: 10  # Wait up to 10ms to batch messages

# Consumer settings
fetch.min.bytes: 50000  # Fetch at least 50KB
fetch.max.wait.ms: 500  # Wait up to 500ms to accumulate data
max.poll.records: 1000  # Process 1000 messages per poll
```

---

## Production Readiness

### Deployment Checklist

- [ ] Database migrations tested in staging
- [ ] Environment variables documented
- [ ] Secrets in vault (not environment variables)
- [ ] Health check endpoints implemented
- [ ] Metrics exported to Prometheus
- [ ] Logs structured (JSON format)
- [ ] Distributed tracing enabled (Jaeger)
- [ ] Rate limiting configured
- [ ] CORS configured for API
- [ ] TLS/SSL certificates installed
- [ ] Backup strategy defined
- [ ] Disaster recovery plan documented
- [ ] Monitoring alerts configured
- [ ] On-call runbook created
- [ ] Load testing completed
- [ ] Security audit performed

### Monitoring Alerts

```yaml
alerts:
  - name: IngesterLagging
    condition: last_indexed_block < chain_head_block - 100
    severity: warning
    
  - name: ProcessorConsumerLag
    condition: kafka_consumer_lag > 10000
    severity: critical
    
  - name: HighAPILatency
    condition: http_request_duration_p95 > 1s
    severity: warning
    
  - name: DatabaseConnectionPoolExhausted
    condition: db_connections_available < 5
    severity: critical
    
  - name: HighErrorRate
    condition: error_rate > 1%
    severity: warning
```

### Capacity Planning

**Current Setup** (per chain):
- Ingester: 2 vCPU, 4GB RAM → ~100 blocks/sec
- Processor: 4 vCPU, 8GB RAM → ~200 blocks/sec
- API: 4 vCPU, 8GB RAM → ~1000 req/sec
- PostgreSQL: 8 vCPU, 32GB RAM → ~10K writes/sec
- Kafka: 4 vCPU, 16GB RAM → ~100K msg/sec

**Scaling Strategy**:
- Horizontal: Add more ingester/processor/API instances
- Vertical: Increase database resources first
- Sharding: Partition database by chain_id when single DB hits limits

---

## Additional Resources

### Official Documentation
- [go-ethereum docs](https://geth.ethereum.org/docs)
- [Kafka documentation](https://kafka.apache.org/documentation/)
- [PostgreSQL documentation](https://www.postgresql.org/docs/)

### Learning Resources
- [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)
- [System Design Interview](https://www.amazon.com/System-Design-Interview-insiders-Second/dp/B08CMF2CQF)
- [Designing Data-Intensive Applications](https://dataintensive.net/)

### Similar Projects
- [The Graph](https://thegraph.com/) - Decentralized indexing protocol
- [Etherscan](https://etherscan.io/) - Blockchain explorer
- [Dune Analytics](https://dune.com/) - Blockchain analytics platform

---

## Updates Log

| Date | Topic | Notes |
|------|-------|-------|
| 2025-11-14 | Initial setup | Created project structure, defined specs |
| | | |

---

_This document should be updated as we implement features and learn new concepts._
