# Architecture & Design Decisions

**Project**: Multi-Chain Blockchain Indexer  
**Last Updated**: December 20, 2025

This document captures key architectural decisions, multi-chain strategy, and implementation rationale.

> **For detailed implementation**: See [docs/TECHNICAL_SPEC.md](docs/TECHNICAL_SPEC.md)  
> **For setup guides**: See [docs/setup/](docs/setup/)

---

## Table of Contents
- [Core Architecture Decisions](#core-architecture-decisions)
- [Multi-Chain Support Strategy](#multi-chain-support-strategy)
- [Anti-Patterns Avoided](#anti-patterns-avoided)
- [Future Considerations](#future-considerations)

---

## Core Architecture Decisions

### 1. Direct PostgreSQL Writes (No Message Queue)

**Decision**: Ingester writes directly to PostgreSQL instead of using Kafka/RabbitMQ

**Rationale**:
- Simpler architecture with fewer moving parts
- Reduced operational overhead (no Kafka cluster to manage)
- Lower latency (no queue hop between ingester and storage)
- Adequate for current scale (5 chains, ~100 blocks/sec total)
- PostgreSQL handles concurrent writes well with partitioned tables

**Trade-offs**:
- ❌ Less flexible for adding processors/consumers
- ❌ No built-in replay capability (must query database)
- ❌ Tighter coupling between ingester and database
- ✅ Simpler debugging and data flow
- ✅ Lower infrastructure costs
- ✅ Faster development iteration

**When to revisit**: If we index >10 chains or need event processing pipeline with multiple consumers

**Code Reference**: [services/ingester/main.go](services/ingester/main.go) - Direct database inserts in `saveBlock()` and `saveTransactions()`

---

### 2. Language: Go (Not Rust)

**Decision**: Go for all backend services

**Rationale**:
- **Ecosystem fit**: go-ethereum is the canonical Ethereum client implementation
- **I/O-bound workload**: Blockchain indexing waits on RPC calls and database writes, not CPU-intensive
- **Development velocity**: Simpler syntax, faster prototyping, easier onboarding
- **Concurrency**: Goroutines make multi-chain indexing trivial (one goroutine per chain)
- **Hiring**: Larger talent pool compared to Rust

**Trade-offs**:
- ❌ Lower performance ceiling than Rust (but not relevant for I/O-bound tasks)
- ❌ No zero-copy deserialization (but network I/O is bottleneck anyway)
- ✅ Faster development (3-5x by some estimates)
- ✅ Better library support for our stack (Gin, lib/pq, ethclient)
- ✅ Good enough performance (handles mainnet 15M txs/day easily)

**When to revisit**: If we build CPU-intensive components (MEV detection, cryptographic operations)

**Code Reference**: All services in [services/](services/) directory

---

### 3. Database: PostgreSQL with Partitioning

**Decision**: PostgreSQL 15 with LIST partitioning by `chain_id`

**Rationale**:
- **ACID transactions**: Critical for reorg rollback atomicity
- **Partitioning**: Each chain gets dedicated partitions (e.g., `blocks_eth`, `blocks_polygon`)
- **Query power**: Complex joins and aggregations for analytics
- **JSONB support**: Flexible storage for event data without schema rigidity
- **Mature ecosystem**: Battle-tested with excellent tooling

**Partition Strategy**:
```sql
CREATE TABLE blocks (...) PARTITION BY LIST (chain_id);
CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks FOR VALUES IN (137);
```

**Trade-offs**:
- ✅ Query planner auto-selects partition when filtering by `chain_id`
- ✅ Can drop/archive old partitions per chain independently
- ✅ Scales to billions of rows with proper indexing
- ❌ Requires `chain_id` in all queries for partition pruning
- ❌ Manual partition creation for new chains (5 min task)

**Alternatives Considered**:
- **TimescaleDB**: Overkill for our time-series patterns, would add unnecessary complexity
- **Cassandra**: Eventual consistency unacceptable for financial data, no join support

**Code Reference**: [database/migrations/000001_initial_schema.up.sql](database/migrations/000001_initial_schema.up.sql)

---

### 4. No Shared Code Between Services

**Decision**: Each service duplicates model definitions instead of importing from `shared/`

**Rationale**:
- **Service independence**: Each service is a standalone binary with its own `go.mod`
- **Deployment flexibility**: Can deploy/version services independently
- **Type specificity**: API needs different fields than ingester (e.g., JSON tags vs. DB tags)
- **Go modules**: Cleaner than workspace modules for our structure

**Implementation**:
- API models inline in [services/api/main.go](services/api/main.go)
- Ingester uses go-ethereum types directly
- `shared/models/models.go` exists but rarely used

**Trade-offs**:
- ❌ Model duplication across services
- ❌ Must sync changes manually
- ✅ No shared module versioning headaches
- ✅ Each service optimized for its use case
- ✅ Easier to reason about service boundaries

**When to revisit**: If model drift causes bugs, consider code generation from schema

---

### 5. WebSocket + HTTP Fallback for Block Ingestion

**Decision**: Try WebSocket subscription first, fall back to HTTP polling

**Rationale**:
- **Real-time**: WebSocket `newHeads` subscription provides instant block notifications
- **Efficiency**: Reduces RPC calls by 10-100x compared to polling
- **Reliability**: HTTP polling as fallback ensures ingestion continues if WS fails
- **RPC provider variability**: Not all providers have stable WebSocket endpoints

**Implementation Pattern**:
```go
if wsClient != nil {
    headers := make(chan *types.Header)
    sub := wsClient.SubscribeNewHead(ctx, headers)
    // Process headers from channel
} else {
    // Fall back to polling every N seconds
    ticker := time.NewTicker(pollInterval)
}
```

**Trade-offs**:
- ✅ Best-case latency: sub-second block detection
- ✅ Worst-case: Still functional with HTTP polling
- ❌ Added code complexity for dual-path ingestion
- ❌ WebSocket connection management (reconnection logic)

**Code Reference**: [services/ingester/main.go](services/ingester/main.go):272 - `ingestChain()`

---

### 6. Transaction Receipts: Separate RPC Calls

**Decision**: Fetch receipts via `eth_getTransactionReceipt` for each transaction

**Rationale**:
- **Block data limitation**: `eth_getBlockByNumber` doesn't include transaction status
- **Status accuracy**: Receipt contains `status`, `gasUsed`, and `logs`
- **No alternative**: Only way to get transaction outcomes from RPC

**Implementation**:
- Fetch block with transactions
- For each transaction, call `eth_getTransactionReceipt`
- Store combined data (transaction + receipt) in database

**Trade-offs**:
- ❌ RPC calls increase linearly with transactions (1000 txs = 1000 receipt calls)
- ❌ Slower ingestion (100ms per receipt call = 100 seconds for 1000 txs)
- ✅ Accurate transaction status (critical for users)
- ✅ Complete event logs (needed for protocol tracking)

**Optimization**: Batch receipt calls with `eth_getTransactionReceiptBatch` (not standard RPC)

**Code Reference**: [services/ingester/main.go](services/ingester/main.go) - Receipt fetching in transaction loop

---

### 7. Skipped Blocks Tracking Table

**Decision**: Dedicated `skipped_blocks` table to track unfetched block ranges

**Rationale**:
- **Failure recovery**: Network errors leave gaps in indexed blocks
- **Audit trail**: Know exactly which blocks are missing
- **Backfill strategy**: Can prioritize filling gaps during low-activity periods
- **Monitoring**: Alert on growing skip count

**Schema**:
```sql
CREATE TABLE skipped_blocks (
    chain_id INT,
    start_block BIGINT,
    end_block BIGINT,
    reason TEXT,
    created_at TIMESTAMP
);
```

**Trade-offs**:
- ✅ Clear visibility into indexing gaps
- ✅ Enables systematic backfill process
- ❌ Additional database writes on failures
- ❌ Must implement backfill logic (future work)

**Code Reference**: [database/migrations/000003_add_skipped_blocks.up.sql](database/migrations/000003_add_skipped_blocks.up.sql)

---

### 8. Migration Tool: golang-migrate (Not Go Migrate)

**Decision**: Use `golang-migrate` CLI tool exclusively

**Rationale**:
- **Naming confusion**: Multiple tools named "Go Migrate"
- **Standardization**: golang-migrate is most popular, actively maintained
- **SQL-first**: Migrations are SQL files, not Go code
- **CLI simplicity**: `migrate -path ./migrations -database $DB_URL up`

**Workflow**:
```bash
make migrate-up      # Apply migrations
make migrate-down    # Rollback
make migrate-create  # New migration
```

**Trade-offs**:
- ✅ Simple, predictable migration workflow
- ✅ Version control friendly (plain SQL)
- ❌ No programmatic migrations (e.g., data transformations in Go)
- ❌ Must install separate CLI tool

**Code Reference**: [Makefile](Makefile) migration commands, [DATABASE_GUIDE.md](DATABASE_GUIDE.md)

---

### 9. Sample Data Strategy (Seed Files)

**Decision**: Provide SQL seed files for UI/API development without running ingester

**Rationale**:
- **Developer experience**: New devs can see working UI immediately
- **No RPC keys needed**: Eliminates barrier to entry
- **Testing**: Consistent test data across environments
- **Demo readiness**: Show features without blockchain sync wait

**Implementation**:
- [database/seeds/001_sample_blocks.sql](database/seeds/001_sample_blocks.sql) with ~100 blocks
- `make db-seed` loads data
- `make db-clear-seeds` removes sample data (keeps real indexed data)

**Trade-offs**:
- ✅ Frictionless onboarding (< 10 min to working UI)
- ✅ Enables frontend work without backend running
- ❌ Must maintain seed data separately
- ❌ Risk of seed data diverging from schema

**Code Reference**: [database/seeds/](database/seeds/), [Makefile](Makefile) seed commands

---

### 10. Monorepo with Independent go.mod Files

**Decision**: Single repository, but each service has its own `go.mod`

**Rationale**:
- **Codebase unity**: All code in one repo (docs, scripts, services, frontend)
- **Service independence**: Each service is a standalone binary
- **Deployment isolation**: Deploy services independently without Go workspace complexity
- **Clearer boundaries**: Prevents unintentional coupling

**Structure**:
```
blockchain-indexer/
├── services/api/go.mod       # Independent module
├── services/ingester/go.mod  # Independent module
├── shared/go.mod             # Independent module (rarely used)
```

**Trade-offs**:
- ✅ Simple deployment (just `cd services/api && go build`)
- ✅ No Go workspace configuration needed
- ❌ Duplicate dependencies across services
- ❌ Must update dependencies per-service

**Alternative**: Go workspaces (go 1.18+) would unify dependencies but add complexity

---

## Anti-Patterns Avoided

### 1. ORM Usage

**Decision**: Use `database/sql` with `lib/pq` driver directly (no GORM/sqlx)

**Rationale**:
- ORMs hide partition awareness (critical for performance)
- Direct SQL makes partition pruning explicit (`WHERE chain_id = 1`)
- Batch inserts easier with raw SQL
- No "magic" - every query is visible and auditable

### 2. Premature Abstraction

**Decision**: Duplicate code first, extract when pattern emerges 3+ times

**Rationale**:
- Wrong abstraction is worse than no abstraction
- Blockchain indexing patterns still evolving (event parsing, reorg handling)
- Easier to refactor concrete code than generic interfaces

### 3. Microservices Over-Engineering

**Decision**: Two services (ingester, API) not three+ (no separate processor service)

**Rationale**:
- Current scale doesn't justify operational overhead of multiple services
- Direct database writes are simpler than message-driven architecture
- Can split out processor later if event parsing becomes complex

---

## Multi-Chain Support Strategy

### Chain Support Tiers

#### **Tier 1: Priority Chains** (Implement First)
High liquidity, large user base, well-established ecosystems

| Chain | Chain ID | Status | Block Time | Finality |
|-------|----------|--------|------------|----------|
| **Ethereum** | 1 | 🟢 Ready | ~12s | 64 blocks (~13 min) |
| **Polygon** | 137 | 🟢 Ready | ~2s | 256 blocks (~8.5 min) |
| **Arbitrum** | 42161 | 🟢 Ready | ~0.25s | ~15 min |
| **Optimism** | 10 | 🟢 Ready | ~2s | ~10 min |
| **Base** | 8453 | 🟢 Ready | ~2s | ~10 min |

**Target**: Launch with all Tier 1 chains

#### **Tier 2: High-Value Additions** (Next Phase)

| Chain | Chain ID | Status | Priority |
|-------|----------|--------|----------|
| **BSC** | 56 | 🟡 Planned | P2 |
| **Avalanche C-Chain** | 43114 | 🟡 Planned | P2 |
| **zkSync Era** | 324 | 🟡 Planned | P2 |
| **Polygon zkEVM** | 1101 | 🟡 Planned | P2 |

**Effort**: 1-2 days per chain (all EVM-compatible)

#### **Tier 3: Non-EVM Chains** (Future)
Requires significant additional work: Solana, Cosmos, Polkadot

### RPC Provider Strategy

- **Use multiple providers per chain** - Load balance, failover
- **Start with Alchemy free tier** - 300M compute units/month
- **Upgrade to paid when needed** - ~$50/month per chain

### Decision Criteria for New Chains

✅ **Add when**:
- TVL > $500M
- Daily Active Users > 10,000
- EVM-compatible
- Stable RPC infrastructure

❌ **Avoid**:
- Chains < 6 months old
- Poor RPC infrastructure
- Low activity (< 1000 txs/day)

---

## Future Considerations

### When to Add Message Queue (Kafka/RabbitMQ)

**Triggers**:
- Indexing >10 chains (need better scalability)
- Adding event processing pipeline (multiple consumers)
- Reorg replay requirements become complex
- Need async event-driven notifications

**Migration Path**: Keep direct writes, add queue as parallel path for event streaming

---

### When to Consider Rust

**Triggers**:
- MEV detection (CPU-intensive pattern matching)
- Zero-copy deserialization requirements
- Sub-millisecond latency API requirements
- Cryptographic operations (signature verification, proof checking)

**Migration Path**: Hybrid approach - keep Go services, add Rust for specific hot paths

---

### When to Shard Database

**Triggers**:
- Single PostgreSQL instance can't handle write load (>10K writes/sec)
- Database size exceeds storage limits (multi-TB)
- Query performance degrades despite indexing/partitioning

**Sharding Strategy**: Shard by `chain_id` - each chain gets dedicated database instance

---

## References

- [docs/TECHNICAL_SPEC.md](docs/TECHNICAL_SPEC.md) - Implementation details
- [PROGRESS_TRACKING.md](PROGRESS_TRACKING.md) - Current state and roadmap
- [docs/setup/](docs/setup/) - Setup guides
- [learning/System-Design-Architecture.md](learning/System-Design-Architecture.md) - Deep dives and interview prep
