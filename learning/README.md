# Blockchain Indexer Learning Guide

Comprehensive learning materials, interview preparation, and technical deep-dives for blockchain indexer architecture and implementation.

---

## 📚 Core Technical Guides

### 1. [System Design & Architecture](./System-Design-Architecture.md) ⭐
**High-Level Architecture & Design Decisions**
- Indexer architecture patterns (Microservices, Event-Driven)
- Handling blockchain reorgs and finality
- Rate limiting and data consistency strategies
- **Interview Q&A**: System design scenarios, trade-offs

### 2. [Go Programming](./Go-Programming.md) ⭐
**Language Fundamentals & Best Practices**
- Go modules, workspaces, and dependency management
- Concurrency patterns (Goroutines, Channels, Context)
- Testing strategies and style guides
- **Interview Q&A**: Concurrency, memory management, idiomatic Go

### 3. [PostgreSQL Database](./PostgreSQL-Database.md) ⭐
**Database Design & Optimization**
- Schema design for blockchain data
- Indexing strategies and partitioning
- Performance tuning and batch processing
- **Interview Q&A**: SQL optimization, ACID properties, scaling

### 4. [Message Queues (Kafka & Redis)](./Message-Queues.md) ⭐
**Event Streaming & Caching**
- Kafka producer/consumer patterns
- Redis caching strategies and data structures
- Handling backpressure and ordering
- **Interview Q&A**: Message delivery semantics, scaling queues

### 5. [Docker & Kubernetes](./Docker-Kubernetes.md) ⭐
**Containerization & Orchestration**
- Dockerfile best practices and multi-stage builds
- Kubernetes deployment patterns (StatefulSets, Deployments)
- Networking, volumes, and health checks
- **Interview Q&A**: Container lifecycle, orchestration concepts

### 6. [Frontend Development](./Frontend-Development.md) ⭐
**Modern Web3 Frontend Stack**
- React, Vite, and TypeScript setup
- State management (Zustand, React Query)
- Web3 integration (Wagmi, RainbowKit)
- Cross-stack skills and framework switching
- **Interview Q&A**: React lifecycle, performance, Web3 challenges

### 7. [Setup & Troubleshooting](./Setup-Troubleshooting.md) ⭐
**Getting Started & Operations**
- Quick start guide for local development
- Common issues and solutions
- Performance tuning and monitoring
- **Interview Q&A**: Debugging scenarios, operational excellence

### 8. [Deployment & Production](./Deployment-Production.md) ⭐
**Going to Production**
- Production readiness checklist
- CI/CD pipelines and security
- Monitoring, alerting, and cost optimization
- Infrastructure setup (Supabase, Railway, Vercel)

---

## 🎯 Learning Paths

### For Backend Engineers
1. Start with [System Design & Architecture](./System-Design-Architecture.md)
2. Deep dive into [Go Programming](./Go-Programming.md)
3. Master [PostgreSQL](./PostgreSQL-Database.md) and [Message Queues](./Message-Queues.md)
4. Learn [Docker & Kubernetes](./Docker-Kubernetes.md)
5. Review [Deployment & Production](./Deployment-Production.md)

### For Full-Stack Engineers
1. Begin with [Setup & Troubleshooting](./Setup-Troubleshooting.md) to get running
2. Understand [System Design](./System-Design-Architecture.md)
3. Focus on [Frontend Development](./Frontend-Development.md)
4. Review [Deployment & Production](./Deployment-Production.md)

### For Interview Preparation
Each guide contains a dedicated **Interview Questions** section at the end.
1. **System Design**: [System-Design-Architecture.md](./System-Design-Architecture.md)
2. **Go Concurrency**: [Go-Programming.md](./Go-Programming.md)
3. **Database Optimization**: [PostgreSQL-Database.md](./PostgreSQL-Database.md)
4. **DevOps**: [Docker-Kubernetes.md](./Docker-Kubernetes.md)
5. **Frontend**: [Frontend-Development.md](./Frontend-Development.md)

---

## 🔗 Related Documentation

- [Business Specification](../docs/BUSINESS_SPEC.md) - Product requirements
- [Technical Specification](../docs/TECHNICAL_SPEC.md) - System architecture
- [Development Status](../docs/DEVELOPMENT_STATUS.md) - Current progress
- [Deployment Guide](../docs/DEPLOYMENT.md) - Infrastructure details

---

## 📂 Archive

Old numbered files (01-11) have been consolidated into the topic-based guides above. See `archive/` folder for historical reference.

---

**Last Updated**: November 28, 2025  
**Status**: ✅ Reorganization complete - 8 consolidated guides (13,300 lines)
