# Go Programming & Modules Guide

> **Purpose**: Go modules, dependency management, and idiomatic Go patterns for blockchain development.

---

## Go Modules & Dependency Management

Go uses a built-in module system for dependency management, introduced in Go 1.11 and default since Go 1.16.

### Core Module Files

**1. `go.mod` (Module Definition)**

The main module file that defines your project:
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

**Components**:
- **Module path**: Unique identifier (usually GitHub URL)
- **Go version**: Minimum Go version required
- **Direct dependencies**: Packages your code imports directly
- **Indirect dependencies**: Dependencies of your dependencies (marked with `// indirect`)

**Advanced directives**:
```go
// Override a dependency (for local development or forks)
replace github.com/old/package => github.com/new/package v1.0.0
replace github.com/local/package => ../local/package

// Prevent specific versions from being used
exclude github.com/problematic/package v1.2.3

// Retract published versions (maintainer says "don't use this")
retract v1.5.0  // Critical bug, use v1.5.1 instead
```

**2. `go.sum` (Cryptographic Checksums)**

Verifies dependency integrity with SHA-256 hashes:
```
github.com/gin-gonic/gin v1.11.0 h1:abc123...
github.com/gin-gonic/gin v1.11.0/go.mod h1:def456...
```

- **First line**: Hash of the module's source code (`.zip` file)
- **Second line**: Hash of the module's `go.mod` file only
- **Purpose**: Detect tampering, ensure reproducible builds across teams
- **Version control**: Always commit both `go.mod` and `go.sum`

**3. Module Cache (`$GOPATH/pkg/mod/`)**

Global cache for downloaded modules:
```
$GOPATH/pkg/mod/
  github.com/
    gin-gonic/
      gin@v1.11.0/
        ...
```

- **Shared across projects**: Download once, use everywhere
- **Read-only**: Prevents accidental modifications
- **Version-specific**: Multiple versions can coexist

---

## Essential Go Module Commands

```bash
# Initialize a new module
go mod init github.com/user/repo

# Add missing dependencies and remove unused ones
go mod tidy

# Download dependencies to local cache
go mod download

# Verify checksums match go.sum (detect tampering)
go mod verify

# Copy dependencies to vendor/ directory
go mod vendor

# Update specific dependency to latest
go get -u github.com/package/name

# Update all dependencies to latest minor/patch
go get -u ./...

# Update all dependencies to latest major version
go get -u=patch ./...  # Patch only (safest)
go get -u ./...         # Minor/patch (safe)
go get -u -t ./...      # Include test dependencies

# Show dependency graph
go mod graph

# Explain why a package is needed
go mod why github.com/package/name

# Show available versions of a module
go list -m -versions github.com/gin-gonic/gin

# Downgrade to specific version
go get github.com/package/name@v1.2.3

# Use latest commit on main branch
go get github.com/package/name@main
```

---

## Multi-Module Workspaces

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

**Benefits**:
- Edit multiple modules at once without publishing
- Changes in `./shared` immediately visible to `./services/api`
- Simplifies local development of microservices

**Create workspace**:
```bash
go work init ./services/api ./services/ingester ./shared
```

---

## Project Structure with Modules

```
blockchain-indexer/
├── services/
│   ├── api/
│   │   ├── go.mod          # Separate module (COMMIT ✅)
│   │   ├── go.sum          # Checksums (COMMIT ✅)
│   │   ├── main.go         # Source code (COMMIT ✅)
│   │   └── api             # Binary (IGNORE ❌)
│   │
│   └── ingester/
│       ├── go.mod          # Separate module (COMMIT ✅)
│       ├── go.sum          # Checksums (COMMIT ✅)
│       └── main.go         # Source code (COMMIT ✅)
│
├── shared/
│   ├── go.mod              # Shared utilities module (COMMIT ✅)
│   ├── go.sum              # Checksums (COMMIT ✅)
│   └── config/
│       └── config.go       # Shared code (COMMIT ✅)
│
└── go.work                 # Workspace (IGNORE ❌)
```

