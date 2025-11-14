# Technical Specification: High-Performance Blockchain Indexer

## System Overview

A distributed, event-driven blockchain indexer built with Rust/Go that ingests, processes, and serves blockchain data with sub-second latency, 99.9% uptime, and comprehensive observability.

---

## Architecture

### High-Level Components

```
┌──────────────────────────────────────────────────────────────────┐
│                      Monitoring & Observability                   │
│   Prometheus │ Grafana │ Jaeger │ AlertManager                   │
└──────────────────────────────────────────────────────────────────┘
                              ▲
                              │ OpenTelemetry
                              │
┌──────────────────────────────────────────────────────────────────┐
│                        Application Layer                          │
│                                                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐            │
│  │  Ingester   │  │  Processor  │  │  API Server  │            │
│  │  Service    │  │  Service    │  │              │            │
│  └─────────────┘  └─────────────┘  └──────────────┘            │
│         │                 │                 │                     │
│         └────────┬────────┴─────────────────┘                     │
│                  │                                                 │
└──────────────────┼─────────────────────────────────────────────────┘
                   │
┌──────────────────┼─────────────────────────────────────────────────┐
│                  │         Message Broker Layer                    │
│         ┌────────▼────────┐                                       │
│         │  Kafka/RabbitMQ │                                       │
│         │  - raw.blocks   │                                       │
│         │  - parsed.events│                                       │
│         │  - system.reorg │                                       │
│         └─────────────────┘                                       │
└──────────────────────────────────────────────────────────────────┘
                   │
┌──────────────────┼─────────────────────────────────────────────────┐
│                  │         Storage Layer                           │
│   ┌──────────────▼──────────┐   ┌──────────────┐   ┌──────────┐ │
│   │   PostgreSQL (Primary)   │   │ Redis Cache  │   │  S3/Blob │ │
│   │   - Partitioned tables   │   │ - Hot data   │   │ Archives │ │
│   │   - Indexed queries      │   │ - Query cache│   │          │ │
│   └──────────────────────────┘   └──────────────┘   └──────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

## Service Architecture

### 1. Ingester Service

**Responsibility**: Fetch blockchain data and publish to message broker

**Technology**: Rust (tokio, ethers-rs) or Go (go-ethereum client)

**Components**:
- **RPC Client**: Connection pool with multiple providers
- **WebSocket Subscriber**: Real-time block notifications
- **Reorg Detector**: Track parent hashes, detect chain reorganizations
- **Checkpoint Manager**: Persist last processed block number
- **Message Producer**: Publish to Kafka/RabbitMQ

**Key Algorithms**:

```rust
// Reorg Detection Algorithm
async fn detect_reorg(current_block: Block, db: &Database) -> Result<Option<u64>> {
    let stored_block = db.get_block(current_block.number)?;
    
    if stored_block.hash != current_block.parent_hash {
        // Reorg detected! Find common ancestor
        let mut rollback_to = current_block.number - 1;
        
        while rollback_to > 0 {
            let stored = db.get_block(rollback_to)?;
            let chain = rpc_client.get_block(rollback_to).await?;
            
            if stored.hash == chain.hash {
                return Ok(Some(rollback_to));
            }
            rollback_to -= 1;
        }
    }
    
    Ok(None)
}
```

**Configuration**:
```yaml
ingester:
  rpc_urls:
    - https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
    - https://mainnet.infura.io/v3/YOUR_KEY
  start_block: 18000000
  batch_size: 100
  checkpoint_interval: 10s
  retry:
    max_attempts: 5
    backoff_ms: 1000
    max_backoff_ms: 30000
```

**Performance**:
- **Target**: 100+ blocks/second ingestion
- **Optimization**: Batch RPC calls, parallel processing
- **Monitoring**: Ingestion lag, RPC latency, error rate

---

### 2. Processor Service

**Responsibility**: Consume messages, parse events, write to database

**Technology**: Rust (sqlx, rdkafka) or Go (gorm, sarama)

**Components**:
- **Message Consumers**: Kafka/RabbitMQ consumers with consumer groups
- **Event Parsers**: ERC20, ERC721, custom ABI decoders
- **Database Repository**: Optimized batch inserts
- **Cache Manager**: Redis integration for hot data

**Event Parsing**:

```rust
// ERC20 Transfer Event Parser
struct ERC20Parser {
    transfer_signature: H256, // keccak256("Transfer(address,address,uint256)")
}

