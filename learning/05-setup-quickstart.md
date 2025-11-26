# Setup & Quick Start Guide

Complete guide to getting the blockchain indexer up and running on your local machine.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Installation Steps](#installation-steps)
- [System Architecture Overview](#system-architecture-overview)
- [Starting the System](#starting-the-system)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Install Requirements (macOS)

```bash
# Install Docker Desktop from https://docker.com or:
brew install --cask docker
# Open Docker Desktop, wait for whale icon in menu bar

# Install Go (if needed)
brew install go

# Verify
docker --version  # Should be 20.10+
go version        # Should be 1.21+
```

### Understanding Go Installation

**Global Installation**:
- Go binary: `/opt/homebrew/bin/go` (installed globally via Homebrew)
- Go toolchain: Available system-wide for all projects
- Module cache: `~/go/pkg/mod` (shared global cache for all downloaded modules)

**Per-Project Dependencies**:
- Each service has its own `go.mod` file defining dependencies and versions
- When you run `go mod download`, modules are cached globally but tracked locally
- This is the standard Go approach: global toolchain + global cache, but project-specific dependency tracking

**Why this matters**:
- You install Go once for the entire system
- Module downloads are cached and reused across projects (saves bandwidth/time)
- Each project independently controls its dependency versions via `go.mod`
- No need for per-project Go installations (unlike Python virtual environments)

---

## System Architecture Overview

```
Blockchain → Ingester → Kafka → Processor → PostgreSQL → API → Users
                                              ↓
                                           Redis Cache
```

**Why event-driven?** 
- Decouples services, enables independent scaling
- Provides replay capability for reorgs
- Allows multiple consumers of blockchain data

**Full details**: See [Technical Spec](../docs/TECHNICAL_SPEC.md)

---

## Installation Steps

### Phase 1: Infrastructure Setup (COMPLETED ✅)

**Date**: 2025-11-14  
**What we built**: Docker Compose setup with PostgreSQL, Redis, and Kafka

#### Files Created:
1. **`infrastructure/docker/docker-compose.yml`**
   - PostgreSQL 15 (port 5432)
   - Redis 7 (port 6379)
   - Redpanda/Kafka (ports 19092, 18081, 18082)
   - Kafka UI (port 8080) - for debugging
   - pgAdmin (port 5050) - for DB management

2. **`infrastructure/docker/init-db.sh`**
   - PostgreSQL initialization script
   - Enables UUID and pg_stat_statements extensions
   - Runs automatically on first container start

3. **`database/migrations/001_initial_schema.sql`**
   - Complete schema with partitioning
   - 5 tables: chains, blocks, transactions, events, checkpoints
   - Partitioned by chain_id for multi-chain support
   - Indexes for performance
   - Triggers for auto-updating timestamps
   - Views for monitoring (latest_blocks, indexing_status)

4. **`.env.example`**
   - Template for environment variables
   - RPC URLs for all chains (Ethereum, Polygon, Arbitrum, Optimism, Base)
   - Database and Redis configuration

5. **Updated `Makefile`**
   - `make setup` - One command to start everything
   - `make docker-up` - Start infrastructure
   - `make migrate` - Run database migrations
   - `make db-shell` - Open PostgreSQL CLI
   - `make status` - Check service health
   - Many more utility commands

---

## Starting the System

### Quick Start Commands

```bash
# 1. Start all infrastructure (PostgreSQL, Redis, Kafka)
make docker-up

# 2. Wait for services to be ready (check logs)
make status

# 3. Run database migrations
make migrate

# 4. Verify tables were created
make db-shell
# Then in psql: \dt

# 5. View indexing status
# In psql:
SELECT * FROM indexing_status;
SELECT * FROM chains;
```

**Troubleshooting**: 
- Docker not running? Open Docker Desktop
- Port conflict? Stop services on 5432/6379/19092

---

## Verification

### Check Infrastructure Services

```bash
# Check if containers are running
make status

# View logs from all containers
make logs

# View specific container logs
docker logs indexer-postgres
docker logs indexer-kafka

# Restart everything
make docker-down && make docker-up

# Reset database (deletes all data!)
make db-reset
```

### Check Database Connection

```bash
# Open PostgreSQL CLI
make db-shell

# Check Redis connection
make redis-cli
# In redis-cli: PING (should return PONG)

# Check Kafka topics
make kafka-topics
```

### Web UIs for Debugging

- **Kafka UI**: http://localhost:8080 - View topics, messages, consumer groups
- **pgAdmin**: http://localhost:5050 - Database GUI (login: admin@indexer.local / admin)

---

## Key Learning Points

### Why Redpanda instead of Apache Kafka?

- Redpanda is Kafka-compatible but much simpler to run locally
- No need for Zookeeper
- Single binary, lower resource usage
- Same Kafka API, so we can switch to real Kafka in production

### Why Partitioned Tables?

- Each chain gets its own partition (blocks_eth, blocks_polygon, etc.)
- Queries filtered by chain_id only scan relevant partitions
- Easier to manage and archive old data per chain
- Better query performance (partition pruning)

### Database Schema Highlights

```sql
-- Multi-chain support with chain metadata
chains (chain_id, chain_name, rpc_url, enabled, last_indexed_block)

-- Blocks partitioned by chain
blocks (chain_id, block_number, block_hash, parent_hash, timestamp, ...)
  ├── blocks_eth (chain_id=1)
  ├── blocks_polygon (chain_id=137)
  └── ... other chains

-- Transactions partitioned by chain
transactions (chain_id, tx_hash, block_number, from_address, to_address, ...)

-- Events/Logs partitioned by chain
events (chain_id, tx_hash, contract_address, event_signature, decoded_data)

-- Checkpoint tracking for each service per chain
checkpoints (service_name, chain_id, last_processed_block)

-- Reorg tracking
reorg_events (chain_id, rollback_from_block, rollback_to_block, handled)
```

### Indexes Strategy

- Hash lookups: `idx_blocks_eth_hash` for fast block lookup by hash
- Time-series queries: `idx_blocks_eth_timestamp DESC` for recent blocks
- Address queries: `idx_tx_eth_from`, `idx_tx_eth_to` for wallet activity
- Event queries: `idx_events_eth_contract`, `idx_events_eth_signature`
- JSONB queries: GIN index on `decoded_data` for flexible event queries

---

## Phase 1 Summary - What We Accomplished

- ✅ **6 files created** (~700 lines of code)
- ✅ **5 Docker services** orchestrated
- ✅ **10 database tables** with 35 partitions
- ✅ **2 analytics views** for transaction and protocol insights
- ✅ **20+ indexes** for performance
- ✅ **20+ Makefile commands** for development
- ✅ **Multi-chain support** for 5 blockchains
- ✅ **Complete schema** with reorg handling
- ✅ **Web UIs** for debugging (Kafka UI, pgAdmin)

**Time to setup**: 2 minutes (after Docker installed)  
**Next**: Build Ingester service to start fetching blocks

---

## Troubleshooting Common Issues

### Issue: Port Already in Use

```bash
# Find what's using port 5432
lsof -i :5432
# Kill the process
kill -9 <PID>
```

### Issue: Docker Containers Won't Start

```bash
# Check Docker daemon is running
docker ps

# View container logs
docker logs indexer-postgres

# Restart Docker Desktop
# or: killall Docker && open -a Docker
```

### Issue: Database Connection Fails

```bash
# Wait for PostgreSQL to fully start (takes ~10 seconds)
docker logs indexer-postgres | grep "ready to accept connections"

# Test connection manually
psql -h localhost -U indexer -d indexer
# Password: password
```

---

## Next Steps

After completing setup:

1. **Explore the database schema**: See [03-databases-messaging.md](./03-databases-messaging.md)
2. **Learn about message parsing**: See [06-implementation-concepts.md](./06-implementation-concepts.md)
3. **Understand Go programming patterns**: See [04-go-programming.md](./04-go-programming.md)
4. **Review Docker and Kubernetes**: See [02-docker-kubernetes.md](./02-docker-kubernetes.md)

---

## Related Documentation

- [Technology Stack Overview](./01-technology-stack.md)
- [Implementation Concepts](./06-implementation-concepts.md)
- [Technical Specification](../docs/TECHNICAL_SPEC.md)
- [Development Status](../docs/DEVELOPMENT_STATUS.md)
