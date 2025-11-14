# Blockchain Indexer

A high-performance, multi-chain blockchain indexer with native support for Ethereum, Polygon, Arbitrum, Optimism, and more.

## Architecture

```
indexer/
├── services/
│   ├── ingester/      # Fetches blocks from blockchain networks
│   ├── processor/     # Processes and parses blockchain events
│   └── api/           # REST API and WebSocket server
├── shared/            # Shared Go packages (models, utils)
├── database/          # SQL migrations and schemas
├── infrastructure/    # Docker, Kubernetes configs
├── monitoring/        # Prometheus, Grafana configs
└── docs/              # Documentation
```

## Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Kafka or RabbitMQ
- Redis

### Development Setup

```bash
# Start dependencies
docker-compose up -d postgres redis kafka

# Run database migrations
make migrate

# Start services
make run-ingester
make run-processor
make run-api
```

## Services

### Ingester
Subscribes to blockchain networks, fetches blocks and transactions, publishes to message broker.

### Processor
Consumes messages, parses events (ERC20, ERC721, custom), writes to database.

### API
Serves blockchain data via REST and WebSocket with caching and rate limiting.

## Features

- ✅ Multi-chain support (Ethereum, Polygon, Arbitrum, Optimism, Base)
- ✅ Real-time indexing with <30s lag
- ✅ Cross-chain queries
- ✅ Event parsing (ERC20, ERC721)
- ✅ Automatic reorg handling
- ✅ Comprehensive observability (Prometheus, Grafana, Jaeger)
- ✅ Horizontal scaling
- ✅ Production-ready with fault tolerance

## Documentation

- [Business Specification](./BUSINESS_SPEC.md)
- [Technical Specification](./TECHNICAL_SPEC.md)
- [Multi-Chain Support](./MULTICHAIN_SPEC.md)
- [Language Choice](./LANGUAGE_CHOICE.md)
- [Learning Guide & Interview Prep](./LEARNING_GUIDE.md) 📚
- [API Documentation](./docs/API.md)

## License

MIT
