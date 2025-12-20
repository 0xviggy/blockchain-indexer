# Blockchain Indexer

Multi-chain blockchain indexing service with real-time data ingestion, REST API, and web UI.

**Status**: ✅ Functional Development Environment  
**Version**: 0.1.0 (Pre-production)

---

## 🚀 Quick Start

### For Developers (< 10 minutes)

```bash
# 1. Clone
git clone https://github.com/0xviggy/blockchain-indexer.git
cd blockchain-indexer

# 2. Install golang-migrate (required)
brew install golang-migrate  # macOS
# Linux: see SANDBOX_SETUP.md

# 3. Load sample data
make db-seed            # Optional but recommended

# 4. Start services
make run-api            # Terminal 1
cd web && npm run dev   # Terminal 2
```

**Result**: Functional UI at http://localhost:5173 with sample blockchain data (no ingester needed!)

**Full setup guide**: [docs/setup/SANDBOX_SETUP.md](docs/setup/SANDBOX_SETUP.md)

---

## 📚 Documentation

### Getting Started

| Document | Purpose | Read Time |
|----------|---------|-----------|
| **[docs/setup/SANDBOX_SETUP.md](docs/setup/SANDBOX_SETUP.md)** | Set up local development environment | 5 min |
| **[docs/setup/DATABASE_GUIDE.md](docs/setup/DATABASE_GUIDE.md)** | Database setup, migrations, seed data | 10 min |
| **[docs/PROGRESS_TRACKING.md](docs/PROGRESS_TRACKING.md)** | Project status, next steps, roadmap | 5 min |

### Architecture & Design

| Document | Purpose |
|----------|---------|
| [docs/DESIGN_DECISIONS.md](docs/DESIGN_DECISIONS.md) | Architecture rationale, multi-chain strategy |
| [docs/TECHNICAL_SPEC.md](docs/TECHNICAL_SPEC.md) | Implementation details, API specs |
| [docs/BUSINESS_SPEC.md](docs/BUSINESS_SPEC.md) | Product requirements |
| [docs/setup/DEPLOYMENT.md](docs/setup/DEPLOYMENT.md) | Production deployment guide |
| [docs/PROJECT_TEMPLATE.md](docs/PROJECT_TEMPLATE.md) | Project structure template for new projects |

### Learning & Reference

| Document | Purpose |
|----------|---------|
| [learning/](learning/) | Educational deep-dives and interview prep |
| [learning/System-Design-Architecture.md](learning/System-Design-Architecture.md) | System design patterns |
| [learning/Go-Programming.md](learning/Go-Programming.md) | Go patterns and concepts |
| [learning/MEV_ANALYSIS.md](learning/MEV_ANALYSIS.md) | MEV detection strategies |

---

## 🎯 What This Does

### Core Features

- **Multi-Chain Indexing** - Ethereum, Polygon, Arbitrum, Optimism, Base
- **Real-Time Ingestion** - Continuous blockchain data indexing
- **REST API** - Query blocks, transactions, logs
- **Web UI** - Block explorer and ingester control panel
- **Data Quality** - Transaction receipts, status tracking, error handling

### Use Cases

- 📊 Blockchain analytics and insights
- 🔍 Transaction and block exploration  
- 📈 Protocol activity monitoring
- 💱 Cross-chain transaction tracking
- 🧪 Development and testing with sample data

---

## 🏗️ Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Blockchain │────▶│   Ingester   │────▶│  PostgreSQL │
│  RPC Nodes  │     │   Service    │     │  Database   │
└─────────────┘     └──────────────┘     └─────────────┘
                                                 │
                                                 ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Web UI    │◀────│  API Service │◀────│  Partitioned│
│   (React)   │     │   (Go)       │     │   Tables    │
└─────────────┘     └──────────────┘     └─────────────┘
```

### Tech Stack

- **Backend**: Go 1.21+
- **Database**: PostgreSQL 15 (partitioned tables, migrations)
- **Frontend**: React 18 + TypeScript + Vite
- **Infrastructure**: Docker Compose
- **Caching**: Redis (planned for production)
- **Messaging**: Direct PostgreSQL writes (Kafka planned for >10 chains)

---

## 📦 Project Structure

```
blockchain-indexer/
├── README.md                  # This file (navigation hub)
├── Makefile                   # Central development commands
│
├── docs/                      # Project documentation
│   ├── DESIGN_DECISIONS.md    # Architecture & design rationale
│   ├── PROGRESS_TRACKING.md   # Project status
│   ├── TECHNICAL_SPEC.md      # Implementation details
│   ├── BUSINESS_SPEC.md       # Product requirements
│   ├── PROJECT_TEMPLATE.md    # Template for new projects
│   └── setup/                 # ⭐ Setup & operational guides
│       ├── SANDBOX_SETUP.md   # Local development
│       ├── DATABASE_GUIDE.md  # Database operations
│       └── DEPLOYMENT.md      # Production deployment
│
├── services/                  # Backend services
│   ├── ingester/              # Blockchain data ingestion
│   └── api/                   # REST API server
│
├── web/                       # Frontend React app
│   └── src/                   # React components
│
├── database/                  # Database files
│   ├── migrations/            # Schema migrations
│   └── seeds/                 # Sample data
│
├── infrastructure/            # Docker & deployment
│   └── docker/                # Docker Compose configs
│
├── learning/                  # ⭐ Educational materials
│   ├── Go-Programming.md      # Go concepts & patterns
│   ├── System-Design-Architecture.md
│   ├── MEV_ANALYSIS.md        # MEV detection research
│   └── ...                    # More interview prep guides
│
└── shared/                    # Shared Go packages
    ├── config/                # Configuration
    └── models/                # Data models
