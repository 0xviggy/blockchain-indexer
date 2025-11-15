# RPC Exploration Scripts

Scripts to explore and validate blockchain RPC data before implementing the indexer.

## Setup

1. Get a free RPC API key:
   - Alchemy: https://www.alchemy.com/
   - Infura: https://infura.io/
   - QuickNode: https://www.quicknode.com/

2. Set environment variable:
   ```bash
   export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY"
   ```

3. Install dependencies:
   ```bash
   cd scripts
   go mod download
   ```

## Run Exploration

```bash
# Explore latest blocks and transactions
go run explore_rpc.go

# This will:
# - Connect to Ethereum mainnet
# - Fetch latest block info
# - Explore recent transactions
# - Extract function signatures
# - Show event logs
# - Test internal transaction tracing
```

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
