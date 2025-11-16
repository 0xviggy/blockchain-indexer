# Learning Guide & Interview Preparation

This document captures all setup steps, learning points, architectural decisions, and technical concepts for interview preparation.

> **📝 Documentation Practice**: As we build this system, every implementation step, decision, and learning point is documented here in real-time. This serves as both a reference and interview preparation material.

---

## 🎯 Technology Stack Overview (Job Requirements Alignment)

This project demonstrates proficiency across multiple technology domains commonly required in modern software engineering roles:

### **Backend Engineering**
| Technology | Project Usage | Interview Relevance |
|------------|---------------|---------------------|
| **Go** | Primary language for services (Ingester, API) | Concurrency, performance-critical systems |
| **Node.js** | *(Planned: WebSocket service for real-time updates)* | Event-driven architecture, microservices |
| **REST APIs** | JSON API with pagination, filtering | Standard web service patterns |
| **WebSocket** | Real-time block subscriptions from RPC nodes | Bidirectional communication |

### **Databases & Data Storage**
| Technology | Project Usage | Interview Relevance |
|------------|---------------|---------------------|
| **PostgreSQL** | Primary datastore with partitioning, triggers, views | ACID transactions, complex queries, indexing strategies |
| **Redis** | Caching layer for API responses | High-speed data structures, pub/sub patterns |
| **MongoDB** | *(Not used - PostgreSQL chosen for ACID guarantees)* | Document stores vs relational trade-offs |

### **Message Queues & Event Streaming**
| Technology | Project Usage | Interview Relevance |
|------------|---------------|---------------------|
| **Kafka (Redpanda)** | Event streaming for parsed transactions | Distributed systems, event sourcing |
| **Redis Streams** | *(Alternative considered for simpler deployment)* | Lightweight message queues |

### **Containerization & Orchestration**
| Technology | Project Usage | Interview Relevance |
|------------|---------------|---------------------|
| **Docker** | All services containerized (PostgreSQL, Redis, Kafka, custom services) | Container fundamentals, networking, volumes |
| **Docker Compose** | Local development orchestration | Multi-container applications |
| **Kubernetes** | *(Production deployment - covered in interview prep below)* | Cloud-native deployments, scaling |

### **Frontend Development**
| Technology | Project Usage | Interview Relevance |
|------------|---------------|---------------------|
| **React** | Dashboard UI with hooks, context API | Component-driven development |
| **Next.js** | *(Considered for SSR dashboard)* | Full-stack framework, SSR/SSG |
| **TypeScript** | Type-safe frontend code | Static typing, large codebases |
| **Vite** | Fast development build tool | Modern bundlers |
| **Tailwind CSS** | Utility-first styling | Rapid UI development |

### **Blockchain Technologies**
| Technology | Project Usage | Interview Relevance |
|------------|---------------|---------------------|
| **Ethereum** | Primary chain, EVM-compatible standard | Smart contracts, RPC/WebSocket APIs |
| **go-ethereum** | Official Go client library | Low-level blockchain interaction |
| **Web3.js / ethers.js** | *(Frontend wallet integration)* | Blockchain frontends, MetaMask |
| **Multi-chain (Polygon, Arbitrum, Optimism, Base)** | Production-scale indexing across L1/L2 | Cross-chain architecture |

### **DevOps & Infrastructure**
| Technology | Project Usage | Interview Relevance |
|------------|---------------|---------------------|
| **Makefile** | Task automation and developer experience | Build systems |
| **GitHub Actions** | *(Planned: CI/CD pipeline)* | Automated testing and deployment |
| **Prometheus + Grafana** | *(Planned: Metrics and monitoring)* | Observability |

### **Key Architecture Patterns Demonstrated**
- ✅ **Microservices Architecture**: Separate ingester, processor, API services
- ✅ **Event-Driven Design**: Kafka for decoupled services
- ✅ **CQRS (Command Query Responsibility Segregation)**: Write-optimized ingester, read-optimized API
- ✅ **Database Partitioning**: Horizontal scaling strategy
- ✅ **Graceful Degradation**: WebSocket with HTTP polling fallback
- ✅ **Idempotency**: Safe to re-run ingestion without duplicates
- ✅ **Checkpointing**: Resume processing from failure points

---

## 📚 Extended Interview Preparation Topics

### Docker Deep Dive

#### Core Concepts
```dockerfile
# Multi-stage builds for smaller images
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o service

FROM alpine:latest
COPY --from=builder /app/service /service
CMD ["/service"]
```

**Interview Questions**:
- **Q: How do multi-stage builds reduce image size?**
  - A: First stage compiles with full SDK (~800MB), second stage copies only the binary to minimal Alpine (~5MB base). Final image is <50MB vs >800MB.
  
- **Q: What's the difference between CMD and ENTRYPOINT?**
  - A: `ENTRYPOINT` defines the executable, `CMD` provides default arguments. `ENTRYPOINT` can't be overridden easily, `CMD` can be replaced at runtime.

- **Q: How do you debug a crashed container?**
  ```bash
  # View logs even after exit
  docker logs <container_id>
  
  # Inspect exit code
  docker inspect <container_id> | jq '.[0].State'
  
  # Start with shell override
  docker run -it --entrypoint /bin/sh <image>
  ```

#### Networking
```yaml
# Docker Compose network configuration
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true  # No external access
```

**Interview Questions**:
- **Q: How do containers communicate in Docker Compose?**
  - A: By service name (DNS resolution). `http://postgres:5432` works because Docker Compose creates a network and DNS entries.

- **Q: What's the difference between bridge and host networking?**
  - **Bridge**: Isolated network with port mapping (default)
  - **Host**: Shares host's network stack (better performance, less isolation)
  - **None**: No networking (for security)

#### Volumes
```bash
# Named volumes (managed by Docker)
docker volume create postgres_data

# Bind mounts (host directory)
-v /host/path:/container/path

# Anonymous volumes (temporary)
-v /container/path
```

**Interview Questions**:
- **Q: When would you use volumes vs bind mounts?**
  - **Volumes**: Production databases, managed by Docker, better performance
  - **Bind mounts**: Development (live code reload), configuration files
  - **tmpfs**: Sensitive temporary data (credentials, never written to disk)

### Kubernetes Fundamentals

#### Translating Docker Compose to Kubernetes

**Our Docker Compose**:
```yaml
services:
  postgres:
    image: postgres:15-alpine
    ports: ["5432:5432"]
    volumes: [postgres_data:/var/lib/postgresql/data]
```

**Equivalent Kubernetes**:
```yaml
# Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc

---
# Service (networking)
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  ports:
  - port: 5432
  selector:
    app: postgres

---
# PersistentVolumeClaim (storage)
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 20Gi
```

**Interview Questions**:
- **Q: What's the difference between Deployment and StatefulSet?**
  - **Deployment**: Stateless apps, pods are interchangeable, random names
  - **StatefulSet**: Stateful apps (databases), stable network identities (postgres-0, postgres-1), ordered deployment
  - **Use StatefulSet for**: Databases, Kafka, ZooKeeper

- **Q: How does Kubernetes service discovery work?**
  - A: Kubernetes DNS creates records for Services. `postgres.default.svc.cluster.local` resolves to Service IP. Pods can use short name `postgres` within same namespace.

- **Q: What's a PersistentVolume vs PersistentVolumeClaim?**
  - **PV**: Actual storage provisioned by admin (like a physical disk)
  - **PVC**: Request for storage by application (like a reservation)
  - **StorageClass**: Dynamic provisioner (auto-creates PVs from cloud provider)

#### Scaling Patterns
```yaml
# Horizontal Pod Autoscaler
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-scaler
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

**Interview Questions**:
- **Q: How would you scale the ingester service?**
  - A: Partition chains across pods (Pod 1: ETH+Polygon, Pod 2: Arbitrum+Base), use Kubernetes Jobs for catch-up mode, scale vertically for real-time ingestion.

### PostgreSQL Advanced Topics

#### Indexing Strategies
```sql
-- B-tree (default) - good for = and range queries
CREATE INDEX idx_blocks_timestamp ON blocks (timestamp);

-- Hash index - only for = queries (faster but limited)
CREATE INDEX idx_tx_hash ON transactions USING HASH (tx_hash);

-- Partial index - smaller, faster for subset queries
CREATE INDEX idx_pending_tx ON transactions (status) 
WHERE status = 'pending';

-- Composite index - order matters!
CREATE INDEX idx_chain_block ON blocks (chain_id, block_number);
-- Good for: WHERE chain_id = 1 AND block_number > 1000
-- Bad for: WHERE block_number > 1000 (chain_id not leading)
```

**Interview Questions**:
- **Q: How do you identify missing indexes?**
  ```sql
  -- Find slow queries
  SELECT query, calls, total_time, mean_time
  FROM pg_stat_statements
  ORDER BY mean_time DESC
  LIMIT 10;
  
  -- Check index usage
  SELECT schemaname, tablename, indexname, idx_scan
  FROM pg_stat_user_indexes
  ORDER BY idx_scan ASC;  -- Unused indexes
  ```

- **Q: What's the trade-off of adding indexes?**
  - **Pros**: Faster reads (can be 1000x speedup)
  - **Cons**: Slower writes (every INSERT/UPDATE must update indexes), more storage (30-50% overhead), maintenance overhead
  - **Rule**: Index columns in WHERE, JOIN, ORDER BY clauses with high selectivity

#### Partitioning Deep Dive
```sql
-- Range partitioning (by time)
CREATE TABLE logs_2025_01 PARTITION OF logs
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- List partitioning (by enum)
CREATE TABLE blocks_ethereum PARTITION OF blocks
FOR VALUES IN (1);

-- Hash partitioning (distribute evenly)
CREATE TABLE users_0 PARTITION OF users
FOR VALUES WITH (MODULUS 4, REMAINDER 0);
```

**Our Choice**: List partitioning by chain_id
```sql
CREATE TABLE blocks_chain_1 PARTITION OF blocks
FOR VALUES IN (1);  -- Ethereum only
```

**Interview Questions**:
- **Q: When should you partition a table?**
  - A: When table > 10GB and queries filter by partition key. Benefits: Faster queries (prune partitions), easier archiving (drop old partitions), parallel maintenance.

- **Q: How does partition pruning work?**
  ```sql
  EXPLAIN SELECT * FROM blocks WHERE chain_id = 1;
  -- Only scans blocks_chain_1, skips other partitions
  ```

### Kafka & Event Streaming

#### Producer Patterns
```go
// Asynchronous with batching (high throughput)
producer.ProduceAsync(&kafka.Message{
    Topic: "raw.blocks",
    Key:   []byte(fmt.Sprintf("%d:%d", chainID, blockNumber)),
    Value: blockJSON,
})

// Synchronous with acknowledgment (guaranteed delivery)
msg, err := producer.ProduceSync(&kafka.Message{
    Topic: "raw.blocks",
    Value: blockJSON,
})
```

**Interview Questions**:
- **Q: How do you ensure exactly-once semantics in Kafka?**
  - A: Enable idempotent producer (retries don't create duplicates) + transactional producer (atomic multi-partition writes) + read_committed consumer isolation level.

- **Q: What's the difference between topics and partitions?**
  - **Topic**: Logical stream (e.g., "raw.blocks")
  - **Partition**: Physical log file (e.g., "raw.blocks-0", "raw.blocks-1")
  - **Ordering guarantee**: Within partition only, not across partitions
  - **Scaling**: Add partitions to parallelize consumers

#### Consumer Group Patterns
```go
// Consumer group ensures each message processed once
consumer := kafka.NewConsumer(&kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092",
    "group.id":          "indexer-processor",
    "auto.offset.reset": "earliest",
})
```

**Interview Questions**:
- **Q: How does Kafka handle consumer failures?**
  - A: Consumer heartbeat mechanism. If consumer stops heartbeat, coordinator triggers rebalance and reassigns partitions to healthy consumers. Offset commits allow resuming from last processed message.

### Redis Patterns

#### Caching Strategies
```go
// Cache-aside (lazy loading)
func GetBlock(chainID, blockNum int64) (*Block, error) {
    // Try cache first
    key := fmt.Sprintf("block:%d:%d", chainID, blockNum)
    if data, err := redis.Get(key).Bytes(); err == nil {
        return unmarshal(data)
    }
    
    // Cache miss - load from DB
    block := db.GetBlock(chainID, blockNum)
    
    // Populate cache with TTL
    redis.Set(key, marshal(block), 10*time.Minute)
    return block, nil
}

// Write-through (always consistent)
func SaveBlock(block *Block) error {
    // Write to DB first
    if err := db.Save(block); err != nil {
        return err
    }
    
    // Then update cache
    key := fmt.Sprintf("block:%d:%d", block.ChainID, block.Number)
    redis.Set(key, marshal(block), 10*time.Minute)
    return nil
}
```

**Interview Questions**:
- **Q: How do you handle cache invalidation?**
  - A: "There are only two hard things in Computer Science: cache invalidation and naming things."
  - **TTL**: Expire after time (good for rarely-changed data)
  - **Event-driven**: Kafka consumer invalidates cache on updates
  - **Version tags**: Increment version number in key

- **Q: What's the thundering herd problem?**
  - A: Many requests hit expired cache key simultaneously, all query DB in parallel. Solution: Lock with Redis SETNX or use probabilistic early expiration.

### React & Next.js Patterns

#### State Management
```typescript
// Context API for global state
const BlockchainContext = createContext<BlockchainState>({
    selectedChain: 'ethereum',
    latestBlock: null,
});

// Custom hooks for business logic
function useBlockStream(chainId: number) {
    const [blocks, setBlocks] = useState<Block[]>([]);
    
    useEffect(() => {
        const ws = new WebSocket(`wss://api/v1/chains/${chainId}/stream`);
        ws.onmessage = (event) => {
            const block = JSON.parse(event.data);
            setBlocks(prev => [block, ...prev].slice(0, 50));
        };
        return () => ws.close();
    }, [chainId]);
    
    return blocks;
}
```

**Interview Questions**:
- **Q: When would you use Context vs Redux vs Zustand?**
  - **Context**: Small apps, simple global state (theme, auth)
  - **Redux**: Large apps, complex state logic, time-travel debugging
  - **Zustand**: Middle ground, less boilerplate than Redux

- **Q: What's the difference between SSR, SSG, and CSR in Next.js?**
  - **SSR (getServerSideProps)**: Render on each request (dynamic data)
  - **SSG (getStaticProps)**: Render at build time (blogs, docs)
  - **CSR (useEffect)**: Render in browser (dashboards, user-specific)
  - **ISR (revalidate)**: SSG with periodic regeneration (best of both)

### Node.js & TypeScript

#### Async Patterns
```typescript
// Promise chaining (readable, sequential)
fetchBlock(1000)
    .then(block => processTransactions(block))
    .then(txs => saveToDB(txs))
    .catch(err => console.error(err));

// Async/await (looks synchronous)
try {
    const block = await fetchBlock(1000);
    const txs = await processTransactions(block);
    await saveToDB(txs);
} catch (err) {
    console.error(err);
}

// Parallel execution
const [block1, block2] = await Promise.all([
    fetchBlock(1000),
    fetchBlock(1001),
]);
```

**Interview Questions**:
- **Q: What's the event loop?**
  - A: Single-threaded execution with callback queue. Async operations (I/O, timers) run in background, callbacks queued for event loop to process when call stack is empty.

- **Q: How do you avoid callback hell?**
  - A: Use Promises or async/await, create named functions, use libraries like async.js for complex flows.

### Go Modules & Dependency Management

Go uses a built-in module system for dependency management, introduced in Go 1.11 and default since Go 1.16.

#### Core Module Files

**1. `go.mod` (Module Definition)**

The main module file that defines your project:
```go
module github.com/0xviggy/blockchain-indexer/services/api

go 1.23.0

require (
    github.com/gin-gonic/gin v1.11.0      // Direct dependency
    github.com/lib/pq v1.10.9             // PostgreSQL driver
)

require (
    github.com/gin-contrib/sse v1.1.0 // indirect  // Transitive dependency
    github.com/mattn/go-isatty v0.0.20 // indirect
)
```

**Components**:
- **Module path**: Unique identifier (usually GitHub URL)
- **Go version**: Minimum Go version required
- **Direct dependencies**: Packages your code imports directly
- **Indirect dependencies**: Dependencies of your dependencies (marked with `// indirect`)

**Advanced directives**:
```go
// Override a dependency (for local development or forks)
replace github.com/old/package => github.com/new/package v1.0.0
replace github.com/local/package => ../local/package

// Prevent specific versions from being used
exclude github.com/problematic/package v1.2.3

// Retract published versions (maintainer says "don't use this")
retract v1.5.0  // Critical bug, use v1.5.1 instead
```

**2. `go.sum` (Cryptographic Checksums)**

Verifies dependency integrity with SHA-256 hashes:
```
github.com/gin-gonic/gin v1.11.0 h1:abc123...
github.com/gin-gonic/gin v1.11.0/go.mod h1:def456...
```

- **First line**: Hash of the module's source code (`.zip` file)
- **Second line**: Hash of the module's `go.mod` file only
- **Purpose**: Detect tampering, ensure reproducible builds across teams
- **Version control**: Always commit both `go.mod` and `go.sum`

**3. Module Cache (`$GOPATH/pkg/mod/`)**

Global cache for downloaded modules:
```
$GOPATH/pkg/mod/
  github.com/
    gin-gonic/
      gin@v1.11.0/
        ...
```

- **Shared across projects**: Download once, use everywhere
- **Read-only**: Prevents accidental modifications
- **Version-specific**: Multiple versions can coexist

#### Essential Go Module Commands

```bash
# Initialize a new module
go mod init github.com/user/repo

# Add missing dependencies and remove unused ones
go mod tidy

# Download dependencies to local cache
go mod download

# Verify checksums match go.sum (detect tampering)
go mod verify

# Copy dependencies to vendor/ directory
go mod vendor

# Update specific dependency to latest
go get -u github.com/package/name

# Update all dependencies to latest minor/patch
go get -u ./...

# Update all dependencies to latest major version
go get -u=patch ./...  # Patch only (safest)
go get -u ./...         # Minor/patch (safe)
go get -u -t ./...      # Include test dependencies

# Show dependency graph
go mod graph

# Explain why a package is needed
go mod why github.com/package/name

# Show available versions of a module
go list -m -versions github.com/gin-gonic/gin

# Downgrade to specific version
go get github.com/package/name@v1.2.3

# Use latest commit on main branch
go get github.com/package/name@main
```

#### Multi-Module Workspaces (`go.work`)

For developing multiple related modules simultaneously:

```go
// go.work (don't commit to version control)
go 1.23

use (
    ./services/api
    ./services/ingester
    ./shared
)
```

