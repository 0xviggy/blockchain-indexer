# Blockchain Explorer - Web Frontend

A modern, real-time blockchain explorer built with React, TypeScript, and Tailwind CSS. Visualizes multi-chain blockchain data from our indexer API.

## Tech Stack

- **Framework**: React 18 + Vite
- **Language**: TypeScript
- **Styling**: Tailwind CSS (utility-first)
- **Routing**: React Router v6
- **State Management**: Zustand (global state)
- **Data Fetching**: TanStack Query (React Query)
- **Icons**: Lucide React
- **Build Tool**: Vite (fast HMR)

## Project Structure

```
web/
├── src/
│   ├── components/          # Reusable UI components
│   │   ├── BlockCard.tsx    # TODO: Display block info
│   │   ├── TransactionRow.tsx  # TODO: Transaction list item
│   │   ├── ChainSwitcher.tsx   # TODO: Multi-chain selector
│   │   └── Layout.tsx       # TODO: App layout with navigation
│   ├── pages/               # Page components
│   │   ├── Home.tsx         # TODO: Latest blocks feed
│   │   ├── BlockDetail.tsx  # TODO: Single block view
│   │   └── TransactionDetail.tsx  # TODO: Single tx view
│   ├── lib/                 # Utilities
│   │   ├── api.ts           # ✅ API client (done)
│   │   └── utils.ts         # ✅ Helper functions (done)
│   ├── types/               # TypeScript types
│   │   └── api.ts           # ✅ API response types (done)
│   ├── App.tsx              # TODO: Root component with routing
│   └── main.tsx             # ✅ Entry point
├── tailwind.config.js       # ✅ Tailwind configuration
└── package.json             # ✅ Dependencies
```

## Prerequisites

- Node.js 18+ and npm
- Backend API running on http://localhost:8000
- Backend must have CORS enabled (already configured in API service)

## Quick Start

```bash
# 1. Install dependencies (already done)
npm install

# 2. Copy environment variables
cp .env.example .env

# 3. Start dev server
npm run dev

# 4. Open browser
# http://localhost:5173
```

## Implementation Guide

### Phase 1: Basic Setup ✅ (DONE)

**Completed:**
- ✅ Vite + React + TypeScript initialized
- ✅ Tailwind CSS configured with dark mode
- ✅ API client created (`src/lib/api.ts`)
- ✅ TypeScript types defined (`src/types/api.ts`)
- ✅ Utility functions (`src/lib/utils.ts`)

**What works now:**
- `api.getChains()` - Fetch all chains
- `api.getBlocks(chainId, limit)` - Fetch blocks
- `api.getTransaction(chainId, hash)` - Fetch transaction
- Helper functions: `formatHash()`, `formatWei()`, `formatTimestamp()`

---

### Phase 2: Layout & Routing (👉 START HERE)

**Goal**: Create app shell with navigation and routing between pages.

#### Step 1: Set up React Router

**Task**: Edit `src/App.tsx` - Replace entire file:

