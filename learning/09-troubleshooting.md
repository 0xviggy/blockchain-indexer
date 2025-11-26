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

## Related Documentation

- [Setup Guide](./05-setup-quickstart.md)
- [Implementation Concepts](./06-implementation-concepts.md)
- [Interview Preparation](./07-interview-prep.md)
- [Technical Specification](../docs/TECHNICAL_SPEC.md)
