# Multi-Chain Support Strategy

## Overview

This document defines which blockchain networks we support, prioritization, and implementation status.

---

## Chain Support Tiers

### **Tier 1: Priority Chains** (Implement First)
High liquidity, large user base, well-established ecosystems

| Chain | Chain ID | Status | Priority | Rationale |
|-------|----------|--------|----------|-----------|
| **Ethereum** | 1 | 🟢 Ready | P0 | Largest TVL, most protocols, essential |
| **Polygon** | 137 | 🟢 Ready | P0 | Low fees, high activity, Ethereum-compatible |
| **Arbitrum** | 42161 | 🟢 Ready | P0 | Major L2, growing ecosystem |
| **Optimism** | 10 | 🟢 Ready | P1 | OP Stack, Coinbase connection |
| **Base** | 8453 | 🟢 Ready | P1 | Coinbase L2, viral apps, growing fast |

**Target**: Launch with all Tier 1 chains  
**Estimated Effort**: 2-3 weeks for all 5 chains  
**Shared Infra**: Same codebase, just different RPC configs

---

### **Tier 2: High-Value Additions** (Next Phase)
Significant activity, easy to integrate (EVM-compatible)

| Chain | Chain ID | Status | Priority | Rationale |
|-------|----------|--------|----------|-----------|
| **BSC** | 56 | 🟡 Planned | P2 | High trading volume, DeFi activity |
| **Avalanche C-Chain** | 43114 | 🟡 Planned | P2 | Fast finality, subnets |
| **Gnosis** | 100 | 🟡 Planned | P3 | xDai stable chain, DAOs |
| **Fantom** | 250 | 🟡 Planned | P3 | Fast, but declining TVL |
| **zkSync Era** | 324 | 🟡 Planned | P2 | zkEVM L2, growing |
| **Polygon zkEVM** | 1101 | 🟡 Planned | P2 | Polygon's zkEVM |

**Target**: Add based on user demand  
**Estimated Effort**: 1-2 days per chain (all EVM-compatible)

---

### **Tier 3: Non-EVM Chains** (Future Consideration)
Requires significant additional work

| Chain | Status | Priority | Challenges |
|-------|--------|----------|-----------|
| **Solana** | 🔴 Future | P3 | Different VM, RPC, data model |
| **Cosmos Chains** | 🔴 Future | P4 | CosmWasm, IBC complexity |
| **Polkadot Parachains** | 🔴 Future | P4 | Substrate, different architecture |
| **Aptos/Sui** | 🔴 Future | P4 | Move VM, nascent ecosystems |

**Target**: Evaluate after Tier 1 + Tier 2 mature  
**Estimated Effort**: 4-6 weeks per ecosystem (requires separate indexer logic)

---

## Technical Considerations by Chain

### Ethereum (Chain ID: 1)
- **Block Time**: ~12 seconds
- **Finality**: 2 epochs (~13 minutes / 64 blocks)
- **RPC Providers**: Alchemy, Infura, QuickNode (all excellent)
- **Tracing**: Requires archive node for `debug_traceTransaction`
- **Unique Features**: Most mature tooling
- **Challenges**: High gas fees, slower blocks

### Polygon (Chain ID: 137)
- **Block Time**: ~2 seconds
- **Finality**: 256 blocks (~8.5 minutes)
- **RPC Providers**: Alchemy, Infura, Polygon native RPC
- **Tracing**: Similar to Ethereum
- **Unique Features**: Very fast, low cost
- **Challenges**: Higher block throughput = more ingestion load

### Arbitrum (Chain ID: 42161)
- **Block Time**: ~0.25 seconds (very fast!)
- **Finality**: ~15 minutes
- **RPC Providers**: Alchemy, Infura, Arbitrum RPC
- **Tracing**: Available via archive nodes
- **Unique Features**: Optimistic rollup, high throughput
- **Challenges**: Extremely fast blocks = highest ingestion load

### Optimism (Chain ID: 10)
- **Block Time**: ~2 seconds
- **Finality**: ~10 minutes
- **RPC Providers**: Alchemy, Infura, Optimism RPC
- **Tracing**: Similar to Ethereum
- **Unique Features**: OP Stack (Base uses same tech)
- **Challenges**: Similar to Polygon

### Base (Chain ID: 8453)
- **Block Time**: ~2 seconds
- **Finality**: ~10 minutes (inherits from OP Stack)
- **RPC Providers**: Alchemy, QuickNode, Base RPC
- **Tracing**: Similar to Optimism
- **Unique Features**: Coinbase-backed, social apps, viral growth
- **Challenges**: Rapid ecosystem growth = frequent schema changes

---

## Implementation Strategy

### Phase 1: Core Infrastructure (Week 1-2)
- ✅ Database schema with multi-chain partitioning
- ✅ Chain metadata table
- ✅ RPC exploration script
- 🟡 Ingester service (chain-agnostic)
- 🟡 Processor service (chain-agnostic)
- 🟡 API service with chain parameter

### Phase 2: Tier 1 Deployment (Week 3-4)
1. **Ethereum** - Launch first, validate everything works
2. **Polygon** - Add high-throughput chain
3. **Arbitrum** - Test with fastest blocks
4. **Optimism + Base** - OP Stack chains together

