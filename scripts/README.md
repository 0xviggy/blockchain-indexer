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

The script connects to Ethereum mainnet and:
- ✅ Fetches latest block info (validates block schema)
- ✅ Explores recent transactions (validates transaction schema)
- ✅ Extracts function signatures (validates our protocol_signatures)
- ✅ Shows event logs (validates event schema)
- ✅ Tests internal transaction tracing (checks RPC capabilities)

**Duration**: ~10-30 seconds depending on RPC provider

## What to Look For

### ✅ Validate Function Signatures
- Confirm 4-byte selectors match our protocol_signatures table
- Find popular functions we're missing
- Check data encoding formats

### ✅ Test Event Parsing
- Verify ERC20/ERC721 event structures
- Check indexed vs non-indexed parameters
- Confirm JSONB storage will work

### ✅ Internal Transactions
- Test which trace methods the RPC supports
- See actual internal tx structure
- Validate our internal_transactions schema

### ✅ Revert Reasons
- Test extraction on failed transactions
- Check error message formats
- Validate revert_reasons schema

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

## Findings Template

After running, document findings in LEARNING_GUIDE.md:

```markdown
## RPC Exploration Findings (Date: YYYY-MM-DD)

### Function Signatures Validated:
- ✅ Uniswap V2 swaps: 0x38ed1739
- ✅ ERC20 transfer: 0xa9059cbb
- ❌ Missing: [list any we found but don't have]

### RPC Provider Capabilities:
- ✅ Standard eth_* methods: Supported
- ✅ eth_getTransactionReceipt: Supported
- ❌ debug_traceTransaction: Not supported (requires archive node)
- ✅ trace_transaction: Supported (Erigon)

### Schema Validations:
- ✅ Block schema: All fields present
- ✅ Transaction schema: Matches receipts
- ✅ Event schema: Topics structure correct
- ⚠️  Internal txs: Need archive node or different RPC

### Action Items:
1. Add missing function signatures to migration
2. Use trace_transaction instead of debug_traceTransaction
3. Update internal_transactions extraction strategy
```

## Next Steps

Based on findings:
1. Update `002_add_calldata_parsing.sql` with any missing signatures
2. Adjust internal transaction extraction method
3. Validate revert reason extraction works
4. Proceed with confidence to implement processor service
