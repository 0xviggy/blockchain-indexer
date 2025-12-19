# Troubleshooting & Performance Guide

Common issues, solutions, and performance optimization strategies for the blockchain indexer.

## Table of Contents
- [Common Issues](#common-issues)
- [Performance Optimization](#performance-optimization)
- [Monitoring & Debugging](#monitoring--debugging)

---

## Common Issues

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

---

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

---

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

---

## Performance Optimization

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

### Kafka Optimizations

```yaml
# Producer settings
producer:
  acks: 1  # Leader acknowledgment (balance between speed and durability)
  compression.type: snappy  # Reduce network bandwidth
  batch.size: 65536  # Larger batches = better throughput
  linger.ms: 10  # Wait up to 10ms to batch messages
  buffer.memory: 67108864  # 64MB buffer

# Consumer settings
consumer:
  fetch.min.bytes: 50000  # Fetch at least 50KB
  fetch.max.wait.ms: 500  # Wait up to 500ms to accumulate data
  max.poll.records: 1000  # Process 1000 messages per poll
  session.timeout.ms: 30000  # 30 second session timeout
  enable.auto.commit: false  # Manual commit for exactly-once
```

### Redis Optimizations

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

## Monitoring & Debugging

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

# Check memory usage
INFO memory

# Monitor commands in real-time
MONITOR

# Check slow queries
SLOWLOG GET 10

# 5. Kafka debugging
# List topics
kafka-topics --bootstrap-server localhost:9092 --list

# Describe topic
kafka-topics --bootstrap-server localhost:9092 --describe --topic blocks

# Consume messages
kafka-console-consumer --bootstrap-server localhost:9092 --topic blocks --from-beginning

# Check consumer groups
kafka-consumer-groups --bootstrap-server localhost:9092 --list
kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group block_processor
```

### Load Testing

```bash
# Apache Bench
ab -n 1000 -c 10 http://localhost:8080/v1/chains

# vegeta (better for APIs)
echo "GET http://localhost:8080/v1/chains" | vegeta attack -duration=30s -rate=100 | vegeta report

# k6 (programmable load testing)
k6 run load_test.js
```

**load_test.js:**
```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '1m', target: 100 },  // Ramp up to 100 users
    { duration: '3m', target: 100 },  // Stay at 100 users
    { duration: '1m', target: 0 },    // Ramp down
  ],
};

export default function () {
  let res = http.get('http://localhost:8080/v1/chains/1/blocks');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  sleep(1);
}
```

---

## Performance Optimization - Real-World Case Studies

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

# Confirmed with timing instrumentation
log.Printf("Insert took: %v", time.Since(start))
// Output: Insert took: 58.2s
```

**Root Cause**:
- Each `INSERT` statement requires a network round-trip between the application and database container
- Even though database execution is fast (~3ms), network latency dominates
- With 188 transactions, the cumulative network overhead exceeded the 60s timeout
- This is NOT a database performance issue - it's a network efficiency issue

**Solution**:
```go
// Before: Individual INSERTs (SLOW)
for _, tx := range transactions {
    _, err := dbTx.Exec(`
        INSERT INTO transactions (chain_id, block_number, tx_hash, ...)
        VALUES ($1, $2, $3, ...)
    `, tx.ChainID, tx.BlockNumber, tx.Hash, ...)
    if err != nil {
        return err
    }
}

// After: Batch INSERT (FAST)
func insertTransactionsBatch(dbTx *sql.Tx, chainID int64, transactions []Transaction) error {
    if len(transactions) == 0 {
        return nil
    }
    
    var query strings.Builder
    query.WriteString(`
        INSERT INTO transactions (
            chain_id, block_number, tx_hash, tx_index, from_address, 
            to_address, value, gas_price, gas_used, status, block_timestamp
        ) VALUES 
    `)
    
    // Build dynamic placeholders: ($1,$2,$3), ($4,$5,$6), ...
    args := make([]interface{}, 0, len(transactions)*11)
    for i, tx := range transactions {
        if i > 0 {
            query.WriteString(", ")
        }
        
        // Calculate placeholder numbers for this row
        base := i * 11
        query.WriteString(fmt.Sprintf(
            "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
            base+1, base+2, base+3, base+4, base+5, base+6,
            base+7, base+8, base+9, base+10, base+11,
        ))
        
        args = append(args,
            chainID,
            tx.BlockNumber,
            tx.Hash,
            tx.Index,
            tx.FromAddress,
            tx.ToAddress,
            tx.Value,
            tx.GasPrice,
            tx.GasUsed,
            tx.Status,
            tx.BlockTimestamp,
        )
    }
    
    _, err := dbTx.Exec(query.String(), args...)
    return err
}
```