impl ERC20Parser {
    async fn parse(&self, log: &Log) -> Result<Option<TokenTransfer>> {
        if log.topics[0] != self.transfer_signature {
            return Ok(None);
        }
        
        Ok(Some(TokenTransfer {
            contract_address: log.address,
            from: Address::from(log.topics[1]),
            to: Address::from(log.topics[2]),
            amount: U256::from_big_endian(&log.data),
            block_number: log.block_number,
            tx_hash: log.transaction_hash,
        }))
    }
}
```

**Database Operations**:
- **Batch Inserts**: Insert 1000s of records at once
- **Upserts**: Handle duplicate data gracefully
- **Transactions**: Atomic operations for reorg rollbacks

```sql
-- Optimized batch insert
INSERT INTO events (tx_hash, block_number, log_index, contract_address, event_signature, data)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tx_hash, log_index) DO NOTHING;
```

**Configuration**:
```yaml
processor:
  consumers:
    - topic: raw.blocks
      group_id: block_processor
      concurrency: 10
    - topic: parsed.events
      group_id: event_processor
      concurrency: 5
  database:
    max_connections: 50
    batch_size: 1000
  cache:
    redis_url: redis://localhost:6379
    ttl: 300s
```

---

### 3. API Service

**Responsibility**: Serve blockchain data via REST and WebSocket

**Technology**: Rust (axum, tower) or Go (gin, gorilla/websocket)

**Endpoints**:

```
# Multi-chain endpoints (chain_id in path or query param)
GET    /api/v1/:chain_id/blocks/:number           # Get block by number on specific chain
GET    /api/v1/:chain_id/blocks/:hash              # Get block by hash
GET    /api/v1/:chain_id/transactions/:hash        # Get transaction details
GET    /api/v1/:chain_id/events                    # Query events with filters
GET    /api/v1/:chain_id/addresses/:address/txs    # Get transactions for address
GET    /api/v1/:chain_id/addresses/:address/balance # Get token balances

# Cross-chain endpoints
GET    /api/v1/addresses/:address/txs              # Get txs across ALL chains
GET    /api/v1/addresses/:address/portfolio        # Portfolio across all chains
GET    /api/v1/chains                              # List supported chains
GET    /api/v1/chains/:chain_id/status             # Chain sync status

# System endpoints
GET    /api/v1/health                              # Health check (all chains)
GET    /api/v1/metrics                             # Prometheus metrics
WS     /api/v1/:chain_id/stream                    # Chain-specific WebSocket
WS     /api/v1/stream                              # Multi-chain WebSocket
```

**Query Parameters** (for `/events`):
- `contract_address`: Filter by contract
- `event_signature`: Filter by event type
- `from_block`, `to_block`: Block range
- `limit`, `offset`: Pagination
- `order`: `asc` or `desc`

**Example Response**:
```json
{
  "data": [
    {
      "tx_hash": "0x123...",
      "block_number": 18234567,
      "log_index": 42,
      "contract_address": "0xA0b...",
      "event_signature": "0xddf...",
      "decoded_data": {
        "from": "0x456...",
        "to": "0x789...",
        "value": "1000000000000000000"
      },
      "timestamp": "2025-11-14T12:34:56Z"
    }
  ],
  "pagination": {
    "limit": 100,
    "offset": 0,
    "total": 15234
  }
}
```

**Caching Strategy**:
1. **In-Memory**: Recent blocks (last 100)
2. **Redis**: Hot queries (5 min TTL)
3. **Database**: Full historical data

**Rate Limiting**:
```rust
// Token bucket algorithm
struct RateLimiter {
    tokens: u32,
    max_tokens: u32,
    refill_rate: Duration,
}

impl RateLimiter {
    async fn allow(&mut self) -> bool {
        self.refill();
        if self.tokens > 0 {
            self.tokens -= 1;
            true
        } else {
            false
        }
    }
}
```

**Configuration**:
```yaml
api:
  listen_addr: 0.0.0.0:8080
  rate_limit:
    requests_per_second: 100
    burst: 200
  cache:
    memory_size_mb: 256
    redis_ttl: 300s
  cors:
    allowed_origins: ["*"]
  tls:
    enabled: true
    cert_path: /certs/server.crt
    key_path: /certs/server.key
