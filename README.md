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

# 3. Start services (coming soon)
make run-ingester
make run-processor
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

**Services**: Ingester (fetch blocks) → Processor (parse events) → API (serve data)  
**Stack**: Go, PostgreSQL (partitioned), Kafka, Redis, Docker  
**Details**: See [Technical Spec](./docs/TECHNICAL_SPEC.md)

## Documentation

| Doc | Purpose |
|-----|---------|
| **[Learning Guide](./docs/LEARNING_GUIDE.md)** | 📚 **Start here** - Setup, implementation log, decisions, interview prep |
| [Technical Spec](./docs/TECHNICAL_SPEC.md) | Architecture, algorithms, multi-chain design |
| [Business Spec](./docs/BUSINESS_SPEC.md) | Requirements, KPIs, use cases |
| [Chain Support Strategy](./docs/CHAIN_SUPPORT.md) | ⛓️ Which chains to support, priorities, cost analysis |

## Project Status

**Phase 1: Infrastructure** ✅ Complete  
**Phase 2: Ingester Service** 🔄 Next  
**Phase 3: Processor Service** 📋 Planned  
**Phase 4: API Service** 📋 Planned

See [Learning Guide](./docs/LEARNING_GUIDE.md#implementation-progress-tracker) for detailed progress.

## License

MIT
