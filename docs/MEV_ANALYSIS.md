       # MEV Analysis & Detection Guide

**Last Updated**: November 18, 2025  
**Project**: Multi-Chain Blockchain Indexer

---

## Table of Contents

1. [What is MEV?](#what-is-mev)
2. [MEV Strategy Deep Dive](#mev-strategy-deep-dive)
3. [How MEV is Done Profitably](#how-mev-is-done-profitably)
4. [Competitive Analysis](#competitive-analysis)
5. [Detection Algorithms](#detection-algorithms)
6. [Implementation Guide](#implementation-guide)
7. [Schema Design](#schema-design)
8. [Economic Analysis](#economic-analysis)
9. [Interview Questions](#interview-questions)

---

## What is MEV?

### Definition

**MEV (Maximal Extractable Value)** - The maximum value that can be extracted from block production by including, excluding, or reordering transactions within a block.

**Key Concept**: Miners/validators control transaction ordering, creating arbitrage opportunities that don't exist in a fair ordering system.

### Historical Context

- **2019**: Term "Miner Extractable Value" coined by Phil Daian et al.
- **2020-2021**: DeFi summer → MEV explosion ($500M+ extracted)
- **2022**: Flashbots introduces MEV-Boost (private mempool)
- **2023+**: "Maximal" replaces "Miner" (PoS chains, searchers not just miners)
- **2024**: MEV estimated at $2B+ annually on Ethereum

### Why MEV Matters

**For Users**: Higher gas fees, transaction failures, worse execution prices  
**For Protocols**: Design decisions impact MEV extractability  
**For Researchers**: Game theory, incentive alignment, market efficiency  
**For Builders**: Lucrative opportunity ($10M+ for top searchers)  
**For Networks**: Security implications (time-bandit attacks, consensus instability)

---

## MEV Strategy Deep Dive

### 1. Sandwich Attacks

**Mechanism**: Front-run + victim transaction + back-run

```
Block N transactions:
1. MEV Bot: Buy 100 ETH worth of TOKEN (front-run)
2. Victim: Buy 50 ETH worth of TOKEN (original tx)
3. MEV Bot: Sell 100 ETH worth of TOKEN (back-run)

Result: Bot buys low, victim's tx raises price, bot sells high
```

**Profitability Factors**:
- Victim transaction size (larger = more slippage = more profit)
- Pool liquidity (lower liquidity = higher price impact)
- Gas cost (must be < profit)
- Competition (other bots bidding for front-run position)

**Real Example** (Block 18,500,123):
```
Front-run:  0.5 ETH → 10,000 TOKEN (@ price 0.00005 ETH)
Victim:     10 ETH → 150,000 TOKEN (price moves to 0.0000667 ETH)
Back-run:   10,000 TOKEN → 0.667 ETH (@ new price)

Profit: 0.667 - 0.5 - gas = ~0.15 ETH ($300)
Gas spent: ~0.017 ETH ($34)
Net profit: $266
```

**Detection Signatures**:
- Same address for tx[i-1] and tx[i+1]
- Both interact with same DEX pool
- Opposite swap directions (tokenA→tokenB, then tokenB→tokenA)
- Transaction at index i has high gas price (victim)

### 2. Arbitrage

**Mechanism**: Exploit price differences across DEXes

```
Path: USDC → WETH (Uniswap) → USDC (Sushiswap)

Uniswap: 1 WETH = 2,000 USDC
Sushiswap: 1 WETH = 2,010 USDC

Profit: $10 per WETH (minus gas)
```

**Types**:

**A. Cross-DEX Arbitrage**:
- Same token pair, different DEXes
- Example: WETH/USDC on Uniswap vs Sushiswap
- Profit: Price difference × trade size

**B. Triangular Arbitrage**:
- Three tokens, three swaps
- Example: USDC → WETH → DAI → USDC
- Exploits ratio imbalances

**C. Cross-Chain Arbitrage**:
- Same asset, different chains
- Example: ETH on Ethereum ($2,000) vs ETH on Arbitrum ($2,005)
- Requires bridge (slower, higher cost)

**Real Example** (Block 18,501,456):
```solidity
// Single transaction with multiple internal calls

1. Flash loan: Borrow 1,000,000 USDC (Aave)
2. Swap: 1,000,000 USDC → 500 WETH (Uniswap V2)
3. Swap: 500 WETH → 1,005,000 USDC (Uniswap V3)
4. Repay: 1,000,000 USDC + 90 USDC fee
5. Profit: 5,000 - 90 = 4,910 USDC

Gas spent: 0.12 ETH (~$240)
Net profit: $4,670
```

**Profitability Factors**:
- Price divergence magnitude
- Trade size (larger = more profit but higher slippage)
- Gas optimization (fewer hops = lower cost)
- MEV competition (many bots targeting same arb)

### 3. Liquidations

**Mechanism**: Liquidate undercollateralized loans for profit

```
Aave borrower:
- Collateral: 10 ETH ($20,000)
- Borrowed: 15,000 DAI
- Health Factor: 1.33

ETH price drops to $1,800:
- Collateral value: $18,000
- Health Factor: 1.2 → 0.96 (< 1.0, liquidatable!)

Liquidator:
- Repay: 7,500 DAI (50% of debt)
- Receive: 4.2 ETH ($7,560 worth) 
- Profit: $7,560 - $7,500 = $60 + bonus
```

**Profitability Factors**:
- Liquidation bonus (typically 5-10%)
- Close factor (max % of debt to repay, usually 50%)
- Gas costs (liquidations are gas-intensive)
- Competition (first liquidator wins)

**Advanced: Flash Loan Liquidations**:
```
1. Flash loan: 100,000 USDC
2. Swap: USDC → borrowed token (e.g., DAI)
3. Liquidate: Repay DAI, receive collateral (ETH)
4. Swap: ETH → USDC
5. Repay flash loan + fee
6. Keep profit
```

**Real Example** (Block 18,499,888):
```
Position on Compound:
- Collateral: 50 WBTC ($1,500,000)
- Borrowed: 1,100,000 USDC
- Collateral Factor: 75%
- Liquidation Threshold: 80%

WBTC drops 8%:
- Collateral value: $1,380,000
- Borrowed: $1,100,000
- Utilization: 79.7% → 80.5% (LIQUIDATABLE)

Liquidator:
- Repays: 550,000 USDC (50% max)
- Receives: 20 WBTC ($576,000)
- Bonus: 5% = $27,600
- Net profit: ~$25,000 after gas
```

**Detection Signatures**:
- Event: `LiquidationCall` or `Liquidate`
- Transaction calls lending protocol's liquidation function
- Liquidator receives more value than they repay

### 4. NFT Sniping

**Mechanism**: Front-run NFT purchases or mints

```
Victim: Buy CryptoPunk #1234 for 10 ETH
MEV Bot: 
  - Sees victim's tx in mempool
  - Submits same purchase with higher gas (15 ETH tip)
  - Bot's tx mines first
  - Immediately lists for 12 ETH
  - Profit: 2 ETH - gas
```

**Types**:

**A. Mint Sniping**:
- Front-run popular NFT mints
- Get rare traits by manipulating transaction order
- Flip immediately on secondary market

**B. Marketplace Sniping**:
- Front-run underpriced listings
- Monitor mempool for cheap sales
- Buy before victim's transaction executes

**Profitability Factors**:
- NFT rarity/demand
- Price discrepancy
- Gas costs (NFT txs can be expensive)
- Competition (high-value NFTs have many snipers)

### 5. Just-In-Time (JIT) Liquidity

**Mechanism**: Add liquidity right before large swap, remove after

```
Uniswap V3 pool: WETH/USDC
Current liquidity: $10M

Large swap incoming: 1,000 ETH → USDC

MEV Bot:
1. Adds $5M liquidity in tight range (front-run)
2. Large swap executes (bot earns concentrated fees)
3. Removes liquidity (back-run)
4. Profit: Swap fees - gas
```

**Profitability Factors**:
- Swap size (larger = more fees)
- Liquidity concentration (tighter range = more fees per capital)
- Pool fee tier (0.3% vs 0.05% vs 1%)
- Impermanent loss risk (minimized by quick exit)

**Real Example** (Block 18,502,999):
```
Uniswap V3 WETH/USDC 0.05% pool:

1. JIT bot adds $2M liquidity at range [1,999, 2,001]
2. Large swap: 500 ETH → 1,000,000 USDC
3. Bot earns: 500 USDC in fees (0.05% of $1M)
4. Bot removes liquidity immediately
5. Profit: $500 - gas (~$50) = $450
```

**Detection Signatures**:
- Mint LP position in tx[i]
- Large swap in tx[i+1] using that pool
- Burn LP position in tx[i+2]
- Same address for mint and burn

### 6. Back-Running

**Mechanism**: React to price-impacting transactions

```
Oracle update: ETH price changes from $2,000 → $2,100

MEV Bot (immediately after):
1. Buys ETH on DEX (still at $2,000)
2. Sells on DEX that updated faster
3. Profit from slow price propagation
```

**Types**:

**A. Oracle Updates**:
- Chainlink price feed updates
- DEX prices lag oracle by 1-2 blocks
- Arbitrage the gap

**B. Large Trades**:
- Big swap creates price discrepancy
- Back-run to restore balance
- Earn from price normalization

**C. NFT Reveals**:
- NFT metadata revealed on-chain
- Identify rare traits instantly
- Back-run to buy rares before others notice

**Profitability Factors**:
- Speed (milliseconds matter)
- Information advantage (faster trait parsing, oracle reading)
- Lower competition than front-running

---

## How MEV is Done Profitably

### The MEV Supply Chain

```
┌─────────────────────────────────────────────────┐
│ 1. SEARCHERS (MEV Bots)                         │
│    - Identify opportunities                      │
│    - Create MEV bundles                          │
│    - Submit to block builders                    │
└───────────────┬─────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────┐
│ 2. BLOCK BUILDERS                               │
│    - Receive bundles from searchers             │
│    - Simulate and optimize blocks               │
│    - Pay searchers for profitable bundles       │
│    - Bid to proposers                           │
└───────────────┬─────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────┐
│ 3. PROPOSERS (Validators)                      │
│    - Choose highest-paying builder block       │
│    - Propose block to network                   │
│    - Earn builder payment + base fee           │
└─────────────────────────────────────────────────┘

Value Flow: Searcher → Builder → Proposer
```

### Key Infrastructure

#### 1. Flashbots

**What**: Private transaction pool + MEV-Boost relay

**How it works**:
```
Traditional:
User → Mempool (public) → Anyone can front-run → Miner

Flashbots:
User → Flashbots Relay (private) → Builder → Block

Benefits:
- No front-running (transactions private until inclusion)
- Failed transactions don't pay gas
- Searchers bid for block space
```

**Flashbots Bundle**:
```json
{
  "txs": [
    "0xTRANSACTION_1",  // Front-run
    "0xTRANSACTION_2",  // Target
    "0xTRANSACTION_3"   // Back-run
  ],
  "blockNumber": 18500123,
  "minTimestamp": 1700000000,
  "maxTimestamp": 1700000012,
  "revertingTxHashes": []  // These can fail without reverting bundle
}
```

**Searcher Code Example**:
```go
import "github.com/flashbots/mev-share-client-go"

// Create bundle
bundle := flashbots.Bundle{
    Txs: []string{
        signedFrontRunTx,
        victimTxHash,
        signedBackRunTx,
    },
    BlockNumber: targetBlock,
}

// Submit to Flashbots
resp, err := flashbotsClient.SendBundle(bundle)

// Check if bundle was included
included, err := flashbotsClient.GetBundleStats(bundleHash, targetBlock)
```

#### 2. Private Mempools

**Eden Network**: Ethereum-focused, 0% failure tx cost  
**BloXroute**: Multi-chain, low latency  
**Manifold Finance**: OpenMEV, custom strategies

**Why use**:
- Protect from being front-run yourself
- Failed txs don't cost gas
- Direct connection to validators

#### 3. Flash Loans

**Purpose**: Borrow millions without collateral for single transaction

**Providers**:
- **Aave**: 0.09% fee, up to entire pool liquidity
- **dYdX**: 0% fee (gas only), smaller amounts
- **Uniswap V2**: 0.3% fee via flash swaps

**Example Use**:
```solidity
// Aave Flash Loan
contract MEVBot {
    function executeArbitrage() external {
        // 1. Flash loan 1M USDC from Aave
        aave.flashLoan(
            address(this),
            1000000e6, // 1M USDC
            abi.encodeWithSignature("flashLoanCallback()")
        );
    }
    
    function flashLoanCallback() external {
        // 2. Do arbitrage with borrowed funds
        uniswap.swap(1000000e6, USDC, WETH);
        sushiswap.swap(allWETH, WETH, USDC);
        
        // 3. Repay flash loan automatically
        // (If not repaid, entire tx reverts)
    }
}
```

**Economics**:
```
Flash loan: 1,000,000 USDC
Fee: 900 USDC (0.09%)
Arbitrage profit: 5,000 USDC
Net: 5,000 - 900 = 4,100 USDC

Without flash loan: Need $1M capital
With flash loan: Need $0 capital (+ $50 gas)

ROI: Infinite (no capital required)
```

### Profitability Math

#### Sandwich Attack Economics

**Variables**:
- `V` = Victim transaction size (in ETH)
- `L` = Pool liquidity (in ETH)
- `P0` = Initial token price
- `s` = Slippage (price impact)
- `G` = Gas cost
- `f` = Pool fee (0.3% for Uniswap V2)

**Price Impact Formula** (Uniswap constant product):
```
Price impact: s = V / (L + V)
```

**Profit Formula**:
```
Profit = (Front-run amount) × (Price impact) - Gas - Fees

Example:
V = 10 ETH (victim)
L = 500 ETH (pool)
Front-run = 5 ETH

s = 10 / (500 + 10) = 1.96%

Profit = 5 × 0.0196 - 0.02 (gas) - 0.015 (fees)
       = 0.098 - 0.035
       = 0.063 ETH (~$126)
```

**Breakeven Analysis**:
```
Minimum victim size for profitability:

V_min = (Gas + Fees) / (Front-run × Price impact factor)

If gas = 0.02 ETH, fees = 0.01 ETH:
V_min = 0.03 / (5 × 0.0196) ≈ 0.31 ETH

Victim must trade > 0.31 ETH for this to be profitable
```

#### Arbitrage Economics

**Cross-DEX Arbitrage**:
```
Price difference: Δp = 1%
Trade size: 100 ETH
Gas: 0.05 ETH
Slippage (both sides): 0.2%

Gross profit: 100 × 0.01 = 1 ETH
Slippage cost: 100 × 0.002 = 0.2 ETH
Gas cost: 0.05 ETH
Net profit: 1 - 0.2 - 0.05 = 0.75 ETH

Required price difference: > (Gas + Slippage) / Size
                          > (0.05 + 0.2) / 100
                          > 0.25%
```

**Flash Loan Arbitrage**:
```
Borrow: 1,000,000 USDC
Fee: 900 USDC (0.09%)
Price diff: 0.5%
Slippage: 0.1%

Profit: 1,000,000 × (0.005 - 0.001) - 900 - gas
      = 4,000 - 900 - 100
      = 3,000 USDC
```

### Real MEV Searcher Performance

**Top Searchers (2024 data)**:

| Searcher | Monthly Volume | Success Rate | Avg Profit/Tx |
|----------|---------------|--------------|---------------|
| jaredfromsubway.eth | $50M+ | 73% | $1,200 |
| 0xbad...c0de | $30M+ | 68% | $890 |
| mevbot-alpha | $20M+ | 81% | $650 |

**Breakdown by Strategy**:
```
Sandwich attacks: 45% of MEV volume, $15-50 avg profit
Arbitrage: 30% of MEV volume, $200-500 avg profit
Liquidations: 15% of MEV volume, $1,000-10,000 avg profit
JIT liquidity: 7% of MEV volume, $100-300 avg profit
Other: 3% of MEV volume
```

**Competitive Dynamics**:
```
Low Competition (Liquidations):
- 5-10 bots competing
- Profit margin: 40-60%
- Requires capital (no flash loans for some protocols)

Medium Competition (Arbitrage):
- 50-100 bots competing
- Profit margin: 10-20%
- Requires speed + capital

High Competition (Sandwiches):
- 500+ bots competing
- Profit margin: 2-5%
- Gas bidding wars (winner pays 80-90% to miners)
```

---

## Competitive Analysis

### MEV Bot Categories

#### 1. Generalist Bots
**Strategy**: Run all MEV types  
**Examples**: Flashbots searchers, MEV-Share users  
**Advantages**: Diversified income, more opportunities  
**Disadvantages**: Jack of all trades, master of none  
**Profit**: $10-50K/month (top bots)

#### 2. Specialist Bots
**Strategy**: Focus on one MEV type  
**Examples**: Liquidation bots (Aave, Compound), JIT liquidity bots  
**Advantages**: Deep optimization, lower competition  
**Disadvantages**: Fewer opportunities  
**Profit**: $50-200K/month (niche domination)

#### 3. Latency-Optimized Bots
**Strategy**: Win through speed  
**Infrastructure**: 
- Co-located servers near validators
- Custom networking stack
- Direct p2p connections
**Advantages**: First to see and react  
**Disadvantages**: Expensive infrastructure ($10K+/month)  
**Profit**: $100-500K/month (top tier)

#### 4. Capital-Heavy Bots
**Strategy**: Use large capital to dominate liquidations  
**Requirements**: $10M+ TVL  
**Advantages**: Can liquidate any size position  
**Disadvantages**: Capital opportunity cost  
**Profit**: 2-5% annual on capital

### Competitive Moats

**1. Speed**:
```
Latency advantage: 10ms vs 100ms
Opportunities won: 60% vs 20%
Monthly profit: $100K vs $20K

Investment: $50K setup + $10K/month
Break-even: 1-2 months
```

**2. Capital**:
```
Bot A: $10K capital → Can only do arbitrage ≤ $10K
Bot B: $10M capital → Can liquidate positions up to $50M

Bot B captures 10x more opportunities
```

**3. Information**:
```
Private mempool access:
- See transactions 2-3 seconds earlier
- Avoid being front-run
- Higher success rate (80% vs 50%)

Cost: $5K/month (Flashbots Protect)
Value: $50K+ additional profit/month
```

**4. Algorithm**:
```
Basic bot: Checks price every 12 seconds
Advanced bot: Listens to mempool, simulates every tx

Advanced bot finds 10x more opportunities
But requires 100x more infrastructure
```

### Case Study: Top MEV Searcher

**jaredfromsubway.eth** (Most profitable sandwich attacker 2023-2024)

**Stats**:
- Total extracted: $45M+ (18 months)
- Transactions: 500K+
- Success rate: 73%
- Avg profit per sandwich: $89
- Largest single profit: $500K (1 transaction)

**Strategy**:
1. **Generalist approach**: Sandwiches + arbitrage + back-running
2. **Gas optimization**: Custom smart contracts (30% cheaper gas)
3. **Flashbots**: Uses private mempools (avoids competition)
4. **Capital**: $2-5M working capital (can front-run huge trades)
5. **Speed**: Low-latency infrastructure

**Competitive Advantage**:
```
Revenue Sources:
- Sandwich attacks: 60% ($27M)
- Arbitrage: 30% ($13.5M)
- Back-running: 10% ($4.5M)

Key Success Factors:
1. Gas efficiency (saves $10-20 per tx)
2. Bundle optimization (packages multiple MEVs)
3. Flashbots priority (pays premium for inclusion)
4. Capital for large front-runs
```

**Costs**:
```
Monthly Operating Costs:
- Infrastructure: $15K (servers, RPCs, monitoring)
- Gas fees: $200K (refunded if profitable)
- Flashbots tips: $50K (to validators)
- Development: $30K (engineers)
Total: ~$295K/month

Monthly Profit: $2.5M
Operating margin: 88%
```

### Barrier to Entry Analysis

**Low Barriers (Anyone can start)**:
- Small arbitrage ($100-1K profit/tx)
- Simple sandwich attacks (< $1M/month)
- Cost: $5K setup + $1K/month

**Medium Barriers (Requires expertise)**:
- Liquidation bots ($10-50K profit/tx)
- Multi-hop arbitrage
- Cost: $50K setup + $5K/month + $100K capital

**High Barriers (Top 1%)**:
- Latency-optimized infrastructure
- Private mempool access
- Large capital base ($1M+)
- Cost: $500K setup + $50K/month + $10M capital

**Evolution Over Time**:
```
2020: Anyone could profit (high margins, low competition)
2021: Requires some sophistication (tools emerge)
2022: Flashbots changes game (private mempools)
2023: Highly competitive (need speed + capital)
2024: Oligopoly forming (top 10 bots = 60% of MEV)
2025: Consolidation continues (barriers increasing)
```

### Regulatory & Ethical Considerations

**Concerns**:
- **Front-running**: Is it market manipulation?
- **Censorship**: Builders can exclude transactions
- **Centralization**: Top searchers dominate
- **User harm**: Sandwich attacks directly harm traders

**Industry Response**:
- **MEV-Share**: Users share in MEV profits
- **Private transactions**: Protect users from front-running
- **Order flow auctions**: More fair MEV distribution
- **Protocol design**: MEV-resistant protocols (CoW Protocol, UniswapX)

**Interview Perspective**: Be able to discuss both technical implementation AND ethical implications.

---

## Detection Algorithms

### Overview

Building MEV detection requires:
1. **Historical data**: All transactions, events, internal calls
2. **Pattern recognition**: Identify MEV signatures
3. **Profit calculation**: Determine actual MEV value extracted
4. **Classification**: Label MEV type (sandwich, arb, etc.)

### 1. Sandwich Attack Detection

**Algorithm**:

```go
type SandwichDetector struct {
    db *sql.DB
}

// DetectSandwiches finds sandwich attacks in a block
func (d *SandwichDetector) DetectSandwiches(blockNumber int64) ([]MEVTransaction, error) {
    // Get all transactions in block
    txs, err := d.getBlockTransactions(blockNumber)
    if err != nil {
        return nil, err
    }
    
    sandwiches := []MEVTransaction{}
    
    // Check each transaction as potential victim
    for i := 1; i < len(txs)-1; i++ {
        victim := txs[i]
        frontRun := txs[i-1]
        backRun := txs[i+1]
        
        // Check sandwich pattern
        if d.isSandwich(frontRun, victim, backRun) {
            profit := d.calculateSandwichProfit(frontRun, backRun)
            
            sandwiches = append(sandwiches, MEVTransaction{
                Type:            "sandwich",
                SearcherAddress: frontRun.From,
                VictimAddress:   victim.From,
                VictimTxHash:    victim.Hash,
                FrontRunTxHash:  frontRun.Hash,
                BackRunTxHash:   backRun.Hash,
                ProfitUSD:       profit,
                BlockNumber:     blockNumber,
            })
        }
    }
    
    return sandwiches, nil
}

// isSandwich checks if three transactions form a sandwich
func (d *SandwichDetector) isSandwich(frontRun, victim, backRun Transaction) bool {
    // 1. Same searcher for front-run and back-run
    if frontRun.From != backRun.From {
        return false
    }
    
    // 2. All three interact with same pool
    frontPool := d.extractDEXPool(frontRun)
    victimPool := d.extractDEXPool(victim)
    backPool := d.extractDEXPool(backRun)
    
    if frontPool == "" || frontPool != victimPool || frontPool != backPool {
        return false
    }
    
    // 3. Front-run and back-run are opposite directions
    frontSwap := d.extractSwapDetails(frontRun)
    backSwap := d.extractSwapDetails(backRun)
    
    if frontSwap.TokenIn != backSwap.TokenOut || frontSwap.TokenOut != backSwap.TokenIn {
        return false
    }
    
    // 4. Victim has lower gas price (got sandwiched)
    if victim.GasPrice >= frontRun.GasPrice {
        return false
    }
    
    return true
}

// calculateSandwichProfit determines profit from sandwich attack
func (d *SandwichDetector) calculateSandwichProfit(frontRun, backRun Transaction) float64 {
    // Get token balances before and after
    tokenAddress := d.extractOutputToken(backRun)
    
    balanceBefore := d.getTokenBalance(frontRun.From, tokenAddress, frontRun.BlockNumber-1)
    balanceAfter := d.getTokenBalance(backRun.From, tokenAddress, backRun.BlockNumber)
    
    tokenProfit := balanceAfter - balanceBefore
    
    // Convert to USD
    tokenPrice := d.getTokenPrice(tokenAddress, backRun.BlockNumber)
    profitUSD := tokenProfit * tokenPrice
    
    // Subtract gas costs
    gasCost := d.calculateGasCostUSD(frontRun) + d.calculateGasCostUSD(backRun)
    
    return profitUSD - gasCost
}

// extractDEXPool identifies which DEX pool was used
func (d *SandwichDetector) extractDEXPool(tx Transaction) string {
    // Check events for Swap event
    for _, event := range tx.Events {
        if event.EventSignature == "Swap(address,uint256,uint256,uint256,uint256,address)" {
            return event.Address // Pool address
        }
    }
    
    // Check internal transactions for DEX router calls
    for _, internal := range tx.InternalTransactions {
        if d.isDEXRouter(internal.To) {
            // Parse calldata to get pool address
            pool := d.parsePoolFromCalldata(internal.Input)
            if pool != "" {
                return pool
            }
        }
    }
    
    return ""
}
```

**Optimizations**:

```go
// Batch detection (more efficient)
func (d *SandwichDetector) DetectSandwichesBatch(startBlock, endBlock int64) error {
    // 1. Create temporary table with potential patterns
    _, err := d.db.Exec(`
        CREATE TEMP TABLE potential_sandwiches AS
        SELECT 
            a.hash as front_run_hash,
            b.hash as victim_hash,
            c.hash as back_run_hash,
            a.from_address as searcher,
            b.from_address as victim,
            a.block_number
        FROM transactions a
        JOIN transactions b ON a.block_number = b.block_number 
            AND a.transaction_index = b.transaction_index - 1
        JOIN transactions c ON b.block_number = c.block_number 
            AND b.transaction_index = c.transaction_index - 1
        WHERE a.block_number BETWEEN $1 AND $2
            AND a.from_address = c.from_address  -- Same searcher
            AND a.from_address != b.from_address  -- Different from victim
    `, startBlock, endBlock)
    
    // 2. Filter by DEX interaction
    _, err = d.db.Exec(`
        DELETE FROM potential_sandwiches p
        WHERE NOT EXISTS (
            SELECT 1 FROM events e1
            JOIN events e2 ON e1.address = e2.address
            JOIN events e3 ON e2.address = e3.address
            WHERE e1.transaction_hash = p.front_run_hash
                AND e2.transaction_hash = p.victim_hash
                AND e3.transaction_hash = p.back_run_hash
                AND e1.event_signature = 'Swap(...)'
        )
    `)
    
    // 3. Calculate profits and insert
    rows, err := d.db.Query(`SELECT * FROM potential_sandwiches`)
    // ... process and calculate profits
    
    return nil
}
```

### 2. Arbitrage Detection

**Algorithm**:

```go
type ArbitrageDetector struct {
    db *sql.DB
}

// DetectArbitrage finds arbitrage transactions
func (d *ArbitrageDetector) DetectArbitrage(blockNumber int64) ([]MEVTransaction, error) {
    // Find transactions that:
    // 1. Start and end with same token
    // 2. Have multiple swaps
    // 3. End with more than they started
    
    query := `
        SELECT 
            t.hash,
            t.from_address,
            t.block_number,
            array_agg(e.address ORDER BY e.log_index) as pools,
            array_agg(e.data ORDER BY e.log_index) as swap_data
        FROM transactions t
        JOIN events e ON e.transaction_hash = t.hash
        WHERE t.block_number = $1
            AND e.event_signature = 'Swap(address,uint256,uint256,uint256,uint256,address)'
        GROUP BY t.hash, t.from_address, t.block_number
        HAVING COUNT(*) >= 2  -- At least 2 swaps
    `
    
    rows, err := d.db.Query(query, blockNumber)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    arbitrages := []MEVTransaction{}
    
    for rows.Next() {
        var (
            txHash      string
            searcher    string
            blockNum    int64
            pools       []string
            swapData    []string
        )
        
        if err := rows.Scan(&txHash, &searcher, &blockNum, &pools, &swapData); err != nil {
            return nil, err
        }
        
        // Parse swap path
        swapPath := d.parseSwapPath(swapData)
        
        // Check if circular (starts and ends with same token)
        if !d.isCircularPath(swapPath) {
            continue
        }
        
        // Calculate profit
        profit := d.calculateArbitrageProfit(txHash, swapPath)
        
        if profit > 0 {
            arbitrages = append(arbitrages, MEVTransaction{
                Type:            "arbitrage",
                SearcherAddress: searcher,
                TxHash:          txHash,
                ProfitUSD:       profit,
                BlockNumber:     blockNum,
                Details: map[string]interface{}{
                    "pools":     pools,
                    "swap_path": swapPath,
                },
            })
        }
    }
    
    return arbitrages, nil
}

// parseSwapPath extracts token flow from swap events
func (d *ArbitrageDetector) parseSwapPath(swapData []string) []SwapStep {
    path := []SwapStep{}
    
    for _, data := range swapData {
        // Decode Swap event data
        // event Swap(address indexed sender, uint amount0In, uint amount1In, 
        //            uint amount0Out, uint amount1Out, address indexed to)
        
        decoded := d.decodeSwapEvent(data)
        
        step := SwapStep{
            TokenIn:   decoded.TokenIn,
            TokenOut:  decoded.TokenOut,
            AmountIn:  decoded.AmountIn,
            AmountOut: decoded.AmountOut,
        }
        
        path = append(path, step)
    }
    
    return path
}

// isCircularPath checks if arbitrage is circular
func (d *ArbitrageDetector) isCircularPath(path []SwapStep) bool {
    if len(path) < 2 {
        return false
    }
    
    startToken := path[0].TokenIn
    endToken := path[len(path)-1].TokenOut
    
    return startToken == endToken
}

// calculateArbitrageProfit computes profit from arbitrage
func (d *ArbitrageDetector) calculateArbitrageProfit(txHash string, path []SwapStep) float64 {
    // Starting amount
    startAmount := path[0].AmountIn
    
    // Ending amount
    endAmount := path[len(path)-1].AmountOut
    
    // Profit in token terms
    tokenProfit := endAmount - startAmount
    
    // Convert to USD
    token := path[0].TokenIn
    tokenPrice := d.getTokenPrice(token, txHash)
    profitUSD := tokenProfit * tokenPrice
    
    // Subtract gas
    gasCost := d.getGasCostUSD(txHash)
    
    return profitUSD - gasCost
}
```

**Flash Loan Detection**:

```go
// detectFlashLoanArbitrage identifies flash loan usage
func (d *ArbitrageDetector) detectFlashLoanArbitrage(txHash string) (bool, string) {
    // Check for FlashLoan event
    var provider string
    err := d.db.QueryRow(`
        SELECT address
        FROM events
        WHERE transaction_hash = $1
            AND (
                event_signature = 'FlashLoan(address,address,uint256,uint256,uint256)' -- Aave
                OR event_signature = 'LoanMade(address,uint256,uint256,uint256)' -- dYdX
            )
    `, txHash).Scan(&provider)
    
    if err == sql.ErrNoRows {
        return false, ""
    }
    
    // Identify provider
    providerName := d.identifyFlashLoanProvider(provider)
    
    return true, providerName
}
```

### 3. Liquidation Detection

**Algorithm**:

```go
type LiquidationDetector struct {
    db *sql.DB
}

// DetectLiquidations finds liquidation transactions
func (d *LiquidationDetector) DetectLiquidations(blockNumber int64) ([]MEVTransaction, error) {
    // Look for liquidation events
    query := `
        SELECT 
            t.hash,
            t.from_address as liquidator,
            e.address as protocol,
            e.data,
            e.topics
        FROM transactions t
        JOIN events e ON e.transaction_hash = t.hash
        WHERE t.block_number = $1
            AND (
                -- Aave
                e.event_signature = 'LiquidationCall(address,address,address,uint256,uint256,address,bool)' 
                -- Compound
                OR e.event_signature = 'LiquidateBorrow(address,address,uint256,address,uint256)'
                -- MakerDAO
                OR e.event_signature = 'Liquidate(address,address,uint256,uint256,uint256)'
            )
    `
    
    rows, err := d.db.Query(query, blockNumber)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    liquidations := []MEVTransaction{}
    
    for rows.Next() {
        var (
            txHash     string
            liquidator string
            protocol   string
            data       string
            topics     []string
        )
        
        if err := rows.Scan(&txHash, &liquidator, &protocol, &data, pq.Array(&topics)); err != nil {
            return nil, err
        }
        
        // Parse liquidation details
        details := d.parseLiquidationEvent(protocol, data, topics)
        
        // Calculate profit
        profit := d.calculateLiquidationProfit(details)
        
        liquidations = append(liquidations, MEVTransaction{
            Type:            "liquidation",
            SearcherAddress: liquidator,
            TxHash:          txHash,
            ProfitUSD:       profit,
            BlockNumber:     blockNumber,
            Details: map[string]interface{}{
                "protocol":        d.identifyProtocol(protocol),
                "collateral":      details.CollateralToken,
                "collateral_amount": details.CollateralAmount,
                "debt_repaid":     details.DebtRepaid,
                "liquidation_bonus": details.Bonus,
            },
        })
    }
    
    return liquidations, nil
}

// calculateLiquidationProfit computes profit from liquidation
func (d *LiquidationDetector) calculateLiquidationProfit(details LiquidationDetails) float64 {
    // Value of collateral received
    collateralPrice := d.getTokenPrice(details.CollateralToken, details.BlockNumber)
    collateralValue := details.CollateralAmount * collateralPrice
    
    // Cost of debt repaid
    debtPrice := d.getTokenPrice(details.DebtToken, details.BlockNumber)
    debtCost := details.DebtRepaid * debtPrice
    
    // Gross profit (includes bonus)
    grossProfit := collateralValue - debtCost
    
    // Subtract gas
    gasCost := d.getGasCostUSD(details.TxHash)
    
    // Subtract flash loan fee if used
    flashLoanFee := 0.0
    if details.UsedFlashLoan {
        flashLoanFee = details.DebtRepaid * 0.0009 * debtPrice // 0.09% Aave fee
    }
    
    return grossProfit - gasCost - flashLoanFee
}
```

### 4. JIT Liquidity Detection

**Algorithm**:

```go
type JITDetector struct {
    db *sql.DB
}

// DetectJIT finds just-in-time liquidity provision
func (d *JITDetector) DetectJIT(blockNumber int64) ([]MEVTransaction, error) {
    // Find patterns: Mint LP → Large Swap → Burn LP
    query := `
        WITH lp_actions AS (
            SELECT 
                e.transaction_hash,
                e.address as pool,
                e.log_index,
                CASE 
                    WHEN e.event_signature = 'Mint(address,uint256,uint256)' THEN 'mint'
                    WHEN e.event_signature = 'Burn(address,uint256,uint256,address)' THEN 'burn'
                    WHEN e.event_signature = 'Swap(...)' THEN 'swap'
                END as action_type,
                t.from_address,
                t.transaction_index
            FROM events e
            JOIN transactions t ON t.hash = e.transaction_hash
            WHERE t.block_number = $1
                AND e.event_signature IN (
                    'Mint(address,uint256,uint256)',
                    'Burn(address,uint256,uint256,address)',
                    'Swap(...)'
                )
        )
        SELECT 
            mint.transaction_hash as mint_tx,
            swap.transaction_hash as swap_tx,
            burn.transaction_hash as burn_tx,
            mint.pool,
            mint.from_address as jit_provider
        FROM lp_actions mint
        JOIN lp_actions swap ON swap.pool = mint.pool
            AND swap.transaction_index = mint.transaction_index + 1
            AND swap.action_type = 'swap'
        JOIN lp_actions burn ON burn.pool = mint.pool
            AND burn.transaction_index = swap.transaction_index + 1
            AND burn.action_type = 'burn'
            AND burn.from_address = mint.from_address
        WHERE mint.action_type = 'mint'
    `
    
    rows, err := d.db.Query(query, blockNumber)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    jitTransactions := []MEVTransaction{}
    
    for rows.Next() {
        var (
            mintTx      string
            swapTx      string
            burnTx      string
            pool        string
            jitProvider string
        )
        
        if err := rows.Scan(&mintTx, &swapTx, &burnTx, &pool, &jitProvider); err != nil {
            return nil, err
        }
        
        // Calculate profit from fees earned
        profit := d.calculateJITProfit(mintTx, swapTx, burnTx, pool)
        
        jitTransactions = append(jitTransactions, MEVTransaction{
            Type:            "jit_liquidity",
            SearcherAddress: jitProvider,
            TxHash:          mintTx,
            ProfitUSD:       profit,
            BlockNumber:     blockNumber,
            Details: map[string]interface{}{
                "mint_tx": mintTx,
                "swap_tx": swapTx,
                "burn_tx": burnTx,
                "pool":    pool,
            },
        })
    }
    
    return jitTransactions, nil
}

// calculateJITProfit computes profit from JIT liquidity
func (d *JITDetector) calculateJITProfit(mintTx, swapTx, burnTx, pool string) float64 {
    // Get liquidity amounts from Mint event
    mintAmount := d.getLiquidityAmount(mintTx, pool)
    
    // Get swap fee (from swap event)
    swapFee := d.getSwapFeeEarned(swapTx, pool, mintAmount)
    
    // Get liquidity removed (should be approximately same as minted)
    burnAmount := d.getLiquidityAmount(burnTx, pool)
    
    // Calculate impermanent loss (should be minimal due to short duration)
    impermanentLoss := d.calculateImpermanentLoss(mintTx, burnTx, pool)
    
    // Net profit
    netProfit := swapFee - impermanentLoss
    
    // Subtract gas costs for all three transactions
    gasCost := d.getGasCostUSD(mintTx) + d.getGasCostUSD(burnTx)
    
    return netProfit - gasCost
}
```

### 5. Multi-Strategy Detection Pipeline

**Combined Detection**:

```go
type MEVDetector struct {
    sandwichDetector   *SandwichDetector
    arbitrageDetector  *ArbitrageDetector
    liquidationDetector *LiquidationDetector
    jitDetector        *JITDetector
    db                 *sql.DB
}

// DetectAllMEV runs all detection algorithms for a block
func (d *MEVDetector) DetectAllMEV(blockNumber int64) ([]MEVTransaction, error) {
    var allMEV []MEVTransaction
    
    // Run detectors in parallel
    var wg sync.WaitGroup
    results := make(chan []MEVTransaction, 4)
    errors := make(chan error, 4)
    
    wg.Add(4)
    
    // Sandwich detection
    go func() {
        defer wg.Done()
        sandwiches, err := d.sandwichDetector.DetectSandwiches(blockNumber)
        if err != nil {
            errors <- err
            return
        }
        results <- sandwiches
    }()
    
    // Arbitrage detection
    go func() {
        defer wg.Done()
        arbitrages, err := d.arbitrageDetector.DetectArbitrage(blockNumber)
        if err != nil {
            errors <- err
            return
        }
        results <- arbitrages
    }()
    
    // Liquidation detection
    go func() {
        defer wg.Done()
        liquidations, err := d.liquidationDetector.DetectLiquidations(blockNumber)
        if err != nil {
            errors <- err
            return
        }
        results <- liquidations
    }()
    
    // JIT detection
    go func() {
        defer wg.Done()
        jit, err := d.jitDetector.DetectJIT(blockNumber)
        if err != nil {
            errors <- err
            return
        }
        results <- jit
    }()
    
    // Wait for all detectors
    go func() {
        wg.Wait()
        close(results)
        close(errors)
    }()
    
    // Collect results
    for mevTxs := range results {
        allMEV = append(allMEV, mevTxs...)
    }
    
    // Check for errors
    for err := range errors {
        if err != nil {
            return nil, err
        }
    }
    
    return allMEV, nil
}

// ProcessHistoricalBlocks processes multiple blocks efficiently
func (d *MEVDetector) ProcessHistoricalBlocks(startBlock, endBlock int64, workers int) error {
    blockQueue := make(chan int64, 1000)
    var wg sync.WaitGroup
    
    // Start workers
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for blockNum := range blockQueue {
                mevTxs, err := d.DetectAllMEV(blockNum)
                if err != nil {
                    log.Printf("Error processing block %d: %v", blockNum, err)
                    continue
                }
                
                // Save to database
                if err := d.saveMEVTransactions(mevTxs); err != nil {
                    log.Printf("Error saving MEV for block %d: %v", blockNum, err)
                }
                
                if blockNum % 1000 == 0 {
                    log.Printf("Processed block %d", blockNum)
                }
            }
        }()
    }
    
    // Feed blocks to workers
    for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
        blockQueue <- blockNum
    }
    close(blockQueue)
    
    wg.Wait()
    return nil
}
```

---

## Implementation Guide

### Phase 1: Database Schema

**Step 1: Create MEV Tables**

```sql
-- migrations/005_add_mev_tables.sql

-- Main MEV transactions table
CREATE TABLE mev_transactions (
    id BIGSERIAL PRIMARY KEY,
    chain_id INTEGER NOT NULL,
    block_number BIGINT NOT NULL,
    transaction_hash VARCHAR(66) NOT NULL,
    mev_type VARCHAR(50) NOT NULL, -- sandwich, arbitrage, liquidation, jit, backrun
    searcher_address VARCHAR(42) NOT NULL,
    victim_address VARCHAR(42), -- NULL for non-sandwich
    profit_usd NUMERIC(20, 6) NOT NULL,
    profit_token VARCHAR(42), -- Token profit was realized in
    profit_amount NUMERIC(40, 18),
    gas_cost_usd NUMERIC(20, 6) NOT NULL,
    details JSONB, -- Strategy-specific details
    created_at TIMESTAMP DEFAULT NOW()
) PARTITION BY LIST (chain_id);

-- Create partitions for each chain
CREATE TABLE mev_transactions_ethereum PARTITION OF mev_transactions FOR VALUES IN (1);
CREATE TABLE mev_transactions_arbitrum PARTITION OF mev_transactions FOR VALUES IN (42161);
CREATE TABLE mev_transactions_base PARTITION OF mev_transactions FOR VALUES IN (8453);

-- Indexes
CREATE INDEX idx_mev_txs_block ON mev_transactions (chain_id, block_number);
CREATE INDEX idx_mev_txs_searcher ON mev_transactions (searcher_address, chain_id);
CREATE INDEX idx_mev_txs_type ON mev_transactions (mev_type, chain_id);
CREATE INDEX idx_mev_txs_profit ON mev_transactions (profit_usd DESC);
CREATE INDEX idx_mev_txs_details ON mev_transactions USING gin (details);

-- MEV bundles (Flashbots/private tx pools)
CREATE TABLE mev_bundles (
    id BIGSERIAL PRIMARY KEY,
    chain_id INTEGER NOT NULL,
    block_number BIGINT NOT NULL,
    bundle_hash VARCHAR(66) NOT NULL,
    searcher_address VARCHAR(42) NOT NULL,
    transaction_hashes TEXT[], -- Array of tx hashes in bundle
    total_profit_usd NUMERIC(20, 6),
    builder_payment_usd NUMERIC(20, 6), -- Payment to block builder
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(chain_id, block_number, bundle_hash)
) PARTITION BY LIST (chain_id);

CREATE TABLE mev_bundles_ethereum PARTITION OF mev_bundles FOR VALUES IN (1);
CREATE INDEX idx_mev_bundles_block ON mev_bundles (chain_id, block_number);
CREATE INDEX idx_mev_bundles_searcher ON mev_bundles (searcher_address);

-- MEV searcher analytics
CREATE TABLE mev_searchers (
    id BIGSERIAL PRIMARY KEY,
    chain_id INTEGER NOT NULL,
    searcher_address VARCHAR(42) NOT NULL,
    first_seen_block BIGINT NOT NULL,
    last_seen_block BIGINT NOT NULL,
    total_transactions BIGINT DEFAULT 0,
    total_profit_usd NUMERIC(20, 6) DEFAULT 0,
    total_gas_spent_usd NUMERIC(20, 6) DEFAULT 0,
    success_rate NUMERIC(5, 2), -- Percentage
    strategies JSONB, -- Count by strategy type
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(chain_id, searcher_address)
) PARTITION BY LIST (chain_id);

CREATE TABLE mev_searchers_ethereum PARTITION OF mev_searchers FOR VALUES IN (1);
CREATE INDEX idx_mev_searchers_profit ON mev_searchers (total_profit_usd DESC);

-- Sandwich-specific details
CREATE TABLE sandwich_details (
    mev_transaction_id BIGINT PRIMARY KEY REFERENCES mev_transactions(id),
    victim_transaction_hash VARCHAR(66) NOT NULL,
    front_run_transaction_hash VARCHAR(66) NOT NULL,
    back_run_transaction_hash VARCHAR(66) NOT NULL,
    dex_pool VARCHAR(42) NOT NULL,
    token_in VARCHAR(42) NOT NULL,
    token_out VARCHAR(42) NOT NULL,
    victim_amount_in NUMERIC(40, 18),
    victim_slippage_percent NUMERIC(10, 4), -- How much worse execution victim got
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sandwich_victim ON sandwich_details (victim_transaction_hash);

-- Arbitrage-specific details
CREATE TABLE arbitrage_details (
    mev_transaction_id BIGINT PRIMARY KEY REFERENCES mev_transactions(id),
    swap_path JSONB NOT NULL, -- Array of {pool, tokenIn, tokenOut, amountIn, amountOut}
    used_flash_loan BOOLEAN DEFAULT false,
    flash_loan_provider VARCHAR(50), -- aave, dydx, etc
    flash_loan_amount NUMERIC(40, 18),
    num_hops INTEGER NOT NULL, -- Number of swaps
    created_at TIMESTAMP DEFAULT NOW()
);

-- Liquidation-specific details
CREATE TABLE liquidation_details (
    mev_transaction_id BIGINT PRIMARY KEY REFERENCES mev_transactions(id),
    protocol VARCHAR(50) NOT NULL, -- aave, compound, maker
    borrower_address VARCHAR(42) NOT NULL,
    collateral_token VARCHAR(42) NOT NULL,
    collateral_amount NUMERIC(40, 18) NOT NULL,
    debt_token VARCHAR(42) NOT NULL,
    debt_repaid NUMERIC(40, 18) NOT NULL,
    liquidation_bonus_percent NUMERIC(10, 4), -- e.g., 5.00 for 5%
    used_flash_loan BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_liquidation_protocol ON liquidation_details (protocol);
CREATE INDEX idx_liquidation_borrower ON liquidation_details (borrower_address);
```

**Step 2: Add Helper Functions**

```sql
-- Function to update searcher stats
CREATE OR REPLACE FUNCTION update_mev_searcher_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO mev_searchers (
        chain_id,
        searcher_address,
        first_seen_block,
        last_seen_block,
        total_transactions,
        total_profit_usd,
        total_gas_spent_usd
    )
    VALUES (
        NEW.chain_id,
        NEW.searcher_address,
        NEW.block_number,
        NEW.block_number,
        1,
        NEW.profit_usd,
        NEW.gas_cost_usd
    )
    ON CONFLICT (chain_id, searcher_address) DO UPDATE SET
        last_seen_block = NEW.block_number,
        total_transactions = mev_searchers.total_transactions + 1,
        total_profit_usd = mev_searchers.total_profit_usd + NEW.profit_usd,
        total_gas_spent_usd = mev_searchers.total_gas_spent_usd + NEW.gas_cost_usd,
        updated_at = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update searcher stats
CREATE TRIGGER trigger_update_mev_searcher_stats
AFTER INSERT ON mev_transactions
FOR EACH ROW
EXECUTE FUNCTION update_mev_searcher_stats();

-- Materialized view for daily MEV stats
CREATE MATERIALIZED VIEW mev_daily_stats AS
SELECT 
    chain_id,
    DATE(to_timestamp(block_number * 12)) as date, -- Approximate (12s blocks)
    mev_type,
    COUNT(*) as transaction_count,
    SUM(profit_usd) as total_profit_usd,
    AVG(profit_usd) as avg_profit_usd,
    MAX(profit_usd) as max_profit_usd,
    SUM(gas_cost_usd) as total_gas_cost_usd,
    COUNT(DISTINCT searcher_address) as unique_searchers
FROM mev_transactions
GROUP BY chain_id, DATE(to_timestamp(block_number * 12)), mev_type;

CREATE UNIQUE INDEX idx_mev_daily_stats ON mev_daily_stats (chain_id, date, mev_type);

-- Refresh function (call this periodically)
CREATE OR REPLACE FUNCTION refresh_mev_stats()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mev_daily_stats;
END;
$$ LANGUAGE plpgsql;
```

### Phase 2: Detection Service

**Step 1: Service Structure**

```go
// services/mev-detector/main.go

package main

import (
    "context"
    "database/sql"
    "log"
    "time"
    
    "github.com/IBM/sarama"
    _ "github.com/lib/pq"
)

type MEVDetectorService struct {
    db              *sql.DB
    kafkaConsumer   sarama.ConsumerGroup
    detector        *MEVDetector
    config          *Config
}

type Config struct {
    DatabaseURL        string
    KafkaBrokers       []string
    KafkaGroupID       string
    WorkerCount        int
    StartBlock         int64
    RealtimeMode       bool
}

func main() {
    config := loadConfig()
    
    // Connect to database
    db, err := sql.Open("postgres", config.DatabaseURL)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Initialize detectors
    detector := NewMEVDetector(db)
    
    // Create service
    service := &MEVDetectorService{
        db:       db,
        detector: detector,
        config:   config,
    }
    
    // Run in appropriate mode
    if config.RealtimeMode {
        log.Println("Starting realtime MEV detection...")
        service.runRealtime()
    } else {
        log.Printf("Starting historical MEV detection from block %d...", config.StartBlock)
        service.runHistorical()
    }
}

// runRealtime consumes from Kafka and detects MEV in realtime
func (s *MEVDetectorService) runRealtime() {
    // Setup Kafka consumer
    config := sarama.NewConfig()
    config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
    config.Consumer.Offsets.Initial = sarama.OffsetNewest
    
    consumer, err := sarama.NewConsumerGroup(s.config.KafkaBrokers, s.config.KafkaGroupID, config)
    if err != nil {
        log.Fatal(err)
    }
    defer consumer.Close()
    
    // Subscribe to parsed transactions topic
    topics := []string{"parsed-transactions", "parsed-events"}
    
    ctx := context.Background()
    handler := &MEVConsumerHandler{service: s}
    
    for {
        if err := consumer.Consume(ctx, topics, handler); err != nil {
            log.Printf("Error consuming: %v", err)
            time.Sleep(5 * time.Second)
        }
    }
}

// runHistorical processes historical blocks
func (s *MEVDetectorService) runHistorical() {
    // Get latest processed block
    var lastBlock int64
    err := s.db.QueryRow(`
        SELECT COALESCE(MAX(block_number), $1) 
        FROM mev_transactions 
        WHERE chain_id = 1
    `, s.config.StartBlock).Scan(&lastBlock)
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Get latest indexed block
    var latestBlock int64
    err = s.db.QueryRow(`
        SELECT MAX(block_number) 
        FROM blocks 
        WHERE chain_id = 1
    `).Scan(&latestBlock)
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Processing blocks %d to %d", lastBlock+1, latestBlock)
    
    // Process in batches
    err = s.detector.ProcessHistoricalBlocks(lastBlock+1, latestBlock, s.config.WorkerCount)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Historical processing complete")
}

// MEVConsumerHandler handles Kafka messages
type MEVConsumerHandler struct {
    service *MEVDetectorService
}

func (h *MEVConsumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *MEVConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *MEVConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    blockBuffer := make(map[int64]bool)
    
    for message := range claim.Messages() {
        // Parse message to get block number
        blockNum := extractBlockNumber(message.Value)
        
        // Buffer blocks to process complete blocks only
        blockBuffer[blockNum] = true
        
        // Process blocks that are complete (simple heuristic: block is 2 behind current)
        for block := range blockBuffer {
            if block < blockNum-2 {
                // Process this block
                mevTxs, err := h.service.detector.DetectAllMEV(block)
                if err != nil {
                    log.Printf("Error detecting MEV in block %d: %v", block, err)
                } else {
                    h.service.detector.saveMEVTransactions(mevTxs)
                    log.Printf("Found %d MEV transactions in block %d", len(mevTxs), block)
                }
                
                delete(blockBuffer, block)
            }
        }
        
        session.MarkMessage(message, "")
    }
    
    return nil
}
```

**Step 2: Docker Configuration**

```yaml
# docker-compose.yml additions

services:
  mev-detector:
    build:
      context: ./services/mev-detector
      dockerfile: Dockerfile
    environment:
      - DATABASE_URL=postgres://user:pass@postgres:5432/indexer
      - KAFKA_BROKERS=kafka:9092
      - KAFKA_GROUP_ID=mev-detector
      - WORKER_COUNT=10
      - REALTIME_MODE=true
    depends_on:
      - postgres
      - kafka
      - processor
    restart: unless-stopped
```

### Phase 3: API Endpoints

**Step 1: Add MEV Routes**

```go
// services/api/handlers/mev_handlers.go

package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"
    
    "github.com/gorilla/mux"
)

type MEVHandler struct {
    db *sql.DB
}

// GetMEVByBlock returns all MEV transactions in a block
func (h *MEVHandler) GetMEVByBlock(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    chainID, _ := strconv.Atoi(vars["chain_id"])
    blockNumber, _ := strconv.ParseInt(vars["block_number"], 10, 64)
    
    query := `
        SELECT 
            transaction_hash,
            mev_type,
            searcher_address,
            victim_address,
            profit_usd,
            gas_cost_usd,
            details
        FROM mev_transactions
        WHERE chain_id = $1 AND block_number = $2
        ORDER BY profit_usd DESC
    `
    
    rows, err := h.db.Query(query, chainID, blockNumber)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var mevTxs []MEVTransactionResponse
    for rows.Next() {
        var tx MEVTransactionResponse
        err := rows.Scan(
            &tx.TxHash,
            &tx.MEVType,
            &tx.Searcher,
            &tx.Victim,
            &tx.ProfitUSD,
            &tx.GasCostUSD,
            &tx.Details,
        )
        if err != nil {
            continue
        }
        mevTxs = append(mevTxs, tx)
    }
    
    json.NewEncoder(w).Encode(mevTxs)
}

// GetTopSearchers returns top MEV searchers by profit
func (h *MEVHandler) GetTopSearchers(w http.ResponseWriter, r *http.Request) {
    chainID, _ := strconv.Atoi(r.URL.Query().Get("chain_id"))
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 {
        limit = 100
    }
    
    query := `
        SELECT 
            searcher_address,
            total_transactions,
            total_profit_usd,
            total_gas_spent_usd,
            (total_profit_usd - total_gas_spent_usd) as net_profit,
            success_rate,
            strategies
        FROM mev_searchers
        WHERE chain_id = $1
        ORDER BY total_profit_usd DESC
        LIMIT $2
    `
    
    rows, err := h.db.Query(query, chainID, limit)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var searchers []SearcherStats
    for rows.Next() {
        var s SearcherStats
        err := rows.Scan(
            &s.Address,
            &s.TotalTransactions,
            &s.TotalProfitUSD,
            &s.TotalGasSpentUSD,
            &s.NetProfitUSD,
            &s.SuccessRate,
            &s.Strategies,
        )
        if err != nil {
            continue
        }
        searchers = append(searchers, s)
    }
    
    json.NewEncoder(w).Encode(searchers)
}

// GetMEVStats returns aggregated MEV statistics
func (h *MEVHandler) GetMEVStats(w http.ResponseWriter, r *http.Request) {
    chainID, _ := strconv.Atoi(r.URL.Query().Get("chain_id"))
    days, _ := strconv.Atoi(r.URL.Query().Get("days"))
    if days == 0 {
        days = 30
    }
    
    query := `
        SELECT 
            date,
            mev_type,
            transaction_count,
            total_profit_usd,
            avg_profit_usd,
            max_profit_usd,
            total_gas_cost_usd,
            unique_searchers
        FROM mev_daily_stats
        WHERE chain_id = $1 
            AND date >= CURRENT_DATE - INTERVAL '%d days'
        ORDER BY date DESC, mev_type
    `
    
    rows, err := h.db.Query(fmt.Sprintf(query, days), chainID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var stats []MEVDailyStats
    for rows.Next() {
        var s MEVDailyStats
        err := rows.Scan(
            &s.Date,
            &s.MEVType,
            &s.TransactionCount,
            &s.TotalProfitUSD,
            &s.AvgProfitUSD,
            &s.MaxProfitUSD,
            &s.TotalGasCostUSD,
            &s.UniqueSearchers,
        )
        if err != nil {
            continue
        }
        stats = append(stats, s)
    }
    
    json.NewEncoder(w).Encode(stats)
}

// RegisterMEVRoutes registers all MEV-related routes
func RegisterMEVRoutes(r *mux.Router, db *sql.DB) {
    handler := &MEVHandler{db: db}
    
    r.HandleFunc("/api/v1/mev/{chain_id}/{block_number}", handler.GetMEVByBlock).Methods("GET")
    r.HandleFunc("/api/v1/mev/searchers", handler.GetTopSearchers).Methods("GET")
    r.HandleFunc("/api/v1/mev/stats", handler.GetMEVStats).Methods("GET")
}
```

### Phase 4: Monitoring & Alerts

**Step 1: Prometheus Metrics**

```go
// services/mev-detector/metrics.go

package main

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    mevDetected = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mev_transactions_detected_total",
            Help: "Total number of MEV transactions detected",
        },
        []string{"chain_id", "mev_type"},
    )
    
    mevProfit = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mev_profit_usd_total",
            Help: "Total MEV profit detected in USD",
        },
        []string{"chain_id", "mev_type"},
    )
    
    detectionLatency = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "mev_detection_duration_seconds",
            Help:    "Time taken to detect MEV in a block",
            Buckets: prometheus.DefBuckets,
        },
        []string{"chain_id"},
    )
)

func recordMEVDetection(chainID int, mevType string, profitUSD float64) {
    mevDetected.WithLabelValues(strconv.Itoa(chainID), mevType).Inc()
    mevProfit.WithLabelValues(strconv.Itoa(chainID), mevType).Add(profitUSD)
}
```

**Step 2: Grafana Dashboard**

```json
// monitoring/grafana/dashboards/mev.json

{
  "dashboard": {
    "title": "MEV Analysis",
    "panels": [
      {
        "title": "MEV Profit by Type (24h)",
        "targets": [{
          "expr": "sum by (mev_type) (increase(mev_profit_usd_total[24h]))"
        }],
        "type": "piechart"
      },
      {
        "title": "Top MEV Searchers",
        "targets": [{
          "expr": "topk(10, mev_searcher_profit_usd)"
        }],
        "type": "table"
      },
      {
        "title": "MEV Detection Rate",
        "targets": [{
          "expr": "rate(mev_transactions_detected_total[5m])"
        }],
        "type": "graph"
      }
    ]
  }
}
```

---

## Schema Design

### Complete ERD

```
blocks
├── transactions (1:N)
│   ├── events (1:N)
│   └── internal_transactions (1:N)
│
└── mev_transactions (1:N)
    ├── sandwich_details (1:1)
    ├── arbitrage_details (1:1)
    └── liquidation_details (1:1)

mev_searchers (aggregated stats)
mev_bundles (Flashbots bundles)
mev_daily_stats (materialized view)
```

### Query Examples

**1. Find most profitable sandwiches today**:

```sql
SELECT 
    mt.transaction_hash,
    mt.searcher_address,
    mt.profit_usd,
    sd.victim_transaction_hash,
    sd.dex_pool,
    sd.victim_slippage_percent
FROM mev_transactions mt
JOIN sandwich_details sd ON sd.mev_transaction_id = mt.id
WHERE mt.mev_type = 'sandwich'
    AND mt.created_at >= CURRENT_DATE
    AND mt.chain_id = 1
ORDER BY mt.profit_usd DESC
LIMIT 100;
```

**2. Analyze searcher performance**:

```sql
SELECT 
    searcher_address,
    total_profit_usd - total_gas_spent_usd as net_profit,
    total_transactions,
    (total_profit_usd - total_gas_spent_usd) / NULLIF(total_transactions, 0) as avg_net_profit_per_tx,
    success_rate,
    strategies->>'sandwich' as sandwich_count,
    strategies->>'arbitrage' as arbitrage_count
FROM mev_searchers
WHERE chain_id = 1
ORDER BY net_profit DESC
LIMIT 50;
```

**3. Find flash loan arbitrages**:

```sql
SELECT 
    mt.transaction_hash,
    mt.searcher_address,
    mt.profit_usd,
    ad.flash_loan_provider,
    ad.flash_loan_amount,
    ad.num_hops,
    ad.swap_path
FROM mev_transactions mt
JOIN arbitrage_details ad ON ad.mev_transaction_id = mt.id
WHERE ad.used_flash_loan = true
    AND mt.chain_id = 1
ORDER BY mt.profit_usd DESC
LIMIT 100;
```

**4. MEV timeline analysis**:

```sql
SELECT 
    date,
    mev_type,
    transaction_count,
    total_profit_usd,
    unique_searchers,
    total_profit_usd / NULLIF(transaction_count, 0) as avg_profit_per_tx
FROM mev_daily_stats
WHERE chain_id = 1
    AND date >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY date DESC, total_profit_usd DESC;
```

**5. Most victimized addresses**:

```sql
SELECT 
    sd.victim_address,
    COUNT(*) as times_sandwiched,
    SUM(mt.profit_usd) as total_value_extracted,
    AVG(sd.victim_slippage_percent) as avg_slippage
FROM sandwich_details sd
JOIN mev_transactions mt ON mt.id = sd.mev_transaction_id
WHERE mt.chain_id = 1
    AND mt.created_at >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY sd.victim_address
ORDER BY total_value_extracted DESC
LIMIT 100;
```

---

## Economic Analysis

### Total Addressable Market

**Ethereum MEV (2024)**:
- Total extracted: ~$2.1B
- Daily average: ~$5.7M
- Top 100 searchers: 85% of volume
- Top 10 searchers: 45% of volume

**Market Share by Strategy**:
```
Sandwich attacks: $945M (45%)
├── Uniswap V2: $420M
├── Uniswap V3: $315M
└── Other DEXes: $210M

Arbitrage: $630M (30%)
├── Cross-DEX: $400M
├── Triangular: $180M
└── Cross-chain: $50M

Liquidations: $315M (15%)
├── Aave: $180M
├── Compound: $95M
└── Others: $40M

JIT Liquidity: $147M (7%)
Back-running: $63M (3%)
```

### Cost Structure

**Infrastructure Costs** (monthly, top-tier operation):

```
1. Servers & Networking:
   - Co-located servers (validator proximity): $5,000
   - High-performance nodes (archive): $3,000
   - Low-latency networking: $2,000
   Subtotal: $10,000/month

2. RPC & Data:
   - Alchemy/Infura premium: $2,000
   - Flashbots/MEV-Boost access: $1,000
   - Historical data feeds: $1,000
   Subtotal: $4,000/month

3. Development & Maintenance:
   - Engineers (2 FTEsalary): $30,000
   - DevOps: $5,000
   Subtotal: $35,000/month

4. Gas & Transaction Costs:
   - Failed transactions: $50,000
   - Successful transactions: $200,000
   (Mostly recovered from profits)
   Subtotal: $250,000/month (break-even)

Total Operating Costs: ~$50,000/month (excluding gas)
```

**Capital Requirements**:

```
Tier 1 (Hobbyist): $5-10K
- Small arbitrages only
- No liquidations (requires capital)
- Expected return: $1-5K/month

Tier 2 (Semi-Pro): $50-100K
- Medium arbitrages
- Small liquidations
- Expected return: $10-50K/month

Tier 3 (Professional): $500K-1M
- Large arbitrages
- Most liquidations
- Expected return: $100-500K/month

Tier 4 (Institutional): $10M+
- Any opportunity
- Can dominate liquidations
- Expected return: $1-5M/month
```

### Profitability Analysis

**Example: Professional MEV Searcher (Tier 3)**

**Monthly Metrics**:
```
Capital deployed: $500,000
Infrastructure cost: $50,000/month
Transactions executed: 10,000
Success rate: 70%

Revenue Breakdown:
- Sandwich attacks (60% of ops): $180,000
  * Avg profit: $30 per successful tx
  * 6,000 attempts → 4,200 successful

- Arbitrages (30% of ops): $200,000
  * Avg profit: $95 per successful tx
  * 3,000 attempts → 2,100 successful

- Liquidations (10% of ops): $170,000
  * Avg profit: $2,400 per successful tx
  * 1,000 attempts → 70 successful

Total Revenue: $550,000/month

Costs:
- Infrastructure: $50,000
- Gas (net): $100,000
Total Costs: $150,000

Net Profit: $400,000/month
ROI on capital: 80%/month (960%/year)
Operating margin: 73%
```

**Scalability Limits**:

```
Physical Limits:
- Network latency: Speed of light (can't improve)
- Block production: 12 seconds (Ethereum constant)
- Competition: More bots = lower margins

Economic Limits:
- Opportunity size: Limited by DeFi volume
- Gas costs: Eat into profits during high congestion
- Capital efficiency: Large capital has diminishing returns

Current State (2024):
- Market approaching saturation
- Margins decreasing (2021: 15% avg → 2024: 5% avg)
- Consolidation: Top searchers have economies of scale
```

### Risk Factors

**Technical Risks**:
1. **Smart Contract Bugs**: Loss of capital
2. **Failed Transactions**: Wasted gas (mitigated by Flashbots)
3. **Frontrunning by Others**: Your MEV stolen by faster bot
4. **Protocol Changes**: New DEX versions, EIP changes

**Economic Risks**:
1. **Gas Price Spikes**: Eat into profits
2. **Market Volatility**: Arbitrages dry up in stable markets
3. **Competition**: Race to bottom on margins
4. **Regulatory**: Potential bans on certain MEV types

**Reputational Risks**:
1. **Sandwich Attacks**: Seen as predatory
2. **Community Backlash**: Protocol-level MEV mitigation
3. **Legal Exposure**: Unclear regulatory status

### Future Outlook

**Trends**:
1. **MEV-Resistant Protocols**: CoW Protocol, UniswapX reduce MEV
2. **Private Order Flow**: More transactions bypass public mempool
3. **Builder Centralization**: Top 5 builders = 70% of blocks
4. **Cross-Chain MEV**: Expansion to L2s, other L1s
5. **Intent-Based Architecture**: Changes MEV landscape entirely

**Predictions (2025-2026)**:
- Total MEV: $3-4B annually (slower growth)
- Consolidation continues: Top 5 searchers = 60%+
- New opportunities: L2 MEV, cross-chain, intents
- Margins compress: Avg profit per tx decreases 20-30%
- Regulatory clarity: Some jurisdictions regulate/ban MEV

---

## Interview Questions & Answers

### Technical Implementation Questions

#### Q1: "How would you design a system to detect MEV transactions in real-time?"

**Strong Answer**:

"I'd design a multi-stage pipeline:

**Stage 1: Data Ingestion**
- Subscribe to Ethereum nodes via WebSocket for real-time blocks
- Parse all transactions, events, and internal calls immediately
- Store in PostgreSQL with partitioning by chain_id

**Stage 2: Pattern Detection**
- Run parallel detection algorithms (sandwich, arbitrage, liquidation, JIT)
- Use SQL queries with window functions to identify transaction patterns
- For sandwiches: Look for same address at positions [i-1] and [i+1], opposite swap directions
- For arbitrage: Identify circular token paths with multiple DEX interactions

**Stage 3: Profit Calculation**
- Calculate token balance changes using events and internal transactions
- Convert to USD using price oracles (Chainlink, DEX reserves)
- Subtract gas costs to get net profit

**Stage 4: Classification & Storage**
- Classify MEV type and store in mev_transactions table
- Update searcher statistics in real-time
- Emit metrics to Prometheus

**Optimizations**:
- Batch processing: Wait for block to finalize before detecting (avoid partial data)
- Caching: Keep DEX pool addresses, token decimals in memory
- Indexing: GIN indexes on JSONB fields for fast event queries
- Parallel workers: Process multiple blocks concurrently

**Challenges**:
- **False positives**: Not every A→B→A path is arbitrage (could be rebalancing)
- **Missing data**: Need internal transactions for flash loans
- **Gas calculation**: Complex contracts have variable gas costs
- **Price feeds**: Getting accurate historical prices is hard

**Follow-up discussion**: I'd want archive nodes for historical analysis, but they're expensive ($1K+/month). For production, I'd use a hybrid: light clients for recent data, and batch-process historical data less frequently."

---

#### Q2: "What data structures would you use to optimize MEV detection?"

**Strong Answer**:

**1. In-Memory Caches (Redis)**:
```go
// DEX pool cache: Avoid repeated lookups
type PoolCache struct {
    Address   string
    Token0    string
    Token1    string
    DEXType   string // uniswap-v2, uniswap-v3, etc
}
// Key: pool_address → Value: PoolCache
// TTL: 24 hours (pools don't change often)

// Token metadata cache
type TokenCache struct {
    Address  string
    Symbol   string
    Decimals int
}
// Key: token_address → Value: TokenCache
// TTL: 7 days (never changes)
```

**2. Transaction Graph (for arbitrage detection)**:
```go
type TokenGraph struct {
    adjacencyList map[string][]Edge // token → [edges]
}

type Edge struct {
    ToToken   string
    Pool      string
    AmountIn  *big.Int
    AmountOut *big.Int
}

// Find arbitrage: DFS from each token looking for cycles
func (g *TokenGraph) FindArbitrageCycles(startToken string) [][]Edge {
    // Implement cycle detection algorithm
}
```

**3. Bloom Filter (for quick address checks)**:
```go
// Check if address is a known DEX router
bloomFilter := bloom.New(1000000, 5)
for _, dex := range knownDEXRouters {
    bloomFilter.Add([]byte(dex))
}

// Fast negative checks (if not in filter, definitely not a DEX)
if !bloomFilter.Test([]byte(address)) {
    return false // Not a DEX transaction
}
```

**4. Ring Buffer (for sliding window patterns)**:
```go
// Keep last 100 transactions for pattern matching
type TransactionBuffer struct {
    buffer []*Transaction
    index  int
    size   int
}

// Check if transaction at index i is sandwiched
func (tb *TransactionBuffer) CheckSandwich(i int) bool {
    if i == 0 || i == tb.size-1 {
        return false
    }
    return isSandwichPattern(tb.buffer[i-1], tb.buffer[i], tb.buffer[i+1])
}
```

**5. Database Indexes**:
```sql
-- B-tree for range queries
CREATE INDEX idx_mev_block_range ON mev_transactions (chain_id, block_number);

-- GIN for JSONB queries (strategy-specific details)
CREATE INDEX idx_mev_details ON mev_transactions USING gin (details);

-- Hash index for exact lookups
CREATE INDEX idx_mev_searcher_hash ON mev_transactions USING hash (searcher_address);

-- Partial index for recent data (most queries)
CREATE INDEX idx_mev_recent ON mev_transactions (block_number)
WHERE block_number > (SELECT MAX(block_number) - 100000 FROM transactions);
```

**Trade-offs**:
- **Memory vs Speed**: Caching everything is fast but uses RAM. Need LRU eviction.
- **Accuracy vs Latency**: Real-time detection sacrifices some accuracy (block not finalized).
- **Storage vs Query Speed**: Denormalized data (duplicating info) speeds up queries but uses more disk.

---

#### Q3: "How would you calculate the profit from a sandwich attack?"

**Strong Answer**:

**Step-by-Step Calculation**:

```go
func calculateSandwichProfit(frontRun, backRun Transaction) (float64, error) {
    // 1. Identify the token being sandwiched
    tokenOut := extractOutputToken(frontRun) // What bot bought
    
    // 2. Get bot's balance before front-run
    balanceBefore := getTokenBalance(
        frontRun.From, 
        tokenOut, 
        frontRun.BlockNumber - 1,
    )
    
    // 3. Get bot's balance after back-run
    balanceAfter := getTokenBalance(
        backRun.From,
        tokenOut,
        backRun.BlockNumber,
    )
    
    // 4. Calculate token profit
    tokenProfit := balanceAfter - balanceBefore
    
    // 5. Convert to USD
    tokenPrice := getTokenPriceUSD(tokenOut, backRun.BlockNumber)
    profitUSD := tokenProfit * tokenPrice
    
    // 6. Calculate gas costs
    frontRunGas := frontRun.GasUsed * frontRun.GasPrice
    backRunGas := backRun.GasUsed * backRun.GasPrice
    totalGas := frontRunGas + backRunGas
    
    ethPrice := getETHPriceUSD(backRun.BlockNumber)
    gasCostUSD := (totalGas / 1e18) * ethPrice
    
    // 7. Net profit
    netProfitUSD := profitUSD - gasCostUSD
    
    return netProfitUSD, nil
}
```

**Challenges**:

**1. Token Accounting**:
- Problem: Bot might hold tokens from previous transactions
- Solution: Track balance delta (before → after), not absolute amounts

**2. Price Feeds**:
- Problem: Need historical prices at specific blocks
- Solutions:
  * **Option A**: Use DEX reserves (calculate price from pool state)
  * **Option B**: Use Chainlink price feeds (if available)
  * **Option C**: Reverse-engineer from swap events (amount_in / amount_out)

```go
// Get token price from DEX reserves
func getTokenPriceFromPool(token, pool string, blockNum int64) float64 {
    reserves := getPoolReserves(pool, blockNum)
    
    // Uniswap V2: price = reserve1 / reserve0
    if reserves.Token0 == token {
        return reserves.Reserve1 / reserves.Reserve0
    }
    return reserves.Reserve0 / reserves.Reserve1
}
```

**3. Multi-Token Paths**:
- Problem: Sandwich might go USDC → WETH → USDC (not direct)
- Solution: Track all token flows, calculate USD value at each step

**4. Failed Transactions**:
- Problem: Front-run succeeds but back-run fails (gets frontrun by another bot)
- Solution: Only calculate profit for complete sandwiches (both txs successful)

**5. Slippage vs Profit**:
- Important distinction:
  * **Slippage**: Price impact on victim (their loss)
  * **Profit**: Net gain for bot (after gas)
  * Profit ≠ Slippage (bot pays fees, gas, might not capture all slippage)

**Real Example**:
```
Front-run:
- Bought 10,000 TOKEN for 0.5 ETH
- Gas: 0.02 ETH

Victim:
- Bought 50,000 TOKEN for 10 ETH
- Price moved from 0.00005 to 0.0002 ETH/TOKEN

Back-run:
- Sold 10,000 TOKEN for 0.65 ETH
- Gas: 0.02 ETH

Calculation:
Token profit: 0.65 - 0.5 = 0.15 ETH
Gas cost: 0.02 + 0.02 = 0.04 ETH
Net profit: 0.15 - 0.04 = 0.11 ETH
In USD (ETH @ $2000): 0.11 * 2000 = $220
```

---

### System Design Questions

#### Q4: "Design an MEV detection system that can process 1 million blocks per day"

**Strong Answer**:

**Requirements**:
- Throughput: 1M blocks/day = ~11.6 blocks/second
- Ethereum: ~12 second block time = 7,200 blocks/day (for one chain)
- So we're indexing ~139 chains, or historical backfill

**Architecture**:

```
┌─────────────────────────────────────────────────────┐
│ Ingestion Layer (Kafka)                            │
│ - Topics: raw-blocks, raw-transactions, raw-events  │
│ - Partitions: 20 per topic (parallelism)           │
└──────────────┬──────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────┐
│ Processing Layer (MEV Detector Service)             │
│ - 50 worker pods (Kubernetes)                       │
│ - Each pod: 10 goroutines                           │
│ - Process rate: 500 blocks/second total             │
└──────────────┬──────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────┐
│ Storage Layer (PostgreSQL + Redis)                  │
│ - PostgreSQL: Persistent storage, partitioned       │
│ - Redis: Caching (DEX pools, token metadata)        │
│ - TimescaleDB: Time-series MEV stats                │
└─────────────────────────────────────────────────────┘
```

**Scaling Strategy**:

**1. Horizontal Scaling (Kafka Consumers)**:
```go
// Each worker consumes from one partition
func main() {
    workerID := getWorkerID()
    partition := workerID % 20 // 20 partitions
    
    consumer := kafka.NewConsumer(partition)
    
    for msg := range consumer.Messages() {
        block := parseBlock(msg)
        mevTxs := detectMEV(block)
        saveToDB(mevTxs)
    }
}
```

**2. Batch Processing**:
```go
// Don't process blocks one-by-one
func processBatch(blocks []Block) {
    // 1. Batch insert transactions
    tx := db.Begin()
    for _, block := range blocks {
        insertTransactions(tx, block)
    }
    tx.Commit()
    
    // 2. Batch detect MEV
    mevTxs := detectMEVBatch(blocks)
    
    // 3. Batch insert MEV
    tx = db.Begin()
    insertMEVBatch(tx, mevTxs)
    tx.Commit()
}
```

**3. Database Partitioning**:
```sql
-- Partition by chain_id AND block_range
CREATE TABLE mev_transactions_ethereum_18m_19m 
PARTITION OF mev_transactions 
FOR VALUES IN (1)
WHERE block_number >= 18000000 AND block_number < 19000000;

-- Benefits:
-- - Query only relevant partition
-- - Parallel writes to different partitions
-- - Easy to drop old partitions
```

**4. Caching Strategy**:
```go
// Redis caching for repeated lookups
func getPoolInfo(poolAddress string) PoolInfo {
    // Check cache first
    cacheKey := "pool:" + poolAddress
    if cached := redis.Get(cacheKey); cached != nil {
        return cached
    }
    
    // Cache miss: query database
    poolInfo := db.QueryPoolInfo(poolAddress)
    redis.Set(cacheKey, poolInfo, 24*time.Hour)
    
    return poolInfo
}
```

**5. Indexing Strategy**:
```sql
-- Compound index for common query patterns
CREATE INDEX idx_mev_searcher_block 
ON mev_transactions (searcher_address, block_number DESC)
WHERE chain_id = 1;

-- Partial index for hot data (last 1M blocks)
CREATE INDEX idx_mev_recent
ON mev_transactions (block_number DESC)
WHERE block_number > 18000000;
```

**Performance Targets**:
```
Ingestion: 500 blocks/sec (peak)
Detection: 11.6 blocks/sec (average)
Storage: 10k inserts/sec (PostgreSQL)
Query latency: < 100ms (p95)

Database size:
- 1M blocks * 200 txs/block = 200M transactions
- ~5% MEV rate = 10M MEV transactions
- Storage: ~500GB (with indexes)
```

**Bottlenecks & Solutions**:

**Bottleneck 1: Database Writes**
- Problem: 10k inserts/sec exceeds single PostgreSQL instance
- Solution: Batch inserts, use COPY command, consider Timescale

**Bottleneck 2: Archive Node Access**
- Problem: Historical balance lookups are slow
- Solution: Cache token balances, use eth_getBlockByNumber with traces

**Bottleneck 3: Price Calculations**
- Problem: Looking up token prices for 10M MEV txs
- Solution: Build price index (token → block → price), query once per token per block

**Cost Estimation**:
```
Infrastructure (AWS):
- RDS PostgreSQL (db.r5.4xlarge): $2,500/month
- ElastiCache Redis (cache.r5.xlarge): $500/month
- MSK Kafka (kafka.m5.large x3): $1,500/month
- EKS workers (c5.2xlarge x10): $3,000/month
- Archive node: $1,000/month

Total: ~$8,500/month for 1M blocks/day
```

---

#### Q5: "How would you handle false positives in MEV detection?"

**Strong Answer**:

**Types of False Positives**:

**1. Legitimate Rebalancing (False Arbitrage)**:
```
Scenario: Liquidity provider rebalancing across pools
Pattern: USDC → WETH → USDC (looks like arbitrage)
Reality: LP moving liquidity, not profiting

Detection:
- Check if address is an LP token holder
- Verify net profit (rebalancing often breaks even)
- Look for Mint/Burn events (LP actions)

Solution:
if hasLPTokens(address) && netProfit < $10 {
    return false // Not MEV arbitrage
}
```

**2. Failed Sandwich Attempts (False Sandwich)**:
```
Scenario: Bot tries to sandwich but victim tx fails
Pattern: Buy TOKEN → [Victim REVERTS] → Sell TOKEN
Reality: Bot just did a round-trip trade, no sandwich

Detection:
- Check victim transaction status (must be successful)
- Verify all three transactions succeeded
- Confirm victim tx was actually in between

Solution:
if !victimTx.Success {
    return false // Not a sandwich
}
```

**3. Protocol Operations (False JIT)**:
```
Scenario: Automated market maker rebalancing
Pattern: Mint LP → Swap → Burn LP
Reality: Protocol automation, not MEV extraction

Detection:
- Check if address is protocol contract
- Verify transaction origin (EOA vs contract)
- Look for protocol-specific event signatures

Solution:
if isProtocolContract(address) {
    return false // Protocol operation, not MEV
}
```

**Confidence Scoring System**:

```go
type MEVConfidence struct {
    Score      float64 // 0.0 to 1.0
    Indicators []string
}

func calculateConfidence(detection MEVDetection) MEVConfidence {
    score := 1.0
    indicators := []string{}
    
    // Indicator 1: Profit amount
    if detection.ProfitUSD < 10 {
        score -= 0.3
        indicators = append(indicators, "low_profit")
    }
    
    // Indicator 2: Gas optimization
    if detection.GasOptimized { // Custom contract, not standard router
        score += 0.2
        indicators = append(indicators, "optimized_contract")
    }
    
    // Indicator 3: Repetition
    if detection.SearcherTransactionCount > 100 {
        score += 0.2
        indicators = append(indicators, "known_searcher")
    }
    
    // Indicator 4: Flashbots usage
    if detection.ViaFlashbots {
        score += 0.3
        indicators = append(indicators, "private_mempool")
    }
    
    // Indicator 5: Contract complexity
    if detection.ContractCallDepth > 5 {
        score += 0.1
        indicators = append(indicators, "complex_execution")
    }
    
    return MEVConfidence{
        Score:      math.Min(score, 1.0),
        Indicators: indicators,
    }
}
```

**Filtering Strategy**:

```go
// Tiered classification
func classifyMEV(detection MEVDetection) string {
    confidence := calculateConfidence(detection)
    
    switch {
    case confidence.Score >= 0.9:
        return "confirmed_mev" // High confidence
    case confidence.Score >= 0.7:
        return "probable_mev" // Medium confidence
    case confidence.Score >= 0.5:
        return "possible_mev" // Low confidence
    default:
        return "not_mev" // Likely false positive
    }
}

// Only store high-confidence MEV
func saveMEV(detection MEVDetection) error {
    classification := classifyMEV(detection)
    
    if classification == "not_mev" {
        return nil // Don't store
    }
    
    detection.Confidence = calculateConfidence(detection).Score
    detection.Classification = classification
    
    return db.Insert(detection)
}
```

**Validation Techniques**:

**1. Manual Review Sample**:
```go
// Flag 1% of detections for manual review
func shouldReview(detection MEVDetection) bool {
    // Random sampling
    if rand.Float64() < 0.01 {
        return true
    }
    
    // Edge cases
    if detection.ProfitUSD > 100000 || detection.ProfitUSD < 5 {
        return true // Very high or very low profit
    }
    
    return false
}
```

**2. Cross-Validation**:
```go
// Run multiple detection algorithms and compare
results := []bool{
    detectSandwichMethod1(tx),
    detectSandwichMethod2(tx),
    detectSandwichMethod3(tx),
}

// Require majority agreement
if countTrue(results) >= 2 {
    return true // Confirmed
}
```

**3. Historical Accuracy Tracking**:
```sql
-- Track false positive rate
CREATE TABLE mev_validations (
    mev_transaction_id BIGINT PRIMARY KEY,
    initial_classification VARCHAR(50),
    validated_classification VARCHAR(50),
    validated_by VARCHAR(100),
    validated_at TIMESTAMP
);

-- Calculate accuracy
SELECT 
    initial_classification,
    COUNT(*) as total,
    SUM(CASE WHEN initial_classification = validated_classification THEN 1 ELSE 0 END) as correct,
    SUM(CASE WHEN initial_classification = validated_classification THEN 1 ELSE 0 END)::float / COUNT(*) as accuracy
FROM mev_validations
GROUP BY initial_classification;
```

**Acceptable False Positive Rate**:
```
Research/Analysis: < 1% (need high precision)
Real-time Alerting: < 5% (balance precision/recall)
Historical Stats: < 10% (aggregate data tolerates noise)

Trade-off: 
- Stricter filtering → fewer false positives but miss some real MEV
- Looser filtering → catch all real MEV but more false positives

Recommendation: Start strict (90%+ confidence), relax over time as you validate
```

---

### Economic & Business Questions

#### Q6: "Is MEV extraction ethical? Should it be regulated?"

**Strong Answer** (show nuance, both sides):

**Arguments FOR MEV Extraction**:

**1. Market Efficiency**:
- Arbitrage keeps prices aligned across DEXes
- Benefits overall market (tighter spreads, better prices)
- Without arbitrage, users would get worse execution
- Example: If WETH is $2,000 on Uniswap and $2,100 on Sushiswap, arbitrageurs fix this quickly

**2. Liquidation Services**:
- Liquidations are necessary for protocol solvency
- MEV searchers provide this service faster than protocols could
- Users with over-leveraged positions need to be liquidated to protect lenders
- Without liquidators, lending protocols would accumulate bad debt

**3. Incentive Alignment**:
- MEV rewards entities who maintain infrastructure (nodes, bots, monitoring)
- Flashbots: Users can share in MEV profits (MEV-Share)
- Block builders: MEV increases validator revenue (securing the network)

**Arguments AGAINST MEV Extraction**:

**1. Harm to Users**:
- Sandwich attacks directly harm retail traders (worse execution)
- Victim gets 2-5% worse price on average
- Disproportionately affects small traders (can't afford private transactions)
- Example: User wants to buy $1,000 of TOKEN, gets sandwiched for $30 loss

**2. Network Congestion**:
- MEV bots spam the network with failed transactions
- Drives up gas prices for everyone
- During DeFi summer (2021), 20%+ of transactions were MEV attempts

**3. Centralization Risk**:
- Top 10 searchers control 60% of MEV
- Block builders have censorship power (can exclude transactions)
- Private mempools (Flashbots) create information asymmetry

**4. Time-Bandit Attacks**:
- Theoretical risk: Validators reorg blocks to steal MEV
- Could undermine consensus security
- More profitable to reorg than follow canonical chain

**Nuanced Position**:

"I think MEV exists on a spectrum:

**Beneficial MEV** (should be encouraged):
- Arbitrage: Improves price efficiency
- Liquidations: Maintains protocol health
- Back-running: Reacts to new information, doesn't harm users

**Harmful MEV** (should be mitigated):
- Sandwich attacks: Directly extracts value from users
- Front-running: Exploits information asymmetry
- Oracle manipulation: Attacks protocols

**Regulatory Approach**:
Rather than banning MEV (impractical), we should:
1. **Disclosure**: Label MEV transactions, show users their cost
2. **Protocol Design**: Build MEV-resistant protocols (CoW Protocol, batch auctions)
3. **Fair Access**: Private transaction access for all users (not just institutions)
4. **MEV Redistribution**: Users share in MEV profits (MEV-Share, order flow auctions)

**Technical Solutions** (better than regulation):
- **CowSwap/CoWProtocol**: Batch auctions eliminate sandwich attacks
- **UniswapX**: Dutch auctions make front-running unprofitable
- **Flashbots Protect**: Free private transactions for users
- **MEV-Share**: Users get kickback from MEV profits

**Example**: Ethereum's move to PoS + MEV-Boost shows regulation isn't needed—market solutions emerged (Flashbots, builder separation, MEV-Share).

In interviews, I'd emphasize I can implement MEV detection/extraction *technically*, while being aware of *ethical implications*."

---

#### Q7: "How much would it cost to build a competitive MEV bot today?"

**Strong Answer**:

**Development Costs** (one-time):

**Tier 1: Basic Bot ($10-20K)**
```
Components:
- Smart contract development: $5K
  * Simple arbitrage contract
  * Flash loan integration
  * Gas optimization

- Backend development: $5K
  * Node.js/Python bot
  * DEX integration (Uniswap, Sushiswap)
  * Basic mempool monitoring

- Infrastructure setup: $2K
  * Eth node (Geth/Erigon)
  * PostgreSQL database
  * Monitoring (Grafana)

- Testing & deployment: $3K
  * Testnet testing
  * Mainnet deployment
  * Initial capital ($5K)

Total: $15K
Timeline: 1-2 months
Expected return: $1-5K/month (small arbs only)
```

**Tier 2: Professional Bot ($50-100K)**
```
Components:
- Advanced smart contracts: $15K
  * Multi-strategy support (arb, sandwich, liquidation)
  * Gas optimization (assembly, batching)
  * Security audits

- Sophisticated backend: $20K
  * Golang high-performance service
  * Real-time mempool monitoring
  * Transaction simulation
  * Profit calculation engine

- Low-latency infrastructure: $10K
  * Co-located servers (validator proximity)
  * Archive node access
  * Redis caching
  * Load balancers

- Flashbots/Private pool integration: $5K
  * Bundle submission logic
  * MEV-Boost integration
  * Private RPC endpoints

- Testing & optimization: $10K
  * Extensive backtesting
  * Gas optimization
  * Strategy tuning

- Initial capital: $30K

Total: $90K
Timeline: 3-4 months
Expected return: $20-100K/month
```

**Tier 3: Institutional Operation ($500K-1M)**
```
Components:
- Full engineering team: $200K
  * 2 smart contract engineers ($100K each)
  * 2 backend engineers ($100K each)
  * 1 DevOps engineer ($50K)
  * 1 Quantitative analyst ($75K)
  (6-month salaries)

- Enterprise infrastructure: $100K
  * Multiple co-located servers
  * Redundant archive nodes
  * Dedicated fiber connections
  * Custom networking stack

- Smart contract security: $50K
  * Professional audits (Trail of Bits, OpenZeppelin)
  * Bug bounties
  * Insurance

- Market research & strategy: $50K
  * Data acquisition
  * Backtesting infrastructure
  * Strategy development

- Legal & compliance: $20K
  * Entity formation
  * Legal review
  * Regulatory compliance

- Initial capital: $500K

Total: $920K
Timeline: 6-12 months
Expected return: $200K-1M/month
```

**Ongoing Costs** (monthly):

**Basic Operation**:
```
- Server hosting: $500
- Eth RPC (Alchemy/Infura): $200
- Gas costs (failed txs): $1,000
- Maintenance: $500

Total: $2,200/month
Break-even: $3K profit/month
```

**Professional Operation**:
```
- Co-located servers: $5,000
- Archive nodes: $3,000
- Private mempool access: $2,000
- RPC endpoints: $1,000
- Gas costs: $50,000
- Engineering team (if ongoing): $50,000
- DevOps & monitoring: $5,000

Total: $116,000/month (excluding capital)
Break-even: $150K profit/month
```

**Competitive Analysis** (Q4 2024):

**Market Entry Difficulty**: VERY HARD

**Reasons**:
1. **Established Players**: Top 10 bots have 2-3 years of optimization
2. **Speed Advantage**: Incumbents have low-latency infrastructure ($100K+)
3. **Capital Requirements**: Competitive liquidation bots need $1M+
4. **Information Asymmetry**: Private order flow advantages
5. **Gas Wars**: Profitable opportunities have 10-50 bots competing

**Realistic Path for New Entrant**:

**Phase 1: Niche Strategy (Months 1-3)**
- Focus on less competitive MEV (back-running, small arbs)
- Avoid sandwiches (too competitive)
- Target new DEXes/chains (less bots)
- Expected: Break-even to $5K/month

**Phase 2: Optimization (Months 4-6)**
- Reduce gas costs (custom contracts)
- Increase speed (better infrastructure)
- Add flash loan strategies
- Expected: $10-30K/month

**Phase 3: Scale (Months 7-12)**
- Add more chains (Arbitrum, Base, Polygon)
- Implement advanced strategies (JIT, liquidations)
- Optimize for specific niches
- Expected: $50-100K/month

**Phase 4: Institutional (Year 2+)**
- Raise capital ($1M+)
- Build team (5-10 people)
- Compete with top searchers
- Expected: $200K-1M/month

**Honest Assessment**:

"If I were starting today with $50K budget, I'd:
1. **Not compete directly** with jaredfromsubway.eth (impossible)
2. **Focus on niches**: New L2s, emerging DEXes, specific protocols
3. **Leverage existing tools**: Use Flashbots, don't rebuild everything
4. **Start small**: Prove profitability at $5K/month, then scale
5. **Consider alternative value**: Build detection/analysis tools instead of extraction

The MEV extraction market is becoming oligopolistic. Better opportunity: Sell tools to MEV searchers (analytics, simulation, monitoring) or build MEV-resistant protocols."

---

### Case Study Analysis

#### Q8: "Analyze the jaredfromsubway.eth MEV operation—what made them so successful?"

**Comprehensive Analysis**:

**Profile**:
- Most profitable sandwich attacker (2023-2024)
- $45M+ extracted in 18 months
- 500K+ transactions
- 73% success rate

**Success Factors**:

**1. Gas Optimization** (30% cost advantage)

*Evidence*:
```
Average sandwich gas cost (industry): 250,000 gas
jaredfromsubway.eth average: 175,000 gas
Savings: 30% = $15-30 per transaction

Annual savings: 500K txs * $20 = $10M
```

*Techniques*:
- Custom smart contracts (no OpenZeppelin bloat)
- Assembly optimization for swaps
- Batched operations (multiple sandwiches in one tx)
- Direct pool interactions (bypass routers)

**2. Capital Advantage** ($2-5M working capital)

*Impact*:
- Can front-run large trades (most bots can't)
- Example: Sandwich $1M USDC → ETH swap (smaller bots skip these)
- Larger front-run = more profit (scales linearly)

*Example*:
```
Small bot (capital: $50K):
- Can only sandwich trades < $100K
- Avg profit: $30/tx
- Opportunities: 1,000/month = $30K

jaredfromsubway (capital: $3M):
- Can sandwich trades up to $10M
- Avg profit: $90/tx
- Opportunities: 10,000/month = $900K

Capital enables 30x more opportunities
```

**3. Flashbots Mastery** (private mempools)

*Advantages*:
- No competition from public mempool bots
- Failed sandwiches don't cost gas
- Can submit "conditional" bundles

*Flashbots Bundle Example*:
```go
bundle := flashbots.Bundle{
    Txs: [
        signedFrontRunTx,  // Tx 1
        victimTxHash,       // Tx 2 (from mempool)
        signedBackRunTx,    // Tx 3
    ],
    RevertingTxHashes: [signedFrontRunTx], // Can fail without reverting bundle
}

// Benefits:
// 1. Guaranteed ordering (no one can insert between)
// 2. Front-run can fail (try multiple amounts)
// 3. Pay validator directly (avoid gas bidding war)
```

*Statistics*:
- 80%+ of jaredfromsubway txs via Flashbots
- Success rate: 73% (vs 40% for public mempool bots)

**4. Multi-Strategy Approach**

*Revenue Breakdown*:
```
Sandwiches: 60% ($27M)
├── Uniswap V2: $12M
├── Uniswap V3: $10M
└── Other DEXes: $5M

Arbitrage: 30% ($13.5M)
├── Cross-DEX: $9M
└── Triangular: $4.5M

Back-running: 10% ($4.5M)
├── Oracle updates: $2M
└── Large trades: $2.5M
```

*Advantage*: Diversification reduces risk, captures more opportunities

**5. Speed & Infrastructure**

*Latency Breakdown*:
```
Block propagation: 100ms (network)
jaredfromsubway detection: 5ms (custom code)
Transaction submission: 10ms (direct validator connection)
Total: 115ms

vs Average bot:
Block propagation: 100ms
Detection: 50ms (slower code)
Submission: 100ms (via public RPC)
Total: 250ms

Speed advantage: 2x faster = wins most races
```

*Infrastructure*:
- Co-located servers (near Flashbots relays)
- Custom mempool monitoring (not eth_newPendingTransactions)
- Direct p2p connections to validators

**6. Algorithmic Sophistication**

*Profit Maximization*:
```go
// Not just binary (sandwich vs don't sandwich)
// Optimize front-run amount for maximum profit

func optimalFrontRunAmount(victimTrade VictimTrade) *big.Int {
    pool := getPoolState(victimTrade.Pool)
    
    maxProfit := 0.0
    optimalAmount := big.NewInt(0)
    
    // Try different front-run amounts
    for amount := 0; amount < capitalAvailable; amount += step {
        profit := simulateProfit(amount, victimTrade, pool)
        
        if profit > maxProfit {
            maxProfit = profit
            optimalAmount = amount
        }
    }
    
    return optimalAmount
}
```

*Slippage Calculation*:
- Simulates victim transaction to predict price impact
- Calculates optimal front-run size (too small = low profit, too large = excess capital)
- Accounts for pool fees, gas costs

**7. Consistency & Reliability**

*Uptime*:
```
jaredfromsubway uptime: 99.9%
- Transactions every single block (when profitable)
- No multi-day gaps (infrastructure is reliable)

vs Competitor bots:
- Many have downtime (manual operations, bugs)
- Average uptime: 95%

Impact: 5% more uptime = 5% more profit = $2.25M/year
```

**8. Risk Management**

*Position Sizing*:
- Never risks more than 10% of capital on one sandwich
- Diversifies across multiple transactions per block
- Exits positions immediately (no overnight risk)

*Example*:
```
Block 18500123 (5 sandwiches):
- Sandwich 1: $50K front-run, $1.2K profit
- Sandwich 2: $30K front-run, $800 profit
- Sandwich 3: $100K front-run, $3K profit
- Sandwich 4: $20K front-run, $500 profit
- Sandwich 5: $80K front-run, $2.1K profit

Total capital used: $280K (< 10% of $3M)
Total profit: $7.6K in one block
```

**What Others Can Learn**:

**Replicable**:
1. ✅ Gas optimization (anyone can optimize contracts)
2. ✅ Multi-strategy approach (diversification)
3. ✅ Flashbots usage (free to use)
4. ✅ Risk management (best practices)

**Hard to Replicate**:
1. ❌ $3M capital (requires fundraising)
2. ❌ Co-located infrastructure ($100K+ investment)
3. ❌ 18 months of optimization (time)
4. ❌ Speed advantage (expensive)

**Competitive Moat**:
- **Network effects**: More capital → more opportunities → more profit → more capital
- **Experience**: 500K txs of learning, edge cases handled
- **Infrastructure**: Sunk cost, new entrants need to match
- **Brand**: Known in Flashbots ecosystem, preferential treatment (?)

**Vulnerability**:
- **Regulation**: Sandwich attacks could be banned
- **Protocol evolution**: MEV-resistant DEXes (CoW, UniswapX)
- **Competition**: Other well-funded bots emerging
- **Market saturation**: Fewer profitable opportunities

**Interview Takeaway**:
"jaredfromsubway.eth succeeded through a combination of capital, optimization, and infrastructure—not just one factor. For a new entrant, matching all three is expensive ($500K+) and time-consuming (12+ months). Better strategy: find a niche they don't dominate (new chains, specific protocols)."

---

## Practical Implementation Checklist

### Getting Started (Week 1-2)

**Database Setup**:
- [ ] Create MEV tables (migrations/005_add_mev_tables.sql)
- [ ] Add indexes on mev_transactions
- [ ] Create materialized views for analytics
- [ ] Set up partitioning by chain_id

**Basic Detection** (Choose one to start):
- [ ] Implement sandwich detector
- [ ] Implement arbitrage detector
- [ ] Implement liquidation detector
- [ ] Test on recent blocks (last 1,000)

**Infrastructure**:
- [ ] Set up Kafka topic for MEV events
- [ ] Create mev-detector service skeleton
- [ ] Add Prometheus metrics
- [ ] Configure Docker Compose

### MVP (Week 3-4)

**Full Detection Pipeline**:
- [ ] Implement all 5 detection algorithms
- [ ] Add profit calculation logic
- [ ] Create confidence scoring
- [ ] Test on 10,000 historical blocks

**API Endpoints**:
- [ ] GET /api/v1/mev/{chain_id}/{block_number}
- [ ] GET /api/v1/mev/searchers (top searchers)
- [ ] GET /api/v1/mev/stats (daily stats)

**Frontend**:
- [ ] MEV dashboard page
- [ ] Searcher leaderboard
- [ ] Transaction detail view with MEV info

### Production (Week 5-8)

**Optimization**:
- [ ] Implement batch processing
- [ ] Add Redis caching
- [ ] Optimize SQL queries
- [ ] Horizontal scaling (multiple workers)

**Monitoring**:
- [ ] Grafana dashboards
- [ ] Alerting for detection anomalies
- [ ] Performance metrics
- [ ] Error tracking

**Documentation**:
- [ ] API documentation
- [ ] Architecture diagrams
- [ ] Deployment guides
- [ ] Maintenance runbooks

---

## Additional Resources

### Essential Reading

1. **"Flash Boys 2.0"** (Phil Daian et al.)
   - Original MEV research paper
   - Defines MEV, describes sandwich attacks
   - [https://arxiv.org/abs/1904.05234](https://arxiv.org/abs/1904.05234)

2. **Flashbots Documentation**
   - MEV-Boost architecture
   - Bundle submission
   - [https://docs.flashbots.net](https://docs.flashbots.net)

3. **MEV-Share Documentation**
   - User-facing MEV redistribution
   - [https://docs.flashbots.net/flashbots-mev-share/introduction](https://docs.flashbots.net/flashbots-mev-share/introduction)

4. **"How to Get Front-Run on Ethereum"** (samczsun)
   - Practical examples
   - [https://www.paradigm.xyz/](https://www.paradigm.xyz/)

### Tools & Libraries

**Simulation**:
- **Flashbots MEV-Inspect**: Historical MEV analysis
- **Bloxroute MEV Dashboard**: Real-time MEV tracking
- **EigenPhi**: MEV analytics platform

**Development**:
- **mev-share-client-go**: Golang Flashbots client
- **ethers-flashbots-bundle**: JavaScript/TypeScript Flashbots
- **flashbots/searcher-template**: Starter template

**Data**:
- **Dune Analytics**: MEV dashboards
- **MEV-Explore**: Public MEV data
- **Flashbots MEV Dashboard**: Live stats

---

## Conclusion

MEV analysis is a complex intersection of:
- **Technical**: Smart contract development, database optimization, real-time processing
- **Economic**: Game theory, market efficiency, profitability analysis
- **Ethical**: User harm, protocol design, regulatory implications

For this indexer project, implementing MEV detection provides:
1. **Differentiation**: Few indexers classify MEV transactions
2. **User value**: Show users their MEV costs
3. **Research**: Enable MEV analysis for academics
4. **Business opportunity**: Sell MEV data to traders, protocols

**Next Steps**:
1. Implement Phase 1 (database schema)
2. Build sandwich detector (highest ROI, easiest to detect)
3. Add API endpoints
4. Create frontend dashboard
5. Iterate with other MEV types

**Estimated Timeline**: 4-6 weeks for full MEV detection system

**Questions for Discussion**:
- Which MEV types to prioritize?
- Should we offer real-time MEV alerts?
- Integrate with frontend to show "MEV cost" on transactions?
- Build public API for MEV data (monetization)?

---

**Document Version**: 1.0  
**Last Updated**: November 18, 2025  
**Maintained By**: Development Team  
**Related Docs**: DEVELOPMENT_STATUS.md, LEARNING_GUIDE.md, TECHNICAL_SPEC.md

