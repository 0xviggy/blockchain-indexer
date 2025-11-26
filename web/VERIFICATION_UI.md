# Data Verification UI - Quick Start

## What This UI Shows

This is a **development UI** for verifying data quality and monitoring blockchain indexing:

### ✅ Features
- **Block-Grouped View**: Transactions organized by block with collapsible sections
- **Transaction Status Distribution**: See success vs failed transactions
- **Gas Usage Verification**: Real gas_used values from receipts (no zeros!)
- **Transaction Limit Controls**: Choose 100, 500, or 1000 transactions to view
- **Block Range Display**: Shows which blocks are currently visible (e.g., Blocks 21000002 - 21000020)
- **Manual Refresh**: Force refresh without waiting for auto-refresh
- **Real-time Updates**: Auto-refresh every 5 seconds (can be toggled)
- **Multi-chain Support**: Switch between Ethereum, Polygon, Arbitrum, etc.
- **Performance Metrics**: Shows transaction count per block
- **Etherscan Integration**: Direct links to blocks and transactions

### 🎯 What to Look For

1. **Block Grouping**: Transactions organized under collapsible block headers
   - Click block headers to expand/collapse transaction details
   - Shows transaction count and success/fail summary per block
   
2. **Failed Transactions** (Red ✗ badges): Should show realistic failure rate
   - If all transactions show "✓ Success", receipts aren't being fetched properly
   
3. **Gas Used Column**: Should show realistic values with commas (e.g., "187,313", "21,000")
   - If showing "⚠️ 0", receipts failed to fetch or defaulted
   - NO MORE "0k" confusing display!
   
4. **Transaction Limit**: Dropdown shows 100/500/1000 options
   - Can view more historical data by increasing limit
   - Current view shows latest N transactions by block
   
5. **Performance**: Database inserts are now 300x faster (~200ms for 200 txs)
   - Batch INSERT instead of individual inserts
   - Proper receipt retry logic with backoff

## Running the UI

```bash
# Terminal 1: Start the database
cd infrastructure/docker
docker-compose up -d

# Terminal 2: Start the API
cd services/api
go run main.go

# Terminal 3: Start the ingester
cd services/ingester
ETH_RPC_URL="your-rpc-url" go run main.go

# Terminal 4: Start the frontend
cd web
npm install
npm run dev
```

Open http://localhost:5173

## Verification Checklist

- [ ] API responds at http://localhost:8000/health
- [ ] Frontend loads without errors
- [ ] Can select chains from dropdown
- [ ] Transactions populate and group by block
- [ ] Block headers are collapsible (click to expand/collapse)
- [ ] **Status badges show both ✓ (success) and ✗ (failed)**
- [ ] **Gas Used shows real values with commas (not zeros)**
- [ ] **Transaction limit selector works (100/500/1000)**
- [ ] **Block range displays correctly** (e.g., "Blocks 21000002 - 21000020")
- [ ] **Manual refresh button updates data**
- [ ] Auto-refresh updates data every 5 seconds
- [ ] Etherscan links open correctly

## Common Issues

### "Failed to load chains"
- API not running on port 8000
- Check: `curl http://localhost:8000/api/v1/chains`

### "No transactions yet"
- Ingester not running or no RPC URL configured
- Check database: `psql -U indexer -d indexer -c "SELECT COUNT(*) FROM transactions;"`

### All transactions show "Success"
- Receipt fetching not working
- Check ingester logs for "❌ Failed to get receipt" errors
- Verify RPC endpoint has archive node access

### Gas Used shows "⚠️ 0"
- Receipt fetching failed - check ingester logs
- Verify RPC endpoint is responding: `curl -X POST $ETH_RPC_URL -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'`
- See backend troubleshooting in [`/docs/DEVELOPMENT_STATUS.md`](../docs/DEVELOPMENT_STATUS.md) or [`/learning/09-troubleshooting.md`](../learning/09-troubleshooting.md)

### UI shows old "0k" values
- Clear browser cache and hard refresh (Cmd+Shift+R / Ctrl+Shift+F5)
- Check API is returning proper values: `curl "http://localhost:8000/api/v1/chains/1/transactions?limit=5"`

## Recent UI Improvements (Nov 26, 2025)

- ✅ **Block Grouping**: Transactions organized under collapsible block headers
- ✅ **Transaction Limit Selector**: Choose 100, 500, or 1000 transactions to view
- ✅ **Block Range Display**: Shows which blocks are currently visible (e.g., "Blocks 21000002 - 21000020")
- ✅ **Manual Refresh Button**: Force refresh without waiting for auto-refresh
- ✅ **Info Banner**: Clear messaging about what's being displayed
- ✅ **Better Gas Display**: Shows comma-separated numbers (e.g., "187,313") instead of confusing "0k"
- ✅ **Fixed State Bug**: Transaction limit no longer glitches back to 500

### Known Limitations
- **No Pagination**: Currently shows latest N transactions only (no "load more" or scroll back)
- **No Block Jump**: Can't jump to specific block number
- **No Time Range**: Can't filter by date/time
- **No Search**: Can't search by tx hash or address

> **Note**: Backend performance improvements (batch INSERT, retry logic, etc.) are documented in [`/docs/DEVELOPMENT_STATUS.md`](../docs/DEVELOPMENT_STATUS.md)

## Next Steps

Once verified:
- [ ] Add pagination/infinite scroll for historical data
- [ ] Add block number jump/search functionality  
- [ ] Add ingester controls (start/stop via API)
- [ ] Task 2: Implement event log parsing
- [ ] Task 3: Implement reorg detection
- [ ] Phase 4.2: Build proper production UI with charts, filters, search
