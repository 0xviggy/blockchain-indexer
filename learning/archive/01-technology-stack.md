# Technology Stack Deep Dive

> **Purpose**: Understanding the technologies powering our blockchain indexer and their architectural significance.

---

## 🎯 Backend Technologies

### Go Programming Language

**Why Go**: Chosen for blockchain indexing due to excellent concurrency primitives, strong blockchain ecosystem (go-ethereum), and faster development velocity compared to Rust.

**Key Features Used**:
- **Goroutines**: Lightweight threads for parallel chain processing (one per blockchain)
- **Channels**: Safe communication between goroutines for block data
- **Context**: Graceful shutdown and timeout handling across service boundaries
- **go-ethereum library**: Official Ethereum client for RPC/WebSocket communication

**Example - Parallel Chain Processing**:
```go
// Start one goroutine per chain
for _, chain := range chains {
    go func(c Chain) {
        processChain(ctx, c.ChainID, c.RPCURL)
    }(chain)
}

// Wait for shutdown signal
<-ctx.Done()
```

### PostgreSQL Database

**Why PostgreSQL**: ACID guarantees, mature partitioning support, excellent query planner, and wide operational knowledge.

**Key Features Used**:
- **Table Partitioning**: Partition by `chain_id` for query isolation and parallel writes
- **JSONB**: Store flexible event data and parsed calldata
- **Triggers**: Automatic timestamp updates on row changes
- **Views**: Pre-computed analytics queries for API performance
- **Foreign Keys**: Data integrity across chains, blocks, transactions

**Partitioning Strategy**:
```sql
-- Parent table definition
CREATE TABLE blocks (
    chain_id INT NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash BYTEA NOT NULL,
    parent_hash BYTEA NOT NULL,
    timestamp BIGINT NOT NULL,
    gas_used BIGINT,
    gas_limit BIGINT,
    PRIMARY KEY (chain_id, block_number)
) PARTITION BY LIST (chain_id);

-- Create partition per chain
CREATE TABLE blocks_eth PARTITION OF blocks 
    FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks 
    FOR VALUES IN (137);
```

**Why Partitioning Works**:
1. Most queries filter by single chain: `WHERE chain_id = 1`
2. PostgreSQL only scans relevant partition (3x faster)
3. Parallel writes: Each chain writes to different partition (no lock contention)
4. Easy maintenance: Drop old chain data without affecting others

### Redis Cache

**Why Redis**: Sub-millisecond latency for frequently accessed data, built-in data structures, pub/sub for real-time updates.

**Key Features Used**:
- **String Cache**: Store serialized blocks/transactions with TTL
- **Hash Sets**: Cache API responses by endpoint+params
- **Sorted Sets**: Rate limiting with sliding window
- **Pub/Sub**: Real-time notifications for new blocks (future)

**Caching Strategy**:
```go
// Cache block with 12-second TTL (1 block time)
key := fmt.Sprintf("block:%d:%s", chainID, blockHash)
data, _ := json.Marshal(block)
rdb.Set(ctx, key, data, 12*time.Second)

// Try cache first, fallback to DB
if cached, err := rdb.Get(ctx, key).Result(); err == nil {
    json.Unmarshal([]byte(cached), &block)
    return block, nil
}
block = fetchFromDB(chainID, blockHash)
```

### Kafka (Redpanda)

**Why Kafka**: Decouples ingestion from processing, replay capability, scales horizontally, industry standard for event streaming.

**Why Redpanda**: Kafka-compatible but written in C++ (faster), no ZooKeeper dependency (simpler ops), single binary deployment.

**Key Features Used**:
- **Topics**: Separate streams for raw blocks, parsed events, reorgs
- **Partitioning**: Partition by chain_id for parallel processing
- **Consumer Groups**: Multiple processor instances for scaling
- **Retention**: 7-day retention for replay/debugging

**Topic Design**:
```yaml
topics:
  raw.blocks:
    partitions: 5  # One per chain
    partition_key: chain_id
    retention: 168h  # 7 days
    
  parsed.events:
    partitions: 10  # More for higher throughput
    partition_key: chain_id
    retention: 168h
    
  system.reorg:
    partitions: 1  # Order matters for reorgs
    retention: 720h  # 30 days (important for auditing)
```

---

## 🎨 Frontend Technologies

### React 18

**Why React**: Component-based architecture, huge ecosystem, excellent TypeScript support, easy to find developers.

**Key Features Used**:
- **Hooks**: useState, useEffect for component state
- **Context**: Share selected chain across components
- **Suspense**: Loading states for async data
- **Error Boundaries**: Graceful error handling

### TypeScript

**Why TypeScript**: Catch bugs at compile time, better IDE support, safer refactoring, self-documenting code.

**Key Features Used**:
- **Interfaces**: Define API response shapes
- **Generics**: Type-safe API client functions
- **Enums**: Chain IDs, transaction status
- **Type Guards**: Runtime type checking