\`\`\`tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Layout } from './components/Layout'
import { Home } from './pages/Home'
import { BlockDetail } from './pages/BlockDetail'
import { TransactionDetail } from './pages/TransactionDetail'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="block/:chainId/:blockNumber" element={<BlockDetail />} />
          <Route path="tx/:chainId/:hash" element={<TransactionDetail />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
\`\`\`

#### Step 2: Create Layout Component

**Task**: Create `src/components/Layout.tsx`:

\`\`\`tsx
import { Outlet, Link } from 'react-router-dom'
import { Activity } from 'lucide-react'

export function Layout() {
  return (
    <div className="min-h-screen bg-background">
      {/* Navigation */}
      <nav className="border-b border-border bg-card">
        <div className="container mx-auto px-4 py-4 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2 text-xl font-bold">
            <Activity className="w-6 h-6 text-primary" />
            <span>Blockchain Explorer</span>
          </Link>
          
          {/* TODO: Add ChainSwitcher here later */}
        </div>
      </nav>

      {/* Main content */}
      <main className="container mx-auto px-4 py-8">
        <Outlet />
      </main>

      {/* Footer */}
      <footer className="border-t border-border mt-16 py-8 text-center text-muted-foreground">
        <p>Multi-chain Blockchain Indexer</p>
      </footer>
    </div>
  )
}
\`\`\`

#### Step 3: Create Placeholder Pages

**Task**: Create `src/pages/Home.tsx`:

\`\`\`tsx
export function Home() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">Latest Blocks</h1>
      <p className="text-muted-foreground">TODO: Fetch and display blocks</p>
    </div>
  )
}
\`\`\`

**Task**: Create `src/pages/BlockDetail.tsx`:

\`\`\`tsx
import { useParams } from 'react-router-dom'

export function BlockDetail() {
  const { chainId, blockNumber } = useParams()
  
  return (
    <div>
      <h1 className="text-3xl font-bold mb-4">Block #{blockNumber}</h1>
      <p className="text-muted-foreground">Chain ID: {chainId}</p>
      <p className="text-muted-foreground">TODO: Fetch block details</p>
    </div>
  )
}
\`\`\`

**Task**: Create `src/pages/TransactionDetail.tsx`:

\`\`\`tsx
import { useParams } from 'react-router-dom'

export function TransactionDetail() {
  const { chainId, hash } = useParams()
  
  return (
    <div>
      <h1 className="text-3xl font-bold mb-4">Transaction</h1>
      <p className="text-muted-foreground">Chain ID: {chainId}</p>
      <p className="text-muted-foreground">Hash: {hash}</p>
      <p className="text-muted-foreground">TODO: Fetch transaction details</p>
    </div>
  )
}
\`\`\`

**Test it:**
\`\`\`bash
npm run dev
# Visit http://localhost:5173
# You should see the layout with navigation
# Click around - routing should work (empty pages for now)
\`\`\`

---

### Phase 3: Data Fetching with React Query

**Goal**: Set up React Query to fetch data from API.

#### Step 1: Configure React Query Provider

**Task**: Edit `src/main.tsx` - Replace entire file:

\`\`\`tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import './index.css'
import App from './App.tsx'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 10000, // 10 seconds
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
)
\`\`\`

#### Step 2: Fetch Blocks on Home Page

**Task**: Update `src/pages/Home.tsx`:

\`\`\`tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export function Home() {
  const chainId = 1 // Ethereum for now (TODO: make dynamic)
  
  const { data: blocks, isLoading, error } = useQuery({
    queryKey: ['blocks', chainId],
    queryFn: () => api.getBlocks(chainId, 20),
    refetchInterval: 12000, // Refetch every 12 seconds (Ethereum block time)
  })

  if (isLoading) return <div>Loading blocks...</div>
  if (error) return <div>Error: {error.message}</div>

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">Latest Blocks</h1>
      
      <div className="space-y-4">
        {blocks?.map((block) => (
          <div key={block.number} className="border border-border rounded-lg p-4">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold text-lg">Block #{block.number}</h3>
                <p className="text-sm text-muted-foreground">
                  {new Date(block.timestamp * 1000).toLocaleString()}
                </p>
              </div>
              <div className="text-right">
                <p className="text-sm">
                  <span className="text-muted-foreground">Transactions:</span>{' '}
                  <span className="font-medium">{block.transaction_count}</span>
                </p>
                <p className="text-sm">
                  <span className="text-muted-foreground">Gas Used:</span>{' '}
                  <span className="font-medium">{Number(block.gas_used).toLocaleString()}</span>
                </p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
\`\`\`

**Test it:**
\`\`\`bash
# Make sure API is running:
# Terminal 1: cd services/api && go run main.go
# Terminal 2: cd web && npm run dev

# Visit http://localhost:5173
# You should see live blocks updating every 12 seconds!
\`\`\`

---

See full implementation guide in the [Learning Guide](../docs/LEARNING_GUIDE.md) for:
- Phase 4: Reusable Components (BlockCard, TransactionRow)
- Phase 5: Block Detail Page
- Phase 6: Chain Switcher (Multi-chain support with Zustand)
- Phase 7: Polish (Dark mode, loading states, error handling)

## Available Scripts

\`\`\`bash
# Development
npm run dev          # Start dev server (http://localhost:5173)

# Build
npm run build        # Build for production (outputs to dist/)
npm run preview      # Preview production build

# Lint
npm run lint         # Check code quality
\`\`\`

## API Integration

The frontend connects to the backend API running on `http://localhost:8000`.

**Available API Methods** (in `src/lib/api.ts`):
- `api.getChains()` - List all chains
- `api.getBlocks(chainId, limit)` - Get recent blocks
- `api.getBlock(chainId, blockNumber)` - Get specific block
- `api.getBlockTransactions(chainId, blockNumber)` - Get block's transactions
- `api.getTransaction(chainId, hash)` - Get specific transaction
- `api.getAddressTransactions(address, limit)` - Get address history

## Next Steps (After Basic UI)

**Advanced Features to Add:**
1. **Search** - Search blocks, transactions, addresses
2. **Charts** - Visualize block times, gas prices, transaction volume
3. **WebSocket** - Real-time updates instead of polling
4. **Infinite Scroll** - Load more blocks as user scrolls
5. **Address Page** - View all transactions for an address
6. **Contract Verification** - Show decoded contract calls
7. **Analytics Dashboard** - Chain comparison, statistics

## Deployment

\`\`\`bash
# Build production bundle
npm run build

# Deploy to Vercel (recommended)
npm install -g vercel
vercel

# Or deploy to Netlify, Cloudflare Pages, etc.
# Just point to the dist/ directory
\`\`\`

## Learning Resources

- **React Query Docs**: https://tanstack.com/query/latest/docs/react/overview
- **Tailwind CSS Docs**: https://tailwindcss.com/docs
- **React Router Docs**: https://reactrouter.com/en/main
- **Zustand Docs**: https://docs.pmnd.rs/zustand/getting-started/introduction

## Troubleshooting

**Problem**: API requests fail with CORS error  
**Solution**: Make sure API service has CORS enabled (already configured in our Go API)

**Problem**: Blocks not updating in real-time  
**Solution**: Check `refetchInterval` in React Query config (should be 12000ms)

**Problem**: Dark mode not working  
**Solution**: Check that `<html>` element has `class="dark"` applied

**Problem**: Build fails with TypeScript errors  
**Solution**: Run `npm run build` to see full error messages, fix type issues
