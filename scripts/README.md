# RPC Exploration Scripts

Scripts to explore and validate blockchain RPC data before implementing the indexer.

## Quick Start (2 minutes)

### 1. Get a Free API Key
Choose one provider and sign up (takes 2 minutes):
- **Alchemy** (recommended): https://www.alchemy.com/
  - Sign up → Create App → Select "Ethereum Mainnet" → Copy API Key
- **Infura**: https://infura.io/
- **QuickNode**: https://www.quicknode.com/

### 2. Set Environment Variable
```bash
# Replace YOUR_API_KEY with your actual key
export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY"
```

**Note**: You only need `ETH_RPC_URL` - no other environment variables required!

### 3. Run Exploration
```bash
# From the project root
make explore-rpc

# Or directly from scripts folder
cd scripts
go run explore_rpc.go
```

## What It Does

The script connects to configured chains and performs:

### Phase 1: Schema Validation
- ✅ Fetches latest block info (validates block schema)
- ✅ Explores sample transactions (validates transaction schema)
- ✅ Shows event logs (validates event schema)
- ✅ Tests internal transaction tracing (checks RPC capabilities)

### Phase 2: Multi-Block Signature Analysis (NEW!)
- 📊 Scans last **10 blocks** (~500-1000 transactions)
- 🔍 Extracts all function signatures from calldata
- 📈 Generates frequency statistics
- ❓ Identifies **unknown signatures** with lookup links
- 🎯 Helps discover which protocols are most active on the chain

**Duration**: ~30-60 seconds depending on RPC provider and chain activity

### Sample Output

```
📊 === SIGNATURE ANALYSIS ACROSS MULTIPLE BLOCKS ===

📈 Statistics:
  Total transactions: 616
  With calldata: 439 (71.3%)
  Unique signatures: 119
  Unknown signatures: 110

🔥 Top 10 Most Common Signatures:
  1. 0xa9059cbb - 152 calls ✅
      transfer (ERC20)
  2. 0x78e111f6 - 43 calls ❌ UNKNOWN
  3. 0x095ea7b3 - 19 calls ✅
      approve (ERC20)
  ...

❓ Unknown Signatures Found (add these to database):
-- Signature: 0x78e111f6
--   Found in: 43 transactions
--   Example tx: 0x459163ce...
--   Lookup: https://www.4byte.directory/signatures/?bytes4_signature=0x78e111f6
```

## What to Look For

### ✅ Schema Validation
- Confirm 4-byte selectors match our protocol_signatures table
- Verify ERC20/ERC721 event structures
- Validate data types handle real-world sizes (gas, value, etc.)
- Test which trace methods the RPC supports

### 🔍 Protocol Discovery (Multi-Block Analysis)
- **High-volume unknowns (>50 calls)**: Critical protocols to add immediately
- **Medium-volume (10-50 calls)**: Important protocols worth researching
- **Low-volume (1-5 calls)**: Likely proprietary contracts, can skip
- **Signature patterns**: Identify protocol families (DEX, bridges, forwarders)

### 📊 Use Cases

#### Before Launching on New Chain
```bash
export ARBITRUM_RPC_URL="https://arb1.arbitrum.io/rpc"
make explore-rpc
```
Review to understand which protocols dominate the chain.

#### Monthly Protocol Discovery
Re-scan to catch emerging protocols and update signature registry.

#### Pre-Production Validation
Verify schemas match actual blockchain data before deploying services

## Expected Output

```
✅ Connected to Ethereum RPC

📦 === LATEST BLOCK EXPLORATION ===
Block Number: 18567890
Block Hash: 0x...
Transaction Count: 145

💸 === TRANSACTION EXPLORATION ===
--- Transaction 1 ---
Hash: 0x...
Function Signature: 0x38ed1739
✅ KNOWN: swapExactTokensForTokens (Uniswap V2)
Status: 1 (success)
Logs/Events: 3
  ✅ KNOWN EVENT: Transfer (ERC20)

🔍 === INTERNAL TRANSACTION TRACING ===
⚠️  Testing trace methods...
```

