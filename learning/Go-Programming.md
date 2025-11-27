# Go Programming

> **Purpose**: Comprehensive Go programming guide covering modules, concurrency, patterns, and best practices for blockchain indexing systems with integrated interview Q&A.

---

## Table of Contents
- [Go Modules & Dependency Management](#go-modules--dependency-management)
- [Goroutines & Concurrency](#goroutines--concurrency)
- [Context Propagation](#context-propagation)
- [Channels](#channels)
- [Synchronization Primitives](#synchronization-primitives)
- [Error Handling](#error-handling)
- [Interfaces & Composition](#interfaces--composition)
- [Memory Management](#memory-management)
- [Performance Optimization](#performance-optimization)
- [Testing](#testing)
- [Interview Questions](#interview-questions)

---

## Go Modules & Dependency Management

Go uses a built-in module system for dependency management, introduced in Go 1.11 and default since Go 1.16.

### Core Module Files

**go.mod (Module Definition)**

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

**Components:**
- **Module path**: Unique identifier (usually GitHub URL)
- **Go version**: Minimum Go version required
- **Direct dependencies**: Packages your code imports directly
- **Indirect dependencies**: Dependencies of your dependencies

**Advanced directives:**
```go
// Override a dependency (for local development or forks)
replace github.com/old/package => github.com/new/package v1.0.0
replace github.com/local/package => ../local/package

// Prevent specific versions from being used
exclude github.com/problematic/package v1.2.3

// Retract published versions
retract v1.5.0  // Critical bug, use v1.5.1 instead
```

**go.sum (Cryptographic Checksums)**

Verifies dependency integrity with SHA-256 hashes:
```
github.com/gin-gonic/gin v1.11.0 h1:abc123...
github.com/gin-gonic/gin v1.11.0/go.mod h1:def456...
```

- First line: Hash of module's source code
- Second line: Hash of module's go.mod file
- **Purpose**: Detect tampering, ensure reproducible builds
- **Always commit both go.mod and go.sum**

### Essential Go Module Commands

```bash
# Initialize a new module
go mod init github.com/user/repo

# Add missing dependencies and remove unused ones
go mod tidy

# Download dependencies to local cache
go mod download

# Verify checksums match go.sum
go mod verify

# Update specific dependency to latest
go get -u github.com/package/name

# Update all dependencies to latest minor/patch
go get -u ./...

# Update to specific version
go get github.com/package/name@v1.2.3

# Use latest commit on main branch
go get github.com/package/name@main

# Show dependency graph
go mod graph

# Explain why a package is needed
go mod why github.com/package/name

# Show available versions
go list -m -versions github.com/gin-gonic/gin
```

### Multi-Module Workspaces

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

**Benefits:**
- Edit multiple modules at once without publishing
- Changes in `./shared` immediately visible to `./services/api`
- Simplifies local development of microservices

```bash
# Create workspace
go work init ./services/api ./services/ingester ./shared

# Add module to workspace
go work use ./services/processor
```

### Semantic Versioning

Go modules follow semantic versioning (semver):
```
v1.2.3
│ │ │
│ │ └─ Patch: Bug fixes (backward compatible)
│ └─── Minor: New features (backward compatible)
└───── Major: Breaking changes
```

**Special rules:**
- **v0.x.x**: Unstable, no compatibility guarantees
- **v1.x.x**: Stable API, must maintain compatibility
- **v2+**: Major version in import path
  ```go
  import "github.com/user/repo/v2"  // v2.x.x
  import "github.com/user/repo/v3"  // v3.x.x
  ```

### Minimal Version Selection (MVS)

Go's unique approach to dependency resolution:

```
Project depends on:
- Package A requires X@v1.2
- Package B requires X@v1.3

Go selects: X@v1.3 (highest requested)
```

**Philosophy**: Use the minimum version that satisfies all requirements. Assumes semantic versioning (v1.3 is backward compatible with v1.2).

**Diamond dependency handling:**
- If both require same major version → pick highest
- If different major versions → both included (different import paths)

---

## Goroutines & Concurrency

### What Are Goroutines?

Lightweight threads managed by Go runtime, much cheaper than OS threads.

**Key characteristics:**
- **Lightweight**: 2KB initial stack (vs 1-2MB for OS threads)
- **Dynamic stack**: Grows/shrinks as needed
- **M:N scheduling**: M goroutines multiplexed on N OS threads
- **Cooperative**: Yields at function calls, channel ops, blocking syscalls

### Goroutines in Our Project

**1. Parallel Chain Ingestion**
```go
// services/ingester/main.go
for _, chain := range chains {
    ingester.wg.Add(1)
    go ingester.ingestChain(chain)  // Spawn goroutine per chain
}
ingester.wg.Wait()  // Block until all chains stop
```

Each chain runs in its own goroutine, allowing parallel processing without blocking.

**2. Signal Handling**
```go
// services/ingester/main.go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigChan
    log.Println("Shutdown signal received...")
    ingester.shutdown()
}()
```

Signal handler runs in background goroutine, allowing main program to continue.

**3. WebSocket Block Subscription**
```go
headers := make(chan *types.Header)
sub, err := client.SubscribeNewHead(ing.ctx, headers)

go func() {
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
}()
```

### Goroutine Best Practices

**1. Always know how goroutines will terminate**
```go
// ❌ BAD: Goroutine leak
go func() {
    for {
        doWork()  // Runs forever, no exit condition
    }
}()

// ✅ GOOD: Context-based termination
go func() {
    for {
        select {
        case <-ctx.Done():
            return  // Exit when context cancelled
        default:
            doWork()
        }
    }
}()
```

**2. Handle panics in goroutines**
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Goroutine panic recovered: %v", r)
        }
    }()
    
    // Work that might panic
    riskyOperation()
}()
```

**3. Pass data explicitly, don't rely on closure variables**
```go
// ❌ BAD: Loop variable captured by closure
for _, chain := range chains {
    go processChain(chain)  // All goroutines see last chain value!
}

// ✅ GOOD: Pass as parameter
for _, chain := range chains {
    chain := chain  // Create new variable (Go 1.21 fixes this)
    go func(c Chain) {
        processChain(c)
    }(chain)
}
```

### Goroutine Patterns

**Worker Pool**
```go
func workerPool(jobs <-chan Job, results chan<- Result, numWorkers int) {
    var wg sync.WaitGroup
    
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            for job := range jobs {
                results <- processJob(job)
            }
        }(i)
    }
    
    wg.Wait()
    close(results)
}
```

**Fan-Out, Fan-In**
```go
func fanOutFanIn(inputs []Input) []Result {
    results := make(chan Result, len(inputs))
    
    // Fan-out: Spawn goroutine per input
    for _, input := range inputs {
        go func(in Input) {
            results <- process(in)
        }(input)
    }
    
    // Fan-in: Collect results
    var collected []Result
    for i := 0; i < len(inputs); i++ {
        collected = append(collected, <-results)
    }
    
    return collected
}
```

---

## Context Propagation

### What Is Context?

Carries deadlines, cancellation signals, and request-scoped values across API boundaries.

**Purpose:**
- Propagate cancellation signals
- Set timeouts on operations
- Pass request-scoped values (trace IDs, auth tokens)
- Enable graceful shutdown

### Context in Our Project

**1. Top-Level Context for Service Lifetime**
```go
// services/ingester/main.go
ctx, cancel := context.WithCancel(context.Background())
ingester := &Ingester{
    ctx:    ctx,
    cancel: cancel,
}

// On shutdown signal
ingester.cancel()  // Cancels ctx, propagates to all operations
```

**2. Timeout for Database Operations**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := db.PingContext(ctx); err != nil {
    return fmt.Errorf("database ping timeout: %w", err)
}
```

**3. RPC Calls with Timeout**
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

header, err := client.HeaderByNumber(ctx, nil)
if err != nil {
    log.Printf("RPC call failed or timed out: %v", err)
}
```

### Context Types

```go
// Root context (never cancelled)
ctx := context.Background()

// Placeholder when unsure
ctx := context.TODO()

// Manual cancellation
ctx, cancel := context.WithCancel(parent)
cancel()  // Trigger cancellation

// Time-based cancellation
ctx, cancel := context.WithTimeout(parent, 5*time.Second)

// Absolute deadline
deadline := time.Now().Add(10 * time.Second)
ctx, cancel := context.WithDeadline(parent, deadline)

// Carry request-scoped values (use sparingly!)
ctx := context.WithValue(parent, "userID", 12345)
userID := ctx.Value("userID").(int)
```

### Context Best Practices

**1. Always pass context as first parameter**
```go
// ✅ GOOD: Context first
func DoWork(ctx context.Context, data string) error

// ❌ BAD: Context elsewhere
func DoWork(data string, ctx context.Context) error
```

**2. Don't store context in structs (except services)**
```go
// ❌ BAD: Storing context
type Worker struct {
    ctx context.Context  // Don't do this
}

// ✅ GOOD: Pass context to methods
type Worker struct {
    // Other fields
}

func (w *Worker) DoWork(ctx context.Context) error {
    // Use ctx here
}

// ✅ EXCEPTION: Long-lived services
type Ingester struct {
    ctx    context.Context  // OK for service lifetime
    cancel context.CancelFunc
}
```

**3. Always defer cancel()**
```go
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()  // Ensures resources are released

// Even if operation finishes early, cancel releases resources
result, err := doWork(ctx)
return result, err
```

**4. Check context cancellation in loops**
```go
func processItems(ctx context.Context, items []Item) error {
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()  // Return context.Canceled or context.DeadlineExceeded
        default:
            if err := process(item); err != nil {
                return err
            }
        }
    }
    return nil
}
```

---

## Channels

### What Are Channels?

Typed conduits for goroutines to communicate safely without explicit locks.

**Philosophy**: "Don't communicate by sharing memory; share memory by communicating."

### Channel Basics

**Creating channels:**
```go
ch := make(chan int)        // Unbuffered (synchronous)
ch := make(chan int, 10)    // Buffered (capacity 10)
ch := make(chan struct{})   // Signal channel (no data)
```

**Sending and receiving:**
```go
ch <- 42       // Send value to channel (blocks if full)
value := <-ch  // Receive value from channel (blocks if empty)

// Check if channel is closed
value, ok := <-ch
if !ok {
    // Channel closed and empty
}
```

**Closing channels:**
```go
close(ch)  // Only sender should close

// Receiving from closed channel
value, ok := <-ch  // ok=false if closed, value=zero value

// Sending to closed channel
ch <- 42  // PANIC!
```

### Channels in Our Project

**1. Signal Handling**
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

<-sigChan  // Block until signal received
```

**2. WebSocket Subscriptions**
```go
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

### Channel Patterns

**1. Done Channel (Signal Pattern)**
```go
done := make(chan struct{})

go func() {
    <-done  // Block until signal
    cleanup()
}()

// Signal goroutine to stop
close(done)  // All receivers unblock immediately
```

**2. Pipeline Pattern**
```go
func pipeline(ctx context.Context, nums []int) <-chan int {
    out := make(chan int)
    
    go func() {
        defer close(out)
        for _, n := range nums {
            select {
            case <-ctx.Done():
                return
            case out <- n * 2:
            }
        }
    }()
    
    return out
}
```

**3. Fan-Out (Multiple Consumers)**
```go
jobs := make(chan Job, 100)

// Start 10 workers
for i := 0; i < 10; i++ {
    go func(id int) {
        for job := range jobs {
            process(job)
        }
    }(i)
}

// Produce jobs
for _, job := range allJobs {
    jobs <- job
}
close(jobs)  // Signal workers to stop after draining
```

**4. Select Statement (Multiplexing)**
```go
select {
case msg := <-ch1:
    handleMessage(msg)
case <-time.After(5 * time.Second):
    log.Println("Timeout")
case <-ctx.Done():
    return ctx.Err()
default:
    // Non-blocking: executes if no channel ready
    log.Println("No activity")
}
```

### Channel Antipatterns

**❌ BAD: Range over nil channel (deadlock)**
```go
var ch chan int  // nil channel
for v := range ch {  // Deadlock! Blocks forever
    fmt.Println(v)
}
```

**❌ BAD: Sending on closed channel (panic)**
```go
close(ch)
ch <- 42  // PANIC: send on closed channel
```

**❌ BAD: Multiple senders closing same channel**
```go
// Both goroutines might try to close ch
go func() { close(ch) }()  // Panic if both execute
go func() { close(ch) }()
```

---

## Synchronization Primitives

### sync.WaitGroup

Wait for multiple goroutines to finish.

**Usage in our project:**
```go
type Ingester struct {
    wg sync.WaitGroup
}

// Before spawning goroutines
for _, chain := range chains {
    ingester.wg.Add(1)
    go func(c Chain) {
        defer ingester.wg.Done()
        ingester.ingestChain(c)
    }(chain)
}

// Wait for all to complete
ingester.wg.Wait()
```

**Best practices:**
- Call `Add()` before spawning goroutine
- Call `Done()` with defer to ensure it's called
- Don't reuse WaitGroup after Wait() returns (create new one)

### sync.Once

Ensures function runs exactly once, even with concurrent calls.

**Usage in our project:**
```go
type Ingester struct {
    shutdownOnce sync.Once
}

func (ing *Ingester) shutdown() {
    ing.shutdownOnce.Do(func() {
        ing.cancel()
        // Close connections...
    })
}

// Multiple calls only execute logic once
ing.shutdown()
ing.shutdown()  // Does nothing
```

**Use cases:**
- Lazy initialization
- Singleton pattern
- Shutdown logic (prevent double-close)

### sync.Mutex

Mutual exclusion lock for protecting shared memory.

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

func (c *SafeCounter) Get(key string) int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.count[key]
}
```

**Why not used in our project:**
- Each goroutine has isolated state (per-chain ingester)
- Database handles concurrency with transactions
- Context cancellation handles coordination
- No shared in-memory cache (would need mutex if added)

### sync.RWMutex

Allows multiple readers OR one writer.

```go
type Cache struct {
    mu    sync.RWMutex
    items map[string]interface{}
}

func (c *Cache) Get(key string) interface{} {
    c.mu.RLock()         // Multiple readers OK
    defer c.mu.RUnlock()
    return c.items[key]
}

func (c *Cache) Set(key string, value interface{}) {
    c.mu.Lock()          // Exclusive write lock
    defer c.mu.Unlock()
    c.items[key] = value
}
```

**When to use:**
- Read-heavy workloads (100:1 read/write ratio)
- Shared cache
- Configuration that rarely changes

---

## Error Handling

### Error Wrapping

Go 1.13+ supports error wrapping for better context:

```go
// Wrap error with context
if err != nil {
    return fmt.Errorf("failed to open database: %w", err)
}

// Check for specific error type
if errors.Is(err, sql.ErrNoRows) {
    // Handle not found
}

// Extract underlying error
var pgErr *pq.Error
if errors.As(err, &pgErr) {
    log.Printf("PostgreSQL error code: %s", pgErr.Code)
}
```

### Custom Errors

```go
type ChainNotFoundError struct {
    ChainID int64
}

func (e *ChainNotFoundError) Error() string {
    return fmt.Sprintf("chain %d not found", e.ChainID)
}

// Usage
if chain == nil {
    return &ChainNotFoundError{ChainID: id}
}

// Checking
var notFound *ChainNotFoundError
if errors.As(err, &notFound) {
    log.Printf("Chain %d doesn't exist", notFound.ChainID)
}
```

### Panic and Recover

**Panic**: For unrecoverable errors (programmer mistakes)
```go
if config == nil {
    panic("config cannot be nil")  // Should never happen
}
```

**Recover**: Catch panics in goroutines
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Recovered panic: %v", r)
            debug.PrintStack()
        }
    }()
    
    riskyOperation()
}()
```

**Best practice**: Don't use panic/recover for normal error handling. Use return errors.

---

## Interfaces & Composition

### Interface Design

**Small interfaces:**
```go
// Good: Single-method interface
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Good: Composed from smaller interfaces
type ReadWriter interface {
    Reader
    Writer
}
```

### Accept Interfaces, Return Structs

```go
// ✅ GOOD: Accept interface (flexible)
func ProcessData(r io.Reader) error {
    // Can accept file, network connection, bytes.Buffer, etc.
}

// ✅ GOOD: Return concrete struct (explicit)
func NewUser(name string) *User {
    return &User{Name: name}
}

// ❌ BAD: Return interface (hard to extend)
func NewUser(name string) UserInterface {
    return &User{Name: name}
}
```

### Empty Interface

```go
var anything interface{}  // Or "any" in Go 1.18+
anything = 42
anything = "hello"
anything = []int{1, 2, 3}

// Type assertion
if str, ok := anything.(string); ok {
    fmt.Println("String:", str)
}

// Type switch
switch v := anything.(type) {
case int:
    fmt.Println("Int:", v)
case string:
    fmt.Println("String:", v)
default:
    fmt.Println("Unknown type")
}
```

---

## Memory Management

### Stack vs Heap Allocation

**Stack allocation (fast):**
```go
func foo() {
    x := 42  // Allocated on stack, freed when function returns
}
```

**Heap allocation (slower, garbage collected):**
```go
func bar() *int {
    x := 42
    return &x  // Escapes to heap (pointer returned)
}
```

**Escape analysis:**
```bash
go build -gcflags="-m" main.go
# Shows what escapes to heap
```

### Reducing Allocations

**1. Preallocate slices**
```go
// ❌ BAD: Multiple allocations as slice grows
var items []Item
for i := 0; i < 1000; i++ {
    items = append(items, Item{})
}

// ✅ GOOD: Single allocation
items := make([]Item, 0, 1000)
for i := 0; i < 1000; i++ {
    items = append(items, Item{})
}
```

**2. Reuse buffers**
```go
// ❌ BAD: New buffer each time
func format(items []Item) string {
    var buf bytes.Buffer
    // ...
    return buf.String()
}

// ✅ GOOD: Pool of buffers
var bufPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func format(items []Item) string {
    buf := bufPool.Get().(*bytes.Buffer)
    defer bufPool.Put(buf)
    buf.Reset()
    // ...
    return buf.String()
}
```

**3. Use strings.Builder for string concatenation**
```go
// ❌ BAD: Creates N temporary strings
query := "INSERT INTO ... VALUES "
for i := 0; i < 1000; i++ {
    query += "($1,$2),"  // New allocation each time
}

// ✅ GOOD: Single buffer
var query strings.Builder
query.WriteString("INSERT INTO ... VALUES ")
for i := 0; i < 1000; i++ {
    query.WriteString("($1,$2),")
}
```

---

## Performance Optimization

### Dynamic SQL Query Building

**Problem**: Inserting many rows with individual INSERTs is slow (network overhead).

**Solution**: Batch INSERT with dynamic SQL

```go
func insertBatch(db *sql.Tx, transactions []Transaction) error {
    if len(transactions) == 0 {
        return nil
    }
    
    const numFields = 9
    
    // Use strings.Builder for efficient string building
    var query strings.Builder
    query.WriteString(`
        INSERT INTO transactions (
            chain_id, block_number, tx_hash, tx_index,
            from_address, to_address, value, gas_price, gas_used
        ) VALUES 
    `)
    
    // Pre-allocate args slice
    args := make([]interface{}, 0, len(transactions)*numFields)
    
    for i, tx := range transactions {
        if i > 0 {
            query.WriteString(", ")
        }
        
        // Build placeholder: ($1,$2,$3), ($4,$5,$6), ...
        base := i * numFields
        query.WriteString(fmt.Sprintf(
            "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
            base+1, base+2, base+3, base+4, base+5,
            base+6, base+7, base+8, base+9,
        ))
        
        args = append(args,
            tx.ChainID, tx.BlockNumber, tx.Hash, tx.Index,
            tx.FromAddress, tx.ToAddress, tx.Value,
            tx.GasPrice, tx.GasUsed,
        )
    }
    
    _, err := db.Exec(query.String(), args...)
    return err
}
```

**Performance**: 188 transactions in 200ms vs 60s (300x improvement)

### Benchmarking

```go
func BenchmarkInsertBatch(b *testing.B) {
    db := setupDB(b)
    defer db.Close()
    
    txs := generateTestTransactions(100)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        insertBatch(db, txs)
    }
}

