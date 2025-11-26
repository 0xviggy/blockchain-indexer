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

**Related Documents**:
- [Technology Stack](./01-technology-stack.md)
- [Docker & Kubernetes](./02-docker-kubernetes.md)
- [Go Concepts Deep Dive](./08-go-concepts-interview.md)
