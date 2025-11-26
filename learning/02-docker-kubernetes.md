# Docker & Kubernetes Guide

> **Purpose**: Understanding containerization, orchestration, and deployment patterns for blockchain infrastructure.

---

## Docker Fundamentals

### What is Docker?

Docker is a platform for building, shipping, and running applications in containers. A container packages an application with all its dependencies into a standardized unit.

**Key Concepts**:
- **Image**: Blueprint for containers (read-only)
- **Container**: Running instance of an image (writable layer)
- **Dockerfile**: Instructions to build an image
- **Registry**: Storage for images (Docker Hub, GitHub Container Registry)

### Multi-Stage Builds

**Problem**: Development images are huge (800MB+ with SDK, tools, cache)  
**Solution**: Build in one image, copy artifacts to minimal runtime image

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
- **Security**: No build tools in production image
- **Speed**: Smaller images = faster deploys
- **Layers**: Only runtime layer changes on code updates

**Best Practices**:
```dockerfile
# Use specific versions (not :latest)
FROM golang:1.21.5-alpine AS builder

# Order layers by change frequency (least → most)
COPY go.mod go.sum ./        # Changes rarely
RUN go mod download           # Cached unless go.mod changes
COPY . .                      # Changes often
RUN go build ./...            # Only rebuilds if source changes

# Use .dockerignore to exclude unnecessary files
# .dockerignore:
# .git
# *.md
# tests/
# .env.local
```

### CMD vs ENTRYPOINT

**CMD**: Default command, can be overridden  
**ENTRYPOINT**: Always runs, CMD becomes arguments

```dockerfile
# CMD only (can override entire command)
CMD ["./service", "--config", "prod.yaml"]
# Run: docker run myimage ./other-command  ✅ Replaces CMD

# ENTRYPOINT only (always runs this)
ENTRYPOINT ["./service"]
# Run: docker run myimage  ✅ Runs ./service
# Run: docker run myimage --debug  ✅ Runs ./service --debug

# Both (flexible configuration)
ENTRYPOINT ["./service"]
CMD ["--config", "prod.yaml"]
# Run: docker run myimage  → ./service --config prod.yaml
# Run: docker run myimage --debug  → ./service --debug
```

**When to Use**:
- **CMD**: Provide default arguments that users might override
- **ENTRYPOINT**: Ensure a specific command always runs
- **Both**: Command in ENTRYPOINT, config in CMD

---

## Docker Networking

### Network Types

1. **Bridge** (default): Isolated network with port mapping
2. **Host**: Share host network stack (no isolation, best performance)
3. **None**: No networking (maximum security)
4. **Overlay**: Multi-host networking (Swarm/Kubernetes)

```yaml
# Docker Compose network configuration
services:
  api:
    networks:
      - frontend  # Accessible from outside
      - backend   # Internal only
  
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

### How Container Communication Works

```bash
# Docker Compose creates DNS entries for service names
# Inside api container:
curl http://postgres:5432  # ✅ DNS resolves to postgres container IP

# Docker Compose also creates network alias
# Can use service name or container name
curl http://indexer-postgres:5432  # ✅ Also works
```

### Port Mapping

```yaml
services:
  api:
    ports:
      - "8000:8000"  # host:container
      - "127.0.0.1:8001:8001"  # Only localhost can access
      - "8002"  # Random host port → container 8002
```

### Network Inspection

```bash
# List networks
docker network ls

# Inspect network
docker network inspect indexer-network

# See which containers are on network
docker network inspect indexer-network | jq '.[0].Containers'

# Connect running container to network
docker network connect indexer-network my-container
```

---

## Docker Volumes

### Volume Types

1. **Named Volumes**: Managed by Docker, best for production
2. **Bind Mounts**: Mount host directory, best for development
3. **tmpfs Mounts**: In-memory, best for sensitive temp data

```yaml
services:
  postgres:
    volumes:
      # Named volume (managed by Docker)
      - postgres_data:/var/lib/postgresql/data
      
      # Bind mount (host directory)
      - ./backups:/backups
      
      # tmpfs (in-memory, not persisted)
      - type: tmpfs
        target: /tmp

volumes:
  postgres_data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /mnt/data/postgres  # Optional: specific location
```

### Named Volumes vs Bind Mounts

| Feature | Named Volumes | Bind Mounts |
|---------|--------------|-------------|
| Management | Docker manages | You manage |
| Location | `/var/lib/docker/volumes/` | Anywhere on host |
| Performance | Optimized by Docker | Direct filesystem |
| Backups | Use `docker cp` | Regular file backups |
| Use Case | Production databases | Development code |

### Volume Operations

```bash
# Create volume
docker volume create postgres_data

# List volumes
docker volume ls

# Inspect volume (see mount point)
docker volume inspect postgres_data

# Backup volume
docker run --rm -v postgres_data:/data -v $(pwd):/backup \
    alpine tar czf /backup/postgres_backup.tar.gz /data