// Run: go test -bench=. -benchmem
```

### Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Live profiling with pprof HTTP endpoint
import _ "net/http/pprof"
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
# Visit http://localhost:6060/debug/pprof/
```

---

## Testing

### Table-Driven Tests

```go
func TestParseChainID(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    int64
        wantErr bool
    }{
        {"valid ethereum", "1", 1, false},
        {"valid polygon", "137", 137, false},
        {"invalid", "abc", 0, true},
        {"empty", "", 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseChainID(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseChainID() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ParseChainID() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Mocking

```go
// Interface for testing
type BlockFetcher interface {
    GetBlock(ctx context.Context, number int64) (*Block, error)
}

// Mock implementation
type MockBlockFetcher struct {
    blocks map[int64]*Block
    err    error
}

func (m *MockBlockFetcher) GetBlock(ctx context.Context, number int64) (*Block, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.blocks[number], nil
}

// Test using mock
func TestProcessBlock(t *testing.T) {
    mock := &MockBlockFetcher{
        blocks: map[int64]*Block{
            1000: {Number: 1000, Hash: "0xabc"},
        },
    }
    
    processor := NewProcessor(mock)
    err := processor.Process(context.Background(), 1000)
    
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
}
```

---

## Interview Questions

### Q1: Explain goroutines vs OS threads. When would you use millions of goroutines?

**Answer:**

**Goroutines**:
- 2KB initial stack (dynamic, grows/shrinks)
- M:N scheduling (M goroutines on N OS threads)
- Managed by Go runtime
- Cheap to create (can spawn millions)

**OS Threads**:
- 1-2MB fixed stack
- 1:1 with kernel threads
- Managed by OS scheduler
- Expensive (typically max ~thousands)

**When to use millions**: Network services handling concurrent connections (e.g., WebSocket server with 1M clients). Each connection gets its own goroutine that blocks on I/O.

**Example from our project**: We spawn one goroutine per blockchain (5-10 total), not millions, because each goroutine does continuous work (polling RPC).

### Q2: What are the three ways to synchronize goroutines? When would you use each?

**Answer:**

**1. Channels** (communication):
- **Use when**: Passing data between goroutines
- **Example**: Producer-consumer, pipeline patterns
```go
jobs := make(chan Job)
go producer(jobs)
go consumer(jobs)
```

**2. WaitGroup** (coordination):
- **Use when**: Waiting for multiple goroutines to complete
- **Example**: Parallel processing with no data passing
```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(i Item) {
        defer wg.Done()
        process(i)
    }(item)
}
wg.Wait()
```

**3. Mutex** (shared memory protection):
- **Use when**: Multiple goroutines access same data structure
- **Example**: Shared cache, counter
```go
var mu sync.Mutex
var counter int

