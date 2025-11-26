# Blockchain Indexer

A production-grade, multi-chain blockchain indexer that ingests, processes, and serves blockchain data with sub-second latency. Built with Go, event-driven architecture, and designed for scale.

## Features

- 🌐 **Multi-chain**: Ethereum, Polygon, Arbitrum, Optimism, Base (5 chains, easily extensible)
- ⚡ **Real-time**: <30s lag, WebSocket streaming
- 🔍 **Queryable**: REST API for blocks, transactions, events, cross-chain queries
- 🔄 **Resilient**: Automatic reorg handling, fault tolerance, retries
- 📊 **Observable**: Prometheus metrics, Grafana dashboards, distributed tracing
- 🚀 **Scalable**: Event-driven, horizontally scalable services

## Quick Start

```bash
# 1. Start infrastructure (PostgreSQL, Redis, Kafka)
make docker-up

# 2. Run database migrations
make migrate

# 3. Explore RPC data and validate schemas
export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
make explore-rpc

# 4. Start services
make run-ingester
make run-api
```

**Prerequisites**: Docker Desktop, Go 1.21+  
**Full setup guide**: See [Learning Guide](./docs/LEARNING_GUIDE.md#prerequisites--installation)



## Architecture

```
Blockchain → Ingester → Kafka → Processor → PostgreSQL → API → Users
                                              ↓
                                           Redis Cache
```

**Stack**: Go, PostgreSQL (partitioned), Kafka, Redis, Docker  
**MVP**: Ingester → PostgreSQL → API (Kafka/Processor deferred)

## Documentation

| Doc | Purpose |
|-----|---------|
| **[Development Status](./docs/DEVELOPMENT_STATUS.md)** | 🎯 **Start here** - Progress tracker, decisions, TODOs, roadmap |
| **[Learning Guide](./docs/LEARNING_GUIDE.md)** | 📚 Deep dive - Implementation details, tutorials, interview prep |
| [Technical Spec](./docs/TECHNICAL_SPEC.md) | Architecture, algorithms, multi-chain design |
| [Business Spec](./docs/BUSINESS_SPEC.md) | Requirements, KPIs, use cases |
| [Chain Support Strategy](./docs/CHAIN_SUPPORT.md) | ⛓️ Which chains to support, priorities, cost analysis |

## Project Status

**Current**: Phase 5.1 (Backend Data Correctness) - Fixing transaction status & event parsing  
**Completed**: Phases 0-4.1 (Infrastructure, Ingester, API, Frontend foundation)  
**Next**: Phase 4.2 (Frontend UI), Phase 5.2-5.3 (Observability, Performance)

See **[DEVELOPMENT_STATUS.md](./docs/DEVELOPMENT_STATUS.md)** for detailed progress, technical decisions, and prioritized roadmap.

### Quick Test

```bash
# Terminal 1: Start infrastructure
make docker-up && make migrate

# Terminal 2: Start ingester (needs RPC URL)
export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
make run-ingester

# Terminal 3: Start API
make run-api

# Terminal 4: Start frontend
cd web && npm run dev

# Visit: http://localhost:5173 (frontend) or http://localhost:8000/health (API)
```

## Development Workflow

### Managing Services

```bash
# Check what's running
make status

# Stop all services cleanly
make stop-services

# Start services (auto-stops duplicates first)
make run-api        # Terminal 1
make run-ingester   # Terminal 2
cd web && npm run dev   # Terminal 3
```

**Note**: Each `make run-*` command automatically stops existing instances before starting to prevent duplicate processes.

### Ingester Control

Use the **⚙️ Ingester Control** tab in the UI (http://localhost:5173) to:
- View current checkpoint and ingester status
- Reset checkpoint to reprocess historical blocks
- Clear skipped blocks (e.g., blocks that failed due to RPC errors)
- Test specific block ranges (e.g., block 23,883,999 for blob transaction testing)

After changing the checkpoint, restart the ingester: `make run-ingester`

### Common Issues

**Duplicate processes running?**
```bash
make stop-services  # Stops all Go services
make status         # Verify everything stopped
```

**Port already in use?**
```bash
lsof -ti:8000 | xargs kill -9  # Kill process on port 8000
```

## License

MIT
