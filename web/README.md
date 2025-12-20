# Blockchain Explorer - Web Frontend

A modern, immersive blockchain explorer built with React, TypeScript, and Tailwind CSS.

> 📖 **Complete documentation**: See [WEB_PROGRESS_TRACKING.md](./WEB_PROGRESS_TRACKING.md) for full implementation guide, architecture, and roadmap.

## Current Status

✅ **Development UI** (85% complete) - Functional single-page dashboard with:
- Real-time transaction monitoring
- Event log viewer
- Health monitoring
- Ingester controls
- Multi-chain support

🔄 **Next Up**: Immersive fullscreen UI refactor with dedicated canvas for each section

## Tech Stack

- **React 18** + TypeScript + Vite
- **Tailwind CSS** (dark mode)
- **TanStack Query** (data fetching)
- **Lucide React** (icons)

## Prerequisites

- Node.js 18+
- Backend API on http://localhost:8000

## Quick Start

```bash
# Install dependencies
npm install

# Start dev server
npm run dev

# Open http://localhost:5174
```

## Available Scripts

```bash
npm run dev       # Start development server
npm run build     # Build for production
npm run preview   # Preview production build
npm run lint      # Check code quality
```

## API Integration

Frontend connects to backend at `http://localhost:8000`.

**Key API methods** (see `src/lib/api.ts`):
```typescript
api.getChains()                           // All chains
api.getBlocks(chainId, limit)             // Recent blocks
api.getTransactions(chainId, limit)       // Recent transactions
api.getEvents(chainId, limit)             // Recent events
api.getHealth()                           // API health check
api.getIngesterStatus()                   // Ingester status
```

**Helper functions** (see `src/lib/utils.ts`):
```typescript
formatHash('0x1234...', 6, 4)  // Shorten addresses
formatWei('1000000000000000000') // Convert to ETH
formatTimestamp(1700000000)      // Human-readable date
formatNumber(1000000)            // Add commas
```

## Documentation

- **[WEB_PROGRESS_TRACKING.md](./WEB_PROGRESS_TRACKING.md)** - Complete guide: architecture, roadmap, implementation
- **[../PROGRESS_TRACKING.md](../PROGRESS_TRACKING.md)** - Overall project status
- **[../DATABASE_GUIDE.md](../DATABASE_GUIDE.md)** - Database setup

## Roadmap

**Current**: Development UI (85% complete)  
→ Single-page dashboard with tabs for transactions, events, health, ingester

**Next**: Immersive Fullscreen UI  
→ Fullscreen canvas design, each section feels like a dedicated workspace

**Future**: Production UI Branch  
→ Public-facing variant, remove dev tools, add search/analytics/refined settings

See [WEB_PROGRESS_TRACKING.md](./WEB_PROGRESS_TRACKING.md) for full details.

---

**Last Updated**: December 20, 2025  
**Status**: Functional dev UI, planning immersive refactor