mu.Lock()
counter++
mu.Unlock()
```

**Rule of thumb**: Prefer channels for communication, mutexes for protecting shared state.

### Q3: Explain context propagation. How does it enable graceful shutdown?

**Answer:**

Context carries cancellation signals, timeouts, and request-scoped values through call chains.

**Graceful shutdown example**:
```go
// Main service context
ctx, cancel := context.WithCancel(context.Background())

// Start workers
go worker1(ctx)
go worker2(ctx)

// On shutdown signal
cancel()  // Cancels ctx

// Workers check context
func worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            cleanup()
            return  // Exit gracefully
        default:
            doWork()
        }
    }
}
```

**Benefits**:
- All workers stop together (no orphaned goroutines)
- Can add timeout: `context.WithTimeout(ctx, 30*time.Second)`
- Works across function boundaries (propagates through call stack)

**In our project**: When SIGTERM received, `ingester.cancel()` stops all chain ingesters, RPC calls respect context timeout.

### Q4: What's the difference between buffered and unbuffered channels? When would you use each?

**Answer:**

**Unbuffered** (`make(chan T)`):
- Synchronous: Sender blocks until receiver ready
- **Use when**: Synchronization required (handshake)
```go
done := make(chan struct{})
go func() {
    work()
    done <- struct{}{}  // Blocks until main receives
}()
<-done  // Guaranteed work() finished
```

**Buffered** (`make(chan T, N)`):
- Asynchronous: Sender blocks only when buffer full
- **Use when**: Decoupling producer/consumer rates
```go
jobs := make(chan Job, 100)
go fastProducer(jobs)    // Can produce 100 before blocking
go slowConsumer(jobs)    // Consumes at its own pace
```

**Real-world**: Signal channels (`os.Signal`) are buffered (capacity 1) so signal isn't lost if handler is busy.

### Q5: Explain Go's error handling philosophy. Why no exceptions?

**Answer:**

**Philosophy**: Errors are values, not exceptional control flow.

```go
// Explicit error handling
result, err := doWork()
if err != nil {
    return fmt.Errorf("work failed: %w", err)  // Wrap with context
}
```

**Why no exceptions?**
- **Explicit**: Forces handling errors where they occur
- **Readable**: Error path is visible in code (no hidden control flow)
- **Performance**: No stack unwinding overhead
- **Composable**: Errors are values, can be inspected, wrapped, logged

**Exception problems**:
- Hidden control flow (function can exit anywhere)
- Encourages ignoring errors (catch-all handlers)
- Stack unwinding is expensive

**Error wrapping** (Go 1.13+):
```go
if err != nil {
    return fmt.Errorf("failed to fetch block %d: %w", blockNum, err)
}

