# Database & Messaging Patterns

> **Purpose**: Advanced patterns for PostgreSQL, Kafka, and Redis in blockchain indexing systems.

---

## PostgreSQL Advanced Topics

### Indexing Strategies

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

### Identifying Missing Indexes

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

### Index Trade-offs

**Pros**: Faster reads (can be 1000x speedup)

**Cons**:
- Slower writes (every INSERT/UPDATE must update indexes)
- More storage (30-50% overhead)
- Maintenance overhead

**Rule**: Index columns in WHERE, JOIN, ORDER BY clauses with high selectivity

---

## Partitioning Deep Dive

### Partitioning Types

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

### Our Choice: List Partitioning

```sql
CREATE TABLE blocks_chain_1 PARTITION OF blocks
FOR VALUES IN (1);  -- Ethereum only
```

### Partitioning Interview Questions

**Q: When should you partition a table?**

**A:** When table > 10GB and queries filter by partition key. Benefits:
- Faster queries (prune partitions)
- Easier archiving (drop old partitions)
- Parallel maintenance

**Q: How does partition pruning work?**

```sql
EXPLAIN SELECT * FROM blocks WHERE chain_id = 1;
-- Only scans blocks_chain_1, skips other partitions
```

---

## Kafka & Event Streaming

### Producer Patterns

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

### Kafka Interview Questions

**Q: How do you ensure exactly-once semantics in Kafka?**

**A:** Enable:
1. Idempotent producer (retries don't create duplicates)
2. Transactional producer (atomic multi-partition writes)
3. `read_committed` consumer isolation level

**Q: What's the difference between topics and partitions?**

- **Topic**: Logical stream (e.g., "raw.blocks")
- **Partition**: Physical log file (e.g., "raw.blocks-0", "raw.blocks-1")
- **Ordering guarantee**: Within partition only, not across partitions
- **Scaling**: Add partitions to parallelize consumers

**Q: How does Kafka handle consumer failures?**

**A:** Consumer heartbeat mechanism. If consumer stops heartbeat, coordinator triggers rebalance and reassigns partitions to healthy consumers. Offset commits allow resuming from last processed message.

### Consumer Group Patterns

```go
// Consumer group ensures each message processed once
consumer := kafka.NewConsumer(&kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092",
    "group.id":          "indexer-processor",
    "auto.offset.reset": "earliest",
})
```

---

## Redis Caching Patterns

### Cache-Aside (Lazy Loading)

```go
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
```

### Write-Through (Always Consistent)

```go
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

### Redis Interview Questions

**Q: How do you handle cache invalidation?**

**A:** "There are only two hard things in Computer Science: cache invalidation and naming things."

Strategies:
- **TTL**: Expire after time (good for rarely-changed data)
- **Event-driven**: Kafka consumer invalidates cache on updates
- **Version tags**: Increment version number in key

**Q: What's the thundering herd problem?**

**A:** Many requests hit expired cache key simultaneously, all query DB in parallel. 

**Solutions**:
- Lock with Redis SETNX
- Probabilistic early expiration
- Cache warming

---

## State Management Patterns

### React State Management

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

### State Management Interview Questions

**Q: When would you use Context vs Redux vs Zustand?**

- **Context**: Small apps, simple global state (theme, auth)
- **Redux**: Large apps, complex state logic, time-travel debugging
- **Zustand**: Middle ground, less boilerplate than Redux

**Q: What's the difference between SSR, SSG, and CSR in Next.js?**

- **SSR (getServerSideProps)**: Render on each request (dynamic data)
- **SSG (getStaticProps)**: Render at build time (blogs, docs)
- **CSR (useEffect)**: Render in browser (dashboards, user-specific)
- **ISR (revalidate)**: SSG with periodic regeneration (best of both)

---

## Async Patterns

### Node.js & TypeScript

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

### Async Interview Questions

**Q: What's the event loop?**

**A:** Single-threaded execution with callback queue. Async operations (I/O, timers) run in background, callbacks queued for event loop to process when call stack is empty.

**Q: How do you avoid callback hell?**

**A:** Use Promises or async/await, create named functions, use libraries like async.js for complex flows.

---

**Related Documents**:
- [Technology Stack](./01-technology-stack.md)
- [Docker & Kubernetes](./02-docker-kubernetes.md)
- [Go Programming Guide](./04-go-programming.md)
- [Troubleshooting](./09-troubleshooting.md)