### Phase 3: Monitoring & Optimization (Week 5-6)
- Monitor ingestion lag per chain
- Optimize for high-throughput chains (Arbitrum, Polygon)
- Load testing
- Cost optimization (RPC calls)

### Phase 4: Tier 2 Expansion (Week 7+)
- Add BSC, Avalanche, zkSync based on demand
- Each takes ~1-2 days to integrate

---

## RPC Provider Strategy

### Free Tier Limits (per provider)
| Provider | Free Requests/Day | Max Requests/Second | Notes |
|----------|-------------------|---------------------|-------|
| **Alchemy** | 300M compute units | ~100 | Best free tier, multiple chains |
| **Infura** | 100K requests | ~10 | More restrictive |
| **QuickNode** | Trial | Varies | Good for paid plans |

### Our Approach
1. **Use multiple providers per chain** - Load balance, failover
2. **Start with Alchemy free tier** - Enough for development
3. **Upgrade to paid when needed** - ~$50/month per chain
4. **Consider dedicated RPC nodes** - For production at scale

---

## Chain-Specific Protocol Differences

### Function Signatures
- **95% identical** across all EVM chains
- Uniswap V2/V3 forks have same signatures
- Some chains have unique DEXs (e.g., Trader Joe on Avalanche)

### Bridge Protocols
Each chain has specific bridges:
- **LayerZero**: Cross all Tier 1 chains
- **Across**: Ethereum ↔ L2s
- **Stargate**: Cross-chain swaps
- **Native bridges**: Polygon Bridge, Arbitrum Bridge, etc.

### NFT Standards
- All chains support ERC721/ERC1155
- Marketplaces vary: OpenSea (multi-chain), Magic Eden, etc.

---

## Monitoring Requirements per Chain

### Key Metrics
```yaml
chains:
  - name: ethereum
    alerts:
      - ingestion_lag > 100 blocks
      - rpc_error_rate > 1%
      - blocks_per_second < 0.05
  
  - name: arbitrum
    alerts:
      - ingestion_lag > 1000 blocks  # Much faster blocks
      - rpc_error_rate > 1%
      - blocks_per_second < 2  # Expect ~4 blocks/sec
```

---

## Cost Estimation

### RPC Costs (Monthly)
| Chain | Free Tier | Paid Tier | Notes |
|-------|-----------|-----------|-------|
| Ethereum | $0 (300M CU) | $50-200/month | Slower blocks = lower cost |
| Polygon | $0 (300M CU) | $100-300/month | Fast blocks = more requests |
| Arbitrum | $0 (300M CU) | $150-400/month | Very fast blocks |
| Optimism | $0 (300M CU) | $75-200/month | Similar to Polygon |
| Base | $0 (300M CU) | $75-200/month | Similar to Optimism |

**Total for Tier 1 (Free)**: $0  
**Total for Tier 1 (Paid)**: $350-1,300/month  
**With self-hosted nodes**: $500-1,000/month (different trade-offs)

---

## Decision Criteria for New Chains

When evaluating new chains, consider:

1. **TVL**: Total Value Locked > $500M
2. **Daily Active Users**: > 10,000
3. **Developer Activity**: Active ecosystem
4. **User Demand**: Customers requesting it
5. **Technical Compatibility**: EVM-compatible = easy, non-EVM = hard
6. **Maintenance Burden**: How stable is the chain?

### Red Flags (Avoid)
- ❌ Chain with frequent hard forks
- ❌ < 6 months old (too unstable)
- ❌ Poor RPC infrastructure
- ❌ No public archive nodes
- ❌ Low activity (< 1000 txs/day)

---

## Action Items

### Immediate (This Week)
- [ ] Run `make explore-rpc` on all Tier 1 chains
- [ ] Document RPC capabilities per chain
- [ ] Update protocol_signatures with chain-specific protocols
- [ ] Create chain-agnostic ingester config

### Short-term (Next 2 Weeks)
- [ ] Launch Ethereum indexing
- [ ] Add Polygon
- [ ] Add Arbitrum, Optimism, Base
- [ ] Monitor performance and costs

### Long-term (1-3 Months)
- [ ] User survey: Which Tier 2 chains to prioritize?
- [ ] Evaluate zkSync Era and Polygon zkEVM
- [ ] Consider Solana (if demand justifies effort)

---

## Resources

### Official Docs
- [Ethereum RPC API](https://ethereum.org/en/developers/docs/apis/json-rpc/)
- [Polygon RPC](https://wiki.polygon.technology/docs/operate/network-rpc-endpoints/)
- [Arbitrum RPC](https://docs.arbitrum.io/build-decentralized-apps/nodeinterface/reference)
- [Optimism RPC](https://docs.optimism.io/builders/node-operators/json-rpc)
- [Base RPC](https://docs.base.org/builders/node-providers)

### Chain Explorers
- Ethereum: https://etherscan.io/
- Polygon: https://polygonscan.com/
- Arbitrum: https://arbiscan.io/
- Optimism: https://optimistic.etherscan.io/
- Base: https://basescan.org/

---

**Last Updated**: November 15, 2025  
**Next Review**: December 1, 2025 (or when adding new chains)