**Result**:
- **Time**: 60s → 200ms (300x improvement)
- **Network round-trips**: 188 → 1
- **Database timeout adjusted**: 60s → 20s (no longer needed, but safe buffer)
- **Throughput**: Can now process blocks with 1000+ transactions without issue

**Lessons Learned**:
1. **Network latency dominates in containerized environments** - Even localhost Docker networking has ~200-300ms round-trip times
2. **Batch operations for >50 rows** - If inserting many rows in one transaction, always batch
3. **Timeouts != database performance** - A timeout doesn't mean the database is slow; check network overhead
4. **Profile first** - Use timing instrumentation to identify the bottleneck (DB query vs network vs serialization)
5. **Security not compromised** - Still using parameterized queries ($1, $2, etc.), just batched

**When to Apply This Pattern**:
- ✅ Ingesting blockchain blocks with many transactions (>50)
- ✅ Bulk data import operations
- ✅ High-latency database connections (cloud, containers, cross-region)
- ✅ Write-heavy workloads with multiple related rows
- ❌ Single/few row inserts (overhead not worth complexity)
- ❌ Highly variable row sizes (memory concerns with large batches)
- ❌ Need immediate per-row error handling

### Case Study 2: Receipt Fetch Retry Logic & Silent Failures

**Context**: Transaction receipts fetched via RPC to get gas_used and status fields

**Problem**:
- Many transactions in database had `gas_used = 0`
- No errors or warnings in logs
- Data silently corrupted with default values

**Investigation**:
```bash
# Check database for zero gas values
SELECT COUNT(*) FROM transactions WHERE gas_used = 0;
# Result: 234 out of 601 transactions (39%)

# This is impossible - every Ethereum transaction uses gas!
# Even a simple transfer uses 21,000 gas

# Root cause: Code had fallback to zero on receipt fetch failure
gasUsed := uint64(0)  // Default
receipt, err := client.TransactionReceipt(ctx, txHash)
if err == nil {
    gasUsed = receipt.GasUsed
}
// Bug: Silently continues with zero if receipt fetch fails!
```

**Root Cause**:
- Transient RPC errors (rate limiting, network glitches, timeout)
- No retry logic - single attempt only
- Fail-silent behavior - defaults to zero instead of failing
- Creates corrupted data that looks valid (no NULL, just wrong value)

**Solution**:
```go
// Fetch receipt with retry logic
var receipt *types.Receipt
var err error
maxRetries := 2
backoff := 300 * time.Millisecond

for attempt := 0; attempt <= maxRetries; attempt++ {
    if attempt > 0 {
        log.Printf("Retry %d/%d for receipt %s after %v", 
            attempt, maxRetries, tx.Hash().Hex(), backoff)
        time.Sleep(backoff)
        backoff *= 2  // Exponential backoff: 300ms, 600ms
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    receipt, err = ethClient.TransactionReceipt(ctx, tx.Hash())
    cancel()
    
    if err == nil {
        break  // Success!
    }
    
    log.Printf("Failed to fetch receipt for %s (attempt %d): %v", 
        tx.Hash().Hex(), attempt+1, err)
}

// Fail-fast: Don't save incomplete data
if err != nil {
    return fmt.Errorf("failed to fetch receipt after %d retries: %w", 
        maxRetries+1, err)
}

// Now safe to use receipt data
gasUsed := receipt.GasUsed
status := receipt.Status
```

**Configuration Tuning**:
```go
// Initial aggressive settings (too much)
Retries: 3
Timeout: 15s per attempt
Backoff: 500ms, 1000ms, 2000ms

// Optimized after testing with Alchemy free tier
Retries: 2               // 3 total attempts (initial + 2 retries)
Timeout: 10s per attempt // Alchemy usually responds in 50-100ms
Backoff: 300ms, 600ms    // Quick retry for transient errors

// Total max time per receipt: 10s + 0.3s + 10s + 0.6s + 10s = ~31s
// This is acceptable for historical data ingestion
```

**Result**:
- **Data quality**: 100% of transactions now have correct gas_used values
- **Error visibility**: Failed blocks now log errors instead of silently corrupting data
- **Success rate**: 98%+ receipts succeed on first attempt, 2% need one retry
- **Zero tolerance**: Database query shows 0 transactions with gas_used = 0