// Check for specific error
if errors.Is(err, sql.ErrNoRows) {
    // Handle not found
}

// Extract error type
var pgErr *pq.Error
if errors.As(err, &pgErr) {
    log.Printf("DB error code: %s", pgErr.Code)
}
```

### Q6: How does Go's garbage collector work? How can you reduce GC pressure?

**Answer:**

**Go GC**: Concurrent mark-and-sweep, runs in parallel with program.

**GC Triggers**: When heap size doubles since last GC (tunable with `GOGC` environment variable).

**Reduce GC pressure**:

**1. Preallocate slices**:
```go
// ❌ BAD: Multiple allocations
items := []Item{}
for i := 0; i < 1000; i++ {
    items = append(items, Item{})  // Reallocates when capacity exceeded
}

// ✅ GOOD: Single allocation
items := make([]Item, 0, 1000)
```

**2. Reuse objects with sync.Pool**:
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func process() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()
    // Use buf
}
```

**3. Avoid pointer-heavy structs**:
```go
// ❌ BAD: GC must scan all pointers
type Node struct {
    Value *int
    Next  *Node
}

// ✅ GOOD: No pointers to scan
type Node struct {
    Value int
    Next  int  // Index into slice
}
```

**4. Use strings.Builder instead of string concatenation**:
```go
// ❌ BAD: Creates many temporary strings
s := "a" + "b" + "c"  // 2 allocations

// ✅ GOOD: One buffer
var sb strings.Builder
sb.WriteString("a")
sb.WriteString("b")
sb.WriteString("c")
```

