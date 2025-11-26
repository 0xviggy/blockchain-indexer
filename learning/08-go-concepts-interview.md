# Go Concepts - Deep Dive for Interviews

Comprehensive guide to Go programming concepts used in this project, with interview-focused explanations and examples.

## Table of Contents
- [Goroutines](#goroutines)
- [Context Propagation](#context-propagation)
- [Channels](#channels)
- [sync Package](#sync-package)
- [HTTP Serialization](#http-serialization)
- [io Package](#io-package)
- [Generics](#generics)
- [Testing](#testing)
- [Google Go Style Guide](#google-go-style-guide)
- [Summary Table](#summary-table)

---

## Goroutines ⭐ (USED)

### What Are Goroutines?

Lightweight threads managed by Go runtime, much cheaper than OS threads.

### Where Used in Project

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

### Interview Points

- **Lightweight**: Can spawn thousands (we spawn one per chain + signal handler)
- **Stack size**: Starts with 2KB, grows/shrinks dynamically (OS threads = 1-2MB fixed)
- **Scheduling**: M:N scheduling (M goroutines on N OS threads)
- **Use case**: Parallel chain ingestion - each chain processes independently
- **Communication**: Use channels for synchronization

### Common Interview Questions

**Q: Goroutine vs Thread?**
A: Goroutines are lighter (2KB vs 1MB), cooperatively scheduled by Go runtime, not OS

**Q: How many goroutines can you spawn?**
A: Hundreds of thousands (limited by memory, not OS thread limits)

**Q: When would you NOT use goroutines?**
A: CPU-bound work with limited cores (no benefit from parallelism beyond core count)

---

## Context Propagation ⭐ (USED EXTENSIVELY)

### What Is Context?

Carries deadlines, cancellation signals, and request-scoped values across API boundaries.

### Where Used in Project

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

### Interview Points

- **Cancellation propagation**: Parent context cancels → all child contexts cancel
- **Timeout handling**: `WithTimeout` prevents hanging RPC calls
- **Graceful shutdown**: `ingester.ctx.Done()` signals all goroutines to stop
- **Best practice**: Always pass context as first parameter: `func DoWork(ctx context.Context, ...)`
- **Never store context**: Contexts should flow through call stack, not stored in structs (except for long-lived services)

### Context Types

```go
context.Background()              // Root context, never cancelled
context.TODO()                    // Placeholder when unsure
context.WithCancel(parent)        // Manual cancellation
context.WithTimeout(parent, 5*s)  // Time-based cancellation
context.WithDeadline(parent, t)   // Absolute deadline
context.WithValue(parent, key, v) // Carry request-scoped values (use sparingly!)
```

### Common Interview Questions

**Q: Why pass context?**
A: Cancellation, timeouts, request tracing, distributed context propagation

**Q: When to use WithTimeout vs WithCancel?**
A: WithTimeout for I/O operations, WithCancel for manual control

**Q: Is context safe for concurrent use?**
A: Yes, all methods are thread-safe

### Real-world Example from Project

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

## Channels ⭐ (USED)

### What Are Channels?

Typed conduits for goroutines to communicate safely without explicit locks.

### Where Used in Project

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

### Interview Points

- **Unbuffered vs Buffered**: 
  - `make(chan T)` → Synchronous, sender blocks until receiver ready
  - `make(chan T, N)` → Asynchronous, sender blocks only when buffer full
- **Directional channels**: `chan<- T` (send-only), `<-chan T` (receive-only)
- **Close semantics**: Only sender closes, receiver detects with `v, ok := <-ch`
- **select statement**: Multiplex on multiple channels (like epoll/kqueue)

### Channel Patterns

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

### Common Interview Questions

**Q: Buffered vs unbuffered?**
A: Buffered = async (N capacity), unbuffered = sync handoff

**Q: When to close channel?**
A: Only sender closes, when no more values will be sent

**Q: What happens if you send to closed channel?**
A: Panic!

**Q: What happens if you receive from closed channel?**
A: Immediate return with zero value, ok=false

---

## sync Package ⭐ (USED)

### What Is sync Package?

Low-level synchronization primitives for sharing memory between goroutines.

### Where Used in Project

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

### Interview Points

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

### Common Interview Questions

**Q: Mutex vs RWMutex?**
A: RWMutex allows multiple readers, one writer (better for read-heavy workloads)

**Q: Channel vs Mutex?**
A: Channel = communication, Mutex = shared memory protection

**Q: WaitGroup vs sync.Cond?**
A: WaitGroup = wait for N tasks, Cond = complex signaling with broadcast/signal

**Why We Don't Need Mutex in This Project**:
- Our design uses goroutines with isolated data (each chain has own state)
- Context cancellation handles coordination, not shared memory
- Database handles concurrency with transactions
- If we had shared cache, we'd need RWMutex

---

## HTTP Serialization ⭐ (USED - API Service)

### What Is It?

Converting Go structs to/from JSON for HTTP APIs.

### Where Used in Project

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

### Struct Tags

```go
type User struct {
    ID        int    `json:"id" db:"user_id"`           // JSON and SQL names differ
    Email     string `json:"email" validate:"required"`  // Multiple tags
    Password  string `json:"-"`                          // Omit from JSON
    IsAdmin   bool   `json:"is_admin,omitempty"`        // Omit if zero value
}
```

### Serialization Methods

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

### Common Interview Questions

**Q: Marshal vs Encoder?**
A: Marshal returns bytes, Encoder writes to stream (better for HTTP response)

**Q: What is omitempty?**
A: Omits field if zero value (0, false, nil, empty string)

**Q: How to handle time.Time?**
A: Marshals to RFC3339 by default, can customize with MarshalJSON method

**Q: Case sensitivity?**
A: JSON keys are case-sensitive, Go uses exact match with case-insensitive fallback

---

## io Package ⚠️ (LIMITED USE)

### What Is It?

Interfaces for I/O operations, enabling composition and streaming.

### Core Interfaces

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

### Common Patterns

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

### Why We Don't Use io Package Heavily

- Database library (`database/sql`) abstracts streaming
- Gin framework handles HTTP request/response bodies
- Blockchain data comes through `go-ethereum` library
- If we added CSV export or file processing, we'd use io.Reader/Writer

### Common Interview Questions

**Q: Why use io.Reader instead of []byte?**
A: Streaming (low memory), composition, testability

**Q: What is io.EOF?**
A: Sentinel error indicating end of stream

**Q: How to chain readers?**
A: Wrap them: `gzip.NewReader(file)` → `io.LimitReader(gzipReader, size)`

---

## Generics ❌ (NOT USED)

### What Are Generics?

Parametric polymorphism added in Go 1.18, allows type-safe generic functions/structs.

### Why Not Used

- Project predates heavy generic adoption patterns
- `interface{}` or code generation sufficient for our use cases
- Database libraries use reflection, not generics
- Blockchain types (`*big.Int`, `common.Address`) are concrete

### Example Use Cases (If We Refactored)

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

### Common Interview Questions

**Q: When to use generics vs interfaces?**
A: Generics = compile-time type safety, interfaces = runtime polymorphism

**Q: Performance difference?**
A: Generics can be faster (no heap allocation for small types)

**Q: What is `any` constraint?**
A: Alias for `interface{}`, means "any type"

**Q: What is `comparable` constraint?**
A: Types that support == and != (used for map keys)

---

## Testing ❌ (NOT IMPLEMENTED)

### What Is testify?

Popular testing library with assertions, mocking, and test suites.

### Why Not Implemented

MVP phase prioritized feature delivery over test coverage.

### Would Look Like (If We Added Tests)

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
```

### Common Interview Questions

**Q: assert vs require?**
A: require stops test immediately, assert continues

**Q: Table-driven tests?**
A: Loop over test cases slice (Go idiom)

**Q: Test coverage tools?**
A: `go test -cover`, `go tool cover -html=coverage.out`

---

## Google Go Style Guide ⭐ (MOSTLY FOLLOWED)

### Key Principles Followed

**✅ 1. Error Wrapping with fmt.Errorf:**
```go
// services/ingester/main.go:184
if err != nil {
    return nil, fmt.Errorf("failed to open database: %w", err)
}
```
- Use `%w` verb to wrap errors (enables `errors.Is`, `errors.As`)
- Provides context at each layer

**✅ 2. Early Returns (Guard Clauses):**
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

**✅ 3. defer for Cleanup:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()  // Ensures cancel is called
```

**✅ 4. Package Naming:**
- `package main` for executables
- Short, lowercase, no underscores: `models`, `config`

**❌ 5. Lacks Commentary:**
- Should add godoc comments for exported functions
- Example: `// Ingester manages blockchain data collection from multiple chains.`

**✅ 6. Struct Initialization:**
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

**✅ 7. Error Messages:**
- Lowercase, no punctuation: `"failed to open database"`
- Context-rich: Include chain name, block number

### Google Go Style Interview Points

- **Simplicity over cleverness**: Readable code > clever tricks
- **Composition over inheritance**: Use interfaces, not class hierarchies
- **"Accept interfaces, return structs"**: Functions take `io.Reader`, return `*User`
- **Package layout**: `cmd/`, `internal/`, `pkg/` for structure
- **Avoid `init()`**: Explicit initialization preferred

### Common Interview Questions

**Q: Go idioms vs other languages?**
A: Errors are values, composition over inheritance, interfaces, channels

**Q: What is 'accept interfaces, return structs'?**
A: Callers can pass anything implementing interface, you return concrete type

**Q: Why no exceptions?**
A: Explicit error handling, errors are values, forces dealing with failures

---

## Summary Table

| Concept | Usage in Project | Complexity | Interview Importance |
|---------|------------------|------------|---------------------|
| **Goroutines** | ✅ Used (per-chain concurrency) | Low | ⭐⭐⭐ Critical |
| **Context** | ✅ Used extensively | Medium | ⭐⭐⭐ Critical |
| **Channels** | ✅ Used (signals, subscriptions) | Medium | ⭐⭐⭐ Critical |
| **sync.WaitGroup** | ✅ Used (wait for chains) | Low | ⭐⭐ Important |
| **sync.Once** | ✅ Used (shutdown) | Low | ⭐⭐ Important |
| **sync.Mutex** | ❌ Not used | Low | ⭐⭐ Important |
| **HTTP/JSON** | ✅ Used (API service) | Low | ⭐⭐ Important |
| **io.Reader/Writer** | ⚠️ Limited use | Medium | ⭐⭐ Important |
| **Generics** | ❌ Not used | Medium | ⭐ Nice to know |
| **Testing** | ❌ Not implemented | Low | ⭐⭐ Important |
| **Style Guide** | ✅ Mostly followed | Low | ⭐⭐ Important |

---

## Related Documentation

- [Interview Preparation](./07-interview-prep.md)
- [Implementation Concepts](./06-implementation-concepts.md)
- [Technology Stack](./01-technology-stack.md)
- [Setup Guide](./05-setup-quickstart.md)
