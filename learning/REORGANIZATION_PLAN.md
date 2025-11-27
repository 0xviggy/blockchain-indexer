# Learning Materials Reorganization Plan

**Status**: In Progress  
**Date**: November 27, 2025

---

## Problem Statement

Current learning materials are scattered across multiple files with overlapping content:
- Database knowledge split between 03-databases-messaging.md and scattered in other files
- Go interview prep separated from Go programming guide
- Generic interview prep file separate from topic-specific files
- No dedicated deployment/production guide

---

## New Structure

### Core Technical Guides (Topic-Based)

1. **PostgreSQL-Database.md** ✅ CREATED
   - Merge: 03-databases-messaging.md (PostgreSQL sections)
   - Add: Database interview Q&A from 07-interview-prep.md
   - Add: Migration management (golang-migrate)
   - Add: Production deployment (Supabase, RDS)
   - Add: Backup, recovery, monitoring

2. **Go-Programming.md** 📋 TODO
   - Merge: 04-go-programming.md + 08-go-concepts-interview.md
   - Consolidate all Go content + interview questions
   - Sections: Basics, Concurrency, Testing, Best Practices, Interview Q&A

3. **Message-Queues.md** 📋 TODO
   - Split from: 03-databases-messaging.md (Kafka/Redis sections)
   - Add: Kafka patterns, Redis caching strategies
   - Add: Interview Q&A about message queues

4. **Docker-Kubernetes.md** ✅ KEEP AS-IS
   - Already well-organized in 02-docker-kubernetes.md
   - Maybe add: Interview Q&A section at end

5. **Frontend-Development.md** ✅ KEEP AS-IS
   - Already well-organized in 10-frontend-guide.md

6. **Deployment-Production.md** 📋 TODO
   - Merge: 11-cross-stack-production.md
   - Add: Content from docs/DEPLOYMENT.md
   - Add: Security best practices
   - Add: CI/CD pipelines
   - Add: Monitoring & alerting
   - Add: Cost optimization

7. **System-Design-Architecture.md** 📋 TODO
   - Merge: 06-implementation-concepts.md + 07-interview-prep.md
   - Focus: High-level system design
   - Sections: Reorg handling, rate limiting, design decisions
   - Interview Q&A integrated throughout

8. **Setup-Troubleshooting.md** 📋 TODO
   - Merge: 05-setup-quickstart.md + 09-troubleshooting.md
   - Getting started guide + common issues

---

## Migration Steps

### Phase 1: Create New Consolidated Files (Week 1)

- [x] PostgreSQL-Database.md
- [ ] Go-Programming.md
- [ ] Message-Queues.md
- [ ] Deployment-Production.md
- [ ] System-Design-Architecture.md
- [ ] Setup-Troubleshooting.md

### Phase 2: Update README.md (Week 1)

- [ ] Rewrite learning/README.md with new structure
- [ ] Add clear navigation
- [ ] Link to relevant docs/ files

### Phase 3: Archive Old Files (Week 2)

- [ ] Move old files to learning/archive/
- [ ] Update any references in docs/

### Phase 4: Validate (Week 2)

- [ ] Check all internal links work
- [ ] Verify no duplicate content
- [ ] Test readability

---

## Principles

1. **One Topic, One File**: All content about PostgreSQL in PostgreSQL-Database.md
2. **Interview Q&A Integrated**: At end of each topic file, not separate
3. **Production-Ready**: Include deployment/security in technical guides
4. **Progressive Complexity**: Basics → Advanced → Production → Interview

---

## File Mapping

| Old Files | New File | Status |
|-----------|----------|--------|
| 03-databases-messaging.md (PostgreSQL) | PostgreSQL-Database.md | ✅ Done |
| 07-interview-prep.md (DB questions) | PostgreSQL-Database.md | ✅ Done |
| 04-go-programming.md | Go-Programming.md | 📋 TODO |
| 08-go-concepts-interview.md | Go-Programming.md | 📋 TODO |
| 03-databases-messaging.md (Kafka/Redis) | Message-Queues.md | 📋 TODO |
| 11-cross-stack-production.md | Deployment-Production.md | 📋 TODO |
| docs/DEPLOYMENT.md | Deployment-Production.md | 📋 TODO |
| 06-implementation-concepts.md | System-Design-Architecture.md | 📋 TODO |
| 07-interview-prep.md (system design) | System-Design-Architecture.md | 📋 TODO |
| 05-setup-quickstart.md | Setup-Troubleshooting.md | 📋 TODO |
| 09-troubleshooting.md | Setup-Troubleshooting.md | 📋 TODO |
| 02-docker-kubernetes.md | Docker-Kubernetes.md | ✅ Keep as-is |
| 10-frontend-guide.md | Frontend-Development.md | ✅ Keep as-is |
| 01-technology-stack.md | DEPRECATE | Move to README |

---

## Next Actions

1. ✅ Create PostgreSQL-Database.md (DONE)
2. Review and approve reorganization plan
3. Create remaining consolidated files
4. Update README.md with new structure
5. Archive old files
