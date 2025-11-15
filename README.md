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

# 4. Start services (coming soon)
make run-ingester
make run-processor
make run-api
```

**Prerequisites**: Docker Desktop, Go 1.21+  
**Full setup guide**: See [Learning Guide](./docs/LEARNING_GUIDE.md#prerequisites--installation)

### Multi-Chain Signature Analysis

Discover unknown function signatures across any EVM chain:

```bash
# Set RPC URLs for chains you want to analyze
export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
export POLYGON_RPC_URL="https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY"

# Run analysis (scans last 10 blocks, ~500-1000 transactions)
make explore-rpc
```

**Output**: Statistics on signature coverage, top unknown functions with 4byte.directory lookup links  
**Use case**: Before launching on a new chain, discover which protocols are popular  
**Details**: See [Multi-Block Analysis Guide](./docs/LEARNING_GUIDE.md#how-to-recreate-multi-block-signature-analysis)

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
