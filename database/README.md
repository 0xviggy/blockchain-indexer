# Database Directory

Database schema, migrations, and seed data for the Blockchain Indexer.

---

## Quick Reference

```bash
# See complete database guide
cat ../DATABASE_GUIDE.md
```

---

## Directory Structure

```
database/
├── README.md              # This file
├── migrations/            # Schema version control
│   ├── 000001_initial_schema.up.sql
│   ├── 000001_initial_schema.down.sql
│   ├── 000002_add_calldata_parsing.up.sql
│   ├── 000002_add_calldata_parsing.down.sql
│   └── 000003_add_skipped_blocks.up.sql
└── seeds/                 # Sample data for development
    └── 001_sample_blocks.sql
```

---

## Migrations

**Current Version**: 3

Migrations use [golang-migrate](https://github.com/golang-migrate/migrate) for version-controlled schema changes.

### Quick Commands

```bash
make migrate-up          # Apply pending migrations
make migrate-status      # Check current version
make migrate-create      # Create new migration
```

### Migration Files

| Version | Name | Description |
|---------|------|-------------|
| 1 | initial_schema | Base tables (chains, blocks, transactions, logs) |
| 2 | add_calldata_parsing | Calldata parsing fields |
| 3 | add_skipped_blocks | Error tracking table |

**Full documentation**: [DATABASE_GUIDE.md](../DATABASE_GUIDE.md#migration-system)

---

## Seed Data

Sample data for development and testing.

### Quick Commands

```bash
make db-seed            # Load sample data
make db-clear-seeds     # Remove sample data
```

### Available Seeds

- **001_sample_blocks.sql** - 8 blocks, 9 transactions, 3 event logs
  - Uses block numbers < 1000 to avoid conflicts with real data
  - Includes Ethereum and Polygon examples

**Full documentation**: [DATABASE_GUIDE.md](../DATABASE_GUIDE.md#seed-data)

---

## Database Schema

### Main Tables

| Table | Purpose | Partition Key |
|-------|---------|---------------|
| chains | Chain metadata | - |
| blocks | Block data | chain_id |
| transactions | Transaction data | chain_id |
| logs | Event logs | chain_id |
| blob_transactions | EIP-4844 blobs | chain_id |
| skipped_blocks | Failed indexing attempts | - |

### Accessing Database

```bash
# PostgreSQL CLI
make db-shell

# Inside psql:
\dt                    # List tables
\d blocks              # Describe table
\q                     # Quit
```

---

## Complete Documentation

For comprehensive database management guide including:
- First-time setup
- Creating migrations
- Seed data management
- Team workflows
- Troubleshooting

See: **[DATABASE_GUIDE.md](../DATABASE_GUIDE.md)**