**Benefits**:
- Edit multiple modules at once without publishing
- Changes in `./shared` immediately visible to `./services/api`
- Simplifies local development of microservices

**Create workspace**:
```bash
go work init ./services/api ./services/ingester ./shared
```

#### Project Structure with Modules

```
blockchain-indexer/
├── services/
│   ├── api/
│   │   ├── go.mod          # Separate module (COMMIT ✅)
│   │   ├── go.sum          # Checksums (COMMIT ✅)
│   │   ├── main.go         # Source code (COMMIT ✅)
│   │   └── api             # Binary (IGNORE ❌)
│   │
│   └── ingester/
│       ├── go.mod          # Separate module (COMMIT ✅)
│       ├── go.sum          # Checksums (COMMIT ✅)
│       └── main.go         # Source code (COMMIT ✅)
│
├── shared/
│   ├── go.mod              # Shared utilities module (COMMIT ✅)
│   ├── go.sum              # Checksums (COMMIT ✅)
│   └── config/
│       └── config.go       # Shared code (COMMIT ✅)
│
└── go.work                 # Workspace (IGNORE ❌)
```

#### Vendor Directory (Optional)

Copy all dependencies into your repository:
```bash
go mod vendor
```

Creates:
```
vendor/
  modules.txt              # List of vendored modules
  github.com/
    gin-gonic/
      gin/
        ... (full source code)
```

**When to use**:
- **Air-gapped builds**: No internet in production
- **Compliance**: Need to audit all dependency source code
- **Stability**: Don't trust module proxy availability

**Trade-offs**:
- ✅ Self-contained builds
- ✅ Works offline
- ❌ Large repository size (can be hundreds of MB)
- ❌ Harder to review pull requests (lots of vendor/ changes)

**Most projects**: Don't vendor, rely on `go.mod`/`go.sum` + module proxy

#### Semantic Versioning in Go

Go modules follow semantic versioning (semver):
```
v1.2.3
│ │ │
│ │ └─ Patch: Bug fixes (backward compatible)
│ └─── Minor: New features (backward compatible)
└───── Major: Breaking changes
```

**Special rules**:
- **v0.x.x**: Unstable, no compatibility guarantees
- **v1.x.x**: Stable API, must maintain compatibility
- **v2+**: Major version in import path
  ```go
  import "github.com/user/repo/v2"  // v2.x.x
  import "github.com/user/repo/v3"  // v3.x.x
  ```

**Pseudo-versions** (commits without tags):
```
v0.0.0-20191109021931-daa7c04131f5
       └─ timestamp ─┘└─ commit hash ─┘
```

#### Common Dependency Issues & Solutions

**Issue 1: "go.sum has unexpected content"**
```bash
# Solution: Re-download and verify
go clean -modcache
go mod download
go mod verify
```

**Issue 2: Conflicting indirect dependencies**
```
module A requires package X v1.0.0
module B requires package X v2.0.0
```

**Solution**: Go automatically resolves to highest compatible version
- If both are v1.x.x or v2.x.x, uses latest
- If major versions differ, keeps both (import path is different)

**Issue 3: Dependency not found (private repository)**
```bash
# Configure Git authentication
git config --global url."git@github.com:".insteadOf "https://github.com/"

# Or use GOPRIVATE environment variable
export GOPRIVATE=github.com/yourcompany/*
```

**Issue 4: Outdated dependencies**
```bash
# Check for updates
go list -u -m all

# Update safely (minor/patch only)
go get -u=patch ./...

# Review and test before updating to latest
go get -u ./...
go test ./...
```

#### .gitignore Best Practices

```gitignore
# Compiled binaries (don't commit)
services/*/api
services/*/api-*
services/*/ingester
services/*/ingester-*
*.exe
*.test

# Vendor directory (only if using vendoring)
vendor/

# Workspace file (local development only)
go.work
go.work.sum

# Keep module files (CRITICAL - always commit)
# go.mod    ← DO NOT IGNORE
# go.sum    ← DO NOT IGNORE
```

#### Interview Questions

**Q: Why do we need both `go.mod` and `go.sum`?**
- **A**: `go.mod` declares what you want (version constraints), `go.sum` proves what you got (cryptographic verification). Together they ensure reproducible builds across machines and detect supply chain attacks.

**Q: When should you run `go mod tidy`?**
- **A**: 
  - After adding/removing imports in code
  - Before committing (cleans unused dependencies)
  - When `go.mod` and actual imports are out of sync
  - After merging branches (resolve dependency conflicts)

**Q: Direct vs indirect dependencies?**
- **A**: 
  - **Direct**: You `import` them in your code
  - **Indirect**: Your dependencies need them (transitive)
  - Go 1.17+ separates them in `go.mod` for clarity and faster builds

**Q: What's the difference between `go get` and `go install`?**
- **A**:
  - `go get`: Update dependencies in `go.mod` (deprecated for installing tools in Go 1.17+)
  - `go install`: Install executables to `$GOPATH/bin` without modifying `go.mod`
  - **Example**: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

**Q: How do you handle diamond dependency problems?**
- **A**: Go uses Minimal Version Selection (MVS):
  - If A requires X@v1.2 and B requires X@v1.3, Go picks v1.3 (highest requested)
  - Assumes semver: v1.3 is compatible with v1.2
  - If major versions differ (v1 vs v2), both are kept (different import paths)