**Example - Type-Safe API Client**:
```typescript
interface Block {
    chain_id: number;
    block_number: number;
    block_hash: string;
    timestamp: number;
    transaction_count: number;
}

async function getBlocks(chainId: number, limit: number): Promise<Block[]> {
    const response = await axios.get<Block[]>(
        `/api/v1/blocks?chain_id=${chainId}&limit=${limit}`
    );
    return response.data;
}
```

### Vite

**Why Vite**: Instant HMR (hot module replacement), faster builds than Webpack, simpler configuration, ESM-native.

**How It Works**:
- Development: Serves source files as ES modules (no bundling)
- Production: Rollup bundling with code splitting
- Plugins: React plugin for JSX transform, TypeScript support

### React Query (TanStack Query)

**Why React Query**: Manages server state lifecycle (loading, caching, refetching), automatic background updates, optimistic UI.

**Key Features Used**:
```typescript
// Automatic caching and refetching
const { data: blocks, isLoading, error } = useQuery({
    queryKey: ['blocks', chainId, limit],
    queryFn: () => api.getBlocks(chainId, limit),
    staleTime: 12000,  // 12 seconds (1 block time)
    refetchInterval: 12000  // Poll for new blocks
});

// Mutation with optimistic updates
const mutation = useMutation({
    mutationFn: api.updateChain,
    onMutate: async (newChain) => {
        // Optimistically update UI
        queryClient.setQueryData(['chains'], (old) => 
            old.map(c => c.id === newChain.id ? newChain : c)
        );
    }
});
```

### Zustand

**Why Zustand**: Simpler than Redux (no actions/reducers), TypeScript-first, minimal boilerplate, devtools support.

**Key Features Used**:
```typescript
interface ChainStore {
    selectedChainId: number;
    setChain: (id: number) => void;
    chains: Chain[];
    loadChains: () => Promise<void>;
}

const useChainStore = create<ChainStore>((set) => ({
    selectedChainId: 1,
    chains: [],
    setChain: (id) => set({ selectedChainId: id }),
    loadChains: async () => {
        const chains = await api.getChains();
        set({ chains });
    }
}));

// Usage in component
function ChainSwitcher() {
    const { selectedChainId, setChain, chains } = useChainStore();
    return (
        <select value={selectedChainId} onChange={e => setChain(+e.target.value)}>
            {chains.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
    );
}
```

### Tailwind CSS

**Why Tailwind**: Utility-first reduces CSS bloat, dark mode built-in, responsive by default, component libraries available.

**Key Features Used**:
```tsx
// Responsive + dark mode
<div className="
    bg-white dark:bg-gray-900
    p-4 md:p-6 lg:p-8
    rounded-lg shadow-md
    hover:shadow-xl transition-shadow
">
    <h2 className="text-xl md:text-2xl font-bold text-gray-900 dark:text-white">
        Block #{block.number}
    </h2>
</div>
```

---

## ⛓️ Blockchain Technologies

### go-ethereum (geth)

**Why go-ethereum**: Official Go implementation, most stable RPC client, used by node operators, extensive documentation.

**Key Features Used**:
- **RPC Client**: HTTP/WebSocket connections to nodes
- **Type Definitions**: Block, Transaction, Receipt structs
- **ABI Encoding**: Parse smart contract calls
- **Crypto Utilities**: Keccak256 hashing, address checksums

**Example - WebSocket Subscription**:
```go
import (
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/core/types"
)

// Connect via WebSocket
client, err := ethclient.Dial("wss://eth-mainnet.g.alchemy.com/v2/KEY")

// Subscribe to new block headers
headers := make(chan *types.Header)
sub, err := client.SubscribeNewHead(ctx, headers)

for {
    select {
    case header := <-headers:
        // Process new block
        processBlock(header.Number.Uint64())
    case err := <-sub.Err():
        // Reconnect on error
        log.Printf("Subscription error: %v", err)
    }
}
```

### Multi-Chain Support

**EVM Chains Supported**: Ethereum, Polygon, Arbitrum, Optimism, Base

**Why These Chains**:
- All EVM-compatible (same RPC interface)
- Combined >$50B TVL
- Cover L1 (Ethereum) and major L2s
- Different characteristics: Polygon (gaming/NFTs), Arbitrum (DeFi), Base (social)

**Chain Differences**:
```go
type ChainConfig struct {
    ChainID         int
    Name            string
    BlockTime       int  // seconds
    Confirmations   int  // blocks for finality
}

var chains = []ChainConfig{
    {1, "Ethereum", 12, 15},      // Slower, higher security
    {137, "Polygon", 2, 128},     // Fast, needs more confirmations
    {42161, "Arbitrum", 0.25, 1}, // Very fast, optimistic rollup
    {10, "Optimism", 2, 1},       // Fast, optimistic rollup
    {8453, "Base", 2, 1},         // Fast, optimistic rollup
}
```

---

## 🏗️ Infrastructure & DevOps

### Docker Compose

**Why Docker Compose**: Simple local development, multi-container orchestration, easy cleanup, matches production Kubernetes.

**Key Features Used**:
```yaml
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: indexer
      POSTGRES_USER: indexer
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: pg_isready -U indexer
      interval: 10s
    networks:
      - indexer-network

networks:
  indexer-network:
    driver: bridge

volumes:
  postgres_data:
    driver: local
```