---

## Semantic Versioning in Go

Go modules follow semantic versioning (semver):
```
v1.2.3
│ │ │
│ │ └─ Patch: Bug fixes (backward compatible)
│ └─── Minor: New features (backward compatible)
└───── Major: Breaking changes
```

**Special rules**:
- **v0.x.x**: Unstable, no compatibility guarantees
- **v1.x.x**: Stable API, must maintain compatibility
- **v2+**: Major version in import path
  ```go
  import "github.com/user/repo/v2"  // v2.x.x
  import "github.com/user/repo/v3"  // v3.x.x
  ```

**Pseudo-versions** (commits without tags):
```
v0.0.0-20191109021931-daa7c04131f5
       └─ timestamp ─┘└─ commit hash ─┘
```

---

## Common Dependency Issues & Solutions

**Issue 1: "go.sum has unexpected content"**
```bash
# Solution: Re-download and verify
go clean -modcache
go mod download
go mod verify
```

**Issue 2: Dependency not found (private repository)**
```bash
# Configure Git authentication
git config --global url."git@github.com:".insteadOf "https://github.com/"

# Or use GOPRIVATE environment variable
export GOPRIVATE=github.com/yourcompany/*
```

**Issue 3: Outdated dependencies**
```bash
# Check for updates
go list -u -m all

# Update safely (minor/patch only)
go get -u=patch ./...

# Review and test before updating to latest
go get -u ./...
go test ./...
```

---

## .gitignore Best Practices

```gitignore
# Compiled binaries (don't commit)
services/*/api
services/*/api-*
services/*/ingester
services/*/ingester-*
*.exe
*.test

# Vendor directory (only if using vendoring)
vendor/

# Workspace file (local development only)
go.work
go.work.sum

# Keep module files (CRITICAL - always commit)
# go.mod    ← DO NOT IGNORE
# go.sum    ← DO NOT IGNORE
```

---

## Go Module Interview Questions

**Q: Why do we need both `go.mod` and `go.sum`?**

**A:** `go.mod` declares what you want (version constraints), `go.sum` proves what you got (cryptographic verification). Together they ensure reproducible builds across machines and detect supply chain attacks.

**Q: When should you run `go mod tidy`?**

**A:** 
- After adding/removing imports in code
- Before committing (cleans unused dependencies)
- When `go.mod` and actual imports are out of sync
- After merging branches (resolve dependency conflicts)

**Q: Direct vs indirect dependencies?**

**A:** 
- **Direct**: You `import` them in your code
- **Indirect**: Your dependencies need them (transitive)
- Go 1.17+ separates them in `go.mod` for clarity and faster builds

**Q: What's the difference between `go get` and `go install`?**

**A:**
- `go get`: Update dependencies in `go.mod` (deprecated for installing tools in Go 1.17+)
- `go install`: Install executables to `$GOPATH/bin` without modifying `go.mod`
- **Example**: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

**Q: How do you handle diamond dependency problems?**

**A:** Go uses Minimal Version Selection (MVS):
- If A requires X@v1.2 and B requires X@v1.3, Go picks v1.3 (highest requested)
- Assumes semver: v1.3 is compatible with v1.2
- If major versions differ (v1 vs v2), both are kept (different import paths)

**Q: What's a module proxy and why use it?**