**Lessons Learned**:
1. **Never fail silently** - Always surface errors or fail the operation
2. **Default values are dangerous** - `0` looks valid but isn't for gas_used
3. **Retry transient RPC errors** - Free tier rate limits, network glitches common
4. **Exponential backoff** - Avoid hammering the RPC provider
5. **Timeouts essential** - Prevent infinite hangs on stuck RPC calls
6. **Fail-fast principle** - Better to not save data than save corrupt data

**RPC Provider Issues** (Alchemy Free Tier):
- Rate limiting: 429 errors when making requests too quickly
- Compute units: Complex queries (like receipts) cost more
- Latency: ~50-100ms for historical receipts (not instant as expected)
- Solution: Retry logic + backoff handles transient issues gracefully

### Case Study 3: React State Management - Default Parameter Bug

**Context**: React frontend with transaction limit dropdown (100, 500, 1000)

**Problem**:
- User selects "1000 transactions" from dropdown
- UI briefly updates, then glitches back to showing "500 transactions"
- Auto-refresh (every 5s) resets the selected value

**Investigation**:
```tsx
// The buggy code
const [txLimit, setTxLimit] = useState(500)

const loadTransactions = async (chainId: number, limit = txLimit) => {
    const data = await api.getTransactions(chainId, limit)
    setTransactions(data)
}

// Auto-refresh effect
useEffect(() => {
    const interval = setInterval(() => {
        loadTransactions(selectedChain)  // BUG: No limit passed!
    }, 5000)
    return () => clearInterval(interval)
}, [autoRefresh, selectedChain])

// What happens:
// 1. User changes dropdown: setTxLimit(1000)
// 2. Component re-renders
// 3. loadTransactions function is RE-DEFINED with NEW default: limit = 1000
// 4. BUT useEffect has old closure with OLD loadTransactions
// 5. After 5 seconds, interval calls OLD function with default limit = 500
// 6. UI resets to 500 transactions
```

**Root Cause**:
- **Stale closure** - useEffect captures `loadTransactions` at component mount time
- **Default parameters capture state** - `limit = txLimit` captures the value at function definition time
- **React doesn't re-run effect** - Dependencies don't include `txLimit` or `loadTransactions`
- This is a classic React pitfall with default parameters and closures

**Why It Happens**:
```tsx
// At mount time (txLimit = 500):
function loadTransactions(chainId, limit = 500) { ... }  // Captured in closure

// User changes to 1000:
setTxLimit(1000)  // State updates

// Component re-renders, function re-defined:
function loadTransactions(chainId, limit = 1000) { ... }  // NEW function

// BUT useEffect still has OLD function:
setInterval(() => {
    loadTransactions(selectedChain)  // Calls OLD function (limit = 500)!
}, 5000)
```

**Solution 1**: Remove default parameters, pass explicitly
```tsx
// Before
const loadTransactions = async (chainId: number, limit = txLimit) => {
    // ...
}

// After
const loadTransactions = async (chainId: number, limit: number) => {
    // limit is now required - no stale closure possible
}

// Update all call sites
loadTransactions(selectedChain, txLimit)  // Explicit everywhere
loadChains() // Pass txLimit here too
handleChainChange() // And here
```

**Solution 2**: Add to useEffect dependencies
```tsx
useEffect(() => {
    if (!autoRefresh || !selectedChain) return
    
    const interval = setInterval(() => {
        loadTransactions(selectedChain, txLimit)  // Pass current state
    }, 5000)
    
    return () => clearInterval(interval)
}, [autoRefresh, selectedChain, txLimit])  // Add txLimit dependency
```

**Result**:
- Transaction limit selection now persists correctly
- Auto-refresh respects the current limit setting
- No more glitching back to default value

**Lessons Learned**:
1. **Avoid default parameters with React state** - They create stale closures
2. **Be explicit with function parameters** - Required params > defaults in React
3. **useEffect dependencies matter** - Include ALL values used inside the effect
4. **ESLint warnings are your friend** - `react-hooks/exhaustive-deps` catches this
5. **Test auto-refresh scenarios** - Manual interactions may work, but intervals expose bugs

**General React Pitfall Pattern**:
```tsx
// ❌ BAD: State in default parameter
const [count, setCount] = useState(0)
const increment = (by = count) => setCount(prev => prev + by)

// ✅ GOOD: Required parameter
const increment = (by: number) => setCount(prev => prev + by)

// ❌ BAD: Missing dependency
useEffect(() => {
    doSomething(count)
}, [])  // count not in deps!

// ✅ GOOD: Complete dependencies
useEffect(() => {
    doSomething(count)
}, [count])
```