```

---

## Database Schema

### PostgreSQL Schema with Partitioning

```sql
-- Blocks table (partitioned by chain_id and block_number)
CREATE TABLE blocks (
    chain_id INT NOT NULL,              -- 1=Ethereum, 137=Polygon, 42161=Arbitrum, etc.
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    parent_hash VARCHAR(66) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    miner VARCHAR(42),
    gas_used BIGINT,
    gas_limit BIGINT,
    transaction_count INT,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chain_id, block_number),
    UNIQUE (chain_id, block_hash)
) PARTITION BY LIST (chain_id);  -- Partition by chain first

-- Create partitions (example: 100k blocks per partition)
CREATE TABLE blocks_18000000_18100000 PARTITION OF blocks
    FOR VALUES FROM (18000000) TO (18100000);

CREATE TABLE blocks_18100000_18200000 PARTITION OF blocks
    FOR VALUES FROM (18100000) TO (18200000);

-- Transactions table (partitioned by block_number)
CREATE TABLE transactions (
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    tx_index INT NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    value NUMERIC(78, 0),
    gas_price BIGINT,
    gas_used BIGINT,
    input BYTEA,
    status BOOLEAN,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (tx_hash, block_number)
) PARTITION BY RANGE (block_number);

-- Events table (partitioned by block_number)
CREATE TABLE events (
    id BIGSERIAL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    log_index INT NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    event_signature VARCHAR(66),
    topic1 VARCHAR(66),
    topic2 VARCHAR(66),
    topic3 VARCHAR(66),
    data BYTEA,
    decoded_data JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (id, block_number),
    UNIQUE (tx_hash, log_index, block_number)
) PARTITION BY RANGE (block_number);

-- Indexes for efficient queries
CREATE INDEX idx_transactions_from_block ON transactions(from_address, block_number DESC);
CREATE INDEX idx_transactions_to_block ON transactions(to_address, block_number DESC);
CREATE INDEX idx_events_contract_block ON events(contract_address, block_number DESC);
CREATE INDEX idx_events_signature_block ON events(event_signature, block_number DESC);
CREATE INDEX idx_events_decoded_data ON events USING GIN (decoded_data);

