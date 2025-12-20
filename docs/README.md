# Blockchain Indexer Documentation

This folder contains **project documentation** for the blockchain indexer application.

---

## 📋 Documentation Index

### Project Documentation

| Document | Purpose | Audience |
|----------|---------|----------|
| [Business Specification](./BUSINESS_SPEC.md) | Product requirements, use cases, business context | Product, Business, Stakeholders |
| [Technical Specification](./TECHNICAL_SPEC.md) | Implementation details, API specs, service design | Engineers, Architects |
| [Project Template](./PROJECT_TEMPLATE.md) | Reusable project structure guide | Engineers starting new projects |

### Setup & Operations (docs/setup/)

| Document | Purpose |
|----------|---------|
| [Sandbox Setup](./setup/SANDBOX_SETUP.md) | Local development environment setup |
| [Database Guide](./setup/DATABASE_GUIDE.md) | Database operations, migrations, seeds |
| [Deployment Guide](./setup/DEPLOYMENT.md) | Production deployment, Supabase setup |

### Root-Level Documents

| Document | Purpose |
|----------|---------|
| [DESIGN_DECISIONS.md](./DESIGN_DECISIONS.md) | Architecture rationale, multi-chain strategy |
| [PROGRESS_TRACKING.md](./PROGRESS_TRACKING.md) | Project status and roadmap |

---

## 📚 Documentation vs Learning Materials

### Docs Folder (You Are Here)
**Purpose**: Project documentation for building and understanding the application

**Contains**:
- ✅ Business requirements and specifications
- ✅ Technical implementation details
- ✅ Setup and deployment guides
- ✅ Project structure template

### Learning Folder ([/learning](../learning/))
**Purpose**: Educational materials, interview prep, technical deep-dives

> ⚠️ Learning files are marked with an educational banner

**Contains**:
- ✅ Technology stack tutorials (Go, PostgreSQL, Docker)
- ✅ System design patterns
- ✅ MEV analysis research
- ✅ Interview preparation Q&A
- ✅ Troubleshooting guides

---

## 🔗 Quick Links

### For New Team Members
1. Start with [SANDBOX_SETUP](./setup/SANDBOX_SETUP.md) to get running
2. Read [DESIGN_DECISIONS](./DESIGN_DECISIONS.md) for architecture context
3. Check [PROGRESS_TRACKING](./PROGRESS_TRACKING.md) for current state
4. Visit [Learning Materials](../learning/) for deep-dives

### For Engineers
- **Architecture Rationale**: [DESIGN_DECISIONS.md](./DESIGN_DECISIONS.md)
- **Implementation Details**: [Technical Specification](./TECHNICAL_SPEC.md)
- **Database Operations**: [Database Guide](./setup/DATABASE_GUIDE.md)
- **Learning**: [Learning Folder](../learning/)

### For Product/Business
- **Overview**: [Business Specification](./BUSINESS_SPEC.md)
- **Roadmap**: [PROGRESS_TRACKING.md](./PROGRESS_TRACKING.md)

---

## 📁 Related Documentation

- **Learning Materials**: [../learning/](../learning/) - Tutorials, guides, interview prep
- **Database Migrations**: [../database/migrations/](../database/migrations/) - SQL schemas
- **Infrastructure**: [../infrastructure/](../infrastructure/) - Docker, config files
- **Service READMEs**: 
  - [Ingester](../services/ingester/README.md)
  - [API](../services/api/)
  - [Web](../web/README.md)

---

**Last Updated**: December 20, 2025  
**Maintained By**: Engineering Team