**Docker Compose vs Kubernetes**:
- **Development**: Docker Compose (simpler, faster iteration)
- **Production**: Kubernetes (scaling, self-healing, rolling updates)
- **CI/CD**: Docker Compose (test environment)

### Makefile

**Why Makefile**: Universal task runner, self-documenting commands, shell integration, no additional dependencies.

**Key Targets**:
```makefile
.PHONY: docker-up migrate run-ingester run-api

# Start infrastructure
docker-up:
	docker compose -f infrastructure/docker/docker-compose.yml up -d

# Run migrations
migrate:
	docker exec -i indexer-postgres psql -U indexer -d indexer < database/migrations/001_initial_schema.sql
	docker exec -i indexer-postgres psql -U indexer -d indexer < database/migrations/002_add_calldata_parsing.sql

# Start services
run-ingester:
	cd services/ingester && go run main.go

run-api:
	cd services/api && go run main.go

# Combined setup
setup: docker-up
	@echo "Waiting for PostgreSQL..."
	@sleep 5
	$(MAKE) migrate
	@echo "Setup complete!"
```

---

## 🏛️ Architecture Patterns

### Microservices

**Why Microservices**: Independent scaling, isolated failures, technology flexibility, easier reasoning about each service.

**Our Services**:
1. **Ingester**: Fetches blocks from RPC nodes
2. **Processor**: Parses events and calldata (deferred for MVP)
3. **API**: Serves data to frontend

**Service Boundaries**:
```
Ingester responsibilities:
- Connect to blockchain nodes
- Fetch blocks and transactions
- Handle WebSocket subscriptions
- Checkpoint management
- Write to PostgreSQL (MVP) or Kafka (future)

Processor responsibilities (future):
- Read from Kafka
- Parse event logs
- Decode function calls
- Extract internal transactions
- Write parsed data to PostgreSQL

API responsibilities:
- Read from PostgreSQL
- Cache with Redis
- Rate limiting
- Serve REST endpoints
- WebSocket for real-time updates
```

### Event-Driven Architecture

**Why Event-Driven**: Decoupling, async processing, replay capability, easier to add new consumers.

**Message Flow** (future with Kafka):
```
1. Ingester fetches block → publishes to raw.blocks topic
2. Processor consumes raw.blocks → parses events → publishes to parsed.events
3. Analytics service consumes parsed.events → updates dashboards
4. Alert service consumes system.reorg → notifies on-call
```

**Event Schema Example**:
```json
{
  "event_type": "block.new",
  "timestamp": 1700000000,
  "chain_id": 1,
  "data": {
    "block_number": 18500000,
    "block_hash": "0xabc123...",
    "transaction_count": 157
  }
}
```

### CQRS (Command Query Responsibility Segregation)

**Why CQRS**: Optimize writes and reads separately, simpler code, better performance.

**In Our System**:
- **Command (Write)**: Ingester writes blocks sequentially, atomic transactions
- **Query (Read)**: API reads with caching, pagination, multiple indexes

```
Write Model (Ingester):
- Optimized for sequential inserts
- Minimal indexes (only primary key)
- Writes to partitioned tables

Read Model (API):
- Optimized for various query patterns
- Multiple indexes (hash, address, timestamp)
- Materialized views for analytics
- Redis cache for hot data
```

### Graceful Degradation

**Why**: System should continue working even when components fail.

**Our Implementations**:
1. **WebSocket → HTTP Polling**: If WebSocket fails, fall back to polling
2. **Cache Miss → Database**: If Redis is down, query PostgreSQL
3. **Multi-RPC**: Try backup RPC if primary fails

```go
// WebSocket with HTTP fallback
if wsClient, err := dialWebSocket(url); err == nil {
    subscribeToBlocks(wsClient)
} else {
    log.Printf("WebSocket failed, using HTTP polling: %v", err)
    pollBlocks(httpClient, 12*time.Second)
}
```

### Checkpointing

**Why**: Resume processing from last successful point after restart/failure.

**Implementation**:
```sql
CREATE TABLE checkpoints (
    service_name VARCHAR(50) NOT NULL,
    chain_id INT NOT NULL,
    last_block BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (service_name, chain_id)
);
```

```go
// Load checkpoint on startup
var lastBlock uint64
err := db.QueryRow(`
    SELECT last_block FROM checkpoints 
    WHERE service_name = 'ingester' AND chain_id = $1
`, chainID).Scan(&lastBlock)

if err == sql.ErrNoRows {
    lastBlock = genesisBlock  // Start from beginning
}

// Update checkpoint after each block
tx.Exec(`
    INSERT INTO checkpoints (service_name, chain_id, last_block)
    VALUES ('ingester', $1, $2)
    ON CONFLICT (service_name, chain_id)
    DO UPDATE SET last_block = $2, updated_at = CURRENT_TIMESTAMP
`, chainID, blockNumber)
```

---

**Related Documents**:
- [Docker & Kubernetes Guide](./02-docker-kubernetes.md)
- [Database & Messaging Patterns](./03-databases-messaging.md)
- [Go Programming Guide](./04-go-programming.md)
