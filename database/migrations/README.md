# Database Migrations

This directory contains database schema migrations managed by [golang-migrate](https://github.com/golang-migrate/migrate).

## Migration Files

Migrations are versioned using sequential numbers with `.up.sql` and `.down.sql` files:

```
000001_initial_schema.up.sql      # Creates initial tables
000001_initial_schema.down.sql    # Removes initial tables
000002_add_calldata_parsing.up.sql
000002_add_calldata_parsing.down.sql
000003_add_skipped_blocks.up.sql
000003_add_skipped_blocks.down.sql
```

## Usage

### Apply Migrations

```bash
# Apply all pending migrations
make migrate-up

# Check current version
make migrate-status

# Rollback last migration
make migrate-down
```

### Create New Migration

```bash
# Create new migration files
make migrate-create
# Enter name: add_user_table

# This creates:
# - database/migrations/000004_add_user_table.up.sql
# - database/migrations/000004_add_user_table.down.sql
```

### Recovery

If migrations get out of sync (e.g., tables exist but tracking is missing):

```bash
# Force set to specific version
make migrate-force
# Enter version: 3

# Then check status
make migrate-status
```

## Migration Best Practices

### 1. Always Create Both .up and .down Files

```sql
-- 000004_add_indexes.up.sql
CREATE INDEX idx_transactions_hash ON transactions(tx_hash);

-- 000004_add_indexes.down.sql
DROP INDEX IF EXISTS idx_transactions_hash;
```

### 2. Use Transactions

```sql
BEGIN;

-- Your migration here
CREATE TABLE ...;

COMMIT;
```

### 3. Make Migrations Idempotent

```sql
-- Use IF EXISTS / IF NOT EXISTS
CREATE TABLE IF NOT EXISTS users (...);
DROP TABLE IF EXISTS old_users;
```

### 4. Test Rollbacks

```bash
# Apply migration
make migrate-up

# Test rollback
make migrate-down

# Re-apply
make migrate-up
```

### 5. Never Modify Existing Migrations

Once a migration is applied in production, create a new migration instead of modifying the old one.

## Production Deployment

### Supabase

```bash
# Set environment variable
export DATABASE_URL="postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:5432/postgres"

# Apply migrations
make migrate-up
```

### Railway / Render

Migrations are automatically applied during deployment via:
- Railway: Build command in `railway.json`
- Render: Build command in `render.yaml`

### CI/CD

Migrations run automatically in GitHub Actions before deploying services:

```yaml
# .github/workflows/deploy.yml
- name: Run migrations
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
  run: make migrate-up
```

## Troubleshooting

### "Dirty database version"

This means a migration failed partway through. To fix:

```bash
# Check what version is marked dirty
make migrate-status

# Force version back
make migrate-force
# Enter: [previous-version]

# Fix the migration SQL
# Then re-run
make migrate-up
```

### "No change" When Running migrate-up

All migrations are already applied. This is normal!

```bash
# Verify current version
make migrate-status
```

### Connection Refused

Make sure Docker containers are running:

```bash
make docker-up
make status
```

## Migration History

| Version | Description | Date | Author |
|---------|-------------|------|--------|
| 000001 | Initial schema (chains, blocks, transactions, events) | Nov 14, 2025 | Initial |
| 000002 | Add calldata parsing tables | Nov 15, 2025 | Initial |
| 000003 | Add skipped blocks tracking | Nov 26, 2025 | Dev |

## Resources

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate/blob/master/database/postgres/TUTORIAL.md)
- [PostgreSQL Migration Best Practices](https://www.postgresql.org/docs/current/ddl-alter.html)
- [Supabase Schema Migrations](https://supabase.com/docs/guides/cli/managing-environments)