## Troubleshooting

### "ETH_RPC_URL not set"
```bash
# Make sure you exported the variable in your current terminal
export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY"

# Verify it's set
echo $ETH_RPC_URL
```

### "Failed to connect to Ethereum client"
- **Check your API key**: Make sure you copied it correctly
- **Check URL format**: Should be `https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY`
- **Rate limit**: Free tier has limits, wait a minute and try again
- **Network issues**: Try `curl $ETH_RPC_URL` to test connectivity

### "Transaction not found" or "Block not found"
- The script uses block 18500000 which might be pruned
- Edit `explore_rpc.go` line 105: Change `blockNumber := big.NewInt(18500000)` to a more recent block
- Find recent blocks at: https://etherscan.io/

### Missing go-ethereum packages
```bash
cd scripts
go mod tidy
```

## Multi-Chain Support

### Adding New Chains

Edit `explore_rpc.go` and add to `SupportedChains`:

```go
var SupportedChains = []ChainConfig{
    // ... existing chains
    {
        ChainID: 56,
        Name:    "BSC",
        EnvVar:  "BSC_RPC_URL",
        ExampleRPCURL: os.Getenv("BSC_RPC_URL"),
    },
}
```

Then set the environment variable:
```bash
export BSC_RPC_URL="https://bsc-dataseed.binance.org/"
make explore-rpc
```

### Chain-Specific Sampling Strategies

**Ethereum** (high diversity):
- Sample 10-20 blocks
- Expect 100-200 unique signatures

**L2s** (Arbitrum, Optimism, Base):
- Sample 20-50 blocks (faster block times)
- Similar protocols to Ethereum

**Alt-L1s** (Polygon, BSC):
- Sample 10-20 blocks
- Expect different protocol mix (more gaming, different DEXs)

## Workflow: From Discovery to Database

### Step 1: Run Analysis
```bash
export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
make explore-rpc
```

### Step 2: Research Unknown Signatures

For high-volume unknowns (>10 calls):
1. Click the 4byte.directory link in output
2. Look up example transaction on Etherscan
3. Identify the protocol name
4. Find the ABI (Etherscan, GitHub, Sourcify)

### Step 3: Update Database

Add to `database/migrations/002_add_calldata_parsing.sql`:
```sql
INSERT INTO protocol_signatures (signature, function_name, protocol, abi, description) VALUES
    ('0x78e111f6', 'executeFFsYo', 'forwarder',
     'executeFFsYo(address,bytes)',
     'Meta-transaction forwarder (43 calls discovered)');
```

### Step 4: Update Exploration Script

Add to `scripts/explore_rpc.go`:

**In `isKnownSignature()`:**
```go
"0x78e111f6": true,
```

**In `getSignatureName()`:**
```go
"0x78e111f6": "executeFFsYo (Meta-tx Forwarder)",
```

### Step 5: Commit and Document

```bash
git add -A
git commit -m "feat: Add meta-transaction forwarder signatures

- Discovered via multi-block analysis (43 calls in sample)
- Added executeFFsYo and related forwarder functions"
git push origin main
```

Update LEARNING_GUIDE.md with findings.

## Priority Guidelines

**Volume-Based Priorities**:
- **>50 calls**: 🔴 Critical - Add immediately with full ABI
- **10-50 calls**: 🟡 High - Research and add within a week
- **5-10 calls**: 🟢 Medium - Add if easily identifiable
- **1-5 calls**: ⚪ Low - Likely proprietary, skip unless recurring

**Protocol Type Priorities**:
- **DEX aggregators** (1inch, CoW, Paraswap): High
- **Bridges** (LayerZero, Across, Stargate): High
- **Meta-tx forwarders** (Biconomy, GSN): Medium
- **Proprietary contracts**: Low

## Next Steps

Based on findings:
1. Update `002_add_calldata_parsing.sql` with any missing signatures
2. Adjust internal transaction extraction method
3. Validate revert reason extraction works
4. Proceed with confidence to implement processor service
