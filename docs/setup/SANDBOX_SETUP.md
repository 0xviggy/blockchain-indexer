# Developer Sandbox Setup

Complete guide to set up your local development environment for the Blockchain Indexer.

---

## 🎯 Goal

Get a **functional development environment** in < 10 minutes with:
- ✅ Database with schema and sample data
- ✅ Working API server
- ✅ Functional UI (without needing to run ingester)
- ✅ Ready to develop features

---

## Prerequisites

### Required Software

- [ ] **Docker Desktop** - For PostgreSQL, Redis
  - Download: https://www.docker.com/products/docker-desktop
  - Verify: `docker --version`

- [ ] **Go 1.21+** - For backend services
  - Download: https://golang.org/dl/
  - Verify: `go version`

- [ ] **Node.js 18+** - For frontend
  - Download: https://nodejs.org/
  - Verify: `node --version`

- [ ] **golang-migrate** - For database migrations
  ```bash
  # macOS
  brew install golang-migrate
  
  # Linux
  curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
  sudo mv migrate /usr/local/bin/
  ```
  - Verify: `migrate -version`

### Optional

- [ ] **Git** - Usually pre-installed
- [ ] **make** - Usually pre-installed on macOS/Linux

---

## Quick Setup (5 Steps)

### Step 1: Clone Repository

```bash
git clone https://github.com/0xviggy/blockchain-indexer.git
cd blockchain-indexer
```

✅ **Verify**: You're in the project directory

### Step 2: Configure Environment

```bash
# Copy template
cp .env.example .env
```

**Default values work fine for local development!** You only need to edit `.env` if:
- You want to run the ingester (requires RPC API keys)
- You're deploying to production

✅ **Verify**: `.env` file exists

### Step 3: Start Infrastructure

```bash
make setup
```

This single command:
1. Starts Docker containers (PostgreSQL, Redis)
2. Waits for services to be healthy
3. Applies all database migrations
4. Verifies everything is ready

**Expected output:**
```
🐳 Starting infrastructure...
✅ Containers started
⏳ Waiting for services to be ready...
✅ All services ready
📈 Applying migrations...
✅ Migrations complete
✅ Setup complete! Infrastructure is ready.
```

✅ **Verify**: `make status` shows all containers running

### Step 4: Load Sample Data

```bash
make db-seed
```

Loads test data so you can immediately see a functional UI:
- 8 sample blocks (Ethereum + Polygon)
- 9 sample transactions
- 3 sample event logs

✅ **Verify**:
```bash
make db-shell
# Then: SELECT COUNT(*) FROM blocks WHERE block_number < 1000;
# Should show: 8
# Type: \q to exit
```

### Step 5: Start Services

**Terminal 1 - API:**
```bash
make run-api
```

**Terminal 2 - Frontend:**
```bash
cd web
npm install  # First time only
npm run dev
```

✅ **Verify**: 
- API: http://localhost:8000/health
- Frontend: http://localhost:5173

---

## 🎉 Success!

You now have:
- ✅ Database with schema and sample data
- ✅ API server responding to requests
- ✅ **Functional UI showing blocks and transactions**

**You can start developing without running the ingester!**

---

## Optional: Run Ingester

To index **real blockchain data**:

### 1. Get API Keys

Sign up for free at:
- **Alchemy**: https://www.alchemy.com/
- **Infura**: https://infura.io/

### 2. Configure .env

```bash
vim .env
```

Add your RPC URLs:
```env
ETH_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
ETH_WS_URL=wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
```

### 3. Start Ingester

**Terminal 3:**
```bash
make run-ingester
```

The ingester will:
- Connect to Ethereum (or other chains)
- Start indexing from current block
- Store data in your local database

✅ **Verify**: Watch Terminal 3 for "Indexed block XXXXX" messages

---

## Daily Development Workflow

### Morning Routine

```bash
# 1. Pull latest code
git pull

# 2. Start infrastructure (if stopped)
make docker-up

# 3. Apply any new migrations
make migrate-up

# 4. Verify you're in sync
make migrate-status
```

### Start Working

```bash
# Terminal 1: API
make run-api

# Terminal 2: Frontend
cd web && npm run dev

# Terminal 3: Ingester (optional)
make run-ingester
```

### Check Status

```bash
# Service status
make status

# View logs
make logs

# Database shell
make db-shell
```

### End of Day

```bash
# Optional: Stop containers to free resources
make docker-down

# Data persists in Docker volumes
# Next `make docker-up` restores everything
```

---

## Troubleshooting

### Docker Commands Hanging

**Problem**: `docker ps` hangs or doesn't respond

**Solution**:
```bash
# Restart Docker Desktop
pkill -f "Docker.app" && sleep 2 && open -a Docker

# Wait 30 seconds, then:
make docker-up
```

### Port Already in Use

**Problem**: `Error: Port 5432 already in use`

