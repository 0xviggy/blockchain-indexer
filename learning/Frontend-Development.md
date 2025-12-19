# Frontend Development & Cross-Stack Skills

Comprehensive guide to frontend development for blockchain applications, covering frameworks, libraries, best practices, cross-stack transferable skills, and production readiness.

## Table of Contents
- [Frontend Landscape Overview](#frontend-landscape-overview)
- [Framework Comparison & Ecosystem](#framework-comparison--ecosystem)
- [Cross-Stack Skills & Framework Switching](#cross-stack-skills--framework-switching)
- [Styling Solutions](#styling-solutions)
- [State Management](#state-management)
- [Data Fetching & Web3 Integration](#data-fetching--web3-integration)
- [Performance Optimization](#performance-optimization)
- [Common React Pitfalls & Anti-Patterns](#common-react-pitfalls--anti-patterns)
- [Production Readiness & Deployment](#production-readiness--deployment)
- [Interview Questions](#interview-questions)
- [Additional Resources](#additional-resources)

---

## Frontend Landscape Overview

Building frontends for blockchain applications requires different considerations than traditional web apps. You're displaying real-time financial data, complex transaction histories, multi-chain support, and need to integrate with wallets.

### Key Differences from Traditional Web Apps

- 🔐 **Wallet Integration**: MetaMask, WalletConnect, Coinbase Wallet
- ⛓️ **Multi-chain Support**: Ethereum, Polygon, Arbitrum, etc.
- 💰 **Real-time Data**: Block updates every 12s, price feeds, pending txs
- 🔍 **Data Visualization**: Charts, graphs, transaction flows
- 🎨 **Consistent UX**: Dark mode standard, crypto-native design patterns

---

## Framework Comparison & Ecosystem

### Why React Dominates Web3/DeFi

**Market Share in Blockchain:**
- Uniswap: Next.js
- Aave: Next.js
- Compound: React
- OpenSea: Next.js
- Etherscan: React
- **~85% of top DeFi protocols use React/Next.js**

**Reasons:**
1. ✅ **Wagmi/Viem ecosystem**: Best Web3 libraries are React-first
2. ✅ **Hiring**: 10x more React devs than Vue/Svelte
3. ✅ **Component libraries**: ShadCN, Radix, Mantine built for React
4. ✅ **Vercel ecosystem**: Next.js has best DX + deployment
5. ✅ **Community**: Largest ecosystem, fastest bug fixes

### React (Vite) ⚡ **RECOMMENDED for Block Explorers**

**Best For**: Block explorers, dashboards, analytics tools, internal tools

**Pros:**
- ✅ **Fastest setup**: `npm create vite@latest` → running in 30 seconds
- ✅ **Maximum flexibility**: No opinions on routing, state management
- ✅ **Excellent DX**: Hot module reload (HMR), instant feedback
- ✅ **Smaller bundle**: No SSR overhead
- ✅ **Simple deployment**: Just static files → Vercel/Netlify/S3
- ✅ **Perfect for SPAs**: Block explorers are single-page apps

**Cons:**
- ❌ No built-in routing (need React Router)
- ❌ No SSR/SEO (fine for authenticated apps)
- ❌ Must handle data fetching yourself
- ❌ No API routes (need separate backend)

**When to Choose React + Vite:**
- Building a block explorer (like we are)
- Internal dashboards for traders
- Portfolio trackers
- Admin panels
- Any SPA where SEO doesn't matter

**Example Stack:**
```bash
Frontend: React 18 + Vite + TypeScript
Styling: Tailwind CSS
State: Zustand or Jotai (lighter than Redux)
Data: TanStack Query (React Query)
Charts: Recharts or Chart.js
Web3: Wagmi + viem
```

### Next.js 14 (App Router) 🚀 **Industry Leader**

**Best For**: Public-facing DeFi protocols, DEX frontends, landing pages, documentation

**Pros:**
- ✅ **SEO-optimized**: Server-side rendering (SSR) & static generation (SSG)
- ✅ **Performance**: Automatic code splitting, image optimization
- ✅ **API routes**: Built-in serverless functions
- ✅ **File-based routing**: No need for React Router
- ✅ **Middleware**: Auth, redirects, rewriting at edge
- ✅ **Production-ready**: Used by Uniswap, Aave, Compound

**Cons:**
- ❌ More complex: SSR concepts, hydration, client vs server components
- ❌ Slower local dev: Webpack rebuilds can be slow
- ❌ More opinionated: Must follow Next.js conventions
- ❌ Vercel lock-in: Best experience on Vercel (but works elsewhere)
- ❌ Overkill for simple apps: Block explorers don't need SSR

**When to Choose Next.js:**
- Public DEX/protocol frontend (Uniswap clone)
- Marketing site + app combo
- Need SEO for Google/Twitter previews
- Blog + documentation + app
- Multi-region edge deployment

**Key Next.js Concepts:**

**Server Components (Default):**
```tsx
// app/blocks/page.tsx
async function BlocksPage({ params }) {
  // Fetch on server, no loading state on client
  const blocks = await fetch(`${API_URL}/blocks`).then(r => r.json());
  return <BlockList blocks={blocks} />;
}
```

**Client Components (for interactivity):**
```tsx
'use client';
import { useAccount } from 'wagmi';

export function WalletButton() {
  const { address, connect } = useAccount();
  return <button onClick={connect}>Connect</button>;
}
```

### Vue 3 (Composition API) 🟢

**Best For**: Teams already using Vue, smaller projects, rapid prototyping

**Pros:**
- ✅ **Gentle learning curve**: Template syntax familiar from HTML
- ✅ **Great docs**: Vue documentation is excellent
- ✅ **Composition API**: Similar to React Hooks
- ✅ **Built-in state**: Pinia (official store)
- ✅ **Smaller bundle**: Vue 3 is very lightweight

**Cons:**
- ❌ Smaller ecosystem: Fewer Web3 libraries
- ❌ Less adoption: Most DeFi uses React
- ❌ Hiring challenge: Fewer Vue + blockchain devs
- ❌ Wagmi/Viem are React-first

---

## Cross-Stack Skills & Framework Switching

**The Truth About Framework Switching**: 80%+ of skills transfer between React, Next.js, Vue, Svelte, and Solid. The underlying concepts are universal.

**Time to Switch Frameworks**: 
- React → Next.js: **1-2 days** (same paradigm, just add SSR concepts)
- React → Vue 3: **1-2 weeks** (different syntax, same concepts)
- React → Svelte: **2-3 weeks** (different reactivity model)
- Next.js → Remix: **2-5 days** (similar SSR, different data loading)

**Why Skills Transfer:**
1. All modern frameworks use **component-based architecture**
2. All have **reactive state** (hooks, signals, or stores)
3. All support **TypeScript**
4. All use **npm ecosystem** (same build tools, libraries)
5. All solve **the same problems** (routing, data fetching, forms)

### Core Transferable Concepts

#### 1. Component Composition ✅ (100% Transferable)

**React:**
```tsx
function BlockCard({ block }: { block: Block }) {
  return <div>{block.number}</div>;
}
```

**Vue 3 (Composition API):**
```vue
<script setup lang="ts">
defineProps<{ block: Block }>();
</script>
<template>
  <div>{{ block.number }}</div>
</template>
```

**Svelte:**
```svelte
<script lang="ts">
  export let block: Block;
</script>
<div>{block.number}</div>
```

**Takeaway**: Once you understand component composition in one framework, you understand it in all. Only syntax changes.

#### 2. Reactive State ✅ (95% Transferable)

**Core Idea**: When state changes, UI updates automatically.

**React (useState):**
```tsx
function Counter() {
  const [count, setCount] = useState(0);
  return <button onClick={() => setCount(count + 1)}>{count}</button>;
}
```

**Vue 3 (ref):**
```vue
<script setup>
import { ref } from 'vue';
const count = ref(0);
</script>
<template>
  <button @click="count++">{{ count }}</button>
</template>
```

**Svelte:**
```svelte
<script>
  let count = 0;
</script>
<button on:click={() => count++}>{count}</button>
```

**Transferable Knowledge:**
- ✅ Concept of "state triggers re-render"
- ✅ Immutability patterns (don't mutate state directly)
- ✅ Derived state / computed values
- ✅ Effect hooks / watchers

#### 3. Side Effects / Lifecycle ✅ (90% Transferable)

**React (useEffect):**
```tsx
useEffect(() => {
  const ws = new WebSocket('ws://api/blocks');
  ws.onmessage = (e) => setBlocks(JSON.parse(e.data));
  return () => ws.close(); // Cleanup
}, []);
```

**Vue 3 (onMounted, onUnmounted):**
```vue
<script setup>
import { onMounted, onUnmounted } from 'vue';

let ws;
onMounted(() => {
  ws = new WebSocket('ws://api/blocks');
  ws.onmessage = (e) => blocks.value = JSON.parse(e.data);
});
onUnmounted(() => ws?.close());
</script>
```

**Svelte (onMount):**
```svelte
<script>
import { onMount } from 'svelte';

onMount(() => {
  const ws = new WebSocket('ws://api/blocks');
  ws.onmessage = (e) => blocks = JSON.parse(e.data);
  return () => ws.close();
});
</script>
```

**Transferable Knowledge:**
- ✅ When to run side effects
- ✅ Cleanup pattern
- ✅ Dependency tracking
- ✅ Avoiding memory leaks

#### 4. Data Fetching Patterns ✅ (100% Transferable)

**TanStack Query works across frameworks:**

```tsx
// Works in React, Vue, Svelte, Solid
import { useQuery } from '@tanstack/react-query'; // or vue-query, solid-query

const { data, isLoading } = useQuery({
  queryKey: ['blocks'],
  queryFn: fetchBlocks,
});
```

**Transferable Patterns:**
- ✅ Caching strategies
- ✅ Optimistic updates
- ✅ Stale-while-revalidate
- ✅ Infinite scrolling
- ✅ Prefetching

---

## Styling Solutions

### Tailwind CSS 🎨 ⭐ **RECOMMENDED**

**Why Tailwind Dominates Web3:**
- ✅ **Fastest development**: No context switching between HTML/CSS
- ✅ **Consistent design**: Pre-defined spacing, colors, shadows
- ✅ **Responsive built-in**: `md:flex lg:grid xl:gap-4`
- ✅ **Dark mode easy**: `dark:bg-gray-900 dark:text-white`
- ✅ **Purge unused**: Final bundle ~10KB
- ✅ **Component libraries**: ShadCN, DaisyUI, HeadlessUI

**Example:**
```tsx
<div className="
  flex items-center justify-between
  p-4 rounded-lg
  bg-white dark:bg-gray-800
  border border-gray-200 dark:border-gray-700
  hover:shadow-lg transition-shadow
">
  <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
    Block #{blockNumber}
  </h3>
  <span className="text-sm text-gray-500">
    {txCount} transactions
  </span>
</div>
```

**Component Libraries on Tailwind:**

| Library | Description | When to Use |
|---------|-------------|-------------|
| **ShadCN** | Copy/paste components | Full control, customization |
| **DaisyUI** | Pre-built components | Fast prototyping, themes |
| **Headless UI** | Unstyled primitives | Custom designs, accessibility |
| **Radix UI** | Unstyled primitives | Low-level control |
| **Mantine** | Full-featured | Admin panels, dashboards |

**ShadCN Usage:**
```bash
# Instead of npm install, copy source code
npx shadcn-ui@latest init
npx shadcn-ui@latest add button card dialog table
```

### Styled Components / Emotion 💅

**When to Use:**
- Team values CSS-in-JS
- Need dynamic theming
- Component-scoped styles critical

**Pros:**
- ✅ Co-located styles with components
- ✅ Dynamic styling with props
- ✅ Automatic critical CSS
- ✅ Theme provider built-in

**Cons:**
- ❌ Runtime cost (styles injected at runtime)
- ❌ Larger bundle size
- ❌ Slower than Tailwind
- ❌ Hydration issues with SSR

**Verdict**: Tailwind wins for blockchain apps. Faster development, smaller bundles, better DX.

---

## State Management

### React State Management Evolution

**Timeline:**
```
2015: Redux (boilerplate hell)
2019: Context API (prop drilling solved)
2020: Recoil (atoms, but complex)
2021: Zustand (perfect balance) ⭐
2022: Jotai (atomic state)
2023-2025: Zustand still winning
```

**Why Zustand Won:**
- 8 lines vs 50 lines for Redux
- No providers, reducers, actions
- Perfect for fast-moving startups

### Zustand 🐻 ⭐ **RECOMMENDED for Web3**

**Why Perfect for Blockchain Apps:**
- ✅ Minimal boilerplate: ~3 lines to create store
- ✅ No providers: Access anywhere
- ✅ DevTools: Time-travel debugging
- ✅ Middleware: Persist to localStorage
- ✅ Tiny: 1KB gzipped

**Example:**
```tsx
// store/useBlockStore.ts
import { create } from 'zustand';

interface BlockStore {
  chainId: number;
  setChainId: (id: number) => void;
  blocks: Block[];
  addBlock: (block: Block) => void;
}

export const useBlockStore = create<BlockStore>((set) => ({
  chainId: 1,
  setChainId: (id) => set({ chainId: id }),
  blocks: [],
  addBlock: (block) => set((state) => ({
    blocks: [block, ...state.blocks]
  })),
}));

// Usage in component
function ChainSwitcher() {
  const { chainId, setChainId } = useBlockStore();
  return <select onChange={e => setChainId(+e.target.value)} value={chainId}>
    <option value={1}>Ethereum</option>
    <option value={137}>Polygon</option>
  </select>;
}
```

**With Persistence:**
```tsx
import { persist } from 'zustand/middleware';

export const useBlockStore = create(
  persist(
    (set) => ({
      chainId: 1,
      setChainId: (id) => set({ chainId: id }),
    }),
    { name: 'block-store' } // localStorage key
  )
);
```

### State Management Comparison

| Tool | Use Case | Complexity | Bundle Size |
|------|----------|-----------|-------------|
| **Zustand** | 90% of apps | Low | 1KB |
| **Jotai** | Atomic state, granular updates | Medium | 2KB |
| **Redux Toolkit** | Legacy codebases, strict patterns | High | 10KB |
| **Context** | Theme, auth (rarely changes) | Low | 0KB |
| **React Query** | Server state only | Low | 10KB |

**Verdict**: Use **Zustand** for 90% of blockchain apps.

---

## Data Fetching & Web3 Integration

### TanStack Query (React Query) ⭐

**The Standard for Blockchain Apps:**

```tsx
import { useQuery } from '@tanstack/react-query';

function BlockList() {
  const { data: blocks, isLoading } = useQuery({
    queryKey: ['blocks', chainId],
    queryFn: () => fetch(`/api/v1/chains/${chainId}/blocks`).then(r => r.json()),
    refetchInterval: 12000, // Refetch every 12s (Ethereum block time)
  });

  if (isLoading) return <div>Loading...</div>;
  return <div>{blocks.map(block => <BlockCard key={block.number} {...block} />)}</div>;
}
```

**Why React Query?**
- ✅ **Automatic caching**: No manual cache management
- ✅ **Background refetching**: Keep data fresh
- ✅ **Optimistic updates**: Instant UI feedback
- ✅ **Infinite scrolling**: Built-in `useInfiniteQuery`
- ✅ **DevTools**: See all queries/cache

### Web3 Integration: Wagmi + RainbowKit

**The Standard Stack:**

```tsx
// providers.tsx
import { WagmiConfig, createConfig } from 'wagmi';
import { mainnet, polygon, arbitrum } from 'wagmi/chains';
import { RainbowKitProvider } from '@rainbow-me/rainbowkit';

const config = createConfig({
  chains: [mainnet, polygon, arbitrum],
  transports: {
    [mainnet.id]: http(),
    [polygon.id]: http(),
    [arbitrum.id]: http(),
  },
});

export function Providers({ children }) {
  return (
    <WagmiConfig config={config}>
      <RainbowKitProvider>
        {children}
      </RainbowKitProvider>
    </WagmiConfig>
  );
}

// WalletButton.tsx
import { ConnectButton } from '@rainbow-me/rainbowkit';

export function WalletButton() {
  return <ConnectButton />;
}

// Read blockchain data
import { useBalance, useContractRead } from 'wagmi';

function UserBalance() {
  const { address } = useAccount();
  const { data: balance } = useBalance({ address });
  return <div>{balance?.formatted} ETH</div>;
}
```

---

## Performance Optimization

### Code Splitting

```tsx
// Lazy load heavy pages
const TransactionDetail = lazy(() => import('./pages/TransactionDetail'));

function App() {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <TransactionDetail />
    </Suspense>
  );
}
```

### Virtual Scrolling (for long lists)

```tsx
import { useVirtualizer } from '@tanstack/react-virtual';

function TransactionList({ transactions }) {
  const virtualizer = useVirtualizer({
    count: transactions.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 60, // Height of each row
  });
  
  return (
    <div ref={parentRef} style={{ height: '600px', overflow: 'auto' }}>
      {virtualizer.getVirtualItems().map(virtualRow => (
        <div key={virtualRow.index}>
          <TransactionRow tx={transactions[virtualRow.index]} />
        </div>
      ))}
    </div>
  );
}
```

### Memoization

```tsx
import { memo } from 'react';

const BlockCard = memo(function BlockCard({ block }: { block: Block }) {
  // Only re-renders if block prop changes
  return <div>{block.number}</div>;
});
```

---

## Common React Pitfalls & Anti-Patterns

### Pitfall 1: State in Function Default Parameters (Stale Closures)

**Problem**: Using React state values in default function parameters creates stale closures that don't update when state changes.

```tsx
// ❌ ANTI-PATTERN: State in default parameter
const [limit, setLimit] = useState(500)

const loadData = async (id: number, limit = limit) => {
    // Problem: 'limit' captures the value at function definition time
    const data = await api.getData(id, limit)
    setData(data)
}

// User changes limit to 1000
setLimit(1000)  // State updates

// Later, auto-refresh calls loadData without limit parameter
useEffect(() => {
    setInterval(() => {
        loadData(selectedId)  // Uses OLD limit value (500)!
    }, 5000)
}, [selectedId])  // limit not in dependencies

// Result: Limit glitches back to 500 every 5 seconds
```

**Why This Happens**:
1. Function with default parameters is created: `limit = 500` (captured at mount)
2. User changes state: `setLimit(1000)`
3. Component re-renders, function re-defined with new default: `limit = 1000`
4. BUT useEffect has closure over OLD function definition
5. Auto-refresh uses old function with old default value

**Solution 1: Remove Default Parameters**
```tsx
// ✅ GOOD: Required parameter
const [limit, setLimit] = useState(500)

const loadData = async (id: number, limit: number) => {
    // limit is required - no stale closure possible
    const data = await api.getData(id, limit)
    setData(data)
}

// All call sites must pass limit explicitly
loadData(selectedId, limit)  // Always uses current state

useEffect(() => {
    setInterval(() => {
        loadData(selectedId, limit)  // Explicit limit
    }, 5000)
}, [selectedId, limit])  // Include limit in dependencies
```

**Key Lessons**:
1. **Avoid default parameters with React state** - They create stale closures
2. **Be explicit with function parameters** - Required params are clearer than defaults
3. **useEffect dependencies matter** - Include ALL values used inside
4. **ESLint exhaustive-deps rule** - Enable `react-hooks/exhaustive-deps` to catch this
5. **Test auto-refresh scenarios** - Manual clicks may work, intervals expose bugs

### Pitfall 2: Missing Dependencies in useEffect

```tsx
// ❌ BAD: count used but not in dependencies
const [count, setCount] = useState(0)

useEffect(() => {
    console.log(count)  // Uses count
}, [])  // Missing dependency!

// ✅ GOOD: Include all dependencies
useEffect(() => {
    console.log(count)
}, [count])
```

### Pitfall 3: Infinite Loops from Object/Array Dependencies

```tsx
// ❌ BAD: New array created every render
const [data, setData] = useState([])

useEffect(() => {
    fetchData()
}, [data])  // data is new array every time → infinite loop!

// ✅ GOOD: Use primitive value
useEffect(() => {
    fetchData()
}, [data.length])  // Or use React Query
```

### Pitfall 4: Forgetting to Cleanup Effects

```tsx
// ❌ BAD: Interval keeps running after unmount
useEffect(() => {
    setInterval(() => {
        fetchData()
    }, 5000)
}, [])

// ✅ GOOD: Return cleanup function
useEffect(() => {
    const interval = setInterval(() => {
        fetchData()
    }, 5000)
    return () => clearInterval(interval)  // Cleanup!
}, [])
```

### Pitfall 5: Not Using useCallback for Event Handlers

```tsx
// ❌ BAD: New function every render, child re-renders unnecessarily
function Parent() {
    const [count, setCount] = useState(0)
    
    const handleClick = () => {  // New function every render
        console.log('clicked')
    }
    
    return <ExpensiveChild onClick={handleClick} />
}

// ✅ GOOD: Stable function reference
function Parent() {
    const [count, setCount] = useState(0)
    
    const handleClick = useCallback(() => {
        console.log('clicked')
    }, [])  // Function never changes
    
    return <ExpensiveChild onClick={handleClick} />
}
```

### Pitfall 6: Derived State (When Not Needed)

```tsx
// ❌ BAD: Storing derived state separately
const [items, setItems] = useState([])
const [itemCount, setItemCount] = useState(0)  // Derived!

useEffect(() => {
    setItemCount(items.length)  // Sync required
}, [items])

// ✅ GOOD: Calculate during render
const [items, setItems] = useState([])
const itemCount = items.length  // No sync needed
```

### Pitfall 7: Not Using React Query for Server State

```tsx
// ❌ BAD: Manual state management for API data
const [data, setData] = useState(null)
const [loading, setLoading] = useState(true)
const [error, setError] = useState(null)

useEffect(() => {
    setLoading(true)
    fetch('/api/data')
        .then(res => res.json())
        .then(data => {
            setData(data)
            setLoading(false)
        })
        .catch(err => {
            setError(err)
            setLoading(false)
        })
}, [])

// ✅ GOOD: React Query handles it all
import { useQuery } from '@tanstack/react-query'

const { data, isLoading, error } = useQuery({
    queryKey: ['data'],
    queryFn: () => fetch('/api/data').then(res => res.json())
})
```

**Common React Patterns Summary**:

| Pattern | Bad Approach | Good Approach |
|---------|-------------|---------------|
| State in defaults | `func(x = stateVal)` | `func(x: Type)` + pass explicitly |
| Dependencies | Omit variables | Include all used values |
| Arrays/objects in deps | `[obj]` | `[obj.id]` or useMemo |
| Effect cleanup | No return | `return () => cleanup()` |
| Event handlers | Inline functions | useCallback |
| Derived state | Separate useState | Calculate in render |
| API data | Manual fetch | React Query/SWR |

**ESLint Rules to Enable**:
```json
{
  "rules": {
    "react-hooks/rules-of-hooks": "error",
    "react-hooks/exhaustive-deps": "warn",
    "react/jsx-no-bind": "warn"
  }
}
```

---

## Production Readiness & Deployment

### Deployment Checklist

**Infrastructure:**
- [ ] Database migrations tested in staging
- [ ] Environment variables documented
- [ ] Secrets in vault (not environment variables)
- [ ] TLS/SSL certificates installed
- [ ] Backup strategy defined
- [ ] Disaster recovery plan documented

**Observability:**
- [ ] Health check endpoints implemented
- [ ] Metrics exported to Prometheus
- [ ] Logs structured (JSON format)
- [ ] Distributed tracing enabled (Jaeger)
- [ ] Monitoring alerts configured
- [ ] On-call runbook created

**Security & Performance:**
- [ ] Rate limiting configured
- [ ] CORS configured for API
- [ ] Load testing completed
- [ ] Security audit performed

### Monitoring Alerts

```yaml
alerts:
  - name: IngesterLagging
    condition: last_indexed_block < chain_head_block - 100
    severity: warning
    
  - name: ProcessorConsumerLag
    condition: kafka_consumer_lag > 10000
    severity: critical
    
  - name: HighAPILatency
    condition: http_request_duration_p95 > 1s
    severity: warning
    
  - name: DatabaseConnectionPoolExhausted
    condition: db_connections_available < 5
    severity: critical
    
  - name: HighErrorRate
    condition: error_rate > 1%
    severity: warning
```

### Capacity Planning

**Current Setup** (per chain):
- Ingester: 2 vCPU, 4GB RAM → ~100 blocks/sec
- Processor: 4 vCPU, 8GB RAM → ~200 blocks/sec
- API: 4 vCPU, 8GB RAM → ~1000 req/sec
- PostgreSQL: 8 vCPU, 32GB RAM → ~10K writes/sec
- Kafka: 4 vCPU, 16GB RAM → ~100K msg/sec

**Scaling Strategy:**
- Horizontal: Add more ingester/processor/API instances
- Vertical: Increase database resources first
- Sharding: Partition database by chain_id when single DB hits limits

---

## Interview Questions

### Q: Why don't block explorers need SEO/SSR?

**A:** "Block explorers are authenticated tools used by developers and traders who bookmark them directly or search for '[chain name] explorer'. They're not content sites competing for organic traffic. The pages are highly dynamic (new blocks every 12s) making SSR cache invalidation complex. Client-side rendering with React + Vite is simpler, faster to build, and delivers better UX with instant updates. We save on infrastructure costs (no Node.js server) and can deploy to CDN as static files."

### Q: How do you handle real-time block updates?

**A:** "We use a hybrid approach:
1. **Polling** with React Query (`refetchInterval: 12000`) as the reliable baseline
2. **WebSocket** for instant updates when available
3. **Optimistic updates** - Add new block to UI immediately, reconcile with server later
4. **Background refetch** - Keep data fresh even when tab inactive

This ensures users see new blocks within 1-2 seconds while handling WebSocket disconnections gracefully."

### Q: How would you optimize rendering 10,000 transactions?

**A:** "Multiple strategies:
1. **Virtual scrolling** - Only render visible rows (TanStack Virtual)
2. **Pagination** - Show 20 per page with cursor-based pagination
3. **Infinite scroll** - Load more as user scrolls (React Query's `useInfiniteQuery`)
4. **Memoization** - `memo()` components that don't need to re-render
5. **Web Workers** - Offload heavy computation (parsing big numbers, hex decoding)
6. **Debounced search** - Don't query on every keystroke

For 10K items, virtual scrolling + pagination is the standard solution."

### Q: What is the "Stale Closure" problem in React hooks?

**A:** "It happens when a closure (like a function inside `useEffect` or `useCallback`) captures variables from a previous render and doesn't update when those variables change. This usually happens because of missing dependencies in the dependency array. For example, if you have a `setInterval` that logs a state variable `count`, but `count` isn't in the dependency array, the interval will always log the initial value of `count` because it's using the closure from the first render. The fix is to include all dependencies or use the functional update form of `setState`."

### Q: Why use Zustand over Redux for this project?

**A:** "Redux is powerful but requires significant boilerplate (actions, reducers, providers, thunks). For a block explorer, our global state is minimal (user preferences, theme, maybe current chain ID). Most of our 'state' is actually server data (blocks, txs), which is better handled by React Query. Zustand provides a tiny (1KB), hook-based store without providers, making it perfect for the small amount of client-side global state we actually have."

### Q: How long does it take to switch from React to Vue or Svelte?

**A:** "80%+ of skills transfer between frameworks because they all use component-based architecture, reactive state, and solve the same problems. React → Next.js takes 1-2 days (same paradigm + SSR). React → Vue 3 takes 1-2 weeks (different syntax, same concepts). React → Svelte takes 2-3 weeks (different reactivity model). The hard problems - state management, data fetching, performance optimization, Web3 integration - are framework-agnostic. Master React + TypeScript + Tailwind + React Query, and you can build 90% of Web3 frontends and switch to any framework in 1-3 weeks if needed."

---

## Additional Resources

### Official Documentation
- [React documentation](https://react.dev/)
- [Next.js documentation](https://nextjs.org/docs)
- [TanStack Query](https://tanstack.com/query/latest)
- [Wagmi](https://wagmi.sh/)
- [Tailwind CSS](https://tailwindcss.com/)
- [Zustand](https://zustand-demo.pmnd.rs/)

### Learning Resources
- [System Design Interview](https://www.amazon.com/System-Design-Interview-insiders-Second/dp/B08CMF2CQF)
- [Designing Data-Intensive Applications](https://dataintensive.net/)
- [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)

### Similar Projects
- [The Graph](https://thegraph.com/) - Decentralized indexing protocol
- [Etherscan](https://etherscan.io/) - Blockchain explorer
- [Dune Analytics](https://dune.com/) - Blockchain analytics platform

---

## Key Takeaway

**The Meta-Skill**: Understanding component-based architecture, reactive state, side effects, and data fetching patterns.

**Once you know React well:**
- Next.js = 2 days to learn (same paradigm + SSR)
- Vue 3 = 1-2 weeks (different syntax, same concepts)
- Svelte = 2-3 weeks (different reactivity)
- Solid = 1 week (similar to React, but signals)

**80% of your knowledge transfers.** The hard problems (state management, data fetching, performance optimization, Web3 integration) are framework-agnostic.

**Bottom Line**: Master React + TypeScript + Tailwind + React Query + Zustand → You can build 90% of Web3 frontends and switch to any other framework in 1-3 weeks if needed.
