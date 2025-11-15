# Ingester Service

Multi-chain blockchain data ingester that fetches blocks and transactions from EVM chains and stores them in PostgreSQL.

## Features

- ✅ **Multi-chain support**: Ethereum, Polygon, Arbitrum, Optimism, Base
- ✅ **WebSocket subscriptions**: Real-time block ingestion when available
- ✅ **HTTP polling fallback**: Works with any RPC provider
- ✅ **Checkpoint management**: Resumes from last processed block
- ✅ **Catch-up mode**: Automatically syncs missed blocks
- ✅ **Graceful shutdown**: Clean stop on SIGINT/SIGTERM
- ✅ **Direct PostgreSQL writes**: Simple, reliable architecture

## Quick Start

### 1. Configure chains

```bash
# Copy environment template
cp .env.local.example .env

# Edit .env and add your RPC URLs
export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
export ETH_WS_URL="wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"  # Optional but recommended
```

### 2. Start infrastructure

```bash
# From project root
make docker-up
make migrate
```

### 3. Run ingester

```bash
# From project root
make run-ingester

# Or directly
cd services/ingester
source ../../.env
go run main.go
```

## Configuration

The ingester automatically detects which chains to index based on environment variables:

| Chain | RPC URL Variable | WebSocket Variable (optional) | Start Block Variable |
|-------|-----------------|------------------------------|---------------------|
| Ethereum | `ETH_RPC_URL` | `ETH_WS_URL` | `ETH_START_BLOCK` |
| Polygon | `POLYGON_RPC_URL` | `POLYGON_WS_URL` | `POLYGON_START_BLOCK` |
| Arbitrum | `ARBITRUM_RPC_URL` | `ARBITRUM_WS_URL` | `ARBITRUM_START_BLOCK` |
| Optimism | `OPTIMISM_RPC_URL` | `OPTIMISM_WS_URL` | `OPTIMISM_START_BLOCK` |
| Base | `BASE_RPC_URL` | `BASE_WS_URL` | `BASE_START_BLOCK` |

**Start Block**: 
- `0` (default): Start from current block (no historical data)
- `> 0`: Start from specific block number
- Checkpoint takes precedence if exists in database

## Architecture

```
┌─────────────────┐
│   RPC Nodes     │
│  (Ethereum,     │
│   Polygon, etc) │
└────────┬────────┘
         │ WebSocket / HTTP
         ▼
┌─────────────────┐
│    Ingester     │
│   (Go Service)  │
└────────┬────────┘
         │ SQL
         ▼
┌─────────────────┐
│   PostgreSQL    │
│  (Partitioned   │
│   by chain_id)  │
└─────────────────┘
```

### Flow

1. **Connection**: Connect to all configured chains via HTTP/WebSocket
2. **Chain Init**: Insert chain metadata into `chains` table
3. **Checkpoint**: Load last processed block from `checkpoints` table
4. **Catch-up**: If behind, fetch missed blocks sequentially
5. **Subscribe**: Listen for new blocks via WebSocket (or poll every 12s)
6. **Process**: For each block:
   - Insert into `blocks` table
   - Insert all transactions into `transactions` table
   - Update checkpoint
   - All in single database transaction (atomic)
7. **Repeat**: Continue until shutdown signal

## Operations

### Monitor Progress

```bash
# Check logs
make logs

# Query database
make db-shell
SELECT chain_id, last_processed_block, updated_at 
FROM checkpoints 
WHERE service_name = 'ingester';
```

### Graceful Shutdown

```bash
# Send SIGINT (Ctrl+C) or SIGTERM
# Ingester will:
#   1. Stop accepting new blocks
#   2. Finish processing current blocks
#   3. Close RPC connections
#   4. Exit cleanly
```

### Restart from Specific Block

```bash
# Update checkpoint in database
make db-shell
UPDATE checkpoints 
SET last_processed_block = 18500000 
WHERE chain_id = 1 AND service_name = 'ingester';

# Restart ingester
make run-ingester
```

### Multiple Chain Indexing

The ingester runs one goroutine per chain, processing them in parallel:

```
Ethereum:  [=====>          ] Block 23,803,500
Polygon:   [====>           ] Block 51,234,567  
Arbitrum:  [======>         ] Block 165,456,789
Optimism:  [=====>          ] Block 115,678,901
Base:      [====>           ] Block 8,901,234
```

## Troubleshooting

### "No chains configured"

Set at least one RPC URL:
```bash
export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
```

### "Chain ID mismatch"

Your RPC URL doesn't match the expected chain. Verify:
```bash
curl -X POST $ETH_RPC_URL \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
```

### "Database connection failed"

Check PostgreSQL is running:
```bash
make status
make docker-up  # If not running
```

### WebSocket subscription failed

Not all RPC providers support WebSocket. The ingester will automatically fall back to HTTP polling.

### Rate limiting

Free tier RPC providers have rate limits. Consider:
- Upgrade to paid tier
- Add delays between requests (modify polling interval)
- Use multiple RPC providers for different chains

## Performance

### Resource Usage (per chain)

- **Memory**: ~50-100 MB
- **CPU**: < 5% (mostly idle waiting for blocks)
- **Network**: ~1-5 KB/s per block
- **Database**: ~500 KB per block (depends on transaction count)

### Throughput

- **Real-time mode**: Keeps up with block production (12s Ethereum, 2s Polygon)
- **Catch-up mode**: ~10-50 blocks/second (depends on RPC rate limits)
- **Database writes**: Batched per block, typically <100ms per block

## Next Steps

1. ✅ **Ingester running** - You have blocks and transactions in database
2. 🔄 **Add API service** - Expose data via REST endpoints
3. 🎨 **Build UI** - Create block explorer interface
4. 🚀 **Add Processor** - Parse events, decode calldata, extract internal txs
5. 📊 **Add monitoring** - Prometheus metrics, Grafana dashboards

## Development

### Build Binary

```bash
make build-ingester
./bin/ingester
```

### Run Tests

```bash
cd services/ingester
go test -v ./...
```

### Code Structure

```
services/ingester/
├── main.go              # Entry point, multi-chain orchestration
├── go.mod               # Dependencies
└── README.md            # This file
```

## Known Limitations (MVP)

- ❌ No transaction receipts (status/gas_used are placeholders)
- ❌ No event log extraction
- ❌ No reorg detection/handling
- ❌ No transaction index calculation
- ❌ No Kafka message production

These will be added in Phase 2 after validating the core E2E flow with UI.