**Solution**:
```bash
# Check what's using the port
lsof -i :5432

# Stop local PostgreSQL if running
brew services stop postgresql
# OR
sudo systemctl stop postgresql

# Then restart
make docker-up
```

### Migration Version Mismatch

**Problem**: `migration version X is dirty`

**Solution**:
```bash
# Check status
make migrate-status

# Force to last good version
make migrate-force
# Enter: 2 (one before failed migration)

# Re-apply
make migrate-up
```

### Database Connection Error

**Problem**: `connection refused` or `could not connect to database`

**Solution**:
```bash
# Check container is running
docker ps | grep indexer-postgres

# If not running:
make docker-up
make migrate-up

# Test connection
make db-shell
```

### Frontend Won't Start

**Problem**: `npm run dev` fails

**Solution**:
```bash
cd web

# Clean install
rm -rf node_modules package-lock.json
npm install

# Try again
npm run dev
```

### Ingester Errors

**Problem**: RPC errors, rate limiting, or connection issues

**Solution**:
```bash
# Check .env has valid API keys
cat .env | grep RPC_URL

# Verify API key works
curl https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# Should return current block number
```

### Seed Data Issues

**Problem**: Sample data not appearing in UI

**Solution**:
```bash
# Clear and reload seed data
make db-clear-seeds
make db-seed

# Verify in database
make db-shell
# Then: SELECT COUNT(*) FROM blocks WHERE block_number < 1000;
```

---

## Command Reference

### Infrastructure

```bash
make setup           # Complete first-time setup
make docker-up       # Start containers
make docker-down     # Stop containers  
make status          # Check service status
make logs            # View container logs
```

### Database

```bash
make migrate-up      # Apply pending migrations
make migrate-status  # Check current version
make migrate-create  # Create new migration
make db-shell        # PostgreSQL CLI
make db-seed         # Load sample data
make db-clear-seeds  # Remove sample data
make db-reset        # ⚠️ Delete all data + re-migrate
```

### Development

```bash
make run-api         # Start API server
make run-ingester    # Start blockchain ingester
make stop-services   # Stop all Go services

# Frontend (run from web/)
npm run dev          # Start dev server
npm run build        # Build for production
npm run preview      # Preview production build
```

### Utilities

```bash
make redis-cli       # Open Redis CLI
make explore-rpc     # Test RPC connections
make test            # Run tests
```

---

## Development Tools

### PostgreSQL CLI

```bash
make db-shell

# Inside psql:
\dt                              # List tables
\d blocks                        # Describe table
\d+ blocks                       # Detailed table info
SELECT * FROM schema_migrations; # Check migration version
SELECT COUNT(*) FROM blocks;     # Count blocks
\q                              # Quit
```

### Database Admin UI

**pgAdmin** - Web interface for PostgreSQL (optional)  
- URL: http://localhost:5050
- Email: `admin@indexer.local`
- Password: `admin`

### API Testing

```bash
# Health check
curl http://localhost:8000/health

# Get blocks
curl http://localhost:8000/api/blocks?chain_id=1&limit=10

# Get single block
curl http://localhost:8000/api/blocks/1/100

# Get transactions
curl http://localhost:8000/api/transactions?chain_id=1&limit=10
```

---

## Next Steps

### After Setup

1. ✅ Verify UI shows sample blocks/transactions at http://localhost:5173
2. ✅ Explore the database: `make db-shell`
3. ✅ Review the codebase:
   - `services/api/` - REST API server
   - `services/ingester/` - Blockchain indexer
   - `web/src/` - React frontend
   - `database/migrations/` - Schema migrations

### Start Contributing

1. Check [../PROGRESS_TRACKING.md](../PROGRESS_TRACKING.md) for current priorities
2. Pick a task from "Immediate Next Steps"
3. Create a branch: `git checkout -b feature/my-feature`
4. Make changes and test locally
5. Submit a pull request

### Learn More

- **Database**: See [DATABASE_GUIDE.md](DATABASE_GUIDE.md)
- **Project Status**: See [../PROGRESS_TRACKING.md](../PROGRESS_TRACKING.md)
- **Architecture**: See [../TECHNICAL_SPEC.md](../TECHNICAL_SPEC.md)
- **Deployment**: See [DEPLOYMENT.md](DEPLOYMENT.md)

---

## Support

- **Setup Issues**: Check troubleshooting section above
- **Database Questions**: See [DATABASE_GUIDE.md](DATABASE_GUIDE.md)
- **Feature Questions**: See [../PROGRESS_TRACKING.md](../PROGRESS_TRACKING.md)
- **Bug Reports**: [Create GitHub issue](https://github.com/0xviggy/blockchain-indexer/issues)

---

**🚀 You're ready to develop!**

Your sandbox is set up with sample data and a functional UI. Start the API and frontend, then begin coding!
