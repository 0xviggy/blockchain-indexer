# PostgreSQL Database Guide

> ⚠️ **EDUCATIONAL MATERIAL** - Interview prep & learning reference  
> For project-specific database setup, see [../DATABASE_GUIDE.md](../DATABASE_GUIDE.md)

> **PostgreSQL-specific guide** for blockchain indexing: schema design, partitioning, performance optimization.  
> For database theory (ACID, normalization, CAP theorem, index theory), see [Database-Fundamentals.md](./Database-Fundamentals.md)

---

## Table of Contents

- [PostgreSQL Fundamentals](#postgresql-fundamentals)
- [Schema Design for Blockchain Data](#schema-design-for-blockchain-data)
- [Table Partitioning](#table-partitioning)
- [Indexing Strategies](#indexing-strategies)
- [Query Optimization](#query-optimization)
- [Connection Pooling](#connection-pooling)
- [Backup & Recovery](#backup--recovery)
- [Migration Management](#migration-management)
- [Production Deployment](#production-deployment)
- [Monitoring & Maintenance](#monitoring--maintenance)
- [Interview Questions & Answers](#interview-questions--answers)

---

## PostgreSQL Fundamentals

### Why PostgreSQL for Blockchain Indexing?

- ✅ **ACID Guarantees**: Critical for financial data integrity
- ✅ **Partitioning**: Handle billions of transactions efficiently
- ✅ **JSONB**: Flexible storage for event data and decoded parameters
- ✅ **Full-Text Search**: Search transaction data, contract ABIs
- ✅ **Mature Tooling**: pgAdmin, pg_stat_statements, EXPLAIN ANALYZE
- ✅ **Horizontal Scaling**: Read replicas, sharding support

### PostgreSQL vs Other Databases

| Feature | PostgreSQL | MySQL | MongoDB |
|---------|-----------|-------|---------|
| ACID Transactions | ✅ Full | ✅ Full | ⚠️ Limited |
| Partitioning | ✅ Native | ✅ Native | ❌ Manual sharding |
| JSONB | ✅ Indexed | ⚠️ JSON (no index) | ✅ Native |
| Full-Text Search | ✅ Built-in | ⚠️ Basic | ✅ Text indexes |
| Analytics Queries | ✅ Excellent | ⚠️ Good | ❌ Poor |
| Time-Series | ✅ TimescaleDB | ⚠️ Extensions | ❌ |

**For blockchain indexing**: PostgreSQL's partitioning + JSONB + ACID = perfect fit

---

## Schema Design for Blockchain Data

### Core Tables

```sql
-- Chains (multi-chain support)
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

-- Blocks (partitioned by chain_id)
CREATE TABLE blocks (
    chain_id INT NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    parent_hash VARCHAR(66) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    gas_used BIGINT,
    gas_limit BIGINT,
    base_fee_per_gas NUMERIC(78, 0),
    transaction_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, block_number),
    FOREIGN KEY (chain_id) REFERENCES chains(chain_id)
) PARTITION BY LIST (chain_id);

-- Transactions (partitioned by chain_id)
CREATE TABLE transactions (
    chain_id INT NOT NULL,
    block_number BIGINT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    tx_index INT NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    value NUMERIC(78, 0),
    gas_price NUMERIC(78, 0),
    gas_used BIGINT,
    status INT NOT NULL, -- 0=failed, 1=success
    block_timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, tx_hash),
    FOREIGN KEY (chain_id) REFERENCES chains(chain_id)
) PARTITION BY LIST (chain_id);

-- Events (partitioned by chain_id)
CREATE TABLE events (
    chain_id INT NOT NULL,
    transaction_hash VARCHAR(66) NOT NULL,
    log_index INT NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    event_signature VARCHAR(66),
    protocol VARCHAR(100),
    topic1 VARCHAR(66),
    topic2 VARCHAR(66),
    topic3 VARCHAR(66),
    data TEXT,
    block_number BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, transaction_hash, log_index),
    FOREIGN KEY (chain_id) REFERENCES chains(chain_id)
) PARTITION BY LIST (chain_id);
```

### Design Principles

1. **Partition by chain_id**: Isolates data per blockchain, enables parallel writes
2. **Use VARCHAR(66) for hashes**: Ethereum hashes are 0x + 64 hex chars = 66 total
3. **Use VARCHAR(42) for addresses**: 0x + 40 hex chars = 42 total
4. **Use NUMERIC(78, 0) for wei values**: Supports up to 2^256-1 (Ethereum max)
5. **Include timestamps**: Critical for time-series queries and debugging
6. **Foreign keys**: Maintain referential integrity

---

## Table Partitioning

### Why Partition?

**Performance Benefits**:
- 3x faster queries (partition pruning)
- Parallel writes (no lock contention)
- Easy data archival (drop old partitions)
- Index size reduction per partition

### Creating Partitions

```sql
-- Create parent table (already shown above)
CREATE TABLE blocks (...) PARTITION BY LIST (chain_id);

-- Create child partitions
CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks FOR VALUES IN (137);
CREATE TABLE blocks_arbitrum PARTITION OF blocks FOR VALUES IN (42161);
CREATE TABLE blocks_optimism PARTITION OF blocks FOR VALUES IN (10);
CREATE TABLE blocks_base PARTITION OF blocks FOR VALUES IN (8453);
```

### Query Behavior

```sql
-- Query automatically uses correct partition
SELECT * FROM blocks WHERE chain_id = 1 AND block_number > 18000000;
-- PostgreSQL only scans blocks_eth, not other partitions

-- Cross-chain query scans all partitions
SELECT chain_id, COUNT(*) FROM blocks GROUP BY chain_id;
```

### Partition Pruning Example

```sql
EXPLAIN SELECT * FROM blocks WHERE chain_id = 1 AND block_number > 18000000;

-- Output:
Seq Scan on blocks_eth  (cost=0.00..150.00 rows=1000 width=100)
  Filter: (block_number > 18000000)
  -- Notice: Only scans blocks_eth partition
```

---

## Indexing Strategies

### Index Types

```sql
-- B-Tree (default, best for =, <, >, <=, >=, BETWEEN)
CREATE INDEX idx_blocks_eth_number ON blocks_eth(block_number DESC);

-- Hash (only for = queries, slightly faster but no range queries)
CREATE INDEX idx_transactions_hash ON transactions USING HASH (tx_hash);

-- GIN (for JSONB, arrays, full-text search)
CREATE INDEX idx_events_data ON events USING GIN (data jsonb_path_ops);

-- BRIN (block range index, for large time-series tables)
CREATE INDEX idx_blocks_timestamp ON blocks_eth USING BRIN (timestamp);
```

### Composite Indexes

```sql
-- Most selective column first
CREATE INDEX idx_txs_from_block ON transactions(from_address, block_number DESC);

-- Query benefits from this index:
SELECT * FROM transactions 
WHERE from_address = '0x123...' 
ORDER BY block_number DESC 
LIMIT 100;
```

### Covering Indexes (INCLUDE clause)

```sql
-- Index includes extra columns (no table lookup needed)
CREATE INDEX idx_blocks_hash_include 
ON blocks(block_hash) 
INCLUDE (timestamp, gas_used);

-- Query answered entirely from index:
SELECT timestamp, gas_used FROM blocks WHERE block_hash = '0xabc...';
```

### Partial Indexes

```sql
-- Index only active chains
CREATE INDEX idx_chains_active ON chains(chain_id) WHERE enabled = TRUE;

-- Index only failed transactions
CREATE INDEX idx_txs_failed ON transactions(tx_hash) WHERE status = 0;
```

---

## Query Optimization

### EXPLAIN ANALYZE

```sql
-- See actual execution time and plan
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM transactions 
WHERE from_address = '0x123...' 
LIMIT 100;

-- Output shows:
-- - Seq Scan vs Index Scan
-- - Rows scanned vs rows returned
-- - Execution time
-- - Buffers (cache hits/misses)
```

### Common Optimization Patterns

#### 1. Use Indexes Effectively

```sql
-- ❌ Bad: Function on indexed column (index not used)
SELECT * FROM blocks WHERE EXTRACT(YEAR FROM timestamp) = 2024;

-- ✅ Good: Index-friendly query
SELECT * FROM blocks WHERE timestamp >= '2024-01-01' AND timestamp < '2025-01-01';
```

#### 2. Avoid SELECT *

```sql
-- ❌ Bad: Fetches unnecessary data
SELECT * FROM transactions WHERE status = 1 LIMIT 100;

-- ✅ Good: Fetch only needed columns
SELECT tx_hash, from_address, value FROM transactions WHERE status = 1 LIMIT 100;
```

#### 3. Use CTEs for Complex Queries

```sql
-- Break complex query into readable parts
WITH recent_blocks AS (
    SELECT chain_id, block_number, timestamp 
    FROM blocks 
    WHERE timestamp > NOW() - INTERVAL '1 hour'
)
SELECT 
    rb.chain_id,
    COUNT(DISTINCT t.tx_hash) as tx_count
FROM recent_blocks rb
JOIN transactions t ON rb.chain_id = t.chain_id AND rb.block_number = t.block_number
GROUP BY rb.chain_id;
```

### Query Performance Checklist

- [ ] Use `EXPLAIN ANALYZE` to see actual plan
- [ ] Check if indexes are being used (`Index Scan` vs `Seq Scan`)
- [ ] Add `WHERE` clause filter on partition key first
- [ ] Use `LIMIT` to avoid scanning millions of rows
- [ ] Avoid functions on indexed columns
- [ ] Use appropriate index type (B-Tree, GIN, BRIN)
- [ ] Update statistics regularly (`ANALYZE tables`)

---

## Connection Pooling

### Why Connection Pooling?

PostgreSQL creates a new OS process per connection (heavyweight). Connection pooling reuses connections.

**Without pooling**: 1000 requests = 1000 PostgreSQL processes = 💥 crash  
**With pooling**: 1000 requests = 25 pooled connections = ✅ works

### Go Implementation

```go
import (
    "database/sql"
    _ "github.com/lib/pq"
)

db, err := sql.Open("postgres", "postgresql://user:pass@localhost/db")

// Configure connection pool
db.SetMaxOpenConns(25)        // Max connections to PostgreSQL
db.SetMaxIdleConns(10)         // Keep 10 warm connections
db.SetConnMaxLifetime(5 * time.Minute)  // Recycle old connections
db.SetConnMaxIdleTime(1 * time.Minute)  // Close idle connections

// Always use context for queries
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

rows, err := db.QueryContext(ctx, "SELECT * FROM blocks LIMIT 10")
```

### PgBouncer (External Pool)

```ini
# pgbouncer.ini
[databases]
indexer = host=localhost port=5432 dbname=indexer

[pgbouncer]
listen_port = 6432
listen_addr = *
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
```

**Connection strings:**
- Direct: `postgresql://user:pass@localhost:5432/indexer` (migrations)
- Pooled: `postgresql://user:pass@localhost:6432/indexer` (application)

---

## Backup & Recovery

### pg_dump (Logical Backup)

```bash
# Full database backup
pg_dump -U indexer -d indexer -F c -f backup_$(date +%Y%m%d).dump

# Specific tables only
pg_dump -U indexer -d indexer -t blocks -t transactions -F c -f tables_backup.dump

# SQL format (human-readable)
pg_dump -U indexer -d indexer --clean --if-exists > backup.sql

# Restore
pg_restore -U indexer -d indexer -F c backup_20251127.dump
```

### Continuous Archiving (Point-in-Time Recovery)

```bash
# Enable WAL archiving in postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'cp %p /archive/%f'

# Take base backup
pg_basebackup -U replication -D /backup/base -F tar -z -P

# Recover to specific point in time
recovery_target_time = '2025-11-27 12:00:00'
```

### Supabase Automatic Backups

- **Free tier**: Daily backups, 7-day retention
- **Pro tier**: Point-in-time recovery, 30-day retention
- **Restore via Dashboard**: Projects → Database → Backups

---

## Migration Management

### golang-migrate Setup

```bash
# Install
brew install golang-migrate

# Create migration
migrate create -ext sql -dir database/migrations -seq add_user_table

# Apply migrations
migrate -path database/migrations \
  -database "postgresql://user:pass@localhost:5432/db" up

# Rollback
migrate -path database/migrations \
  -database "postgresql://user:pass@localhost:5432/db" down 1

# Check version
migrate -path database/migrations \
  -database "postgresql://user:pass@localhost:5432/db" version
```

### Migration Best Practices

1. **Always create .up and .down files**
2. **Use transactions** (BEGIN/COMMIT)
3. **Make idempotent** (IF EXISTS, IF NOT EXISTS)
4. **Test rollbacks** before production
5. **Never modify existing migrations** after deployment

---

## Production Deployment

### Supabase (Recommended for Dev/Small Production)

**Pros**:
- ✅ Free tier (500MB database)
- ✅ Automatic backups
- ✅ Connection pooling (PgBouncer)
- ✅ Monitoring dashboard
- ✅ API auto-generated

**Setup**:
```bash
# 1. Create project at supabase.com
# 2. Copy connection string
# 3. Run migrations
export DATABASE_URL="postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:5432/postgres"
make migrate-up
```

**Pricing**:
- Free: 500MB, 2GB bandwidth
- Pro: $25/mo, 8GB, 50GB bandwidth
- Team: $599/mo, 100GB, 250GB bandwidth

### AWS RDS PostgreSQL

**Pros**:
- ✅ Production-grade
- ✅ Automated backups
- ✅ Read replicas
- ✅ Multi-AZ failover

**Pricing**: ~$30-200/month depending on instance size

### Self-Hosted

```yaml
# Docker Compose production setup
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: indexer
      POSTGRES_USER: indexer
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    command:
      - "postgres"
      - "-c" 
      - "shared_buffers=256MB"
      - "-c"
      - "max_connections=200"
      - "-c"
      - "effective_cache_size=1GB"
```

---

## Monitoring & Maintenance

### Key Metrics to Track

```sql
-- Database size
SELECT pg_size_pretty(pg_database_size('indexer'));

-- Table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Slow queries (requires pg_stat_statements extension)
SELECT 
    query,
    calls,
    mean_exec_time,
    total_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Connection count
SELECT count(*) FROM pg_stat_activity;

-- Cache hit ratio (should be >99%)
SELECT 
    sum(heap_blks_read) as heap_read,
    sum(heap_blks_hit) as heap_hit,
    sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) as ratio
FROM pg_statio_user_tables;
```

### Regular Maintenance Tasks

```sql
-- Update statistics (helps query planner)
ANALYZE blocks;
ANALYZE transactions;

-- Reclaim space (after large deletes)
VACUUM ANALYZE blocks;

-- Full vacuum (locks table, rarely needed)
VACUUM FULL transactions;

-- Reindex (if index bloat)
REINDEX TABLE blocks;
```

### Automated Maintenance

```sql
-- Enable autovacuum (default in modern PostgreSQL)
ALTER TABLE blocks SET (autovacuum_enabled = true);
ALTER TABLE transactions SET (autovacuum_vacuum_scale_factor = 0.1);
```

---

## Interview Questions & Answers

### Q1: How do you handle billions of blockchain transactions in PostgreSQL?

**Answer:**

Use table partitioning by `chain_id` and time-based partitioning:

```sql
-- Partition by chain_id (5 chains = 5 partitions)
CREATE TABLE transactions (...) PARTITION BY LIST (chain_id);

-- Further partition by month for time-series data
CREATE TABLE transactions_eth PARTITION OF transactions 
FOR VALUES IN (1) 
PARTITION BY RANGE (block_timestamp);

CREATE TABLE transactions_eth_2024_11 PARTITION OF transactions_eth
FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
```

**Benefits**:
- Queries only scan relevant partition (100x faster)
- Can archive old months to S3 (DROP PARTITION)
- Parallel writes (no lock contention)
- Index size per partition stays small

---

### Q2: How would you optimize query performance for address transaction history?

**Answer:**

Multi-pronged approach:

1. **Composite Index** (most important):
```sql
CREATE INDEX idx_txs_from_addr ON transactions(from_address, block_number DESC);
CREATE INDEX idx_txs_to_addr ON transactions(to_address, block_number DESC);
```

2. **Cursor-Based Pagination** (not OFFSET):
```sql
-- ❌ Bad: OFFSET 1000000 scans 1M rows
SELECT * FROM transactions 
WHERE from_address = '0x123...' 
ORDER BY block_number DESC 
OFFSET 1000000 LIMIT 20;

-- ✅ Good: Cursor-based (index seek)
SELECT * FROM transactions 
WHERE from_address = '0x123...' 
  AND block_number < 18500000  -- cursor from previous page
ORDER BY block_number DESC 
LIMIT 20;
```

3. **Redis Caching** for hot addresses:
```go
key := fmt.Sprintf("txs:%s:recent", address)
if cached := redis.Get(key); cached != nil {
    return cached
}
// Query DB, cache for 5 minutes
```

4. **Materialized View** for top addresses:
```sql
CREATE MATERIALIZED VIEW address_stats AS
SELECT 
    address,
    COUNT(*) as tx_count,
    MAX(block_number) as last_active
FROM (
    SELECT from_address as address FROM transactions
    UNION ALL
    SELECT to_address FROM transactions WHERE to_address IS NOT NULL
) GROUP BY address;

-- Refresh periodically
REFRESH MATERIALIZED VIEW CONCURRENTLY address_stats;
```

---

### Q3: How do you handle database schema changes in production?

**Answer:**

Use `golang-migrate` with zero-downtime migrations:

1. **Backwards Compatible Changes** (safe):
```sql
-- Add new column (nullable)
ALTER TABLE transactions ADD COLUMN nonce BIGINT;

-- Add new index (concurrently, no table lock)
CREATE INDEX CONCURRENTLY idx_txs_nonce ON transactions(nonce);

-- Add new table
CREATE TABLE token_transfers (...);
```

2. **Breaking Changes** (requires app coordination):
```sql
-- Phase 1: Add new column
ALTER TABLE transactions ADD COLUMN status_v2 VARCHAR(20);

-- Phase 2: Backfill data
UPDATE transactions SET status_v2 = CASE status WHEN 1 THEN 'success' ELSE 'failed' END;

-- Phase 3: Deploy app to use status_v2

-- Phase 4: Drop old column
ALTER TABLE transactions DROP COLUMN status;
ALTER TABLE transactions RENAME COLUMN status_v2 TO status;
```

3. **Migration Testing**:
```bash
# Apply migration
make migrate-up

# Test rollback
make migrate-down

# Re-apply
make migrate-up
```

4. **Production Deployment**:
```yaml
# CI/CD pipeline
- name: Run migrations
  run: make migrate-up
- name: Deploy application
  run: railway up
```

---

### Q4: What's the difference between VACUUM and ANALYZE?

**Answer:**

**VACUUM**: Reclaims storage space from deleted/updated rows
```sql
-- PostgreSQL doesn't actually delete rows immediately
DELETE FROM transactions WHERE block_number < 10000000;
-- Rows marked as "dead tuples" but space not freed

-- VACUUM reclaims space
VACUUM transactions;

-- VACUUM FULL locks table and rewrites (avoid in production)
VACUUM FULL transactions;
```

**ANALYZE**: Updates statistics for query planner
```sql
-- Query planner uses statistics to choose best execution plan
ANALYZE transactions;

-- Check statistics
SELECT 
    tablename,
    n_live_tup,
    n_dead_tup,
    last_vacuum,
    last_analyze
FROM pg_stat_user_tables
WHERE tablename = 'transactions';
```

**When to run**:
- VACUUM: After large DELETE/UPDATE operations
- ANALYZE: After bulk INSERT, before running complex queries
- **autovacuum** does both automatically in modern PostgreSQL

---

### Q5: How do you implement read replicas for scaling?

**Answer:**

**Architecture**:
```
Application
├── Write queries → Primary (Master) Database
└── Read queries → Read Replicas (Slaves)
```

**PostgreSQL Streaming Replication**:
```sql
-- On primary
ALTER SYSTEM SET wal_level = replica;
ALTER SYSTEM SET max_wal_senders = 3;
CREATE USER replicator REPLICATION LOGIN PASSWORD 'secret';

-- On replica
CREATE USER replicator WITH REPLICATION PASSWORD 'secret';
```

**Go Application Code**:
```go
// Primary database (writes)
primaryDB, _ := sql.Open("postgres", PRIMARY_DB_URL)

// Read replicas
replica1DB, _ := sql.Open("postgres", REPLICA1_DB_URL)
replica2DB, _ := sql.Open("postgres", REPLICA2_DB_URL)

// Route queries
func (db *DB) GetBlocks() []Block {
    // Use replica for reads (round-robin)
    replica := selectReplica()
    return replica.Query("SELECT * FROM blocks ...")
}

func (db *DB) InsertBlock(block Block) error {
    // Use primary for writes
    return primaryDB.Exec("INSERT INTO blocks ...")
}
```

**Replication Lag Monitoring**:
```sql
-- On replica
SELECT 
    pg_last_wal_receive_lsn() - pg_last_wal_replay_lsn() AS lag_bytes,
    EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())) AS lag_seconds;
```

---

### Q6: What are the pros/cons of JSONB vs separate columns?

**Answer:**

**JSONB Pros**:
- ✅ Flexible schema (add fields without migrations)
- ✅ Store nested data (decoded events, ABIs)
- ✅ Indexed with GIN indexes
- ✅ Query with JSONPath

**JSONB Cons**:
- ❌ Slower than native columns
- ❌ No foreign keys on JSON fields
- ❌ Harder to analyze (can't see schema easily)

**Separate Columns Pros**:
- ✅ Faster queries
- ✅ Type safety (INT, VARCHAR constraints)
- ✅ Foreign keys
- ✅ Better statistics for query planner

**Separate Columns Cons**:
- ❌ Migrations needed for schema changes
- ❌ Can't store nested/variable data

**Best Practice**:
```sql
-- Use columns for frequently queried, fixed fields
CREATE TABLE transactions (
    tx_hash VARCHAR(66) PRIMARY KEY,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    value NUMERIC(78, 0),
    status INT NOT NULL,
    
    -- Use JSONB for flexible/nested data
    decoded_input JSONB,
    raw_receipt JSONB
);

-- Index JSONB fields you query
CREATE INDEX idx_decoded_function ON transactions 
USING GIN ((decoded_input->'function_name'));

-- Query JSONB
SELECT * FROM transactions 
WHERE decoded_input->>'function_name' = 'swap';
```

---

## Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/current/)
- [Use The Index, Luke](https://use-the-index-luke.com/) - SQL indexing guide
- [postgres.ai](https://postgres.ai/) - Database optimization platform
- [Supabase Docs](https://supabase.com/docs)
- [golang-migrate](https://github.com/golang-migrate/migrate)
