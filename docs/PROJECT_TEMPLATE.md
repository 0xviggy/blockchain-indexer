# Project Structure Template

> **Purpose**: A reusable template for how to structure complex software projects.  
> This project (blockchain-indexer) follows this pattern and can serve as a reference for new projects.

**Last Updated**: December 20, 2025

---

## 📋 Table of Contents

1. [Documentation Philosophy](#documentation-philosophy)
2. [Repository Structure](#repository-structure)
3. [Documentation Hierarchy](#documentation-hierarchy)
4. [Technology Stack Decisions](#technology-stack-decisions)
5. [Development Workflow](#development-workflow)
6. [Infrastructure & DevOps](#infrastructure--devops)
7. [Code Organization Patterns](#code-organization-patterns)
8. [Development Lifecycle](#development-lifecycle)
9. [Best Practices & Conventions](#best-practices--conventions)
10. [Lessons Learned](#lessons-learned)

---

## 1. Documentation Philosophy

### Core Principle: README as the Hub

**The root README.md should provide access to everything a developer needs:**

```
README.md (Hub)
├── Quick Start (< 5 min to running)
├── Links to Setup Guides → docs/setup/
├── Links to Architecture → DESIGN_DECISIONS.md
├── Links to Specs → docs/TECHNICAL_SPEC.md
├── Links to Learning → learning/
└── Links to Progress → PROGRESS_TRACKING.md
```

### Documentation Types

| Type | Location | Purpose |
|------|----------|---------|
| **Setup Guides** | `docs/setup/` | How to run the project |
| **Architecture** | Root `DESIGN_DECISIONS.md` | Why decisions were made |
| **Technical Spec** | `docs/TECHNICAL_SPEC.md` | What is implemented |
| **Business Spec** | `docs/BUSINESS_SPEC.md` | What problem we solve |
| **Progress** | Root `PROGRESS_TRACKING.md` | Current state |
| **Learning** | `learning/` | Educational, interview prep |

### Separation of Concerns

- **Project Docs**: How to work on THIS project
- **Educational Docs**: General concepts, interview prep (clearly labeled with ⚠️)

---

## 2. Repository Structure

### Recommended Layout

```
project-name/
├── README.md                       # ⭐ Hub - links to everything
├── DESIGN_DECISIONS.md             # Architecture rationale
├── PROGRESS_TRACKING.md            # Current status
├── Makefile                        # Central development commands
├── .env.example                    # Environment template
│
├── docs/                           # Project documentation
│   ├── README.md                   # Docs index
│   ├── TECHNICAL_SPEC.md           # Implementation details
│   ├── BUSINESS_SPEC.md            # Product requirements
│   ├── PROJECT_TEMPLATE.md         # This file - structure guide
│   └── setup/                      # ⭐ All setup guides together
│       ├── SANDBOX_SETUP.md        # Local dev environment
│       ├── DATABASE_GUIDE.md       # Database operations
│       └── DEPLOYMENT.md           # Production deployment
│
├── learning/                       # ⭐ Educational materials
│   ├── README.md                   # Learning guide index
│   └── [Topic].md                  # Topic guides with interview Q&A
│
├── services/                       # Backend services
│   ├── api/                        # REST API service
│   └── ingester/                   # Data ingestion service
│
├── web/                            # Frontend application
│   └── README.md                   # Frontend-specific docs
│
├── database/                       # Database files
│   ├── migrations/                 # Schema migrations
│   └── seeds/                      # Sample data
│
├── infrastructure/                 # DevOps & deployment
│   └── docker/                     # Docker configs
│
└── scripts/                        # Utility scripts
```

---

## 3. Documentation Hierarchy

### For This Project (blockchain-indexer)

```
README.md
├── Quick Start → docs/setup/SANDBOX_SETUP.md
├── Database Guide → docs/setup/DATABASE_GUIDE.md
├── Architecture → DESIGN_DECISIONS.md
│   └── Multi-chain strategy (merged from CHAIN_SUPPORT)
├── Technical Details → docs/TECHNICAL_SPEC.md
│   └── API specs, service details
├── Business Context → docs/BUSINESS_SPEC.md
├── Project Status → PROGRESS_TRACKING.md
├── Deployment → docs/setup/DEPLOYMENT.md
└── Learning Resources → learning/
    ├── System-Design-Architecture.md
    ├── Go-Programming.md
    ├── MEV_ANALYSIS.md
    └── ... other educational content
```

---

## 4. Technology Stack Decisions

### Backend

| Component | Technology | Version | Rationale |
|-----------|-----------|---------|-----------|
| **Language** | Go | 1.21+ | Performance, concurrency, blockchain ecosystem |
| **HTTP Framework** | Gin | v1.10+ | Fast router, middleware support |
| **Database Driver** | lib/pq | v1.10+ | PostgreSQL native driver |
| **Blockchain Client** | go-ethereum | v1.13+ | Official Ethereum client library |
| **Migrations** | golang-migrate | v4.17+ | Database version control |

### Frontend

| Component | Technology | Version | Rationale |
|-----------|-----------|---------|-----------|
| **Framework** | React | 19.2+ | Component-based, large ecosystem |
| **Build Tool** | Vite | 7.2+ | Fast HMR, modern build system |
| **Language** | TypeScript | 5.9+ | Type safety, better DX |
| **Styling** | Tailwind CSS | 4.1+ | Utility-first, fast development |
| **State Management** | Zustand | 5.0+ | Simple, performant state |
| **Data Fetching** | TanStack Query | 5.90+ | Server state management |
| **Routing** | React Router | 7.9+ | Client-side routing |
| **Icons** | Lucide React | 0.553+ | Modern icon library |

### Infrastructure

| Component | Technology | Version | Rationale |
|-----------|-----------|---------|-----------|
| **Database** | PostgreSQL | 15+ | JSONB, partitioning, performance |
| **Cache** | Redis | 7+ | In-memory speed, pub/sub |
| **Message Queue** | Kafka/Redpanda | v23.2+ | Event streaming, horizontal scaling |
| **Container Runtime** | Docker | 24+ | Local dev consistency |
| **Orchestration** | Docker Compose | v2.20+ | Multi-container management (local) |

### DevOps & Production

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Database (Prod)** | Supabase | Managed PostgreSQL, free tier, auto-backups |
| **Backend Hosting** | Railway.app / Render | Docker support, environment variables |
| **Frontend Hosting** | Vercel / Netlify | Static hosting, CDN, edge functions |
| **Monitoring** | Grafana Cloud | Metrics visualization (planned) |
| **Error Tracking** | Sentry | Error aggregation (planned) |

---

## 5. Development Workflow

### 🎯 Central Command: Makefile

**Philosophy**: All common operations accessible via `make` commands

```makefile
# Quick reference - see actual Makefile for full implementation
make help           # Show all available commands
make setup          # Complete initial setup (docker + migrations)
make docker-up      # Start infrastructure (postgres, redis, kafka)
make migrate-up     # Apply database migrations
make run-ingester   # Start ingester service
make run-api        # Start API service
make test           # Run all tests
make status         # Check all service status
make clean          # Clean up everything
```

### Daily Development Flow

```bash
# 1. Start your day
make docker-up          # Start infrastructure
make migrate-up         # Ensure DB is up-to-date
make status             # Verify everything is running

# 2. Start services (separate terminals)
make run-ingester       # Terminal 1
make run-api            # Terminal 2
cd web && npm run dev   # Terminal 3

# 3. Make changes
# Edit code...
# Services auto-reload (ingester/api restart via make commands)
# Frontend has HMR (hot module reload)

# 4. Test changes
make test               # Run backend tests
cd web && npm test      # Run frontend tests (if configured)

# 5. Database changes
make migrate-create NAME=add_new_table
# Edit migration files
make migrate-up

# 6. End of day
make stop-services      # Stop all Go services
make docker-down        # Stop infrastructure (optional - can keep running)
```

### Git Workflow

```bash
# 1. Feature branch
git checkout -b feature/add-event-parsing

# 2. Make changes with clear commits
git commit -m "feat: Add ERC20 Transfer event parsing"
git commit -m "test: Add event parsing tests"
git commit -m "docs: Update DEVELOPMENT_STATUS.md with progress"

# 3. Keep DEVELOPMENT_STATUS.md updated
# Edit docs/DEVELOPMENT_STATUS.md after each significant change

# 4. Push and create PR
git push origin feature/add-event-parsing
```

### Commit Message Convention

```
feat: Add new feature
fix: Bug fix
docs: Documentation only
style: Code style (formatting, no logic change)
refactor: Code refactoring
test: Add or modify tests
chore: Build/tooling changes
perf: Performance improvement
```

---

## 6. Infrastructure & DevOps

### Local Development Setup

**Docker Compose Strategy**:
```yaml
# infrastructure/docker/docker-compose.yml
services:
  postgres:  # Primary database
  redis:     # Caching layer
  kafka:     # Message queue (Redpanda for simplicity)
  kafka-ui:  # Development tool
  pgadmin:   # Database admin tool
```

**Benefits**:
- ✅ Consistent environment across developers
- ✅ One command setup (`make docker-up`)
- ✅ No conflicts with local system services
- ✅ Easy to reset/clean (`make clean`)

### Database Migrations

**Tool**: golang-migrate (industry standard)

**File Naming**: `<version>_<description>.<up|down>.sql`
```
000001_initial_schema.up.sql
000001_initial_schema.down.sql
000002_add_calldata_parsing.up.sql
000002_add_calldata_parsing.down.sql
```

**Migration Workflow**:
```bash
# 1. Create new migration
make migrate-create NAME=add_user_table

# 2. Edit generated files
# database/migrations/000004_add_user_table.up.sql
# database/migrations/000004_add_user_table.down.sql

# 3. Apply migration
make migrate-up

# 4. Test rollback
make migrate-down
make migrate-up  # Re-apply to ensure idempotency
```

**Migration Best Practices**:
- Always write both UP and DOWN migrations
- Test rollback before committing
- One logical change per migration
- Use transactions (`BEGIN;` ... `COMMIT;`)
- Add comments explaining complex logic

### Environment Configuration

**.env Pattern**:
```bash
.env.example        # Committed - template with dummy values
.env                # Git-ignored - actual values
.env.local          # Git-ignored - developer overrides
```

**Configuration Loading** (Go):
```go
// shared/config/config.go
func Load() *Config {
    return &Config{
        DatabaseURL: getEnv("DATABASE_URL", "postgres://..."),
        RedisURL:    getEnv("REDIS_URL", "redis://..."),
    }
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
```

### Production Deployment

**Multi-Environment Strategy**:
```
Development  → Supabase Free (dev-project)
Staging      → Supabase Pro (staging-project) [Optional]
Production   → Supabase Pro (prod-project)
```

**Deployment Checklist**:
- [ ] Environment variables configured
- [ ] Database migrations applied
- [ ] Health checks working
- [ ] Monitoring enabled
- [ ] Secrets in vault (not env vars)
- [ ] Rate limiting configured
- [ ] CORS configured
- [ ] Backups scheduled

---

## 7. Code Organization Patterns

### Go Service Structure

```go
// services/ingester/main.go
package main

// 1. Types & Structs (at top)
type ChainConfig struct { ... }
type Ingester struct { ... }

// 2. Main function (entry point)
func main() {
    // Load config
    // Connect dependencies
    // Setup graceful shutdown
    // Start goroutines
}

// 3. Core business logic
func (i *Ingester) ingestChain(chain ChainConfig) { ... }
func (i *Ingester) processBlock(block *Block) { ... }

// 4. Database operations
func insertBlock(db *sql.DB, block Block) error { ... }
func insertTransactionsBatch(db *sql.Tx, txs []Transaction) error { ... }

// 5. Utility functions
func connectDB() (*sql.DB, error) { ... }
func loadChainConfigs() []ChainConfig { ... }
```

**Key Patterns**:
- Receiver methods for service operations
- Clear separation of concerns
- Batch operations for performance
- Context-based cancellation
- Graceful shutdown with sync.WaitGroup

### API Service Structure (Gin)

```go
// services/api/main.go
package main

// 1. Types (request/response models)
type Chain struct { ... }
type Transaction struct { ... }

// 2. Main function
func main() {
    // Initialize DB
    // Create API instance
    // Setup routes
    // Start server
}

// 3. Route setup
func (a *API) setupRoutes() {
    a.router.GET("/api/chains", a.getChains)
    a.router.GET("/api/chains/:id/transactions", a.getTransactions)
}

// 4. Handler functions
func (a *API) getChains(c *gin.Context) { ... }
func (a *API) getTransactions(c *gin.Context) { ... }

// 5. Database queries
func (a *API) queryChains() ([]Chain, error) { ... }
```

**API Best Practices**:
- RESTful endpoint naming
- Consistent response format
- Proper HTTP status codes
- CORS configuration
- Connection pooling
- Query parameter validation

### Frontend Structure (React)

```typescript
// web/src/App.tsx
import { useState, useEffect } from 'react'

// 1. Types (imported or defined)
type TabType = 'transactions' | 'events' | 'health'

// 2. Component
function App() {
    // State declarations
    const [data, setData] = useState<Data[]>([])
    
    // Effects
    useEffect(() => {
        loadData()
    }, [])
    
    // Handlers
    const handleRefresh = async () => { ... }
    
    // Render
    return (
        <div>...</div>
    )
}

// 3. Helper components (in same file if small)
function DataTable({ data }: { data: Data[] }) { ... }
```

**Frontend Patterns**:
- Custom hooks for reusable logic
- API client abstraction (`lib/api.ts`)
- Type-safe API calls
- Error boundaries
- Loading states
- Optimistic updates

### Shared Code (Go Modules)

```
shared/
├── go.mod                  # Module definition
├── config/
│   └── config.go           # Configuration loading
└── models/
    └── models.go           # Shared data structures
```

**Usage in services**:
```go
// services/ingester/go.mod
require (
    github.com/0xviggy/blockchain-indexer/shared v0.0.0
)

// In code
import "github.com/0xviggy/blockchain-indexer/shared/models"
```

**Note**: This project keeps Go modules separate (not using Go workspaces) for simplicity.

---

## 8. Development Lifecycle

### Phase-Based Development

```
Phase 0: Planning & Architecture (1-2 days)
├── Write BUSINESS_SPEC.md
├── Write TECHNICAL_SPEC.md
├── Design database schema
└── Create repository structure

Phase 1: Infrastructure (1 day)
├── Docker Compose setup
├── Database migrations (001_initial_schema)
├── Makefile with basic commands
└── Environment configuration

Phase 2: Core Services (3-5 days)
├── Ingester service (basic block/tx ingestion)
├── API service (basic CRUD endpoints)
└── Testing & validation

Phase 3: Frontend Foundation (2-3 days)
├── Vite + React + TypeScript setup
├── Tailwind CSS configuration
├── API client
└── Basic UI components

Phase 4: Feature Iteration (ongoing)
├── Add advanced features (event parsing, reorg detection)
├── Improve UI/UX
├── Add monitoring
└── Performance optimization

Phase 5: Production Readiness (2-3 days)
├── Deployment guide
├── CI/CD pipeline
├── Monitoring & alerting
└── Documentation polish
```

### Sprint Workflow

**Sprint Planning** (Start of week):
1. Review DEVELOPMENT_STATUS.md
2. Identify highest priority tasks
3. Break down into actionable items
4. Estimate effort (hours/days)
5. Commit to sprint goals

**Daily Development**:
1. Pick task from sprint backlog
2. Create feature branch
3. Implement + test
4. Update DEVELOPMENT_STATUS.md
5. Commit with clear message
6. Push and create PR (if working with team)

**Sprint Review** (End of week):
1. Demo completed work
2. Update DEVELOPMENT_STATUS.md completion dates
3. Document technical decisions
4. Identify blockers for next sprint
5. Plan next sprint

### Testing Strategy

**Backend Testing**:
```go
// services/ingester/main_test.go
func TestBlockIngestion(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    defer db.Close()
    
    // Execute
    err := insertBlock(db, testBlock)
    
    // Verify
    assert.NoError(t, err)
    assertBlockExists(t, db, testBlock.Number)
}
```

**Run tests**: `make test` or `go test ./...`

**Frontend Testing** (planned):
```typescript
// web/src/__tests__/App.test.tsx
import { render, screen } from '@testing-library/react'
import App from '../App'

test('renders transactions tab', () => {
    render(<App />)
    expect(screen.getByText('Transactions')).toBeInTheDocument()
})
```

---

## 9. Best Practices & Conventions

### Code Style

**Go**:
- `gofmt` for formatting (automatic)
- Variables: `camelCase` for private, `PascalCase` for exported
- Errors: Return error as last return value
- Context: First parameter in functions
- Comments: Full sentences with proper capitalization

**TypeScript/React**:
- ESLint configuration (see `web/eslint.config.js`)
- Components: PascalCase (`TransactionTable.tsx`)
- Files: kebab-case for utilities (`api-client.ts`)
- Props: Destructure in function signature

### Database Design

**Partitioning Strategy**:
```sql
-- Partition by chain_id for multi-chain data
CREATE TABLE blocks (...) PARTITION BY LIST (chain_id);
CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks FOR VALUES IN (137);
```

**Indexing Strategy**:
```sql
-- Index frequently queried fields
CREATE INDEX idx_transactions_from ON transactions_eth(from_address);
CREATE INDEX idx_transactions_block ON transactions_eth(block_number DESC);

-- GIN index for JSONB
CREATE INDEX idx_events_decoded ON events_eth USING GIN(decoded_data);
```

### Error Handling

**Go**:
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to fetch block %d: %w", blockNum, err)
}

// Retry logic for transient errors
for attempt := 0; attempt <= maxRetries; attempt++ {
    if attempt > 0 {
        time.Sleep(backoff * time.Duration(attempt))
    }
    err := tryOperation()
    if err == nil {
        break
    }
}
```

**TypeScript**:
```typescript
// API client with error handling
async function fetchTransactions(chainId: number): Promise<Transaction[]> {
    try {
        const response = await fetch(`/api/chains/${chainId}/transactions`)
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }
        return await response.json()
    } catch (error) {
        console.error('Failed to fetch transactions:', error)
        throw error
    }
}
```

### Performance Optimization

**Database**:
```go
// Batch INSERT instead of individual inserts
func insertTransactionsBatch(db *sql.Tx, txs []Transaction) error {
    var query strings.Builder
    query.WriteString("INSERT INTO transactions (...) VALUES ")
    
    args := make([]interface{}, 0, len(txs)*11)
    for i, tx := range txs {
        if i > 0 {
            query.WriteString(", ")
        }
        query.WriteString(fmt.Sprintf("($%d, $%d, ...)", i*11+1, i*11+2))
        args = append(args, tx.ChainID, tx.TxHash, ...)
    }
    
    _, err := db.Exec(query.String(), args...)
    return err
}
```

**API**:
```go
// Connection pooling
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(time.Hour)

// Response caching (planned)
// Use Redis for hot data
```

---

## 10. Lessons Learned

### What Worked Well

1. **Makefile as Central Hub**
   - Single source of truth for all commands
   - Easy onboarding (just run `make help`)
   - Consistency across developers

2. **DEVELOPMENT_STATUS.md Living Document**
   - Invaluable for tracking progress
   - Great for resume material
   - Helps with project handoffs

3. **Documentation First Approach**
   - Write specs before code
   - Reduces rework and confusion
   - Facilitates better design decisions

4. **Separate Learning Directory**
   - Great for onboarding new developers
   - Interview preparation materials
   - Reference for best practices

5. **Docker Compose for Local Dev**
   - Fast setup (5 minutes to running)
   - No dependency conflicts
   - Easy to reset/clean

6. **golang-migrate for Database**
   - Version control for schema
   - Easy rollback
   - Works with multiple environments

7. **Batch Processing for Performance**
   - 300x speedup (188 individual INSERTs → 1 batch)
   - Reduced database connection overhead
   - Lower transaction commit overhead

### What Could Be Improved

1. **Go Workspace Not Used**
   - Kept modules separate for simplicity
   - Could use Go workspaces for easier local development
   - Trade-off: Simplicity vs convenience

2. **Frontend Component Organization**
   - Large App.tsx file (983 lines)
   - Should split into smaller components
   - Better separation of concerns needed

3. **Testing Coverage**
   - Limited test coverage currently
   - Should add tests alongside features (not after)
   - Integration tests needed

4. **Monitoring Not Implemented Yet**
   - Grafana/Prometheus planned but not done
   - Should be part of initial setup
   - Harder to add later

5. **API Documentation**
   - No OpenAPI/Swagger spec yet
   - Endpoints documented in code comments only
   - Should generate API docs automatically

6. **CI/CD Pipeline**
   - Manual deployment currently
   - GitHub Actions not set up
   - Automated testing on PR needed

### Key Insights

1. **Start with Infrastructure**
   - Get Docker, DB, migrations working first
   - Everything else builds on this foundation

2. **Document as You Go**
   - Don't wait until the end
   - Update DEVELOPMENT_STATUS.md after each feature
   - Future you will thank present you

3. **Batch Operations Are Critical**
   - Single biggest performance improvement
   - Always consider batching for database operations
   - Network latency matters more than you think

4. **Fail-Fast Philosophy**
   - Don't save incomplete data
   - Retry transient errors, fail on permanent ones
   - Better to skip a block than corrupt data

5. **Makefile Saves Time**
   - Invest time in good Makefile upfront
   - Pays off every single day
   - Reduces cognitive load

6. **Multi-Chain Adds Complexity**
   - Partition tables by chain_id
   - Independent checkpoints per chain
   - Different finality rules per chain

---

## 🚀 Quick Start Checklist for New Projects

Using this template for a new project:

### Phase 0: Planning (Day 1)
- [ ] Create repository structure (copy template)
- [ ] Write BUSINESS_SPEC.md (Why? Who? What?)
- [ ] Write TECHNICAL_SPEC.md (How? Architecture?)
- [ ] Design database schema
- [ ] Choose technology stack
- [ ] Create .env.example

### Phase 1: Infrastructure (Day 1-2)
- [ ] Create docker-compose.yml
- [ ] Write initial database migration
- [ ] Create Makefile with basic commands
- [ ] Test `make setup` works end-to-end
- [ ] Write infrastructure README

### Phase 2: Backend Foundation (Day 2-3)
- [ ] Create main service (Go/Python/Node)
- [ ] Implement basic CRUD operations
- [ ] Add database connection pooling
- [ ] Write initial tests
- [ ] Document API endpoints

### Phase 3: Frontend Foundation (Day 3-4)
- [ ] Initialize frontend (Vite/Next/etc)
- [ ] Configure styling (Tailwind/etc)
- [ ] Create API client
- [ ] Build basic UI
- [ ] Test API integration

### Phase 4: Development Loop (Ongoing)
- [ ] Create DEVELOPMENT_STATUS.md
- [ ] Implement features iteratively
- [ ] Update docs after each feature
- [ ] Write tests alongside code
- [ ] Commit with clear messages

### Phase 5: Production Prep (Final Week)
- [ ] Write DEPLOYMENT.md
- [ ] Set up CI/CD pipeline
- [ ] Add monitoring/logging
- [ ] Security audit
- [ ] Load testing
- [ ] Final documentation polish

---

## 📚 Additional Resources

### Project Documentation
- [BUSINESS_SPEC.md](./BUSINESS_SPEC.md) - Business requirements
- [TECHNICAL_SPEC.md](./TECHNICAL_SPEC.md) - Technical architecture
- [DEVELOPMENT_STATUS.md](./DEVELOPMENT_STATUS.md) - Current progress
- [DEPLOYMENT.md](./DEPLOYMENT.md) - Deployment guide

### Learning Materials
- [learning/README.md](../learning/README.md) - Learning guide index
- [learning/System-Design-Architecture.md](../learning/System-Design-Architecture.md) - System design patterns

### Tools & References
- [golang-migrate](https://github.com/golang-migrate/migrate) - Database migrations
- [Gin Framework](https://gin-gonic.com/) - Go HTTP framework
- [Supabase](https://supabase.com/) - Managed PostgreSQL
- [Railway](https://railway.app/) - Backend hosting
- [Vercel](https://vercel.com/) - Frontend hosting

---

## 📝 Template Usage Notes

**When adapting this template**:

1. **Keep the structure** - It's proven to work
2. **Adapt the content** - Replace blockchain-specific with your domain
3. **Maintain Makefile** - Adapt commands but keep the pattern
4. **Keep DEVELOPMENT_STATUS.md** - This is the secret sauce
5. **Document decisions** - Future you will need context

**What to customize**:
- [ ] Technology stack (Go → Python/Node, React → Vue/Svelte)
- [ ] Service names (ingester → crawler, processor → analyzer)
- [ ] Documentation content (blockchain → your domain)
- [ ] Database schema (multi-chain → your data model)

**What to keep**:
- [x] Repository structure (docs/, services/, learning/, infrastructure/)
- [x] Makefile pattern (central command hub)
- [x] Documentation strategy (specs, status, deployment)
- [x] Development workflow (feature branches, clear commits)

---

## 🎯 Success Metrics for Using This Template

Your project is on track if:

✅ New developer can run the project in <15 minutes  
✅ All common operations accessible via Makefile  
✅ DEVELOPMENT_STATUS.md updated after each feature  
✅ Documentation explains *why*, not just *what*  
✅ Database schema under version control  
✅ Environment config via .env files  
✅ Docker Compose for local development  
✅ Clear separation between services  

---

**Template Version**: 1.0  
**Last Updated**: December 19, 2025  
**Maintained By**: 0xviggy  
**License**: MIT

