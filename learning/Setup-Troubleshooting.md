# Troubleshooting & Performance Guide

> ⚠️ **EDUCATIONAL MATERIAL** - Interview prep & real-world debugging reference  
> For project setup, see [../SANDBOX_SETUP.md](../SANDBOX_SETUP.md)

> **Purpose**: Troubleshooting common issues, performance optimization, debugging techniques, and real-world case studies with integrated interview Q&A.

**Last Updated**: December 20, 2025

---

## Table of Contents

- [Quick Reference Commands](#quick-reference-commands)
- [Part 1: Troubleshooting Common Issues](#part-1-troubleshooting-common-issues)
  - [Ingester Lag](#issue-ingester-falling-behind-chain-head)
  - [Consumer Lag](#issue-processor-consumer-lag)
  - [API Latency](#issue-api-slow-response-times)
  - [Docker & Port Issues](#docker--port-issues)
- [Part 2: Performance Optimization](#part-2-performance-optimization)
  - [Database Tuning](#database-optimizations)
  - [Go Performance Tips](#go-performance-tips)
  - [Kafka & Redis Tuning](#kafka--redis-optimizations)
- [Part 3: Real-World Case Studies](#part-3-real-world-case-studies)
  - [Database Batch INSERT (300x Speedup)](#case-study-1-database-batch-insert-300x-speedup)
  - [Receipt Fetch Retry Logic](#case-study-2-receipt-fetch-retry-logic--silent-failures)
  - [React State Management Bug](#case-study-3-react-state-management---default-parameter-bug)
  - [Multiple Process Detection](#case-study-4-multiple-process-detection--cleanup)
- [Part 4: Monitoring & Debugging](#part-4-monitoring--debugging)
  - [Metrics to Monitor](#metrics-to-monitor)
  - [Debugging Commands](#debugging-commands)
- [Interview Questions & Answers](#interview-questions--answers)

---

## Quick Reference Commands

```bash
# Check if containers are running
make status

# View logs from all containers
docker compose -f infrastructure/docker/docker-compose.yml logs -f

# Open PostgreSQL CLI
make db-shell

# Check Redis connection
docker exec -it indexer-redis redis-cli PING

# Web UIs for Debugging
# - Kafka UI: http://localhost:8080
# - pgAdmin: http://localhost:5050 (admin@indexer.local / admin)
```

---

## Part 1: Troubleshooting Common Issues

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

1. **Add more RPC providers** (load balance)
   ```yaml
   # config.yaml
   chains:
     - id: 1
       rpc_urls:
         - https://eth-mainnet.infura.io/v3/YOUR_KEY
         - https://eth-mainnet.alchemyapi.io/v2/YOUR_KEY
         - https://rpc.ankr.com/eth
   ```

2. **Increase batch size** in config
   ```yaml
   ingester:
     batch_size: 10  # Fetch 10 blocks at once
   ```

3. **Scale horizontally** - Multiple ingesters with block range sharding
   ```
   Ingester 1: blocks 0-1M
   Ingester 2: blocks 1M-2M
   Ingester 3: blocks 2M+ (real-time)
   ```

4. **Check network connectivity**
   ```bash
   curl -X POST https://eth-mainnet.infura.io/v3/YOUR_KEY \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
   ```

### Issue: Processor consumer lag

**Symptoms**: Kafka consumer lag increasing

**Diagnosis**:
```bash
# Check consumer group lag
kafka-consumer-groups --bootstrap-server localhost:9092 \
    --describe --group block_processor

# Output:
# TOPIC     PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG
# blocks    0          1000            5000            4000  # 4000 messages behind!
```

**Solutions**:

1. **Increase processor concurrency**
   ```go
   // Process multiple messages in parallel
   for i := 0; i < 10; i++ {
       go func() {
           for msg := range messages {
               processMessage(msg)
           }
       }()
   }
   ```

2. **Optimize database batch inserts**
   ```go
   // Instead of 1 insert per message
   INSERT INTO events VALUES ($1, $2, $3);
   
   // Batch 100 inserts
   INSERT INTO events VALUES 
       ($1, $2, $3),
       ($4, $5, $6),
       ...
       ($298, $299, $300);  // 100 rows at once
   ```

3. **Add database indexes**
   ```sql
   -- Find missing indexes
   SELECT schemaname, tablename, indexname, idx_scan
   FROM pg_stat_user_indexes
   WHERE idx_scan = 0;  -- Unused indexes (maybe wrong ones?)
   
   -- Add missing indexes
   CREATE INDEX idx_events_contract_addr ON events (contract_address);
   ```

4. **Scale processor horizontally**
   ```bash
   # Kafka will rebalance partitions across consumers
   docker-compose up --scale processor=3
   ```

5. **Check database connection pool**
   ```go
   db.SetMaxOpenConns(25)  // Increase from 10
   db.SetMaxIdleConns(5)
   db.SetConnMaxLifetime(5 * time.Minute)
   ```

### Issue: API slow response times

**Symptoms**: P95 latency >1 second

**Diagnosis**:

```bash
# 1. Check slow queries
psql -U indexer -d indexer -c "
SELECT 
    query,
    mean_exec_time,
    calls 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC 
LIMIT 10;"

# 2. Check cache hit rate
redis-cli INFO stats | grep hit_rate

# 3. Check API server metrics
curl http://localhost:9090/metrics | grep http_request_duration
```

**Solutions**:

1. **Add missing database indexes**
   ```sql
   -- Check which queries are doing sequential scans
   SELECT 
       schemaname,
       tablename,
       seq_scan,
       seq_tup_read,
       idx_scan,
       seq_tup_read / seq_scan as avg_seq_tuples
   FROM pg_stat_user_tables
   WHERE seq_scan > 0
   ORDER BY seq_tup_read DESC;
   
   -- Add index for slow query
   CREATE INDEX idx_txs_from_block ON transactions (from_address, block_number DESC);
   ```

2. **Increase Redis cache TTL**
   ```go
   // Cache blocks for 5 minutes instead of 1 minute
   redis.Set(ctx, key, value, 5*time.Minute)
   ```

3. **Enable query result caching**
   ```go
   // Cache expensive queries
   func (api *API) getAddressTransactions(address string) ([]Transaction, error) {
       cacheKey := fmt.Sprintf("txs:%s", address)
       
       // Try cache first
       if cached, err := api.redis.Get(ctx, cacheKey).Result(); err == nil {
           var txs []Transaction
           json.Unmarshal([]byte(cached), &txs)
           return txs, nil
       }
       
       // Cache miss - query database
       txs := api.db.Query(...)
       
       // Cache for 5 minutes
       data, _ := json.Marshal(txs)
       api.redis.Set(ctx, cacheKey, data, 5*time.Minute)
       
       return txs, nil
   }
   ```

4. **Use connection pooling**
   ```go
   // services/api/main.go
   db.SetMaxOpenConns(50)
   db.SetMaxIdleConns(10)
   ```

5. **Add read replicas**
   ```
   Primary: Writes only
   Replica 1-5: Read queries distributed round-robin
   ```

### Docker & Port Issues

#### Issue: Port Already in Use

```bash
# Find what's using port 5432
lsof -i :5432
# Kill the process
kill -9 <PID>
```

#### Issue: Docker Containers Won't Start

```bash
# Check Docker daemon is running
docker ps

# View container logs
docker logs indexer-postgres

# Restart Docker Desktop
# or: killall Docker && open -a Docker
```

#### Issue: Database Connection Fails

```bash
# Wait for PostgreSQL to fully start (takes ~10 seconds)
docker logs indexer-postgres | grep "ready to accept connections"

# Test connection manually
psql -h localhost -U indexer -d indexer
# Password: password
```

---

## Part 2: Performance Optimization

### Database Optimizations

```sql
-- 1. Analyze tables regularly (update query planner statistics)
ANALYZE blocks;
ANALYZE transactions;
ANALYZE events;

-- 2. Vacuum to reclaim space
VACUUM FULL events;  -- Reclaims disk space, requires table lock

-- 3. Reindex periodically (rebuilds indexes)
REINDEX TABLE events;

-- 4. Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0;  -- Unused indexes (consider dropping)

-- 5. Tune PostgreSQL configuration
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
ALTER SYSTEM SET maintenance_work_mem = '1GB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;  -- For SSD
ALTER SYSTEM SET effective_io_concurrency = 200;  -- For SSD

-- Reload configuration
SELECT pg_reload_conf();
```

### Go Performance Tips

```go
// 1. Use sync.Pool for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processBlock(block Block) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    // Use buf...
}

// 2. Pre-allocate slices when size is known
events := make([]Event, 0, 1000)  // Capacity 1000

// 3. Use buffered channels to prevent blocking
ch := make(chan Block, 100)

// 4. Limit goroutines with worker pool
sem := make(chan struct{}, 10)  // Max 10 concurrent workers

for _, block := range blocks {
    sem <- struct{}{}  // Acquire semaphore
    go func(b Block) {
        defer func() { <-sem }()  // Release semaphore
        processBlock(b)
    }(block)
}

// 5. Profile in production
import _ "net/http/pprof"

func main() {
    // Start profiler endpoint
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // Your app...
}

// Access profiles:
// http://localhost:6060/debug/pprof/
// http://localhost:6060/debug/pprof/heap
// http://localhost:6060/debug/pprof/goroutine
```

### Kafka & Redis Optimizations

**Kafka Producer**:
```yaml
producer:
  acks: 1  # Leader acknowledgment (balance between speed and durability)
  compression.type: snappy  # Reduce network bandwidth
  batch.size: 65536  # Larger batches = better throughput
  linger.ms: 10  # Wait up to 10ms to batch messages
  buffer.memory: 67108864  # 64MB buffer
```

**Kafka Consumer**:
```yaml
consumer:
  fetch.min.bytes: 50000  # Fetch at least 50KB
  fetch.max.wait.ms: 500  # Wait up to 500ms to accumulate data
  max.poll.records: 1000  # Process 1000 messages per poll
  session.timeout.ms: 30000  # 30 second session timeout
  enable.auto.commit: false  # Manual commit for exactly-once
```

**Redis**:
```bash
# redis.conf optimizations
maxmemory 2gb
maxmemory-policy allkeys-lru  # Evict least recently used keys
save ""  # Disable persistence (optional for cache-only)

# Connection pooling
redis-cli CONFIG SET tcp-keepalive 60
redis-cli CONFIG SET timeout 300
```

---

## Part 3: Real-World Case Studies

These are actual production issues encountered and solved during development, with detailed analysis and solutions.

### Case Study 1: Database Batch INSERT (300x Speedup)

**Context**: Ingesting Ethereum blocks with 100-200 transactions per block

**Problem**:
- Database transaction timeout after 60 seconds
- 188 individual INSERT statements executed in a loop within a single DB transaction
- Each INSERT taking ~300ms due to network round-trip overhead

**Investigation**:
```bash
# Symptom in logs
ERROR: database transaction timeout exceeded (60s)
Context: Inserting 188 transactions from block 19000100

# Root Cause Analysis
# Time breakdown for 188 transactions:
- Database query execution time: ~2-3ms per INSERT
- Network latency (app ↔ postgres container): ~300ms per round-trip
- Total time: 188 × 300ms = 56.4 seconds (just overhead!)
```

**Solution**:
```go
// Before: Individual INSERTs (SLOW)
for _, tx := range transactions {
    _, err := dbTx.Exec(`INSERT INTO transactions ... VALUES ...`)
}

// After: Batch INSERT (FAST)
func insertTransactionsBatch(dbTx *sql.Tx, chainID int64, transactions []Transaction) error {
    var query strings.Builder
    query.WriteString(`INSERT INTO transactions (...) VALUES `)
    
    args := make([]interface{}, 0, len(transactions)*11)
    for i, tx := range transactions {
        if i > 0 { query.WriteString(", ") }
        // Build dynamic placeholders: ($1,$2,$3), ($4,$5,$6)...
        // ...
    }
    _, err := dbTx.Exec(query.String(), args...)
    return err
}
```

**Result**:
- **Time**: 60s → 200ms (300x improvement)
- **Network round-trips**: 188 → 1
- **Throughput**: Can now process blocks with 1000+ transactions without issue

### Case Study 2: Receipt Fetch Retry Logic & Silent Failures

**Context**: Transaction receipts fetched via RPC to get gas_used and status fields

**Problem**:
- Many transactions in database had `gas_used = 0`
- No errors or warnings in logs
- Data silently corrupted with default values

**Root Cause**:
- Transient RPC errors (rate limiting, network glitches)
- No retry logic - single attempt only
- Fail-silent behavior - defaults to zero instead of failing

**Solution**:
```go
// Fetch receipt with retry logic
maxRetries := 2
backoff := 300 * time.Millisecond

for attempt := 0; attempt <= maxRetries; attempt++ {
    if attempt > 0 {
        time.Sleep(backoff)
        backoff *= 2  // Exponential backoff
    }
    
    receipt, err = ethClient.TransactionReceipt(ctx, tx.Hash())
    if err == nil { break }
}

// Fail-fast: Don't save incomplete data
if err != nil {
    return fmt.Errorf("failed to fetch receipt after retries: %w", err)
}
```

**Result**:
- **Data quality**: 100% of transactions now have correct gas_used values
- **Error visibility**: Failed blocks now log errors instead of silently corrupting data

### Case Study 3: React State Management - Default Parameter Bug

**Context**: React frontend with transaction limit dropdown (100, 500, 1000)

**Problem**:
- User selects "1000 transactions"
- UI briefly updates, then glitches back to "500 transactions"
- Auto-refresh (every 5s) resets the selected value

**Root Cause**:
- **Stale closure**: `useEffect` captures `loadTransactions` at mount time
- **Default parameters**: `limit = txLimit` captures state at function definition time
- **React doesn't re-run effect**: Dependencies didn't include `txLimit`

**Solution**:
```tsx
// Before (Buggy)
const loadTransactions = async (chainId, limit = txLimit) => { ... }
useEffect(() => {
    setInterval(() => loadTransactions(selectedChain), 5000)
}, [selectedChain])

// After (Fixed)
const loadTransactions = async (chainId, limit: number) => { ... } // Explicit param
useEffect(() => {
    setInterval(() => loadTransactions(selectedChain, txLimit), 5000)
}, [selectedChain, txLimit]) // Add dependency
```

### Case Study 4: Multiple Process Detection & Cleanup

**Context**: Alchemy API request count increasing unexpectedly fast

**Problem**:
- User reports: "Alchemy shows requests increasing but ingester is not running"
- `ps aux | grep ingester` returns nothing
- But rate limiting (429 errors) suggests active ingestion

**Root Cause**:
- User ran `go run main.go` multiple times
- Go processes show as `/tmp/go-build.../main`, not "ingester"
- `grep ingester` failed to find them
- 6 hidden instances were running simultaneously

**Solution**:
```bash
# Kill all Go processes related to this project
pkill -f "go-build.*main|go run main.go"

# Find by working directory (reliable)
lsof +D /path/to/project/services/ingester
```

---

## Part 4: Monitoring & Debugging

### Metrics to Monitor

**Ingester:**
- Blocks ingested per second
- RPC request latency (P50, P95, P99)
- Reorg count
- Error rate

**Processor:**
- Kafka consumer lag
- Messages processed per second
- Database write latency
- Error rate

**API:**
- Request latency (P50, P95, P99)
- Requests per second
- Cache hit rate
- Error rate (4xx, 5xx)

**Database:**
- Connection pool usage
- Query latency
- Deadlocks
- Disk usage

### Debugging Commands

```bash
# 1. Check Docker container health
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 2. View container logs
docker logs -f indexer-postgres
docker logs -f indexer-kafka --tail 100

# 3. Database queries
psql -U indexer -d indexer

-- Active queries
SELECT pid, query, state, query_start 
FROM pg_stat_activity 
WHERE state = 'active';

-- Long-running queries
SELECT pid, now() - query_start as duration, query 
FROM pg_stat_activity 
WHERE state = 'active' 
ORDER BY duration DESC;

-- Kill long query
SELECT pg_terminate_backend(12345);  -- Replace with pid

-- Table sizes
SELECT 
    tablename,
    pg_size_pretty(pg_total_relation_size(tablename::text)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(tablename::text) DESC;

# 4. Redis debugging
redis-cli
INFO memory
MONITOR  # Watch commands in real-time
SLOWLOG GET 10

# 5. Kafka debugging
# List topics
docker exec -it indexer-kafka rpk topic list

# Consume messages
docker exec -it indexer-kafka rpk topic consume blocks --offset start

# Check consumer groups
docker exec -it indexer-kafka rpk group list
docker exec -it indexer-kafka rpk group describe block_processor
```

---

## Interview Questions & Answers

### Q1: How would you debug a production issue where the ingester is falling behind the chain head?

**Answer**: I would follow a systematic diagnosis process:
1. **Check Metrics**: Look at `blocks_ingested_total` vs `chain_head_block`. Is the gap increasing?
2. **Check RPC Latency**: High `rpc_request_duration` suggests the provider is the bottleneck.
   - *Solution*: Add more RPC providers (load balancing) or switch to a paid tier.
3. **Check Processing Time**: If RPC is fast but ingestion is slow, the bottleneck is processing/DB.
   - *Solution*: Profile the application (pprof), check DB write latency, or increase batch size.
4. **Check Errors**: Look for 429 (Rate Limit) or 5xx errors in logs.
   - *Solution*: Implement exponential backoff or reduce concurrency.
5. **Scale**: If single instance is maxed out, shard by block range (e.g., Ingester A: 0-1M, Ingester B: 1M+).

### Q2: Explain the "Stale Closure" problem in React hooks and how to fix it.

**Answer**: A stale closure happens when a function (like an effect or callback) captures variables from a previous render and continues to use those old values even after the component re-renders with new state.

**Example**:
```javascript
const [count, setCount] = useState(0);

useEffect(() => {
  const timer = setInterval(() => {
    console.log(count); // Always prints 0 because 'count' is captured from initial render
  }, 1000);
  return () => clearInterval(timer);
}, []); // Empty dependency array causes the staleness
```

**Fixes**:
1. **Add dependency**: Add `count` to the dependency array `[count]`. This restarts the interval on every change.
2. **Functional update**: Use `setCount(prev => prev + 1)` which doesn't rely on the closure variable.
3. **Refs**: Use `useRef` to hold mutable values that don't trigger re-renders but are always current.

### Q3: Why is batching database inserts critical for performance in this architecture?

**Answer**: In a containerized or cloud environment, network latency dominates small transactions.
- **Scenario**: Inserting 100 transactions individually.
- **Overhead**: 100 network round-trips. If latency is 2ms, that's 200ms just in network time, plus DB overhead.
- **Batching**: Sending 100 rows in 1 `INSERT` statement requires only 1 network round-trip.
- **Impact**: We observed a 300x speedup (60s → 200ms) by switching to batch inserts.
- **Trade-off**: Large batches consume more memory and might hit packet size limits, so we cap batches (e.g., 1000 rows).

### Q4: How do you handle "Fail-Silent" errors in data ingestion?

**Answer**: Fail-silent errors (like defaulting `gas_used` to 0 on error) corrupt data integrity without triggering alarms.
**Strategy**:
1. **Fail Fast**: If a critical field (receipt, timestamp) cannot be fetched, return an error immediately. Do not save partial data.
2. **Retry Logic**: Implement retries with exponential backoff for transient errors (network, rate limits).
3. **Dead Letter Queue**: If retries fail, send the block/tx ID to a DLQ for manual inspection, rather than skipping or corrupting it.
4. **Validation**: Add database constraints (e.g., `CHECK (gas_used > 0)`) to prevent invalid data at the schema level.

### Q5: What is the difference between `pkill`, `kill`, and `kill -9`?

**Answer**:
- **`kill <PID>`**: Sends `SIGTERM` (15). Asks the process to stop nicely. The process can catch this signal to clean up resources (close DB connections, flush logs) before exiting.
- **`kill -9 <PID>`**: Sends `SIGKILL` (9). Forcefully terminates the process immediately. The process cannot catch this and cannot clean up. Use only as a last resort.
- **`pkill <name>`**: Finds processes by name and sends `SIGTERM`. Useful for killing multiple processes (e.g., `pkill -f ingester`).

### Q6: How would you identify a memory leak in a Go service running in production?

**Answer**:
1. **Metrics**: Monitor `go_memstats_heap_inuse_bytes` over time. A sawtooth pattern is normal (GC), but a rising floor indicates a leak.
2. **pprof**: Enable `net/http/pprof`.
   - Take a heap profile: `go tool pprof http://localhost:6060/debug/pprof/heap`
   - Compare two profiles (diff) taken minutes apart to see what objects are accumulating.
3. **Common Causes**:
   - Goroutine leaks (blocked on channel/mutex).
   - Global maps growing without deletion.
   - `time.Ticker` not stopped.
   - Unclosed response bodies (`resp.Body.Close()`).

### Q7: Your API latency spiked to 5 seconds. How do you debug it?

**Answer**:
1. **Identify the bottleneck**: Is it DB, Redis, or CPU?
2. **Check DB**: Run `SELECT * FROM pg_stat_activity WHERE state = 'active'` to see stuck queries. Check `pg_stat_statements` for slow query logs.
   - *Fix*: Add missing indexes or optimize queries.
3. **Check Redis**: Is the cache hit rate low?
   - *Fix*: Increase TTL or cache more aggressive keys.
4. **Check CPU**: Is the service CPU throttled?
   - *Fix*: Increase container CPU limits.
5. **Tracing**: If using distributed tracing (Jaeger/OpenTelemetry), look at the trace waterfall to see which span is taking the most time.

### Q8: Explain the "Exponential Backoff" algorithm and why it's used.

**Answer**: Exponential backoff is a retry strategy where the wait time between retries increases exponentially (e.g., 1s, 2s, 4s, 8s).
- **Purpose**: To prevent "thundering herd" problems where failing clients retry immediately and simultaneously, overwhelming the already struggling server.
- **Jitter**: We often add random "jitter" (e.g., wait 2s ± 100ms) to desynchronize clients so they don't all retry at the exact same millisecond.
- **Implementation**: Used in our RPC client to handle rate limits (429) and network timeouts gracefully.

---

**Related Documentation**:
- [System Design & Architecture](./System-Design-Architecture.md)
- [Docker & Kubernetes](./Docker-Kubernetes.md)
- [Go Programming](./Go-Programming.md)

**Last Updated**: November 28, 2025
