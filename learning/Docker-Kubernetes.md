# Docker & Kubernetes

> ⚠️ **EDUCATIONAL MATERIAL** - Interview prep & learning reference  
> For project-specific setup, see [../SANDBOX_SETUP.md](../SANDBOX_SETUP.md)

> **Purpose**: Understanding containerization, orchestration, and deployment patterns for blockchain infrastructure. This guide covers Docker fundamentals, Kubernetes concepts, and production deployment strategies.

**Last Updated**: November 27, 2025

---

## Table of Contents

- [Docker Fundamentals](#docker-fundamentals)
  - [What is Docker?](#what-is-docker)
  - [Multi-Stage Builds](#multi-stage-builds)
  - [CMD vs ENTRYPOINT](#cmd-vs-entrypoint)
- [Docker Networking](#docker-networking)
  - [Network Types](#network-types)
  - [Container Communication](#how-container-communication-works)
  - [Port Mapping](#port-mapping)
  - [Network Inspection](#network-inspection)
- [Docker Volumes](#docker-volumes)
  - [Volume Types](#volume-types)
  - [Named Volumes vs Bind Mounts](#named-volumes-vs-bind-mounts)
  - [Volume Operations](#volume-operations)
  - [Performance Tips](#volume-performance-tips)
- [Debugging Containers](#debugging-containers)
  - [View Logs](#view-logs)
  - [Inspect Container](#inspect-container)
  - [Execute Commands](#execute-commands-in-running-container)
  - [Debug Crashed Containers](#debug-crashed-container)
  - [Resource Monitoring](#resource-monitoring)
- [Health Checks](#health-checks)
- [Kubernetes Fundamentals](#kubernetes-fundamentals)
  - [Docker Compose to Kubernetes](#translating-docker-compose-to-kubernetes)
  - [Scaling Patterns](#scaling-patterns)
- [Interview Questions & Answers](#interview-questions--answers)

---

## Docker Fundamentals

### What is Docker?

Docker is a platform for building, shipping, and running applications in containers. A container packages an application with all its dependencies into a standardized unit, ensuring consistency across development, testing, and production environments.

**Key Concepts**:
- **Image**: Blueprint for containers (read-only template)
- **Container**: Running instance of an image (writable layer on top of image)
- **Dockerfile**: Text file with instructions to build an image
- **Registry**: Storage for images (Docker Hub, GitHub Container Registry, private registries)

**Why Containers?**
- **Consistency**: "Works on my machine" → "Works everywhere"
- **Isolation**: Each container has its own filesystem, network, processes
- **Efficiency**: Share host OS kernel, faster than VMs
- **Portability**: Run anywhere Docker is installed

### Multi-Stage Builds

**Problem**: Development images are huge (800MB+ with SDK, build tools, test dependencies, cache files)  
**Solution**: Build in one image, copy only runtime artifacts to minimal final image

```dockerfile
# Stage 1: Build
FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service ./cmd/service

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/service .
CMD ["./service"]
```

**Benefits**:
- **Size**: 800MB → 15MB (53x smaller)
- **Security**: No build tools in production image (reduced attack surface)
- **Speed**: Smaller images = faster deploys, less bandwidth
- **Layers**: Only runtime layer changes on code updates (efficient caching)

**Best Practices**:
```dockerfile
# 1. Use specific versions (not :latest) for reproducibility
FROM golang:1.21.5-alpine AS builder

# 2. Order layers by change frequency (least → most) for better caching
COPY go.mod go.sum ./        # Changes rarely (dependencies)
RUN go mod download           # Cached unless go.mod changes
COPY . .                      # Changes often (source code)
RUN go build ./...            # Only rebuilds if source changes

# 3. Use .dockerignore to exclude unnecessary files
# .dockerignore:
# .git
# *.md
# tests/
# .env.local
# node_modules/
# *.log
```

**Multi-Stage Build for Our Indexer**:
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY shared/ ./shared/
COPY services/ingester/go.mod services/ingester/go.sum ./
RUN go mod download
COPY services/ingester/ ./
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o ingester main.go

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/ingester .
USER nobody
CMD ["./ingester"]
```

### CMD vs ENTRYPOINT

Understanding the difference is crucial for flexible, production-ready containers.

**CMD**: Default command, can be completely overridden  
**ENTRYPOINT**: Always runs, CMD becomes arguments to ENTRYPOINT

```dockerfile
# CMD only (can override entire command)
CMD ["./service", "--config", "prod.yaml"]
# Run: docker run myimage ./other-command  ✅ Replaces entire CMD

# ENTRYPOINT only (always runs this)
ENTRYPOINT ["./service"]
# Run: docker run myimage  ✅ Runs ./service
# Run: docker run myimage --debug  ✅ Runs ./service --debug

# Both (flexible configuration) - RECOMMENDED
ENTRYPOINT ["./service"]
CMD ["--config", "prod.yaml"]
# Run: docker run myimage  → ./service --config prod.yaml
# Run: docker run myimage --debug  → ./service --debug (overrides CMD)
```

**When to Use**:
- **CMD**: Provide default arguments that users might override
- **ENTRYPOINT**: Ensure a specific command always runs
- **Both**: Command in ENTRYPOINT, default config in CMD (most flexible)

**Real Example - Ingester with Configuration**:
```dockerfile
FROM alpine:latest
WORKDIR /app
COPY ingester .
ENTRYPOINT ["./ingester"]
CMD ["--chain-id", "1", "--start-block", "latest"]

# Usage:
# Default: docker run ingester → ./ingester --chain-id 1 --start-block latest
# Override: docker run ingester --chain-id 137 --start-block 50000000
```

---

## Docker Networking

### Network Types

Docker provides multiple network drivers for different use cases:

1. **Bridge** (default): Isolated network with port mapping
2. **Host**: Share host network stack (no isolation, best performance)
3. **None**: No networking (maximum security, isolated containers)
4. **Overlay**: Multi-host networking (Docker Swarm/Kubernetes)

```yaml
# Docker Compose network configuration
services:
  api:
    networks:
      - frontend  # Accessible from outside
      - backend   # Internal communication
  
  postgres:
    networks:
      - backend   # Not accessible from outside

networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true  # No internet access, only inter-container
```

**Network Isolation Strategy**:
- Public-facing services (API, web): `frontend` + `backend` networks
- Databases, caches: `backend` network only
- Internal services (workers): `backend` network only

### How Container Communication Works

Docker Compose creates automatic DNS resolution for service names:

```bash
# Inside api container:
curl http://postgres:5432  # ✅ DNS resolves to postgres container IP

# Docker Compose also creates network alias
# Can use service name or container name
curl http://indexer-postgres:5432  # ✅ Also works

# Check container IP from host
docker inspect indexer-postgres | jq '.[0].NetworkSettings.Networks'
```

**DNS Resolution Process**:
1. Container queries `postgres`
2. Docker's embedded DNS server (127.0.0.11) resolves to container IP
3. Connection established within Docker network (no port mapping needed)

**Important**: Containers on the same network communicate via internal IPs (no need for port publishing). Port publishing (`ports:`) is only needed for external access.

### Port Mapping

```yaml
services:
  api:
    ports:
      - "8000:8000"              # host:container (accessible from anywhere)
      - "127.0.0.1:8001:8001"    # Only localhost can access
      - "8002"                   # Random host port → container 8002
      
  postgres:
    ports:
      - "5432:5432"              # Development: direct DB access
    # Production: Remove port mapping (internal access only via backend network)
```

**Security Best Practice**:
```yaml
# Development
ports:
  - "5432:5432"  # Convenient for local tools (pgAdmin, DBeaver)

# Production
expose:
  - "5432"  # Only accessible within Docker network, not from host
```

### Network Inspection

```bash
# List networks
docker network ls

# Inspect network (see connected containers, IP ranges)
docker network inspect indexer-network

# See which containers are on network
docker network inspect indexer-network | jq '.[0].Containers'

# Connect running container to network
docker network connect indexer-network my-container

# Disconnect container from network
docker network disconnect indexer-network my-container

# Create custom network
docker network create --driver bridge --subnet 172.25.0.0/16 custom-network
```

---

## Docker Volumes

### Volume Types

Docker provides three ways to persist data:

1. **Named Volumes**: Managed by Docker, best for production
2. **Bind Mounts**: Mount host directory, best for development
3. **tmpfs Mounts**: In-memory, best for sensitive temporary data

```yaml
services:
  postgres:
    volumes:
      # Named volume (managed by Docker, recommended for production)
      - postgres_data:/var/lib/postgresql/data
      
      # Bind mount (host directory, good for development)
      - ./backups:/backups
      
      # tmpfs (in-memory, not persisted, good for temp data)
      - type: tmpfs
        target: /tmp

volumes:
  postgres_data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /mnt/data/postgres  # Optional: specific location on host
```

### Named Volumes vs Bind Mounts

| Feature | Named Volumes | Bind Mounts |
|---------|--------------|-------------|
| Management | Docker manages location | You manage location |
| Location | `/var/lib/docker/volumes/` | Anywhere on host |
| Performance | Optimized by Docker | Direct filesystem access |
| Backups | Use `docker cp` or volume commands | Regular file backups |
| Portability | Portable across systems | Depends on host paths |
| Use Case | Production databases, persistent data | Development code, config files |
| Permissions | Docker handles | Host filesystem permissions apply |

### Volume Operations

```bash
# Create volume
docker volume create postgres_data

# List volumes
docker volume ls

# Inspect volume (see mount point, size, driver)
docker volume inspect postgres_data

# Backup volume
docker run --rm \
  -v postgres_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/postgres_backup.tar.gz /data

# Restore volume
docker run --rm \
  -v postgres_data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/postgres_backup.tar.gz -C /

# Remove volume (after stopping containers)
docker volume rm postgres_data

# Remove all unused volumes (careful!)
docker volume prune

# Copy data from container to host
docker cp container_name:/var/lib/postgresql/data ./backup

# Copy data from host to container
docker cp ./backup/data.sql container_name:/tmp/
```

### Volume Performance Tips

```yaml
# For macOS/Windows: Use delegated/cached for better performance
services:
  app:
    volumes:
      - ./src:/app/src:delegated  # Host → Container (writes delayed)
      - ./cache:/app/cache:cached  # Container → Host (reads cached)
      
  # For databases: Use named volumes (better performance than bind mounts)
  postgres:
    volumes:
      - postgres_data:/var/lib/postgresql/data  # Named volume (fast)
      # NOT: ./data:/var/lib/postgresql/data    # Bind mount (slower on Mac/Win)
```

**Performance Comparison** (macOS/Windows):
- Named volumes: ~native performance
- Bind mounts: 2-10x slower (due to filesystem translation layer)
- tmpfs: fastest (in-memory)

---

## Debugging Containers

### View Logs

```bash
# Follow logs (real-time)
docker logs -f container_name

# Last 100 lines
docker logs --tail 100 container_name

# Logs since timestamp
docker logs --since 2024-11-27T10:00:00 container_name

# Logs for specific time range
docker logs --since 2024-11-27T10:00:00 --until 2024-11-27T11:00:00 container_name

# Logs for specific service in Compose
docker compose logs -f postgres

# Logs with timestamps
docker logs -t container_name

# Filter logs (search for errors)
docker logs container_name 2>&1 | grep ERROR
```

### Inspect Container

```bash
# Full container details (network, volumes, env, etc.)
docker inspect container_name

# Get specific field using jq
docker inspect container_name | jq '.[0].State.Status'
docker inspect container_name | jq '.[0].NetworkSettings.IPAddress'

# See mounts (volumes and bind mounts)
docker inspect container_name | jq '.[0].Mounts'

# See environment variables
docker inspect container_name | jq '.[0].Config.Env'

# See health check status
docker inspect container_name | jq '.[0].State.Health'

# See resource limits
docker inspect container_name | jq '.[0].HostConfig.Memory'
docker inspect container_name | jq '.[0].HostConfig.CpuShares'
```

### Execute Commands in Running Container

```bash
# Open interactive shell
docker exec -it container_name /bin/sh
docker exec -it container_name /bin/bash  # If bash available

# Run single command
docker exec container_name ls /var/lib/postgresql/data

# Run as different user
docker exec -u postgres container_name psql -U indexer

# Run command with environment variable
docker exec -e DEBUG=true container_name ./script.sh

# Copy files from container
docker exec container_name cat /app/config.yaml > local_config.yaml
```

### Debug Crashed Container

```bash
# View logs even after container exit
docker logs container_name

# Check exit code
docker inspect container_name | jq '.[0].State.ExitCode'
# Common exit codes:
# 0   = success
# 1   = application error
# 2   = misuse of shell command
# 126 = command cannot execute
# 127 = command not found
# 137 = killed (SIGKILL) - usually OOM
# 143 = terminated (SIGTERM)

# Start container with shell override (debug startup issues)
docker run -it --entrypoint /bin/sh image_name

# Keep container running for debugging
docker run -it --entrypoint /bin/sh image_name
# Inside: manually run the failing command

# Copy files from stopped container
docker cp container_name:/app/logs ./logs

# Restart crashed container
docker start container_name
```

### Resource Monitoring

```bash
# Real-time stats (CPU, memory, network, disk I/O)
docker stats

# Stats for specific container
docker stats container_name

# One-time snapshot (no streaming)
docker stats --no-stream

# Format output
docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}"

# Check disk usage
docker system df

# Detailed disk usage (images, containers, volumes)
docker system df -v

# See top processes in container
docker top container_name

# See resource limits
docker inspect container_name | jq '.[0].HostConfig.Memory'
```

---

## Health Checks

**Why Health Checks**: Container might be running but application is unhealthy (e.g., database accepting connections but deadlocked, API responding but failing all requests).

```yaml
services:
  postgres:
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U indexer"]
      interval: 10s        # Check every 10 seconds
      timeout: 5s          # Timeout after 5 seconds
      retries: 3           # Fail after 3 consecutive failures
      start_period: 30s    # Grace period on startup (don't count failures)
  
  api:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    depends_on:
      postgres:
        condition: service_healthy  # Wait for postgres to be healthy
```

**Health Check States**:
- **starting**: During `start_period` (failures ignored)
- **healthy**: Check passed `retries` times
- **unhealthy**: Check failed `retries` consecutive times

**Check Health Status**:
```bash
# See health status in ps output
docker ps

# Detailed health info
docker inspect container_name | jq '.[0].State.Health'

# See last 5 health check results
docker inspect container_name | jq '.[0].State.Health.Log[-5:]'

# Auto-restart unhealthy containers
docker run --restart=unless-stopped \
  --health-cmd="curl -f http://localhost:8000/health" \
  --health-interval=30s \
  myimage
```

**Best Practices**:
- **Shallow checks**: Fast, cheap operations (e.g., `pg_isready`, not full query)
- **Realistic checks**: Test what matters (e.g., API endpoint that exercises critical path)
- **Appropriate intervals**: Balance responsiveness vs overhead (10-30s typical)
- **Startup period**: Give services time to initialize (databases need 10-60s)

---

## Kubernetes Fundamentals

### Translating Docker Compose to Kubernetes

Kubernetes uses multiple resource types to achieve what Docker Compose does in one file.

**Our Docker Compose**:
```yaml
services:
  postgres:
    image: postgres:15-alpine
    ports: ["5432:5432"]
    volumes: [postgres_data:/var/lib/postgresql/data]
    environment:
      POSTGRES_USER: indexer
      POSTGRES_PASSWORD: indexer_pass
```

**Equivalent Kubernetes** (3 separate resources):

```yaml
# 1. Deployment (defines the application)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc

---
# 2. Service (networking and discovery)
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: default
spec:
  type: ClusterIP  # Internal only (default)
  ports:
  - port: 5432
    targetPort: 5432
  selector:
    app: postgres  # Routes traffic to pods with this label

---
# 3. PersistentVolumeClaim (storage request)
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: default
spec:
  accessModes:
    - ReadWriteOnce  # Single node can mount read-write
  resources:
    requests:
      storage: 20Gi
  storageClassName: standard  # Cloud provider's storage class

---
# 4. Secret (sensitive data)
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: default
type: Opaque
data:
  username: aW5kZXhlcg==  # base64 encoded "indexer"
  password: aW5kZXhlcl9wYXNz  # base64 encoded "indexer_pass"
```

**Key Differences**:

| Docker Compose | Kubernetes | Purpose |
|----------------|------------|---------|
| `services` | Deployment | Defines application pods |
| (implicit) | Service | Networking and discovery |
| `volumes` | PersistentVolumeClaim | Storage request |
| `environment` | ConfigMap/Secret | Configuration |
| `ports` | Service.spec.ports | Port exposure |
| `depends_on` | Init containers | Startup ordering |

### Kubernetes Core Concepts

**Pod**: Smallest deployable unit, one or more containers  
**Deployment**: Manages replicas of pods, rolling updates  
**Service**: Load balancer and DNS name for pods  
**PersistentVolume (PV)**: Actual storage (like physical disk)  
**PersistentVolumeClaim (PVC)**: Storage request (like reservation)  
**ConfigMap**: Non-sensitive configuration  
**Secret**: Sensitive configuration (passwords, API keys)  
**Namespace**: Logical cluster subdivision  

**Deployment vs StatefulSet**:
- **Deployment**: Stateless apps, pods interchangeable, random names (`api-7d9f8b-xyz`)
- **StatefulSet**: Stateful apps, stable identities (`postgres-0`, `postgres-1`), ordered startup
- **Use StatefulSet for**: Databases, Kafka, ZooKeeper, any app needing stable network identity

**Service Discovery**:
```bash
# Full DNS name
postgres.default.svc.cluster.local

# Short form (within same namespace)
postgres

# Cross-namespace
postgres.production.svc.cluster.local
```

**PersistentVolume Lifecycle**:
1. **PV** provisioned (admin or dynamic provisioner)
2. **PVC** created (developer requests storage)
3. **Binding** (PVC binds to suitable PV)
4. **Usage** (pod mounts PVC)
5. **Reclaim** (after PVC deleted, PV retained/deleted based on policy)

---

## Scaling Patterns

### Horizontal Pod Autoscaler

Automatically scales pods based on CPU, memory, or custom metrics.

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-scaler
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70  # Scale up if CPU > 70%
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80  # Scale up if memory > 80%
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300  # Wait 5 min before scaling down
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60  # Scale down max 50% per minute
    scaleUp:
      stabilizationWindowSeconds: 0  # Scale up immediately
      policies:
      - type: Percent
        value: 100
        periodSeconds: 60  # Scale up max 100% (double) per minute
```

**Custom Metrics Example** (scale on Kafka lag):
```yaml
metrics:
- type: External
  external:
    metric:
      name: kafka_consumer_lag
      selector:
        matchLabels:
          topic: blocks
    target:
      type: AverageValue
      averageValue: "1000"  # Scale up if lag > 1000 messages per pod
```

### Vertical Pod Autoscaler

Automatically adjusts CPU and memory requests/limits.

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: postgres-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: postgres
  updatePolicy:
    updateMode: "Auto"  # Automatically restart pods with new resources
  resourcePolicy:
    containerPolicies:
    - containerName: postgres
      minAllowed:
        cpu: 1
        memory: 2Gi
      maxAllowed:
        cpu: 4
        memory: 16Gi
```

### Scaling Strategies for Our Indexer

**API Service** (stateless):
- Use HorizontalPodAutoscaler
- Scale based on CPU and request count
- min: 2, max: 20

**Ingester Service** (stateful - maintains chain position):
- Partition by chain: One pod per chain
- Use StatefulSet with stable names (`ingester-eth-0`, `ingester-polygon-0`)
- Scale vertically (larger pods) rather than horizontally
- Use Kubernetes Jobs for historical catch-up

**Processor Service** (stateless consumer):
- Use HorizontalPodAutoscaler
- Scale based on Kafka consumer lag
- Partitioned by chain_id (Kafka partitions)

---

## Interview Questions & Answers

### Q1: Explain Docker multi-stage builds and why they matter for production

**Answer**: Multi-stage builds allow you to use different base images for building and running your application, dramatically reducing final image size and improving security.

**How it works**:
1. **Build stage**: Use full SDK image (e.g., `golang:1.21`) to compile application
2. **Runtime stage**: Use minimal image (e.g., `alpine`) and copy only compiled binary
3. **Result**: 800MB build image → 15MB production image (53x smaller)

**Benefits for production**:
- **Faster deploys**: Smaller images transfer quicker (critical for auto-scaling)
- **Reduced attack surface**: No compiler, build tools, or source code in production
- **Lower costs**: Less storage, bandwidth, and memory usage
- **Better caching**: Separate layers for dependencies (cached) and code (changes often)

**Real example from our indexer**:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download  # ← Cached until dependencies change
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o ingester

FROM alpine:latest
COPY --from=builder /build/ingester .
USER nobody  # ← Run as non-root
CMD ["./ingester"]
```

### Q2: What's the difference between CMD and ENTRYPOINT, and when would you use both together?

**Answer**: CMD provides default arguments that can be overridden, while ENTRYPOINT defines the executable that always runs. Using both together creates a flexible container that has a fixed command but configurable arguments.

**CMD only**:
```dockerfile
CMD ["./app", "--port", "8000"]
# docker run myimage → ./app --port 8000
# docker run myimage ./other-app → ./other-app (completely replaces CMD)
```

**ENTRYPOINT only**:
```dockerfile
ENTRYPOINT ["./app"]
# docker run myimage → ./app
# docker run myimage --port 9000 → ./app --port 9000
```

**Both (recommended pattern)**:
```dockerfile
ENTRYPOINT ["./app"]
CMD ["--port", "8000", "--log-level", "info"]
# docker run myimage → ./app --port 8000 --log-level info
# docker run myimage --port 9000 → ./app --port 9000 (overrides CMD)
```

**Use cases**:
- **CMD only**: General-purpose image that could run different commands
- **ENTRYPOINT only**: Simple apps with no configuration needed
- **Both**: Production apps needing default config but allowing overrides (most common)

**Our ingester example**:
```dockerfile
ENTRYPOINT ["./ingester"]
CMD ["--chain-id", "1", "--start-block", "latest"]
# Default: Ethereum mainnet from latest
# Override: docker run ingester --chain-id 137 --start-block 50000000
```

### Q3: How does Docker networking work? Explain bridge networks and service discovery

**Answer**: Docker creates isolated networks where containers can communicate using service names as DNS entries, without exposing ports to the host.

**Bridge Network** (default):
- Each container gets a private IP (e.g., 172.17.0.2)
- Docker runs embedded DNS server (127.0.0.11) for service discovery
- Containers on same bridge can communicate via service names
- Port publishing (ports:) only needed for external access

**Example**:
```yaml
services:
  api:
    networks: [backend]
    # No ports: needed - internal only
  
  postgres:
    networks: [backend]

networks:
  backend:
    driver: bridge
```

Inside `api` container:
```bash
curl http://postgres:5432  # ✅ DNS resolves to postgres container IP
ping postgres  # ✅ Works
telnet postgres 5432  # ✅ Direct connection
```

**Service Discovery Process**:
1. Container queries `postgres` DNS name
2. Docker's DNS server (127.0.0.11) intercepts query
3. Returns IP of container with service name `postgres`
4. Direct TCP connection established within Docker network

**Network Isolation**:
```yaml
networks:
  frontend:  # Public services
  backend:   # Internal services
    internal: true  # No internet access
```

**Best practice**: Only expose ports for services that need external access (API, web). Keep databases internal.

### Q4: What are Docker volumes and when would you use named volumes vs bind mounts?

**Answer**: Docker volumes persist data beyond container lifecycle. Named volumes are managed by Docker (best for production), bind mounts link to host directories (best for development).

**Named Volumes**:
```yaml
volumes:
  - postgres_data:/var/lib/postgresql/data
```
- Docker manages location (`/var/lib/docker/volumes/`)
- Better performance (especially on Mac/Windows)
- Portable across systems
- Can be backed up with `docker` commands
- Use for: Production databases, persistent application data

**Bind Mounts**:
```yaml
volumes:
  - ./src:/app/src
  - ./config.yaml:/app/config.yaml
```
- Direct mount of host directory
- Changes instantly reflected in container (hot reload)
- Easy to edit with host tools
- Performance penalty on Mac/Windows (2-10x slower)
- Use for: Development code, configuration files, logs

**tmpfs Mounts** (bonus):
```yaml
volumes:
  - type: tmpfs
    target: /tmp
```
- In-memory, never written to disk
- Fastest performance
- Data lost when container stops
- Use for: Temporary files, sensitive data (passwords in memory)

**Production setup**:
```yaml
services:
  postgres:
    volumes:
      - postgres_data:/var/lib/postgresql/data  # Named volume (production)
      - ./backups:/backups:ro  # Bind mount read-only (backup scripts)
```

### Q5: How would you debug a container that keeps crashing on startup?

**Answer**: Systematic approach using logs, exit codes, and interactive debugging.

**Step 1: Check logs and exit code**
```bash
docker logs container_name
docker inspect container_name | jq '.[0].State.ExitCode'
```

**Common exit codes**:
- `1`: Application error (check logs)
- `137`: Killed by OOM (out of memory)
- `143`: SIGTERM (graceful shutdown)
- `127`: Command not found (wrong ENTRYPOINT/CMD)

**Step 2: Override entrypoint for interactive debugging**
```bash
docker run -it --entrypoint /bin/sh image_name
# Inside container:
./app  # Manually run the app to see errors
env    # Check environment variables
ls -la # Verify files are present
```

**Step 3: Check resource limits**
```bash
docker stats container_name  # See if hitting memory limits
docker inspect container_name | jq '.[0].HostConfig.Memory'
```

**Step 4: Check dependencies**
```bash
# Is database ready?
docker logs postgres_container
docker exec postgres_container pg_isready

# Can container reach dependencies?
docker exec api_container ping postgres
```

**Step 5: Common issues**:
- **Missing environment variables**: Check `docker inspect` or add `env` command
- **File permissions**: Run `ls -la` inside container, check USER directive
- **Network issues**: Verify containers are on same network
- **OOM**: Increase memory limits or optimize application
- **Wrong working directory**: Check WORKDIR and file paths

**Example debugging session**:
```bash
# Container crashes immediately
docker logs ingester
# Error: "config.yaml not found"

# Override entrypoint to investigate
docker run -it --entrypoint /bin/sh ingester-image
ls -la  # config.yaml missing
pwd     # /root/, but Dockerfile sets WORKDIR /app

# Fix: Update COPY or WORKDIR in Dockerfile
```

### Q6: Explain Kubernetes Deployments vs StatefulSets and when to use each

**Answer**: Deployments manage stateless applications where pods are interchangeable. StatefulSets manage stateful applications that need stable identities and ordered deployment.

**Deployment** (stateless):
- Pods are identical and interchangeable
- Random names: `api-7d9f8b-xyz`, `api-9c2k1l-abc`
- Can be replaced, reordered, scaled freely
- Use for: REST APIs, stateless workers, web servers

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3  # All pods identical
  template:
    spec:
      containers:
      - name: api
        image: api:latest
```

**StatefulSet** (stateful):
- Pods have stable, unique identities
- Ordered names: `postgres-0`, `postgres-1`, `postgres-2`
- Ordered deployment and scaling (0 → 1 → 2)
- Stable network identities (DNS entries per pod)
- Each pod can have its own PersistentVolumeClaim
- Use for: Databases, Kafka, ZooKeeper, distributed systems

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
spec:
  serviceName: postgres  # ← Required for stable network identity
  replicas: 3
  template:
    spec:
      containers:
      - name: postgres
        image: postgres:15
  volumeClaimTemplates:  # ← Each pod gets its own PVC
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 10Gi
```

**StatefulSet features**:
- **Stable DNS**: `postgres-0.postgres.default.svc.cluster.local`
- **Ordered scaling**: When scaling 1→3, creates 0→1→2 (waits for each to be ready)
- **Ordered updates**: Updates in reverse order 2→1→0 (safer for databases)
- **Persistent identity**: If `postgres-1` dies, new pod gets same name and PVC

**Our indexer use case**:
- **API service**: Deployment (stateless, any pod can handle any request)
- **Ingester service**: StatefulSet (each pod tracks specific chain state)
- **Processor service**: Deployment (stateless Kafka consumers)

### Q7: How does Kubernetes service discovery work? Explain ClusterIP, NodePort, and LoadBalancer

**Answer**: Kubernetes Services provide stable DNS names and load balancing for pods. Different service types expose applications to different audiences.

**ClusterIP** (default - internal only):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  type: ClusterIP
  ports:
  - port: 5432
    targetPort: 5432
  selector:
    app: postgres
```
- Only accessible within cluster
- Gets cluster-internal IP (e.g., 10.96.0.10)
- DNS: `postgres.default.svc.cluster.local` → 10.96.0.10
- Use for: Internal services (databases, caches, internal APIs)

**NodePort** (external access via node IP):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  type: NodePort
  ports:
  - port: 8000
    targetPort: 8000
    nodePort: 30100  # Exposed on all nodes
  selector:
    app: api
```
- Accessible via `<NodeIP>:30100`
- Port range: 30000-32767
- Use for: Development, testing, on-premise clusters

**LoadBalancer** (cloud-provisioned external load balancer):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: api-public
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8000
  selector:
    app: api
```
- Cloud provider provisions external LB (AWS ELB, GCP LB, Azure LB)
- Gets public IP automatically
- Use for: Production public-facing services

**Service Discovery**:
```bash
# From any pod in cluster:
curl http://postgres:5432  # ← Short name (same namespace)
curl http://postgres.default:5432  # ← With namespace
curl http://postgres.default.svc.cluster.local:5432  # ← Fully qualified

# Kubernetes DNS returns:
# - ClusterIP for load balancing across all pods
# - Service automatically updates as pods are added/removed
```

**How it works**:
1. Service selector matches pods by labels
2. Service creates endpoints list (pod IPs)
3. kube-proxy configures iptables rules for load balancing
4. DNS entry points to ClusterIP
5. Traffic distributed across healthy pods

**Best practice**:
- Internal services: ClusterIP
- Development: NodePort
- Production public: LoadBalancer + Ingress

### Q8: What are liveness, readiness, and startup probes? When would you use each?

**Answer**: Kubernetes probes detect unhealthy containers and control traffic routing. Each probe serves a different purpose in the container lifecycle.

**Liveness Probe** - "Is the container healthy?"
- If fails: Kubernetes restarts container
- Use for: Detecting deadlocks, infinite loops, app crashes
- Example: Application frozen, accepting connections but not responding

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8000
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3  # Restart after 3 consecutive failures
```

**Readiness Probe** - "Can the container accept traffic?"
- If fails: Removes pod from Service endpoints (no traffic sent)
- Container keeps running (not restarted)
- Use for: Temporary issues (high load, warming up, dependencies unavailable)
- Example: Database connection pool exhausted, cache warming up

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8000
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2  # Remove from Service after 2 failures
```

**Startup Probe** - "Has the container finished starting?"
- If fails: Kubernetes restarts container
- Disables liveness/readiness checks during startup
- Use for: Slow-starting applications (databases, large applications)
- Example: Application takes 2 minutes to initialize

```yaml
startupProbe:
  httpGet:
    path: /healthz
    port: 8000
  initialDelaySeconds: 0
  periodSeconds: 10
  failureThreshold: 30  # 30 * 10s = 5 minutes max startup time
```

**Real-world example - PostgreSQL**:
```yaml
spec:
  containers:
  - name: postgres
    startupProbe:
      exec:
        command: ["pg_isready", "-U", "indexer"]
      failureThreshold: 30  # 5 minutes to start
      periodSeconds: 10
    
    livenessProbe:
      exec:
        command: ["pg_isready", "-U", "indexer"]
      periodSeconds: 10
      failureThreshold: 3
    
    readinessProbe:
      exec:
        command:
        - bash
        - -c
        - "pg_isready -U indexer && psql -U indexer -c 'SELECT 1'"
      periodSeconds: 5
      failureThreshold: 2
```

**Probe lifecycle**:
1. Container starts
2. **Startup probe** runs until success (liveness/readiness disabled)
3. **Liveness probe** runs continuously (restarts if fails)
4. **Readiness probe** runs continuously (controls traffic)

**Best practices**:
- **Startup**: For slow-starting apps (databases, JVM apps)
- **Liveness**: For detecting fatal issues (deadlocks, crashes)
- **Readiness**: For temporary issues (warming up, high load)
- **Probe endpoints**: Lightweight, fast checks (< 1s)

### Q9: How would you scale a blockchain ingester service across multiple chains using Kubernetes?

**Answer**: Use StatefulSet with one pod per chain, leveraging stable identities for checkpoint management and ordered scaling.

**Architecture**:
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: ingester
spec:
  serviceName: ingester
  replicas: 5  # 5 chains: ETH, Polygon, Arbitrum, Base, Optimism
  template:
    metadata:
      labels:
        app: ingester
    spec:
      containers:
      - name: ingester
        image: indexer/ingester:latest
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: CHAIN_ID
          value: "$(POD_NAME | extract chain)"  # ingester-0 → chain 1
        command:
        - /bin/sh
        - -c
        - |
          case $POD_NAME in
            ingester-0) CHAIN_ID=1 ;;      # Ethereum
            ingester-1) CHAIN_ID=137 ;;    # Polygon
            ingester-2) CHAIN_ID=42161 ;;  # Arbitrum
            ingester-3) CHAIN_ID=8453 ;;   # Base
            ingester-4) CHAIN_ID=10 ;;     # Optimism
          esac
          ./ingester --chain-id=$CHAIN_ID
        resources:
          requests:
            cpu: "1"
            memory: "2Gi"
          limits:
            cpu: "2"
            memory: "4Gi"
        volumeMounts:
        - name: checkpoint
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: checkpoint
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi  # Store last synced block per chain
```

**Alternative approach - ConfigMap for chain assignment**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ingester-chains
data:
  ingester-0: "1"      # Ethereum
  ingester-1: "137"    # Polygon
  ingester-2: "42161"  # Arbitrum
  ingester-3: "8453"   # Base
  ingester-4: "10"     # Optimism
```

**Benefits of StatefulSet**:
- **Stable identity**: `ingester-0` always handles Ethereum
- **Persistent storage**: Each pod has dedicated PVC for checkpoints
- **Ordered scaling**: Add chains one at a time (ingester-5, ingester-6)
- **DNS per pod**: `ingester-0.ingester.default.svc.cluster.local`

**Horizontal scaling per chain** (if needed):
```yaml
# For high-throughput chains (Ethereum), use multiple pods
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ingester-eth
spec:
  replicas: 3  # 3 pods for Ethereum
  template:
    spec:
      containers:
      - name: ingester
        env:
        - name: CHAIN_ID
          value: "1"
        - name: BLOCK_RANGE
          value: "$(POD_NAME | extract range)"  # Each pod handles block range
```

**HorizontalPodAutoscaler for processor** (downstream Kafka consumers):
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: processor-scaler
spec:
  scaleTargetRef:
    kind: Deployment
    name: processor
  minReplicas: 5
  maxReplicas: 50
  metrics:
  - type: External
    external:
      metric:
        name: kafka_consumer_lag
      target:
        type: AverageValue
        averageValue: "10000"  # Scale up if lag > 10k messages per pod
```

**Why not simple Deployment?**
- Deployment pods are interchangeable (any pod could handle any chain)
- No persistent identity (pod names change: `ingester-7d9f8b-xyz`)
- Checkpoint storage harder (need external state tracking)
- StatefulSet provides built-in identity and storage management

### Q10: Explain Kubernetes resource requests and limits, and how they affect scheduling and QoS

**Answer**: Requests reserve resources (guarantee), limits cap usage (hard ceiling). They determine pod scheduling and Quality of Service class.

**Requests vs Limits**:
```yaml
resources:
  requests:
    cpu: "500m"     # 0.5 CPU cores (guaranteed)
    memory: "1Gi"   # 1 GiB RAM (guaranteed)
  limits:
    cpu: "2"        # Max 2 CPU cores
    memory: "4Gi"   # Max 4 GiB RAM (OOM kill if exceeded)
```

**How it works**:
- **Requests**: Kubernetes scheduler finds node with available capacity
- **Limits**: Kubelet enforces on the node (cgroups)

**CPU behavior**:
- Requests: Guaranteed CPU time (500m = 50% of one core)
- Limits: Throttled if exceeded (slowed down, not killed)
- Burstable: Can use more than request if available

**Memory behavior**:
- Requests: Guaranteed memory (1Gi minimum)
- Limits: OOM killed if exceeded (hard limit)
- Non-burstable: Cannot exceed limit

**Quality of Service Classes**:

**1. Guaranteed** (highest priority):
```yaml
resources:
  requests:
    cpu: "1"
    memory: "2Gi"
  limits:
    cpu: "1"      # Limits = Requests
    memory: "2Gi"
```
- Last to be evicted under node pressure
- Use for: Critical services (databases, core APIs)

**2. Burstable** (medium priority):
```yaml
resources:
  requests:
    cpu: "500m"
    memory: "1Gi"
  limits:
    cpu: "2"      # Limits > Requests
    memory: "4Gi"
```
- Can burst above requests if resources available
- Evicted before Guaranteed pods
- Use for: Most applications (APIs, workers)

**3. BestEffort** (lowest priority):
```yaml
# No requests or limits specified
```
- No guarantees, uses whatever is available
- First to be evicted under node pressure
- Use for: Batch jobs, non-critical tasks

**Scheduling example**:
```
Node capacity: 4 CPU, 16Gi RAM
Running pods:
- postgres (Guaranteed): 1 CPU, 4Gi requests
- api (Burstable): 1 CPU, 2Gi requests (2 CPU, 8Gi limits)
- worker (Burstable): 500m CPU, 1Gi requests (1 CPU, 4Gi limits)

Available for scheduling: 1.5 CPU, 9Gi RAM
New pod requests: 2 CPU, 4Gi → REJECTED (insufficient CPU)
New pod requests: 1 CPU, 8Gi → REJECTED (insufficient RAM)
New pod requests: 1 CPU, 4Gi → SCHEDULED
```

**Best practices for our indexer**:
```yaml
# Ingester (critical, predictable load)
resources:
  requests:
    cpu: "1"
    memory: "2Gi"
  limits:
    cpu: "1"      # Guaranteed QoS
    memory: "2Gi"

# API (burstable, variable load)
resources:
  requests:
    cpu: "500m"
    memory: "1Gi"
  limits:
    cpu: "2"      # Burstable QoS
    memory: "4Gi"

# Processor (can tolerate eviction)
resources:
  requests:
    cpu: "200m"
    memory: "512Mi"
  limits:
    cpu: "1"      # Burstable QoS
    memory: "2Gi"
```

**Common mistakes**:
- No limits: Pod can consume all node resources (OOM kills other pods)
- Limits too low: Pod constantly throttled or OOM killed
- Requests too high: Wasted resources, poor bin packing
- Requests too low: Node overcommitted, performance issues

### Q11: How would you implement rolling updates and rollbacks for a Kubernetes deployment?

**Answer**: Kubernetes Deployments support declarative rolling updates with automatic rollback capabilities. You control update speed and health checks during rollout.

**Rolling Update Strategy**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 10
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%        # Max 2-3 extra pods during update (10 + 2.5 = 12-13)
      maxUnavailable: 25%  # Max 2-3 pods can be unavailable (10 - 2.5 = 7-8)
  template:
    spec:
      containers:
      - name: api
        image: api:v2.0
        readinessProbe:  # ← Critical for safe rollouts
          httpGet:
            path: /ready
            port: 8000
          periodSeconds: 5
          failureThreshold: 3
```

**Update process**:
```bash
# Update image
kubectl set image deployment/api api=api:v2.0

# Or apply new manifest
kubectl apply -f deployment.yaml

# Watch rollout
kubectl rollout status deployment/api

# Rollout process:
# 1. Create 2-3 new pods (v2.0) - maxSurge
# 2. Wait for readiness probes to pass
# 3. Terminate 2-3 old pods (v1.9) - maxUnavailable
# 4. Repeat until all pods updated
# 5. Terminate extra surge pods
```

**Rollback**:
```bash
# View rollout history
kubectl rollout history deployment/api

# Rollback to previous version
kubectl rollout undo deployment/api

# Rollback to specific revision
kubectl rollout undo deployment/api --to-revision=3

# Pause rollout (if issues detected)
kubectl rollout pause deployment/api

# Resume after fixing
kubectl rollout resume deployment/api
```

**Blue-Green Deployment** (zero downtime):
```yaml
# Blue deployment (current)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-blue
spec:
  replicas: 10
  selector:
    matchLabels:
      app: api
      version: blue
  template:
    metadata:
      labels:
        app: api
        version: blue

---
# Green deployment (new version)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-green
spec:
  replicas: 10
  selector:
    matchLabels:
      app: api
      version: green
  template:
    metadata:
      labels:
        app: api
        version: green

---
# Service (switch by updating selector)
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  selector:
    app: api
    version: blue  # ← Change to 'green' to switch traffic
```

**Canary Deployment** (gradual rollout):
```yaml
# Stable deployment (90%)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-stable
spec:
  replicas: 9
  selector:
    matchLabels:
      app: api
      track: stable

---
# Canary deployment (10%)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-canary
spec:
  replicas: 1
  selector:
    matchLabels:
      app: api
      track: canary

---
# Service (routes to both based on replica count)
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  selector:
    app: api  # Matches both stable and canary
```

**Automated rollback with health checks**:
```yaml
spec:
  progressDeadlineSeconds: 600  # Fail rollout after 10 min
  minReadySeconds: 30  # Wait 30s after ready before considering healthy
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0  # Zero downtime
  template:
    spec:
      containers:
      - name: api
        readinessProbe:
          httpGet:
            path: /ready
            port: 8000
          periodSeconds: 5
          failureThreshold: 3  # Fail after 15s unhealthy
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8000
          periodSeconds: 10
          failureThreshold: 3  # Restart after 30s unhealthy
```

**Best practices**:
- Always use readiness probes (prevent traffic to unhealthy pods)
- Set appropriate maxSurge/maxUnavailable (balance speed vs safety)
- Monitor metrics during rollout (error rates, latency)
- Use progressive delivery (canary → 50% → 100%)
- Automate rollback triggers (error rate > 5%, latency > 1s)

### Q12: What's the difference between ConfigMaps and Secrets? How would you manage sensitive configuration in Kubernetes?

**Answer**: ConfigMaps store non-sensitive configuration, Secrets store sensitive data (passwords, API keys). Both inject config into pods, but Secrets have additional security features.

**ConfigMap** (non-sensitive):
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  database.host: "postgres.default.svc.cluster.local"
  database.port: "5432"
  log.level: "info"
  feature.flags: |
    {
      "new_ui": true,
      "beta_features": false
    }
```

**Secret** (sensitive):
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
type: Opaque
data:
  database.username: aW5kZXhlcg==  # base64 encoded
  database.password: c2VjcmV0cGFzcw==
  api.key: YWJjMTIzNDU2Nzg5MA==
```

**Using in pods**:
```yaml
spec:
  containers:
  - name: api
    env:
    # From ConfigMap
    - name: DB_HOST
      valueFrom:
        configMapKeyRef:
          name: app-config
          key: database.host
    
    # From Secret
    - name: DB_PASSWORD
      valueFrom:
        secretKeyRef:
          name: app-secrets
          key: database.password
    
    # Mount entire ConfigMap as files
    volumeMounts:
    - name: config
      mountPath: /etc/config
      readOnly: true
    
    # Mount entire Secret as files
    - name: secrets
      mountPath: /etc/secrets
      readOnly: true
  
  volumes:
  - name: config
    configMap:
      name: app-config
  - name: secrets
    secret:
      secretName: app-secrets
      defaultMode: 0400  # Read-only for owner
```

**Key differences**:

| Feature | ConfigMap | Secret |
|---------|-----------|--------|
| Purpose | Non-sensitive config | Sensitive data |
| Storage | Plain text in etcd | Base64 encoded (not encrypted by default) |
| Size limit | 1 MiB | 1 MiB |
| Mount permissions | Default (0644) | Restrictive (0400) |
| Updates | Automatic (may take 60s) | Automatic (may take 60s) |
| Encryption at rest | No (unless etcd encrypted) | Possible with encryption provider |

**Production secret management**:

**1. External Secrets Operator** (recommended):
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: app-secrets
spec:
  secretStoreRef:
    name: aws-secrets-manager  # Or Vault, GCP Secret Manager
  target:
    name: app-secrets
  data:
  - secretKey: database_password
    remoteRef:
      key: prod/indexer/db-password
```

**2. Sealed Secrets** (GitOps-friendly):
```bash
# Encrypt secret (can be committed to Git)
kubeseal --format yaml < secret.yaml > sealed-secret.yaml

# Controller decrypts in cluster
kubectl apply -f sealed-secret.yaml
```

**3. Vault Integration**:
```yaml
annotations:
  vault.hashicorp.com/agent-inject: "true"
  vault.hashicorp.com/role: "indexer"
  vault.hashicorp.com/agent-inject-secret-db: "secret/data/indexer/database"
```

**Best practices**:
- Never commit secrets to Git (use external secret management)
- Enable etcd encryption at rest
- Use RBAC to restrict secret access
- Rotate secrets regularly (automate with External Secrets)
- Use separate secrets per environment (dev/staging/prod)
- Mount secrets as volumes (not env vars - visible in `docker inspect`)
- Use secret scanning tools (git-secrets, trufflehog)

**Our indexer setup**:
```yaml
# ConfigMap: Non-sensitive
apiVersion: v1
kind: ConfigMap
metadata:
  name: indexer-config
data:
  chains.json: |
    {
      "ethereum": {"chain_id": 1, "rpc": "https://eth.llamarpc.com"},
      "polygon": {"chain_id": 137, "rpc": "https://polygon.llamarpc.com"}
    }
  features.yaml: |
    enable_events: true
    enable_traces: false

---
# Secret: Sensitive (from AWS Secrets Manager via External Secrets)
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: indexer-secrets
spec:
  target:
    name: indexer-secrets
  data:
  - secretKey: database_url
    remoteRef:
      key: prod/indexer/database-url
  - secretKey: alchemy_api_key
    remoteRef:
      key: prod/indexer/alchemy-key
```

---

## Related Documentation

- [Database Fundamentals](./Database-Fundamentals.md) - Database theory and design patterns
- [PostgreSQL Database](./PostgreSQL-Database.md) - PostgreSQL-specific features and optimization
- [Go Programming](./Go-Programming.md) - Go language patterns for containerized apps
- [Message Queues](./Message-Queues.md) - Kafka and Redis for distributed systems
- [System Design & Architecture](./System-Design-Architecture.md) - Architecture decisions and scaling patterns
- [Deployment & Production](./Deployment-Production.md) - CI/CD, security, monitoring
- [Setup & Troubleshooting](./Setup-Troubleshooting.md) - Local setup and debugging

---

**Last Updated**: November 27, 2025