**A:** Default: `proxy.golang.org` (Google's public proxy)
- **Benefits**: Fast, cached, immutable (prevents disappearing dependencies)
- **Privacy**: Leaks which modules you download to Google
- **Disable**: `export GOPROXY=direct` (fetch from source repos directly)
- **Private modules**: `export GOPRIVATE=github.com/yourcompany/*`

**Q: How would you audit dependencies for security vulnerabilities?**

```bash
# Official Go vulnerability checker
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Check dependency licenses
go-licenses report ./... --template=licenses.tpl

# List all dependencies
go list -m all

# Show dependency tree
go mod graph | grep github.com/specific/package
```

---

## Dynamic SQL Query Building Patterns

When working with databases in Go, you'll often need to build SQL queries dynamically, such as batch INSERT statements. Here's how to do it efficiently and securely.

### The Problem: Batch INSERTs

When inserting many rows (e.g., blockchain transactions in a block), individual INSERT statements are inefficient due to network round-trips:

```go
// ❌ SLOW: Network round-trip per transaction
for _, tx := range transactions {
    _, err := db.Exec(`
        INSERT INTO transactions (chain_id, block_number, tx_hash, ...)
        VALUES ($1, $2, $3, ...)
    `, tx.ChainID, tx.BlockNumber, tx.Hash, ...)
}
// With 188 transactions and 300ms network latency = 56 seconds!
```

### Solution: Batch INSERT with Dynamic SQL

Build a single INSERT query with multiple value rows:

```go
// ✅ FAST: Single network round-trip
import (
    "database/sql"
    "fmt"
    "strings"
)

func insertBatch(db *sql.Tx, transactions []Transaction) error {
    if len(transactions) == 0 {
        return nil
    }
    
    // Use strings.Builder for efficient string building
    var query strings.Builder
    query.WriteString(`
        INSERT INTO transactions (
            chain_id, block_number, tx_hash, tx_index,
            from_address, to_address, value, gas_price, gas_used
        ) VALUES 
    `)
    
    // Pre-allocate args slice (9 fields × N transactions)
    args := make([]interface{}, 0, len(transactions)*9)
    
    for i, tx := range transactions {
        if i > 0 {
            query.WriteString(", ")
        }
        
        // Build placeholder string: ($1,$2,...,$9), ($10,$11,...,$18), ...
        base := i * 9
        query.WriteString(fmt.Sprintf(
            "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
            base+1, base+2, base+3, base+4, base+5,
            base+6, base+7, base+8, base+9,
        ))
        
        // Append values for this transaction
        args = append(args,
            tx.ChainID,
            tx.BlockNumber,
            tx.Hash,
            tx.Index,
            tx.FromAddress,
            tx.ToAddress,
            tx.Value,
            tx.GasPrice,
            tx.GasUsed,
        )
    }
    
    // Single Exec call with all data
    _, err := db.Exec(query.String(), args...)
    return err
}
```

### Why strings.Builder?

**String Concatenation Creates Garbage**:
```go
// ❌ BAD: Creates N temporary strings (memory allocations)
query := "INSERT INTO ... VALUES "
for i := 0; i < 1000; i++ {
    query += "($1,$2,$3),"  // New string allocation each time!
}
// With 1000 iterations, creates 1000 intermediate string objects
```

**strings.Builder is Efficient**:
```go
// ✅ GOOD: Single buffer, no allocations
var query strings.Builder
query.WriteString("INSERT INTO ... VALUES ")
for i := 0; i < 1000; i++ {
    query.WriteString("($1,$2,$3),")  // Appends to buffer
}
// Only one buffer, grows as needed
```

**Performance Comparison**:
```go
// Benchmark results (building 1000-placeholder query)
BenchmarkStringConcat    100  12,500,000 ns/op  500 MB allocated
BenchmarkStringsBuilder  5000    250,000 ns/op    2 MB allocated
// strings.Builder is 50x faster, 250x less memory
```

### PostgreSQL Placeholder Numbering

PostgreSQL uses `$1, $2, $3` for placeholders (unlike MySQL's `?`):

```go
// Single row: ($1, $2, $3)
// Two rows: ($1, $2, $3), ($4, $5, $6)
// Three rows: ($1, $2, $3), ($4, $5, $6), ($7, $8, $9)

// Formula for row i with n fields:
base := i * numFields
placeholders := fmt.Sprintf("($%d, $%d, ... $%d)",
    base+1, base+2, base+numFields)
```

### Security: Still Using Parameterized Queries

**CRITICAL**: We're building query **structure** dynamically, but values are still parameterized:

```go
// ✅ SAFE: Placeholders in query, values separate
query := "INSERT INTO users (name, email) VALUES ($1, $2), ($3, $4)"
args := []interface{}{"Alice", "alice@example.com", "Bob", "bob@example.com"}
db.Exec(query, args...)

// ❌ UNSAFE: Values concatenated into query string
query := fmt.Sprintf("INSERT INTO users VALUES ('%s', '%s')", name, email)
db.Exec(query)  // SQL injection vulnerable!
```

### Complete Example with Error Handling

```go
func insertTransactionsBatch(
    ctx context.Context,
    db *sql.Tx,
    chainID int64,
    transactions []Transaction,
) error {
    if len(transactions) == 0 {
        return nil
    }
    
    const numFields = 9
    
    // Build query
    var query strings.Builder
    query.WriteString(`
        INSERT INTO transactions (
            chain_id, block_number, tx_hash, tx_index,
            from_address, to_address, value, gas_price, gas_used
        ) VALUES 
    `)
    
    args := make([]interface{}, 0, len(transactions)*numFields)
    
    for i, tx := range transactions {
        if i > 0 {
            query.WriteString(", ")
        }
        
        base := i * numFields
        query.WriteString(fmt.Sprintf(
            "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
            base+1, base+2, base+3, base+4, base+5,
            base+6, base+7, base+8, base+9,
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
        )
    }
    
    // Execute with context timeout
    ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
    defer cancel()
    
    result, err := db.ExecContext(ctx, query.String(), args...)
    if err != nil {
        return fmt.Errorf("batch insert failed: %w", err)
    }
    
    // Verify rows affected
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if int(rowsAffected) != len(transactions) {
        return fmt.Errorf("expected %d rows, inserted %d",
            len(transactions), rowsAffected)
    }
    
    return nil
}
```

### Alternative: PostgreSQL COPY

For very large datasets (10,000+ rows), use the COPY protocol:

```go
import "github.com/lib/pq"

func insertWithCopy(db *sql.Tx, transactions []Transaction) error {
    stmt, err := db.Prepare(pq.CopyIn("transactions",
        "chain_id", "block_number", "tx_hash", "tx_index",
        "from_address", "to_address", "value", "gas_price", "gas_used"))
    if err != nil {
        return err
    }
    defer stmt.Close()
    
    for _, tx := range transactions {
        _, err := stmt.Exec(
            tx.ChainID, tx.BlockNumber, tx.Hash, tx.Index,
            tx.FromAddress, tx.ToAddress, tx.Value,
            tx.GasPrice, tx.GasUsed,
        )
        if err != nil {
            return err
        }
    }
    
    _, err = stmt.Exec()  // Flush
    return err
}
```

**COPY vs Batch INSERT**:
| Method | Speed | Complexity | Use Case |
|--------|-------|------------|----------|
| Batch INSERT | Fast (300x vs individual) | Medium | 50-1000 rows |
| COPY | Fastest (10x vs batch) | Higher | 1000+ rows |
| Individual | Slow | Simple | <50 rows or need per-row errors |

### Key Takeaways

1. **Use `strings.Builder` for dynamic SQL** - 50x faster than concatenation
2. **Pre-allocate slices** - `make([]interface{}, 0, expectedSize)` avoids reallocation
3. **Parameterized queries always** - Never concatenate user input into SQL
4. **Calculate placeholders correctly** - `base := i * numFields; $base+1, $base+2, ...`
5. **Context timeouts** - Prevent runaway queries
6. **Verify rows affected** - Ensure all data was inserted
7. **Benchmark alternatives** - COPY is faster for very large batches

**Real-world impact**: Reduced 188-transaction insert from 60 seconds to 200ms (300x improvement) by switching from individual INSERTs to batch INSERT with `strings.Builder`.

---

**Related Documents**:
- [Technology Stack](./01-technology-stack.md)
- [Docker & Kubernetes](./02-docker-kubernetes.md)
- [Go Concepts Deep Dive](./08-go-concepts-interview.md)
- [Implementation Concepts](./06-implementation-concepts.md)
- [Troubleshooting Guide](./09-troubleshooting.md)