### Case Study 4: Multiple Process Detection & Cleanup

**Context**: Alchemy API request count increasing unexpectedly fast

**Problem**:
- User reports: "Alchemy shows requests increasing but ingester is not running"
- Command `ps aux | grep ingester | grep -v grep` returns no results (exit code 1)
- But rate limiting (429 errors) suggests active ingestion happening

**Investigation**:
```bash
# Standard check shows nothing
ps aux | grep -i ingester | grep -v grep
# Exit code: 1 (not found)

# Broader search for Go processes
ps aux | grep -E "(go run|go-build.*main)" | grep -v grep
# Found: 6 different Go processes running!

# Check working directory of processes
lsof -p 46034 2>/dev/null | grep cwd | awk '{print $NF}'
# /Users/viggy/devPlayground/blockchain-indexer/services/ingester

# Multiple ingester instances discovered:
# PID 46034 - Started 7:15PM
# PID 43276 - Started 7:10PM  
# PID 63351 - Started 8:19PM
# ... and more
```

**Root Cause**:
- User ran `go run main.go` multiple times in different terminals
- Each creates a compiled binary in `/tmp/go-build*/` or cache
- Process name shows as generic "main" or path to go-build cache
- `grep ingester` doesn't match these process names
- All instances independently fetch blocks, multiplying RPC usage

**Why Standard Commands Failed**:
```bash
# This looks for "ingester" in process command line
ps aux | grep ingester
# But Go processes show as:
# /Users/viggy/Library/Caches/go-build/.../main
# /tmp/go-build3829713833/b001/exe/main
# go run main.go

# "ingester" never appears in the process name!
```

**Solution**:
```bash
# Kill all Go processes related to this project
pkill -f "go-build.*main|go run main.go"

# More targeted: Kill by working directory
for pid in $(lsof +D /Users/viggy/devPlayground/blockchain-indexer/services/ingester 2>/dev/null | awk 'NR>1 {print $2}' | sort -u); do
    kill $pid
done

# Verify cleanup
ps aux | grep -E "(go run|go-build.*main)" | grep -v grep
# Should return nothing (exit code 1)
```

**Prevention Strategies**:
```bash
# 1. Use Makefile for consistent process management
make run-ingester  # Creates traceable process
make stop-ingester # Reliable cleanup

# 2. Name your processes
go run -ldflags="-X main.processName=ingester-dev" main.go

# 3. Use PID files
echo $$ > /tmp/ingester.pid
# On stop: kill $(cat /tmp/ingester.pid)

# 4. Check before starting
if pgrep -f "ingester.*main.go" > /dev/null; then
    echo "Ingester already running!"
    exit 1
fi
```

**Better Process Detection**:
```bash
# Find by working directory
lsof +D /path/to/ingester | grep main

# Find by executable path
pgrep -f "blockchain-indexer/services/ingester"

# Find by port (if applicable)
lsof -i :8080

# Find with full command line
ps auxww | grep -E "(ingester|blockchain)" | grep -v grep
```

**Result**:
- Discovered 6 simultaneous ingester instances
- Killed all with single `pkill` command
- API request rate returned to normal
- Added process count check to monitoring

**Lessons Learned**:
1. **Process names != binary names** - Go compiled binaries have generic names
2. **grep ingester is unreliable** - Use working directory or full path
3. **pkill -f is powerful** - Matches full command line, not just process name
4. **lsof for working directory** - Reliable way to find processes by project location
5. **Makefile process management** - Prevents duplicate process issues
6. **Monitor unexpected resource usage** - Rate limits revealed the hidden processes

**Tool Comparison**:
```bash
# pgrep: Fast, but matches only process name
pgrep ingester  # Won't find "main" processes

# pkill -f: Matches full command line
pkill -f "go-build.*main"  # Finds Go binaries

# ps + grep: Verbose but flexible
ps auxww | grep ingester  # Shows full info

# lsof: Best for finding by directory/port
lsof +D /path/to/project  # Most reliable for Go projects
```

---

## Related Documentation

- [Setup Guide](./05-setup-quickstart.md)
- [Implementation Concepts](./06-implementation-concepts.md)
- [Interview Preparation](./07-interview-prep.md)
- [Technical Specification](../docs/TECHNICAL_SPEC.md)