### Q7: What's the purpose of go.mod and go.sum? Why do we need both?

**Answer:**

**go.mod**: Declares dependencies and versions
- Human-readable
- Defines what you want (version constraints)
- Contains direct and indirect dependencies

**go.sum**: Cryptographic checksums of dependencies
- SHA-256 hashes of module content
- Proves what you got (verifies integrity)
- Detects tampering, ensures reproducible builds

**Why both?**
- **go.mod** alone: Could download different code with same version (supply chain attack)
- **go.sum** ensures you get exact same code as your teammates
- Together: Reproducible, secure builds

**Example**:
```go
// go.mod
require github.com/gin-gonic/gin v1.11.0

// go.sum
github.com/gin-gonic/gin v1.11.0 h1:abc123...  // Module zip hash
github.com/gin-gonic/gin v1.11.0/go.mod h1:def456...  // go.mod hash
```

**Verification**: `go mod verify` checks downloaded modules match go.sum

### Q8: Explain Go's "accept interfaces, return structs" principle.

**Answer:**

**Principle**: Functions accept interfaces (flexible), return concrete types (explicit).

```go
// ✅ GOOD: Accept interface
func ProcessData(r io.Reader) error {
    // Caller can pass *os.File, *bytes.Buffer, net.Conn, etc.
}

// ✅ GOOD: Return concrete type
func NewUser(name string) *User {
    return &User{Name: name}
}

// ❌ BAD: Return interface
func NewUser(name string) UserInterface {
    return &User{Name: name}
}
```

