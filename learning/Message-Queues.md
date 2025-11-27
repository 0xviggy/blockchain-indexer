# Message Queues & Caching

> **Purpose**: Comprehensive guide to Kafka, Redis, and event-driven architecture for blockchain indexing systems with integrated interview Q&A.

---

## Table of Contents
- [Kafka Architecture](#kafka-architecture)
- [Producer Patterns](#producer-patterns)
- [Consumer Patterns](#consumer-patterns)
- [Exactly-Once Semantics](#exactly-once-semantics)
- [Redis Fundamentals](#redis-fundamentals)
- [Caching Strategies](#caching-strategies)
- [Redis Advanced Patterns](#redis-advanced-patterns)
- [Event-Driven Architecture](#event-driven-architecture)
- [Interview Questions](#interview-questions)

---

## Kafka Architecture

### Core Concepts

**Topics**: Logical streams of events
```
raw.blocks       - Blockchain blocks
parsed.events    - Decoded smart contract events
processed.txs    - Enriched transactions
```

**Partitions**: Physical log files that make up a topic
```
raw.blocks-0  (Partition 0)
raw.blocks-1  (Partition 1)
raw.blocks-2  (Partition 2)
```

**Brokers**: Kafka servers that store partitions
```
Broker 1: raw.blocks-0, parsed.events-1
Broker 2: raw.blocks-1, parsed.events-2
Broker 3: raw.blocks-2, parsed.events-0
```

**Consumer Groups**: Multiple consumers share the load
```
Consumer Group "indexer-processor"
  - Consumer 1: Processes Partition 0
  - Consumer 2: Processes Partition 1
  - Consumer 3: Processes Partition 2
```

### Ordering Guarantees

**Within partition**: Total ordering guaranteed
```
Partition 0: [Block 1000, Block 1001, Block 1002]  ✅ Order preserved
```

**Across partitions**: No ordering guarantee
```
Partition 0: [Block 1000, Block 1002]
Partition 1: [Block 1001, Block 1003]
Consumer may receive: 1000, 1001, 1002, 1003 OR 1001, 1000, 1003, 1002
```

**Key-based partitioning**: Same key always goes to same partition
```go
msg := &kafka.Message{
    Topic: "raw.blocks",
    Key:   []byte(fmt.Sprintf("%d", chainID)),  // All Ethereum blocks to same partition
    Value: blockJSON,
}
```

### Replication & Fault Tolerance

**Replication factor**: Number of copies per partition
```
Partition raw.blocks-0:
  - Leader: Broker 1 (handles reads/writes)
  - Follower: Broker 2 (replicates)
  - Follower: Broker 3 (replicates)
```

**Leader election**: If leader fails, follower becomes leader
```
Before failure:
  Leader: Broker 1
  Followers: Broker 2, Broker 3

After Broker 1 fails:
  Leader: Broker 2 (promoted)
  Follower: Broker 3
```

**In-Sync Replicas (ISR)**: Replicas caught up with leader
```yaml
# docker-compose.yml
KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 3
KAFKA_MIN_INSYNC_REPLICAS: 2  # At least 2 replicas must acknowledge write
```

---

## Producer Patterns

### Asynchronous Production (High Throughput)

```go
import "github.com/confluentinc/confluent-kafka-go/kafka"

producer, err := kafka.NewProducer(&kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092",
    "acks":              "all",        // Wait for all ISR replicas
    "retries":           10,           // Retry on failure
    "enable.idempotence": true,        // Exactly-once delivery
})

// Non-blocking send
err = producer.Produce(&kafka.Message{
    Topic: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
    Key:   []byte(fmt.Sprintf("%d:%d", chainID, blockNumber)),
    Value: blockJSON,
}, nil)

// Handle delivery reports asynchronously
go func() {
    for e := range producer.Events() {
        switch ev := e.(type) {
        case *kafka.Message:
            if ev.TopicPartition.Error != nil {
                log.Printf("Delivery failed: %v", ev.TopicPartition.Error)
            } else {
                log.Printf("Delivered to partition %d at offset %d",
                    ev.TopicPartition.Partition, ev.TopicPartition.Offset)
            }
        }
    }
}()
```

### Synchronous Production (Guaranteed Delivery)

```go
// Blocking send with acknowledgment
deliveryChan := make(chan kafka.Event)
err = producer.Produce(&kafka.Message{
    Topic: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
    Value: blockJSON,
}, deliveryChan)

e := <-deliveryChan
m := e.(*kafka.Message)
if m.TopicPartition.Error != nil {
    log.Printf("Delivery failed: %v", m.TopicPartition.Error)
}
```

### Batching for Performance

```go
producer, err := kafka.NewProducer(&kafka.ConfigMap{
    "bootstrap.servers":  "localhost:9092",
    "linger.ms":          100,    // Wait up to 100ms to batch messages
    "batch.size":         16384,  // Max batch size in bytes
    "compression.type":   "snappy", // Compress batches
})
```

**Trade-offs:**
- ✅ **Higher throughput**: 10-100x more messages/sec
- ❌ **Higher latency**: Messages delayed up to `linger.ms`
- ✅ **Lower network overhead**: Fewer round-trips

### Partitioning Strategies

**1. Explicit partition**
```go
partition := chainID % 3
producer.Produce(&kafka.Message{
    Topic: kafka.TopicPartition{Topic: &topic, Partition: partition},
    Value: blockJSON,
}, nil)
```

**2. Key-based partitioning (default)**
```go
// Same key → same partition (preserves ordering per chain)
producer.Produce(&kafka.Message{
    Topic: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
    Key:   []byte(fmt.Sprintf("chain:%d", chainID)),
    Value: blockJSON,
}, nil)
```

**3. Round-robin (no key)**
```go
// Distributes evenly, no ordering guarantee
producer.Produce(&kafka.Message{
    Topic: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
    Value: blockJSON,
}, nil)
```

---

## Consumer Patterns

### Basic Consumer

```go
consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092",
    "group.id":          "indexer-processor",
    "auto.offset.reset": "earliest",  // Start from beginning if no offset stored
})

consumer.Subscribe("raw.blocks", nil)

for {
    msg, err := consumer.ReadMessage(-1)  // Block indefinitely
    if err != nil {
        log.Printf("Consumer error: %v", err)
        continue
    }
    
    // Process message
    if err := processBlock(msg.Value); err != nil {
        log.Printf("Processing failed: %v", err)
        continue
    }
    
    // Commit offset after successful processing
    consumer.CommitMessage(msg)
}
```

### Manual Offset Management

```go
consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
    "bootstrap.servers":  "localhost:9092",
    "group.id":           "indexer-processor",
    "enable.auto.commit": false,  // Manual commit
})

for {
    msg, err := consumer.ReadMessage(-1)
    if err != nil {
        continue
    }
    
    // Process in database transaction
    tx, _ := db.Begin()
    if err := processBlock(tx, msg.Value); err != nil {
        tx.Rollback()
        continue
    }
    
    // Only commit offset after DB commit
    tx.Commit()
    consumer.CommitMessage(msg)  // Guarantees at-least-once
}
```

### Consumer Rebalancing

**Trigger**: Consumer added/removed, partition count changes

```go
consumer.Subscribe("raw.blocks", &kafka.RebalanceCb{
    Partitions: func(c *kafka.Consumer, event kafka.Event) error {
        switch e := event.(type) {
        case kafka.AssignedPartitions:
            log.Printf("Partitions assigned: %v", e.Partitions)
            c.Assign(e.Partitions)
        case kafka.RevokedPartitions:
            log.Printf("Partitions revoked: %v", e.Partitions)
            // Commit pending work before revocation
            c.Unassign()
        }
        return nil
    },
})
```

**Rebalance protocol:**
1. Coordinator detects consumer change
2. All consumers stop processing
3. Partitions reassigned
4. Consumers resume from last committed offset

### Offset Reset Strategies

```go
// Start from beginning (replay all data)
"auto.offset.reset": "earliest"

// Start from latest (ignore old data)
"auto.offset.reset": "latest"

// Fail if no offset stored (explicit control)
"auto.offset.reset": "error"
```

---

## Exactly-Once Semantics

### At-Most-Once (May Lose Data)

```go
// ❌ Commit before processing
msg, _ := consumer.ReadMessage(-1)
consumer.CommitMessage(msg)  // Committed

// If crash here, message lost
processBlock(msg.Value)
```

**Use case**: Metrics, non-critical events

### At-Least-Once (May Duplicate)

```go
// ✅ Commit after processing
msg, _ := consumer.ReadMessage(-1)
processBlock(msg.Value)

// If crash here, message reprocessed
consumer.CommitMessage(msg)
```

**Use case**: Most applications (combine with idempotent operations)

### Exactly-Once (No Loss, No Duplicates)

**Requirements:**
1. Idempotent producer
2. Transactional producer
3. Read committed isolation

**Producer setup:**
```go
producer, err := kafka.NewProducer(&kafka.ConfigMap{
    "bootstrap.servers":   "localhost:9092",
    "enable.idempotence":  true,  // Prevents duplicate sends
    "transactional.id":    "indexer-txn-1",
})

producer.InitTransactions(nil)

// Produce in transaction
producer.BeginTransaction()
producer.Produce(&kafka.Message{
    Topic: kafka.TopicPartition{Topic: &topic1},
    Value: data1,
}, nil)
producer.Produce(&kafka.Message{
    Topic: kafka.TopicPartition{Topic: &topic2},
    Value: data2,
}, nil)
producer.CommitTransaction(nil)  // Atomic commit
```

**Consumer setup:**
```go
consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092",
    "group.id":          "indexer-processor",
    "isolation.level":   "read_committed",  // Only read committed transactions
})
```

### Idempotent Database Operations

**Using ON CONFLICT:**
```sql
INSERT INTO blocks (chain_id, block_number, hash, timestamp)
VALUES ($1, $2, $3, $4)
ON CONFLICT (chain_id, block_number) DO NOTHING;
-- Reprocessing same message has no effect
```

**Using upserts:**
```sql
INSERT INTO checkpoints (chain_id, last_block, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (chain_id) DO UPDATE SET
    last_block = EXCLUDED.last_block,
    updated_at = NOW();
-- Always safe to reprocess
```

---

## Redis Fundamentals

### Data Structures

**1. Strings** (key-value)
```redis
SET block:1:18500000 '{"hash":"0xabc","timestamp":1699564800}'
GET block:1:18500000
SETEX block:1:18500000 300 '...'  # Expire in 300 seconds
```

**2. Hashes** (field-value pairs)
```redis
HSET user:1000 name "Alice" email "alice@example.com" balance 100
HGET user:1000 name  # Returns "Alice"
HINCRBY user:1000 balance 50  # Atomic increment
```

**3. Lists** (ordered collection)
```redis
LPUSH recent_blocks:1 18500000  # Push to head
LTRIM recent_blocks:1 0 99      # Keep only 100 most recent
LRANGE recent_blocks:1 0 9      # Get 10 most recent
```

**4. Sets** (unique values)
```redis
SADD active_chains 1 137 42161  # Add chain IDs
SISMEMBER active_chains 1       # Check if exists
SINTER set1 set2                # Intersection
```

**5. Sorted Sets** (scored values)
```redis
ZADD leaderboard 1000 "user1" 2000 "user2"
ZRANGE leaderboard 0 9 WITHSCORES  # Top 10
ZINCRBY leaderboard 100 "user1"    # Add to score
```

### Expiration Policies

**TTL (Time To Live)**
```redis
SET session:abc "data"
EXPIRE session:abc 3600  # Expire in 1 hour

SETEX session:abc 3600 "data"  # Atomic set + expire

TTL session:abc  # Returns remaining seconds
```

**Eviction policies** (when memory full):
```
noeviction       - Return errors, don't evict
allkeys-lru      - Evict least recently used (any key)
volatile-lru     - Evict LRU (only keys with TTL)
allkeys-random   - Evict random keys
volatile-random  - Evict random (only keys with TTL)
volatile-ttl     - Evict keys with shortest TTL
```

---

## Caching Strategies

### Cache-Aside (Lazy Loading)

**Read pattern:**
```go
func GetBlock(ctx context.Context, chainID, blockNum int64) (*Block, error) {
    key := fmt.Sprintf("block:%d:%d", chainID, blockNum)
    
    // Try cache first
    if data, err := redisClient.Get(ctx, key).Bytes(); err == nil {
        var block Block
        json.Unmarshal(data, &block)
        return &block, nil
    }
    
    // Cache miss - query database
    block, err := db.QueryBlock(ctx, chainID, blockNum)
    if err != nil {
        return nil, err
    }
    
    // Populate cache
    data, _ := json.Marshal(block)
    redisClient.Set(ctx, key, data, 10*time.Minute)
    
    return block, nil
}
```

**Trade-offs:**
- ✅ Simple to implement
- ✅ Only cache what's requested
- ❌ Cache miss penalty (extra latency)
- ❌ Stale data until TTL expires

### Write-Through (Always Consistent)

```go
func SaveBlock(ctx context.Context, block *Block) error {
    // Write to database first
    if err := db.Insert(ctx, block); err != nil {
        return err
    }
    
    // Update cache
    key := fmt.Sprintf("block:%d:%d", block.ChainID, block.Number)
    data, _ := json.Marshal(block)
    redisClient.Set(ctx, key, data, 10*time.Minute)
    
    return nil
}
```

**Trade-offs:**
- ✅ Cache always consistent with database
- ❌ Slower writes (two operations)
- ❌ Wastes cache space on infrequently accessed data

### Write-Behind (Asynchronous)

```go
func SaveBlock(ctx context.Context, block *Block) error {
    // Write to cache immediately
    key := fmt.Sprintf("block:%d:%d", block.ChainID, block.Number)
    data, _ := json.Marshal(block)
    redisClient.Set(ctx, key, data, 10*time.Minute)
    
    // Queue database write for later
    writeQueue <- block
    
    return nil
}

// Background worker
go func() {
    for block := range writeQueue {
        db.Insert(context.Background(), block)
    }
}()
```

**Trade-offs:**
- ✅ Very fast writes
- ❌ Risk of data loss if cache fails before DB write
- ❌ Complex error handling

### Cache Invalidation

**1. TTL-based (Time To Live)**
```go
// Cache expires after 5 minutes
redisClient.Set(ctx, key, data, 5*time.Minute)
```

**2. Event-driven (Kafka consumer)**
```go
// Listen for block updates
for msg := range kafkaConsumer.Consume() {
    var block Block
    json.Unmarshal(msg.Value, &block)
    
    // Invalidate cache
    key := fmt.Sprintf("block:%d:%d", block.ChainID, block.Number)
    redisClient.Del(ctx, key)
}
```

**3. Version tags**
```go
// Increment version on update
version := redisClient.Incr(ctx, "block:version").Val()
key := fmt.Sprintf("block:%d:%d:v%d", chainID, blockNum, version)
```

---

## Redis Advanced Patterns

### Distributed Rate Limiting

**Token bucket with sliding window:**
```go
func RateLimit(ctx context.Context, userID string, limit int, window time.Duration) (bool, error) {
    now := time.Now().Unix()
    key := fmt.Sprintf("ratelimit:%s:%d", userID, now/int64(window.Seconds()))
    
    // Increment counter
    count, err := redisClient.Incr(ctx, key).Result()
    if err != nil {
        return false, err
    }
    
    // Set expiration on first request
    if count == 1 {
        redisClient.Expire(ctx, key, window)
    }
    
    // Check limit
    return count <= int64(limit), nil
}

// Usage
allowed, _ := RateLimit(ctx, "user123", 100, 1*time.Minute)
if !allowed {
    return errors.New("rate limit exceeded")
}
```

**Sliding window with sorted sets:**
```go
func RateLimitSlidingWindow(ctx context.Context, userID string, limit int, window time.Duration) (bool, error) {
    now := time.Now()
    key := fmt.Sprintf("ratelimit:%s", userID)
    
    // Remove old entries
    cutoff := now.Add(-window).UnixNano()
    redisClient.ZRemRangeByScore(ctx, key, "0", fmt.Sprint(cutoff))
    
    // Count recent requests
    count := redisClient.ZCard(ctx, key).Val()
    if count >= int64(limit) {
        return false, nil
    }
    
    // Add current request
    redisClient.ZAdd(ctx, key, &redis.Z{
        Score:  float64(now.UnixNano()),
        Member: now.UnixNano(),
    })
    redisClient.Expire(ctx, key, window)
    
    return true, nil
}
```

### Pub/Sub for Real-Time Updates

**Publisher (API sends block updates):**
```go
func PublishBlock(ctx context.Context, chainID int64, block *Block) error {
    channel := fmt.Sprintf("blocks:chain:%d", chainID)
    data, _ := json.Marshal(block)
    return redisClient.Publish(ctx, channel, data).Err()
}
```

**Subscriber (Frontend receives updates):**
```go
func SubscribeBlocks(ctx context.Context, chainID int64) <-chan *Block {
    channel := fmt.Sprintf("blocks:chain:%d", chainID)
    pubsub := redisClient.Subscribe(ctx, channel)
    
    blocks := make(chan *Block)
    go func() {
        defer close(blocks)
        for msg := range pubsub.Channel() {
            var block Block
            json.Unmarshal([]byte(msg.Payload), &block)
            blocks <- &block
        }
    }()
    
    return blocks
}

// Usage
blocks := SubscribeBlocks(ctx, 1)
for block := range blocks {
    fmt.Printf("New block: %d\n", block.Number)
}
```

### Distributed Locks

**Simple lock with SETNX:**
```go
func AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
    return redisClient.SetNX(ctx, key, "locked", ttl).Result()
}

func ReleaseLock(ctx context.Context, key string) error {
    return redisClient.Del(ctx, key).Err()
}

// Usage
lockKey := "reorg:chain:1"
if acquired, _ := AcquireLock(ctx, lockKey, 30*time.Second); acquired {
    defer ReleaseLock(ctx, lockKey)
    handleReorg()
}
```

**Redlock algorithm (multiple Redis instances):**
```go
import "github.com/go-redsync/redsync/v4"

pool := goredis.NewPool(redisClient1, redisClient2, redisClient3)
rs := redsync.New(pool)

mutex := rs.NewMutex("reorg:chain:1")
if err := mutex.Lock(); err == nil {
    defer mutex.Unlock()
    handleReorg()
}
```

### Thundering Herd Prevention

**Problem**: Cache expires, 1000 requests hit DB simultaneously

**Solution 1: Probabilistic early expiration**
```go
func GetWithEarlyExpiration(ctx context.Context, key string, ttl time.Duration) (*Block, error) {
    // Random early expiration (10% chance in last 10% of TTL)
    if rand.Float64() < 0.1 {
        remaining := redisClient.TTL(ctx, key).Val()
        if remaining < ttl/10 {
            redisClient.Del(ctx, key)  // Force refresh
        }
    }
    
    // Normal cache-aside logic
    return GetBlock(ctx, key)
}
```

**Solution 2: Lock-based cache warming**
```go
func GetWithLock(ctx context.Context, key string) (*Block, error) {
    // Try cache
    if data, err := redisClient.Get(ctx, key).Bytes(); err == nil {
        var block Block
        json.Unmarshal(data, &block)
        return &block, nil
    }
    
    // Acquire lock to refresh
    lockKey := "lock:" + key
    if acquired, _ := redisClient.SetNX(ctx, lockKey, "1", 10*time.Second).Result(); acquired {
        defer redisClient.Del(ctx, lockKey)
        
        // Refresh cache
        block, _ := db.QueryBlock(ctx, key)
        data, _ := json.Marshal(block)
        redisClient.Set(ctx, key, data, 10*time.Minute)
        return block, nil
    }
    
    // Another request is refreshing, wait and retry
    time.Sleep(100 * time.Millisecond)
    return GetWithLock(ctx, key)
}
```

---

## Event-Driven Architecture

### Message Flow in Our System

```
Ingester → Kafka → Processor → Database
                            ↓
                      Redis Cache ← API → Frontend
```

**1. Ingester produces raw blocks:**
```go
producer.Produce(&kafka.Message{
    Topic: kafka.TopicPartition{Topic: strPtr("raw.blocks"), Partition: kafka.PartitionAny},
    Key:   []byte(fmt.Sprintf("%d", chainID)),
    Value: blockJSON,
}, nil)
```

**2. Processor consumes and enriches:**
```go
for msg := range consumer.Consume() {
    var block Block
    json.Unmarshal(msg.Value, &block)
    
    // Parse events, decode transactions
    events := parseEvents(block)
    
    // Store in database
    db.Insert(block, events)
    
    // Invalidate cache
    redisClient.Del(ctx, fmt.Sprintf("block:%d:%d", block.ChainID, block.Number))
}
```

**3. API serves from cache/database:**
```go
func GetBlock(c *gin.Context) {
    block, _ := GetFromCacheOrDB(chainID, blockNumber)
    c.JSON(200, block)
}
```

### Benefits of Event-Driven Architecture

**Decoupling:**
- Ingester doesn't know about processor
- Can add new consumers without changing ingester
- Services can be deployed independently

**Scalability:**
- Add more partitions to scale Kafka
- Add more consumer instances to process faster
- Each service scales independently

**Replay capability:**
- Reset offset to replay events
- Critical for blockchain reorgs
- Useful for bug fixes (reprocess historical data)

**Fault tolerance:**
- Kafka persists messages (default 7 days)
- Consumer can crash and resume from last offset
- No data loss

---

## Interview Questions

### Q1: Explain Kafka's ordering guarantees. How do you ensure ordered processing?

**Answer:**

Kafka guarantees **ordering within a partition**, but not across partitions.

**Within partition**: Messages are totally ordered
```
Partition 0: [Block 1000, Block 1001, Block 1002]
Consumer receives in exact order: 1000 → 1001 → 1002
```

**Across partitions**: No ordering
```
Partition 0: [Block 1000, Block 1002]
Partition 1: [Block 1001, Block 1003]
Consumer may receive: 1000, 1001, 1002, 1003 OR 1001, 1000, 1003, 1002
```

**To ensure ordered processing:**

**1. Use partition key** (same key → same partition):
```go
// All Ethereum blocks go to same partition
producer.Produce(&kafka.Message{
    Key:   []byte(fmt.Sprintf("chain:%d", chainID)),
    Value: blockJSON,
}, nil)
```

**2. Single partition** (if order critical across all messages):
```go
// Force all messages to partition 0
producer.Produce(&kafka.Message{
    Topic: kafka.TopicPartition{Topic: &topic, Partition: 0},
    Value: blockJSON,
}, nil)
```

**Trade-off**: Single partition = no parallelism, limited throughput.

### Q2: How do you achieve exactly-once semantics in Kafka?

**Answer:**

Exactly-once requires three components:

**1. Idempotent Producer** (prevents duplicate sends):
```go
producer, err := kafka.NewProducer(&kafka.ConfigMap{
    "enable.idempotence": true,  // Kafka assigns sequence numbers
})
```

**2. Transactional Producer** (atomic multi-partition writes):
```go
producer.InitTransactions(nil)
producer.BeginTransaction()
producer.Produce(&kafka.Message{Topic: &topic1, Value: data1}, nil)
producer.Produce(&kafka.Message{Topic: &topic2, Value: data2}, nil)
producer.CommitTransaction(nil)  // Both or neither
```

**3. Read Committed Consumer** (only see committed transactions):
```go
consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
    "isolation.level": "read_committed",
})
```

**Combined with idempotent database operations:**
```sql
INSERT INTO blocks (chain_id, block_number, hash)
VALUES ($1, $2, $3)
ON CONFLICT (chain_id, block_number) DO NOTHING;
```

**Result**: Crash at any point = message processed exactly once.

### Q3: Explain cache-aside vs write-through caching. When would you use each?

**Answer:**

**Cache-Aside (Lazy Loading)**:
1. Check cache
2. If miss, query database
3. Populate cache
4. Return data

```go
func Get(key string) (value, error) {
    if cached := redis.Get(key); cached != nil {
        return cached  // Cache hit
    }
    value := db.Query(key)  // Cache miss
    redis.Set(key, value, ttl)
    return value
}
```

**Pros**: Only cache what's accessed, simple
**Cons**: Cache miss penalty, stale data until expiration
**Use when**: Read-heavy, tolerate staleness

**Write-Through**:
1. Write to database
2. Write to cache
3. Return success

```go
func Set(key, value string) error {
    db.Insert(key, value)
    redis.Set(key, value, ttl)
    return nil
}
```

**Pros**: Cache always consistent, no stale data
**Cons**: Slower writes, wastes cache on infrequent data
**Use when**: Strong consistency required, frequent reads after writes

**For blockchain indexing**: Cache-aside with short TTL (5 min)
- Blocks are immutable (no consistency issues)
- Only recent blocks accessed frequently
- TTL handles cache invalidation automatically

### Q4: How does Kafka handle consumer failures and rebalancing?

**Answer:**

**Consumer Group Coordination**:
- Each consumer sends heartbeat to group coordinator
- If heartbeat stops, coordinator marks consumer dead
- Triggers rebalance to reassign partitions

**Rebalance Protocol**:
1. Coordinator detects change (new consumer, dead consumer, partition count change)
2. All consumers stop processing
3. Coordinator reassigns partitions using strategy (range, round-robin, sticky)
4. Consumers receive new assignments
5. Resume processing from last committed offset

**Example scenario**:
```
Before:
  Consumer 1: Partitions [0, 1]
  Consumer 2: Partitions [2, 3]

Consumer 2 crashes:

After rebalance:
  Consumer 1: Partitions [0, 1, 2, 3]  # Took over crashed consumer's work
```

**Offset management**:
- Consumer commits offset after processing
- On rebalance, new consumer starts from last committed offset
- If offset not committed, may reprocess messages (at-least-once)

**Configuration**:
```go
"session.timeout.ms": 10000,  // Max time without heartbeat before considered dead
"heartbeat.interval.ms": 3000,  // How often to send heartbeat
"max.poll.interval.ms": 300000,  // Max time between polls
```

### Q5: What's the thundering herd problem and how do you solve it?

**Answer:**

**Problem**: Many requests hit expired cache key simultaneously, all query database in parallel.

**Scenario**:
```
Time 0s: Cache key "block:1:18500000" expires
Time 1s: 1000 concurrent requests arrive
Result: All 1000 requests query database (database overload)
```

**Solution 1: Lock-based cache refresh**:
```go
func GetWithLock(key string) (value, error) {
    // Try cache
    if cached := redis.Get(key); cached != nil {
        return cached
    }
    
    // Acquire lock to refresh
    if redis.SetNX("lock:"+key, "1", 10*time.Second) {
        defer redis.Del("lock:" + key)
        
        // Refresh cache
        value := db.Query(key)
        redis.Set(key, value, ttl)
        return value
    }
    
    // Another request is refreshing, wait and retry
    time.Sleep(100 * time.Millisecond)
    return GetWithLock(key)
}
```

Only one request queries database, others wait.

**Solution 2: Probabilistic early expiration**:
```go
// Randomly refresh cache before expiration (in last 10% of TTL)
if rand.Float64() < 0.1 {
    remaining := redis.TTL(key)
    if remaining < ttl/10 {
        redis.Del(key)  // Force single request to refresh
    }
}
```

Spreads cache refreshes over time, avoids simultaneous expirations.

**Solution 3: Cache warming**:
```go
// Background job refreshes popular keys before expiration
go func() {
    for {
        time.Sleep(1 * time.Minute)
        for _, key := range popularKeys {
            value := db.Query(key)
            redis.Set(key, value, ttl)
        }
    }
}()
```

### Q6: How do you implement distributed rate limiting with Redis?

**Answer:**

**Sliding Window with Sorted Sets**:

```go
func RateLimit(userID string, limit int, window time.Duration) (bool, error) {
    now := time.Now()
    key := fmt.Sprintf("ratelimit:%s", userID)
    
    // Remove entries older than window
    cutoff := now.Add(-window).UnixNano()
    redis.ZRemRangeByScore(key, "0", fmt.Sprint(cutoff))
    
    // Count recent requests
    count := redis.ZCard(key)
    if count >= limit {
        return false, nil  // Rate limit exceeded
    }
    
    // Add current request
    redis.ZAdd(key, now.UnixNano(), now.UnixNano())
    redis.Expire(key, window)
    
    return true, nil
}
```

**Why sorted sets?**
- Score = timestamp (sorted by time)
- `ZRemRangeByScore` removes old entries (O(log N))
- `ZCard` counts remaining entries (O(1))
- Accurate sliding window (vs fixed window with counters)

**Fixed window alternative (less accurate, faster)**:
```go
func RateLimitFixed(userID string, limit int, window time.Duration) (bool, error) {
    now := time.Now().Unix()
    key := fmt.Sprintf("ratelimit:%s:%d", userID, now/int64(window.Seconds()))
    
    count := redis.Incr(key)
    if count == 1 {
        redis.Expire(key, window)
    }
    
    return count <= limit, nil
}
```

**Trade-offs:**
| Approach | Accuracy | Performance | Use Case |
|----------|----------|-------------|----------|
| Sorted Set | ✅ Precise | 🐢 O(log N) | Financial APIs |
| Fixed Window | ❌ Can burst at boundary | ⚡ O(1) | General APIs |
| Token Bucket | ✅ Smooth | ⚡ O(1) | High throughput |

### Q7: Explain Redis Pub/Sub vs Kafka. When would you use each?

**Answer:**

**Redis Pub/Sub**:
- Fire-and-forget messaging
- No persistence (if subscriber offline, message lost)
- Subscriber receives only messages published while connected
- Low latency (microseconds)

**Kafka**:
- Persisted messaging (default 7 days)
- Subscribers can replay messages
- Consumer groups for load balancing
- Higher latency (milliseconds)

**Comparison:**

| Feature | Redis Pub/Sub | Kafka |
|---------|---------------|-------|
| Persistence | ❌ No | ✅ Yes (configurable) |
| Replay | ❌ No | ✅ Yes |
| Ordering | ❌ No guarantee | ✅ Per partition |
| Throughput | 🐢 ~50K msg/sec | ⚡ ~100K msg/sec/partition |
| Latency | ⚡ <1ms | 🐢 ~10ms |
| Consumer Groups | ❌ No | ✅ Yes |

**Use Redis Pub/Sub when:**
- Real-time notifications (WebSocket updates)
- Low latency critical
- Losing messages acceptable
- Simple broadcast (all subscribers get message)

**Use Kafka when:**
- Need persistence
- Replay capability required
- Complex processing pipelines
- High throughput
- Multiple consumers need coordination

**Example - Our system:**
- **Kafka**: Ingester → Processor (need persistence, replay for reorgs)
- **Redis Pub/Sub**: API → Frontend (real-time block updates, can miss occasional message)

### Q8: How would you debug Kafka consumer lag?

**Answer:**

**1. Check consumer lag**:
```bash
# Kafka command-line tools
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group indexer-processor --describe

# Output shows lag per partition:
GROUP           TOPIC     PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG
indexer-proc    blocks    0          1000            1500            500  # 500 messages behind
```

**2. Common causes**:

**Slow processing**:
```go
// Check message processing time
start := time.Now()
processMessage(msg)
duration := time.Since(start)
if duration > 1*time.Second {
    log.Printf("Slow processing: %v", duration)
}
```

**Database bottleneck**:
```sql
-- Find slow queries
SELECT query, mean_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

**Too few consumers**:
```
3 partitions, 1 consumer = one consumer handles all load
Solution: Add consumers (up to number of partitions)
```

**Consumer rebalancing**:
```go
// Log rebalances
consumer.Subscribe("topic", &kafka.RebalanceCb{
    Partitions: func(c *kafka.Consumer, event kafka.Event) error {
        log.Printf("Rebalance: %v", event)
        return nil
    },
})
```

**3. Solutions**:

**Batch processing**:
```go
// Process in batches instead of one-by-one
batch := []Message{}
for i := 0; i < 100; i++ {
    msg, _ := consumer.ReadMessage(100 * time.Millisecond)
    batch = append(batch, msg)
}
processBatch(batch)  // Single DB transaction
consumer.CommitMessages(batch)
```

**Parallel processing**:
```go
// Process messages in parallel (careful with ordering!)
for i := 0; i < 10; i++ {
    go func() {
        for msg := range msgChan {
            processMessage(msg)
        }
    }()
}
```

**Scale consumers**:
```bash
# Add more consumer instances
docker-compose up --scale processor=5
```

---

## Related Documentation

- [Database-Fundamentals.md](./Database-Fundamentals.md) - Database theory
- [Go-Programming.md](./Go-Programming.md) - Go concurrency patterns
- [System-Design-Architecture.md](./System-Design-Architecture.md) - Architecture patterns
- [Deployment-Production.md](./Deployment-Production.md) - Production setup

---

**Last Updated**: 2025-11-27