-- Checkpoint table (no partitioning needed)
CREATE TABLE checkpoints (
    service_name VARCHAR(50) PRIMARY KEY,
    last_processed_block BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Index Strategy
- **Primary Keys**: On frequently queried fields
- **Composite Indexes**: For multi-column filters (address + block_number)
- **GIN Indexes**: For JSONB queries (decoded_data)
- **Partial Indexes**: For specific query patterns

---

## Message Broker Design

### Kafka Topics

```yaml
topics:
  raw.blocks:
    partitions: 10
    replication_factor: 3
    retention_ms: 86400000  # 1 day
    
  raw.transactions:
    partitions: 20
    replication_factor: 3
    retention_ms: 86400000
    
  parsed.events:
    partitions: 20
    replication_factor: 3
    retention_ms: 604800000  # 7 days
    
  system.reorg:
    partitions: 1
    replication_factor: 3
    retention_ms: 2592000000  # 30 days
```

### Message Schema

```protobuf
// Protocol Buffers schema for efficient serialization
message Block {
  uint64 block_number = 1;
  string block_hash = 2;
  string parent_hash = 3;
  int64 timestamp = 4;
  string miner = 5;
  uint64 gas_used = 6;
  uint64 gas_limit = 7;
  repeated Transaction transactions = 8;
}

message Transaction {
  string tx_hash = 1;
  uint64 block_number = 2;
  string from_address = 3;
  string to_address = 4;
  string value = 5;
  bytes input = 6;
  repeated Log logs = 7;
}

message Log {
  uint32 log_index = 1;
  string contract_address = 2;
  repeated string topics = 3;
  bytes data = 4;
}

message ReorgEvent {
  uint64 rollback_to_block = 1;
  uint64 detected_at_block = 2;
  int64 timestamp = 3;
}
```

---

## Observability

### Metrics (Prometheus)

**Ingester Metrics**:
```
# Ingestion rate
indexer_blocks_ingested_total
indexer_blocks_ingested_per_second

# Lag
indexer_block_lag_seconds
indexer_block_lag_count

# Errors
indexer_rpc_errors_total
indexer_reorg_detected_total

# RPC latency
indexer_rpc_request_duration_seconds{quantile="0.5"}
indexer_rpc_request_duration_seconds{quantile="0.95"}
indexer_rpc_request_duration_seconds{quantile="0.99"}
```

**Processor Metrics**:
```
# Processing rate
indexer_events_processed_total
indexer_events_processed_per_second

# Queue depth
indexer_message_queue_depth

# Database latency
indexer_db_query_duration_seconds{operation="insert"}
indexer_db_query_duration_seconds{operation="select"}

# Cache performance
indexer_cache_hit_total
indexer_cache_miss_total
```

**API Metrics**:
```
# Request rate
indexer_api_requests_total{method="GET", endpoint="/blocks"}
indexer_api_requests_per_second

# Response time
indexer_api_request_duration_seconds{quantile="0.5"}
indexer_api_request_duration_seconds{quantile="0.95"}
indexer_api_request_duration_seconds{quantile="0.99"}

# Error rate
indexer_api_errors_total{status_code="500"}

# Cache
indexer_api_cache_hit_rate
```

### Distributed Tracing (OpenTelemetry)

**Trace Spans**:
1. `ingester.fetch_block` → RPC call to fetch block
2. `ingester.publish_message` → Publish to Kafka
3. `processor.consume_message` → Consume from Kafka
4. `processor.parse_event` → Event parsing
5. `processor.db_insert` → Database write
6. `api.handle_request` → API request handling
7. `api.db_query` → Database query
8. `api.cache_lookup` → Cache lookup

**Example Trace**:
```
[ingester.fetch_block] (100ms)
  └─ [rpc.eth_getBlockByNumber] (80ms)
  └─ [ingester.publish_message] (20ms)
     └─ [kafka.produce] (15ms)

[processor.consume_message] (150ms)
  └─ [processor.parse_event] (10ms)
  └─ [processor.db_insert] (130ms)
     └─ [postgres.batch_insert] (120ms)

[api.handle_request] (250ms)
  └─ [api.cache_lookup] (5ms) [MISS]
  └─ [api.db_query] (200ms)
     └─ [postgres.select] (190ms)
  └─ [api.serialize_response] (10ms)
```

### Grafana Dashboards

**Dashboard 1: Ingestion Overview**
- Current block height vs. chain head
- Ingestion rate (blocks/sec, txs/sec)
- RPC latency (p50, p95, p99)
- Reorg events timeline
- Error rate by service

**Dashboard 2: Processing Performance**
- Message queue depth
- Processing latency per event type
- Database write latency
- Failed message count
- Consumer lag

**Dashboard 3: API Performance**
- Requests per second by endpoint
- Response time distribution
- Error rate by status code
- Cache hit rate
- Active WebSocket connections

**Dashboard 4: System Health**
- CPU and memory usage per service
- Database connections pool
- Disk I/O and storage usage
- Network throughput
- Uptime percentage

---

## Fault Tolerance

### Retry Mechanisms

**Exponential Backoff with Jitter**:
```rust
async fn retry_with_backoff<F, T>(
    operation: F,
    max_attempts: u32,
    base_delay_ms: u64,
) -> Result<T>
where
    F: Fn() -> Future<Output = Result<T>>,
{
    let mut attempt = 0;
    loop {
        match operation().await {
            Ok(result) => return Ok(result),
            Err(e) if attempt >= max_attempts => return Err(e),
            Err(_) => {
                let delay = calculate_backoff(attempt, base_delay_ms);
                tokio::time::sleep(delay).await;
                attempt += 1;
            }
        }
    }
}

fn calculate_backoff(attempt: u32, base_ms: u64) -> Duration {
    let exponential = base_ms * 2_u64.pow(attempt);
    let jitter = rand::random::<u64>() % (base_ms / 2);
    Duration::from_millis(exponential + jitter)
}
```

### Circuit Breaker

```rust
struct CircuitBreaker {
    state: Mutex<CircuitState>,
    failure_threshold: u32,
    timeout: Duration,
}

enum CircuitState {
    Closed { failures: u32 },
    Open { opened_at: Instant },
    HalfOpen,
}

impl CircuitBreaker {
    async fn call<F, T>(&self, operation: F) -> Result<T>
    where
        F: Future<Output = Result<T>>,
    {
        let state = self.state.lock().await.clone();
        
        match state {
            CircuitState::Open { opened_at } => {
                if opened_at.elapsed() > self.timeout {
                    *self.state.lock().await = CircuitState::HalfOpen;
                    self.try_operation(operation).await
                } else {
                    Err(Error::CircuitOpen)
                }
            }
            CircuitState::Closed { .. } | CircuitState::HalfOpen => {
                self.try_operation(operation).await
            }
        }
    }
}
```

### Graceful Shutdown

```rust
async fn graceful_shutdown(services: Vec<Service>) {
    // 1. Stop accepting new requests
    api_server.stop_accepting().await;
    
    // 2. Wait for in-flight requests to complete (max 30s)
    tokio::select! {
        _ = api_server.wait_for_completion() => {},
        _ = tokio::time::sleep(Duration::from_secs(30)) => {
            warn!("Forcefully shutting down API server");
        }
    }
    
    // 3. Stop consumers (finish processing current messages)
    for consumer in consumers {
        consumer.stop().await;
    }
    
    // 4. Close database connections
    db_pool.close().await;
    
    // 5. Flush remaining metrics
    metrics_exporter.flush().await;
}
```

---

## Performance Optimization

### Database Optimizations

1. **Connection Pooling**:
```rust
let pool = PgPoolOptions::new()
    .max_connections(50)
    .min_connections(10)
    .acquire_timeout(Duration::from_secs(5))
    .idle_timeout(Duration::from_secs(600))
    .connect(&db_url).await?;
```

2. **Batch Operations**:
```rust
// Instead of individual inserts
for event in events {
    db.insert_event(event).await?; // ❌ Slow
}

// Use batch insert
db.insert_events_batch(events).await?; // ✅ Fast
```

3. **Prepared Statements**:
```rust
let stmt = pool.prepare("SELECT * FROM events WHERE contract_address = $1 AND block_number >= $2").await?;
let events = stmt.query(&[&address, &block]).await?;
```

4. **Partitioning Strategy**:
- Create partitions for every 100k blocks
- Automate partition creation with a cron job
- Drop old partitions after archival to S3

### Caching Strategy

**Multi-Tier Cache**:
1. **L1 (In-Memory)**: Recent blocks (last 100), hot addresses
2. **L2 (Redis)**: Query results (5 min TTL), address balances (1 min TTL)
3. **L3 (Database)**: Full historical data

```rust
async fn get_block(block_number: u64, cache: &Cache, db: &Database) -> Result<Block> {
    // Try L1 cache
    if let Some(block) = cache.memory.get(&block_number) {
        return Ok(block);
    }
    
    // Try L2 cache
    if let Some(block) = cache.redis.get(&block_number).await? {
        cache.memory.insert(block_number, block.clone());
        return Ok(block);
    }
    
    // Fallback to database
    let block = db.get_block(block_number).await?;
    cache.redis.set(&block_number, &block, Duration::from_secs(300)).await?;
    cache.memory.insert(block_number, block.clone());
    
    Ok(block)
}
```

### Async I/O

Use async/await for all I/O operations:
```rust
// ❌ Blocking
let block1 = rpc.get_block(1000000)?;
let block2 = rpc.get_block(1000001)?;

// ✅ Concurrent
let (block1, block2) = tokio::join!(
    rpc.get_block(1000000),
    rpc.get_block(1000001),
);
```

---

## Security

### API Authentication

**JWT Tokens**:
```rust
#[derive(Serialize, Deserialize)]
struct Claims {
    sub: String,      // User ID
    exp: u64,         // Expiration
    rate_limit: u32,  // Requests per second
}

async fn verify_token(token: &str) -> Result<Claims> {
    let validation = Validation::default();
    let token_data = decode::<Claims>(token, &DECODING_KEY, &validation)?;
    Ok(token_data.claims)
}
```

### Input Validation

```rust
#[derive(Deserialize, Validate)]
struct QueryParams {
    #[validate(range(min = 0, max = 20_000_000))]
    from_block: u64,
    
    #[validate(range(min = 0, max = 20_000_000))]
    to_block: u64,
    
    #[validate(length(equal = 42))]
    contract_address: Option<String>,
    
    #[validate(range(min = 1, max = 1000))]
    limit: u32,
}
```

### Rate Limiting

**Token Bucket per IP/API Key**:
```rust
let rate_limiter = governor::RateLimiter::keyed(
    governor::Quota::per_second(nonzero!(100u32))
);

async fn rate_limit_middleware(
    req: Request,
    limiter: &RateLimiter,
) -> Result<Response> {
    let key = extract_api_key(&req)?;
    
    if limiter.check_key(&key).is_err() {
        return Err(Error::RateLimitExceeded);
    }
    
    Ok(next(req).await)
}
```

---

## Deployment

### Docker Compose (Development)

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: indexer
      POSTGRES_USER: indexer
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7
    ports:
      - "6379:6379"

  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"

  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181

  ingester:
    build: ./services/ingester
    environment:
      RPC_URL: https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
      KAFKA_BROKERS: kafka:9092
      DATABASE_URL: postgres://indexer:password@postgres:5432/indexer
    depends_on:
      - postgres
      - kafka

  processor:
    build: ./services/processor
    environment:
      KAFKA_BROKERS: kafka:9092
      DATABASE_URL: postgres://indexer:password@postgres:5432/indexer
      REDIS_URL: redis://redis:6379
    depends_on:
      - postgres
      - kafka
      - redis

  api:
    build: ./services/api
    environment:
      DATABASE_URL: postgres://indexer:password@postgres:5432/indexer
      REDIS_URL: redis://redis:6379
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./monitoring/grafana/datasources.yml:/etc/grafana/provisioning/datasources/datasources.yml

volumes:
  postgres_data:
```

### Kubernetes (Production)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ingester
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ingester
  template:
    metadata:
      labels:
        app: ingester
    spec:
      containers:
      - name: ingester
        image: indexer/ingester:latest
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        env:
        - name: RPC_URL
          valueFrom:
            secretKeyRef:
              name: indexer-secrets
              key: rpc_url
        - name: KAFKA_BROKERS
          value: "kafka:9092"
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 5
```

---

## Testing Strategy

### Unit Tests
- Test individual functions and modules
- Mock external dependencies (RPC, database, message broker)
- Target: >90% code coverage

### Integration Tests
- Test service interactions (ingester → Kafka → processor → database)
- Use testcontainers for real dependencies
- Test reorg scenarios end-to-end

### Load Tests
- Simulate high ingestion rate (1000 blocks/sec)
- Stress test API endpoints (10k req/sec)
- Measure latency under load

```rust
#[tokio::test]
async fn test_reorg_handling() {
    let mut db = MockDatabase::new();
    let mut rpc = MockRPC::new();
    
    // Setup: Initial chain
    db.insert_block(Block { number: 100, hash: "0xaaa", parent: "0x999" });
    db.insert_block(Block { number: 101, hash: "0xbbb", parent: "0xaaa" });
    
    // Simulate reorg: block 101 has different hash
    rpc.set_block(101, Block { number: 101, hash: "0xccc", parent: "0xaaa" });
    
    let ingester = Ingester::new(rpc, db);
    ingester.sync_block(101).await.unwrap();
    
    // Assert: Reorg detected and handled
    assert_eq!(db.get_block(101).hash, "0xccc");
    assert_eq!(ingester.reorg_count(), 1);
}
```

---

## Conclusion

This technical specification provides a comprehensive blueprint for building a **production-grade blockchain indexer** with:

- **High performance**: 100+ blocks/sec ingestion, <500ms API latency
- **Fault tolerance**: Reorg handling, retries, circuit breakers
- **Scalability**: Event-driven architecture, horizontal scaling
- **Observability**: Metrics, traces, dashboards, SLOs

The system is designed with **senior engineering best practices**:
- Clean abstractions and separation of concerns
- Comprehensive error handling and logging
- Test-driven development with high coverage
- Infrastructure as code (Docker, Kubernetes)
- Security-first approach (auth, rate limiting, validation)

**Ready for production deployment** with all the tooling needed for operations, monitoring, and continuous improvement.