# Restore volume
docker run --rm -v postgres_data:/data -v $(pwd):/backup \
    alpine tar xzf /backup/postgres_backup.tar.gz -C /

# Remove volume (after stopping containers)
docker volume rm postgres_data

# Remove all unused volumes
docker volume prune
```

### Volume Performance Tips

```yaml
# For macOS/Windows: Use delegated/cached for better performance
services:
  app:
    volumes:
      - ./src:/app/src:delegated  # Host → Container (writes delayed)
      - ./cache:/app/cache:cached  # Container → Host (reads cached)
```

---

## Debugging Containers

### View Logs

```bash
# Follow logs
docker logs -f container_name

# Last 100 lines
docker logs --tail 100 container_name

# Logs since timestamp
docker logs --since 2024-11-16T10:00:00 container_name

# Logs for specific service in Compose
docker compose logs -f postgres
```

### Inspect Container

```bash
# Full container details
docker inspect container_name

# Get specific field
docker inspect container_name | jq '.[0].State.Status'
docker inspect container_name | jq '.[0].NetworkSettings.IPAddress'

# See mounts
docker inspect container_name | jq '.[0].Mounts'

# See environment variables
docker inspect container_name | jq '.[0].Config.Env'
```

### Execute Commands in Running Container

```bash
# Open shell
docker exec -it container_name /bin/sh
docker exec -it container_name /bin/bash  # If bash available

# Run single command
docker exec container_name ls /var/lib/postgresql/data

# Run as different user
docker exec -u postgres container_name psql -U indexer
```

### Debug Crashed Container

```bash
# View logs even after exit
docker logs container_name

# Check exit code
docker inspect container_name | jq '.[0].State.ExitCode'
# 0 = success, 1 = app error, 137 = killed (OOM), 143 = SIGTERM

# Start container with shell override (debug startup issues)
docker run -it --entrypoint /bin/sh image_name

# Copy files from stopped container
docker cp container_name:/app/logs ./logs
```

### Resource Monitoring

```bash
# Real-time stats
docker stats

# Stats for specific container
docker stats container_name

# One-time snapshot
docker stats --no-stream

# Check disk usage
docker system df

# Detailed disk usage
docker system df -v
```

---

## Health Checks

**Why Health Checks**: Container might be running but application is unhealthy (e.g., database accepting connections but locked up)

```yaml
services:
  postgres:
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U indexer"]
      interval: 10s        # Check every 10 seconds
      timeout: 5s          # Timeout after 5 seconds
      retries: 3           # Fail after 3 consecutive failures
      start_period: 30s    # Grace period on startup
  
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
- **starting**: During start_period
- **healthy**: Check passed
- **unhealthy**: Check failed retries times

**Check Health**:
```bash
# See health status
docker ps

# Detailed health info
docker inspect container_name | jq '.[0].State.Health'

# Auto-restart unhealthy containers
docker run --restart=unless-stopped --health-cmd="curl -f http://localhost" ...
```

---

## Kubernetes Fundamentals

### Translating Docker Compose to Kubernetes

**Our Docker Compose**:
```yaml
services:
  postgres:
    image: postgres:15-alpine
    ports: ["5432:5432"]
    volumes: [postgres_data:/var/lib/postgresql/data]
```

**Equivalent Kubernetes**:
```yaml
# Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc

---
# Service (networking)
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  ports:
  - port: 5432
  selector:
    app: postgres

---
# PersistentVolumeClaim (storage)
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 20Gi
```

### Kubernetes Interview Questions

**Q: What's the difference between Deployment and StatefulSet?**
- **Deployment**: Stateless apps, pods are interchangeable, random names
- **StatefulSet**: Stateful apps (databases), stable network identities (postgres-0, postgres-1), ordered deployment
- **Use StatefulSet for**: Databases, Kafka, ZooKeeper

**Q: How does Kubernetes service discovery work?**
- A: Kubernetes DNS creates records for Services. `postgres.default.svc.cluster.local` resolves to Service IP. Pods can use short name `postgres` within same namespace.

**Q: What's a PersistentVolume vs PersistentVolumeClaim?**
- **PV**: Actual storage provisioned by admin (like a physical disk)
- **PVC**: Request for storage by application (like a reservation)
- **StorageClass**: Dynamic provisioner (auto-creates PVs from cloud provider)

---

## Scaling Patterns

### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-scaler
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
        averageUtilization: 70
```

### Scaling Interview Questions

**Q: How would you scale the ingester service?**

**A:** Partition chains across pods (Pod 1: ETH+Polygon, Pod 2: Arbitrum+Base), use Kubernetes Jobs for catch-up mode, scale vertically for real-time ingestion.

---

**Related Documents**:
- [Technology Stack](./01-technology-stack.md)
- [Database & Messaging Patterns](./03-databases-messaging.md)
- [Setup Guide](./05-setup-quickstart.md)