**Why?**

**Accept interfaces**:
- Flexible: Caller provides any implementation
- Testable: Can mock with test implementation
- Composable: Works with wrapper types

**Return structs**:
- Explicit: Caller knows exact type
- Extensible: Can add methods to struct without breaking callers
- Efficient: No dynamic dispatch

**Example from standard library**:
```go
// Accepts interface (flexible)
func Copy(dst io.Writer, src io.Reader) (written int64, err error)

// Returns struct (explicit)
func Open(name string) (*File, error)
```

### Q9: How would you debug a goroutine leak?

**Answer:**

**1. Detect leak with runtime**:
```go
import "runtime"

before := runtime.NumGoroutine()
doWork()
after := runtime.NumGoroutine()
if after > before {
    log.Printf("Goroutine leak! Before: %d, After: %d", before, after)
}
```

**2. Get goroutine stack traces**:
```go
// Send SIGQUIT to running process
kill -QUIT <pid>

// Programmatically
buf := make([]byte, 1<<20)
stacklen := runtime.Stack(buf, true)
fmt.Printf("%s", buf[:stacklen])
```

**3. Use pprof**:
```go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Visit http://localhost:6060/debug/pprof/goroutine
```

**Common causes**:
- Goroutines waiting on channel that's never closed
- Missing context cancellation
- Infinite loop without exit condition

