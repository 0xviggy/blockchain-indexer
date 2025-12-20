# Deployment & Production

> ⚠️ **EDUCATIONAL MATERIAL** - Interview prep & learning reference  
> For project-specific deployment, see [../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)

> **Purpose**: Comprehensive guide to deploying blockchain indexer to production - database hosting, service deployment, CI/CD, security, monitoring, and cost optimization with integrated interview Q&A.

---

## Table of Contents
- [Development vs Production Architecture](#development-vs-production-architecture)
- [Database Hosting](#database-hosting)
- [Service Hosting](#service-hosting)
- [Frontend Deployment](#frontend-deployment)
- [CI/CD Pipeline](#cicd-pipeline)
- [Security Best Practices](#security-best-practices)
- [Monitoring & Logging](#monitoring--logging)
- [Cost Optimization](#cost-optimization)
- [Interview Questions](#interview-questions)

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
└── React Frontend (Vite dev server :5173)
```

### Production (Target)

```
Cloud Infrastructure
├── Database: Supabase PostgreSQL (managed, hosted)
├── Cache: Upstash Redis (managed, serverless)
├── Message Queue: Upstash Kafka (managed, serverless) [optional]
├── Backend Services: Railway / Render / Fly.io
│   ├── Ingester (Docker container, auto-scale)
│   └── API (Docker container, auto-scale)
├── Frontend: Vercel / Netlify (CDN + edge functions)
└── Monitoring: Grafana Cloud + Sentry
```

**Key Changes:**
- ✅ Local PostgreSQL → Supabase (managed, backups, replication)
- ✅ Local Redis → Upstash (serverless, global)
- ✅ Local processes → Containerized services (Railway/Render)
- ✅ Vite dev server → Static CDN deployment (Vercel)
- ✅ All config via environment variables (12-factor app)

---

## Database Hosting

### Option 1: Supabase ⭐ Recommended for MVP

**Why Supabase?**
- ✅ **Free Tier**: 500MB database, 2GB bandwidth/month
- ✅ **Instant Setup**: Database ready in 2 minutes
- ✅ **Production Ready**: Auto-backups, monitoring, connection pooling
- ✅ **Multi-Environment**: Easy dev/staging/prod separation
- ✅ **Team Collaboration**: Share database across developers
- ✅ **PostgREST API**: Optional REST API auto-generated from schema

**Setup:**
```bash
# 1. Create project at https://supabase.com/dashboard
#    Name: blockchain-indexer-dev
#    Region: us-east-1 (closest to users)
#    Password: [generate strong password]

# 2. Get connection strings
# Direct Connection (for migrations):
postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:5432/postgres

# Connection Pooling (for app):
postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:6543/postgres?pgbouncer=true
```

**Environment Configuration:**
```bash
# .env.production
DATABASE_URL=postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:6543/postgres?pgbouncer=true
DATABASE_URL_DIRECT=postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:5432/postgres
```

**Best Practices:**
- Use pooled connection for application (port 6543)
- Use direct connection for migrations (port 5432)
- Enable Row Level Security (RLS) for API access
- Set up read replicas for high traffic

**Pricing:**
- Free: 500MB, 2GB bandwidth
- Pro ($25/month): 8GB, 50GB bandwidth
- Team ($599/month): 50GB, 250GB bandwidth

### Option 2: Railway PostgreSQL

**Pros:**
- ✅ Bundled with backend hosting
- ✅ Easy setup (one click)
- ✅ $5 free credit/month

**Cons:**
- ❌ No free tier beyond $5 credit
- ❌ Less database-focused features than Supabase

**Pricing:** ~$15-30/month

### Option 3: AWS RDS / GCP Cloud SQL

**Pros:**
- ✅ Production-grade, infinite scale
- ✅ Advanced features (read replicas, multi-AZ)
- ✅ Enterprise support

**Cons:**
- ❌ Expensive ($50-500/month minimum)
- ❌ Complex setup
- ❌ Requires DevOps expertise

**Use when:** Scaling beyond 100K+ requests/day

### Migration Management

**Using golang-migrate:**
```bash
# Install
brew install golang-migrate  # macOS
# or download from: https://github.com/golang-migrate/migrate/releases

# Create migration
migrate create -ext sql -dir database/migrations -seq add_user_table

# Apply migrations
migrate -path database/migrations \
  -database "$DATABASE_URL_DIRECT" \
  up

# Rollback
migrate -path database/migrations \
  -database "$DATABASE_URL_DIRECT" \
  down 1

# Check status
migrate -path database/migrations \
  -database "$DATABASE_URL_DIRECT" \
  version
```

**Makefile integration:**
```makefile
migrate-up:
	@migrate -path database/migrations -database "$(DATABASE_URL_DIRECT)" up

migrate-down:
	@migrate -path database/migrations -database "$(DATABASE_URL_DIRECT)" down 1

migrate-status:
	@migrate -path database/migrations -database "$(DATABASE_URL_DIRECT)" version
```

---

## Service Hosting

### Backend Services (Ingester + API)

#### Option 1: Railway.app ⭐ Recommended

**Pros:**
- ✅ Easiest deployment (git push to deploy)
- ✅ Free $5/month credit
- ✅ Auto-scaling, zero-downtime deploys
- ✅ Built-in monitoring and logs
- ✅ Environment variables per service

**Setup:**
```bash
# 1. Install CLI
npm install -g @railway/cli

# 2. Login and init
railway login
railway init

# 3. Create services
railway add  # Create "ingester" service
railway add  # Create "api" service

# 4. Set environment variables
railway variables set DATABASE_URL="postgresql://..."
railway variables set ETH_RPC_URL="https://..."
railway variables set REDIS_URL="redis://..."

# 5. Deploy from Dockerfile
railway up
```

**railway.json:**
```json
{
  "$schema": "https://railway.app/railway.schema.json",
  "build": {
    "builder": "DOCKERFILE",
    "dockerfilePath": "services/ingester/Dockerfile"
  },
  "deploy": {
    "restartPolicyType": "ON_FAILURE",
    "restartPolicyMaxRetries": 10
  }
}
```

**Dockerfile (services/ingester/Dockerfile):**
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ingester .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ingester .
CMD ["./ingester"]
```

**Pricing:** $10-20/month per service (Starter tier)

#### Option 2: Render.com

**Pros:**
- ✅ Free tier available
- ✅ Docker support
- ✅ Auto-deploy from GitHub

**Setup:**
```yaml
# render.yaml
services:
  - type: web
    name: indexer-api
    env: docker
    dockerfilePath: ./services/api/Dockerfile
    envVars:
      - key: DATABASE_URL
        sync: false
      - key: PORT
        value: 8000
    healthCheckPath: /health
```

**Pricing:** Free tier, then $7-25/month

#### Option 3: Fly.io

**Pros:**
- ✅ Global edge deployment (low latency worldwide)
- ✅ Free tier (3 shared-cpu VMs)
- ✅ Excellent for multi-region

**Setup:**
```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Launch app
fly launch

# Deploy
fly deploy
```

**fly.toml:**
```toml
app = "blockchain-indexer-api"
primary_region = "iad"

[build]
  dockerfile = "services/api/Dockerfile"

[[services]]
  internal_port = 8000
  protocol = "tcp"

  [[services.ports]]
    handlers = ["http"]
    port = 80
```

**Pricing:** Free tier, then pay-as-you-go

---

## Frontend Deployment

### Option 1: Vercel ⭐ Recommended

**Pros:**
- ✅ Built for React/Vite
- ✅ Auto-deploy from GitHub (push to deploy)
- ✅ Global CDN, instant cache invalidation
- ✅ Edge functions for API routes
- ✅ Free SSL, custom domains
- ✅ Preview deployments for PRs

**Setup:**
```bash
# 1. Install CLI
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

**vercel.json:**
```json
{
  "buildCommand": "npm run build",
  "outputDirectory": "dist",
  "framework": "vite",
  "rewrites": [
    {
      "source": "/api/:path*",
      "destination": "https://api.your-domain.com/:path*"
    }
  ]
}
```

**Pricing:** Free for personal, $20/month for teams

### Option 2: Netlify

**Pros:**
- ✅ Similar to Vercel
- ✅ Generous free tier
- ✅ Netlify Forms and Functions

**Setup:**
```bash
npm install -g netlify-cli
netlify deploy --prod
```

**netlify.toml:**
```toml
[build]
  command = "npm run build"
  publish = "dist"

[[redirects]]
  from = "/api/*"
  to = "https://api.your-domain.com/:splat"
  status = 200
```

### Option 3: Cloudflare Pages

**Pros:**
- ✅ Fastest global CDN
- ✅ Free unlimited bandwidth
- ✅ Workers for edge functions

**Pricing:** Free

---

## CI/CD Pipeline

### GitHub Actions Workflow

**.github/workflows/deploy.yml:**
```yaml
name: Deploy to Production

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  GO_VERSION: '1.23'
  NODE_VERSION: '20'

jobs:
  test-backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
      
      - name: Run tests
        run: |
          cd services/ingester && go test ./...
          cd ../api && go test ./...
      
      - name: Lint
        run: |
          go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
          golangci-lint run ./...

  test-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
      
      - name: Install dependencies
        run: cd web && npm ci
      
      - name: Run tests
        run: cd web && npm test
      
      - name: Build
        run: cd web && npm run build

  migrate-staging:
    needs: [test-backend, test-frontend]
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v4
      
      - name: Run migrations
        env:
          DATABASE_URL: ${{ secrets.STAGING_DATABASE_URL }}
        run: |
          curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
          ./migrate -path database/migrations -database "$DATABASE_URL" up

  deploy-backend:
    needs: migrate-staging
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v4
      
      - name: Deploy to Railway
        env:
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
        run: |
          npm install -g @railway/cli
          railway up --service ingester
          railway up --service api

  deploy-frontend:
    needs: [test-frontend, deploy-backend]
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v4
      
      - name: Deploy to Vercel
        env:
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
          VERCEL_ORG_ID: ${{ secrets.VERCEL_ORG_ID }}
          VERCEL_PROJECT_ID: ${{ secrets.VERCEL_PROJECT_ID }}
        run: |
          cd web
          npm install -g vercel
          vercel --prod --token=$VERCEL_TOKEN
```

### Deployment Checklist

**Before deploying:**
- [ ] All tests passing locally
- [ ] Migrations tested on staging database
- [ ] Environment variables documented in `.env.example`
- [ ] Secrets added to hosting provider (not committed to git)
- [ ] Health check endpoints implemented (`/health`, `/ready`)
- [ ] CORS configured for production domains
- [ ] Rate limiting enabled on API
- [ ] Database connection pooling configured
- [ ] Logging configured (structured JSON logs)

**After deploying:**
- [ ] Smoke test all critical endpoints
- [ ] Check monitoring dashboards (no errors)
- [ ] Verify database migrations applied
- [ ] Test frontend loads and connects to API
- [ ] Check SSL certificate valid
- [ ] Monitor logs for first 30 minutes

---

## Security Best Practices

### Secrets Management

**❌ DON'T:**
```bash
# Hardcode secrets
DATABASE_URL="postgresql://user:password@host/db"

# Commit .env files to git
git add .env.production
```

**✅ DO:**
```bash
# Use environment variables
export DATABASE_URL="postgresql://..."

# Use secrets management
railway variables set DATABASE_URL="postgresql://..."
vercel env add DATABASE_URL production

# Or use secret manager
aws secretsmanager get-secret-value --secret-id prod/database-url
```

**vault.yml (if using HashiCorp Vault):**
```yaml
path "secret/blockchain-indexer/prod/*" {
  capabilities = ["read"]
}
```

### API Security

**1. CORS Configuration:**
```go
// services/api/main.go
import "github.com/gin-contrib/cors"

func main() {
    r := gin.Default()
    
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"https://yourapp.com"},
        AllowMethods:     []string{"GET", "POST"},
        AllowHeaders:     []string{"Origin", "Content-Type"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))
}
```

**2. Rate Limiting:**
```go
import "github.com/ulule/limiter/v3"

// 100 requests per minute per IP
rate := limiter.Rate{
    Period: 1 * time.Minute,
    Limit:  100,
}

middleware := limiter.NewMiddleware(limiter.New(
    memory.NewStore(),
    rate,
))

r.Use(middleware)
```

**3. Request Validation:**
```go
type GetBlockRequest struct {
    ChainID     int64 `uri:"chain_id" binding:"required,min=1"`
    BlockNumber int64 `uri:"block_number" binding:"required,min=0"`
}

func getBlock(c *gin.Context) {
    var req GetBlockRequest
    if err := c.ShouldBindUri(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid parameters"})
        return
    }
    // ...
}
```

**4. SQL Injection Prevention:**
```go
// ✅ GOOD: Parameterized queries
db.Query(`SELECT * FROM blocks WHERE chain_id = $1 AND block_number = $2`, chainID, blockNumber)

// ❌ BAD: String concatenation
query := fmt.Sprintf("SELECT * FROM blocks WHERE chain_id = %d", chainID)
```

### TLS/SSL

**Automatic with hosting providers:**
- Vercel: Free SSL (Let's Encrypt)
- Railway: Free SSL (Let's Encrypt)
- Render: Free SSL (Let's Encrypt)

**Custom domain:**
```bash
# Vercel
vercel domains add yourapp.com

# Railway
railway domain add yourapp.com
```

### Database Security

**1. Use connection pooling with auth:**
```go
import "github.com/jackc/pgx/v5/pgxpool"

config, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
config.MaxConns = 20
config.MinConns = 5
config.MaxConnLifetime = time.Hour

pool, _ := pgxpool.NewWithConfig(context.Background(), config)
```

**2. Enable SSL mode:**
```bash
DATABASE_URL="postgresql://user:pass@host/db?sslmode=require"
```

**3. Row Level Security (Supabase):**
```sql
-- Enable RLS
ALTER TABLE blocks ENABLE ROW LEVEL SECURITY;

-- Allow public read access
CREATE POLICY "Public read access"
  ON blocks FOR SELECT
  USING (true);

-- Restrict writes to authenticated users
CREATE POLICY "Authenticated write access"
  ON blocks FOR INSERT
  USING (auth.role() = 'authenticated');
```

---

## Monitoring & Logging

### Structured Logging

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("Block processed",
    zap.Int64("chain_id", chainID),
    zap.Int64("block_number", blockNumber),
    zap.Duration("processing_time", elapsed),
)

logger.Error("RPC call failed",
    zap.Int64("chain_id", chainID),
    zap.Error(err),
)
```

**Output (JSON):**
```json
{
  "level": "info",
  "ts": 1699564800.123,
  "msg": "Block processed",
  "chain_id": 1,
  "block_number": 18500000,
  "processing_time": 0.234
}
```

### Metrics Export

**Prometheus metrics:**
```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    blocksProcessed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "blocks_processed_total",
            Help: "Total number of blocks processed",
        },
        []string{"chain_id"},
    )
    
    processingDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "block_processing_duration_seconds",
            Help:    "Block processing duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"chain_id"},
    )
)

func processBlock(chainID int64, block *Block) {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        processingDuration.WithLabelValues(fmt.Sprint(chainID)).Observe(duration)
        blocksProcessed.WithLabelValues(fmt.Sprint(chainID)).Inc()
    }()
    
    // Process block...
}
```

**Expose metrics endpoint:**
```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

http.Handle("/metrics", promhttp.Handler())
http.ListenAndServe(":9090", nil)
```

### Grafana Dashboards

**Connect Prometheus data source:**
```yaml
# grafana-datasource.yml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
```

**Example queries:**
```promql
# Blocks processed per second (per chain)
rate(blocks_processed_total[1m])

# 95th percentile processing time
histogram_quantile(0.95, rate(block_processing_duration_seconds_bucket[5m]))

# Consumer lag (if exported from Kafka)
kafka_consumer_lag{group="indexer-processor"}
```

### Alerting

**Prometheus alerting rules:**
```yaml
# alerts.yml
groups:
  - name: indexer
    interval: 30s
    rules:
      - alert: IngesterLagging
        expr: (chain_head_block - last_indexed_block) > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Ingester lagging behind chain head"
          description: "Chain {{ $labels.chain_id }} is {{ $value }} blocks behind"
      
      - alert: HighAPILatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API latency high"
          description: "P95 latency is {{ $value }}s"
      
      - alert: DatabaseConnectionPoolExhausted
        expr: db_connections_available < 5
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database connection pool nearly exhausted"
```

### Error Tracking with Sentry

```go
import "github.com/getsentry/sentry-go"

func main() {
    sentry.Init(sentry.ClientOptions{
        Dsn: os.Getenv("SENTRY_DSN"),
        Environment: os.Getenv("ENVIRONMENT"),
    })
    defer sentry.Flush(2 * time.Second)
    
    // Capture errors
    if err != nil {
        sentry.CaptureException(err)
    }
}
```

### Health Checks

```go
// /health - liveness probe (is process running?)
func healthHandler(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
}

// /ready - readiness probe (can accept traffic?)
func readyHandler(c *gin.Context) {
    // Check database connection
    ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
    defer cancel()
    
    if err := db.PingContext(ctx); err != nil {
        c.JSON(503, gin.H{
            "status": "not ready",
            "database": "unavailable",
        })
        return
    }
    
    // Check Redis
    if err := redis.Ping(ctx).Err(); err != nil {
        c.JSON(503, gin.H{
            "status": "not ready",
            "cache": "unavailable",
        })
        return
    }
    
    c.JSON(200, gin.H{"status": "ready"})
}
```

---

## Cost Optimization

### Cost Estimates

#### Development (Shared Database)

| Service | Provider | Tier | Cost |
|---------|----------|------|------|
| Database | Supabase | Free | $0 |
| Redis | Upstash | Free | $0 |
| Hosting | Local | N/A | $0 |
| **Total** | | | **$0/month** |

#### Staging

| Service | Provider | Tier | Cost |
|---------|----------|------|------|
| Database | Supabase | Free | $0 |
| Backend (2 services) | Railway | Hobby | $10 |
| Frontend | Vercel | Free | $0 |
| Redis | Upstash | Free | $0 |
| **Total** | | | **$10/month** |

#### Production (Low Traffic: <100K requests/day)

| Service | Provider | Tier | Cost |
|---------|----------|------|------|
| Database | Supabase | Pro | $25 |
| Backend (2 services) | Railway | Starter | $20 |
| Frontend | Vercel | Pro | $20 |
| Redis | Upstash | Pay-as-you-go | $5 |
| Monitoring | Grafana Cloud | Free | $0 |
| RPC Providers | Alchemy | Growth | $49 |
| **Total** | | | **$119/month** |

#### Production (High Traffic: 1M+ requests/day)

| Service | Provider | Tier | Cost |
|---------|----------|------|------|
| Database | Supabase | Pro + Compute | $25-100 |
| Backend | Railway | Pro (4 instances) | $50-200 |
| Frontend | Vercel | Pro | $20 |
| Redis | Upstash | Pay-as-you-go | $20-50 |
| Monitoring | Grafana Cloud | Pro | $50 |
| RPC Providers | Alchemy | Scale | $199-499 |
| CDN | Cloudflare | Pro | $20 |
| **Total** | | | **$384-1,019/month** |

### Optimization Strategies

**1. Database:**
- Use connection pooling (reduces connections by 90%)
- Index optimization (faster queries = fewer resources)
- Partition old data (move to cheaper storage)
- Read replicas for queries (scale reads independently)

**2. Caching:**
- Cache immutable data (blocks never change)
- Short TTL for recent data (5 minutes)
- Use Redis for rate limiting (cheaper than compute)

**3. Compute:**
- Auto-scale based on traffic (Railway/Render)
- Use serverless functions for infrequent tasks
- Batch database writes (reduce transaction overhead)
- Optimize RPC calls (cache chain head, use WebSockets)

**4. Bandwidth:**
- Use CDN for static assets (Cloudflare/Vercel)
- Compress API responses (gzip)
- Paginate large result sets
- Use GraphQL for precise data fetching

**5. RPC Providers:**
- Use multiple providers for failover
- Cache blockchain data aggressively
- Use WebSocket subscriptions (cheaper than polling)
- Archive nodes only when necessary

---

## Interview Questions

### Q1: How would you design a CI/CD pipeline for a blockchain indexer?

**Answer:**

**Pipeline stages:**

**1. Test** (on every PR):
```yaml
- Unit tests (Go, TypeScript)
- Integration tests (database, Kafka)
- Linting (golangci-lint, ESLint)
- Security scan (gosec, npm audit)
```

**2. Build** (on merge to main):
```yaml
- Build Docker images
- Tag with git commit SHA
- Push to container registry
```

**3. Deploy Staging**:
```yaml
- Run database migrations (staging DB)
- Deploy containers to staging
- Run smoke tests
- Wait for manual approval
```

**4. Deploy Production**:
```yaml
- Run database migrations (production DB)
- Blue-green deployment (zero downtime)
- Health check validation
- Rollback if health checks fail
```

**Key considerations:**
- **Database migrations first**: Ensure schema changes before code
- **Feature flags**: Enable new features gradually
- **Monitoring**: Watch error rates for 15 minutes post-deploy
- **Rollback plan**: One-command rollback to previous version

**For blockchain indexer specifically:**
- Test against mainnet fork (Hardhat/Anvil)
- Validate reorg handling in staging
- Check consumer lag doesn't spike after deploy

### Q2: How do you implement zero-downtime deployments?

**Answer:**

**Blue-Green Deployment:**

**Setup:**
```
Blue Environment (current production):
  - API v1.2.3
  - Ingester v1.2.3
  
Green Environment (new version):
  - API v1.3.0
  - Ingester v1.3.0
```

**Process:**
1. Deploy green environment
2. Run health checks on green
3. Switch load balancer to green (instant cutover)
4. Monitor for 15 minutes
5. If successful, decommission blue
6. If errors, switch back to blue (instant rollback)

**Database migrations:**
```sql
-- Make changes backward compatible
-- Add new column (nullable first)
ALTER TABLE blocks ADD COLUMN new_field TEXT;

-- Deploy code that writes to both old and new columns
-- (deployed in previous release)

-- Backfill data
UPDATE blocks SET new_field = old_field WHERE new_field IS NULL;

-- Make column NOT NULL
ALTER TABLE blocks ALTER COLUMN new_field SET NOT NULL;

-- Remove old column (in next release)
ALTER TABLE blocks DROP COLUMN old_field;
```

**For Railway/Render:**
- Enable "Wait for health check" before switching traffic
- Set health check endpoint: `/ready`
- Configure: 30s timeout, 3 retry attempts

### Q3: How would you handle secrets management in production?

**Answer:**

**❌ Bad practices:**
- Hardcoding secrets in code
- Committing `.env` files to git
- Storing secrets in CI/CD config files
- Sharing secrets via Slack/email

**✅ Good practices:**

**1. Environment Variables (hosting provider):**
```bash
# Railway
railway variables set DATABASE_URL="postgresql://..."
railway variables set SENTRY_DSN="https://..."

# Secrets are encrypted at rest
# Only available to running containers
# Never logged or exposed in UI
```

**2. HashiCorp Vault (enterprise):**
```go
import "github.com/hashicorp/vault/api"

client, _ := api.NewClient(&api.Config{
    Address: "https://vault.company.com",
})

secret, _ := client.Logical().Read("secret/blockchain-indexer/prod/database")
dbURL := secret.Data["url"].(string)
```

**3. AWS Secrets Manager:**
```bash
# Store secret
aws secretsmanager create-secret \
  --name prod/database-url \
  --secret-string "postgresql://..."

# Retrieve in app
aws secretsmanager get-secret-value --secret-id prod/database-url
```

**4. Kubernetes Secrets:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: database-credentials
type: Opaque
data:
  url: cG9zdGdyZXNxbDovLy4uLg==  # base64 encoded
```

**Rotation strategy:**
- Rotate database passwords quarterly
- Rotate API keys every 90 days
- Use short-lived tokens (JWT with 1-hour expiry)
- Automate rotation with Vault or AWS Secrets Manager

### Q4: How do you monitor and debug production issues?

**Answer:**

**Layers of monitoring:**

**1. Infrastructure Metrics (Prometheus + Grafana):**
```
- CPU, memory, disk usage per service
- Database connection pool utilization
- Network throughput
- Container restart count
```

**2. Application Metrics:**
```
- Blocks processed per second
- API request rate and latency (P50, P95, P99)
- Consumer lag (Kafka)
- Cache hit rate
```

**3. Logs (Structured JSON):**
```go
logger.Info("Block processed",
    zap.Int64("chain_id", 1),
    zap.Int64("block_number", 18500000),
    zap.Duration("duration", 234*time.Millisecond),
)
```

Ship logs to: Grafana Loki, Datadog, or CloudWatch

**4. Distributed Tracing (Jaeger):**
```go
import "go.opentelemetry.io/otel"

ctx, span := tracer.Start(ctx, "processBlock")
defer span.End()

// Trace follows request through: API → Kafka → Processor → Database
```

**5. Error Tracking (Sentry):**
```go
sentry.CaptureException(err)
sentry.CaptureMessage("Unexpected behavior detected")
```

**Debugging workflow:**
1. Alert fires: "IngesterLagging"
2. Check Grafana: Which chain? How far behind?
3. Check logs: Any RPC errors? Database timeouts?
4. Check Sentry: Any recent errors?
5. Check traces: Where is bottleneck?
6. Fix root cause
7. Deploy fix
8. Verify alert clears

### Q5: How would you optimize costs for a blockchain indexer?

**Answer:**

**Database optimization (biggest cost driver):**

**1. Connection pooling:**
```go
// ❌ BAD: 100 containers × 20 connections = 2000 DB connections
db, _ := sql.Open("postgres", dbURL)

// ✅ GOOD: Use PgBouncer (Supabase port 6543)
db, _ := sql.Open("postgres", dbURL+"?pgbouncer=true")
// 100 containers → 50 pooled connections → database
```
**Savings:** $50-100/month (can use smaller database tier)

**2. Index optimization:**
```sql
-- Find slow queries
SELECT query, mean_exec_time FROM pg_stat_statements
ORDER BY mean_exec_time DESC LIMIT 10;

-- Add indexes
CREATE INDEX idx_blocks_chain_number ON blocks (chain_id, block_number);
```
**Result:** 100ms queries → 5ms = can handle 20x more traffic with same database

**3. Partition old data:**
```sql
-- Move data older than 6 months to cheaper storage
CREATE TABLE blocks_archive PARTITION OF blocks
FOR VALUES FROM ('2023-01-01') TO ('2023-07-01');

-- Store on HDD instead of SSD
ALTER TABLE blocks_archive SET TABLESPACE hdd_tablespace;
```
**Savings:** $20-50/month

**RPC optimization:**

**1. Use multiple providers:**
```go
// Free tiers: Alchemy (300M CU/month) + Infura (100K req/day)
providers := []string{
    os.Getenv("ALCHEMY_RPC"),
    os.Getenv("INFURA_RPC"),
    os.Getenv("QUICKNODE_RPC"),
}
// Round-robin or fallback on rate limit
```
**Savings:** $50-100/month vs paid tier

**2. Cache blockchain data:**
```go
// Cache chain head for 12 seconds (1 block time)
redis.Set("eth:head", blockNumber, 12*time.Second)
```
**Savings:** 90% fewer RPC calls

**3. Use WebSocket subscriptions:**
```go
// ❌ Polling: 5 requests/second = 432K req/day
// ✅ WebSocket: 1 persistent connection
```

**Compute optimization:**

**1. Auto-scaling:**
```yaml
# Scale down during low traffic (nights, weekends)
min_instances: 1
max_instances: 5
target_cpu: 70%
```
**Savings:** $30-50/month

**2. Use spot instances (AWS/GCP):**
**Savings:** 70% cheaper than on-demand

**Total potential savings: $200-400/month** (from $600 → $200-400)

### Q6: What's your database backup and disaster recovery strategy?

**Answer:**

**Backup strategy:**

**1. Automated daily backups (Supabase):**
```
- Automatic daily backups (retained 7 days)
- Point-in-time recovery (PITR) for last 7 days
- Can restore to any second within retention window
```

**2. Manual backups before risky operations:**
```bash
# Before major migration
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d_%H%M%S).sql

# Compressed
pg_dump $DATABASE_URL | gzip > backup.sql.gz
```

**3. WAL archival (production):**
```sql
-- Continuous archiving of Write-Ahead Logs to S3
archive_mode = on
archive_command = 'aws s3 cp %p s3://backups/wal/%f'
```
**Enables:** Point-in-time recovery for any timestamp

**Disaster recovery:**

**RTO (Recovery Time Objective): 15 minutes**
**RPO (Recovery Point Objective): 5 minutes**

**Scenarios:**

**1. Database corruption:**
```bash
# Restore from last backup
supabase db restore --backup-id 20250127_0300

# Or from pg_dump
psql $DATABASE_URL < backup_20250127.sql
```

**2. Accidental data deletion:**
```sql
-- PITR to 5 minutes before incident
-- (Supabase dashboard: Restore to timestamp)
```

**3. Region outage:**
```
Primary: us-east-1 (Supabase)
Failover: us-west-2 (read replica promoted to primary)

DNS: Update to point to failover region (TTL 60s)
Result: 5-10 minute downtime
```

**4. Complete data loss:**
```bash
# Create new database
# Restore from S3 backup
aws s3 sync s3://backups/wal/ /var/lib/postgresql/wal/
pg_ctl start

# Replay WAL logs
# Rebuild Kafka topics from blockchain (slow but possible)
```

**Testing:**
- Quarterly backup restoration drill
- Document recovery procedures
- Time the recovery process
- Validate data integrity after restoration

### Q7: How do you handle configuration across multiple environments?

**Answer:**

**12-Factor App principles:**

**1. Environment hierarchy:**
```
.env.example          # Template (committed to git)
.env.local            # Local overrides (git-ignored)
.env.development      # Dev database (git-ignored)
.env.staging          # Staging (git-ignored)
.env.production       # Production (git-ignored)
```

**2. Environment-specific settings:**
```bash
# .env.development
DATABASE_URL=postgresql://localhost:5432/indexer_dev
LOG_LEVEL=debug
RATE_LIMIT=1000

# .env.production
DATABASE_URL=postgresql://db.prod.supabase.co:6543/postgres
LOG_LEVEL=info
RATE_LIMIT=100
```

**3. Config loading precedence:**
```go
// 1. Environment variables (highest priority)
dbURL := os.Getenv("DATABASE_URL")

// 2. .env file
if dbURL == "" {
    godotenv.Load()
    dbURL = os.Getenv("DATABASE_URL")
}

// 3. Defaults (lowest priority)
if dbURL == "" {
    dbURL = "postgresql://localhost:5432/indexer"
}
```

**4. Validation:**
```go
type Config struct {
    DatabaseURL string `env:"DATABASE_URL,required"`
    RedisURL    string `env:"REDIS_URL,required"`
    LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
    Port        int    `env:"PORT" envDefault:"8000"`
}

func LoadConfig() (*Config, error) {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, err
    }
    return cfg, nil
}
```

**5. Secrets vs Config:**

**Config** (can be in git):
- Port numbers
- Log levels
- Feature flags
- Timeout values

**Secrets** (never in git):
- Database passwords
- API keys
- Private keys
- OAuth tokens

**Best practices:**
- One config per environment
- No secrets in code or git
- Validate on startup (fail fast)
- Document all variables in `.env.example`
- Use type-safe config parsing

---

## Related Documentation

- [Database-Fundamentals.md](./Database-Fundamentals.md) - Database concepts
- [PostgreSQL-Database.md](./PostgreSQL-Database.md) - PostgreSQL specifics
- [System-Design-Architecture.md](./System-Design-Architecture.md) - Architecture patterns
- [Docker-Kubernetes.md](./Docker-Kubernetes.md) - Container orchestration

---

**Last Updated**: 2025-11-27