**Q: What's a module proxy and why use it?**
- **A**: Default: `proxy.golang.org` (Google's public proxy)
  - **Benefits**: Fast, cached, immutable (prevents disappearing dependencies)
  - **Privacy**: Leaks which modules you download to Google
  - **Disable**: `export GOPROXY=direct` (fetch from source repos directly)
  - **Private modules**: `export GOPRIVATE=github.com/yourcompany/*`

**Q: How would you audit dependencies for security vulnerabilities?**
```bash
# Official Go vulnerability checker
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Check dependency licenses
go-licenses report ./... --template=licenses.tpl

# List all dependencies
go list -m all

# Show dependency tree
go mod graph | grep github.com/specific/package
```

---

## Implementation Progress Tracker

> **📋 For comprehensive progress tracking, see [DEVELOPMENT_STATUS.md](./DEVELOPMENT_STATUS.md)**
> 
> This section provides a historical log of implementation phases. For current status, next steps, technical decisions, and prioritized roadmap, refer to the Development Status document.

### ✅ Phase 0: Planning & Architecture (Completed - Nov 14, 2025)
- [x] Language decision: Go vs Rust analysis
- [x] Architecture design (event-driven, microservices)
- [x] Multi-chain strategy
- [x] Tech stack selection
- [x] Documentation structure

**Decision**: **Go** (see [DEVELOPMENT_STATUS.md - Technical Decisions](./DEVELOPMENT_STATUS.md#technical-decisions-log))  
**Key Docs**: See `TECHNICAL_SPEC.md`, `MULTICHAIN_SPEC.md` for detailed specs

### ✅ Phase 1: Infrastructure & Database (Completed - Nov 14, 2025)
- [x] Docker Compose setup with PostgreSQL, Redis, Kafka
- [x] Database schema with multi-chain support
- [x] Partitioned tables for performance
- [x] Database migrations
- [x] Development Makefile with utilities
- [x] Environment configuration template
- [x] Documentation consolidation

**Status**: Ready for service implementation  
**Files Created**: 6 core files  
**Lines of Code**: ~700 (SQL + YAML + Makefile)

---

## Infrastructure Setup & DevOps (Interview Focus)

### Docker Infrastructure Components

Our blockchain indexer uses a containerized infrastructure with four core services:

#### 1. **PostgreSQL Database** (`indexer-postgres`)
- **Image**: `postgres:15-alpine`
- **Port**: 5432
- **Purpose**: Primary data store for blocks, transactions, and analytics
- **Configuration**:
  - Database: `indexer`
  - User: `indexer`
  - Password: `password` (change in production!)
  - UTF-8 encoding with `POSTGRES_INITDB_ARGS: "-E UTF8"`
- **Health Check**: `pg_isready -U indexer` every 10s
- **Volume**: Persistent storage at `postgres_data:/var/lib/postgresql/data`
- **Features**:
  - Partitioned tables by chain_id for horizontal scaling
  - BTREE indexes on frequently queried columns
  - Triggers for automatic timestamp updates
  - Foreign key constraints for data integrity

#### 2. **Redis Cache** (`indexer-redis`)
- **Image**: `redis:7-alpine`
- **Port**: 6379
- **Purpose**: High-speed caching layer for API responses and session data
- **Configuration**:
  - Append-only file (AOF) persistence enabled
  - Command: `redis-server --appendonly yes`
- **Health Check**: `redis-cli ping` every 10s
- **Volume**: `redis_data:/data`
- **Use Cases**:
  - Cache frequently accessed blocks/transactions
  - Rate limiting for API endpoints
  - Temporary storage for WebSocket connections

#### 3. **Kafka/Redpanda** (`indexer-kafka`)
- **Image**: `docker.redpanda.com/redpandadata/redpanda:v23.2.15`
- **Ports**: 9092 (internal), 19092 (external), 18081-18082 (schema registry)
- **Purpose**: Event streaming for real-time data pipelines
- **Why Redpanda**: Kafka-compatible but simpler setup (no ZooKeeper)
- **Configuration**:
  - Internal address: `internal://0.0.0.0:9092`
  - External address: `external://0.0.0.0:19092`
  - Schema registry on 18081
- **Health Check**: Cluster health monitoring
- **Planned Topics**:
  - `raw.blocks` - Raw block data from RPC
  - `parsed.events` - Decoded smart contract events
  - `system.reorg` - Blockchain reorganization events

#### 4. **Kafka UI** (`indexer-kafka-ui`)
- **Image**: `provectuslabs/kafka-ui`
- **Port**: 8080
- **Purpose**: Web interface to monitor Kafka topics, messages, and consumers
- **Access**: `http://localhost:8080`

### Docker Network
- **Name**: `indexer-network`
- **Type**: Bridge network for inter-container communication
- **Purpose**: Isolate indexer services from other Docker containers

### Essential Docker Commands

#### Starting Infrastructure
```bash
# Start all services
make docker-up
# OR
docker compose -f infrastructure/docker/docker-compose.yml up -d

# View logs
make logs
# OR
docker compose -f infrastructure/docker/docker-compose.yml logs -f

# Check service status
make status
# OR
docker ps --filter "name=indexer-"
```

#### Health Checks
```bash
# PostgreSQL
docker exec indexer-postgres pg_isready -U indexer

# Redis
docker exec indexer-redis redis-cli ping

# Kafka cluster health
docker exec indexer-kafka rpk cluster health

# List Kafka topics
docker exec indexer-kafka rpk topic list
```

#### Database Operations
```bash
# Run migrations
make migrate
# Manual migration via Docker
docker exec -i indexer-postgres psql -U indexer -d indexer < database/migrations/001_initial_schema.sql

# Open psql shell
make db-shell
# OR
docker exec -it indexer-postgres psql -U indexer -d indexer

# Check table structure
docker exec indexer-postgres psql -U indexer -d indexer -c "\d chains"

# Query data
docker exec indexer-postgres psql -U indexer -d indexer -c "SELECT * FROM chains;"
```

#### Troubleshooting
```bash
# View container logs
docker logs indexer-postgres -f
docker logs indexer-kafka -f

# Restart a service
docker restart indexer-postgres

# Stop and remove all services
make docker-down
# OR
docker compose -f infrastructure/docker/docker-compose.yml down

# Nuclear option: Remove volumes (DELETES ALL DATA)
docker compose -f infrastructure/docker/docker-compose.yml down -v
```

### Database Schema Setup

#### Migration Strategy
- **Tool**: Raw SQL migrations (no ORM migration framework)
- **Location**: `database/migrations/`
- **Naming**: `001_initial_schema.sql`, `002_add_calldata_parsing.sql`, etc.
- **Execution**: Sequential, idempotent where possible

#### Migration 001: Core Schema
Creates the foundation tables:
- `chains` - Supported blockchain networks
- `blocks` - Block headers (partitioned by chain_id)
- `transactions` - Transaction data (partitioned by chain_id)
- `logs` - Smart contract event logs (partitioned by chain_id)
- `checkpoints` - Service resume points

**Key Design Decisions**:
- **Partitioning by chain_id**: Each chain's data in separate partitions for query performance
- **BIGINT for block numbers**: Supports chains with >2B blocks
- **BYTEA for hashes**: Binary storage more efficient than hex strings
- **Indexes**: Created on hash lookups, timestamp ranges, address filters

#### Migration 002: Advanced Parsing
Adds intelligence layer:
- `protocol_signatures` - Registry of known smart contract functions
- `parsed_calldata` - Decoded function calls with parameters
- `internal_transactions` - Contract-to-contract calls
- `transaction_revert_reasons` - Why transactions failed

### Service Configuration

#### Environment Variables
```bash
# Database
DATABASE_URL="postgres://indexer:password@localhost:5432/indexer?sslmode=disable"

# RPC Endpoints (one per chain)
ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
ETH_WS_URL="wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"  # Optional
POLYGON_RPC_URL="https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY"
ARBITRUM_RPC_URL="https://arb-mainnet.g.alchemy.com/v2/YOUR_KEY"
OPTIMISM_RPC_URL="https://opt-mainnet.g.alchemy.com/v2/YOUR_KEY"
BASE_RPC_URL="https://base-mainnet.g.alchemy.com/v2/YOUR_KEY"

# Ingester Settings
INGESTER_BATCH_SIZE=10
INGESTER_POLL_INTERVAL=2s
```

#### PostgreSQL Client Setup (macOS)
```bash
# Install PostgreSQL client tools
brew install postgresql@15

# Add to PATH (required for psql command)
export PATH="/opt/homebrew/opt/postgresql@15/bin:$PATH"

# Add to shell profile for persistence
echo 'export PATH="/opt/homebrew/opt/postgresql@15/bin:$PATH"' >> ~/.zshrc
```

### Common Issues & Solutions

#### Issue: Database Authentication Failed
**Error**: `pq: password authentication failed for user "indexer"`

**Cause**: Service using wrong credentials (common during development)

**Solution**:
1. Check Docker Compose credentials: `indexer:password@localhost:5432/indexer`
2. Update service code to match
3. Rebuild service: `go build -o service main.go`
4. If persists, restart PostgreSQL: `docker restart indexer-postgres`

#### Issue: Column Name Mismatch
**Error**: `pq: column "name" does not exist`

**Cause**: Database uses `chain_name` but code queries `name`

**Solution**: Update all SQL queries to use correct column names:
```sql
-- Wrong
SELECT name FROM chains;

-- Correct
SELECT chain_name FROM chains;
```

#### Issue: Go Build Cache Confusion
**Error**: Code changes not reflected in running service

**Solution**:
```bash
# Clear Go build cache
go clean -cache

# Kill old process
pkill -9 -f "service/main.go"

# Rebuild and run
go build -o service main.go
./service
```

#### Issue: Port Already in Use
**Error**: `bind: address already in use`

**Solution**:
```bash
# Find process on port (e.g., 8000)
lsof -ti:8000

# Kill it
lsof -ti:8000 | xargs kill -9

# Or use pkill
pkill -9 -f "api/main.go"
```

### Makefile Automation

Our `Makefile` provides shortcuts for common operations:

```makefile
# Complete setup from scratch
make setup          # = docker-up + wait + migrate

# Infrastructure
make docker-up      # Start all containers
make docker-down    # Stop all containers
make logs           # Tail all container logs
make status         # Check running services

# Database
make migrate        # Run all migrations
make db-shell       # Open psql interactive shell
make db-reset       # Drop and recreate schema (DESTRUCTIVE)

# Services
make run-ingester   # Start ingester service
make run-api        # Start API service

# Development
make test           # Run test suite
make clean          # Stop services and remove volumes
```

### Interview-Ready Talking Points

**Q: How do you ensure database schema changes don't break production?**
- Sequential migrations with version numbers
- Test migrations locally first
- Use transactions for atomic schema changes
- Maintain backward compatibility (add columns with defaults)
- Blue-green deployment: Run old and new code side-by-side

**Q: Why partition tables by chain_id?**
- **Query performance**: Most queries filter by chain (ETH only, Polygon only)
- **Parallel writes**: Each chain writes to its own partition (no lock contention)
- **Maintenance**: Can drop old chain data without affecting others
- **Future scaling**: Easy to move partitions to different databases

**Q: Why use Docker Compose instead of Kubernetes for development?**
- **Simplicity**: Fast local setup (2 minutes vs 20+ minutes for k8s)
- **Resource efficiency**: Docker Compose uses ~500MB RAM, k8s uses 2-4GB
- **Iteration speed**: Change docker-compose.yml and restart in seconds
- **Production**: We'd use Kubernetes, but for development Docker Compose is ideal
- **Cost**: Free locally, matches CI/CD pipeline tools

**Q: How do you handle database connection pooling?**
```go
db.SetMaxOpenConns(25)        // Limit total connections
db.SetMaxIdleConns(10)         // Keep connections warm
db.SetConnMaxLifetime(5 * time.Minute)  // Recycle old connections
```
- **Reasoning**: 25 is safe for RDS t3.medium (100 max connections)
- **Monitoring**: Track connection metrics via Prometheus
- **Tuning**: Adjust based on service concurrency and query latency

**Q: What's your backup and recovery strategy?**
- **PostgreSQL**: Automated daily snapshots via AWS RDS
- **Point-in-time recovery**: RDS transaction logs (5-35 day retention)
- **Disaster recovery**: Cross-region replicas for critical chains
- **Testing**: Monthly restore drills to validate backups
- **Data loss tolerance**: Blocks can be re-indexed from blockchain (source of truth)

---

### ✅ Phase 2: Ingester Service - MVP (Completed - Nov 15, 2025)
- [x] Go module initialization
- [x] Multi-chain RPC client setup (Ethereum, Polygon, Arbitrum, Optimism, Base)
- [x] WebSocket block subscription with HTTP polling fallback
- [x] Checkpoint management (resume from last processed block)
- [x] Graceful shutdown handling
- [x] Direct PostgreSQL writes (atomic per-block transactions)
- [x] Catch-up mode for missed blocks
- [x] Parallel chain processing (one goroutine per chain)

**What we built**: Production-ready multi-chain ingester that:
- Connects to all configured EVM chains simultaneously
- Uses WebSocket for real-time blocks (falls back to polling if unavailable)
- Stores blocks and transactions in partitioned PostgreSQL tables
- Maintains checkpoints for reliable resumption
- Handles shutdown gracefully without data loss

**Files Created**: 
- `services/ingester/main.go` (588 lines)
- `services/ingester/go.mod`
- `services/ingester/README.md` (comprehensive operations guide)

**Phase 2.1 - Advanced Features (Deferred)**:
- [ ] Transaction receipts (status, gas_used, logs)
- [ ] Reorg detection and handling
- [ ] Kafka message production
- [ ] Retry logic and error recovery
- [ ] Prometheus metrics

**Decision**: Skip advanced features for MVP to reach E2E demo faster. Will add after UI is working.

### ✅ Phase 2.1: Advanced Message Parsing (Completed - Nov 14, 2025)
- [x] Database schema for calldata parsing
- [x] Internal transactions tracking table
- [x] Revert reasons extraction table
- [x] Protocol signatures registry (Uniswap, LayerZero, 1inch, etc.)
- [x] Analytics views for transaction insights
- [x] RPC validation against real blockchain data

**What we built**: Extended the database to capture not just events, but also:
- **Parsed calldata**: Decode function calls to understand user intent (swaps, bridges, etc.)
- **Internal transactions**: Track contract-to-contract calls within transactions
- **Revert reasons**: Capture why transactions fail
- **Protocol registry**: Pre-populated with 20+ common protocols

**Files Created**: `database/migrations/002_add_calldata_parsing.sql` (330+ lines)

#### Phase 2.1 Findings: RPC Validation ✅ COMPLETED

**Execution Date**: November 15, 2025
**Block Analyzed**: Ethereum Mainnet Block 18,500,000 (157 transactions)

**Key Discoveries**:
1. ✅ All schema fields match actual blockchain data perfectly
2. ✅ Data types (BIGINT, BYTEA, VARCHAR) handle real-world sizes correctly
3. ❌ **Critical Gap Found**: Uniswap Universal Router (0x3593564c) appeared in 2 out of 3 transactions but was missing from our protocol signature registry
4. ✅ ERC20 Transfer event signature (0xddf252ad...) correctly recognized
5. ⚠️ Internal transaction tracing requires archive node (expected limitation)

**Added Missing Signatures**:
- `0x3593564c` - execute (Uniswap Universal Router) - **EXTREMELY HIGH VOLUME**
- `0x24856bc3` - execute (Uniswap Universal Router - no deadline variant)
- `0xa08edebc` - swap (Metamask Swap Router)
- `0x13d79a0b` - settle (CoW Protocol)
- `0x30f28b7a` - permit (Permit2 token approvals)

**Conclusion**: Database schemas are production-ready. Protocol signature coverage now includes the most common transaction types.

---

#### Phase 2.2 Findings: Multi-Block Signature Analysis ✅ COMPLETED

**Execution Date**: November 15, 2025
**Blocks Analyzed**: Ethereum Mainnet Latest 10 blocks (3 successful, 616 transactions)

**Statistics**:
- Total transactions scanned: **616**
- Transactions with calldata: **439 (71.3%)**
- Unique signatures found: **119**
- Unknown signatures: **110 (92% of unique signatures!)**

**Top High-Volume Unknown Signatures Discovered**:
1. `0x78e111f6` - executeFFsYo (Meta-transaction Forwarder) - **43 calls** (2nd most common!)
2. `0x122067ed` - Unknown Aggregator Swap - **17 calls**
3. `0x88ffe867` - pledge (Staking Protocol) - **12 calls**
4. `0x6fadcf72` - forward (Generic Forwarder) - **7 calls**
5. `0x791ac947` - swapExactTokensForETHSupportingFeeOnTransferTokens (Uniswap V2) - **6 calls**
6. `0x3d0e3ec5` - swapExactTokensForETHSupportingFeeOnTransferTokens (Custom DEX) - **6 calls**

**Key Insights**:
- **Meta-transaction forwarders** are surprisingly common (50+ calls total)
- ERC20 `transfer` (0xa9059cbb) remains #1 at **152 calls** (24.7% of all transactions)
- Many protocols use custom variants of standard DEX functions
- 4byte.directory doesn't have entries for ~20% of discovered signatures (very new or proprietary protocols)

**Protocol Signature Coverage**: Now **26 protocols** covering ~8% of unique signatures but likely 40-50% of transaction volume based on frequency distribution.

**Recommendation**: 
- Continue iterative scanning as we encounter new chains
- Focus on high-volume unknowns (10+ calls) for database inclusion
- Low-volume signatures (1-2 calls) are likely proprietary contracts and can be parsed generically

---

### 🔄 How to Recreate Multi-Block Signature Analysis

This section documents the complete workflow to run signature analysis on any EVM chain.

#### Prerequisites
1. **RPC URL** for the target chain (archive node NOT required for signature analysis)
2. **Chain configured** in `scripts/explore_rpc.go` `SupportedChains` array
3. **Environment variable** set (e.g., `ETH_RPC_URL`, `POLYGON_RPC_URL`, etc.)

#### Step 1: Configure Target Chain(s)

Edit `scripts/explore_rpc.go` to add your chain if not already present:

```go
var SupportedChains = []ChainConfig{
    {
        ChainID: 1,
        Name:    "Ethereum",
        EnvVar:  "ETH_RPC_URL",
        ExampleRPCURL: os.Getenv("ETH_RPC_URL"),
    },
    {
        ChainID: 137,
        Name:    "Polygon",
        EnvVar:  "POLYGON_RPC_URL",
        ExampleRPCURL: os.Getenv("POLYGON_RPC_URL"),
    },
    // Add more chains...
}
```

#### Step 2: Set Environment Variables

Create or update `.env.local`:

```bash
# For Ethereum
export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"

# For Polygon
export POLYGON_RPC_URL="https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY"

# For Arbitrum
export ARBITRUM_RPC_URL="https://arb-mainnet.g.alchemy.com/v2/YOUR_KEY"

# ... etc
```

Then load them:
```bash
source .env.local
```

#### Step 3: Run the Analysis

```bash
make explore-rpc
```

This will:
1. Connect to all configured chains (with valid RPC URLs)
2. Fetch the latest 10 blocks from each chain
3. Extract all function signatures from transaction calldata
4. Compare against known signature registry
5. Report statistics and unknown signatures

#### Step 4: Analyze Output

The script outputs:
- **Statistics**: Total transactions, calldata coverage, unique signatures
- **Top 10 signatures**: By call frequency with known/unknown status
- **Unknown signatures list**: With example transaction hashes and 4byte.directory lookup URLs

Example output:
```
📈 Statistics:
  Total transactions: 616
  With calldata: 439 (71.3%)
  Unique signatures: 119
  Unknown signatures: 110

🔥 Top 10 Most Common Signatures:
  1. 0xa9059cbb - 152 calls ✅
      transfer (ERC20)
  2. 0x78e111f6 - 43 calls ❌ UNKNOWN
  ...
```

#### Step 5: Research Unknown Signatures

For each high-volume unknown (>10 calls):

1. **Click the 4byte.directory link** provided in output:
   ```
   https://www.4byte.directory/signatures/?bytes4_signature=0x78e111f6
   ```

2. **Look up the transaction** on block explorer (Etherscan, etc.):
   ```
   Example tx: 0xabc123...
   ```

3. **Identify the protocol**: Check the `to` address and contract name

4. **Find the ABI**: 
   - Check Etherscan's contract page
   - Search GitHub for the protocol
   - Use Sourcify or other ABI databases

#### Step 6: Update Database Migration

Add discovered signatures to `database/migrations/002_add_calldata_parsing.sql`:

```sql
INSERT INTO protocol_signatures (signature, function_name, protocol, abi, description) VALUES
    ('0x78e111f6', 'executeFFsYo', 'forwarder',
     'executeFFsYo(address,bytes)',
     'Meta-transaction forwarder (43 calls in sample)');
```

#### Step 7: Update Exploration Script

Add to `scripts/explore_rpc.go` in both functions:

**In `isKnownSignature()`:**
```go
signatures := map[string]bool{
    // ... existing
    "0x78e111f6": true, // New signature
}
```

**In `getSignatureName()`:**
```go
signatures := map[string]string{
    // ... existing
    "0x78e111f6": "executeFFsYo (Meta-tx Forwarder)",
}
```

#### Step 8: Commit and Document

```bash
git add -A
git commit -m "feat: Add [PROTOCOL_NAME] signatures from multi-block analysis

- Discovered X new signatures across Y blocks
- Added high-volume signatures: 0xXXXXXXXX (N calls)
- Updated protocol registry with ABIs"

git push origin main
```

Update this learning guide with findings in Phase 2.X sections.

---

#### Multi-Chain Analysis Best Practices

**Sampling Strategy**:
- **Latest 10 blocks**: Good for current protocol landscape (~500-1000 transactions)
- **Random historical blocks**: Sample different time periods to catch older protocols
- **High-activity periods**: Sample during DeFi summer, NFT mints, etc.

**Chain-Specific Considerations**:
- **Ethereum**: Highest protocol diversity, sample more blocks (10-20)
- **L2s (Arbitrum, Optimism)**: Similar protocols to Ethereum but faster blocks, sample 20-50
- **Polygon**: More gaming/NFT activity, different protocol mix
- **BSC**: More trading bots, different DEX landscape

**Volume Thresholds**:
- **>50 calls**: Critical - Add immediately with full ABI
- **10-50 calls**: High priority - Research and add
- **5-10 calls**: Medium priority - Add if easily identifiable
- **1-5 calls**: Low priority - Track but don't spend time researching (likely proprietary)

**Iteration Frequency**:
- **Weekly**: Run analysis on new chains before launch
- **Monthly**: Re-scan existing chains for emerging protocols
- **On-demand**: When seeing high volumes of decode failures in production

---

### 📋 Phase 3: Processor Service (Planned)
- [ ] Kafka consumer setup
- [ ] Event parsing (ERC20, ERC721)
- [ ] Calldata parsing implementation
- [ ] Internal transaction extraction
- [ ] Revert reason extraction
- [ ] ABI decoding
- [ ] Database writer with batching
- [ ] Error handling & retries

### 📋 Phase 4: API Service (Planned)
- [ ] REST API with Gin/Echo
- [ ] WebSocket for real-time updates
- [ ] Redis caching layer
- [ ] Rate limiting
- [ ] API documentation (Swagger)

### 📋 Phase 5: Observability (Planned)
- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Distributed tracing (Jaeger)
- [ ] Log aggregation

---

## Table of Contents

### Core Sections
1. [Implementation Progress Tracker](#implementation-progress-tracker)
   - [Phase 0: Planning & Architecture](#-phase-0-planning--architecture-completed---nov-14-2025)
   - [Phase 1: Infrastructure & Database](#-phase-1-infrastructure--database-completed---nov-14-2025)
   - [Phase 2: Ingester Service - MVP](#-phase-2-ingester-service---mvp-completed---nov-15-2025)
   - [Phase 2.1: Advanced Message Parsing](#-phase-21-advanced-message-parsing-completed---nov-14-2025)
   - [Phase 2.1 Findings: RPC Validation](#phase-21-findings-rpc-validation--completed)
   - [Phase 2.2 Findings: Multi-Block Signature Analysis](#phase-22-findings-multi-block-signature-analysis--completed)
   - [How to Recreate Multi-Block Signature Analysis](#-how-to-recreate-multi-block-signature-analysis)
   - [Phase 3: Processor Service (Planned)](#-phase-3-processor-service-planned)
   - [Phase 4: API Service (Planned)](#-phase-4-api-service-planned)
   - [Phase 5: Observability (Planned)](#-phase-5-observability-planned)

2. [Prerequisites & Quick Start](#prerequisites--quick-start)
   - [Install Requirements (macOS)](#install-requirements-macos)
   - [Understanding Go Installation](#understanding-go-installation)
   - [Start the System](#start-the-system)

3. [System Architecture](#system-architecture)

4. [Setup Steps & Commands](#setup-steps--commands)
   - [Phase 1: Infrastructure Setup](#phase-1-infrastructure-setup-completed-)
   - [Phase 2.1: Calldata Parsing Implementation](#phase-21-calldata-parsing-implementation-completed-)
   - [Phase 2: Ingester Service](#phase-2-ingester-service-next)
   - [Docker Infrastructure](#docker-infrastructure-legacy-instructions)
   - [Database Setup](#database-setup-legacy-instructions)
   - [Running Services](#running-services)

5. [Key Technical Concepts](#key-technical-concepts)
   - [1. Message Parsing Overview](#1-message-parsing-overview)
     - [Internal Transactions](#internal-transactions)
     - [Calldata Parsing](#calldata-parsing)
     - [Revert Reason Extraction](#revert-reason-extraction)
   - [2. Blockchain Reorg Handling](#2-blockchain-reorg-handling)
   - [3. Event Parsing (ERC20 Transfer Example)](#3-event-parsing-erc20-transfer-example)
   - [4. Database Partitioning Strategy](#4-database-partitioning-strategy)
   - [5. Rate Limiting with Token Bucket](#5-rate-limiting-with-token-bucket)
   - [6. Kafka Message Ordering](#6-kafka-message-ordering)

6. [Design Decisions & Trade-offs](#design-decisions--trade-offs)
   - [1. Language Choice: Go vs Rust](#1-language-choice-go-vs-rust-)
   - [2. Message Broker: Kafka vs RabbitMQ](#2-message-broker-kafka-vs-rabbitmq)
   - [3. Database: PostgreSQL vs TimescaleDB vs Cassandra](#3-database-postgresql-vs-timescaledb-vs-cassandra)
   - [4. Monorepo vs Separate Repos](#4-monorepo-vs-separate-repos)

7. [Common Interview Questions](#common-interview-questions)
   - [Q1: How do you handle blockchain reorganizations?](#q1-how-do-you-handle-blockchain-reorganizations)
   - [Q2: How do you scale the ingester for multiple chains?](#q2-how-do-you-scale-the-ingester-for-multiple-chains)
   - [Q3: How do you ensure data consistency during high load?](#q3-how-do-you-ensure-data-consistency-during-high-load)
   - [Q4: How would you optimize query performance for address transaction history?](#q4-how-would-you-optimize-query-performance-for-address-transaction-history)
   - [Q5: How do you handle API rate limiting at scale?](#q5-how-do-you-handle-api-rate-limiting-at-scale)

8. [Go Programming Concepts - Interview Guide](#go-programming-concepts---interview-guide)
   - [1. Goroutines](#1-goroutines--used)
   - [2. Context Propagation](#2-context-propagation--used-extensively)
   - [3. Channels](#3-channels--used)
   - [4. sync Package](#4-sync-package--used)
   - [5. HTTP Serialization/Deserialization](#5-http-serializationdeserialization--used---api-service)
   - [6. io Package (Reader, Writer, Stream Processing)](#6-io-package-reader-writer-stream-processing--limited-use)
   - [7. Generics (Type Parameters)](#7-generics-type-parameters--not-used)
   - [8. Testing with testify/require](#8-testing-with-testifyrequire--not-implemented)
   - [9. Testcontainers](#9-testcontainers--not-implemented)
   - [10. Google Go Style Guide (Idiomatic Go)](#10-google-go-style-guide-idiomatic-go--mostly-followed)

9. [Go Concepts Summary Table](#go-concepts-summary-table)

10. [Troubleshooting Guide](#troubleshooting-guide)
    - [Issue: Ingester falling behind chain head](#issue-ingester-falling-behind-chain-head)
    - [Issue: Processor consumer lag](#issue-processor-consumer-lag)
    - [Issue: API slow response times](#issue-api-slow-response-times)

11. [Performance Optimization](#performance-optimization)
    - [Database Optimizations](#database-optimizations)
    - [Go Performance Tips](#go-performance-tips)
    - [Kafka Optimizations](#kafka-optimizations)

12. [Frontend Development for Blockchain Engineers](#frontend-development-for-blockchain-engineers)
    - [Overview: Frontend Landscape for DeFi/Web3](#overview-frontend-landscape-for-defiweb3)
    - [Framework Comparison: React vs Next.js vs Vue](#framework-comparison-react-vs-nextjs-vs-vue)
    - [Styling: Tailwind CSS vs Styled Components vs CSS Modules](#styling-tailwind-css-vs-styled-components-vs-css-modules)
    - [State Management Comparison](#state-management-comparison)

13. [Cross-Stack Learning: Transferable Skills & Concepts](#cross-stack-learning-transferable-skills--concepts)

14. [Production Readiness](#production-readiness)

15. [Additional Resources](#additional-resources)

16. [Updates Log](#updates-log)

---

## Prerequisites & Quick Start

### Install Requirements (macOS)
```bash
# Install Docker Desktop from https://docker.com or:
brew install --cask docker
# Open Docker Desktop, wait for whale icon in menu bar

# Install Go (if needed)
brew install go

# Verify
docker --version  # Should be 20.10+
go version        # Should be 1.21+
```

#### Understanding Go Installation

**Global Installation**:
- Go binary: `/opt/homebrew/bin/go` (installed globally via Homebrew)
- Go toolchain: Available system-wide for all projects
- Module cache: `~/go/pkg/mod` (shared global cache for all downloaded modules)

**Per-Project Dependencies**:
- Each service has its own `go.mod` file defining dependencies and versions
- When you run `go mod download`, modules are cached globally but tracked locally
- This is the standard Go approach: global toolchain + global cache, but project-specific dependency tracking

**Why this matters**:
- You install Go once for the entire system
- Module downloads are cached and reused across projects (saves bandwidth/time)
- Each project independently controls its dependency versions via `go.mod`
- No need for per-project Go installations (unlike Python virtual environments)

### Start the System
```bash
# 1. Start infrastructure
make docker-up

# 2. Run migrations
make migrate

# 3. Verify
make status
# Web UIs: Kafka (http://localhost:8080), pgAdmin (http://localhost:5050)
```

**Troubleshooting**: Docker not running? Open Docker Desktop. Port conflict? Stop services on 5432/6379/19092.

---

## System Architecture

```
Blockchain → Ingester → Kafka → Processor → PostgreSQL → API → Users
                                              ↓
                                           Redis Cache
```

**Why event-driven?** Decouples services, enables independent scaling, provides replay capability for reorgs.  
**Full details**: See [Technical Spec](./TECHNICAL_SPEC.md)

---

## Setup Steps & Commands

### Phase 1: Infrastructure Setup (COMPLETED ✅)
**Date**: 2025-11-14  
**What we built**: Docker Compose setup with PostgreSQL, Redis, and Kafka

#### Files Created:
1. **`infrastructure/docker/docker-compose.yml`**
   - PostgreSQL 15 (port 5432)
   - Redis 7 (port 6379)
   - Redpanda/Kafka (ports 19092, 18081, 18082)
   - Kafka UI (port 8080) - for debugging
   - pgAdmin (port 5050) - for DB management

2. **`infrastructure/docker/init-db.sh`**
   - PostgreSQL initialization script
   - Enables UUID and pg_stat_statements extensions
   - Runs automatically on first container start

3. **`database/migrations/001_initial_schema.sql`**
   - Complete schema with partitioning
   - 5 tables: chains, blocks, transactions, events, checkpoints
   - Partitioned by chain_id for multi-chain support
   - Indexes for performance
   - Triggers for auto-updating timestamps
   - Views for monitoring (latest_blocks, indexing_status)

4. **`.env.example`**
   - Template for environment variables
   - RPC URLs for all chains (Ethereum, Polygon, Arbitrum, Optimism, Base)
   - Database and Redis configuration

5. **Updated `Makefile`**
   - `make setup` - One command to start everything
   - `make docker-up` - Start infrastructure
   - `make migrate` - Run database migrations
   - `make db-shell` - Open PostgreSQL CLI
   - `make status` - Check service health
   - Many more utility commands

#### Quick Start Commands:
```bash
# 1. Start all infrastructure (PostgreSQL, Redis, Kafka)
make docker-up

# 2. Wait for services to be ready (check logs)
make status

# 3. Run database migrations
make migrate

# 4. Verify tables were created
make db-shell
# Then in psql: \dt

# 5. View indexing status
# In psql:
SELECT * FROM indexing_status;
SELECT * FROM chains;
```

#### Key Learning Points:

**Why Redpanda instead of Apache Kafka?**
- Redpanda is Kafka-compatible but much simpler to run locally
- No need for Zookeeper
- Single binary, lower resource usage
- Same Kafka API, so we can switch to real Kafka in production

**Why Partitioned Tables?**
- Each chain gets its own partition (blocks_eth, blocks_polygon, etc.)
- Queries filtered by chain_id only scan relevant partitions
- Easier to manage and archive old data per chain
- Better query performance (partition pruning)

**Database Schema Highlights:**
```sql
-- Multi-chain support with chain metadata
chains (chain_id, chain_name, rpc_url, enabled, last_indexed_block)

-- Blocks partitioned by chain
blocks (chain_id, block_number, block_hash, parent_hash, timestamp, ...)
  ├── blocks_eth (chain_id=1)
  ├── blocks_polygon (chain_id=137)
  └── ... other chains

-- Transactions partitioned by chain
transactions (chain_id, tx_hash, block_number, from_address, to_address, ...)

-- Events/Logs partitioned by chain
events (chain_id, tx_hash, contract_address, event_signature, decoded_data)

-- Checkpoint tracking for each service per chain
checkpoints (service_name, chain_id, last_processed_block)

-- Reorg tracking
reorg_events (chain_id, rollback_from_block, rollback_to_block, handled)
```

**Indexes Strategy:**
- Hash lookups: `idx_blocks_eth_hash` for fast block lookup by hash
- Time-series queries: `idx_blocks_eth_timestamp DESC` for recent blocks
- Address queries: `idx_tx_eth_from`, `idx_tx_eth_to` for wallet activity
- Event queries: `idx_events_eth_contract`, `idx_events_eth_signature`
- JSONB queries: GIN index on `decoded_data` for flexible event queries

#### Phase 1 Summary - What We Accomplished:
- ✅ **6 files created** (~700 lines of code)
- ✅ **5 Docker services** orchestrated

---

### Phase 2.1: Calldata Parsing Implementation (COMPLETED ✅)
**Date**: 2025-11-14  
**What we built**: Advanced message parsing capabilities

#### Files Created:
**`database/migrations/002_add_calldata_parsing.sql`** (300+ lines)

This migration adds four new tables:

1. **`parsed_calldata`** - Decoded function calls
   ```sql
   - chain_id, tx_hash
   - function_signature (4-byte selector, e.g., 0x38ed1739)
   - function_name (e.g., "swapExactTokensForTokens")
   - protocol (e.g., "uniswap-v2", "layerzero", "1inch")
   - decoded_params (JSONB with function arguments)
   ```

2. **`internal_transactions`** - Contract-to-contract calls
   ```sql
   - chain_id, tx_hash, internal_tx_index
   - call_type ('call', 'delegatecall', 'staticcall', 'create', 'create2')
   - from_address, to_address, value
   - input, output, success
   ```

3. **`revert_reasons`** - Failed transaction error messages
   ```sql
   - chain_id, tx_hash
   - revert_reason (string error message)
   - error_signature, error_name (for custom errors)
   - error_params (JSONB for structured errors)
   ```

4. **`protocol_signatures`** - Function signature registry
   ```sql
   - signature (4-byte selector)
   - function_name, protocol
   - abi (full function signature for decoding)
   - description
   ```

#### Pre-populated Protocol Signatures:

**DEX Protocols:**
- Uniswap V2: `swapExactTokensForTokens`, `swapExactETHForTokens`, `swapExactTokensForETH`
- Uniswap V3: `exactInputSingle`, `exactInput`
- Curve: `exchange`
- 1inch: `swap`

**Bridge Protocols:**
- LayerZero: `send`
- Across: `deposit`
- Stargate: `swap`

**NFT Marketplaces:**
- OpenSea Seaport: `fulfillOrder`

**DeFi Lending:**
- Aave V3: `supply`, `withdraw`

#### Example Use Cases:

**1. Track DEX Swap Activity:**
```sql
-- Find all Uniswap swaps
SELECT 
    t.tx_hash,
    t.from_address,
    pc.function_name,
    pc.decoded_params->>'amountIn' as amount_in,
    pc.decoded_params->>'path' as swap_path
FROM transactions t
JOIN parsed_calldata pc ON t.tx_hash = pc.tx_hash
WHERE pc.protocol = 'uniswap-v2'
  AND pc.function_name = 'swapExactTokensForTokens';
```

**2. Analyze Bridge Usage:**
```sql
-- Cross-chain activity via LayerZero
SELECT 
    chain_id,
    COUNT(*) as bridge_count,
    decoded_params->>'dstChainId' as destination_chain
FROM parsed_calldata
WHERE protocol = 'layerzero'
GROUP BY chain_id, decoded_params->>'dstChainId';
```

**3. Find Common Failure Reasons:**
```sql
-- Most common revert reasons
SELECT 
    revert_reason,
    COUNT(*) as failure_count
FROM revert_reasons
WHERE chain_id = 1
GROUP BY revert_reason
ORDER BY failure_count DESC
LIMIT 10;
```

**4. Track Internal Value Transfers:**
```sql
-- See where ETH flows within a complex transaction
SELECT 
    it.internal_tx_index,
    it.from_address,
    it.to_address,
    it.value,
    it.call_type
FROM internal_transactions it
WHERE it.tx_hash = '0x123...'
  AND it.value > 0
ORDER BY it.internal_tx_index;
```

#### New Analytics Views:

**`transaction_analytics`**: Enriched transaction data
```sql
SELECT * FROM transaction_analytics 
WHERE protocol = 'uniswap-v3'
  AND status = FALSE;  -- Failed Uniswap swaps with revert reasons
```

**`protocol_usage_stats`**: Protocol popularity metrics
```sql
SELECT * FROM protocol_usage_stats
ORDER BY call_count DESC;  -- Most used protocols
```

#### Why This Matters for Interviews:

**System Design Question**: "How would you track DEX trading activity across multiple chains?"

**Answer**: 
"We parse transaction calldata to extract swap details:
1. Extract 4-byte function signature from transaction input
2. Look up protocol in our signature registry
3. Decode parameters using the stored ABI
4. Store in `parsed_calldata` with structured JSONB
5. Query aggregated stats with views like `protocol_usage_stats`

This enables analytics like:
- Daily volume per DEX per chain
- Most popular trading pairs
- Swap success/failure rates
- Cross-chain DEX usage patterns

We also track internal transactions to capture the full flow of funds through DEX routers."

#### Implementation Status:
- ✅ Database schema completed
- ✅ 12+ protocols pre-configured
- ✅ Partitioned for multi-chain support
- ✅ Indexes optimized for common queries
- ✅ Analytics views created
- ⏳ Go parser implementation (Phase 3)
- ⏳ RPC trace extraction (Phase 3)
- ✅ **10 database tables** with 35 partitions (6 in initial schema + 4 in calldata parsing)
- ✅ **2 analytics views** for transaction and protocol insights
- ✅ **20+ indexes** for performance
- ✅ **20+ Makefile commands** for development
- ✅ **Multi-chain support** for 5 blockchains
- ✅ **Complete schema** with reorg handling
- ✅ **Web UIs** for debugging (Kafka UI, pgAdmin)

**Time to setup**: 2 minutes (after Docker installed)  
**Next**: Build Ingester service to start fetching blocks

#### Troubleshooting:

```bash
# Check if containers are running
make status

# View logs from all containers
make logs

# View specific container logs
docker logs indexer-postgres
docker logs indexer-kafka

# Restart everything
make docker-down && make docker-up

# Reset database (deletes all data!)
make db-reset

# Check PostgreSQL connection
make db-shell

# Check Redis connection
make redis-cli
# In redis-cli: PING (should return PONG)

# Check Kafka topics
make kafka-topics
```

#### Web UIs for Debugging:
- **Kafka UI**: http://localhost:8080 - View topics, messages, consumer groups
- **pgAdmin**: http://localhost:5050 - Database GUI (login: admin@indexer.local / admin)

---

### Phase 2: Ingester Service (NEXT)
**Status**: Not started  
**Goal**: Fetch blocks from blockchain RPC and publish to Kafka

#### What we'll build:
1. RPC client with connection pooling
2. WebSocket subscription for real-time blocks
3. Reorg detection logic
4. Checkpoint management
5. Kafka producer

---

### Docker Infrastructure (Legacy Instructions)
```bash
# We now use 'make docker-up' instead
# But if you want to run docker-compose directly:
docker-compose -f infrastructure/docker/docker-compose.yml up -d

# Verify services are running
docker ps --filter "name=indexer-"

# Check logs
docker logs indexer-postgres
docker logs indexer-kafka
```

### Database Setup (Legacy Instructions)
```bash
# We now use 'make migrate' instead
# But if you want to run manually:
PGPASSWORD=password psql -h localhost -U indexer -d indexer -f database/migrations/001_initial_schema.sql

# Verify tables
PGPASSWORD=password psql -h localhost -U indexer -d indexer -c "\dt"
```

### Running Services
```bash
# Terminal 1: Ingester
make run-ingester

# Terminal 2: Processor
make run-processor

# Terminal 3: API
make run-api

# Or run all with Docker
make docker-up
```

---

## Key Technical Concepts

### 1. Message Parsing Overview

**What types of onchain messages exist?**

```
Raw Transaction
    ├─> Standard Fields (to, from, value) ✅ Already tracked
    ├─> Logs/Events (ERC20, ERC721) ✅ Already parsed
    ├─> Calldata Parsing ⭐ NEW (Phase 2.1)
    │   ├─> Function signature detection
    │   ├─> ABI decoding
    │   └─> Protocol-specific parsers
    ├─> Internal Transactions ⭐ NEW (Phase 2.1)
    │   ├─> Contract calls
    │   ├─> Contract creations
    │   └─> Value transfers
    └─> Revert Reasons ⭐ NEW (Phase 2.1)
        ├─> Error extraction
        └─> Error classification
```

#### **Internal Transactions**
Internal transactions are contract-to-contract calls that happen within a transaction.

**Example Flow:**
```
User calls Uniswap Router
  → Router calls WETH contract (internal tx)
  → Router calls USDC contract (internal tx)
  → Router calls Pool contract (internal tx)
```

**Why track them:**
- See full flow of funds through contracts
- Track contract creation by other contracts
- Understand complex DeFi interactions
- Essential for accurate balance tracking

**How to extract:**
```go
// Use debug_traceTransaction or trace_transaction RPC methods
traces, err := client.TraceTransaction(ctx, txHash)
// Returns all internal calls with: from, to, value, gas, callType
```

#### **Calldata Parsing**
Calldata is the encoded function call sent to a contract. Parsing reveals what function was called and with what parameters.

**Common Protocol Examples:**

**Uniswap V2 Swap:**
```solidity
// Function: swapExactTokensForTokens(uint amountIn, uint amountOutMin, address[] path, address to, uint deadline)
// Signature: 0x38ed1739

// Decoded data reveals:
// - amountIn: 1000000000000000000 (1 token)
// - amountOutMin: 950000000000000000 (0.95 token min)
// - path: [USDC, WETH] (swap route)
// - to: 0x123... (recipient)
// - deadline: 1700000000 (unix timestamp)
```

**LayerZero Bridge:**
```solidity
// Function: send(uint16 dstChainId, bytes dstAddress, bytes payload, ...)
// Signature: 0x001a0a6e

// Decoded data shows:
// - dstChainId: 137 (Polygon)
// - dstAddress: 0x456... (destination address)
// - payload: cross-chain message content
```

**Why parse calldata:**
- Understand user intent (swap, bridge, mint, vote)
- Track cross-chain activity flows
- Build protocol-specific analytics dashboards
- Enable smart contract interaction tracking

**Implementation:**
```go
// 1. Extract function signature (first 4 bytes)
signature := txInput[:4] // e.g., 0x38ed1739

// 2. Look up in protocol signatures database
protocol := db.LookupProtocol(signature)

// 3. Decode using ABI
abi := protocol.GetABI()
decoded, err := abi.Unpack(signature, txInput[4:])

// 4. Store in parsed_calldata table
db.InsertParsedCalldata(txHash, protocol, decoded)
```

#### **Revert Reason Extraction**
When a transaction fails, the EVM returns an error message. By default, you only see `status: false`.

**Example:**
```solidity
require(balance >= amount, "Insufficient balance");
// If this fails, revert reason = "Insufficient balance"
```

**Common Revert Reasons:**
- "Insufficient balance"
- "Transfer amount exceeds allowance"
- "Slippage tolerance exceeded"
- "Deadline expired"
- "Reentrancy guard"

**How to extract:**
```go
// For failed transactions
if receipt.Status == 0 {
    // Method 1: Replay transaction with eth_call
    _, err := client.CallContract(ctx, msg, receipt.BlockNumber)
    revertReason := parseRevertReason(err)
    
    // Method 2: Use debug_traceTransaction
    trace, _ := client.TraceTransaction(ctx, txHash)
    revertReason := parseTraceOutput(trace.Output)
}
```

**Why extract revert reasons:**
- Debug failed transactions
- Understand common failure patterns
- Alert users with meaningful errors
- Analytics on smart contract issues
- Improve UX by showing why tx failed

### 2. Blockchain Reorg Handling

**What is a Reorg?**
A blockchain reorganization occurs when a competing chain becomes longer than the current canonical chain, causing blocks to be replaced.

**Detection Algorithm**:
```go
// Compare parent hash of new block with stored block hash
if storedBlock.Hash != newBlock.ParentHash {
    // Reorg detected! Find common ancestor
    rollbackToBlock := findCommonAncestor(db, rpc)
    
    // Delete conflicting data
    db.DeleteBlocksAfter(rollbackToBlock)
    
    // Re-index from common ancestor
    resumeIngestion(rollbackToBlock + 1)
}
```

**Interview Points**:
- Ethereum finality: ~13 minutes (2 epochs), so reorgs typically affect last 15-20 blocks
- Polygon has shorter finality: ~256 blocks
- Always track parent hashes to detect reorgs
- Use database transactions for atomic rollback

### 3. Event Parsing (ERC20 Transfer Example)

**Event Signature**:
```solidity
// ERC20 Transfer event
event Transfer(address indexed from, address indexed to, uint256 value);

// Keccak256 hash of signature
0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
```

**Parsing Logic**:
```go
// topics[0] = event signature
// topics[1] = from address (indexed)
// topics[2] = to address (indexed)
// data = value (non-indexed, 32 bytes)

func parseTransfer(log *types.Log) (*TokenTransfer, error) {
    if len(log.Topics) < 3 {
        return nil, errors.New("invalid transfer event")
    }
    
    return &TokenTransfer{
        From:   common.BytesToAddress(log.Topics[1].Bytes()),
        To:     common.BytesToAddress(log.Topics[2].Bytes()),
        Amount: new(big.Int).SetBytes(log.Data),
    }, nil
}
```

**Interview Points**:
- Indexed parameters go in topics (max 3 indexed params)
- Non-indexed parameters go in data field
- ABI encoding/decoding is crucial for custom events
- Use go-ethereum's `abigen` for type-safe parsing

### 4. Database Partitioning Strategy

**Why Partition?**
- Faster queries on recent data
- Easier archival of old data
- Better maintenance (vacuum, reindex)

**Implementation**:
```sql
-- Partition by chain_id and block_number range
CREATE TABLE blocks (
    chain_id INT NOT NULL,
    block_number BIGINT NOT NULL,
    -- other columns
) PARTITION BY RANGE (chain_id, block_number);

-- Create partitions for each chain
CREATE TABLE blocks_eth_0_to_1m PARTITION OF blocks
    FOR VALUES FROM (1, 0) TO (1, 1000000);

CREATE TABLE blocks_eth_1m_to_2m PARTITION OF blocks
    FOR VALUES FROM (1, 1000000) TO (1, 2000000);
```

**Interview Points**:
- Query planner automatically selects correct partition
- Can drop old partitions for archival
- Composite key (chain_id, block_number) enables multi-chain partitioning

### 5. Rate Limiting with Token Bucket

**Algorithm**:
- Bucket has capacity N tokens
- Tokens refill at rate R per second
- Each request consumes 1 token
- Request denied if bucket is empty

**Implementation**:
```go
type TokenBucket struct {
    tokens       float64
    capacity     float64
    refillRate   float64 // tokens per second
    lastRefill   time.Time
    mu           sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()
    
    // Refill tokens
    tb.tokens = math.Min(tb.capacity, tb.tokens + elapsed * tb.refillRate)
    tb.lastRefill = now
    
    if tb.tokens >= 1.0 {
        tb.tokens -= 1.0
        return true
    }
    return false
}
```

**Interview Points**:
- More flexible than fixed window (allows bursts)
- Can implement per-IP and per-API-key limiting
- Use Redis for distributed rate limiting

### 6. Kafka Message Ordering

**Challenge**: Maintain block ordering across consumers

**Solution**: Partition by chain_id
```go
// Producer
producer.SendMessage(&sarama.ProducerMessage{
    Topic: "raw.blocks",
    Key:   sarama.StringEncoder(fmt.Sprintf("%d", chainID)),
    Value: sarama.ByteEncoder(blockData),
})

// Same chain_id always goes to same partition
// Ensures ordering within a chain
```

**Interview Points**:
- Kafka guarantees ordering within a partition
- Use chain_id as message key for consistent partitioning
- Consumer group ensures each partition is consumed by one consumer
- Enables parallel processing across chains

---

## Design Decisions & Trade-offs

### 1. Language Choice: Go vs Rust ⭐

**Decision**: **Go**

**TL;DR**:
- ✅ Better Ethereum ecosystem (go-ethereum is canonical)
- ✅ Faster development velocity (~3x faster to write)
- ✅ Easier to hire/onboard developers
- ✅ Sufficient performance for I/O-bound workloads
- ✅ Simpler concurrency model (goroutines)

**Rationale**:
- `go-ethereum` is the official Ethereum Foundation implementation
- Blockchain indexing is **I/O-bound** (network + database), not CPU-bound
- Go's goroutines handle concurrent block fetching perfectly
- Development speed matters more than 20-30% performance difference
- Larger talent pool for blockchain teams

**Performance Comparison** (10,000 blocks):
- **Go**: 42s, 520MB RAM, 3s compile
- **Rust**: 35s, 340MB RAM, 45s compile
- **Verdict**: Rust is 20% faster, but Go is "fast enough" and 15x faster to compile

**When Rust Would Be Better**:
- Absolute maximum performance required
- Team already has Rust expertise
- Building for embedded/resource-constrained environments
- Low-level protocol implementation

**Production Examples**:
- **Go**: Etherscan, The Graph, Alchemy, QuickNode
- **Rust**: Reth (Ethereum client), Lighthouse, Solana indexers

**Interview Answer**: "We chose Go because blockchain indexing is I/O-bound, not CPU-bound. We spend most time waiting for network responses and database writes. Go's goroutines handle this well, and go-ethereum is the canonical implementation with excellent documentation. The ~20% performance difference with Rust doesn't justify slower development velocity and hiring challenges. Major indexers like Etherscan and The Graph use Go successfully."

### 2. Message Broker: Kafka vs RabbitMQ

**Decision**: Kafka

**Rationale**:
- ✅ Better for high-throughput streaming data
- ✅ Built-in partitioning for parallel processing
- ✅ Message replay capability (important for reorgs)
- ✅ Better log retention for debugging
- ❌ More complex setup than RabbitMQ
- ❌ Higher resource usage

**Interview Answer**: "Kafka is better suited for append-only event streams like blockchain data. The replay capability is crucial for handling reorgs, and partitioning enables parallel processing across chains. RabbitMQ excels at task queues with complex routing, which isn't our use case."

### 3. Database: PostgreSQL vs TimescaleDB vs Cassandra

**Decision**: PostgreSQL with partitioning

**Rationale**:
- ✅ Strong consistency (important for financial data)
- ✅ Complex query support (joins, aggregations)
- ✅ Mature ecosystem and tooling
- ✅ Partitioning handles time-series nature
- ❌ TimescaleDB has better time-series optimizations
- ❌ Cassandra has better horizontal scaling

**Interview Answer**: "PostgreSQL offers the right balance of consistency, query flexibility, and operational maturity. The data has relational aspects (blocks → transactions → events), and users need complex queries. Native partitioning handles the time-series aspect adequately. We can shard by chain_id if we need horizontal scaling later."

### 4. Monorepo vs Separate Repos

**Decision**: Monorepo

**Rationale**:
- ✅ Shared models/config across services
- ✅ Atomic changes across services
- ✅ Simpler dependency management
- ✅ Easier local development
- ❌ Larger repo size
- ❌ All services version together

**Interview Answer**: "A monorepo makes sense for tightly coupled microservices. All three services share data models and evolve together. Separate repos would require versioning shared packages and coordinating deployments. As the team grows, we could split out services."

---

## Common Interview Questions

### Q1: How do you handle blockchain reorganizations?

**Answer**:
"We detect reorgs by comparing parent hashes. When ingesting block N, we check if our stored block N-1's hash matches the new block's parent hash. If not, we've detected a reorg.

We then:
1. Find the common ancestor by walking back through parent hashes
2. Use a database transaction to delete all blocks after the common ancestor
3. Resume ingestion from the common ancestor + 1
4. Kafka's replay capability allows the processor to re-process affected events

For production, we only mark blocks as 'finalized' after the chain-specific finality threshold (e.g., 13 minutes for Ethereum, 256 blocks for Polygon)."

### Q2: How do you scale the ingester for multiple chains?

**Answer**:
"We deploy separate ingester instances per chain, each with:
- Dedicated checkpoint tracking (last_indexed_block per chain)
- Chain-specific configuration (RPC URLs, finality rules, block times)
- Independent failure domains (one chain's RPC issues don't affect others)

Each ingester publishes to a chain-specific Kafka topic (e.g., 'eth.blocks', 'poly.blocks'). This enables:
- Parallel processing across chains
- Independent scaling per chain (Ethereum might need 3 ingesters, Polygon 1)
- Chain-specific monitoring and alerting

For efficiency, we could run multiple chains in one process with goroutines, but separate deployments provide better isolation."

### Q3: How do you ensure data consistency during high load?

**Answer**:
"We use several strategies:

1. **Database transactions**: All reorg rollbacks and event processing use ACID transactions

2. **Idempotency**: All operations are idempotent using `ON CONFLICT DO NOTHING` or upserts
   ```sql
   INSERT INTO events (...) VALUES (...) 
   ON CONFLICT (tx_hash, log_index) DO NOTHING;
   ```

3. **Exactly-once semantics**: Kafka consumers commit offsets only after successful database writes

4. **Checkpointing**: We persist last_processed_block and resume from there on restart

5. **Backpressure handling**: Kafka acts as a buffer between ingestion and processing, preventing overload

6. **Connection pooling**: Limit concurrent database connections to prevent saturation"

### Q4: How would you optimize query performance for address transaction history?

**Answer**:
"Multiple approaches:

1. **Composite indexes**:
   ```sql
   CREATE INDEX idx_txs_from_addr ON transactions (from_address, block_number DESC);
   CREATE INDEX idx_txs_to_addr ON transactions (to_address, block_number DESC);
   ```

2. **Materialized view** for frequently accessed patterns:
   ```sql
   CREATE MATERIALIZED VIEW address_activity AS
   SELECT address, COUNT(*) as tx_count, MAX(block_number) as last_active
   FROM (
       SELECT from_address as address FROM transactions
       UNION ALL
       SELECT to_address FROM transactions WHERE to_address IS NOT NULL
   ) GROUP BY address;
   ```

3. **Redis caching**: Cache recent 100 transactions per address (TTL 5 minutes)

4. **Pagination**: Always use limit/offset or cursor-based pagination

5. **Partitioning**: Partition by block_number to limit scan range

6. **Denormalization**: For high-value addresses, maintain a separate summary table"

### Q5: How do you handle API rate limiting at scale?

**Answer**:
"We implement multi-tier rate limiting:

1. **Per-IP limiting**: Token bucket with 100 req/sec, burst 200 (prevents DDoS)
2. **Per-API-key limiting**: Higher limits for authenticated users (1000 req/sec)
3. **Distributed rate limiting**: Use Redis with sliding window counters
   ```go
   key := fmt.Sprintf("ratelimit:%s:%d", apiKey, time.Now().Unix()/60)
   count := redis.Incr(key)
   redis.Expire(key, 120) // 2 minute window
   if count > limit { return 429 }
   ```

4. **Circuit breaker**: Temporarily block IPs that consistently hit limits
5. **Response headers**: Return `X-RateLimit-Remaining` and `X-RateLimit-Reset`
6. **Tiered pricing**: Free tier 100 req/sec, paid tiers 1000-10000 req/sec

We use Redis for distributed state, so any API server can enforce limits consistently."

---

## Go Programming Concepts - Interview Guide

This section covers key Go concepts used in this project, with examples from our codebase and interview-oriented explanations.

### 1. Goroutines ⭐ (USED)

**What**: Lightweight threads managed by Go runtime, much cheaper than OS threads.

**Where Used in Project**:
```go
// services/ingester/main.go:97
for _, chain := range chains {
    ingester.wg.Add(1)
    go ingester.ingestChain(chain)  // Spawn goroutine per chain
}

// services/ingester/main.go:87
go func() {
    <-sigChan
    log.Println("Shutdown signal received...")
    ingester.shutdown()
}()
```

**Interview Points**:
- **Lightweight**: Can spawn thousands (we spawn one per chain + signal handler)
- **Stack size**: Starts with 2KB, grows/shrinks dynamically (OS threads = 1-2MB fixed)
- **Scheduling**: M:N scheduling (M goroutines on N OS threads)
- **Use case**: Parallel chain ingestion - each chain processes independently
- **Communication**: Use channels for synchronization (see Channel section)

**Common Interview Questions**:
- Q: "Goroutine vs Thread?" → A: Goroutines are lighter (2KB vs 1MB), cooperatively scheduled
- Q: "How many goroutines can you spawn?" → A: Hundreds of thousands (limited by memory)
- Q: "When would you NOT use goroutines?" → A: CPU-bound work with limited cores (no benefit)

---

### 2. Context Propagation ⭐ (USED EXTENSIVELY)

**What**: Carries deadlines, cancellation signals, and request-scoped values across API boundaries.

**Where Used in Project**:
```go
// services/ingester/main.go:63 - Top-level context
ctx, cancel := context.WithCancel(context.Background())
ingester := &Ingester{
    ctx:    ctx,
    cancel: cancel,
}

// services/ingester/main.go:193 - Timeout for database operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := db.PingContext(ctx); err != nil {
    return nil, fmt.Errorf("failed to ping database: %w", err)
}

// services/ingester/main.go:280 - Propagated through RPC calls
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
header, err := client.HeaderByNumber(ctx, nil)
```

**Interview Points**:
- **Cancellation propagation**: Parent context cancels → all child contexts cancel
- **Timeout handling**: `WithTimeout` prevents hanging RPC calls
- **Graceful shutdown**: `ingester.ctx.Done()` signals all goroutines to stop
- **Best practice**: Always pass context as first parameter: `func DoWork(ctx context.Context, ...)`
- **Never store context**: Contexts should flow through call stack, not stored in structs (except for long-lived services like our Ingester)

**Context Types**:
```go
context.Background()              // Root context, never cancelled
context.TODO()                    // Placeholder when unsure
context.WithCancel(parent)        // Manual cancellation
context.WithTimeout(parent, 5*s)  // Time-based cancellation
context.WithDeadline(parent, t)   // Absolute deadline
context.WithValue(parent, key, v) // Carry request-scoped values (use sparingly!)
```

**Common Interview Questions**:
- Q: "Why pass context?" → A: Cancellation, timeouts, request tracing, distributed context
- Q: "When to use WithTimeout vs WithCancel?" → A: WithTimeout for I/O, WithCancel for manual control
- Q: "Is context safe for concurrent use?" → A: Yes, all methods are thread-safe

**Real-world Example from Project**:
```go
// When shutdown signal received, all RPC calls stop
select {
case <-ing.ctx.Done():
    log.Printf("🛑 Stopping block polling")
    return
case <-ticker.C:
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    header, err := client.HeaderByNumber(ctx, nil)  // Respects timeout
    cancel()
}
```

---

### 3. Channels ⭐ (USED)

**What**: Typed conduits for goroutines to communicate safely without explicit locks.

**Where Used in Project**:
```go
// services/ingester/main.go:84 - Signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

// services/ingester/main.go:303 - WebSocket block subscription
headers := make(chan *types.Header)
sub, err := client.SubscribeNewHead(ing.ctx, headers)

for {
    select {
    case <-ing.ctx.Done():
        return
    case err := <-sub.Err():
        log.Printf("Subscription error: %v", err)
    case header := <-headers:
        processBlock(header.Number.Int64())
    }
}
```

**Interview Points**:
- **Unbuffered vs Buffered**: 
  - `make(chan T)` → Synchronous, sender blocks until receiver ready
  - `make(chan T, N)` → Asynchronous, sender blocks only when buffer full
- **Directional channels**: `chan<- T` (send-only), `<-chan T` (receive-only)
- **Close semantics**: Only sender closes, receiver detects with `v, ok := <-ch`
- **select statement**: Multiplex on multiple channels (like epoll/kqueue)

**Channel Patterns**:
```go
// 1. Signal channel (done/cancel)
done := make(chan struct{})
go func() {
    <-done  // Block until signal
}()
close(done)  // Signal all receivers

// 2. Worker pool
jobs := make(chan Job, 100)
for i := 0; i < 10; i++ {
    go worker(jobs)
}

// 3. Fan-out, fan-in (map-reduce)
results := make(chan Result, numWorkers)
for _, input := range inputs {
    go func(in Input) {
        results <- process(in)
    }(input)
}
```

**Common Interview Questions**:
- Q: "Buffered vs unbuffered?" → A: Buffered = async (N capacity), unbuffered = sync handoff
- Q: "When to close channel?" → A: Only sender closes, when no more values will be sent
- Q: "What happens if you send to closed channel?" → A: Panic!
- Q: "What happens if you receive from closed channel?" → A: Immediate return with zero value

---

### 4. sync Package ⭐ (USED)

**What**: Low-level synchronization primitives for sharing memory between goroutines.

**Where Used in Project**:
```go
// services/ingester/main.go:33 - Wait for all chains to finish
type Ingester struct {
    wg           sync.WaitGroup
    shutdownOnce sync.Once
}

// services/ingester/main.go:95
for _, chain := range chains {
    ingester.wg.Add(1)
    go ingester.ingestChain(chain)
}
ingester.wg.Wait()  // Block until all chains stop

// services/ingester/main.go:564 - Ensure shutdown happens once
func (ing *Ingester) shutdown() {
    ing.shutdownOnce.Do(func() {
        ing.cancel()
        // Close all connections...
    })
}
```

**Interview Points**:

**sync.WaitGroup**:
- Use when you need to wait for multiple goroutines to finish
- `Add(n)` before spawning, `Done()` in goroutine, `Wait()` to block
- Counter-based (Add increments, Done decrements, Wait blocks until 0)

**sync.Once**:
- Ensures function runs exactly once, even with concurrent calls
- Use for lazy initialization, singletons, shutdown logic
- Thread-safe, no explicit locking needed

**sync.Mutex** (NOT USED - but important):
```go
type SafeCounter struct {
    mu    sync.Mutex
    count map[string]int
}

func (c *SafeCounter) Inc(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count[key]++
}
```

**sync.RWMutex** (NOT USED - but important):
```go
type Cache struct {
    mu    sync.RWMutex
    items map[string]interface{}
}

func (c *Cache) Get(key string) interface{} {
    c.mu.RLock()         // Multiple readers allowed
    defer c.mu.RUnlock()
    return c.items[key]
}

func (c *Cache) Set(key string, value interface{}) {
    c.mu.Lock()          // Exclusive write lock
    defer c.mu.Unlock()
    c.items[key] = value
}
```

**Common Interview Questions**:
- Q: "Mutex vs RWMutex?" → A: RWMutex allows multiple readers, one writer (read-heavy workloads)
- Q: "Channel vs Mutex?" → A: Channel = communication, Mutex = shared memory protection
- Q: "WaitGroup vs sync.Cond?" → A: WaitGroup = wait for N tasks, Cond = complex signaling

**Why We Don't Need Mutex**:
- Our design uses goroutines with isolated data (each chain has own state)
- Context cancellation handles coordination, not shared memory
- Database handles concurrency with transactions
- If we had shared cache, we'd need RWMutex

---

### 5. HTTP Serialization/Deserialization ⭐ (USED - API Service)

**What**: Converting Go structs to/from JSON for HTTP APIs.

**Where Used in Project**:
```go
// services/api/main.go - Response types
type Chain struct {
    ChainID     int64     `json:"chain_id"`
    Name        string    `json:"name"`
    IsActive    bool      `json:"is_active"`
    LastBlock   *int64    `json:"last_block,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

// services/api/main.go - Handler
func (api *API) handleGetChains(c *gin.Context) {
    rows, err := api.db.Query(query)
    // ... scan rows
    chains = append(chains, chain)
    
    c.JSON(http.StatusOK, chains)  // Auto-serialization
}

// Client-side deserialization
var chains []Chain
json.Unmarshal(body, &chains)
```

**Interview Points**:

**Struct Tags**:
```go
type User struct {
    ID        int    `json:"id" db:"user_id"`           // JSON and SQL names differ
    Email     string `json:"email" validate:"required"`  // Multiple tags
    Password  string `json:"-"`                          // Omit from JSON
    IsAdmin   bool   `json:"is_admin,omitempty"`        // Omit if zero value
}
```

**Serialization Methods**:
```go
// 1. Marshal (struct → JSON bytes)
data, err := json.Marshal(user)

// 2. Encoder (stream to io.Writer)
json.NewEncoder(w).Encode(user)  // More efficient for HTTP

// 3. Unmarshal (JSON bytes → struct)
var user User
json.Unmarshal(data, &user)

// 4. Decoder (stream from io.Reader)
json.NewDecoder(r.Body).Decode(&user)
```

**Common Interview Questions**:
- Q: "Marshal vs Encoder?" → A: Marshal returns bytes, Encoder writes to stream (HTTP response)
- Q: "What is omitempty?" → A: Omits field if zero value (0, false, nil, empty string)
- Q: "How to handle time.Time?" → A: Marshals to RFC3339 by default, can customize with MarshalJSON
- Q: "Case sensitivity?" → A: JSON keys are case-sensitive, Go uses exact match (case-insensitive fallback)

---

### 6. io Package (Reader, Writer, Stream Processing) ⚠️ (LIMITED USE)

**What**: Interfaces for I/O operations, enabling composition and streaming.

**Where Used in Project**: Minimal direct usage, mostly through libraries (database/sql, HTTP).

**Core Interfaces**:
```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type ReadWriter interface {
    Reader
    Writer
}
```

**Common Patterns** (Not in our project, but interview-relevant):
```go
// 1. Copy streams
io.Copy(dst Writer, src Reader)  // Efficient, no buffering

// 2. Read all (use sparingly, loads entire content to memory)
body, err := io.ReadAll(resp.Body)

// 3. Limit reader (prevent memory exhaustion)
limited := io.LimitReader(resp.Body, 1<<20)  // Max 1MB

// 4. Tee reader (duplicate stream)
tee := io.TeeReader(reader, logWriter)  // Read from reader, copy to logWriter

// 5. Pipe (connect reader/writer)
pr, pw := io.Pipe()
go func() {
    pw.Write(data)
    pw.Close()
}()
io.Copy(os.Stdout, pr)
```

**Why We Don't Use io Package Heavily**:
- Database library (`database/sql`) abstracts streaming
- Gin framework handles HTTP request/response bodies
- Blockchain data comes through `go-ethereum` library
- If we added CSV export or file processing, we'd use io.Reader/Writer

**Common Interview Questions**:
- Q: "Why use io.Reader instead of []byte?" → A: Streaming (low memory), composition, testability
- Q: "What is io.EOF?" → A: Sentinel error indicating end of stream
- Q: "How to chain readers?" → A: Wrap them: `gzip.NewReader(file)` → `io.LimitReader(gzipReader, size)`

---

### 7. Generics (Type Parameters) ❌ (NOT USED)

**What**: Parametric polymorphism added in Go 1.18, allows type-safe generic functions/structs.

**Why Not Used**:
- Project predates heavy generic adoption patterns
- `interface{}` or code generation sufficient for our use cases
- Database libraries use reflection, not generics
- Blockchain types (`*big.Int`, `common.Address`) are concrete

**Example Use Cases** (If we refactored):
```go
// Generic cache
type Cache[K comparable, V any] struct {
    mu    sync.RWMutex
    items map[K]V
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.items[key]
    return val, ok
}

// Usage
blockCache := Cache[int64, *Block]{}
txCache := Cache[string, *Transaction]{}

// Generic filter
func Filter[T any](slice []T, predicate func(T) bool) []T {
    result := make([]T, 0)
    for _, item := range slice {
        if predicate(item) {
            result = append(result, item)
        }
    }
    return result
}

// Usage
activeChains := Filter(chains, func(c Chain) bool { return c.IsActive })
```

**Common Interview Questions**:
- Q: "When to use generics vs interfaces?" → A: Generics = compile-time type safety, interfaces = runtime polymorphism
- Q: "Performance difference?" → A: Generics can be faster (no heap allocation for small types)
- Q: "What is `any` constraint?" → A: Alias for `interface{}`, means "any type"
- Q: "What is `comparable` constraint?" → A: Types that support == and != (used for map keys)

---

### 8. Testing with testify/require ❌ (NOT IMPLEMENTED)

**What**: Popular testing library with assertions, mocking, and test suites.

**Why Not Implemented**: MVP phase prioritized feature delivery over test coverage.

**Would Look Like** (If we added tests):
```go
// services/ingester/ingester_test.go
package main

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestIngestBlock(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    defer db.Close()
    
    ingester := &Ingester{db: db}
    
    // Execute
    err := ingester.processBlock(mockChain, 12345)
    
    // Assert
    require.NoError(t, err)  // Fail fast if error
    assert.Equal(t, 1, getBlockCount(db))
    assert.Equal(t, int64(12345), getLastCheckpoint(db))
}

func TestRateLimiter(t *testing.T) {
    limiter := NewTokenBucket(10, 5)  // 10 capacity, 5/sec refill
    
    // Allow first 10 requests (drain bucket)
    for i := 0; i < 10; i++ {
        assert.True(t, limiter.Allow(), "request %d should be allowed", i)
    }
    
    // 11th request should be denied
    assert.False(t, limiter.Allow())
}
```

**Common Interview Questions**:
- Q: "assert vs require?" → A: require stops test immediately, assert continues
- Q: "Table-driven tests?" → A: Loop over test cases slice (Go idiom)
- Q: "Test coverage tools?" → A: `go test -cover`, `go tool cover -html=coverage.out`

---

### 9. Testcontainers ❌ (NOT IMPLEMENTED)

**What**: Library for spinning up Docker containers in tests (real databases, Redis, Kafka).

**Why Not Implemented**: No integration tests yet, would add for production-readiness.

**Would Look Like**:
```go
import (
    "testing"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

func TestWithPostgres(t *testing.T) {
    ctx := context.Background()
    
    // Start PostgreSQL container
    req := testcontainers.ContainerRequest{
        Image:        "postgres:15",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_PASSWORD": "test",
        },
        WaitingFor: wait.ForLog("database system is ready"),
    }
    
    container, err := testcontainers.GenericContainer(ctx, req)
    require.NoError(t, err)
    defer container.Terminate(ctx)
    
    // Get connection string
    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "5432")
    
    // Run migrations
    db := connectDB(fmt.Sprintf("postgres://postgres:test@%s:%s/test", host, port))
    runMigrations(db)
    
    // Run tests against real database
    // ...
}
```

**Benefits**:
- Test against real database (not mocks)
- Catch SQL syntax errors
- Test migrations
- Reproducible environment

**Common Interview Questions**:
- Q: "Why not mock database?" → A: Mocks can't catch SQL errors, schema issues, performance problems
- Q: "Isn't it slow?" → A: Yes, but worth it for integration tests (run unit tests fast, integration slower)
- Q: "When to use?" → A: Integration tests, CI/CD, database migration testing

---

### 10. Google Go Style Guide (Idiomatic Go) ⭐ (MOSTLY FOLLOWED)

**What**: Official style guide for writing idiomatic, maintainable Go code.

**Key Principles Followed in Project**:

**✅ 1. Error Wrapping with fmt.Errorf**:
```go
// services/ingester/main.go:184
if err != nil {
    return nil, fmt.Errorf("failed to open database: %w", err)
}
```
- Use `%w` verb to wrap errors (enables `errors.Is`, `errors.As`)
- Provides context at each layer

**✅ 2. Early Returns (Guard Clauses)**:
```go
func (api *API) handleGetChain(c *gin.Context) {
    chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
        return  // Early return on error
    }
    
    // Happy path continues at normal indentation
    var chain Chain
    err = api.db.QueryRow(`...`, chainID).Scan(...)
}
```

**✅ 3. Named Return Values (Sparingly)**:
```go
func connectDB() (*sql.DB, error) {  // Unnamed returns (preferred)
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        return nil, fmt.Errorf("failed to open: %w", err)
    }
    return db, nil
}
```
- Only use named returns for documentation or complex functions

**✅ 4. defer for Cleanup**:
```go
// services/ingester/main.go:193
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()  // Ensures cancel is called
```

**✅ 5. Package Naming**:
- `package main` for executables
- Short, lowercase, no underscores: `models`, `config`

**❌ 6. Lacks Commentary**:
- Should add godoc comments for exported functions
- Example: `// Ingester manages blockchain data collection from multiple chains.`

**✅ 7. Struct Initialization**:
```go
ingester := &Ingester{
    db:        db,
    chains:    chains,
    clients:   make(map[int64]*ethclient.Client),
    ctx:       ctx,
    cancel:    cancel,
}
```
- Named fields (not positional)
- Clear, readable

**✅ 8. Error Messages**:
- Lowercase, no punctuation: `"failed to open database"`
- Context-rich: Include chain name, block number

**Google Go Style Interview Points**:
- **Simplicity over cleverness**: Readable code > clever tricks
- **Composition over inheritance**: Use interfaces, not class hierarchies
- **"Accept interfaces, return structs"**: Functions take `io.Reader`, return `*User`
- **Package layout**: `cmd/`, `internal/`, `pkg/` for structure
- **Avoid `init()`**: Explicit initialization preferred

**Common Interview Questions**:
- Q: "Go idioms vs other languages?" → A: Errors are values, composition, interfaces, channels
- Q: "What is 'accept interfaces, return structs'?" → A: Callers can pass anything implementing interface, you return concrete type
- Q: "Why no exceptions?" → A: Explicit error handling, errors are values

---

## Go Concepts Summary Table

| Concept | Status | Where Used | Interview Importance |
|---------|--------|------------|---------------------|
| **Goroutines** | ✅ Used | Multi-chain ingestion | ⭐⭐⭐ High |
| **Context** | ✅ Extensive | Timeouts, cancellation | ⭐⭐⭐ High |
| **Channels** | ✅ Used | Signals, subscriptions | ⭐⭐⭐ High |
| **sync.WaitGroup** | ✅ Used | Goroutine coordination | ⭐⭐⭐ High |
| **sync.Once** | ✅ Used | Shutdown guarantee | ⭐⭐ Medium |
| **sync.Mutex** | ❌ Not used | N/A | ⭐⭐⭐ High (still important!) |
| **HTTP Serialization** | ✅ Used | API responses | ⭐⭐ Medium |
| **io Package** | ⚠️ Limited | Indirect via libs | ⭐⭐ Medium |
| **Generics** | ❌ Not used | N/A | ⭐ Low (too new) |
| **testify/require** | ❌ No tests | N/A | ⭐⭐⭐ High (production need) |
| **Testcontainers** | ❌ Not implemented | N/A | ⭐⭐ Medium |
| **Idiomatic Go** | ✅ Mostly | Error handling, defer | ⭐⭐⭐ High |

**Key Takeaway for Interviews**:
This project demonstrates production-grade Go concurrency (goroutines, contexts, channels, sync primitives) applied to a real-world blockchain indexing system. The missing pieces (tests, generics, advanced io) are intentional MVP trade-offs, not knowledge gaps.

---

## Troubleshooting Guide

### Issue: Ingester falling behind chain head

**Symptoms**: `last_indexed_block` is 1000+ blocks behind current block

**Diagnosis**:
```bash
# Check ingestion rate
curl http://localhost:9090/metrics | grep blocks_ingested_total

# Check RPC latency
curl http://localhost:9090/metrics | grep rpc_request_duration
```

**Solutions**:
1. Add more RPC providers (load balance)
2. Increase batch size in config
3. Scale horizontally (multiple ingesters with block range sharding)
4. Check network connectivity to RPC provider

### Issue: Processor consumer lag

**Symptoms**: Kafka consumer lag increasing

**Diagnosis**:
```bash
# Check consumer group lag
kafka-consumer-groups --bootstrap-server localhost:9092 \
    --describe --group block_processor
```

**Solutions**:
1. Increase processor concurrency in config
2. Optimize database batch inserts (increase batch size)
3. Add database indexes for faster writes
4. Scale processor horizontally (Kafka will rebalance)
5. Check database connection pool settings

### Issue: API slow response times

**Symptoms**: P95 latency >1 second

**Diagnosis**:
```bash
# Check slow queries
psql -U indexer -d indexer -c "
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC LIMIT 10;"

# Check cache hit rate
redis-cli INFO stats | grep hit_rate
```

**Solutions**:
1. Add missing database indexes
2. Increase Redis cache TTL
3. Enable query result caching
4. Use connection pooling
5. Add read replicas for queries

---

## Performance Optimization

### Database Optimizations

```sql
-- 1. Analyze tables regularly
ANALYZE blocks;
ANALYZE transactions;
ANALYZE events;

-- 2. Vacuum to reclaim space
VACUUM FULL events;

-- 3. Reindex periodically
REINDEX TABLE events;

-- 4. Check index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0;  -- Unused indexes

-- 5. Connection pooling
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
```

### Go Performance Tips

```go
// 1. Use sync.Pool for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

// 2. Pre-allocate slices
events := make([]Event, 0, 1000)

// 3. Use buffered channels
ch := make(chan Block, 100)

// 4. Limit goroutines with worker pool
sem := make(chan struct{}, 10) // Max 10 concurrent workers

// 5. Profile in production
import _ "net/http/pprof"
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

### Kafka Optimizations

```yaml
# Producer settings
acks: 1  # Leader acknowledgment (balance between speed and durability)
compression.type: snappy  # Reduce network bandwidth
batch.size: 65536  # Larger batches
linger.ms: 10  # Wait up to 10ms to batch messages

# Consumer settings
fetch.min.bytes: 50000  # Fetch at least 50KB
fetch.max.wait.ms: 500  # Wait up to 500ms to accumulate data
max.poll.records: 1000  # Process 1000 messages per poll
```

---

## Frontend Development for Blockchain Engineers

### Overview: Frontend Landscape for DeFi/Web3

Building frontends for blockchain applications requires different considerations than traditional web apps. You're displaying real-time financial data, complex transaction histories, multi-chain support, and need to integrate with wallets.

**Key Differences from Traditional Web Apps:**
- 🔐 **Wallet Integration**: MetaMask, WalletConnect, Coinbase Wallet
- ⛓️ **Multi-chain Support**: Ethereum, Polygon, Arbitrum, etc.
- 💰 **Real-time Data**: Block updates every 12s, price feeds, pending txs
- 🔍 **Data Visualization**: Charts, graphs, transaction flows
- 🎨 **Consistent UX**: Dark mode standard, crypto-native design patterns

---

### Framework Comparison: React vs Next.js vs Vue

#### **1. React (Vite) ⚡**

**Best For**: Block explorers, dashboards, analytics tools, internal tools

**Pros:**
- ✅ **Fastest setup**: `npm create vite@latest` → running in 30 seconds
- ✅ **Maximum flexibility**: No opinions on routing, state management
- ✅ **Excellent DX**: Hot module reload (HMR), instant feedback
- ✅ **Smaller bundle**: No SSR overhead
- ✅ **Simple deployment**: Just static files → Vercel/Netlify/S3
- ✅ **Perfect for SPAs**: Block explorers are single-page apps

**Cons:**
- ❌ No built-in routing (need React Router)
- ❌ No SSR/SEO (fine for authenticated apps)
- ❌ Must handle data fetching yourself
- ❌ No API routes (need separate backend)

**When to Choose React + Vite:**
- Building a block explorer (like we are)
- Internal dashboards for traders
- Portfolio trackers
- Admin panels
- Any SPA where SEO doesn't matter

**Example Stack:**
```bash
Frontend: React 18 + Vite + TypeScript
Styling: Tailwind CSS
State: Zustand or Jotai (lighter than Redux)
Data: TanStack Query (React Query)
Charts: Recharts or Chart.js
Web3: Wagmi + viem
```

**Typical File Structure:**
```
src/
├── components/
│   ├── BlockCard.tsx
│   ├── TransactionList.tsx
│   └── ChainSwitcher.tsx
├── pages/
│   ├── Home.tsx
│   ├── BlockDetail.tsx
│   └── TransactionDetail.tsx
├── hooks/
│   ├── useBlocks.ts
│   ├── useChains.ts
│   └── useWebSocket.ts
├── services/
│   └── api.ts
└── App.tsx
```

---

#### **2. Next.js 14 (App Router) 🚀**

**Best For**: Public-facing DeFi protocols, DEX frontends, landing pages, documentation

**Pros:**
- ✅ **SEO-optimized**: Server-side rendering (SSR) & static generation (SSG)
- ✅ **Performance**: Automatic code splitting, image optimization
- ✅ **API routes**: Built-in serverless functions
- ✅ **File-based routing**: No need for React Router
- ✅ **Middleware**: Auth, redirects, rewriting at edge
- ✅ **Production-ready**: Used by Uniswap, Aave, Compound

**Cons:**
- ❌ More complex: SSR concepts, hydration, client vs server components
- ❌ Slower local dev: Webpack rebuilds can be slow
- ❌ More opinionated: Must follow Next.js conventions
- ❌ Vercel lock-in: Best experience on Vercel (but works elsewhere)
- ❌ Overkill for simple apps: Block explorers don't need SSR

**When to Choose Next.js:**
- Public DEX/protocol frontend (Uniswap clone)
- Marketing site + app combo
- Need SEO for Google/Twitter previews
- Blog + documentation + app
- Multi-region edge deployment

**Example Stack:**
```bash
Framework: Next.js 14 (App Router)
Styling: Tailwind CSS
State: Zustand + React Query
Web3: RainbowKit + Wagmi
Database: Prisma + PostgreSQL (if using API routes)
Deployment: Vercel Edge
```

**File Structure (App Router):**
```
app/
├── (marketing)/
│   ├── page.tsx          # Landing page
│   └── about/page.tsx
├── dashboard/
│   ├── page.tsx          # /dashboard
│   ├── layout.tsx
│   └── [chainId]/
│       └── blocks/page.tsx
├── api/
│   └── blocks/route.ts   # API endpoint
└── layout.tsx            # Root layout
```

**Key Next.js Features for Blockchain Apps:**

**1. Server Components (Default in App Router):**
```tsx
// app/blocks/page.tsx
async function BlocksPage({ params }) {
  // Fetch on server, no loading state on client
  const blocks = await fetch(`${API_URL}/blocks`).then(r => r.json());
  
  return <BlockList blocks={blocks} />;
}
```

**2. Client Components (for interactivity):**
```tsx
'use client'; // Mark as client component

import { useAccount } from 'wagmi';

export function WalletButton() {
  const { address, connect } = useAccount();
  return <button onClick={connect}>Connect</button>;
}
```

**3. API Routes (serverless functions):**
```tsx
// app/api/price/route.ts
export async function GET(request: Request) {
  const price = await fetchCoinGeckoPrice();
  return Response.json({ price });
}
```

**4. Middleware (edge functions):**
```tsx
// middleware.ts
export function middleware(request: NextRequest) {
  // Run auth, redirect, rewrite at edge
  const isAuthenticated = request.cookies.get('session');
  if (!isAuthenticated) {
    return NextResponse.redirect('/login');
  }
}
```

---

#### **3. Vue 3 (Composition API) 🟢**

**Best For**: Teams already using Vue, smaller projects, rapid prototyping

**Pros:**
- ✅ **Gentle learning curve**: Template syntax familiar from HTML
- ✅ **Great docs**: Vue documentation is excellent
- ✅ **Composition API**: Similar to React Hooks
- ✅ **Built-in state**: Pinia (official store)
- ✅ **Smaller bundle**: Vue 3 is very lightweight

**Cons:**
- ❌ Smaller ecosystem: Fewer Web3 libraries
- ❌ Less adoption: Most DeFi uses React
- ❌ Hiring challenge: Fewer Vue + blockchain devs
- ❌ Wagmi/Viem are React-first

**When to Choose Vue:**
- Team already knows Vue
- Building quick MVP/prototype
- Smaller codebase (<10K LOC)
- Not prioritizing Web3 libraries

**Example Stack:**
```bash
Framework: Vue 3 + Vite
Styling: Tailwind CSS
State: Pinia
Web3: web3-onboard or ethers.js
Charts: ApexCharts
```

---

### Styling: Tailwind CSS vs Styled Components vs CSS Modules

#### **Tailwind CSS 🎨** ⭐ RECOMMENDED

**Why Tailwind Dominates Web3:**
- ✅ **Fastest development**: No context switching between HTML/CSS
- ✅ **Consistent design**: Pre-defined spacing, colors, shadows
- ✅ **Responsive built-in**: `md:flex lg:grid xl:gap-4`
- ✅ **Dark mode easy**: `dark:bg-gray-900 dark:text-white`
- ✅ **Purge unused**: Final bundle ~10KB
- ✅ **Component libraries**: ShadCN, DaisyUI, HeadlessUI

**Example:**
```tsx
<div className="
  flex items-center justify-between
  p-4 rounded-lg
  bg-white dark:bg-gray-800
  border border-gray-200 dark:border-gray-700
  hover:shadow-lg transition-shadow
">
  <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
    Block #{blockNumber}
  </h3>
  <span className="text-sm text-gray-500">
    {txCount} transactions
  </span>
</div>
```

**Tailwind + Component Libraries:**
```bash
# ShadCN (most popular for Web3)
npx shadcn-ui@latest init
npx shadcn-ui@latest add button card dialog

# DaisyUI (pre-built components)
npm install daisyui
```

**Pros:**
- ⚡ Ultra-fast development (~3x faster)
- 🎨 Enforced consistency
- 📱 Responsive by default
- 🌙 Dark mode trivial
- 📦 Tiny production bundle
- 🧩 Works with all frameworks

**Cons:**
- 📚 Learning curve (new utility names)
- 🔤 Long className strings (use `clsx` helper)
- 🎭 Not designer-friendly (devs love it, designers hate it)

---

#### **Styled Components / Emotion 💅**

**When to Use:**
- Team values CSS-in-JS
- Need dynamic theming
- Component-scoped styles critical

**Pros:**
- ✅ Co-located styles with components
- ✅ Dynamic styling with props
- ✅ Automatic critical CSS
- ✅ Theme provider built-in

**Cons:**
- ❌ Runtime cost (styles injected at runtime)
- ❌ Larger bundle size
- ❌ Slower than Tailwind
- ❌ Hydration issues with SSR

**Example:**
```tsx
const BlockCard = styled.div<{ $isNew: boolean }>`
  padding: 1rem;
  background: ${props => props.$isNew ? '#dcfce7' : '#fff'};
  border-radius: 0.5rem;
  &:hover {
    box-shadow: 0 10px 15px rgba(0,0,0,0.1);
  }
`;
```

**Verdict**: Tailwind wins for blockchain apps. Faster development, smaller bundles, better DX.

---

#### **CSS Modules 📦**

**When to Use:**
- No build step desired
- Team prefers traditional CSS
- Legacy codebase

**Pros:**
- ✅ Scoped by default
- ✅ No runtime cost
- ✅ Familiar CSS syntax
- ✅ Good for migrations

**Cons:**
- ❌ Verbose imports
- ❌ Separate files
- ❌ No dynamic styling
- ❌ Responsive more manual

---

### State Management Comparison

#### **1. Zustand 🐻** ⭐ RECOMMENDED for Web3

**Why Perfect for Blockchain Apps:**
- ✅ Minimal boilerplate: ~3 lines to create store
- ✅ No providers: Access anywhere
- ✅ DevTools: Time-travel debugging
- ✅ Middleware: Persist to localStorage
- ✅ Tiny: 1KB gzipped

**Example:**
```tsx
// store/useBlockStore.ts
import { create } from 'zustand';

interface BlockStore {
  chainId: number;
  setChainId: (id: number) => void;
  blocks: Block[];
  addBlock: (block: Block) => void;
}

export const useBlockStore = create<BlockStore>((set) => ({
  chainId: 1,
  setChainId: (id) => set({ chainId: id }),
  blocks: [],
  addBlock: (block) => set((state) => ({
    blocks: [block, ...state.blocks]
  })),
}));

// Usage in component
function ChainSwitcher() {
  const { chainId, setChainId } = useBlockStore();
  return <select onChange={e => setChainId(+e.target.value)} value={chainId}>
    <option value={1}>Ethereum</option>
    <option value={137}>Polygon</option>
  </select>;
}
```

**With Persistence:**
```tsx
import { persist } from 'zustand/middleware';

export const useBlockStore = create(
  persist(
    (set) => ({
      chainId: 1,
      setChainId: (id) => set({ chainId: id }),
    }),
    { name: 'block-store' } // localStorage key
  )
);
```

---

#### **2. Jotai 👻** (Atomic State)

**When to Use:**
- Need granular re-renders
- Many independent pieces of state
- Want React Suspense integration

**Example:**
```tsx
import { atom, useAtom } from 'jotai';

const chainIdAtom = atom(1);
const blocksAtom = atom([]);

function ChainSwitcher() {
  const [chainId, setChainId] = useAtom(chainIdAtom);
  return <select onChange={e => setChainId(+e.target.value)} />;
}
```

---

#### **3. Redux Toolkit 📦** (Legacy/Large Teams)

**When to Use:**
- Very large codebase (50K+ LOC)
- Many developers (10+)
- Need strict patterns
- Migrating from Redux

**Cons:**
- ❌ Much more boilerplate
- ❌ Slower development
- ❌ Overkill for most Web3 apps

**Verdict**: Use **Zustand** for 90% of blockchain apps. It's simpler, faster, and the ecosystem loves it.

---

## Cross-Stack Learning: Transferable Skills & Concepts

### The Truth About Framework Switching

**Good News**: 80%+ of skills transfer between React, Next.js, Vue, Svelte, and Solid. The underlying concepts are universal.

**Time to Switch Frameworks**: 
- React → Next.js: **1-2 days** (same paradigm, just add SSR concepts)
- React → Vue 3: **1-2 weeks** (different syntax, same concepts)
- React → Svelte: **2-3 weeks** (different reactivity model)
- Next.js → Remix: **2-5 days** (similar SSR, different data loading)

**Why Skills Transfer:**
1. All modern frameworks use **component-based architecture**
2. All have **reactive state** (hooks, signals, or stores)
3. All support **TypeScript**
4. All use **npm ecosystem** (same build tools, libraries)
5. All solve **the same problems** (routing, data fetching, forms)

---

### Core Concepts That Transfer Everywhere

#### **1. Component Composition** ✅ (100% Transferable)

**Universal Pattern:**
```tsx
// Works conceptually in React, Vue, Svelte, Solid
function BlockList({ blocks }) {
  return blocks.map(block => <BlockCard key={block.id} block={block} />);
}
```

**React:**
```tsx
function BlockCard({ block }: { block: Block }) {
  return <div>{block.number}</div>;
}
```

**Vue 3 (Composition API):**
```vue
<script setup lang="ts">
defineProps<{ block: Block }>();
</script>
<template>
  <div>{{ block.number }}</div>
</template>
```

**Svelte:**
```svelte
<script lang="ts">
  export let block: Block;
</script>
<div>{block.number}</div>
```

**Takeaway**: Once you understand component composition in one framework, you understand it in all. Only syntax changes.

---

#### **2. Reactive State** ✅ (95% Transferable)

**The Core Idea**: When state changes, UI updates automatically.

**React (useState):**
```tsx
function Counter() {
  const [count, setCount] = useState(0);
  return <button onClick={() => setCount(count + 1)}>{count}</button>;
}
```

**Vue 3 (ref):**
```vue
<script setup>
import { ref } from 'vue';
const count = ref(0);
</script>
<template>
  <button @click="count++">{{ count }}</button>
</template>
```

**Svelte (reactive declarations):**
```svelte
<script>
  let count = 0;
</script>
<button on:click={() => count++}>{count}</button>
```

**Solid.js (signals):**
```tsx
function Counter() {
  const [count, setCount] = createSignal(0);
  return <button onClick={() => setCount(count() + 1)}>{count()}</button>;
}
```

**Transferable Knowledge:**
- ✅ Concept of "state triggers re-render"
- ✅ Immutability patterns (don't mutate state directly)
- ✅ Derived state / computed values
- ✅ Effect hooks / watchers

**What Changes:**
- ❌ API syntax (`useState` vs `ref` vs `let`)
- ❌ How reactivity is tracked (Virtual DOM vs Compiler vs Signals)

---

#### **3. Side Effects / Lifecycle** ✅ (90% Transferable)

**Universal Pattern**: Run code when component mounts, updates, or unmounts.

**React (useEffect):**
```tsx
useEffect(() => {
  const ws = new WebSocket('ws://api/blocks');
  ws.onmessage = (e) => setBlocks(JSON.parse(e.data));
  return () => ws.close(); // Cleanup
}, []);
```

**Vue 3 (onMounted, onUnmounted):**
```vue
<script setup>
import { onMounted, onUnmounted } from 'vue';

let ws;
onMounted(() => {
  ws = new WebSocket('ws://api/blocks');
  ws.onmessage = (e) => blocks.value = JSON.parse(e.data);
});
onUnmounted(() => ws?.close());
</script>
```

**Svelte (onMount):**
```svelte
<script>
import { onMount } from 'svelte';

onMount(() => {
  const ws = new WebSocket('ws://api/blocks');
  ws.onmessage = (e) => blocks = JSON.parse(e.data);
  return () => ws.close();
});
</script>
```

**Transferable Knowledge:**
- ✅ When to run side effects
- ✅ Cleanup pattern (return function or onUnmounted)
- ✅ Dependency tracking
- ✅ Avoiding memory leaks

---

#### **4. Data Fetching Patterns** ✅ (100% Transferable)

**Same Libraries Work Across Frameworks:**

**React Query → TanStack Query** (framework-agnostic now!)
```tsx
// Works in React, Vue, Svelte, Solid
import { useQuery } from '@tanstack/react-query'; // or vue-query, solid-query

const { data, isLoading } = useQuery({
  queryKey: ['blocks'],
  queryFn: fetchBlocks,
});
```

**Transferable Patterns:**
- ✅ Caching strategies
- ✅ Optimistic updates
- ✅ Stale-while-revalidate
- ✅ Infinite scrolling
- ✅ Prefetching

**Learn Once, Use Everywhere**: TanStack Query has adapters for React, Vue, Svelte, Solid. Your mental model transfers perfectly.

---

#### **5. Routing Concepts** ✅ (85% Transferable)

**Universal Patterns:**
- File-based routing (Next.js, Nuxt, SvelteKit, SolidStart)
- Dynamic routes (`/blocks/[id]`)
- Nested layouts
- Middleware / Guards
- Programmatic navigation

**React Router:**
```tsx
<Routes>
  <Route path="/" element={<Home />} />
  <Route path="/blocks/:id" element={<BlockDetail />} />
</Routes>
```

**Next.js (App Router):**
```
app/
  page.tsx          → /
  blocks/[id]/page.tsx → /blocks/123
```

**Vue Router:**
```tsx
const routes = [
  { path: '/', component: Home },
  { path: '/blocks/:id', component: BlockDetail }
];
```

**Transferable Knowledge:**
- ✅ Route parameters
- ✅ Query strings
- ✅ Nested routes
- ✅ Navigation guards/middleware
- ✅ Programmatic navigation

---

### React Ecosystem Deep Dive

#### **Why React Dominates Web3/DeFi**

**Market Share in Blockchain:**
- Uniswap: Next.js
- Aave: Next.js
- Compound: React
- OpenSea: Next.js
- Etherscan: React
- **~85% of top DeFi protocols use React/Next.js**

**Reasons:**
1. ✅ **Wagmi/Viem ecosystem**: Best Web3 libraries are React-first
2. ✅ **Hiring**: 10x more React devs than Vue/Svelte
3. ✅ **Component libraries**: ShadCN, Radix, Mantine built for React
4. ✅ **Vercel ecosystem**: Next.js has best DX + deployment
5. ✅ **Community**: Largest ecosystem, fastest bug fixes

---

#### **React Framework Landscape 2025**

**1. Next.js 14** (App Router) 🏆 **Industry Leader**

**When to Use:**
- Public-facing protocol frontends (Uniswap, Aave)
- Marketing site + app combo
- Need SEO for landing pages
- Multi-region edge deployment
- API routes for backend logic

**Pros:**
- ✅ Used by 70%+ of top DeFi protocols
- ✅ Best-in-class DX (Turbopack, Fast Refresh)
- ✅ Vercel deployment = zero config
- ✅ Server Components = faster initial load
- ✅ Middleware for auth/redirects at edge

**Cons:**
- ❌ More complex (client vs server components)
- ❌ Vercel lock-in (best experience)
- ❌ App Router learning curve
- ❌ Overkill for simple block explorers

**Real-World Usage:**
```tsx
// app/swap/page.tsx - Server Component (default)
async function SwapPage() {
  const pools = await fetchPools(); // Runs on server
  return <PoolList pools={pools} />;
}

// components/SwapForm.tsx - Client Component
'use client';
export function SwapForm() {
  const { address } = useAccount(); // Wagmi hook requires client
  return <form>...</form>;
}
```

---

**2. Vite + React** ⚡ **Best for SPAs**

**When to Use:**
- Block explorers (our use case)
- Dashboards and analytics tools
- Internal admin panels
- Portfolio trackers
- Any SPA where SEO doesn't matter

**Pros:**
- ✅ Fastest dev server (HMR in <100ms)
- ✅ Simplest setup (`npm create vite@latest`)
- ✅ No SSR complexity
- ✅ Smallest learning curve
- ✅ Deploy as static files to CDN

**Cons:**
- ❌ No built-in routing (add React Router)
- ❌ No SEO (fine for authenticated apps)
- ❌ No API routes (need separate backend)

**Perfect For:**
```bash
# Block Explorer Stack
Vite + React 18 + TypeScript
React Router v6
Tailwind CSS
Zustand (state)
React Query (data)
Wagmi (Web3)

# Deploy to: Vercel, Netlify, Cloudflare Pages (free)
```

---

**3. Remix** 🎵 **Progressive Enhancement**

**When to Use:**
- Need forms to work without JavaScript
- Progressive web apps (PWAs)
- Want React Router on server
- Prefer co-located data loading

**Pros:**
- ✅ Built on Web Standards (fetch, FormData)
- ✅ Excellent UX (optimistic UI, prefetching)
- ✅ No client-side state management needed
- ✅ Nested routes with layouts

**Cons:**
- ❌ Smaller ecosystem than Next.js
- ❌ Less tooling (no Vercel-level DX)
- ❌ Fewer Web3 examples

**Key Difference from Next.js:**
```tsx
// Remix - loader runs on server
export async function loader({ params }) {
  return json({ blocks: await fetchBlocks(params.chainId) });
}

export default function Blocks() {
  const { blocks } = useLoaderData();
  return <BlockList blocks={blocks} />;
}

// Next.js - same concept, different API
async function BlocksPage({ params }) {
  const blocks = await fetchBlocks(params.chainId);
  return <BlockList blocks={blocks} />;
}
```

**Transferable Skills:** If you know Next.js, you know 80% of Remix. Main difference is data loading API.

---

**4. Astro** 🚀 **Content-Heavy Sites**

**When to Use:**
- Documentation sites
- Marketing pages
- Blogs with occasional interactivity
- Landing pages for protocols

**Pros:**
- ✅ Ship **zero JavaScript** by default
- ✅ Use React components only where needed
- ✅ Fastest possible page loads
- ✅ Markdown/MDX built-in

**Cons:**
- ❌ Not for SPAs (by design)
- ❌ Minimal interactivity
- ❌ No Web3 wallet integration (unless islands of React)

**Example:**
```astro
---
// Runs at build time, not in browser
const blocks = await fetchBlocks();
---
<Layout>
  <h1>Latest Blocks</h1>
  <!-- Static HTML, no JS -->
  {blocks.map(block => <BlockCard block={block} />)}
  
  <!-- Interactive island (ships React bundle) -->
  <WalletButton client:load />
</Layout>
```

**Use Case**: Protocol documentation site (docs.uniswap.org style) where 95% is static content, 5% needs wallet connection.

---

**5. TanStack Start** 🏁 **New Kid on Block (2024)**

**When to Use:**
- Want Next.js-style SSR with more control
- Prefer TanStack ecosystem (Router, Query, Table)
- Building data-heavy apps

**Status**: Early (beta), but backed by Tanner Linsley (React Query creator)

**Watch This Space**: Could become React Router's official SSR framework.

---

#### **React State Management Evolution**

**Timeline:**
```
2015: Redux (boilerplate hell)
2019: Context API (prop drilling solved)
2020: Recoil (atoms, but complex)
2021: Zustand (perfect balance) ⭐
2022: Jotai (atomic state)
2023: Zustand still winning
```

**Why Zustand Won:**
```tsx
// Redux: ~50 lines for counter
// Zustand: 8 lines
import { create } from 'zustand';

const useStore = create((set) => ({
  count: 0,
  increment: () => set((state) => ({ count: state.count + 1 })),
}));

// No providers, no reducers, no actions, no connect()
function Counter() {
  const { count, increment } = useStore();
  return <button onClick={increment}>{count}</button>;
}
```

**When to Use Each:**

| Tool | Use Case | Complexity |
|------|----------|-----------|
| **Zustand** | 90% of apps | Low |
| **Jotai** | Atomic state, granular updates | Medium |
| **Redux Toolkit** | Legacy codebases, strict patterns | High |
| **Context** | Theme, auth (rarely changes) | Low |
| **React Query** | Server state only (not app state) | Low |

---

#### **Styling Evolution in React Ecosystem**

**Timeline:**
```
2015: CSS Modules (scoped styles)
2017: Styled Components (CSS-in-JS hype)
2019: Tailwind CSS (utility-first)
2021: Tailwind dominates Web3
2023: Tailwind + ShadCN standard
2025: Still Tailwind everywhere
```

**Why Tailwind Won DeFi:**

**Before (Styled Components):**
```tsx
const Card = styled.div`
  display: flex;
  padding: 1rem;
  background: white;
  border-radius: 0.5rem;
  &:hover {
    box-shadow: 0 10px 15px rgba(0,0,0,0.1);
  }
`;

function BlockCard({ block }) {
  return <Card>
    <Title>{block.number}</Title>
  </Card>;
}
```

**After (Tailwind):**
```tsx
function BlockCard({ block }) {
  return (
    <div className="flex p-4 bg-white rounded-lg hover:shadow-xl">
      <h3 className="text-lg font-semibold">{block.number}</h3>
    </div>
  );
}
```

**Speed Difference:** Tailwind is 3-5x faster for most developers.

**Component Libraries Built on Tailwind:**

| Library | Description | When to Use |
|---------|-------------|-------------|
| **ShadCN** | Copy/paste components | Full control, customization |
| **DaisyUI** | Pre-built components | Fast prototyping, themes |
| **Headless UI** | Unstyled primitives | Custom designs, accessibility |
| **Radix UI** | Unstyled primitives | Low-level control |
| **Mantine** | Full-featured | Admin panels, dashboards |

**ShadCN Dominance:**
```bash
# Instead of npm install
npx shadcn-ui@latest add button

# Copies source code to your project
# ✅ Full ownership
# ✅ Customize freely
# ✅ No dependency updates breaking your app
# ✅ TypeScript-first
```

**Used by**: Vercel, OpenAI, Cloudflare, and 60%+ of new React projects.

---

### Alternative Frameworks: When to Look Beyond React

#### **Svelte / SvelteKit** 🧡

**When to Consider:**
- Team values simplicity over ecosystem size
- Building smaller apps (<10K LOC)
- Performance is critical (Svelte compiles to vanilla JS)
- Want faster onboarding for junior devs

**Pros:**
- ✅ **No Virtual DOM**: Compiles to efficient vanilla JS
- ✅ **Less code**: Same app is ~30% fewer lines
- ✅ **Built-in reactivity**: No hooks confusion
- ✅ **Smooth animations**: Built-in transitions

**Cons:**
- ❌ Tiny Web3 ecosystem (no Wagmi equivalent)
- ❌ Harder hiring (5-10x fewer Svelte devs)
- ❌ Less mature tooling

**Transferable Skills:** 85% - Component structure, lifecycle, routing concepts all similar.

**Real Example:**
```svelte
<script>
  let blocks = [];
  
  // No useEffect, no useState - just reactive
  $: latestBlock = blocks[0];
  
  async function loadBlocks() {
    blocks = await fetch('/api/blocks').then(r => r.json());
  }
</script>

<button on:click={loadBlocks}>Load</button>
{#each blocks as block}
  <div>{block.number}</div>
{/each}
```

---

#### **Solid.js** 💠 **React-like, But Faster**

**When to Consider:**
- Need React-like DX with better performance
- Building complex SPAs with lots of state
- Want fine-grained reactivity (no re-renders)

**Pros:**
- ✅ **JSX syntax** (looks like React)
- ✅ **Fastest framework** (benchmarks beat Svelte)
- ✅ **No Virtual DOM**: Direct DOM updates
- ✅ **Signals**: Fine-grained reactivity

**Cons:**
- ❌ Smallest ecosystem of all
- ❌ No major Web3 apps using it
- ❌ Very hard hiring

**Comparison:**
```tsx
// React - whole component re-renders
function Counter() {
  const [count, setCount] = useState(0);
  console.log('re-rendered'); // Logs on every click
  return <button onClick={() => setCount(count + 1)}>{count}</button>;
}

// Solid - only signal updates, no re-render
function Counter() {
  const [count, setCount] = createSignal(0);
  console.log('only logs once'); // Never re-runs
  return <button onClick={() => setCount(count() + 1)}>{count()}</button>;
}
```

**Transferable Skills:** 95% if you know React. Signals are the only new concept.

---

#### **Vue 3** 💚 **Second Most Popular**

**When to Consider:**
- Team already knows Vue
- Building admin dashboards (Vue excels here)
- Want gentler learning curve than React

**Pros:**
- ✅ Excellent documentation
- ✅ Composition API similar to React Hooks
- ✅ Built-in state (Pinia)
- ✅ Great TypeScript support

**Cons:**
- ❌ Smaller Web3 ecosystem (but has web3-onboard)
- ❌ Less adoption in DeFi (maybe 10% of protocols)
- ❌ Template syntax polarizing

**Transferable Skills:** 75% - Concepts transfer, but syntax very different.

---

### Decision Matrix: Which Stack for Which Project?

| Project Type | Recommended Stack | Why |
|--------------|------------------|-----|
| **Block Explorer** | Vite + React + Tailwind | SPA, no SEO needed, fastest dev |
| **DEX Frontend** | Next.js + RainbowKit | Public, needs SEO, wallet integration |
| **Portfolio Tracker** | Vite + React + Wagmi | SPA, authenticated, real-time updates |
| **Protocol Docs** | Astro + React islands | Mostly static, occasional interactivity |
| **Admin Dashboard** | Next.js or Vite + React | Authenticated, no SEO needed |
| **Landing Page** | Next.js or Astro | SEO critical, static content |
| **NFT Marketplace** | Next.js + Wagmi | Public, SEO, complex state |
| **Analytics Tool** | Vite + React + Recharts | Charts, real-time, no SEO |

---

### Learning Roadmap: React Ecosystem Mastery

**Phase 1: React Fundamentals (1-2 weeks)**
```
✅ Components & Props
✅ useState & useEffect
✅ Conditional rendering & lists
✅ Forms & controlled inputs
✅ Context API
✅ Custom hooks
```

**Phase 2: Modern Tooling (1 week)**
```
✅ Vite setup & dev server
✅ TypeScript basics
✅ ESLint & Prettier
✅ React DevTools
```

**Phase 3: State & Data (1 week)**
```
✅ Zustand for global state
✅ React Query for server state
✅ Optimistic updates
✅ Error boundaries
```

**Phase 4: Styling (3-5 days)**
```
✅ Tailwind CSS utilities
✅ ShadCN component installation
✅ Dark mode implementation
✅ Responsive design
```

**Phase 5: Routing (3-5 days)**
```
✅ React Router v6
✅ Dynamic routes
✅ Protected routes
✅ Programmatic navigation
```

**Phase 6: Web3 Integration (1 week)**
```
✅ Wagmi + RainbowKit setup
✅ Wallet connection
✅ Reading contract data
✅ Sending transactions
✅ Multi-chain support
```

**Phase 7: Next.js (Optional, 1 week)**
```
✅ App Router vs Pages Router
✅ Server vs Client Components
✅ Data fetching patterns
✅ API routes
✅ Deployment to Vercel
```

**Total Time: 5-8 weeks to job-ready React developer**

---

### Interview Prep: React Ecosystem Questions

**Q: When would you choose Next.js over Vite + React?**

**A:** "I'd choose Next.js when:
1. **SEO matters** - Public landing pages, marketing sites, protocol homepages
2. **Need API routes** - Want backend logic in same repo (serverless functions)
3. **Multi-region deployment** - Vercel Edge gives global CDN automatically
4. **Large team** - Next.js conventions reduce bikeshedding

I'd choose Vite + React when:
1. **Building SPAs** - Block explorers, dashboards, authenticated tools
2. **Speed matters** - Vite's HMR is faster, simpler mental model
3. **No SEO needed** - Most Web3 tools are bookmarked directly
4. **Smaller team** - Less complexity, faster iteration"

---

**Q: Explain the difference between Server and Client Components in Next.js.**

**A:** "Server Components (default in App Router):
- Run on server only, never shipped to browser
- Can access databases directly
- Cannot use hooks or browser APIs
- Reduce JavaScript bundle size

Client Components (marked with 'use client'):
- Run in browser
- Can use useState, useEffect, event handlers
- Required for interactivity and Web3 (Wagmi hooks)

Best practice: Start with Server Components, only add 'use client' when needed. For blockchain apps, wallet connection and transaction forms are client components, but block lists and analytics can be server components."

---

**Q: How does Zustand compare to Redux?**

**A:** "Zustand vs Redux:

**Redux:**
- Actions, reducers, middleware - lots of boilerplate
- ~50-100 lines for simple counter
- Best for large teams needing strict patterns
- Redux DevTools excellent for debugging
- Steep learning curve

**Zustand:**
- ~8 lines for same counter
- No providers, no connect(), no reducers
- Perfect for small-medium teams
- Also has DevTools
- Learn in 30 minutes

For blockchain apps, Zustand is usually better because:
1. Faster development (3-5x less code)
2. Smaller bundle (~1KB vs ~10KB)
3. Most Web3 apps don't need Redux's complexity
4. Uniswap v3 migrated from Redux to Zustand

Use Redux only if: large team (10+), very complex state, or legacy codebase."

---

**Q: Why is Tailwind CSS so popular in Web3?**

**A:** "Tailwind dominates Web3 for several reasons:

1. **Speed**: ~3x faster development. No context switching between HTML/CSS files.

2. **Consistency**: Pre-defined spacing scale (4px, 8px, 16px) prevents arbitrary values. Every dev uses `p-4` not `padding: 17px`.

3. **Dark mode**: Built-in `dark:` variant. Critical for crypto apps (95% use dark mode).

4. **Responsive**: `md:flex lg:grid` - responsive design is trivial.

5. **Component libraries**: ShadCN, DaisyUI, HeadlessUI all built on Tailwind.

6. **No naming**: No bikeshedding over class names (`.header-button-primary-large` vs `<button className='px-4 py-2 bg-blue-500'>`).

7. **Tiny bundle**: Purges unused classes → ~10KB final CSS.

8. **Hiring**: Most new React devs learn Tailwind first now.

Alternatives (Styled Components, CSS Modules) are slower and have runtime costs. For fast-moving startups, Tailwind is the pragmatic choice."

---

### Key Takeaway: Learn React Deeply, Switch Frameworks Easily

**The Meta-Skill**: Understanding component-based architecture, reactive state, side effects, and data fetching patterns.

**Once you know React well:**
- Next.js = 2 days to learn (same paradigm + SSR)
- Vue 3 = 1-2 weeks (different syntax, same concepts)
- Svelte = 2-3 weeks (different reactivity)
- Solid = 1 week (similar to React, but signals)

**80% of your knowledge transfers.** The hard problems (state management, data fetching, performance optimization, Web3 integration) are framework-agnostic.

**Bottom Line**: Master React + TypeScript + Tailwind + React Query + Zustand → You can build 90% of Web3 frontends and switch to any other framework in 1-3 weeks if needed.

---

### Data Fetching: TanStack Query (React Query)

**The Standard for Blockchain Apps:**

```tsx
import { useQuery } from '@tanstack/react-query';

function BlockList() {
  const { data: blocks, isLoading } = useQuery({
    queryKey: ['blocks', chainId],
    queryFn: () => fetch(`/api/v1/chains/${chainId}/blocks`).then(r => r.json()),
    refetchInterval: 12000, // Refetch every 12s (Ethereum block time)
  });

  if (isLoading) return <div>Loading...</div>;
  
  return <div>{blocks.map(block => <BlockCard key={block.number} {...block} />)}</div>;
}
```

**Why React Query?**
- ✅ **Automatic caching**: No manual cache management
- ✅ **Background refetching**: Keep data fresh
- ✅ **Optimistic updates**: Instant UI feedback
- ✅ **Infinite scrolling**: Built-in `useInfiniteQuery`
- ✅ **DevTools**: See all queries/cache

**Advanced Example (with real-time updates):**
```tsx
function useRealtimeBlocks(chainId: number) {
  const queryClient = useQueryClient();
  
  // Poll API
  const { data } = useQuery({
    queryKey: ['blocks', chainId],
    queryFn: () => fetchBlocks(chainId),
    refetchInterval: 12000,
  });
  
  // WebSocket subscription for instant updates
  useEffect(() => {
    const ws = new WebSocket(`ws://api/chains/${chainId}/blocks`);
    ws.onmessage = (event) => {
      const newBlock = JSON.parse(event.data);
      queryClient.setQueryData(['blocks', chainId], (old: Block[]) => 
        [newBlock, ...old]
      );
    };
    return () => ws.close();
  }, [chainId]);
  
  return data;
}
```

---

### Web3 Integration: Wagmi + RainbowKit

**The Standard Stack for Wallet Connection:**

```tsx
// providers.tsx
import { WagmiConfig, createConfig } from 'wagmi';
import { mainnet, polygon, arbitrum } from 'wagmi/chains';
import { RainbowKitProvider } from '@rainbow-me/rainbowkit';

const config = createConfig({
  chains: [mainnet, polygon, arbitrum],
  transports: {
    [mainnet.id]: http(),
    [polygon.id]: http(),
    [arbitrum.id]: http(),
  },
});

export function Providers({ children }) {
  return (
    <WagmiConfig config={config}>
      <RainbowKitProvider>
        {children}
      </RainbowKitProvider>
    </WagmiConfig>
  );
}

// WalletButton.tsx
import { ConnectButton } from '@rainbow-me/rainbowkit';

export function WalletButton() {
  return <ConnectButton />;
}

// Read blockchain data
import { useBalance, useContractRead } from 'wagmi';

function UserBalance() {
  const { address } = useAccount();
  const { data: balance } = useBalance({ address });
  
  return <div>{balance?.formatted} ETH</div>;
}
```

---

### Complete Frontend Stack Recommendation

#### **For Block Explorer / Analytics (Our Project):**

```bash
Framework: React 18 + Vite + TypeScript
Styling: Tailwind CSS + ShadCN components
State: Zustand (global) + React Query (server state)
Charts: Recharts
Icons: Lucide React
Deployment: Vercel or Cloudflare Pages
```

**Why This Stack:**
- ⚡ Fastest development speed
- 🎨 Beautiful UI with minimal effort
- 📦 Small bundle (~150KB)
- 🔄 Real-time updates easy
- 💰 Free hosting (Vercel)

#### **For Public DEX / Protocol Frontend:**

```bash
Framework: Next.js 14 (App Router) + TypeScript
Styling: Tailwind CSS + ShadCN
State: Zustand + React Query
Web3: Wagmi + RainbowKit + viem
Analytics: Vercel Analytics
Deployment: Vercel Edge
```

**Why This Stack:**
- 🔍 SEO for landing pages
- 🌍 Global edge deployment
- 🔐 API routes for backend logic
- 📊 Built-in analytics
- 🚀 Used by Uniswap, Aave, etc.

---

### Component Library Recommendations

**1. ShadCN UI** ⭐ (Most Popular)
- Not installed via npm (copy/paste source)
- Full control + customization
- Beautiful by default
- Dark mode built-in

```bash
npx shadcn-ui@latest init
npx shadcn-ui@latest add button card dialog table
```

**2. DaisyUI** (Fast Prototyping)
- Tailwind component classes
- 30+ themes
- No JS, pure CSS

```bash
npm install daisyui
# Add to tailwind.config.js plugins
```

**3. HeadlessUI** (Accessible Primitives)
- Unstyled components
- Perfect for custom designs
- Made by Tailwind team

---

### Performance Optimization for Blockchain UIs

**1. Code Splitting:**
```tsx
// Lazy load heavy pages
const TransactionDetail = lazy(() => import('./pages/TransactionDetail'));

function App() {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <TransactionDetail />
    </Suspense>
  );
}
```

**2. Virtual Scrolling (for long lists):**
```tsx
import { useVirtualizer } from '@tanstack/react-virtual';

function TransactionList({ transactions }) {
  const virtualizer = useVirtualizer({
    count: transactions.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 60, // Height of each row
  });
  
  return (
    <div ref={parentRef} style={{ height: '600px', overflow: 'auto' }}>
      {virtualizer.getVirtualItems().map(virtualRow => (
        <div key={virtualRow.index} style={{
          height: `${virtualRow.size}px`,
          transform: `translateY(${virtualRow.start}px)`,
        }}>
          <TransactionRow tx={transactions[virtualRow.index]} />
        </div>
      ))}
    </div>
  );
}
```

**3. Memoization:**
```tsx
import { memo } from 'react';

const BlockCard = memo(function BlockCard({ block }: { block: Block }) {
  // Only re-renders if block prop changes
  return <div>{block.number}</div>;
});
```

**4. Debounce Search:**
```tsx
import { useDebouncedValue } from '@mantine/hooks';

function Search() {
  const [search, setSearch] = useState('');
  const [debounced] = useDebouncedValue(search, 500);
  
  const { data } = useQuery({
    queryKey: ['search', debounced],
    queryFn: () => searchTransactions(debounced),
    enabled: debounced.length > 3,
  });
}
```

---

### Interview Questions: Frontend for Blockchain

**Q: Why don't block explorers need SEO/SSR?**

**A:** "Block explorers are authenticated tools used by developers and traders who bookmark them directly or search for '[chain name] explorer'. They're not content sites competing for organic traffic. The pages are highly dynamic (new blocks every 12s) making SSR cache invalidation complex. Client-side rendering with React + Vite is simpler, faster to build, and delivers better UX with instant updates. We save on infrastructure costs (no Node.js server) and can deploy to CDN as static files."

**Q: How do you handle real-time block updates?**

**A:** "We use a hybrid approach:
1. **Polling** with React Query (`refetchInterval: 12000`) as the reliable baseline
2. **WebSocket** for instant updates when available
3. **Optimistic updates** - Add new block to UI immediately, reconcile with server later
4. **Background refetch** - Keep data fresh even when tab inactive

This ensures users see new blocks within 1-2 seconds while handling WebSocket disconnections gracefully."

**Q: How would you optimize rendering 10,000 transactions?**

**A:** "Multiple strategies:
1. **Virtual scrolling** - Only render visible rows (TanStack Virtual)
2. **Pagination** - Show 20 per page with cursor-based pagination
3. **Infinite scroll** - Load more as user scrolls (React Query's `useInfiniteQuery`)
4. **Memoization** - `memo()` components that don't need to re-render
5. **Web Workers** - Offload heavy computation (parsing big numbers, hex decoding)
6. **Debounced search** - Don't query on every keystroke

For 10K items, virtual scrolling + pagination is the standard solution."

---

## Production Readiness

### Deployment Checklist

- [ ] Database migrations tested in staging
- [ ] Environment variables documented
- [ ] Secrets in vault (not environment variables)
- [ ] Health check endpoints implemented
- [ ] Metrics exported to Prometheus
- [ ] Logs structured (JSON format)
- [ ] Distributed tracing enabled (Jaeger)
- [ ] Rate limiting configured
- [ ] CORS configured for API
- [ ] TLS/SSL certificates installed
- [ ] Backup strategy defined
- [ ] Disaster recovery plan documented
- [ ] Monitoring alerts configured
- [ ] On-call runbook created
- [ ] Load testing completed
- [ ] Security audit performed

### Monitoring Alerts

```yaml
alerts:
  - name: IngesterLagging
    condition: last_indexed_block < chain_head_block - 100
    severity: warning
    
  - name: ProcessorConsumerLag
    condition: kafka_consumer_lag > 10000
    severity: critical
    
  - name: HighAPILatency
    condition: http_request_duration_p95 > 1s
    severity: warning
    
  - name: DatabaseConnectionPoolExhausted
    condition: db_connections_available < 5
    severity: critical
    
  - name: HighErrorRate
    condition: error_rate > 1%
    severity: warning
```

### Capacity Planning

**Current Setup** (per chain):
- Ingester: 2 vCPU, 4GB RAM → ~100 blocks/sec
- Processor: 4 vCPU, 8GB RAM → ~200 blocks/sec
- API: 4 vCPU, 8GB RAM → ~1000 req/sec
- PostgreSQL: 8 vCPU, 32GB RAM → ~10K writes/sec
- Kafka: 4 vCPU, 16GB RAM → ~100K msg/sec

**Scaling Strategy**:
- Horizontal: Add more ingester/processor/API instances
- Vertical: Increase database resources first
- Sharding: Partition database by chain_id when single DB hits limits

---

## Additional Resources

### Official Documentation
- [go-ethereum docs](https://geth.ethereum.org/docs)
- [Kafka documentation](https://kafka.apache.org/documentation/)
- [PostgreSQL documentation](https://www.postgresql.org/docs/)

### Learning Resources
- [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)
- [System Design Interview](https://www.amazon.com/System-Design-Interview-insiders-Second/dp/B08CMF2CQF)
- [Designing Data-Intensive Applications](https://dataintensive.net/)

### Similar Projects
- [The Graph](https://thegraph.com/) - Decentralized indexing protocol
- [Etherscan](https://etherscan.io/) - Blockchain explorer
- [Dune Analytics](https://dune.com/) - Blockchain analytics platform

---

## Updates Log

| Date | Topic | Notes |
|------|-------|-------|
| 2025-11-14 | Initial setup | Created project structure, defined specs |
| | | |

---

_This document should be updated as we implement features and learn new concepts._