**Prevention**:
```go
// ✅ GOOD: Context-based termination
func worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return  // Exit when cancelled
        default:
            doWork()
        }
    }
}

// ❌ BAD: No exit condition
func worker() {
    for {
        doWork()  // Runs forever
    }
}
```

### Q10: Explain Minimal Version Selection (MVS) in Go modules.

**Answer:**

MVS is Go's dependency resolution algorithm that picks the **minimum version** that satisfies all requirements.

**Example**:
```
Your project depends on:
- Package A requires X@v1.2.0
- Package B requires X@v1.3.0

Go selects: X@v1.3.0 (highest required minimum)
```

**Why MVS?**
- **Predictable**: Always picks lowest compatible version
- **Safe**: Assumes semantic versioning (v1.3.0 compatible with v1.2.0)
- **No SAT solver**: Simple, deterministic algorithm

**Contrast with npm/cargo**:
- They pick **latest** version within constraints
- Can silently upgrade to untested versions
- Requires complex SAT solving

**Diamond dependency**:
```
Project
├── A v1.0 → X v1.2
└── B v2.0 → X v1.3

Go picks X v1.3 (satisfies both A and B)
```

**Major version conflict**:
```
Project
├── A v1.0 → X v1.5
└── B v2.0 → X v2.0

Go includes both:
- import "github.com/user/x"     // v1.5
- import "github.com/user/x/v2"  // v2.0
```

---

## Related Documentation

- [Database-Fundamentals.md](./Database-Fundamentals.md) - Database theory
- [PostgreSQL-Database.md](./PostgreSQL-Database.md) - PostgreSQL specifics
- [System-Design-Architecture.md](./System-Design-Architecture.md) - Architecture patterns
- [Setup-Troubleshooting.md](./Setup-Troubleshooting.md) - Setup and debugging

---

**Last Updated**: 2025-11-27