```

---

## 🛠️ Development Commands

### Infrastructure

```bash
make setup           # Complete first-time setup
make docker-up       # Start Docker containers
make docker-down     # Stop Docker containers
make status          # Check service status
make logs            # View container logs
```

### Database

```bash
make migrate-up      # Apply pending migrations
make migrate-status  # Check migration version
make db-seed         # Load sample data
make db-shell        # PostgreSQL CLI
```

### Services

```bash
make run-api         # Start API server (port 8000)
make run-ingester    # Start blockchain ingester
make stop-services   # Stop all Go services
```

### Frontend

```bash
cd web
npm run dev          # Development server (port 5173)
npm run build        # Production build
npm run preview      # Preview production build
```

**Full command reference**: `make help`

---

## 🎓 For New Developers

### 1. Environment Setup

Follow **[docs/setup/SANDBOX_SETUP.md](docs/setup/SANDBOX_SETUP.md)** for complete setup guide including:
- Prerequisites installation
- Repository configuration  
- Infrastructure startup
- Sample data loading
- Service verification

### 2. Database Workflow

See **[docs/setup/DATABASE_GUIDE.md](docs/setup/DATABASE_GUIDE.md)** for:
- Migration system
- Creating schema changes
- Seed data management
- Team coordination
- Troubleshooting

### 3. Current Status

Check **[docs/PROGRESS_TRACKING.md](docs/PROGRESS_TRACKING.md)** for:
- What's implemented
- What's in progress
- What's next
- How to contribute

### 4. Architecture

Read **[docs/DESIGN_DECISIONS.md](docs/DESIGN_DECISIONS.md)** for:
- Why decisions were made
- Multi-chain support strategy
- Database partitioning approach

---

## 🔄 Development Workflow

### Daily Routine

```bash
# 1. Sync with team
git pull
make migrate-up

# 2. Start development
make docker-up
make run-api
cd web && npm run dev

# 3. Make changes, test, commit
git checkout -b feature/my-feature
# ... make changes ...
git commit -am "Add feature X"
git push origin feature/my-feature
```

### Working with Database

```bash
# Create migration
make migrate-create
# Enter name: add_my_table

# Edit migration files
vim database/migrations/000004_add_my_table.up.sql
vim database/migrations/000004_add_my_table.down.sql

# Test
make migrate-up
make migrate-down
make migrate-up
```

---

## 📊 Current Status

### ✅ Completed

- Infrastructure (Docker Compose)
- Database schema with migrations
- Ingester service (blocks + transactions)
- API service (REST endpoints)
- Frontend UI (block explorer)
- Transaction receipt fetching
- Error tracking and retries
- Sample seed data

### 🔄 In Progress

- Event log parsing
- Protocol signature matching
- UI polish and enhancements

### 📋 Planned

- Analytics features
- MEV detection
- Production deployment
- Monitoring and metrics

**Detailed status**: [docs/PROGRESS_TRACKING.md](docs/PROGRESS_TRACKING.md)

---

## 🤝 Contributing

### Prerequisites

- Read [docs/setup/SANDBOX_SETUP.md](docs/setup/SANDBOX_SETUP.md) for environment setup
- Review [docs/PROGRESS_TRACKING.md](docs/PROGRESS_TRACKING.md) for current priorities
- Check [docs/setup/DATABASE_GUIDE.md](docs/setup/DATABASE_GUIDE.md) for database workflow

### Process

1. Create feature branch: `git checkout -b feature/name`
2. Make changes and test locally
3. Ensure migrations are tested (up + down)
4. Commit with clear message
5. Push and create pull request

### Code Style

- **Go**: Follow standard Go conventions (`gofmt`, `golint`)
- **TypeScript**: ESLint configuration in `web/`
- **SQL**: Use migrations for all schema changes

---

## 📞 Support

### Documentation

Start here based on your need:

- **Setup help**: [docs/setup/SANDBOX_SETUP.md](docs/setup/SANDBOX_SETUP.md)
- **Database questions**: [docs/setup/DATABASE_GUIDE.md](docs/setup/DATABASE_GUIDE.md)
- **Project status**: [docs/PROGRESS_TRACKING.md](docs/PROGRESS_TRACKING.md)
- **Architecture**: [docs/DESIGN_DECISIONS.md](docs/DESIGN_DECISIONS.md)
- **Implementation**: [docs/TECHNICAL_SPEC.md](docs/TECHNICAL_SPEC.md)

### Troubleshooting

Common issues and solutions:

- **Docker hanging**: See [docs/setup/SANDBOX_SETUP.md#troubleshooting](docs/setup/SANDBOX_SETUP.md#troubleshooting)
- **Migration errors**: See [docs/setup/DATABASE_GUIDE.md#troubleshooting](docs/setup/DATABASE_GUIDE.md#troubleshooting)
- **Port conflicts**: See [docs/setup/SANDBOX_SETUP.md#port-already-in-use](docs/setup/SANDBOX_SETUP.md#port-already-in-use)

### Getting Help

- Check documentation first (links above)
- Search existing issues
- Create [GitHub issue](https://github.com/0xviggy/blockchain-indexer/issues)

---

## 📝 License

[Your License Here]

---

## 🙏 Acknowledgments

Built with:
- [golang-migrate](https://github.com/golang-migrate/migrate) - Database migrations
- [go-ethereum](https://github.com/ethereum/go-ethereum) - Ethereum client library
- [React](https://react.dev/) - UI framework
- [PostgreSQL](https://www.postgresql.org/) - Database
- [Docker](https://www.docker.com/) - Containerization

---

**Ready to start?** → [docs/setup/SANDBOX_SETUP.md](docs/setup/SANDBOX_SETUP.md)



