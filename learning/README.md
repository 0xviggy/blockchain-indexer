# Blockchain Indexer Learning Guide

> ⚠️ **EDUCATIONAL MATERIAL** - Interview prep & senior/lead role preparation  
> These guides are for learning, NOT project-specific documentation.

Comprehensive learning materials, interview preparation, and technical deep-dives for blockchain indexer architecture and implementation.

> **Project Design Decisions**: See [../DESIGN_DECISIONS.md](../DESIGN_DECISIONS.md) for actual architectural choices in this project.

---

## 📚 Core Technical Guides

| Guide | Lines | Focus |
|-------|-------|-------|
| [System Design & Architecture](./System-Design-Architecture.md) ⭐ | ~4,400 | Comprehensive architecture patterns, reorgs, scaling |
| [Go Programming](./Go-Programming.md) ⭐ | ~1,600 | Concurrency, modules, testing patterns |
| [Docker & Kubernetes](./Docker-Kubernetes.md) ⭐ | ~1,900 | Containerization, orchestration |
| [Deployment & Production](./Deployment-Production.md) ⭐ | ~1,400 | CI/CD, hosting, monitoring |
| [Message Queues](./Message-Queues.md) ⭐ | ~1,300 | Kafka, Redis patterns (educational) |
| [Database Fundamentals](./Database-Fundamentals.md) ⭐ | ~1,200 | ACID, normalization, CAP theorem |
| [Frontend Development](./Frontend-Development.md) ⭐ | ~950 | React, Web3, cross-stack skills |
| [PostgreSQL Database](./PostgreSQL-Database.md) ⭐ | ~870 | PostgreSQL-specific features |
| [Troubleshooting](./Setup-Troubleshooting.md) ⭐ | ~760 | Debugging, performance, case studies |
| [MEV Analysis](./MEV_ANALYSIS.md) 🔬 | ~500 | MEV detection research & strategies |

**Total**: ~15,000 lines of educational content

---

### Quick Guide Summaries

**[System Design & Architecture](./System-Design-Architecture.md)** - The big one!
- Indexer architecture patterns, reorg handling, rate limiting
- Technology trade-offs (Go vs Rust, PostgreSQL vs alternatives)
- API design patterns, Kafka message ordering
- **Interview Q&A**: System design scenarios, trade-offs

**[Go Programming](./Go-Programming.md)**
- Go modules, concurrency (Goroutines, Channels, Context)
- Testing strategies and idiomatic patterns
- **Interview Q&A**: Concurrency, memory management

**[Database Fundamentals](./Database-Fundamentals.md)** + **[PostgreSQL](./PostgreSQL-Database.md)**
- Theory: ACID, normalization, CAP theorem, indexing theory
- Practice: PostgreSQL partitioning, query optimization
- **Interview Q&A**: SQL optimization, scaling strategies

**[Message Queues](./Message-Queues.md)**
- Kafka producer/consumer patterns
- Redis caching strategies
- *Note: Kafka NOT currently used in this project*

**[Docker & Kubernetes](./Docker-Kubernetes.md)**
- Dockerfile best practices, multi-stage builds
- Kubernetes deployment patterns
- **Interview Q&A**: Container lifecycle

**[Troubleshooting](./Setup-Troubleshooting.md)**
- Real-world debugging case studies
- Performance optimization patterns
- **Interview Q&A**: Debugging scenarios

---

## 🎯 Learning Paths

### For Backend Engineers
1. [System Design & Architecture](./System-Design-Architecture.md) - Start here
2. [Go Programming](./Go-Programming.md) - Language deep-dive
3. [Database Fundamentals](./Database-Fundamentals.md) + [PostgreSQL](./PostgreSQL-Database.md)
4. [Docker & Kubernetes](./Docker-Kubernetes.md)
5. [Deployment & Production](./Deployment-Production.md)

### For Full-Stack Engineers
1. [Troubleshooting](./Setup-Troubleshooting.md) - Get running quickly
2. [System Design](./System-Design-Architecture.md) - Understand architecture
3. [Frontend Development](./Frontend-Development.md) - React/Web3 patterns
4. [Deployment & Production](./Deployment-Production.md)

### For Interview Preparation
Each guide contains a dedicated **Interview Questions** section:
- **System Design**: [System-Design-Architecture.md](./System-Design-Architecture.md#interview-questions)
- **Go Concurrency**: [Go-Programming.md](./Go-Programming.md#interview-questions)
- **Databases**: [Database-Fundamentals.md](./Database-Fundamentals.md#interview-questions) + [PostgreSQL](./PostgreSQL-Database.md#interview-questions--answers)
- **DevOps**: [Docker-Kubernetes.md](./Docker-Kubernetes.md#interview-questions--answers)
- **Debugging**: [Troubleshooting](./Setup-Troubleshooting.md#interview-questions--answers)

---

## 🔗 Related Project Documentation

| Document | Purpose |
|----------|---------|
| [../docs/DESIGN_DECISIONS.md](../docs/DESIGN_DECISIONS.md) | Actual project architecture choices |
| [../docs/setup/SANDBOX_SETUP.md](../docs/setup/SANDBOX_SETUP.md) | Developer setup guide |
| [../docs/setup/DATABASE_GUIDE.md](../docs/setup/DATABASE_GUIDE.md) | Database setup & migrations |
| [../docs/PROGRESS_TRACKING.md](../docs/PROGRESS_TRACKING.md) | Project status |
| [../docs/TECHNICAL_SPEC.md](../docs/TECHNICAL_SPEC.md) | Implementation details |

---

**Last Updated**: December 20, 2025  
**Status**: ✅ 9 consolidated guides (~14,400 lines of educational content)
