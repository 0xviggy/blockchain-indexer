# Database Setup & Management

Complete guide for database setup, migrations, seed data, and team workflows.

## Quick Start

```bash
# First time setup
cp .env.example .env
make setup              # Starts infrastructure + applies migrations
make db-seed            # (Optional) Load sample data

# Verify setup
make migrate-status     # Check migration version
make db-shell           # Open PostgreSQL CLI
```

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Initial Setup](#initial-setup)
3. [Migration System](#migration-system)
4. [Seed Data](#seed-data)
5. [Daily Workflow](#daily-workflow)
6. [Database State](#database-state)
7. [Troubleshooting](#troubleshooting)

---

## Prerequisites

- Docker Desktop installed and running
- `golang-migrate` installed:
  ```bash
  # macOS
  brew install golang-migrate
  
  # Linux
  curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
  sudo mv migrate /usr/local/bin/
  ```

---

## Initial Setup

### 1. Configure Environment

```bash
# Copy template
cp .env.example .env

# Edit if needed (defaults work for local development)
vim .env
```

Default database connection:
```
DATABASE_URL=postgres://indexer:password@localhost:5432/indexer?sslmode=disable
```

### 2. Start Infrastructure

```bash
make setup
```

This command:
- ✅ Starts Docker containers (PostgreSQL, Redis, Kafka)
- ✅ Waits for services to be healthy
- ✅ Applies all database migrations
- ✅ Verifies setup

### 3. Load Sample Data (Optional)

```bash
make db-seed
```

Loads test data:
- 8 sample blocks (Ethereum + Polygon)
- 9 sample transactions
- 3 sample event logs
- All using block numbers < 1000 (won't conflict with real data)

---

## Migration System

### Current Migration Version

**Version**: 3  
**Migrations**:
1. `000001_initial_schema` - Base tables (chains, blocks, transactions, logs)
2. `000002_add_calldata_parsing` - Calldata parsing fields
3. `000003_add_skipped_blocks` - Error tracking table

### Migration Commands

```bash
# Check current version
make migrate-status

# Apply pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create
# Enter name: add_my_feature

# Force version (recovery only)
make migrate-force
# Enter version: 3
```

### Creating a Migration

```bash
# 1. Create files
make migrate-create
# Name: add_analytics_table

# 2. Edit .up.sql (apply changes)
vim database/migrations/000004_add_analytics_table.up.sql
```

```sql
-- Example: database/migrations/000004_add_analytics_table.up.sql
CREATE TABLE IF NOT EXISTS analytics (
    id SERIAL PRIMARY KEY,
    metric_name TEXT NOT NULL,
    value NUMERIC,
    timestamp TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analytics_timestamp 
ON analytics(timestamp);
```

```bash
# 3. Edit .down.sql (rollback changes)
vim database/migrations/000004_add_analytics_table.down.sql
```

```sql
-- Example: database/migrations/000004_add_analytics_table.down.sql
DROP INDEX IF EXISTS idx_analytics_timestamp;
DROP TABLE IF EXISTS analytics;
```

```bash
# 4. Test migration
make migrate-up     # Apply
make migrate-down   # Rollback
make migrate-up     # Re-apply

# 5. Verify in database
make db-shell
# Then: \d analytics
```

### Migration Best Practices

- ✅ **Never edit merged migrations** - Create a new migration instead
- ✅ **Always create both .up and .down** - Must be reversible
- ✅ **Test rollback** - Ensure `make migrate-down` works
- ✅ **Use IF NOT EXISTS** - Makes migrations idempotent
- ✅ **Communicate with team** - Announce new migrations in chat

---

## Seed Data

### Purpose

Seed data provides:
- Consistent test data across all developers
- Functional UI without running ingester
- Faster development iterations
- Predictable testing scenarios

### Seed Data Location

```
database/seeds/
└── 001_sample_blocks.sql    # Blocks, transactions, logs
```

### Using Seed Data

```bash
# Load all seed data
make db-seed

# Clear seed data (keeps real ingester data)
make db-clear-seeds

# Verify seed data loaded
make db-shell
# Then: SELECT COUNT(*) FROM blocks WHERE block_number < 1000;
```

### Seed Data Contents

**8 Blocks:**
- 5 Ethereum blocks (chain_id=1)
- 3 Polygon blocks (chain_id=137)
- Block numbers: 100-104

**9 Transactions:**
- ETH transfers
- ERC20 token transfers (USDC, USDT)
- Uniswap swap
- NFT mint
- Failed transaction example

**3 Event Logs:**
- ERC20 Transfer events
- Uniswap Swap event

All seed data uses `block_number < 1000` to avoid conflicts with real blockchain data.

---

## Daily Workflow

### Morning Routine

```bash
# 1. Pull latest code
git pull

# 2. Start infrastructure (if stopped)
make docker-up

# 3. Apply any new migrations
make migrate-up

# 4. Verify you're in sync with team
make migrate-status
```

### Working on Features

```bash
# Check service status
make status

# Open database shell
make db-shell
# Commands: \dt (list tables), \d table_name (describe table)

# View logs
make logs
```

### Ending Your Day

```bash
# Optional: Stop infrastructure to free resources
make docker-down

# Data persists in Docker volumes
# Next `make docker-up` will restore your data
```

---

## Database State

### Schema Overview

| Table | Purpose | Partition | Primary Key |
|-------|---------|-----------|-------------|
| chains | Chain metadata | - | chain_id |
| blocks | Block data | chain_id | (chain_id, block_number) |
| transactions | Transaction data | chain_id | tx_hash |
| logs | Event logs | chain_id | (tx_hash, log_index) |
| blob_transactions | EIP-4844 blobs | chain_id | tx_hash |
| skipped_blocks | Failed blocks | - | (chain_id, block_number) |
| schema_migrations | Migration tracking | - | version |

### Checking Database State

```bash
# Current migration version
make migrate-status

# Open PostgreSQL shell
make db-shell

# Inside psql:
\dt                              # List all tables
\d blocks                        # Describe blocks table
SELECT * FROM schema_migrations; # Check migration version
SELECT COUNT(*) FROM blocks;     # Count indexed blocks
\q                              # Quit
```

### Environment States

**Development (Local):**
- Connection: `localhost:5432`
- State: Migrations + Optional seeds + Real data
- Volume: Docker volume `indexer_postgres_data`

**Production (Supabase):**
- Connection: Supabase connection string
- State: Migrations + Live blockchain data
- Backups: Automatic daily backups via Supabase

---

## Troubleshooting

### Migration Version is Dirty

**Symptom:** `make migrate-status` shows `dirty: true`

**Solution:**
```bash
# Check what happened
make migrate-status

# Force to last good version
make migrate-force
# Enter: 2 (if migration 3 failed)

# Fix the problematic migration
vim database/migrations/000003_*.sql

# Try again
make migrate-up
```

### Tables Exist But No Migration Record

**Symptom:** Tables exist but `schema_migrations` is empty

**Solution:**
```bash
# Force set to current version
make migrate-force
# Enter: 3 (if you have 3 migrations)

# Verify
make migrate-status
```

### Different Schema Than Team

**Solution:** Reset to same baseline

```bash
# Communicate with team first!
make db-reset       # ⚠️ Deletes ALL data
make migrate-up     # Apply all migrations
make migrate-status # Verify version matches team
make db-seed        # Optional: reload test data
```

### Docker Database Resets on Restart

**Problem:** Data disappears when Docker restarts

**Solution:**
```bash
# Check volumes exist
docker volume ls | grep indexer

# Should see: indexer_postgres_data

# If missing, recreate properly:
make docker-down
make docker-up
make migrate-up
```

### Port 5432 Already in Use

**Problem:** Another PostgreSQL is running

**Solution:**
```bash
# Check what's using the port
lsof -i :5432

# Stop local PostgreSQL
brew services stop postgresql
# OR
sudo systemctl stop postgresql

# Then restart Docker
make docker-up
```

### Migration Files Out of Order

**Problem:** Two developers created same migration number

**Solution:**
```bash
# Renumber your migration
cd database/migrations
mv 000005_your_feature.up.sql 000006_your_feature.up.sql
mv 000005_your_feature.down.sql 000006_your_feature.down.sql

# Commit
git add database/migrations
git commit -m "Renumber migration to avoid conflict"
```

---

## Team Coordination

### When Creating a Migration

**Communicate:**
```
📢 New migration: 000004_add_analytics
Branch: feature/analytics
Remember to run `make migrate-up` after pulling!
```

### When Migration Fails

**Communicate:**
```
⚠️ Migration 000005 has issues
Do NOT merge yet
If you pulled: run `make migrate-down` to rollback
```

### Team Reset Required

**Communicate:**
```
🔄 Database reset needed
New baseline: migration version 5
Please run:
  1. git pull
  2. make db-reset
  3. make migrate-up
  4. Verify: make migrate-status shows "5"
```

---

## Command Reference

```bash
# Infrastructure
make setup           # Complete first-time setup
make docker-up       # Start containers
make docker-down     # Stop containers
make status          # Check all services
make logs            # View container logs

# Database
make migrate-up      # Apply pending migrations
make migrate-down    # Rollback last migration
make migrate-status  # Check current version
make migrate-create  # Create new migration
make migrate-force   # Force version (recovery)
make db-shell        # PostgreSQL CLI
make db-reset        # ⚠️ Delete all data + re-migrate

# Seed Data
make db-seed         # Load sample data
make db-clear-seeds  # Remove sample data

# Development
make run-api         # Start API server
make run-ingester    # Start blockchain ingester
```

---

## Additional Resources

- [Deployment Guide](../docs/DEPLOYMENT.md) - Production setup with Supabase
- [Development Status](../docs/DEVELOPMENT_STATUS.md) - Current progress
- [golang-migrate docs](https://github.com/golang-migrate/migrate) - Migration tool documentation
- [PostgreSQL docs](https://www.postgresql.org/docs/) - Database documentation
