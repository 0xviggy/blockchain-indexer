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

## License

MIT
