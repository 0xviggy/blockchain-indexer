# Database Fundamentals

> **Purpose**: Core database theory and concepts applicable across all database systems - ACID properties, normalization, indexing theory, transactions, consistency models, and database design principles.

---

## Table of Contents
- [ACID Properties](#acid-properties)
- [Database Normalization](#database-normalization)
- [Indexing Theory](#indexing-theory)
- [Transaction Isolation Levels](#transaction-isolation-levels)
- [CAP Theorem](#cap-theorem)
- [Replication & Consistency](#replication--consistency)
- [Sharding Strategies](#sharding-strategies)
- [SQL vs NoSQL](#sql-vs-nosql)
- [Database Design Principles](#database-design-principles)
- [Interview Questions](#interview-questions)

---

## ACID Properties

ACID is the foundation of relational database reliability. It guarantees that database transactions are processed reliably.

### A - Atomicity

**All or nothing**: A transaction's operations either all succeed or all fail together.

**Example: Bank Transfer**
```sql
BEGIN TRANSACTION;
    UPDATE accounts SET balance = balance - 100 WHERE user_id = 1;  -- Withdraw
    UPDATE accounts SET balance = balance + 100 WHERE user_id = 2;  -- Deposit
COMMIT;
```

If the second UPDATE fails (e.g., user_id 2 doesn't exist), the first UPDATE is rolled back. You never end up with money deducted but not credited.

**In blockchain indexing:**
```sql
BEGIN TRANSACTION;
    DELETE FROM events WHERE block_number > 18500000;
    DELETE FROM transactions WHERE block_number > 18500000;
    DELETE FROM blocks WHERE block_number > 18500000;
    UPDATE checkpoints SET last_block = 18500000 WHERE chain_id = 1;
COMMIT;
```

If any DELETE fails during reorg handling, the entire rollback is aborted, preventing partial/corrupted state.

### C - Consistency

**Data always valid**: The database enforces all rules (constraints, triggers, cascades) before committing.

**Example: Foreign Keys**
```sql
CREATE TABLE blocks (
    id BIGSERIAL PRIMARY KEY,
    chain_id INTEGER NOT NULL,
    block_number BIGINT NOT NULL
);

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    block_id BIGINT NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    tx_hash TEXT NOT NULL
);
```

You **cannot** insert a transaction with `block_id = 999` if block 999 doesn't exist. The database rejects the INSERT, maintaining referential integrity.

**Other consistency mechanisms:**
- `CHECK` constraints: `CHECK (balance >= 0)`
- `UNIQUE` constraints: `UNIQUE (chain_id, block_number)`
- `NOT NULL` constraints
- Triggers: Auto-update `updated_at` timestamp

### I - Isolation

**Transactions don't see each other's uncommitted changes**: Concurrent transactions appear to execute sequentially.

**Without isolation (broken example):**
```
Time    Transaction 1                   Transaction 2
----    ---------------------------     ---------------------------
t1      BEGIN
t2      balance = 100 (read)
t3                                      BEGIN
t4                                      balance = 100 (read)
t5      balance = balance - 50 (50)
t6                                      balance = balance - 30 (70)
t7      COMMIT (balance = 50)
t8                                      COMMIT (balance = 70)
```

Final balance is 70, but should be 20. Transaction 2 overwrote Transaction 1's change because it didn't see the uncommitted update.

**With isolation (correct):**
Transaction 2 waits for Transaction 1 to commit, then reads the updated value (50), and correctly computes 50 - 30 = 20.

### D - Durability

**Committed data survives crashes**: Once a transaction commits, the changes are permanent, even if the server crashes 1 second later.

**Mechanism**: Write-Ahead Logging (WAL)
1. Database writes changes to a durable log file (WAL) **before** modifying data files
2. On crash, database replays the WAL to restore committed transactions
3. PostgreSQL WAL file: `/var/lib/postgresql/data/pg_wal/`

**Production considerations:**
- WAL files stored on separate disk for performance
- WAL archival to S3 for point-in-time recovery
- Synchronous vs asynchronous commit trade-off (durability vs latency)

---

## Database Normalization

Normalization eliminates redundancy and ensures data integrity by organizing data into tables according to specific rules.

### First Normal Form (1NF)

**Rule**: No repeating groups or arrays; each field contains atomic values.

**Violation:**
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT,
    emails TEXT  -- "alice@x.com, alice@y.com"  ❌ Multiple values
);
```

**Fixed (1NF compliant):**
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT
);

CREATE TABLE user_emails (
    user_id INTEGER REFERENCES users(id),
    email TEXT,
    PRIMARY KEY (user_id, email)
);
```

### Second Normal Form (2NF)

**Rule**: Meets 1NF + no partial dependencies (every non-key column depends on the **entire** primary key).

**Violation:**
```sql
CREATE TABLE order_items (
    order_id INTEGER,
    product_id INTEGER,
    quantity INTEGER,
    product_name TEXT,  -- ❌ Depends only on product_id, not (order_id, product_id)
    PRIMARY KEY (order_id, product_id)
);
```

**Problem**: `product_name` is stored redundantly in every order containing that product.

**Fixed (2NF compliant):**
```sql
CREATE TABLE products (
    product_id INTEGER PRIMARY KEY,
    product_name TEXT
);

CREATE TABLE order_items (
    order_id INTEGER,
    product_id INTEGER REFERENCES products(product_id),
    quantity INTEGER,
    PRIMARY KEY (order_id, product_id)
);
```

### Third Normal Form (3NF)

**Rule**: Meets 2NF + no transitive dependencies (non-key columns depend only on primary key, not on other non-key columns).

**Violation:**
```sql
CREATE TABLE employees (
    employee_id INTEGER PRIMARY KEY,
    name TEXT,
    department_id INTEGER,
    department_name TEXT,  -- ❌ Depends on department_id, not employee_id
    department_location TEXT  -- ❌ Same issue
);
```

**Fixed (3NF compliant):**
```sql
CREATE TABLE departments (
    department_id INTEGER PRIMARY KEY,
    department_name TEXT,
    department_location TEXT
);

CREATE TABLE employees (
    employee_id INTEGER PRIMARY KEY,
    name TEXT,
    department_id INTEGER REFERENCES departments(department_id)
);
```

### Denormalization Trade-offs

Sometimes we **intentionally violate** normalization for performance.

**Example: Blockchain Indexer**
```sql
-- Normalized (3NF)
CREATE TABLE blocks (
    block_number BIGINT PRIMARY KEY,
    timestamp BIGINT
);

CREATE TABLE transactions (
    tx_hash TEXT PRIMARY KEY,
    block_number BIGINT REFERENCES blocks(block_number),
    from_address TEXT
);

-- Query requires JOIN
SELECT tx_hash, timestamp FROM transactions 
JOIN blocks ON transactions.block_number = blocks.block_number
WHERE from_address = '0x123';
```

**Denormalized (violates 3NF, but faster):**
```sql
CREATE TABLE transactions (
    tx_hash TEXT PRIMARY KEY,
    block_number BIGINT,
    block_timestamp BIGINT,  -- ❌ Duplicate from blocks table
    from_address TEXT
);

-- No JOIN needed
SELECT tx_hash, block_timestamp FROM transactions 
WHERE from_address = '0x123';
```

**Trade-offs:**
- ✅ **Faster reads**: No JOIN, direct column access
- ❌ **Slower writes**: Must update `block_timestamp` in transactions table if block data changes (rare for immutable blockchain data)
- ❌ **More storage**: Duplicate data across tables
- ❌ **Risk of inconsistency**: If update logic has bugs

**When to denormalize:**
- Read-heavy workloads (100:1 read/write ratio)
- Immutable data (blockchain blocks never change once confirmed)
- Query performance is critical (sub-100ms SLA)

---

## Indexing Theory

Indexes are data structures that improve query speed by avoiding full table scans.

### Index Types

#### B-Tree Index (Default)

**Structure**: Balanced tree where each node contains sorted keys.

```
               [50]
              /    \
          [25]      [75]
         /   \      /   \
     [10] [30]  [60] [90]
```

**Best for:**
- Equality queries: `WHERE id = 100`
- Range queries: `WHERE created_at BETWEEN '2024-01-01' AND '2024-12-31'`
- Sorting: `ORDER BY timestamp DESC`
- Prefix matching: `WHERE name LIKE 'John%'` (but NOT `WHERE name LIKE '%John'`)

**Performance**: O(log n) lookup time

#### Hash Index

**Structure**: Hash table mapping keys to row pointers.

**Best for:**
- Exact equality queries only: `WHERE tx_hash = '0x123...'`

**Cannot handle:**
- Range queries: `WHERE block_number > 1000`
- Sorting: `ORDER BY`
- Prefix matching: `LIKE`

**Performance**: O(1) lookup time, but limited use cases

#### Composite Index

**Structure**: B-tree with multiple columns.

```sql
CREATE INDEX idx_chain_block ON blocks (chain_id, block_number);
```

**Column order matters:**

✅ **Works well:**
```sql
WHERE chain_id = 1 AND block_number > 18000000  -- Uses full index
WHERE chain_id = 1  -- Uses first column of index
```

❌ **Cannot use index efficiently:**
```sql
WHERE block_number > 18000000  -- chain_id not specified, cannot use index
```

**Rule**: Put most selective (high cardinality) columns first, unless query patterns dictate otherwise.

#### Partial Index

**Structure**: B-tree that indexes only rows matching a condition.

```sql
CREATE INDEX idx_pending_tx ON transactions (created_at) 
WHERE status = 'pending';
```

**Benefits:**
- Smaller index (faster to scan, less storage)
- Only useful for queries that include the WHERE clause

**Example:**
```sql
-- Uses index
SELECT * FROM transactions WHERE status = 'pending' AND created_at > '2024-01-01';

-- Cannot use index (status != 'pending')
SELECT * FROM transactions WHERE status = 'confirmed' AND created_at > '2024-01-01';
```

### Index Trade-offs

| Aspect | Impact |
|--------|--------|
| **Read performance** | ✅ Up to 1000x faster (table scan → index seek) |
| **Write performance** | ❌ Slower (must update index on INSERT/UPDATE/DELETE) |
| **Storage** | ❌ 30-50% overhead per index |
| **Maintenance** | ❌ Requires VACUUM, REINDEX on high-write tables |

### Identifying Missing Indexes

**Check for sequential scans (slow):**
```sql
-- Find slow queries
SELECT query, calls, total_exec_time, mean_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Check if indexes are being used
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read
FROM pg_stat_user_indexes
WHERE idx_scan = 0  -- Unused indexes (consider dropping)
ORDER BY idx_scan ASC;
```

**Using EXPLAIN:**
```sql
EXPLAIN ANALYZE 
SELECT * FROM transactions WHERE from_address = '0x123';

-- Bad: Seq Scan on transactions (cost=0.00..1000.00 rows=10)
-- Good: Index Scan using idx_from_addr (cost=0.42..8.44 rows=10)
```

### When to Index

✅ **Index these:**
- Primary keys (automatic)
- Foreign keys (for JOINs)
- Columns in WHERE clauses with high selectivity
- Columns in ORDER BY clauses
- Columns in JOIN conditions

❌ **Don't index these:**
- Small tables (< 10,000 rows) - full scan is faster
- Low cardinality columns (`gender` with 2 values: M/F)
- Frequently updated columns (write overhead > read benefit)
- Columns not used in queries

---

## Transaction Isolation Levels

Isolation levels control what uncommitted data one transaction can see from another.

### Read Uncommitted (Weakest)

**Dirty Reads**: Can see uncommitted changes from other transactions.

```sql
-- Transaction 1
BEGIN TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
UPDATE accounts SET balance = 1000 WHERE id = 1;
-- Not committed yet

-- Transaction 2 (reads uncommitted value)
SELECT balance FROM accounts WHERE id = 1;  -- Returns 1000

-- Transaction 1 rolls back
ROLLBACK;

-- Transaction 2 read "dirty" data that never existed
```

**Problem**: Reading data that might get rolled back.

**Use case**: Almost never. PostgreSQL doesn't even support this level (treats as Read Committed).

### Read Committed (Default)

**Prevents**: Dirty reads

**Allows**: Non-repeatable reads

```sql
-- Transaction 1
BEGIN;
SELECT balance FROM accounts WHERE id = 1;  -- Returns 500

-- Transaction 2 (commits change)
UPDATE accounts SET balance = 1000 WHERE id = 1;
COMMIT;

-- Transaction 1 (reads again, sees different value)
SELECT balance FROM accounts WHERE id = 1;  -- Returns 1000
```

**Problem**: Same query returns different results within a transaction.

**Use case**: Default for most applications. Good balance of consistency and concurrency.

### Repeatable Read

**Prevents**: Dirty reads, non-repeatable reads

**Allows**: Phantom reads

```sql
SET TRANSACTION ISOLATION LEVEL REPEATABLE READ;

BEGIN;
SELECT COUNT(*) FROM orders WHERE status = 'pending';  -- Returns 10

-- Another transaction inserts a new pending order
INSERT INTO orders (status) VALUES ('pending');
COMMIT;

-- This transaction sees same count (no phantom reads in PostgreSQL's implementation)
SELECT COUNT(*) FROM orders WHERE status = 'pending';  -- Still 10
```

**Note**: PostgreSQL's implementation actually prevents phantom reads too (uses snapshot isolation), so it's closer to Serializable.

**Use case**: Financial transactions, reporting queries that require consistency.

### Serializable (Strongest)

**Prevents**: All anomalies (dirty reads, non-repeatable reads, phantom reads, serialization anomalies)

**Guarantee**: Transactions appear to execute serially, even if concurrent.

```sql
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;

BEGIN;
SELECT SUM(balance) FROM accounts;  -- Returns 10000

-- Another transaction transfers money between accounts
UPDATE accounts SET balance = balance - 100 WHERE id = 1;
UPDATE accounts SET balance = balance + 100 WHERE id = 2;
COMMIT;

-- This transaction still sees same sum (snapshot isolation)
SELECT SUM(balance) FROM accounts;  -- Still 10000
COMMIT;
```

**Cost**: Potential serialization failures (transaction might abort with "could not serialize access" error).

**Use case**: Critical operations where absolute consistency is required (banking, inventory management).

### Comparison Table

| Isolation Level | Dirty Reads | Non-Repeatable Reads | Phantom Reads | Performance |
|-----------------|-------------|----------------------|---------------|-------------|
| Read Uncommitted | ❌ Possible | ❌ Possible | ❌ Possible | ⚡ Fastest |
| Read Committed | ✅ Prevented | ❌ Possible | ❌ Possible | ⚡ Fast |
| Repeatable Read | ✅ Prevented | ✅ Prevented | ❌ Possible* | 🐢 Slower |
| Serializable | ✅ Prevented | ✅ Prevented | ✅ Prevented | 🐢 Slowest |

*PostgreSQL's MVCC prevents phantom reads even at Repeatable Read level.

---

## CAP Theorem

**CAP Theorem**: In a distributed database, you can only guarantee 2 of 3 properties:

### Consistency (C)

All nodes see the same data at the same time. A read after a write always returns the latest value.

**Example**: Bank account balance must be consistent across all replicas immediately after a transfer.

### Availability (A)

Every request receives a response (success or failure), even if some nodes are down.

**Example**: System continues serving requests even if 1 of 3 database replicas crashes.

### Partition Tolerance (P)

The system continues operating despite network partitions (nodes can't communicate).

**Example**: Data center 1 can't reach data center 2, but both continue serving requests.

### The Trade-off

**In practice, you must choose P** (network failures are inevitable), so the real choice is:

#### CP Systems (Consistency + Partition Tolerance)

**Sacrifice availability during partitions**.

**Examples**: 
- PostgreSQL with synchronous replication
- HBase
- MongoDB (with w=majority)

**Behavior during partition**: Refuse writes to maintain consistency, return errors instead.

**Use case**: Financial systems, inventory management (can't sell same item twice).

#### AP Systems (Availability + Partition Tolerance)

**Sacrifice consistency during partitions** (eventual consistency).

**Examples**:
- Cassandra (tunable consistency)
- DynamoDB
- Riak

**Behavior during partition**: Accept writes on both sides, resolve conflicts later (last-write-wins, vector clocks).

**Use case**: Social media feeds, caching, analytics (stale data is acceptable).

### Our Choice: CP (PostgreSQL)

For blockchain indexing, **consistency is critical**:
- Cannot serve stale block data (users would see wrong balances)
- Reorg handling requires atomic rollbacks across multiple tables
- Financial data must be accurate

We accept that during a database outage, the API returns errors (503 Service Unavailable) rather than serving potentially incorrect data.

---

## Replication & Consistency

Replication improves availability and read scalability by maintaining multiple copies of data.

### Replication Types

#### Primary-Replica (Master-Slave)

**Structure**: 1 primary accepts writes, N replicas handle reads.

```
      [Primary]
       /  |  \
      /   |   \
  [Replica1] [Replica2] [Replica3]
```

**Write path**: Client → Primary → Replicas (async or sync)

**Read path**: Client → Any replica

**Pros:**
- ✅ Simple to implement
- ✅ Scales reads horizontally
- ✅ No write conflicts

**Cons:**
- ❌ Single point of failure (primary)
- ❌ Replication lag (reads might be stale)
- ❌ Cannot scale writes

#### Multi-Primary (Multi-Master)

**Structure**: Multiple nodes accept writes, replicate to each other.

**Pros:**
- ✅ No single point of failure
- ✅ Scales writes across regions

**Cons:**
- ❌ Write conflicts (two primaries update same row)
- ❌ Complex conflict resolution
- ❌ Harder to maintain consistency

**Use case**: Rare. Mostly for geo-distributed systems (e.g., CockroachDB).

### Consistency Models

#### Strong Consistency

**Guarantee**: Read always returns the latest write.

**Implementation**: Synchronous replication
```sql
-- PostgreSQL synchronous replication
ALTER SYSTEM SET synchronous_commit = 'on';
ALTER SYSTEM SET synchronous_standby_names = 'replica1,replica2';

-- Write only completes after replicas confirm
INSERT INTO blocks VALUES (...);  -- Blocks until replicas ACK
```

**Trade-offs:**
- ✅ Always consistent
- ❌ Slower writes (wait for network round-trip)
- ❌ Unavailable if replicas unreachable

#### Eventual Consistency

**Guarantee**: Reads will eventually see latest write (after some delay).

**Implementation**: Asynchronous replication
```sql
-- PostgreSQL async replication (default)
ALTER SYSTEM SET synchronous_commit = 'off';

-- Write completes immediately, replicas catch up later
INSERT INTO blocks VALUES (...);  -- Returns instantly
```

**Trade-offs:**
- ✅ Fast writes
- ✅ Available during partition
- ❌ Replicas might be seconds/minutes behind (replication lag)

**Measuring replication lag:**
```sql
-- On replica
SELECT now() - pg_last_xact_replay_timestamp() AS replication_lag;
```

#### Read-Your-Writes Consistency

**Guarantee**: A user always sees their own writes, even if reading from replica.

**Implementation**: Route user's reads to primary for N seconds after their write, then allow replica reads.

```go
func (api *API) WriteBlock(block Block) {
    db.Primary.Insert(block)  // Write to primary
    redis.Set("user:123:must_read_primary", "true", 5*time.Second)  // Flag for 5s
}

func (api *API) ReadBlock(blockNum int64, userID string) {
    if redis.Exists("user:" + userID + ":must_read_primary") {
        return db.Primary.Get(blockNum)  // Read from primary
    }
    return db.Replica.Get(blockNum)  // Read from replica
}
```

---

## Sharding Strategies

Sharding distributes data across multiple databases to scale beyond a single machine's capacity.

### Horizontal Sharding (Sharding)

**Concept**: Split rows across multiple databases.

```
Database 1: users where id % 3 = 0
Database 2: users where id % 3 = 1
Database 3: users where id % 3 = 2
```

**Types:**

#### 1. Hash-Based Sharding

**Method**: `shard = hash(shard_key) % num_shards`

```python
def get_shard(user_id, num_shards=4):
    return hash(user_id) % num_shards

# user_id 12345 → Shard 1
# user_id 67890 → Shard 2
```

**Pros:**
- ✅ Even distribution
- ✅ Simple logic

**Cons:**
- ❌ Hard to rebalance (adding shards requires rehashing everything)
- ❌ Range queries span all shards

#### 2. Range-Based Sharding

**Method**: Partition by ranges of shard key.

```
Shard 1: users where id BETWEEN 1 AND 1,000,000
Shard 2: users where id BETWEEN 1,000,001 AND 2,000,000
Shard 3: users where id BETWEEN 2,000,001 AND 3,000,000
```

**Pros:**
- ✅ Range queries stay within one shard
- ✅ Easy to add new shards

**Cons:**
- ❌ Uneven distribution (hotspots if new users always on latest shard)

#### 3. Geographic Sharding

**Method**: Shard by user location.

```
Shard US-East: users in Eastern US
Shard US-West: users in Western US
Shard EU: users in Europe
```

**Pros:**
- ✅ Low latency (data near users)
- ✅ Compliance (GDPR data locality)

**Cons:**
- ❌ Uneven distribution
- ❌ Cross-shard queries slow (e.g., global analytics)

### Vertical Sharding (Vertical Partitioning)

**Concept**: Split columns across databases.

```
Database 1: user_id, username, email
Database 2: user_id, profile_image, bio, preferences
```

**Use case**: Separate hot columns (frequently accessed) from cold columns (rarely accessed).

### Sharding Trade-offs

**Benefits:**
- ✅ Scale beyond single machine (100M → 1B rows)
- ✅ Isolate failures (one shard down doesn't affect others)

**Challenges:**
- ❌ Cross-shard queries slow (fan-out to all shards, merge results)
- ❌ Cross-shard transactions hard (two-phase commit required)
- ❌ Rebalancing data when adding shards is complex
- ❌ Application logic must be shard-aware

**For blockchain indexing:**
```sql
-- Natural sharding key: chain_id
Shard 1: Ethereum (chain_id = 1)
Shard 2: Polygon (chain_id = 137)
Shard 3: Arbitrum (chain_id = 42161)
```

---

## SQL vs NoSQL

### SQL Databases (Relational)

**Examples**: PostgreSQL, MySQL, SQL Server

**Characteristics:**
- Fixed schema (columns defined upfront)
- ACID transactions
- JOINs across tables
- Vertical scaling (bigger server)

**Pros:**
- ✅ Strong consistency
- ✅ Flexible queries (JOINs, aggregations)
- ✅ Referential integrity (foreign keys)
- ✅ Mature tooling and knowledge base

**Cons:**
- ❌ Schema changes require migrations
- ❌ Harder to scale horizontally
- ❌ Slower for massive writes (ACID overhead)

**Use cases:**
- Financial systems
- E-commerce
- CRM/ERP systems
- Blockchain indexers (need ACID for reorgs)

### NoSQL Databases

#### Document Store (MongoDB, CouchDB)

**Data model**: JSON documents

```json
{
  "_id": "block_18500000",
  "chain_id": 1,
  "timestamp": 1699564800,
  "transactions": [
    {"hash": "0x123...", "from": "0xabc..."},
    {"hash": "0x456...", "from": "0xdef..."}
  ]
}
```

**Pros:**
- ✅ Flexible schema (add fields anytime)
- ✅ Horizontally scalable
- ✅ Natural fit for hierarchical data

**Cons:**
- ❌ No JOINs (denormalization required)
- ❌ Eventual consistency
- ❌ Large document updates are inefficient

#### Wide-Column Store (Cassandra, HBase)

**Data model**: Rows with dynamic columns

```
RowKey: user:12345
Columns: {
  "email": "alice@example.com",
  "login:2024-01-01": "true",
  "login:2024-01-02": "true",
  ...
}
```

**Pros:**
- ✅ Massive scale (billions of rows)
- ✅ High write throughput
- ✅ Good for time-series data

**Cons:**
- ❌ No JOINs
- ❌ Eventual consistency
- ❌ Query flexibility limited (must design schema for query patterns)

#### Key-Value Store (Redis, DynamoDB)

**Data model**: Simple key → value

```
"block:1:18500000" → {"hash": "0x123...", "timestamp": 1699564800}
```

**Pros:**
- ✅ Extremely fast (O(1) lookups)
- ✅ Simple API (GET, SET, DELETE)

**Cons:**
- ❌ No complex queries
- ❌ No relationships between keys

**Use case**: Caching, session storage, rate limiting

### Choosing SQL vs NoSQL

**Choose SQL if:**
- Strong consistency is critical
- Complex relationships between entities
- Ad-hoc queries required
- ACID transactions needed

**Choose NoSQL if:**
- Massive scale (billions of rows)
- High write throughput (millions/sec)
- Schema flexibility required
- Eventual consistency acceptable

**For blockchain indexing: SQL (PostgreSQL)**
- Reorgs require ACID transactions
- Need JOINs for analytics (blocks ↔ transactions ↔ events)
- Strong consistency (no stale data)
- Scale is manageable with partitioning (millions of rows)

---

## Database Design Principles

### Entity-Relationship (ER) Diagrams

Visual representation of database structure.

**Example: Blockchain Indexer**

```
┌─────────────┐         ┌─────────────────┐         ┌──────────────┐
│   Chains    │         │     Blocks      │         │Transactions  │
├─────────────┤         ├─────────────────┤         ├──────────────┤
│ id (PK)     │◄──┐     │ id (PK)         │◄──┐     │ id (PK)      │
│ name        │   │     │ chain_id (FK)   │   │     │ block_id (FK)│
│ rpc_url     │   └─────│ block_number    │   └─────│ tx_hash      │
│ block_time  │         │ hash            │         │ from_address │
└─────────────┘         │ timestamp       │         │ to_address   │
                        └─────────────────┘         │ value        │
                                                    └──────────────┘
```

**Relationships:**
- One chain has many blocks (1:N)
- One block has many transactions (1:N)

### Primary Keys

**Purpose**: Uniquely identify each row.

**Types:**

#### Natural Key

**Example**: `(chain_id, block_number)` uniquely identifies a block

```sql
CREATE TABLE blocks (
    chain_id INTEGER,
    block_number BIGINT,
    PRIMARY KEY (chain_id, block_number)
);
```

**Pros:**
- ✅ Meaningful to humans
- ✅ No extra column needed

**Cons:**
- ❌ Composite keys are verbose in foreign keys
- ❌ If key changes, must update all referencing rows

#### Surrogate Key

**Example**: Auto-incrementing ID

```sql
CREATE TABLE blocks (
    id BIGSERIAL PRIMARY KEY,
    chain_id INTEGER,
    block_number BIGINT,
    UNIQUE (chain_id, block_number)
);
```

**Pros:**
- ✅ Simple to reference (single column)
- ✅ Immutable (never changes)
- ✅ Smaller foreign keys

**Cons:**
- ❌ Extra column
- ❌ No inherent meaning

**Best practice**: Use surrogate keys (BIGSERIAL) for simplicity.

### Foreign Keys

**Purpose**: Enforce referential integrity.

```sql
CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    block_id BIGINT NOT NULL REFERENCES blocks(id) ON DELETE CASCADE
);
```

**Benefits:**
- ✅ Cannot insert transaction for non-existent block
- ✅ `ON DELETE CASCADE`: Deleting a block auto-deletes its transactions (useful for reorgs)
- ✅ Database enforces consistency

**Trade-offs:**
- ❌ Slightly slower writes (must check constraint)
- ❌ Delete operations slower (must delete children first or cascade)

### Constraints

**Types:**

```sql
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,            -- Exactly one email per account
    balance NUMERIC CHECK (balance >= 0),  -- No negative balances
    status TEXT CHECK (status IN ('active', 'suspended', 'closed')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Best practice**: Enforce data integrity at the database level, not just in application code (code has bugs, database is reliable).

---

## Interview Questions

### Q1: Explain ACID properties and why they matter for blockchain indexing.

**Answer:**

ACID ensures reliable transaction processing:

- **Atomicity**: During a reorg, we must rollback all affected blocks, transactions, and events atomically. If one DELETE fails, the entire reorg rollback is aborted, preventing partial/corrupted state.

- **Consistency**: Foreign keys ensure we never have orphaned transactions (transaction without its parent block). Constraints prevent invalid data (e.g., negative block numbers).

- **Isolation**: Multiple API queries can run concurrently without seeing each other's partial results. An ongoing reorg rollback is invisible to read queries until committed.

- **Durability**: Once a block is committed to the database, it survives server crashes. PostgreSQL's Write-Ahead Log (WAL) ensures we can replay committed transactions after a crash.

For blockchain indexing, **atomicity** is critical because reorgs require deleting data from multiple tables (blocks, transactions, events) in an all-or-nothing operation. Without ACID, we'd risk serving inconsistent data.

### Q2: When should you denormalize a database?

**Answer:**

Denormalize when:

1. **Read-heavy workload**: 100:1 or higher read/write ratio
2. **JOIN performance is critical**: Queries have strict SLA (e.g., < 100ms)
3. **Data is immutable**: Blockchain data rarely changes (except reorgs)
4. **Storage is cheap**: Acceptable to duplicate data

**Example**: In our indexer, we store `block_timestamp` in the transactions table (denormalized) to avoid JOINs when querying transaction history:

```sql
-- Normalized (requires JOIN)
SELECT tx_hash, b.timestamp 
FROM transactions t 
JOIN blocks b ON t.block_number = b.block_number
WHERE t.from_address = '0x123';

-- Denormalized (no JOIN, faster)
SELECT tx_hash, block_timestamp 
FROM transactions 
WHERE from_address = '0x123';
```

**Trade-off**: Uses more storage, but blockchain data is immutable so no consistency risk.

### Q3: Explain the CAP theorem and how it applies to database selection.

**Answer:**

CAP theorem states you can only guarantee 2 of 3 properties in a distributed database:
- **Consistency**: All nodes see same data
- **Availability**: System responds even if nodes are down
- **Partition tolerance**: System works despite network failures

In practice, partition tolerance is mandatory (networks fail), so the choice is:

**CP (Consistency + Partition Tolerance)**: PostgreSQL with synchronous replication
- Sacrifices availability during network issues
- Best for: Financial systems, blockchain indexers (accurate data is critical)

**AP (Availability + Partition Tolerance)**: Cassandra, DynamoDB
- Sacrifices consistency (eventual consistency)
- Best for: Social media, analytics (stale data is acceptable)

For blockchain indexing, we chose **PostgreSQL (CP)** because serving stale block data would cause users to see wrong balances. We accept that during a database outage, API returns errors rather than incorrect data.

### Q4: How do indexes improve performance, and what are the trade-offs?

**Answer:**

Indexes create auxiliary data structures (usually B-trees) that allow the database to find rows without scanning the entire table.

**Example without index:**
```sql
SELECT * FROM transactions WHERE from_address = '0x123';
-- Scans all 10M rows → 5 seconds
```

**Example with index:**
```sql
CREATE INDEX idx_from_addr ON transactions (from_address);
SELECT * FROM transactions WHERE from_address = '0x123';
-- Index seek finds 100 matching rows → 5 milliseconds (1000x faster)
```

**Trade-offs:**

✅ **Pros:**
- Dramatically faster reads (up to 1000x)
- Enables efficient sorting and range queries

❌ **Cons:**
- Slower writes (every INSERT/UPDATE must update indexes)
- Storage overhead (30-50% per index)
- Maintenance overhead (VACUUM, REINDEX on high-write tables)

**When to index:**
- Primary keys (automatic)
- Foreign keys (for JOINs)
- Columns in WHERE, ORDER BY, JOIN conditions
- High cardinality columns (many unique values)

**When NOT to index:**
- Small tables (< 10K rows, full scan is faster)
- Low cardinality columns (e.g., boolean flags)
- Frequently updated columns (write overhead > read benefit)

### Q5: Explain transaction isolation levels and when to use each.

**Answer:**

**Read Committed (Default)**:
- Prevents dirty reads (reading uncommitted data)
- Allows non-repeatable reads (same query returns different results)
- **Use case**: Most applications (good balance of consistency and performance)

**Repeatable Read**:
- Prevents dirty reads and non-repeatable reads
- PostgreSQL's MVCC also prevents phantom reads
- **Use case**: Reporting queries that require consistency throughout the transaction

**Serializable**:
- Prevents all anomalies (equivalent to serial execution)
- May abort transactions with "serialization failure"
- **Use case**: Financial transactions where absolute consistency is critical (e.g., bank transfers)

**Example: Blockchain reorg handling**
```go
// Use Repeatable Read to ensure consistent snapshot during reorg
tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
tx.Exec("DELETE FROM blocks WHERE block_number > ?", reorgBlock)
tx.Exec("DELETE FROM transactions WHERE block_number > ?", reorgBlock)
tx.Commit()
```

This ensures that if new blocks are inserted during the reorg rollback, they don't interfere with our consistent view.

### Q6: How would you design a schema for 1 billion transactions?

**Answer:**

1. **Partitioning**: Partition by time and chain_id
   ```sql
   CREATE TABLE transactions (...) PARTITION BY RANGE (created_at);
   CREATE TABLE tx_2024_01 PARTITION OF transactions
   FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
   ```

2. **Archival strategy**:
   - Hot data (< 3 months): SSD, full indexes
   - Warm data (3-12 months): HDD, partial indexes
   - Cold data (> 12 months): S3 + Athena for queries

3. **Vertical partitioning**: Split hot columns (hash, block_number) from cold columns (input_data, raw_receipt)

4. **Compression**: Use TimescaleDB's compression (10x space savings for old data)

5. **Sharding**: If single database can't handle load, shard by chain_id
   ```
   DB1: Ethereum (chain_id = 1)
   DB2: Polygon (chain_id = 137)
   DB3: Arbitrum (chain_id = 42161)
   ```

6. **Read replicas**: 5-10 read replicas for query scaling

**Expected performance**:
- 10K writes/sec (with batching)
- 100K reads/sec (across replicas)
- Sub-second queries (with proper indexes and partition pruning)

### Q7: What's the difference between horizontal and vertical scaling?

**Answer:**

**Vertical Scaling** (Scale Up):
- Add more resources to single machine (CPU, RAM, disk)
- **Pros**: Simple, no code changes
- **Cons**: Hardware limits (max 1TB RAM), single point of failure, expensive
- **Example**: Upgrade database from 16GB RAM to 128GB RAM

**Horizontal Scaling** (Scale Out):
- Add more machines
- **Pros**: No hardware limits, better fault tolerance, cost-effective
- **Cons**: Complex (sharding, load balancing, data consistency)
- **Example**: Run 5 database replicas instead of 1

**For blockchain indexing:**
- **Ingester**: Horizontal (deploy 1 instance per chain)
- **API**: Horizontal (load balancer → N API servers)
- **Database**: Vertical first (up to 1B rows), then horizontal (sharding by chain_id)

**Best practice**: Scale vertically until you can't, then scale horizontally.

---

## Related Documentation

- [PostgreSQL-Database.md](./PostgreSQL-Database.md) - PostgreSQL-specific implementation
- [System-Design-Architecture.md](./System-Design-Architecture.md) - Architecture patterns
- [Deployment-Production.md](./Deployment-Production.md) - Production database hosting

---

**Last Updated**: 2025-11-27
