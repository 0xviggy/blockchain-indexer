# Deployment & Production Infrastructure Guide

**Last Updated**: November 27, 2025  
**Project**: Multi-Chain Blockchain Indexer

---

## Table of Contents

1. [Development vs Production Architecture](#development-vs-production-architecture)
2. [Database: Supabase Setup](#database-supabase-setup)
3. [Migration Management: golang-migrate](#migration-management-golang-migrate)
4. [Service Hosting Options](#service-hosting-options)
5. [Deployment Pipeline](#deployment-pipeline)
6. [Environment Configuration](#environment-configuration)
7. [Cost Estimates](#cost-estimates)

---

## Development vs Production Architecture

### Development (Current)
```
Local Machine
├── Docker Compose
│   ├── PostgreSQL (localhost:5432)
│   ├── Redis (localhost:6379)
│   └── Kafka/Redpanda (localhost:19092)
├── Go Services (local processes)
│   ├── Ingester
│   └── API
└── React Frontend (Vite dev server)
```

### Production (Target)
```
Cloud Infrastructure
├── Database: Supabase PostgreSQL (managed, hosted)
├── Cache: Upstash Redis (managed, serverless)
├── Message Queue: Upstash Kafka (managed, serverless) [optional]
├── Backend Services: Railway.app / Render.com
│   ├── Ingester (Docker container)
│   └── API (Docker container)
├── Frontend: Vercel / Netlify (static hosting + edge functions)
└── Monitoring: Grafana Cloud (free tier) + Sentry
```

**Key Changes**:
- ✅ Replace local PostgreSQL with Supabase (shared across devs + production)
- ✅ Replace local Redis with Upstash (serverless, pay-per-use)
- ✅ Backend services containerized and deployed to Railway/Render
- ✅ Frontend deployed as static site to Vercel
- ✅ All config via environment variables

---

## Database: Supabase Setup

### Why Supabase?

- ✅ **Free Tier**: 500MB database, 2GB bandwidth/month
- ✅ **Instant Setup**: Database ready in 2 minutes
- ✅ **Production Ready**: Auto-backups, monitoring, connection pooling
- ✅ **Multi-Environment**: Easy to create dev/staging/prod databases
- ✅ **Team Collaboration**: Share same database across developers
- ✅ **Migration Tracking**: Built-in support for schema migrations

### Setup Steps

#### 1. Create Supabase Project

```bash
# Visit: https://supabase.com/dashboard
# Click "New Project"
# 
# Project Settings:
#   Name: blockchain-indexer-dev
#   Database Password: [generate strong password]
#   Region: us-east-1 (closest to your users)
#   Pricing: Free
```

#### 2. Get Connection String

```bash
# In Supabase Dashboard:
# Settings → Database → Connection String

# Direct Connection (for migrations):
postgresql://postgres:[YOUR-PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres

# Connection Pooling (for app):
postgresql://postgres:[YOUR-PASSWORD]@db.[PROJECT-REF].supabase.co:6543/postgres?pgbouncer=true
```

#### 3. Update Local Environment

```bash
# Create .env.local (git-ignored)
cat > .env.local << 'EOF'
# Supabase Database (shared dev environment)
DATABASE_URL=postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:6543/postgres?pgbouncer=true
DATABASE_URL_DIRECT=postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:5432/postgres

# For migrations (needs direct connection)
MIGRATION_DATABASE_URL=${DATABASE_URL_DIRECT}

# RPC Providers
ETH_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
POLYGON_RPC_URL=https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY

# Redis (Upstash - optional for now)
REDIS_URL=redis://default:[PASSWORD]@[HOST]:6379

# Environment
NODE_ENV=development
EOF

# Load environment
source .env.local
```

#### 4. Run Migrations

```bash
# Using golang-migrate (see next section)
make migrate-up
```

### Multiple Environments

```bash
# Development Database
supabase.com/dashboard → blockchain-indexer-dev

# Staging Database (optional)
supabase.com/dashboard → blockchain-indexer-staging

# Production Database
supabase.com/dashboard → blockchain-indexer-prod
```

**Environment Variables**:
```bash
# .env.development
DATABASE_URL=postgresql://postgres:...@db.dev-project.supabase.co:6543/postgres

# .env.staging
DATABASE_URL=postgresql://postgres:...@db.staging-project.supabase.co:6543/postgres

# .env.production
DATABASE_URL=postgresql://postgres:...@db.prod-project.supabase.co:6543/postgres
```

---

## Migration Management: golang-migrate

### Why golang-migrate?

- ✅ **Industry Standard**: Used by Kubernetes, Docker, HashiCorp
- ✅ **Idempotent**: Tracks which migrations have run
- ✅ **Rollback Support**: Can revert migrations safely
- ✅ **Works Everywhere**: Local, Docker, CI/CD, Supabase
- ✅ **No Dependencies**: Single binary, cross-platform

### Installation

```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Verify
migrate -version
```

### Migration File Structure

```bash
database/migrations/
├── 000001_initial_schema.up.sql          # Creates tables
├── 000001_initial_schema.down.sql        # Drops tables
├── 000002_add_calldata_parsing.up.sql
├── 000002_add_calldata_parsing.down.sql
├── 000003_add_skipped_blocks.up.sql
├── 000003_add_skipped_blocks.down.sql
└── README.md
```

**Naming Convention**: `{version}_{description}.{up|down}.sql`

### Convert Existing Migrations

```bash
# Current files:
# - 001_initial_schema.sql
# - 002_add_calldata_parsing.sql
# - 003_add_skipped_blocks.sql

# Split into .up.sql and .down.sql
cd database/migrations/

# Rename to golang-migrate format
mv 001_initial_schema.sql 000001_initial_schema.up.sql

# Create down migration (rollback)
cat > 000001_initial_schema.down.sql << 'SQL'
-- Rollback initial schema
DROP VIEW IF EXISTS indexing_status;
DROP VIEW IF EXISTS latest_blocks;
DROP TABLE IF EXISTS skipped_blocks;
DROP TABLE IF EXISTS checkpoints;
DROP TABLE IF EXISTS events_eth CASCADE;
DROP TABLE IF EXISTS events_polygon CASCADE;
DROP TABLE IF EXISTS events_arbitrum CASCADE;
DROP TABLE IF EXISTS events_optimism CASCADE;
DROP TABLE IF EXISTS events_base CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS transactions_eth CASCADE;
DROP TABLE IF EXISTS transactions_polygon CASCADE;
DROP TABLE IF EXISTS transactions_arbitrum CASCADE;
DROP TABLE IF EXISTS transactions_optimism CASCADE;
DROP TABLE IF EXISTS transactions_base CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS blocks_eth CASCADE;
DROP TABLE IF EXISTS blocks_polygon CASCADE;
DROP TABLE IF EXISTS blocks_arbitrum CASCADE;
DROP TABLE IF EXISTS blocks_optimism CASCADE;
DROP TABLE IF EXISTS blocks_base CASCADE;
DROP TABLE IF EXISTS blocks CASCADE;
DROP TABLE IF EXISTS protocol_signatures;
DROP TABLE IF EXISTS chains;
SQL
```

### Usage

```bash
# Apply all pending migrations
migrate -path database/migrations \
  -database "$DATABASE_URL_DIRECT" \
  up

# Check migration status
migrate -path database/migrations \
  -database "$DATABASE_URL_DIRECT" \
  version

# Rollback last migration
migrate -path database/migrations \
  -database "$DATABASE_URL_DIRECT" \
  down 1

# Rollback to specific version
migrate -path database/migrations \
  -database "$DATABASE_URL_DIRECT" \
  goto 2

# Force set version (if migrations table is corrupted)
migrate -path database/migrations \
  -database "$DATABASE_URL_DIRECT" \
  force 3
```

### Makefile Integration

```makefile
# Add to Makefile
MIGRATION_PATH=database/migrations
DATABASE_URL_DIRECT ?= postgresql://indexer:password@localhost:5432/indexer?sslmode=disable

migrate-up:
	@echo "📈 Applying migrations..."
	@migrate -path $(MIGRATION_PATH) -database "$(DATABASE_URL_DIRECT)" up
	@echo "✅ Migrations complete"

migrate-down:
	@echo "📉 Rolling back last migration..."
	@migrate -path $(MIGRATION_PATH) -database "$(DATABASE_URL_DIRECT)" down 1
	@echo "✅ Rollback complete"

migrate-status:
	@echo "📊 Migration status:"
	@migrate -path $(MIGRATION_PATH) -database "$(DATABASE_URL_DIRECT)" version

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir $(MIGRATION_PATH) -seq $$name
	@echo "✅ Created migration files"
```

---

## Service Hosting Options

### Backend Services (Ingester + API)

#### Option 1: Railway.app ⭐ **Recommended**

**Pros**:
- ✅ Easiest deployment (git push to deploy)
- ✅ Free $5/month credit (good for testing)
- ✅ Auto-scaling, zero-downtime deploys
- ✅ Built-in monitoring and logs
- ✅ Environment variables per service

**Pricing**: ~$10-20/month per service

**Setup**:
```bash
# 1. Install Railway CLI
npm install -g @railway/cli

# 2. Login and link project
railway login
railway init

# 3. Create services
railway add  # Select "Empty Service" → name it "ingester"
railway add  # Select "Empty Service" → name it "api"

# 4. Set environment variables
railway variables set DATABASE_URL="postgresql://..."
railway variables set ETH_RPC_URL="https://..."

# 5. Deploy
railway up
```

#### Option 2: Render.com

**Pros**:
- ✅ Free tier available (good for API)
- ✅ Docker support
- ✅ Auto-deploy from GitHub

**Pricing**: Free tier + $7-25/month for paid tiers

#### Option 3: Fly.io

**Pros**:
- ✅ Global edge deployment
- ✅ Free tier (3 shared-cpu VMs)
- ✅ Excellent for low-latency worldwide

**Pricing**: Free tier, then pay-as-you-go

#### Option 4: AWS ECS / GCP Cloud Run

**Pros**:
- ✅ Production-grade, infinite scale
- ✅ Pay only for what you use

**Cons**:
- ❌ More complex setup
- ❌ Requires DevOps knowledge

---

### Frontend Hosting

#### Option 1: Vercel ⭐ **Recommended**

**Pros**:
- ✅ Built for React/Vite
- ✅ Auto-deploy from GitHub
- ✅ Global CDN, instant cache invalidation
- ✅ Edge functions for API routes
- ✅ Free SSL, custom domains

**Pricing**: Free for personal projects, $20/month for teams

**Setup**:
```bash
# 1. Install Vercel CLI
npm install -g vercel

# 2. Deploy
cd web
vercel

# 3. Set environment variables
vercel env add VITE_API_URL production
# Enter: https://api.your-domain.com

# 4. Deploy to production
vercel --prod
```

#### Option 2: Netlify

Similar to Vercel, slightly different feature set

---

### Redis Cache

#### Option 1: Upstash Redis ⭐ **Recommended**

**Pros**:
- ✅ Serverless (pay per request)
- ✅ Free tier: 10k commands/day
- ✅ Global replication
- ✅ Works with Railway/Vercel edge functions

**Pricing**: Free tier, then $0.20 per 100k commands

**Setup**:
```bash
# 1. Create database at upstash.com
# 2. Copy REDIS_URL
# 3. Add to Railway:
railway variables set REDIS_URL="redis://..."
```

---

### Message Queue (Optional, for >10 chains)

#### Option 1: Upstash Kafka

**Pros**:
- ✅ Serverless Kafka
- ✅ Compatible with Redpanda

**Pricing**: Free tier, then pay-per-use

#### Option 2: Confluent Cloud

**Pros**:
- ✅ Managed Kafka by creators of Kafka
- ✅ Production-ready

**Pricing**: Starts at $1/hour

---

## Deployment Pipeline

### CI/CD with GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      
      - name: Run tests
        run: |
          cd services/ingester && go test ./...
          cd ../api && go test ./...
      
      - name: Run migrations (staging)
        env:
          DATABASE_URL: ${{ secrets.STAGING_DATABASE_URL }}
        run: |
          migrate -path database/migrations -database "$DATABASE_URL" up

  deploy-backend:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to Railway
        run: |
          railway up --service ingester
          railway up --service api

  deploy-frontend:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to Vercel
        run: |
          cd web
          vercel --prod --token=${{ secrets.VERCEL_TOKEN }}
```

---

## Environment Configuration

### Environment Files Structure

```bash
.env.example          # Template (committed to git)
.env.local            # Local development (git-ignored)
.env.development      # Dev database (git-ignored)
.env.staging          # Staging environment (git-ignored)
.env.production       # Production (git-ignored, set in Railway/Vercel)
```

### .env.example

```bash
# Database
DATABASE_URL=postgresql://user:password@host:port/database
DATABASE_URL_DIRECT=postgresql://user:password@host:port/database

# RPC Providers
ETH_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
POLYGON_RPC_URL=https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY
ARBITRUM_RPC_URL=https://arb-mainnet.g.alchemy.com/v2/YOUR_KEY
OPTIMISM_RPC_URL=https://opt-mainnet.g.alchemy.com/v2/YOUR_KEY
BASE_RPC_URL=https://base-mainnet.g.alchemy.com/v2/YOUR_KEY

# Redis
REDIS_URL=redis://default:password@host:6379

# API
API_PORT=8000
CORS_ORIGINS=http://localhost:5173,https://yourapp.com

# Monitoring
SENTRY_DSN=https://...@sentry.io/...
GRAFANA_API_KEY=...

# Environment
NODE_ENV=development
```

---

## Cost Estimates

### Development (Shared Database)

| Service | Provider | Tier | Cost |
|---------|----------|------|------|
| Database | Supabase | Free | $0 |
| Redis | Upstash | Free | $0 |
| Hosting | Local | N/A | $0 |
| **Total** | | | **$0/month** |

### Staging

| Service | Provider | Tier | Cost |
|---------|----------|------|------|
| Database | Supabase | Free | $0 |
| Backend | Railway | Hobby | $10 |
| Frontend | Vercel | Free | $0 |
| Redis | Upstash | Free | $0 |
| **Total** | | | **$10/month** |

### Production (Low Traffic)

| Service | Provider | Tier | Cost |
|---------|----------|------|------|
| Database | Supabase | Pro | $25 |
| Backend (2 services) | Railway | Starter | $20 |
| Frontend | Vercel | Pro | $20 |
| Redis | Upstash | Pay-as-you-go | $5 |
| Monitoring | Grafana Cloud | Free | $0 |
| RPC Providers | Alchemy | Growth | $49 |
| **Total** | | | **$119/month** |

### Production (High Traffic)

| Service | Provider | Tier | Cost |
|---------|----------|------|------|
| Database | Supabase | Pro | $25-100 |
| Backend | Railway | Pro | $50-200 |
| Frontend | Vercel | Pro | $20 |
| Redis | Upstash | Pay-as-you-go | $20-50 |
| Monitoring | Grafana Cloud | Pro | $50 |
| RPC Providers | Alchemy | Scale | $199-499 |
| CDN | Cloudflare | Pro | $20 |
| **Total** | | | **$384-1,019/month** |

---

## Next Steps

### Immediate (This Week)

1. ✅ Install golang-migrate
2. ✅ Create Supabase dev database
3. ✅ Update Makefile with new migration commands
4. ✅ Test migrations on Supabase
5. ✅ Update team documentation

### Short Term (Next 2 Weeks)

1. Create staging environment
2. Set up Railway for backend services
3. Deploy frontend to Vercel
4. Add CI/CD pipeline
5. Set up monitoring (Grafana Cloud)

### Long Term (Future)

1. Add database connection pooling (PgBouncer)
2. Implement blue-green deployments
3. Add load testing (k6)
4. Set up disaster recovery procedures
5. Document incident response playbook

---

**Updated By**: GitHub Copilot  
**Review Status**: Ready for implementation
