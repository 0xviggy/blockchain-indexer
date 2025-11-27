# System Design & Architecture

> **Purpose**: Architectural decisions, design patterns, and implementation concepts for the blockchain indexer - covering reorg handling, scaling strategies, technology trade-offs, and API design with integrated interview Q&A.

---

## Table of Contents
- [Message Parsing & Transaction Analysis](#message-parsing--transaction-analysis)
- [Blockchain Reorg Handling](#blockchain-reorg-handling)
- [Event Parsing](#event-parsing)
- [Database Strategy](#database-strategy)
- [Rate Limiting](#rate-limiting)
- [Kafka Message Ordering](#kafka-message-ordering)
- [Technology Trade-offs](#technology-trade-offs)
- [API Design](#api-design)
- [Interview Questions](#interview-questions)

---

## Message Parsing & Transaction Analysis

### Overview

Advanced message parsing enables deep transaction analysis beyond basic block/transaction data:
- **Function call decoding**: Identify which smart contract functions were called
- **Internal transaction tracking**: See contract-to-contract calls within a transaction
- **Revert reason extraction**: Understand why transactions failed
- **Protocol classification**: Categorize by DeFi protocol (Uniswap, Aave, etc.)

### Internal Transactions

**What are they?** Contract-to-contract calls that happen within a transaction.

**Why track them?**
- See the full flow of funds through complex DeFi transactions
- Understand how a swap goes through multiple routers
- Track delegatecall patterns and contract interactions
- Debug complex MEV transactions

**Example**: Uniswap V2 swap transaction flow
```
1. User calls Router.swapExactTokensForTokens()
   └─ from: 0xUser
   └─ to: 0xUniswapRouter
   └─ value: 0 ETH

2. Router transfers tokens from user (CALL to ERC20)
   └─ internal_tx_index: 0
   └─ call_type: 'call'
   └─ from: 0xUniswapRouter
   └─ to: 0xUSDC (ERC20 contract)
   └─ function: transferFrom(user, pair, amount)

3. Router calls Pair.swap() (CALL)
   └─ internal_tx_index: 1
   └─ call_type: 'call'
   └─ from: 0xUniswapRouter
   └─ to: 0xUniswapPair
   └─ function: swap(amounts, to, data)

4. Pair transfers tokens to user (CALL to ERC20)
   └─ internal_tx_index: 2
   └─ call_type: 'call'
   └─ from: 0xUniswapPair
   └─ to: 0xWETH (ERC20 contract)
   └─ function: transfer(user, amountOut)
```

**Database Schema:**
```sql
CREATE TABLE internal_transactions (
    chain_id BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    internal_tx_index INT NOT NULL,
    call_type TEXT NOT NULL,  -- 'call', 'delegatecall', 'staticcall', 'create', 'create2'
    from_address TEXT NOT NULL,
    to_address TEXT,
    value NUMERIC(78, 0),
    input TEXT,
    output TEXT,
    gas_used BIGINT,
    success BOOLEAN NOT NULL,
    error TEXT,
    PRIMARY KEY (chain_id, tx_hash, internal_tx_index)
) PARTITION BY LIST (chain_id);
```

**Query Examples:**

```sql
-- Find all internal transactions for a specific transaction
SELECT 
    internal_tx_index,
    call_type,
    from_address,
    to_address,
    value,
    success
FROM internal_transactions
WHERE tx_hash = '0x123...'
ORDER BY internal_tx_index;

-- Find failed internal calls (contract reverts)
SELECT 
    tx_hash,
    internal_tx_index,
    from_address,
    to_address,
    error
FROM internal_transactions
WHERE chain_id = 1
  AND success = false
  AND block_number > 18000000;

-- Track delegatecall patterns (often used in proxies)
SELECT 
    from_address as proxy_contract,
    to_address as implementation_contract,
    COUNT(*) as delegatecall_count
FROM internal_transactions
WHERE call_type = 'delegatecall'
  AND chain_id = 1
GROUP BY from_address, to_address
ORDER BY delegatecall_count DESC;
```

### Calldata Parsing

**What is calldata?** The input data to a transaction, containing:
- Function signature (first 4 bytes) - hash of function name and parameter types
- Encoded parameters (ABI-encoded values)

**Why parse it?**
- Identify which function was called (transfer, swap, deposit, etc.)
- Extract parameters (swap amounts, token addresses, recipient addresses)
- Categorize transactions by protocol (Uniswap, Aave, LayerZero)
- Enable protocol-specific analytics and tracking

**Calldata Structure:**
```
0x38ed1739                                                         ← Function signature (4 bytes)
  0000000000000000000000000000000000000000000000000de0b6b3a7640000 ← amountIn (uint256)
  0000000000000000000000000000000000000000000000000000000000000000 ← amountOutMin (uint256)
  0000000000000000000000000000000000000000000000000000000000000080 ← path offset (uint256)
  000000000000000000000000742d35cc6634c0532925a3b844bc454e4438f44e ← to address
  0000000000000000000000000000000000000000000000000000000063a5c8e0 ← deadline (uint256)
  ...
```

**Database Schema:**
```sql
CREATE TABLE parsed_calldata (
    chain_id BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    function_signature TEXT NOT NULL,  -- e.g., "0x38ed1739"
    function_name TEXT,                -- e.g., "swapExactTokensForTokens"
    protocol TEXT,                     -- e.g., "uniswap-v2"
    decoded_params JSONB,              -- Flexible JSON storage
    PRIMARY KEY (chain_id, tx_hash)
) PARTITION BY LIST (chain_id);

-- Index for protocol-specific queries
CREATE INDEX idx_parsed_calldata_protocol ON parsed_calldata(chain_id, protocol);
CREATE INDEX idx_parsed_calldata_function ON parsed_calldata(chain_id, function_name);
```

**Example Query - Track DEX Swaps:**
```sql
SELECT 
    t.tx_hash,
    t.from_address,
    t.block_number,
    t.block_timestamp,
    pc.function_name,
    pc.decoded_params->>'amountIn' as amount_in,
    pc.decoded_params->>'amountOutMin' as amount_out_min,
    pc.decoded_params->>'path' as swap_path
FROM transactions t
JOIN parsed_calldata pc ON t.tx_hash = pc.tx_hash AND t.chain_id = pc.chain_id
WHERE pc.chain_id = 1
  AND pc.protocol = 'uniswap-v2'
  AND pc.function_name = 'swapExactTokensForTokens'
  AND t.block_number > 18000000
ORDER BY t.block_number DESC
LIMIT 100;
```

**Example Query - Most Popular Protocols:**
```sql
SELECT 
    protocol,
    function_name,
    COUNT(*) as call_count,
    COUNT(DISTINCT t.from_address) as unique_users
FROM parsed_calldata pc
JOIN transactions t ON pc.tx_hash = t.tx_hash AND pc.chain_id = t.chain_id
WHERE pc.chain_id = 1
  AND t.block_timestamp > NOW() - INTERVAL '24 hours'
GROUP BY protocol, function_name
ORDER BY call_count DESC
LIMIT 20;
```

### Revert Reason Extraction

**What is it?** Error messages from failed transactions that explain why they reverted.

**Why important?**
- Debug why transactions failed
- Identify common failure patterns (slippage too high, insufficient balance)
- Improve protocol UX based on failure analysis
- Detect smart contract vulnerabilities or bugs

**Types of Reverts:**

1. **String Revert**: `revert("Insufficient balance")`
2. **Custom Error**: `revert InsufficientBalance(required, available)` (Solidity 0.8.4+)
3. **Require Statement**: `require(balance >= amount, "Not enough tokens")`
4. **Out of Gas**: Transaction ran out of gas
5. **Invalid Opcode**: Attempted illegal operation

**Database Schema:**
```sql
CREATE TABLE revert_reasons (
    chain_id BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    revert_reason TEXT,              -- Human-readable error message
    error_signature TEXT,            -- For custom errors: "0x..." (4 bytes)
    error_name TEXT,                 -- For custom errors: "InsufficientBalance"
    error_params JSONB,              -- Decoded error parameters
    revert_type TEXT,                -- 'string', 'custom', 'out_of_gas', 'invalid'
    PRIMARY KEY (chain_id, tx_hash)
) PARTITION BY LIST (chain_id);

-- Index for failure analysis
CREATE INDEX idx_revert_reasons_type ON revert_reasons(chain_id, revert_type);
```

**Example Query - Most Common Failures:**
```sql
SELECT 
    revert_reason,
    COUNT(*) as failure_count,
    COUNT(DISTINCT tx_hash) as unique_failures
FROM revert_reasons
WHERE chain_id = 1
  AND revert_type = 'string'
GROUP BY revert_reason
ORDER BY failure_count DESC
LIMIT 10;

-- Typical results:
-- "UniswapV2: INSUFFICIENT_OUTPUT_AMOUNT" | 45,234
-- "ERC20: transfer amount exceeds balance" | 32,156
-- "Pausable: paused" | 12,845
-- "Ownable: caller is not the owner" | 8,234
```

**Example Query - Protocol-Specific Failures:**
```sql
SELECT 
    pc.protocol,
    rr.revert_reason,
    COUNT(*) as failure_count
FROM revert_reasons rr
JOIN parsed_calldata pc ON rr.tx_hash = pc.tx_hash AND rr.chain_id = pc.chain_id
WHERE rr.chain_id = 1
GROUP BY pc.protocol, rr.revert_reason
ORDER BY failure_count DESC;
```

### Protocol Signatures Registry

Pre-configured function signatures for major protocols to enable automatic classification:

**DEX Protocols:**
```
Uniswap V2:
  - 0x38ed1739: swapExactTokensForTokens(uint,uint,address[],address,uint)
  - 0x7ff36ab5: swapExactETHForTokens(uint,address[],address,uint)
  - 0x18cbafe5: swapExactTokensForETH(uint,uint,address[],address,uint)

Uniswap V3:
  - 0x414bf389: exactInputSingle((address,address,uint24,address,uint,uint,uint,uint160))
  - 0xc04b8d59: exactInput((bytes,address,uint,uint,uint))
  - 0xdb3e2198: exactOutputSingle((address,address,uint24,address,uint,uint,uint,uint160))

Curve:
  - 0x3df02124: exchange(int128,int128,uint256,uint256)
  - 0xa6417ed6: exchange_underlying(int128,int128,uint256,uint256)

1inch:
  - 0x7c025200: swap(address,(address,address,address,address,uint256,uint256,uint256),bytes,bytes)
```

**Bridge Protocols:**
```
LayerZero:
  - 0x7d25a05e: send(uint16,bytes,bytes,address,address,bytes)

Across:
  - 0x9a7c2a8f: deposit(address,address,uint256,uint64,uint64,uint32,bytes)

Stargate:
  - 0x0e0a8e51: swap(uint16,uint256,uint256,address,uint256,uint256,bytes,address,bytes)
```

**DeFi Lending:**
```
Aave V3:
  - 0x617ba037: supply(address,uint256,address,uint16)
  - 0x69328dec: withdraw(address,uint256,address)
  - 0xa415bcad: borrow(address,uint256,uint256,uint16,address)
  - 0x563dd613: repay(address,uint256,uint256,address)
```

**NFT Marketplaces:**
```
OpenSea Seaport:
  - 0xfb0f3ee1: fulfillOrder((address,address,(uint8,address,uint256,uint256,uint256)[],...),bytes32)
  - 0xf2d12b12: fulfillBasicOrder((address,uint256,uint256,...))

Blur:
  - 0x78e4d5c6: execute((address,address,uint256,...),bytes)
```

**Usage in Code:**
```go
// Map of function signatures to protocol info
var protocolSignatures = map[string]ProtocolInfo{
    "0x38ed1739": {
        Protocol: "uniswap-v2",
        Function: "swapExactTokensForTokens",
        ParamTypes: []string{"uint256", "uint256", "address[]", "address", "uint256"},
    },
    "0x414bf389": {
        Protocol: "uniswap-v3",
        Function: "exactInputSingle",
        ParamTypes: []string{"tuple"},
    },
    // ... more signatures
}

func ParseCalldata(txHash string, input []byte) (*ParsedCalldata, error) {
    if len(input) < 4 {
        return nil, errors.New("calldata too short")
    }
    
    signature := hex.EncodeToString(input[:4])
    protocolInfo, exists := protocolSignatures[signature]
    
    if !exists {
        // Unknown function signature
        return &ParsedCalldata{
            FunctionSignature: signature,
            Protocol: "unknown",
        }, nil
    }
    
    // Decode parameters using ABI
    decoded, err := abi.Decode(protocolInfo.ParamTypes, input[4:])
    if err != nil {
        return nil, err
    }
    
    return &ParsedCalldata{
        FunctionSignature: signature,
        FunctionName: protocolInfo.Function,
        Protocol: protocolInfo.Protocol,
        DecodedParams: decoded,
    }, nil
}
```

---

## Blockchain Reorg Handling

### What is a Reorg?

**Definition**: A blockchain reorganization occurs when the canonical chain changes, invalidating previously indexed blocks.

**Visual Example:**
```
Before Reorg:
... ← 100 ← 101 ← 102 ← 103 ← 104 (canonical)
              ↖ 102' ← 103' (orphaned)

After Reorg (longer chain wins):
... ← 100 ← 101 ← 102' ← 103' ← 104' ← 105' (new canonical)
              ↖ 102 ← 103 ← 104 (now orphaned)

Result: Blocks 102-104 must be deleted and replaced with 102'-105'
```

**Why it happens:**
- **Network latency**: Two miners produce blocks simultaneously
- **Chain splits**: Temporary divergence in the network
- **Consensus rules**: Chain resolves to the longest valid chain (most cumulative difficulty/work)
- **Finality differences**: 
  - Ethereum (PoS): Probabilistic finality, reorgs rare after 2-3 epochs (~13 minutes)
  - Bitcoin: Common up to 1-2 blocks, rare beyond 6 confirmations (~60 minutes)
  - Polygon: More frequent (2-3 second block time, less network agreement time)

### Detection Strategy

**Parent Hash Validation:**

Every block contains a `parent_hash` field pointing to the previous block. We validate this chain continuity:

```go
// Pseudocode for reorg detection
func detectReorg(newBlock Block) (bool, int64) {
    // Get the block we have stored at parent_number
    storedParentBlock := db.GetBlock(newBlock.ChainID, newBlock.Number - 1)
    
    if storedParentBlock == nil {
        // Parent doesn't exist - gap in our data (not necessarily a reorg)
        return false, 0
    }
    
    // Compare parent hashes
    if storedParentBlock.Hash != newBlock.ParentHash {
        // Reorg detected! Our stored block is on an orphaned chain
        log.Warn("Reorg detected",
            "chain_id", newBlock.ChainID,
            "block_number", newBlock.Number,
            "expected_parent", storedParentBlock.Hash,
            "actual_parent", newBlock.ParentHash)
        
        // Find the common ancestor
        commonAncestor := findCommonAncestor(newBlock)
        return true, commonAncestor
    }
    
    return false, 0
}

func findCommonAncestor(newBlock Block) int64 {
    // Walk backwards through parent hashes until we find a match
    currentBlock := newBlock
    
    for {
        currentBlock = rpc.GetBlockByHash(currentBlock.ParentHash)
        storedBlock := db.GetBlock(currentBlock.Number)
        
        if storedBlock != nil && storedBlock.Hash == currentBlock.Hash {
            // Found the common ancestor
            return currentBlock.Number
        }
        
        if currentBlock.Number == 0 {
            // Reached genesis without finding common ancestor (shouldn't happen)
            log.Error("No common ancestor found - critical issue")
            return 0
        }
    }
}
```

**Detection Example:**

```
Stored in DB:     ... ← 100 ← 101 ← 102 ← 103 ← 104
New from RPC:     ... ← 100 ← 101 ← 102' ← 103' ← 104' ← 105'

Processing block 105':
1. Check parent hash: 105'.ParentHash == "0x...104'"
2. Get stored block 104 from DB: Hash = "0x...104" (different!)
3. Reorg detected at block 105
4. Walk backward: Check 104', 103', 102' against DB
5. Find common ancestor: Block 101 (matches in both chains)
6. Rollback: Delete blocks 102-104 from DB
7. Resume: Re-ingest blocks 102'-105' from RPC
```

### Handling Process

**Step-by-Step Rollback:**

1. **Detect**: Compare parent hashes when ingesting new blocks
2. **Find common ancestor**: Walk backwards through parent hashes until we find a matching block
3. **Rollback**: Delete all blocks after common ancestor in a database transaction (ACID critical!)
4. **Log**: Record the reorg event for monitoring and analysis
5. **Resume**: Continue ingestion from common ancestor + 1
6. **Replay**: Kafka allows processor to re-process affected events

**Database Implementation:**

```sql
-- Track reorg events for monitoring
CREATE TABLE reorg_events (
    id SERIAL PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    common_ancestor_block BIGINT NOT NULL,
    rollback_from_block BIGINT NOT NULL,
    rollback_to_block BIGINT NOT NULL,
    blocks_removed INT NOT NULL,
    blocks_reprocessed INT,
    handled BOOLEAN DEFAULT FALSE,
    handled_at TIMESTAMP
);

-- Rollback transaction (ATOMIC - all or nothing!)
BEGIN;

-- Delete affected data in reverse dependency order
DELETE FROM events 
WHERE chain_id = 1 AND block_number > 12345;

DELETE FROM transactions 
WHERE chain_id = 1 AND block_number > 12345;

DELETE FROM blocks 
WHERE chain_id = 1 AND block_number > 12345;

-- Record the reorg event
INSERT INTO reorg_events (
    chain_id, 
    common_ancestor_block, 
    rollback_from_block, 
    rollback_to_block,
    blocks_removed
) VALUES (
    1,       -- Ethereum
    12345,   -- Common ancestor
    12346,   -- Start of orphaned chain
    12350,   -- End of orphaned chain
    5        -- Number of blocks removed
);

COMMIT;
```

**Kafka Replay for Event Reprocessing:**

```go
// After rollback, reset Kafka consumer offset to reprocess affected blocks
func reprocessAfterReorg(commonAncestor int64) error {
    // Kafka consumer tracks offset (message position)
    // We can reset to an earlier offset to replay messages
    
    // Find the Kafka offset for the common ancestor block
    offset := getKafkaOffsetForBlock(commonAncestor)
    
    // Reset consumer to that offset
    consumer.Seek(offset)
    
    // Consumer will now re-read and re-process all blocks from common ancestor onward
    // Idempotent database operations (ON CONFLICT DO NOTHING) ensure no duplicates
    return nil
}
```

### Finality Considerations

**Chain-Specific Finality Thresholds:**

Different blockchains have different finality guarantees:

| Chain | Block Time | Finality Type | Safe Threshold | Reasoning |
|-------|-----------|---------------|----------------|-----------|
| **Ethereum (PoS)** | ~12s | Probabilistic | 2-3 epochs (13-20 min) | After 2 epochs, reorg requires 1/3 of validators to be slashed |
| **Bitcoin** | ~10min | Probabilistic | 6 confirmations (~60 min) | 51% attack cost exceeds reward after 6 blocks |
| **Polygon** | ~2s | Probabilistic | 256 blocks (~8-10 min) | Checkpoint on Ethereum every ~30 min provides finality |
| **Arbitrum** | ~0.25s | Deterministic | 1 confirmation (~1 week) | Challenge period allows fraud proofs |
| **Cosmos** | ~6s | Deterministic | 1 block (~6s) | BFT consensus, instant finality |
| **Polkadot** | ~6s | Deterministic | 1 block (~6s) | GRANDPA finality gadget |

**Production Strategy:**

```go
// Only mark blocks as "finalized" after safe threshold
var finalityThresholds = map[int64]int64{
    1:     128,   // Ethereum: ~26 minutes (conservative)
    137:   256,   // Polygon: ~8-10 minutes
    42161: 1,     // Arbitrum: Instant (challenge period handled separately)
    10:    128,   // Optimism: ~26 minutes (similar to Ethereum)
}

func updateFinalityStatus(chainID int64, currentBlock int64) {
    threshold := finalityThresholds[chainID]
    finalizedBlock := currentBlock - threshold
    
    if finalizedBlock > 0 {
        db.Exec(`
            UPDATE blocks 
            SET finalized = true 
            WHERE chain_id = $1 
              AND block_number = $2
              AND finalized = false
        `, chainID, finalizedBlock)
    }
}
```

**Query Strategy - Use Finalized Data:**

```sql
-- For critical applications (financial reporting, settlements)
-- Only query finalized blocks
SELECT * FROM transactions
WHERE chain_id = 1
  AND block_number IN (
      SELECT block_number FROM blocks 
      WHERE chain_id = 1 AND finalized = true
  )
  AND from_address = '0x123...';

-- For real-time dashboards (accept small reorg risk)
-- Query latest blocks
SELECT * FROM transactions
WHERE chain_id = 1
  AND from_address = '0x123...';
```

---

## Event Parsing

### Understanding Ethereum Events (Logs)

Events are the primary way smart contracts communicate state changes. They're stored in transaction receipts and are indexed by block explorers.

**Event Structure:**

```solidity
// Solidity event definition
event Transfer(address indexed from, address indexed to, uint256 value);

// When emitted:
emit Transfer(msg.sender, recipient, amount);
```

**Resulting Ethereum Log:**

```json
{
  "address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",  // Contract address (USDC)
  "topics": [
    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",  // keccak256("Transfer(address,address,uint256)")
    "0x000000000000000000000000742d35cc6634c0532925a3b844bc454e4438f44e",  // from (indexed, padded)
    "0x000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"   // to (indexed, padded)
  ],
  "data": "0x0000000000000000000000000000000000000000000000056bc75e2d63100000",  // value (100 USDC)
  "blockNumber": "0x11a7c34",
  "transactionHash": "0x123...",
  "logIndex": "0x2a"
}
```

**Key Concepts:**

1. **topics[0]**: Event signature (keccak256 hash of event name and parameter types)
2. **topics[1..3]**: Indexed parameters (up to 3, used for filtering)
3. **data**: Non-indexed parameters (ABI-encoded, cheaper storage)

### ERC20 Transfer Example

**Decoding Process:**

```go
// Event signature
eventSig := "Transfer(address,address,uint256)"
eventHash := crypto.Keccak256Hash([]byte(eventSig))
// Result: 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef

// Parse topics
if log.Topics[0] == eventHash {
    // This is a Transfer event
    from := common.BytesToAddress(log.Topics[1].Bytes())
    to := common.BytesToAddress(log.Topics[2].Bytes())
    
    // Decode data field
    value := new(big.Int).SetBytes(log.Data)
    
    fmt.Printf("Transfer: %s → %s: %s\n", from, to, value)
}
```

**Decoded Result:**

```json
{
  "event": "Transfer",
  "contract": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",  // USDC
  "from": "0x742d35cc6634c0532925a3b844bc454e4438f44e",
  "to": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
  "value": "100000000",  // 100 USDC (6 decimals)
  "value_formatted": "100.00 USDC"
}
```

### Database Storage

**Schema:**

```sql
CREATE TABLE events (
    chain_id BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    log_index INT NOT NULL,
    block_number BIGINT NOT NULL,
    block_timestamp TIMESTAMP NOT NULL,
    contract_address TEXT NOT NULL,
    event_signature TEXT NOT NULL,  -- e.g., "Transfer"
    topic0 TEXT NOT NULL,           -- Event hash for raw access
    topic1 TEXT,                    -- First indexed parameter
    topic2 TEXT,                    -- Second indexed parameter
    topic3 TEXT,                    -- Third indexed parameter
    data TEXT,                      -- Raw data field
    decoded_data JSONB,             -- Parsed parameters
    PRIMARY KEY (chain_id, tx_hash, log_index)
) PARTITION BY LIST (chain_id);

-- Indexes for common queries
CREATE INDEX idx_events_contract ON events(chain_id, contract_address, block_number DESC);
CREATE INDEX idx_events_signature ON events(chain_id, event_signature, block_number DESC);
CREATE INDEX idx_events_topic1 ON events(chain_id, topic1) WHERE topic1 IS NOT NULL;
CREATE INDEX idx_events_topic2 ON events(chain_id, topic2) WHERE topic2 IS NOT NULL;
```

**Inserting Events:**

```sql
INSERT INTO events (
    chain_id,
    tx_hash,
    log_index,
    block_number,
    block_timestamp,
    contract_address,
    event_signature,
    topic0,
    topic1,
    topic2,
    decoded_data
) VALUES (
    1,                                                            -- Ethereum
    '0x123...',                                                   -- Transaction hash
    42,                                                           -- Log index in tx
    18500000,                                                     -- Block number
    '2024-01-15 10:30:00',                                       -- Block timestamp
    '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48',              -- USDC contract
    'Transfer',                                                   -- Event name
    '0xddf252ad...',                                             -- Event hash
    '0x000000000000000000000000742d35cc...',                    -- from (indexed)
    '0x000000000000000000000000c02aaa39...',                    -- to (indexed)
    '{"from": "0x742...", "to": "0xc02...", "value": "100000000"}'::jsonb
);
```

### Query Examples

**Find all transfers to/from an address:**

```sql
-- Using indexed topics for fast lookup
SELECT 
    block_number,
    block_timestamp,
    tx_hash,
    contract_address,
    decoded_data->>'from' as from_address,
    decoded_data->>'to' as to_address,
    decoded_data->>'value' as value
FROM events
WHERE chain_id = 1
  AND event_signature = 'Transfer'
  AND (
      -- Address is sender (topic1)
      topic1 = '0x000000000000000000000000742d35cc6634c0532925a3b844bc454e4438f44e'
      OR 
      -- Address is recipient (topic2)
      topic2 = '0x000000000000000000000000742d35cc6634c0532925a3b844bc454e4438f44e'
  )
ORDER BY block_number DESC
LIMIT 100;
```

**Token balance changes for an address:**

```sql
-- Calculate net flow per token
SELECT 
    contract_address as token,
    SUM(CASE 
        WHEN decoded_data->>'from' = '0x742...' THEN -(decoded_data->>'value')::NUMERIC
        WHEN decoded_data->>'to' = '0x742...' THEN (decoded_data->>'value')::NUMERIC
        ELSE 0
    END) as net_change
FROM events
WHERE chain_id = 1
  AND event_signature = 'Transfer'
  AND (
      decoded_data->>'from' = '0x742...'
      OR decoded_data->>'to' = '0x742...'
  )
GROUP BY contract_address;
```

**Most active contracts (by event volume):**

```sql
SELECT 
    contract_address,
    event_signature,
    COUNT(*) as event_count,
    COUNT(DISTINCT tx_hash) as unique_transactions
FROM events
WHERE chain_id = 1
  AND block_timestamp > NOW() - INTERVAL '24 hours'
GROUP BY contract_address, event_signature
ORDER BY event_count DESC
LIMIT 20;
```

### Common Event Signatures

**ERC20 (Tokens):**
```
Transfer(address indexed from, address indexed to, uint256 value)
Approval(address indexed owner, address indexed spender, uint256 value)
```

**ERC721 (NFTs):**
```
Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
ApprovalForAll(address indexed owner, address indexed operator, bool approved)
```

**ERC1155 (Multi-Token):**
```
TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
```

**Uniswap V2:**
```
Swap(address indexed sender, uint amount0In, uint amount1In, uint amount0Out, uint amount1Out, address indexed to)
Sync(uint112 reserve0, uint112 reserve1)
Mint(address indexed sender, uint amount0, uint amount1)
Burn(address indexed sender, uint amount0, uint amount1, address indexed to)
```

**Uniswap V3:**
```
Swap(address indexed sender, address indexed recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
Mint(address sender, address indexed owner, int24 indexed tickLower, int24 indexed tickUpper, uint128 amount, uint256 amount0, uint256 amount1)
```

---

## Database Strategy

### Partitioning Strategy: Why LIST by chain_id?

**Decision Context:**

Our blockchain indexer handles multiple chains (Ethereum, Polygon, Arbitrum, etc.) with different characteristics:
- Different block times (Ethereum ~12s, Polygon ~2s)
- Different transaction volumes (Ethereum 1M tx/day, Polygon 3M tx/day)
- Different query patterns (users often query single chain)
- Need for independent maintenance (archive old Ethereum data without affecting Polygon)

**Why LIST Partitioning (not RANGE or HASH)?**

```sql
-- Our choice: LIST partitioning
CREATE TABLE blocks (
    chain_id BIGINT NOT NULL,
    block_number BIGINT NOT NULL,
    -- ... other fields
    PRIMARY KEY (chain_id, block_number)
) PARTITION BY LIST (chain_id);

CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks FOR VALUES IN (137);
CREATE TABLE blocks_arbitrum PARTITION OF blocks FOR VALUES IN (42161);
```

**Why NOT RANGE partitioning?**
```sql
-- ❌ RANGE doesn't work well for chain_id
-- Chain IDs aren't sequential: Ethereum=1, Polygon=137, Arbitrum=42161
CREATE TABLE blocks_range (...) PARTITION BY RANGE (chain_id);
CREATE TABLE blocks_p1 PARTITION OF blocks_range FOR VALUES FROM (1) TO (100);
CREATE TABLE blocks_p2 PARTITION OF blocks_range FOR VALUES FROM (100) TO (200);
-- Problem: Wasted partitions for gaps (2-136, 138-42160, etc.)
```

**Why NOT HASH partitioning?**
```sql
-- ❌ HASH distributes chains randomly across partitions
CREATE TABLE blocks_hash (...) PARTITION BY HASH (chain_id);
CREATE TABLE blocks_h0 PARTITION OF blocks_hash FOR VALUES WITH (MODULUS 4, REMAINDER 0);
CREATE TABLE blocks_h1 PARTITION OF blocks_hash FOR VALUES WITH (MODULUS 4, REMAINDER 1);
-- Problem: Ethereum and Polygon might be in same partition (no isolation)
-- Problem: Can't archive just Ethereum data (mixed with other chains)
```

**Benefits of LIST Partitioning:**

1. **Partition Pruning** - Query performance optimization
2. **Independent Maintenance** - Archive/optimize per chain
3. **Isolation** - One chain's issues don't affect others
4. **Scalability** - Move hot chains to SSD, cold chains to HDD
5. **Clear Ownership** - Each partition has explicit chain assignment

### Partition Pruning Deep Dive

**What is Partition Pruning?**

PostgreSQL's query planner automatically eliminates (prunes) irrelevant partitions when the WHERE clause filters by the partition key.

**Example: Fast Query (Partition Pruning)**

```sql
-- Query with chain_id filter
EXPLAIN ANALYZE
SELECT * FROM blocks 
WHERE chain_id = 1 
  AND block_number > 18000000;

-- Query Plan:
Seq Scan on blocks_eth  (cost=0.00..150.00 rows=1000 width=200) (actual time=0.5..12.3 rows=1000)
  Filter: (block_number > 18000000)
-- Notice: Only scans blocks_eth partition (not blocks_polygon, blocks_arbitrum)
-- Partitions pruned: 4 of 5
```

**Performance Impact:**
- **Without partitioning**: Scans 10M rows across all chains
- **With partitioning + pruning**: Scans 2M rows in Ethereum partition only
- **Result**: 5x faster query

**Example: Slow Query (No Partition Pruning)**

```sql
-- Query WITHOUT chain_id filter
EXPLAIN ANALYZE
SELECT * FROM blocks 
WHERE block_number > 18000000;

-- Query Plan:
Append  (cost=0.00..750.00 rows=5000 width=200) (actual time=0.5..60.0 rows=5000)
  -> Seq Scan on blocks_eth  (cost=0.00..150.00 rows=1000 width=200)
       Filter: (block_number > 18000000)
  -> Seq Scan on blocks_polygon  (cost=0.00..150.00 rows=1000 width=200)
       Filter: (block_number > 18000000)
  -> Seq Scan on blocks_arbitrum  (cost=0.00..150.00 rows=1000 width=200)
       Filter: (block_number > 18000000)
  -> Seq Scan on blocks_optimism  (cost=0.00..150.00 rows=1000 width=200)
       Filter: (block_number > 18000000)
  -> Seq Scan on blocks_base  (cost=0.00..150.00 rows=1000 width=200)
       Filter: (block_number > 18000000)
-- Notice: Scans ALL 5 partitions (no pruning)
```

**Performance Impact:**
- Scans all 10M rows across all partitions
- 5x slower than pruned query

**Best Practice: Always Include chain_id in WHERE Clause**

```sql
-- ✅ GOOD: Fast (partition pruning)
SELECT * FROM transactions WHERE chain_id = 1 AND tx_hash = '0x123...';

-- ❌ BAD: Slow (scans all partitions)
SELECT * FROM transactions WHERE tx_hash = '0x123...';
```

**API Implementation:**

```go
// Always require chain_id in API endpoints
// GET /v1/chains/{chain_id}/transactions/{hash}  ✅ Good
// GET /v1/transactions/{hash}                     ❌ Bad (no chain_id)

func GetTransaction(chainID int64, txHash string) (*Transaction, error) {
    var tx Transaction
    err := db.QueryRow(`
        SELECT * FROM transactions 
        WHERE chain_id = $1 AND tx_hash = $2
    `, chainID, txHash).Scan(&tx)
    // Partition pruning: Only scans transactions_eth (if chainID = 1)
    return &tx, err
}
```

### Batch INSERT Optimization

**Problem: Network Latency Bottleneck**

When ingesting blockchain blocks with many transactions (100-200+ per block), individual INSERT statements become a performance bottleneck due to network round-trip latency.

**The Issue:**

```go
// ❌ ANTI-PATTERN: Individual INSERTs in a loop
func insertTransactionsSlowly(dbTx *sql.Tx, transactions []Transaction) error {
    for _, tx := range transactions {
        _, err := dbTx.Exec(`
            INSERT INTO transactions (chain_id, block_number, tx_hash, ...)
            VALUES ($1, $2, $3, ...)
        `, tx.ChainID, tx.BlockNumber, tx.Hash, ...)
        if err != nil {
            return err
        }
    }
    return nil
}

// Problem with 188 transactions:
// - Network round-trip: ~300ms per INSERT (Docker containers, even localhost)
// - Database execution: ~3ms per INSERT (fast!)
// - Total time: 188 × 300ms = 56+ seconds
// - Result: Database timeout exceeded (60s default)
```

**Why Network Dominates:**

Each `Exec()` call requires a complete network round-trip:

```
Application → Network → PostgreSQL Container → Network → Application
    |          200ms         3ms exec         200ms         |
    └────────────────── 400ms total ──────────────────────┘
```

Even with localhost Docker containers, network overhead is ~200-400ms per query due to:
- TCP handshake overhead
- Docker bridge network
- Container networking stack
- PostgreSQL protocol overhead (parse, bind, execute, sync)

**Visualization:**

```
Individual INSERTs (188 transactions):
[INSERT #1]──400ms──[INSERT #2]──400ms──[INSERT #3]──400ms──...──[INSERT #188]
Total: 188 × 400ms = 75 seconds ❌ TIMEOUT!

Batch INSERT (188 transactions):
[INSERT (all 188 rows)]──400ms──[DONE]
Total: 400ms ✅ FAST!
```

**Solution: Batch INSERT with Dynamic SQL**

```go
// ✅ OPTIMIZED: Single INSERT with multiple value rows
func insertTransactionsBatch(dbTx *sql.Tx, chainID int64, transactions []Transaction) error {
    if len(transactions) == 0 {
        return nil
    }
    
    // Use strings.Builder for efficient string concatenation
    var query strings.Builder
    query.WriteString(`
        INSERT INTO transactions (
            chain_id, block_number, tx_hash, tx_index, from_address, 
            to_address, value, gas_price, gas_used, status, block_timestamp
        ) VALUES 
    `)
    
    // Pre-allocate args slice for efficiency
    // 11 fields per transaction
    args := make([]interface{}, 0, len(transactions)*11)
    
    for i, tx := range transactions {
        if i > 0 {
            query.WriteString(", ")
        }
        
        // Build placeholder string: ($1,$2,$3,...,$11), ($12,$13,$14,...,$22), ...
        base := i * 11
        query.WriteString(fmt.Sprintf(
            "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
            base+1, base+2, base+3, base+4, base+5, base+6,
            base+7, base+8, base+9, base+10, base+11,
        ))
        
        // Append all values for this transaction
        args = append(args,
            chainID,
            tx.BlockNumber,
            tx.Hash,
            tx.Index,
            tx.FromAddress,
            tx.ToAddress,
            tx.Value,
            tx.GasPrice,
            tx.GasUsed,
            tx.Status,
            tx.BlockTimestamp,
        )
    }
    
    // Single network round-trip for all inserts!
    _, err := dbTx.Exec(query.String(), args...)
    return err
}

// Result: 60 seconds → 200ms (300x faster!)
```

**Performance Comparison:**

| Metric | Individual INSERTs | Batch INSERT | Improvement |
|--------|-------------------|--------------|-------------|
| Time for 188 txs | 60 seconds | 200ms | 300x faster |
| Network round-trips | 188 | 1 | 188x fewer |
| Database timeout needed | 60s | 20s | More resilient |
| Memory usage | Low (~1KB) | ~50KB | Negligible |
| Code complexity | Simple | Moderate | Worth it! |

**Key Implementation Details:**

**1. strings.Builder for Efficiency**

```go
// ❌ BAD: String concatenation creates many temporary strings (slow!)
query := "INSERT INTO ... VALUES "
for i := range txs {
    query += "($1,$2,$3),"  // Creates new string each iteration
}
// With 188 transactions: Creates 188 temporary strings (GC pressure)

// ✅ GOOD: strings.Builder writes to buffer once (fast!)
var query strings.Builder
query.WriteString("INSERT INTO ... VALUES ")
for i := range txs {
    query.WriteString("($1,$2,$3),")  // Efficient append to buffer
}
// Single buffer allocation, no intermediate strings
```

**Why it matters:**
- String concatenation (`str += "text"`) creates a new string object each time
- With 188 iterations: 188 allocations + garbage collection overhead
- `strings.Builder` pre-allocates a buffer and appends without new allocations
- Result: 10-20x faster string building

**2. Dynamic Placeholder Numbering**

PostgreSQL uses numbered placeholders (`$1, $2, $3, ...`), not `?` like MySQL.

```go
// PostgreSQL requires unique numbered placeholders
// With 11 fields and 188 rows:
// Row 0: ($1, $2, $3, ..., $11)
// Row 1: ($12, $13, $14, ..., $22)
// Row 2: ($23, $24, $25, ..., $33)
// ...
// Row 187: ($2057, $2058, $2059, ..., $2068)

base := i * numFields  // i = row index, numFields = 11
for fieldIdx := 1; fieldIdx <= numFields; fieldIdx++ {
    placeholder := base + fieldIdx
    fmt.Sprintf("$%d", placeholder)
}
```

**3. Security - Still Using Parameterized Queries**

```go
// ✅ SAFE: We're building the query structure dynamically,
//          but still using placeholders ($1, $2, etc.)
//          Values are passed separately in args slice
query := "INSERT INTO tx (...) VALUES ($1, $2), ($3, $4)"
dbTx.Exec(query, val1, val2, val3, val4)  // Values parameterized

// ❌ UNSAFE: Don't do this! (SQL injection vulnerability)
query := fmt.Sprintf("INSERT INTO tx VALUES ('%s', '%s')", tx.Hash, tx.From)
dbTx.Exec(query)  // Values embedded directly in SQL string
```

**Why batch INSERT is safe:**
1. Query structure is generated by trusted code (not user input)
2. All values use parameterized placeholders
3. PostgreSQL driver handles escaping and type conversion
4. No possibility of SQL injection

**4. Memory Considerations**

```go
// For 1000 transactions with 11 fields each:
// - Query string: ~50-100KB (depending on field name lengths)
// - Args slice: 1000 × 11 × 8 bytes (pointer size) = ~88KB
// - Total: ~150-200KB per batch (acceptable for most systems)

// If memory is a concern, batch in chunks:
const maxBatchSize = 500  // Tune based on available memory

for i := 0; i < len(transactions); i += maxBatchSize {
    end := min(i+maxBatchSize, len(transactions))
    batch := transactions[i:end]
    
    err := insertTransactionsBatch(dbTx, chainID, batch)
    if err != nil {
        return fmt.Errorf("batch %d-%d failed: %w", i, end, err)
    }
}
```

**Chunking Strategy:**

| Batch Size | Query String | Args Slice | Total Memory | Typical Use Case |
|-----------|-------------|------------|--------------|------------------|
| 100 txs | ~10KB | ~9KB | ~20KB | Low memory environments |
| 500 txs | ~50KB | ~44KB | ~100KB | **Recommended default** |
| 1000 txs | ~100KB | ~88KB | ~200KB | High memory, max performance |
| 5000 txs | ~500KB | ~440KB | ~1MB | Risk of query too large errors |

**5. Error Handling Trade-offs**

```go
// Individual INSERTs: Per-row error handling
for _, tx := range transactions {
    err := insertOne(tx)
    if err != nil {
        log.Warn("Failed to insert tx", "hash", tx.Hash, "error", err)
        continue  // Skip this transaction, continue with others
    }
}

// Batch INSERT: All-or-nothing (within database transaction)
err := insertBatch(transactions)
if err != nil {
    // All 188 inserts failed together
    // Need to handle as a group
    return err
}
```

**Trade-off Decision:**

| Aspect | Individual INSERTs | Batch INSERT |
|--------|-------------------|--------------|
| Error granularity | Per-row | All-or-nothing |
| Partial success | Possible | No (within DB transaction) |
| Performance | Slow (network bound) | Fast (single round-trip) |
| Debugging | Easy (see which row failed) | Harder (whole batch fails) |

**Our choice: Batch INSERT because:**
1. Blockchain data is atomic (block either fully succeeds or fails)
2. We use database transactions anyway (reorg handling requires atomicity)
3. 300x performance gain is worth the complexity
4. Failures are rare in production (schema/constraints validated in tests)

**When to Use Batch INSERT:**

✅ **Good use cases:**
- Ingesting blockchain blocks with many transactions (>50)
- Bulk data import operations
- High-latency database connections (cloud, containers, cross-region)
- Write-heavy workloads with batch accumulation
- When atomicity is required (all succeed or all fail)

❌ **When to use individual INSERTs:**
- Single or few row inserts (<10 rows)
- Need per-row error feedback to users
- Highly variable row sizes (large JSONB, bytea columns >1MB)
- Complex INSERT conflicts requiring individual handling
- Real-time applications where latency matters more than throughput

**Production Lessons:**

1. **Profile first**: Use timing instrumentation to identify bottleneck
   ```go
   start := time.Now()
   err := insertTransactions(transactions)
   duration := time.Since(start)
   log.Info("Insert performance", "count", len(transactions), "duration", duration)
   // Before optimization: count=188 duration=60s
   // After optimization:  count=188 duration=200ms
   ```

2. **Network latency dominates in containerized/cloud environments**: Even "localhost" Docker has 200-400ms overhead

3. **Reduce timeouts after optimization**: We reduced from 60s → 20s for faster failure detection

4. **PostgreSQL handles large batch INSERTs well**: No need to limit unless memory constrained

5. **Transaction atomicity maintained**: All inserts succeed or fail together (ACID preserved)

### Alternative Bulk Loading Approaches

**1. COPY Command (Fastest for Very Large Datasets)**

```go
// Even faster for 10,000+ rows
import "github.com/lib/pq"

func insertWithCopy(dbTx *sql.Tx, transactions []Transaction) error {
    stmt, err := dbTx.Prepare(pq.CopyIn(
        "transactions",
        "chain_id", "block_number", "tx_hash", "tx_index",
        "from_address", "to_address", "value", "gas_price",
        "gas_used", "status", "block_timestamp",
    ))
    if err != nil {
        return err
    }
    defer stmt.Close()
    
    for _, tx := range transactions {
        _, err = stmt.Exec(
            tx.ChainID, tx.BlockNumber, tx.Hash, tx.Index,
            tx.FromAddress, tx.ToAddress, tx.Value, tx.GasPrice,
            tx.GasUsed, tx.Status, tx.BlockTimestamp,
        )
        if err != nil {
            return err
        }
    }
    
    // Flush
    _, err = stmt.Exec()
    return err
}
```

**COPY vs Batch INSERT:**

| Feature | Batch INSERT | COPY |
|---------|-------------|------|
| Speed | Fast | 10x faster |
| Use case | 100-1000 rows | 10,000+ rows |
| RETURNING clause | ✅ Supported | ❌ Not supported |
| ON CONFLICT | ✅ Supported | ❌ Not supported |
| Portability | ✅ Standard SQL | ❌ PostgreSQL-specific |
| Complexity | Moderate | Simple API |

**When to use COPY:**
- Initial database population (loading years of historical data)
- Nightly bulk imports (loading 100K+ transactions)
- Data warehouse ETL pipelines

**When to use Batch INSERT:**
- Real-time blockchain ingestion (100-200 txs per block)
- Need ON CONFLICT handling for idempotency
- Need RETURNING clause to get generated IDs
- Maintaining cross-database compatibility

**2. Temporary Table + INSERT SELECT**

```sql
-- Create temporary staging table
CREATE TEMP TABLE temp_transactions (LIKE transactions INCLUDING DEFAULTS);

-- Load data into temp table (fast, no constraints)
COPY temp_transactions FROM STDIN WITH (FORMAT csv);

-- Validate and insert into main table (with constraints and indexes)
INSERT INTO transactions 
SELECT * FROM temp_transactions
WHERE valid_tx_hash(tx_hash)  -- Custom validation function
ON CONFLICT (chain_id, tx_hash) DO NOTHING;

-- Cleanup
DROP TABLE temp_transactions;
```

**Benefits:**
- Pre-validate data before inserting into main table
- Apply transformations (UPPER(tx_hash), normalize addresses)
- Deduplicate within batch before inserting
- Can resume from last successful batch on failure

**Drawbacks:**
- More complex code (multiple SQL statements)
- Requires temp table management
- Additional disk I/O for temp table

**When to use:**
- Complex data validation required
- Data transformation needed (format conversion, enrichment)
- Loading from external sources (CSV, JSON files)
- Need to deduplicate within batch

---

## Rate Limiting

### Why Rate Limiting?

**Without rate limiting:**
- Malicious users can DDoS your API (millions of requests)
- Single user can exhaust database connections
- Costs spiral out of control (especially with serverless databases)
- Legitimate users suffer degraded performance

**With rate limiting:**
- Fair resource allocation across users
- Predictable infrastructure costs
- Protection against abuse and attacks
- Better quality of service for all users

### Token Bucket Algorithm

**Concept**: A bucket holds tokens that refill at a constant rate. Each request consumes one token. If the bucket is empty, requests are denied.

**Parameters:**
- **Capacity**: Maximum tokens (burst size)
- **Refill Rate**: Tokens added per second
- **Example**: Capacity=100, Rate=10 → Allows 10 req/sec sustained, 100 req burst

**Visual Representation:**

```
Initial State:
Bucket: [●●●●●●●●●●] (10 tokens, capacity 10)
Refill: +2 tokens/second

Time 0s: Request arrives → Consumes 1 token
Bucket: [●●●●●●●●● ] (9 tokens)

Time 0.5s: Refill → +1 token
Bucket: [●●●●●●●●●●] (10 tokens, at capacity)

Time 1s: 5 requests arrive → Consume 5 tokens
Bucket: [●●●●●     ] (5 tokens)

Time 2s: Refill → +2 tokens
Bucket: [●●●●●●●   ] (7 tokens)

Time 3s: 10 requests arrive → Only 7 succeed, 3 denied (bucket empty)
Bucket: [          ] (0 tokens)
Response: 429 Too Many Requests (for 3 requests)
```

**Implementation in Go:**

```go
type TokenBucket struct {
    capacity    int64          // Maximum tokens
    tokens      int64          // Current tokens
    refillRate  int64          // Tokens per second
    lastRefill  time.Time      // Last refill timestamp
    mu          sync.Mutex     // Thread-safe access
}

func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
    return &TokenBucket{
        capacity:   capacity,
        tokens:     capacity,  // Start full
        refillRate: refillRate,
        lastRefill: time.Now(),
    }
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    // Refill tokens based on time elapsed
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()
    tokensToAdd := int64(elapsed * float64(tb.refillRate))
    
    if tokensToAdd > 0 {
        tb.tokens = min(tb.capacity, tb.tokens + tokensToAdd)
        tb.lastRefill = now
    }
    
    // Try to consume a token
    if tb.tokens > 0 {
        tb.tokens--
        return true  // Request allowed
    }
    
    return false  // Request denied (rate limited)
}

func min(a, b int64) int64 {
    if a < b {
        return a
    }
    return b
}
```

**Usage in API Handler:**

```go
// Global rate limiter (shared across all requests)
var globalLimiter = NewTokenBucket(1000, 100)  // 100 req/sec, burst 1000

func apiHandler(w http.ResponseWriter, r *http.Request) {
    if !globalLimiter.Allow() {
        w.Header().Set("X-RateLimit-Remaining", "0")
        w.Header().Set("Retry-After", "1")  // Retry after 1 second
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)  // 429
        return
    }
    
    // Process request normally
    w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", globalLimiter.tokens))
    // ... handle request
}
```

### Multi-Tier Rate Limiting

Real-world APIs need different limits for different users:

**Tier 1: Per-IP Limiting (Prevent DDoS)**

```go
var ipLimiters = sync.Map{}  // map[string]*TokenBucket

func getIPLimiter(ip string) *TokenBucket {
    if limiter, ok := ipLimiters.Load(ip); ok {
        return limiter.(*TokenBucket)
    }
    
    // Create new limiter for this IP
    limiter := NewTokenBucket(200, 100)  // 100 req/sec, burst 200
    ipLimiters.Store(ip, limiter)
    return limiter
}

func rateLimitByIP(r *http.Request) bool {
    ip := getClientIP(r)
    limiter := getIPLimiter(ip)
    return limiter.Allow()
}
```

**Tier 2: Per-API-Key Limiting (Authenticated Users)**

```go
var apiKeyLimits = map[string]struct{ Capacity, Rate int64 }{
    "free_tier":    {100, 10},      // 10 req/sec, burst 100
    "basic_tier":   {500, 50},      // 50 req/sec, burst 500
    "pro_tier":     {2000, 200},    // 200 req/sec, burst 2000
    "enterprise":   {10000, 1000},  // 1000 req/sec, burst 10000
}

func rateLimitByAPIKey(apiKey string, tier string) bool {
    limits := apiKeyLimits[tier]
    limiter := getOrCreateLimiter(apiKey, limits.Capacity, limits.Rate)
    return limiter.Allow()
}
```

**Tier 3: Per-Endpoint Limiting (Protect Expensive Operations)**

```go
// Different endpoints have different costs
var endpointLimits = map[string]*TokenBucket{
    "/v1/transactions": NewTokenBucket(1000, 100),  // Cheap query
    "/v1/analytics":    NewTokenBucket(100, 10),    // Expensive aggregation
    "/v1/export":       NewTokenBucket(10, 1),      // Very expensive, CSV export
}

func rateLimitByEndpoint(path string) bool {
    if limiter, ok := endpointLimits[path]; ok {
        return limiter.Allow()
    }
    return true  // No limit for unlisted endpoints
}
```

**Combined Rate Limiting Middleware:**

```go
func RateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check 1: IP-based rate limit (prevent DDoS)
        if !rateLimitByIP(r) {
            http.Error(w, "IP rate limit exceeded", 429)
            return
        }
        
        // Check 2: API key rate limit (if authenticated)
        apiKey := r.Header.Get("X-API-Key")
        if apiKey != "" {
            tier := getUserTier(apiKey)
            if !rateLimitByAPIKey(apiKey, tier) {
                http.Error(w, "API key rate limit exceeded", 429)
                return
            }
        }
        
        // Check 3: Endpoint-specific rate limit
        if !rateLimitByEndpoint(r.URL.Path) {
            http.Error(w, "Endpoint rate limit exceeded", 429)
            return
        }
        
        // All checks passed
        next.ServeHTTP(w, r)
    })
}
```

### Distributed Rate Limiting with Redis

**Problem with In-Memory Rate Limiting:**

With multiple API servers, in-memory rate limiters don't share state:

```
Client makes 100 req/sec to Server 1 → Allowed (under limit)
Client makes 100 req/sec to Server 2 → Allowed (under limit)
Client makes 100 req/sec to Server 3 → Allowed (under limit)

Total: 300 req/sec (bypassing 100 req/sec limit!)
```

**Solution: Redis as Shared State**

All API servers check the same Redis counter:

```go
import (
    "github.com/go-redis/redis/v8"
    "time"
)

var redisClient = redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

func rateLimitWithRedis(key string, limit int, window time.Duration) (bool, error) {
    ctx := context.Background()
    
    // Sliding window rate limiting
    now := time.Now().Unix()
    windowStart := now - int64(window.Seconds())
    
    // Use sorted set with timestamps as scores
    pipe := redisClient.Pipeline()
    
    // Remove old entries outside the window
    pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
    
    // Count entries in current window
    countCmd := pipe.ZCard(ctx, key)
    
    // Add current request
    pipe.ZAdd(ctx, key, &redis.Z{
        Score:  float64(now),
        Member: fmt.Sprintf("%d-%d", now, rand.Int()),
    })
    
    // Set expiration on the key
    pipe.Expire(ctx, key, window*2)
    
    _, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }
    
    count := countCmd.Val()
    return count < int64(limit), nil
}

// Usage in API handler
func apiHandlerWithRedis(w http.ResponseWriter, r *http.Request) {
    apiKey := r.Header.Get("X-API-Key")
    key := fmt.Sprintf("ratelimit:%s", apiKey)
    
    allowed, err := rateLimitWithRedis(key, 100, time.Minute)
    if err != nil {
        http.Error(w, "Rate limit check failed", 500)
        return
    }
    
    if !allowed {
        w.Header().Set("X-RateLimit-Limit", "100")
        w.Header().Set("X-RateLimit-Remaining", "0")
        w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
        http.Error(w, "Rate limit exceeded", 429)
        return
    }
    
    // Process request
    // ...
}
```

**Simpler Fixed Window Implementation:**

```go
func rateLimitFixedWindow(apiKey string, limit int) (bool, error) {
    ctx := context.Background()
    
    // Key includes current minute: ratelimit:apikey123:1699564800
    key := fmt.Sprintf("ratelimit:%s:%d", apiKey, time.Now().Unix()/60)
    
    // Increment counter
    count, err := redisClient.Incr(ctx, key).Result()
    if err != nil {
        return false, err
    }
    
    // Set expiration on first request
    if count == 1 {
        redisClient.Expire(ctx, key, 2*time.Minute)
    }
    
    return count <= int64(limit), nil
}
```

**Trade-offs:**

| Implementation | Accuracy | Performance | Complexity | Distributed |
|---------------|----------|-------------|------------|-------------|
| **In-memory token bucket** | Perfect | Very fast (~1μs) | Low | ❌ No |
| **Redis fixed window** | Good (edge cases) | Fast (~1ms) | Low | ✅ Yes |
| **Redis sliding window** | Perfect | Moderate (~2-3ms) | Medium | ✅ Yes |
| **Redis token bucket** | Perfect | Fast (~1-2ms) | Medium | ✅ Yes |

**Why Redis over in-memory?**

✅ **Use Redis when:**
- Multiple API servers (horizontal scaling)
- Need consistent limits across servers
- Rate limits are critical (billing, abuse prevention)
- Can tolerate 1-2ms latency

❌ **Use in-memory when:**
- Single API server
- Rate limiting is advisory (not critical)
- Need sub-millisecond performance
- Redis adds operational complexity

### Response Headers for Rate Limiting

**Standard headers (RFC 6585):**

```go
func setRateLimitHeaders(w http.ResponseWriter, remaining, limit int64, resetTime time.Time) {
    w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
    w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
    w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
    
    // Optional: Tell client how long to wait
    if remaining == 0 {
        retryAfter := int(time.Until(resetTime).Seconds())
        w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
    }
}

// Example response headers:
// HTTP/1.1 200 OK
// X-RateLimit-Limit: 100
// X-RateLimit-Remaining: 73
// X-RateLimit-Reset: 1699564800

// When rate limited:
// HTTP/1.1 429 Too Many Requests
// X-RateLimit-Limit: 100
// X-RateLimit-Remaining: 0
// X-RateLimit-Reset: 1699564860
// Retry-After: 60
```

---

## Kafka Message Ordering

### Why Ordering Matters for Blockchain

Blockchain data has strict ordering requirements:

1. **Block sequence**: Block N must be processed before Block N+1
2. **Reorg handling**: When rolling back, must process blocks in reverse order
3. **Event ordering**: Events within a block must maintain log_index order
4. **Transaction ordering**: Transactions within a block have specific order (tx_index)
5. **State consistency**: Processing out-of-order corrupts derived state (balances, counts)

**Example of ordering violation:**

```
Correct order:
Block 100: Alice balance = 50 ETH
Block 101: Alice sends 10 ETH → Alice balance = 40 ETH
Block 102: Alice sends 5 ETH → Alice balance = 35 ETH ✅ Correct

Wrong order (Block 102 processed before Block 101):
Block 100: Alice balance = 50 ETH
Block 102: Alice sends 5 ETH → Alice balance = 45 ETH
Block 101: Alice sends 10 ETH → Alice balance = 35 ETH ✅ Same final result, but...

Problem: Between Block 102 and 101, our database shows Alice has 45 ETH,
         but she actually had 40 ETH. Queries return incorrect historical data!
```

### Kafka Ordering Guarantees

**Within a partition:**
- ✅ Messages are **strictly ordered**
- ✅ Consumers see messages in order
- ✅ Offsets are sequential (0, 1, 2, 3, ...)
- ✅ Append-only log structure

**Across partitions:**
- ❌ **No ordering guarantee**
- Messages can be processed in any order
- Different consumers may be at different offsets

**Visual:**

```
Partition 0: [msg0] → [msg1] → [msg2] → [msg3]  (ordered ✅)
Partition 1: [msgA] → [msgB] → [msgC] → [msgD]  (ordered ✅)

Cross-partition: msg0 and msgA can be processed in any order ❌
```

### Design Decision: Partition by chain_id

**Strategy:**

Use `chain_id` as the Kafka partition key so all blocks from the same chain go to the same partition.

```go
func publishBlock(producer *kafka.Producer, block Block) error {
    // Use chain_id as partition key
    key := []byte(fmt.Sprintf("chain-%d", block.ChainID))
    
    blockJSON, err := json.Marshal(block)
    if err != nil {
        return err
    }
    
    message := &kafka.Message{
        Topic: "blocks",
        Key:   key,  // Same chain_id → same partition
        Value: blockJSON,
    }
    
    return producer.Produce(message, nil)
}
```

**How Kafka assigns partitions:**

```go
// Kafka's partitioner (simplified):
func selectPartition(key []byte, numPartitions int) int {
    hash := murmur2(key)  // Hash the key
    return hash % numPartitions  // Modulo to get partition number
}

// Examples:
// chain-1 (Ethereum)   → hash % 5 = 2 → Partition 2
// chain-137 (Polygon)  → hash % 5 = 4 → Partition 4
// chain-42161 (Arbitrum) → hash % 5 = 1 → Partition 1

// Same chain_id always goes to same partition!
```

**Result:**

```
Kafka Topic: "blocks" (5 partitions)

Partition 0: [empty]
Partition 1: [Arbitrum blocks in order] → 42161.block1000, 42161.block1001, 42161.block1002
Partition 2: [Ethereum blocks in order] → 1.block18500000, 1.block18500001, 1.block18500002
Partition 3: [empty]
Partition 4: [Polygon blocks in order] → 137.block50000000, 137.block50000001
```

**Benefits:**

1. ✅ **Ordering per chain**: All Ethereum blocks processed in order
2. ✅ **Parallel processing**: Different chains processed simultaneously
3. ✅ **Independent failure**: Ethereum consumer crash doesn't affect Polygon
4. ✅ **Scalability**: Add more consumers per chain (consumer groups)

### Consumer Group for Parallel Processing

**Single Consumer (Slow):**

```
Consumer 1 processes:
  Partition 1 (Arbitrum)
  Partition 2 (Ethereum)
  Partition 4 (Polygon)
  
Bottleneck: One consumer handles all chains
```

**Consumer Group (Fast):**

```
Consumer Group "indexer-processors" (3 consumers):

Consumer 1: Partition 1 (Arbitrum only)
Consumer 2: Partition 2 (Ethereum only)
Consumer 3: Partition 4 (Polygon only)

Result: 3x throughput, each chain processed independently
```

**Code:**

```go
import "github.com/segmentio/kafka-go"

func startConsumer(groupID string, partitions []int) {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  []string{"localhost:19092"},
        Topic:    "blocks",
        GroupID:  groupID,  // Consumer group for load balancing
        MinBytes: 10e3,     // 10KB min batch
        MaxBytes: 10e6,     // 10MB max batch
    })
    defer reader.Close()
    
    for {
        msg, err := reader.ReadMessage(context.Background())
        if err != nil {
            log.Error("Failed to read message", "error", err)
            continue
        }
        
        // Process block (guaranteed in order for this partition)
        var block Block
        json.Unmarshal(msg.Value, &block)
        processBlock(block)
        
        // Commit offset after successful processing (exactly-once semantics)
        reader.CommitMessages(context.Background(), msg)
    }
}
```

### Handling Reorgs with Kafka Replay

**Problem:**

Reorg detected at block 105 → Need to re-process blocks 102-105 with new canonical data

**Solution:**

Kafka allows resetting consumer offset to replay messages:

```go
func handleReorg(chainID int64, commonAncestor int64) error {
    // Step 1: Rollback database to common ancestor
    tx, _ := db.Begin()
    tx.Exec("DELETE FROM blocks WHERE chain_id = $1 AND block_number > $2", chainID, commonAncestor)
    tx.Exec("DELETE FROM transactions WHERE chain_id = $1 AND block_number > $2", chainID, commonAncestor)
    tx.Exec("DELETE FROM events WHERE chain_id = $1 AND block_number > $2", chainID, commonAncestor)
    tx.Commit()
    
    // Step 2: Find Kafka offset for common ancestor block
    offset := getKafkaOffsetForBlock(chainID, commonAncestor)
    
    // Step 3: Reset consumer to that offset
    reader.SetOffset(offset)
    
    // Step 4: Consumer will now re-read blocks from common ancestor onward
    // Ingester will publish new canonical blocks (102', 103', 104', 105')
    // Processor will re-process them with idempotent database operations
    
    log.Info("Reorg handled", 
        "chain_id", chainID, 
        "common_ancestor", commonAncestor,
        "kafka_offset", offset)
    
    return nil
}
```

**Offset Tracking:**

```sql
-- Track Kafka offset for each block
CREATE TABLE kafka_offsets (
    chain_id BIGINT NOT NULL,
    block_number BIGINT NOT NULL,
    kafka_offset BIGINT NOT NULL,
    partition INT NOT NULL,
    PRIMARY KEY (chain_id, block_number)
);

-- Insert when processing each block
INSERT INTO kafka_offsets (chain_id, block_number, kafka_offset, partition)
VALUES (1, 18500000, 12345678, 2);

-- Query when reorg happens
SELECT kafka_offset FROM kafka_offsets
WHERE chain_id = 1 AND block_number = 12345
LIMIT 1;
```

### Data Consistency Patterns

**Challenge:**

Maintain consistency between Kafka (message queue) and PostgreSQL (database):

**Pattern 1: Idempotent Database Operations**

```sql
-- Use ON CONFLICT to make inserts idempotent
INSERT INTO blocks (chain_id, block_number, block_hash, ...)
VALUES (1, 18500000, '0x123...', ...)
ON CONFLICT (chain_id, block_number) 
DO UPDATE SET 
    block_hash = EXCLUDED.block_hash,
    updated_at = NOW();

-- Result: Processing same Kafka message twice = safe (no duplicates)
```

**Pattern 2: Offset Commit After Database Write**

```go
func processMessage(msg kafka.Message) error {
    var block Block
    json.Unmarshal(msg.Value, &block)
    
    // Step 1: Write to database (in transaction)
    tx, _ := db.Begin()
    err := insertBlock(tx, block)
    if err != nil {
        tx.Rollback()
        return err  // Don't commit Kafka offset
    }
    tx.Commit()
    
    // Step 2: Commit Kafka offset ONLY after successful database write
    reader.CommitMessages(context.Background(), msg)
    
    // Result: Exactly-once semantics
    // - If database fails: Kafka offset not committed → message replayed
    // - If commit fails: Idempotent INSERT handles duplicate gracefully
    
    return nil
}
```

**Pattern 3: Database Transaction with Checkpointing**

```go
func processBatch(messages []kafka.Message) error {
    // Begin database transaction
    tx, _ := db.Begin()
    
    for _, msg := range messages {
        var block Block
        json.Unmarshal(msg.Value, &block)
        
        // Insert block (all in same transaction)
        err := insertBlock(tx, block)
        if err != nil {
            tx.Rollback()
            return err
        }
    }
    
    // Store checkpoint in database (part of same transaction!)
    lastOffset := messages[len(messages)-1].Offset
    tx.Exec(`
        INSERT INTO consumer_checkpoints (topic, partition, offset)
        VALUES ('blocks', 2, $1)
        ON CONFLICT (topic, partition) 
        DO UPDATE SET offset = EXCLUDED.offset
    `, lastOffset)
    
    // Commit database transaction (atomic!)
    tx.Commit()
    
    // Commit Kafka offset
    reader.CommitMessages(context.Background(), messages...)
    
    // Result: Database writes and checkpoint stored atomically
    // On restart, resume from checkpoint
    
    return nil
}
```

### Backpressure Handling

**Problem:**

Ingester produces messages faster than Processor can consume:

```
Ingester: 1000 blocks/sec → Kafka
Processor: 500 blocks/sec ← Kafka

Result: Kafka lag increases (millions of unprocessed messages)
```

**Solution 1: Kafka as Buffer**

Kafka naturally handles backpressure by buffering messages:

```go
// Kafka configuration for buffering
config := kafka.WriterConfig{
    Brokers:      []string{"localhost:19092"},
    Topic:        "blocks",
    BatchSize:    100,           // Batch up to 100 messages
    BatchTimeout: 10 * time.Millisecond,
    RequiredAcks: 1,             // Wait for leader acknowledgment
    MaxAttempts:  3,             // Retry on failure
}

// Kafka will buffer millions of messages (limited by retention policy)
```

**Solution 2: Monitor Lag and Alert**

```go
func monitorConsumerLag(groupID string) {
    // Get consumer group lag from Kafka
    lag := getConsumerGroupLag(groupID)
    
    if lag > 1000000 {  // More than 1M messages behind
        alert.Send("Consumer lag critical", map[string]interface{}{
            "group_id": groupID,
            "lag": lag,
        })
    }
}

// Prometheus metric
var consumerLag = prometheus.NewGauge(prometheus.GaugeOpts{
    Name: "kafka_consumer_lag_messages",
    Help: "Number of messages consumer is behind",
})

consumerLag.Set(float64(lag))
```

**Solution 3: Dynamic Scaling**

```yaml
# Kubernetes HPA (Horizontal Pod Autoscaler)
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: processor-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: processor
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: External
    external:
      metric:
        name: kafka_consumer_lag
      target:
        type: Value
        value: "100000"  # Scale up if lag > 100K messages

# Result: Automatically adds more processor pods when lag increases
```

**Solution 4: Throttle Ingester**

```go
func ingestWithBackpressure(block Block) error {
    // Check processor lag
    lag := getProcessorLag()
    
    if lag > 500000 {  // More than 500K messages behind
        // Slow down ingestion
        time.Sleep(100 * time.Millisecond)
        log.Warn("Ingester throttled due to processor lag", "lag", lag)
    }
    
    return publishBlock(block)
}
```

---

## Technology Trade-offs

### Language Choice: Go vs Rust vs Python

**Our Decision: Go** ✅

#### Why Go?

**1. I/O-Bound Workload (Not CPU-Bound)**

Blockchain indexing spends most time waiting:
- **RPC calls**: Fetching blocks from Ethereum node (~100-500ms per call)
- **Database writes**: Inserting transactions into PostgreSQL (~50-200ms per batch)
- **Network latency**: Kafka message publishing (~5-10ms)

```
Time breakdown for indexing 1 block:
├── RPC call (fetch block):        400ms  ████████████████████████████████████████ 80%
├── Database write (insert txs):   80ms   ████████ 16%
├── Kafka publish:                 10ms   ██ 2%
└── CPU processing (parse, decode): 10ms   ██ 2%
                                  ─────
                                  500ms total

CPU usage: ~4% of total time
I/O waiting: ~96% of total time

Conclusion: Language performance doesn't matter much here!
```

**2. Excellent Concurrency for I/O**

Go's goroutines make it trivial to parallelize I/O-bound work:

```go
// Fetch blocks from 5 chains simultaneously
var wg sync.WaitGroup
chains := []int64{1, 137, 42161, 10, 8453}  // Ethereum, Polygon, Arbitrum, Optimism, Base

for _, chainID := range chains {
    wg.Add(1)
    go func(cid int64) {
        defer wg.Done()
        
        // Each goroutine independently fetches blocks
        for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
            block := rpcClient.GetBlock(cid, blockNum)  // Parallel RPC calls!
            publishToKafka(block)
        }
    }(chainID)
}

wg.Wait()

// Result: 5x throughput with ~100 lines of code
// Rust equivalent: Would need async/await, tokio runtime (~300 lines)
// Python equivalent: asyncio or threading (GIL limits true parallelism)
```

**3. Faster Development Velocity**

| Aspect | Go | Rust | Python |
|--------|----|----|--------|
| **Learning curve** | Moderate (1-2 weeks) | Steep (1-3 months) | Easy (1 week) |
| **Compilation** | Fast (~5s) | Slow (~60s) | Interpreted |
| **Error messages** | Clear | Cryptic (borrow checker) | Runtime errors |
| **Refactoring** | Easy | Hard (ownership changes) | Easy |
| **Hiring** | ✅ Many developers | ❌ Scarce talent | ✅ Many developers |
| **Iteration speed** | Fast | Slow | Very fast |

**Team productivity matters:** 
- Building MVP in Go: 2-3 months
- Building MVP in Rust: 4-6 months (fighting borrow checker)
- Result: Go gets to market faster

**4. Mature Ecosystem for Our Stack**

| Library | Go | Rust | Python |
|---------|----|----|--------|
| **Ethereum client** | go-ethereum (canonical) ✅ | ethers-rs | web3.py |
| **Kafka client** | segmentio/kafka-go ✅ | rdkafka | confluent-kafka |
| **PostgreSQL driver** | pgx (excellent) ✅ | tokio-postgres | psycopg2 |
| **JSON parsing** | encoding/json ✅ | serde_json | json |

go-ethereum is the **reference Ethereum implementation** - using Go gives us direct access to the canonical library.

**5. Good Enough Performance**

```
Benchmark: Decode 1M Ethereum transactions

Rust:   850ms  ████████████████████████████████████████████████ (fastest)
Go:     1200ms ██████████████████████████████████████████████████████████████ (41% slower)
Python: 8500ms ████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████ (10x slower)

For our workload (I/O-bound):
- Rust: 500ms total (400ms I/O + 100ms CPU)
- Go:   504ms total (400ms I/O + 104ms CPU)
- Difference: 4ms = 0.8% slower (negligible!)

Conclusion: Go's "slower" CPU performance doesn't matter when I/O dominates
```

#### When Rust Would Be Better

✅ **Use Rust if:**
- CPU-bound workload (cryptography, zero-knowledge proofs, consensus algorithms)
- Need zero-copy deserialization (millions of small messages per second)
- Building high-frequency trading engine (microsecond latency requirements)
- Memory safety is critical (kernel, embedded systems)
- Team has Rust expertise

**Example: Zero-copy parsing**

```rust
// Rust: Zero-copy deserialization with serde
// Data stays in original buffer (no allocations)
let block: Block = serde_json::from_slice(&bytes)?;  // ~50μs

// Go: Must allocate new memory for each field
var block Block
json.Unmarshal(bytes, &block)  // ~150μs

// Difference matters at 1M+ operations/sec
// Doesn't matter at 100 operations/sec (our scale)
```

#### When Python Would Be Better

✅ **Use Python if:**
- Prototyping and experimentation (data science, research)
- Heavy use of ML libraries (TensorFlow, PyTorch)
- Glue code between systems
- Team is primarily data scientists (not SREs)

❌ **Python limitations for our use case:**
- GIL (Global Interpreter Lock) prevents true parallelism
- 10x slower than Go for CPU-intensive tasks
- No compile-time type checking (runtime errors in production)
- High memory usage (everything is an object)

#### Interview Answer

**Q: Why did you choose Go over Rust?**

**A:** "Blockchain indexing is I/O-bound - we spend 96% of time waiting on RPC calls and database writes, not doing CPU work. Go's goroutines make it trivial to parallelize these I/O operations across multiple chains. We can spin up thousands of goroutines with minimal overhead.

Rust would be faster for CPU-intensive work like cryptography, but for our workload, the 40% CPU performance difference translates to only 0.8% slower overall because I/O dominates. Go's faster development velocity and larger talent pool made it the pragmatic choice.

If we were building a consensus algorithm or zero-knowledge proof system where CPU performance is critical, Rust would be the better choice."

---

### Message Broker: Kafka vs RabbitMQ vs Redis Pub/Sub

**Our Decision: Kafka** ✅

#### Why Kafka?

**1. Replay Capability (Critical for Reorgs)**

```
Kafka: Messages persist on disk (configurable retention)
├── Block 100: Offset 1000 [stored for 7 days]
├── Block 101: Offset 1001 [stored for 7 days]
├── Block 102: Offset 1002 [stored for 7 days]
└── Block 103: Offset 1003 [stored for 7 days]

Reorg detected at block 103!
→ Reset consumer offset to 1000
→ Re-process blocks 100-103 with new canonical data ✅

RabbitMQ: Messages deleted after consumption
├── Block 100: Consumed → DELETED ❌
├── Block 101: Consumed → DELETED ❌
├── Block 102: Consumed → DELETED ❌
└── Block 103: Consumed → DELETED ❌

Reorg detected!
→ Messages are gone, can't replay ❌
→ Must re-fetch from RPC (slow, expensive)
```

**2. High Throughput**

| Metric | Kafka | RabbitMQ | Redis Pub/Sub |
|--------|-------|----------|---------------|
| **Messages/sec** | 100,000+ | ~20,000 | 1,000,000+ |
| **Batching** | ✅ Native | ⚠️ Limited | ❌ No |
| **Persistence** | ✅ Disk | ✅ Disk (slower) | ❌ Memory only |
| **Ordering** | ✅ Per partition | ⚠️ Per queue | ❌ None |
| **Backpressure** | ✅ Natural buffering | ⚠️ Queues fill up | ❌ Drop messages |

**Ethereum mainnet:**
- ~15 seconds per block
- ~150 transactions per block
- 10 tx/sec average, 300 tx/sec peak

Kafka handles this easily. RabbitMQ would work but with less headroom.

**3. Consumer Groups (Parallel Processing)**

```
Kafka Consumer Group:
Topic: "blocks" (3 partitions)

Consumer 1 ← Partition 0 (Ethereum)
Consumer 2 ← Partition 1 (Polygon)
Consumer 3 ← Partition 2 (Arbitrum)

Each consumer independently processes its partition
If Consumer 2 crashes, Kafka rebalances → Consumer 1 or 3 takes over

RabbitMQ:
Queue: "blocks"

Consumer 1 ← Message 1
Consumer 2 ← Message 2
Consumer 3 ← Message 3

Can't guarantee ordering across consumers (same chain might be processed by different consumers)
```

**4. Retention Policy (Time-Travel Queries)**

```go
// Kafka: Query historical data
consumer.SeekToOffset(1000000)  // Go back to offset from 3 days ago
for msg := range consumer.Messages() {
    // Re-process old data for analytics
}

// Use case: Backfill new analytics table with historical data
// Without Kafka: Must re-fetch from RPC (expensive, slow)
// With Kafka: Replay from retained messages (fast, free)
```

#### When RabbitMQ Would Be Better

✅ **Use RabbitMQ if:**
- Need complex routing (topic exchanges, headers routing)
- RPC-style request/response patterns
- Task queue with priorities
- Don't need message replay
- Lower throughput requirements (<10K msg/sec)

**Example: Job Queue**

```
RabbitMQ excels at:
Queue: "video_encoding"
├── Job 1: Priority HIGH  ← Worker 1 (picked first)
├── Job 2: Priority LOW
└── Job 3: Priority HIGH  ← Worker 2 (picked second)

Kafka: All messages treated equally (no priorities)
```

#### When Redis Pub/Sub Would Be Better

✅ **Use Redis Pub/Sub if:**
- Real-time notifications (chat, live dashboards)
- Fire-and-forget messaging
- Very low latency required (<1ms)
- Ephemeral data (don't need persistence)

**Example: Live Dashboard**

```go
// Redis Pub/Sub: Push real-time updates to dashboard
redis.Publish("new_block", blockJSON)

// All connected browsers receive update instantly
// If browser is offline: Message is lost (acceptable for dashboards)

// Kafka: Overkill for ephemeral real-time notifications
```

#### Trade-off Summary

| Feature | Kafka | RabbitMQ | Redis Pub/Sub |
|---------|-------|----------|---------------|
| **Throughput** | Very high (100K+ msg/s) | Medium (20K msg/s) | Very high (1M+ msg/s) |
| **Persistence** | ✅ Disk (durable) | ✅ Disk (durable) | ❌ Memory (ephemeral) |
| **Ordering** | ✅ Per partition | ⚠️ Per queue | ❌ None |
| **Replay** | ✅ Yes | ❌ No | ❌ No |
| **Complexity** | High | Medium | Low |
| **Latency** | ~5-10ms | ~1-5ms | <1ms |
| **Best for** | Event streaming, logs | Task queues, RPC | Real-time notifications |

#### Interview Answer

**Q: Why Kafka over RabbitMQ?**

**A:** "Kafka's replay capability is critical for blockchain reorg handling. When we detect a reorg, we need to rollback the database and re-process blocks from the common ancestor. Kafka lets us reset the consumer offset and replay messages. RabbitMQ deletes messages after consumption, so we'd have to re-fetch from RPC (slow and expensive).

Additionally, Kafka's high throughput (100K+ msg/sec) and partitioning model fit perfectly with our multi-chain architecture. Each chain gets its own partition, ensuring ordered processing per chain while allowing parallel processing across chains.

The trade-off is operational complexity - Kafka requires managing partitions, retention policies, and rebalancing. But for event streaming with replay requirements, Kafka is the right tool."

---

### Database: PostgreSQL vs Cassandra vs TimescaleDB

**Our Decision: PostgreSQL** ✅

#### Why PostgreSQL?

**1. ACID Transactions (Critical for Reorg Rollbacks)**

```sql
-- PostgreSQL: Atomic reorg rollback
BEGIN;

DELETE FROM events WHERE chain_id = 1 AND block_number > 12345;
DELETE FROM transactions WHERE chain_id = 1 AND block_number > 12345;
DELETE FROM blocks WHERE chain_id = 1 AND block_number > 12345;
INSERT INTO reorg_events (...) VALUES (...);

COMMIT;  -- All succeed or all fail (atomic!)

-- Cassandra: No transactions
DELETE FROM events WHERE chain_id = 1 AND block_number > 12345;
-- Succeeds
DELETE FROM transactions WHERE chain_id = 1 AND block_number > 12345;
-- Fails (network error)
-- Result: Inconsistent state (events deleted but transactions remain) ❌
```

**2. Rich Query Capabilities**

```sql
-- PostgreSQL: Complex analytical query
SELECT 
    DATE(block_timestamp) as date,
    from_address,
    COUNT(*) as tx_count,
    SUM(value) as total_value,
    AVG(gas_price) as avg_gas
FROM transactions
WHERE chain_id = 1
  AND block_timestamp > NOW() - INTERVAL '30 days'
  AND status = 'success'
GROUP BY DATE(block_timestamp), from_address
HAVING COUNT(*) > 100
ORDER BY total_value DESC
LIMIT 100;

-- Cassandra: Must denormalize everything
-- Would need 5+ separate tables pre-computed
-- Can't do ad-hoc queries like above ❌
```

**3. Flexible Schema with JSONB**

```sql
-- Store flexible event data with JSONB
CREATE TABLE events (
    tx_hash TEXT,
    event_signature TEXT,
    decoded_data JSONB  -- Different events have different fields
);

-- Query JSON fields efficiently
SELECT * FROM events
WHERE decoded_data->>'from' = '0x123...'
  AND (decoded_data->>'value')::NUMERIC > 1000000000000000000;

-- Index JSON fields
CREATE INDEX idx_events_from ON events USING GIN ((decoded_data->>'from'));

-- Cassandra: Would need rigid schema or store as TEXT (can't query efficiently)
```

**4. Excellent Partitioning Support**

```sql
-- LIST partitioning by chain_id
CREATE TABLE blocks (...) PARTITION BY LIST (chain_id);
CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);

-- Query performance with partition pruning
SELECT * FROM blocks WHERE chain_id = 1 AND block_number > 18000000;
-- Only scans blocks_eth partition (5x faster)

-- Cassandra: Would need separate table per chain (no cross-chain queries)
```

**5. Scale Appropriate for Our Needs**

```
Current scale:
- 5 chains
- ~10M transactions/day
- ~2TB database size
- 100-1000 queries/sec

PostgreSQL handles this easily:
- Can scale to 100TB with partitioning
- 10K+ writes/sec with proper indexes
- Read replicas for query scaling

Cassandra designed for:
- Hundreds of nodes
- Petabytes of data
- Millions of writes/sec
- Global distribution

Conclusion: PostgreSQL is right-sized, Cassandra is overkill
```

#### When Cassandra Would Be Better

✅ **Use Cassandra if:**
- Massive scale (billions of rows, petabytes)
- Write-heavy workload (millions of writes/sec)
- Global distribution (multi-region, low latency worldwide)
- Can accept eventual consistency
- Have devops team for cluster management

**Example: Time-series sensor data**

```
IoT Platform:
- 1 million sensors
- 1 write/sec per sensor
- 1 billion writes/day
- Simple queries (last 24 hours for sensor X)

Cassandra excels here:
- Linear scalability (add more nodes → more throughput)
- Tunable consistency
- Efficient time-series storage

PostgreSQL would struggle:
- Single primary bottleneck
- Expensive to scale writes
```

#### When TimescaleDB Would Be Better

✅ **Use TimescaleDB if:**
- Time-series workload (continuous monitoring, metrics)
- Need both SQL and time-series optimizations
- Want PostgreSQL compatibility
- Automatic data retention policies

**TimescaleDB = PostgreSQL + time-series extensions**

```sql
-- TimescaleDB: Hypertable (automatic partitioning by time)
CREATE TABLE metrics (
    time TIMESTAMPTZ NOT NULL,
    chain_id INT,
    metric_name TEXT,
    value DOUBLE PRECISION
);

SELECT create_hypertable('metrics', 'time');

-- Automatic compression (10x space savings)
ALTER TABLE metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'chain_id'
);

-- Continuous aggregates (pre-computed rollups)
CREATE MATERIALIZED VIEW hourly_metrics
WITH (timescaledb.continuous) AS
SELECT time_bucket('1 hour', time) AS hour,
       chain_id,
       AVG(value) as avg_value
FROM metrics
GROUP BY hour, chain_id;
```

**When to migrate from PostgreSQL → TimescaleDB:**
- Database > 1TB and still growing
- Most queries filter by timestamp
- Need automated data retention (drop old partitions)
- Want better compression

**Migration is easy:**
1. Install TimescaleDB extension: `CREATE EXTENSION timescaledb;`
2. Convert tables: `SELECT create_hypertable('blocks', 'block_timestamp');`
3. No code changes needed (still PostgreSQL wire protocol)

#### Trade-off Summary

| Feature | PostgreSQL | TimescaleDB | Cassandra |
|---------|-----------|-------------|-----------|
| **ACID transactions** | ✅ Yes | ✅ Yes | ❌ No |
| **Complex queries** | ✅ Yes | ✅ Yes | ❌ No (limited) |
| **Time-series optimized** | ❌ No | ✅ Yes | ⚠️ Yes |
| **Horizontal scaling** | ⚠️ Limited | ⚠️ Limited | ✅ Excellent |
| **Consistency** | ✅ Strong | ✅ Strong | ⚠️ Eventual |
| **Operational complexity** | Low | Low | High |
| **Scale limit** | ~10TB | ~100TB | Petabytes |

#### Interview Answer

**Q: Why PostgreSQL instead of Cassandra?**

**A:** "PostgreSQL's ACID transactions are essential for reorg handling. When we rollback a reorg, we need to delete blocks, transactions, and events atomically. Cassandra's eventual consistency would allow partial state, corrupting our index.

Additionally, PostgreSQL's rich query capabilities let us run ad-hoc analytics without pre-computing everything. JSONB support gives us flexibility for event data with varying schemas.

Our scale (10M transactions/day, 2TB database) is well within PostgreSQL's capabilities. Cassandra is designed for petabyte-scale, globally-distributed systems - overkill for our needs and adds significant operational complexity.

If we grow to billions of transactions or need global distribution, we could migrate to TimescaleDB (drop-in PostgreSQL replacement) or consider Cassandra."

---

### Repository Structure: Monorepo vs Separate Repos

**Our Decision: Monorepo** ✅

#### Why Monorepo?

**1. Shared Types Across Services**

```
services/
├── ingester/
│   └── main.go        → imports shared/models
├── processor/
│   └── main.go        → imports shared/models
├── api/
│   └── main.go        → imports shared/models
└── shared/
    ├── models/
    │   └── models.go   ← Single source of truth
    └── config/
        └── config.go

// shared/models/models.go
type Block struct {
    ChainID     int64     `json:"chain_id"`
    BlockNumber int64     `json:"block_number"`
    BlockHash   string    `json:"block_hash"`
    Timestamp   time.Time `json:"timestamp"`
}

// All services use the same Block struct
// Change schema once, all services get the update ✅
```

**Separate repos:** Would need to copy `models.go` to each repo or publish as a library (versioning nightmare).

**2. Atomic Cross-Service Changes**

```
Example: Add new field to Block struct

Monorepo (single commit):
├── shared/models/models.go          → Add ParentHash field
├── services/ingester/main.go        → Populate ParentHash
├── services/processor/main.go       → Process ParentHash
└── services/api/main.go             → Return ParentHash in API

One commit, all services updated ✅
CI/CD tests entire system together

Separate repos (3 commits):
1. Update models library → v1.2.0
2. Update ingester → depends on models v1.2.0
3. Update processor → depends on models v1.2.0
4. Update API → depends on models v1.2.0

Version mismatch hell:
- Ingester on v1.2.0
- Processor on v1.1.0 (missed upgrade)
- API on v1.2.0
→ Processor can't parse new field, crashes in production ❌
```

**3. Easier Local Development**

```bash
# Monorepo: Clone once
git clone github.com/user/blockchain-indexer
cd blockchain-indexer
make docker-up      # Start all services
make test-all       # Test all services

# Separate repos: Clone 5 repos
git clone github.com/user/indexer-models
git clone github.com/user/indexer-ingester
git clone github.com/user/indexer-processor
git clone github.com/user/indexer-api
git clone github.com/user/indexer-web

# Then figure out how to run them together...
# Docker Compose spans 5 repos? Git submodules? Scripts?
```

**4. Single CI/CD Pipeline**

```yaml
# .github/workflows/ci.yml (monorepo)
jobs:
  test-backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: cd services/ingester && go test ./...
      - run: cd services/processor && go test ./...
      - run: cd services/api && go test ./...
  
  test-frontend:
    runs-on: ubuntu-latest
    steps:
      - run: cd web && npm test
  
  deploy:
    needs: [test-backend, test-frontend]
    runs-on: ubuntu-latest
    steps:
      - run: ./deploy.sh  # Deploy all services together

# Separate repos: 5 different CI/CD configurations
# Coordination nightmare for deployments
```

#### When Separate Repos Would Be Better

✅ **Use separate repos if:**
- Services are truly independent (different teams, different release cycles)
- Need different access controls (public API, private backend)
- Services written in different languages with different tooling
- Organization is large (100+ developers)

**Example: Microservices at scale**

```
Company with 50 teams:
├── team-payments/ (10 devs, Python, weekly releases)
├── team-auth/ (5 devs, Go, bi-weekly releases)
├── team-analytics/ (8 devs, Scala, monthly releases)
└── team-notifications/ (3 devs, Node.js, daily releases)

Separate repos make sense:
- Different release cycles
- Different tech stacks
- Different team permissions
- Minimize cross-team coordination
```

**Our case:**
- 3 services (ingester, processor, API)
- All in Go
- Tightly coupled (same data models)
- Same release cycle
- Small team (1-3 devs)

→ Monorepo is the right choice

#### Trade-offs

| Aspect | Monorepo | Separate Repos |
|--------|----------|----------------|
| **Shared code** | ✅ Easy (import paths) | ❌ Hard (library versioning) |
| **Atomic changes** | ✅ Single commit | ❌ Multiple PRs, version coordination |
| **Local dev** | ✅ Clone once | ❌ Clone many |
| **CI/CD** | ✅ Single pipeline | ❌ Multiple pipelines |
| **Independent releases** | ❌ All or nothing | ✅ Per-service |
| **Access control** | ❌ Same for all | ✅ Granular |
| **Repository size** | ⚠️ Can grow large | ✅ Small |

#### Hybrid Approach

Some companies use both:

```
blockchain-indexer/ (monorepo)
├── services/
│   ├── ingester/
│   ├── processor/
│   └── api/
└── shared/

blockchain-indexer-web/ (separate repo)
└── Web frontend (public, different team, different stack)
```

**Why separate web?**
- Public open-source (backend is private)
- Different team (frontend specialists)
- Different tech stack (TypeScript vs Go)
- Different release cycle (hourly vs weekly)

#### Interview Answer

**Q: Why did you choose a monorepo?**

**A:** "Our services are tightly coupled - they share the same data models (Block, Transaction, Event structs). In a monorepo, we can make atomic changes across all services in a single commit. For example, adding a new field to the Block struct updates ingester, processor, and API together, with the entire system tested as one unit in CI.

With separate repos, we'd need to publish models as a versioned library. Then we'd have the coordination problem: ingester on v1.2, processor still on v1.1, leading to version mismatches in production.

Our team is small (1-3 devs) and all services are in Go with the same release cycle. The monorepo reduces friction without the downsides of separate repos.

If we grew to 10+ teams with different stacks and release cycles, we'd consider splitting. But for now, monorepo is the pragmatic choice."

---

## API Design

### RESTful API Architecture

**Core Principles:**

1. **Resource-based URLs**: Nouns, not verbs
2. **HTTP methods**: GET, POST, PUT, DELETE
3. **Stateless**: Each request contains all necessary information
4. **Versioning**: `/v1/` in path for backward compatibility
5. **Consistent response format**: Standard error and success structures

**Base URL Structure:**

```
https://api.indexer.com/v1/{resource}

Examples:
GET    /v1/chains                          # List all chains
GET    /v1/chains/{id}                     # Get chain details
GET    /v1/chains/{id}/blocks              # List recent blocks
GET    /v1/chains/{id}/blocks/{number}     # Get specific block
GET    /v1/chains/{id}/transactions/{hash} # Get transaction
GET    /v1/addresses/{address}/transactions # Address transaction history
```

### Core Endpoints

**1. Chain Endpoints**

```go
// GET /v1/chains
// List supported chains
{
  "chains": [
    {
      "chain_id": 1,
      "name": "Ethereum",
      "symbol": "ETH",
      "block_time": 12,
      "status": "synced",
      "current_block": 18500000,
      "blocks_behind": 0
    },
    {
      "chain_id": 137,
      "name": "Polygon",
      "symbol": "MATIC",
      "block_time": 2,
      "status": "syncing",
      "current_block": 50000000,
      "blocks_behind": 125
    }
  ]
}

// GET /v1/chains/{id}/status
// Get indexing status for a chain
{
  "chain_id": 1,
  "name": "Ethereum",
  "current_block": 18500000,
  "latest_block": 18500000,
  "blocks_behind": 0,
  "indexing_rate": 12.5,  // blocks per second
  "last_updated": "2024-11-27T10:30:00Z",
  "status": "synced"  // synced, syncing, error
}
```

**2. Block Endpoints**

```go
// GET /v1/chains/{id}/blocks?limit=20&offset=0
// List recent blocks (paginated)
{
  "blocks": [
    {
      "block_number": 18500000,
      "block_hash": "0x123...",
      "parent_hash": "0x456...",
      "timestamp": "2024-11-27T10:30:00Z",
      "transaction_count": 156,
      "gas_used": 15000000,
      "gas_limit": 30000000,
      "miner": "0x789..."
    }
  ],
  "pagination": {
    "total": 18500000,
    "limit": 20,
    "offset": 0,
    "has_more": true
  }
}

// GET /v1/chains/{id}/blocks/{number}
// Get specific block with full details
{
  "block_number": 18500000,
  "block_hash": "0x123...",
  "parent_hash": "0x456...",
  "timestamp": "2024-11-27T10:30:00Z",
  "transaction_count": 156,
  "transactions": [
    {
      "tx_hash": "0xabc...",
      "from": "0x742...",
      "to": "0xc02...",
      "value": "1000000000000000000",  // 1 ETH in wei
      "gas_price": "50000000000",      // 50 gwei
      "gas_used": 21000,
      "status": "success"
    }
  ],
  "gas_used": 15000000,
  "gas_limit": 30000000
}
```

**3. Transaction Endpoints**

```go
// GET /v1/chains/{id}/transactions/{hash}
// Get transaction details
{
  "tx_hash": "0xabc...",
  "block_number": 18500000,
  "block_hash": "0x123...",
  "timestamp": "2024-11-27T10:30:00Z",
  "from_address": "0x742...",
  "to_address": "0xc02...",
  "value": "1000000000000000000",
  "gas_price": "50000000000",
  "gas_used": 21000,
  "gas_limit": 100000,
  "nonce": 42,
  "status": "success",
  "input": "0x",  // Calldata
  "events": [
    {
      "log_index": 0,
      "contract_address": "0xA0b...",
      "event_signature": "Transfer",
      "decoded_data": {
        "from": "0x742...",
        "to": "0xc02...",
        "value": "100000000"
      }
    }
  ]
}

// GET /v1/addresses/{address}/transactions?limit=100&cursor=18500000
// Get transaction history for an address
{
  "address": "0x742d35cc6634c0532925a3b844bc454e4438f44e",
  "transaction_count": 1523,
  "transactions": [
    {
      "tx_hash": "0xabc...",
      "block_number": 18500000,
      "timestamp": "2024-11-27T10:30:00Z",
      "from": "0x742...",
      "to": "0xc02...",
      "value": "1000000000000000000",
      "direction": "out"  // "in" or "out" relative to address
    }
  ],
  "pagination": {
    "next_cursor": "18499900",
    "has_more": true
  }
}
```

**4. Event Endpoints**

```go
// GET /v1/chains/{id}/events?contract=0xA0b...&event=Transfer&limit=100
// Query events with filters
{
  "events": [
    {
      "tx_hash": "0xabc...",
      "block_number": 18500000,
      "log_index": 0,
      "contract_address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
      "event_signature": "Transfer",
      "decoded_data": {
        "from": "0x742...",
        "to": "0xc02...",
        "value": "100000000"
      },
      "timestamp": "2024-11-27T10:30:00Z"
    }
  ],
  "pagination": {
    "limit": 100,
    "cursor": "18500000-0",
    "has_more": true
  }
}
```

### Pagination Strategies

#### Offset/Limit Pagination (Simple, but slow at scale)

**How it works:**

```sql
-- Client requests page 1
SELECT * FROM transactions 
WHERE from_address = '0x123...'
ORDER BY block_number DESC
LIMIT 100 OFFSET 0;

-- Client requests page 2
SELECT * FROM transactions 
WHERE from_address = '0x123...'
ORDER BY block_number DESC
LIMIT 100 OFFSET 100;

-- Client requests page 1000
SELECT * FROM transactions 
WHERE from_address = '0x123...'
ORDER BY block_number DESC
LIMIT 100 OFFSET 100000;
-- Problem: Database must scan and skip 100,000 rows! Slow!
```

**Performance Issue:**

```
Page 1:    0.5ms  ▌
Page 10:   5ms    █████
Page 100:  50ms   ██████████████████████████████████████████████████
Page 1000: 500ms  ████████████████████████████████████████████████████████████████████████████████████████████████

Problem: Linear degradation with deep pagination
```

**API Design:**

```
GET /v1/addresses/{address}/transactions?limit=100&offset=0
GET /v1/addresses/{address}/transactions?limit=100&offset=100
GET /v1/addresses/{address}/transactions?limit=100&offset=200
```

**When to use:**
- ✅ Small datasets (<10,000 rows)
- ✅ Users rarely paginate deep (typically view first 3-5 pages)
- ✅ Need page numbers (UI shows "Page 1, 2, 3...")
- ❌ Large datasets (millions of rows)
- ❌ Deep pagination required

#### Cursor-Based Pagination (Fast, scalable)

**How it works:**

```sql
-- Client requests first page (no cursor)
SELECT * FROM transactions 
WHERE from_address = '0x123...'
ORDER BY block_number DESC
LIMIT 100;
-- Returns: block_number 18500000 to 18499901

-- Client requests next page (cursor = last block_number from previous page)
SELECT * FROM transactions 
WHERE from_address = '0x123...'
  AND block_number < 18499901  -- Use cursor as filter!
ORDER BY block_number DESC
LIMIT 100;
-- Returns: block_number 18499900 to 18499801

-- No matter how deep, always uses index seek (fast!)
```

**Performance:**

```
Page 1:    0.5ms  ▌
Page 10:   0.5ms  ▌
Page 100:  0.5ms  ▌
Page 1000: 0.5ms  ▌

Constant time! Database uses index seek, not scan.
```

**API Design:**

```go
// First request (no cursor)
GET /v1/addresses/{address}/transactions?limit=100

Response:
{
  "transactions": [...],  // 100 items
  "pagination": {
    "next_cursor": "18499901",  // Last block_number from results
    "has_more": true
  }
}

// Next request (with cursor)
GET /v1/addresses/{address}/transactions?limit=100&cursor=18499901

Response:
{
  "transactions": [...],  // Next 100 items
  "pagination": {
    "next_cursor": "18499801",
    "has_more": true
  }
}
```

**Implementation:**

```go
func getAddressTransactions(address string, limit int, cursor *int64) ([]Transaction, string, error) {
    query := `
        SELECT * FROM transactions
        WHERE (from_address = $1 OR to_address = $1)
    `
    
    args := []interface{}{address}
    
    // Add cursor filter if provided
    if cursor != nil {
        query += ` AND block_number < $2`
        args = append(args, *cursor)
    }
    
    query += ` ORDER BY block_number DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1)
    args = append(args, limit)
    
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, "", err
    }
    defer rows.Close()
    
    var transactions []Transaction
    for rows.Next() {
        var tx Transaction
        rows.Scan(&tx.Hash, &tx.BlockNumber, ...)
        transactions = append(transactions, tx)
    }
    
    // Generate next cursor (last block_number)
    var nextCursor string
    if len(transactions) == limit {
        nextCursor = fmt.Sprintf("%d", transactions[len(transactions)-1].BlockNumber)
    }
    
    return transactions, nextCursor, nil
}
```

**Trade-offs:**

| Feature | Offset/Limit | Cursor-Based |
|---------|-------------|--------------|
| **Performance** | Slow (deep pages) | Fast (constant time) |
| **Page numbers** | ✅ Yes (Page 1, 2, 3...) | ❌ No |
| **Jump to page** | ✅ Yes | ❌ No (sequential only) |
| **Real-time updates** | ⚠️ Inconsistent | ✅ Consistent |
| **Implementation** | Simple | Moderate |

**Real-time consistency example:**

```
Offset/Limit Problem:
Page 1 (offset 0):  Items 1-10
New items inserted while user navigates
Page 2 (offset 10): Items 12-21 (missed item 11!)

Cursor-Based:
Page 1: Items A-J (cursor = J's ID)
New items inserted
Page 2: Items K-T (uses cursor J, sees all items after J)
```

**Our Choice:** Cursor-based for address transactions (can have millions), offset/limit for chain/block listings (small datasets).

### Real-Time Streaming: WebSocket vs SSE

#### WebSocket (Bidirectional)

**Use case:** Real-time block updates for dashboard

```go
// Server implementation
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    chainID := r.URL.Query().Get("chain_id")
    
    // Subscribe to Redis Pub/Sub for new blocks
    subscriber := redis.Subscribe("new_block:" + chainID)
    defer subscriber.Close()
    
    for msg := range subscriber.Channel() {
        var block Block
        json.Unmarshal([]byte(msg.Payload), &block)
        
        // Send to client
        conn.WriteJSON(block)
    }
}

// Client usage (JavaScript)
const ws = new WebSocket('wss://api.indexer.com/v1/ws/blocks?chain_id=1');

ws.onmessage = (event) => {
    const block = JSON.parse(event.data);
    console.log('New block:', block.block_number);
    updateUI(block);
};

ws.onerror = (error) => {
    console.error('WebSocket error:', error);
};

ws.onclose = () => {
    console.log('Connection closed, reconnecting...');
    setTimeout(connect, 1000);  // Reconnect
};
```

#### Server-Sent Events (SSE) (Unidirectional, simpler)

**Use case:** Server pushes updates, client doesn't need to send data

```go
// Server implementation
func handleSSE(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported", 500)
        return
    }
    
    chainID := r.URL.Query().Get("chain_id")
    subscriber := redis.Subscribe("new_block:" + chainID)
    defer subscriber.Close()
    
    for msg := range subscriber.Channel() {
        fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
        flusher.Flush()
    }
}

// Client usage (JavaScript)
const eventSource = new EventSource('https://api.indexer.com/v1/chains/1/blocks/stream');

eventSource.onmessage = (event) => {
    const block = JSON.parse(event.data);
    console.log('New block:', block.block_number);
    updateUI(block);
};

eventSource.onerror = (error) => {
    console.error('SSE error:', error);
    // Browser automatically reconnects
};
```

#### Polling with Long Timeout (Fallback)

**Use case:** Networks that block WebSocket/SSE

```go
// Server implementation
func handleLongPoll(w http.ResponseWriter, r *http.Request) {
    chainID := r.URL.Query().Get("chain_id")
    since := r.URL.Query().Get("since_block")
    
    // Hold connection for up to 30 seconds
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    
    // Check for new blocks every second
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            // Timeout, return empty
            json.NewEncoder(w).Encode(map[string]interface{}{
                "blocks": []Block{},
            })
            return
            
        case <-ticker.C:
            // Check for new blocks
            blocks := getBlocksSince(chainID, since)
            if len(blocks) > 0 {
                json.NewEncoder(w).Encode(map[string]interface{}{
                    "blocks": blocks,
                })
                return
            }
        }
    }
}

// Client usage
async function poll(sinceBlock) {
    const response = await fetch(
        `/v1/chains/1/blocks?since=${sinceBlock}&timeout=30`
    );
    const data = await response.json();
    
    if (data.blocks.length > 0) {
        data.blocks.forEach(updateUI);
        sinceBlock = data.blocks[data.blocks.length - 1].block_number;
    }
    
    // Immediately poll again (server holds connection)
    poll(sinceBlock);
}
```

**Comparison:**

| Method | Pros | Cons | Use Case |
|--------|------|------|----------|
| **WebSocket** | Bidirectional, low latency | Complex, requires connection management | Interactive apps, trading platforms |
| **SSE** | Simple, auto-reconnect | Unidirectional only | Dashboards, notifications |
| **Long Polling** | Works everywhere | Higher latency, more connections | Fallback for restricted networks |

**Production Strategy:**

```go
// Try WebSocket first, fallback to SSE, then polling
if (supportsWebSocket()) {
    connectWebSocket();
} else if (supportsSSE()) {
    connectSSE();
} else {
    startPolling();
}
```

### Caching Strategy

**Cache Layers:**

```
Client Request
    ↓
1. CDN Cache (CloudFlare) - 1ms
    ↓ (miss)
2. Redis Cache - 5ms
    ↓ (miss)
3. Database - 50ms
```

**What to Cache:**

```go
// ✅ Cache: Immutable data (old blocks never change)
// GET /v1/chains/1/blocks/18000000
// Cache-Control: public, max-age=31536000, immutable (1 year!)

// ⚠️ Cache briefly: Recent blocks (might reorg)
// GET /v1/chains/1/blocks/18500000
// Cache-Control: public, max-age=60 (1 minute)

// ❌ Don't cache: Real-time data
// GET /v1/chains/1/status
// Cache-Control: no-cache
```

**Implementation:**

```go
func getBlock(chainID, blockNumber int64) (*Block, error) {
    // Check if block is finalized (old enough to be immutable)
    currentBlock := getCurrentBlock(chainID)
    isFinalized := blockNumber < (currentBlock - 128)  // 128 blocks behind
    
    // Try cache first
    cacheKey := fmt.Sprintf("block:%d:%d", chainID, blockNumber)
    if cached, err := redis.Get(cacheKey); err == nil {
        var block Block
        json.Unmarshal([]byte(cached), &block)
        return &block, nil
    }
    
    // Cache miss, query database
    block := queryDatabase(chainID, blockNumber)
    
    // Cache with appropriate TTL
    blockJSON, _ := json.Marshal(block)
    if isFinalized {
        redis.Set(cacheKey, blockJSON, 24*time.Hour)  // Long TTL
    } else {
        redis.Set(cacheKey, blockJSON, 1*time.Minute)  // Short TTL (might reorg)
    }
    
    return block, nil
}
```

**Cache Headers:**

```go
func setCacheHeaders(w http.ResponseWriter, isFinalized bool) {
    if isFinalized {
        // Immutable data - cache aggressively
        w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
        w.Header().Set("ETag", generateETag(data))
    } else {
        // Recent data - cache briefly
        w.Header().Set("Cache-Control", "public, max-age=60")
        w.Header().Set("ETag", generateETag(data))
    }
}
```

**Cache Invalidation (on Reorg):**

```go
func handleReorg(chainID, fromBlock, toBlock int64) {
    // Invalidate cache for affected blocks
    for block := fromBlock; block <= toBlock; block++ {
        cacheKey := fmt.Sprintf("block:%d:%d", chainID, block)
        redis.Del(cacheKey)
    }
    
    log.Info("Cache invalidated for reorg", 
        "chain_id", chainID, 
        "from", fromBlock, 
        "to", toBlock)
}
```

### Error Handling & Response Format

**Standard Response Structure:**

```go
// Success response
{
  "data": {
    // Actual response data
  },
  "meta": {
    "timestamp": "2024-11-27T10:30:00Z",
    "request_id": "req_abc123"
  }
}

// Error response
{
  "error": {
    "code": "TRANSACTION_NOT_FOUND",
    "message": "Transaction 0x123... not found on chain 1",
    "details": {
      "chain_id": 1,
      "tx_hash": "0x123..."
    }
  },
  "meta": {
    "timestamp": "2024-11-27T10:30:00Z",
    "request_id": "req_abc123"
  }
}
```

**HTTP Status Codes:**

```go
// 2xx Success
200 OK              // Request succeeded
201 Created         // Resource created (rare in read-heavy API)
204 No Content      // Success, no body

// 4xx Client Errors
400 Bad Request     // Invalid parameters
401 Unauthorized    // Missing/invalid API key
403 Forbidden       // API key doesn't have permission
404 Not Found       // Resource doesn't exist
429 Too Many Requests // Rate limited

// 5xx Server Errors
500 Internal Server Error  // Our bug
502 Bad Gateway           // RPC provider down
503 Service Unavailable   // Database down
504 Gateway Timeout       // Request took too long
```

**Error Code Examples:**

```go
const (
    ErrInvalidChainID      = "INVALID_CHAIN_ID"
    ErrBlockNotFound       = "BLOCK_NOT_FOUND"
    ErrTransactionNotFound = "TRANSACTION_NOT_FOUND"
    ErrInvalidAddress      = "INVALID_ADDRESS"
    ErrRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
    ErrDatabaseUnavailable = "DATABASE_UNAVAILABLE"
)

func handleError(w http.ResponseWriter, code string, message string, statusCode int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    
    response := map[string]interface{}{
        "error": map[string]interface{}{
            "code":    code,
            "message": message,
        },
        "meta": map[string]interface{}{
            "timestamp":  time.Now().UTC(),
            "request_id": getRequestID(),
        },
    }
    
    json.NewEncoder(w).Encode(response)
}

// Usage
if !isValidAddress(address) {
    handleError(w, ErrInvalidAddress, 
        "Address format is invalid", 
        http.StatusBadRequest)
    return
}
```

### API Versioning

**URL Versioning (Recommended):**

```
/v1/chains
/v2/chains  ← Breaking changes

Pros:
- Clear version in URL
- Easy to route different versions to different servers
- Can deprecate old versions cleanly

Cons:
- URL changes when upgrading
```

**Header Versioning (Alternative):**

```
GET /chains HTTP/1.1
Accept: application/vnd.indexer.v1+json

GET /chains HTTP/1.1
Accept: application/vnd.indexer.v2+json

Pros:
- Same URL for all versions

Cons:
- Harder to route
- Less obvious to API consumers
```

**Deprecation Strategy:**

```go
// /v1 endpoints (deprecated)
func handleV1(w http.ResponseWriter, r *http.Request) {
    // Add deprecation header
    w.Header().Set("X-API-Deprecation", "true")
    w.Header().Set("X-API-Sunset", "2025-12-31")
    w.Header().Set("X-API-Upgrade-Guide", "https://docs.indexer.com/v2-migration")
    
    // Still serve v1 responses
    // ...
}

// Timeline:
// 2024-11-27: Launch v2, keep v1 working
// 2025-06-01: Add deprecation headers to v1
// 2025-12-31: Shut down v1
```

---

## Interview Questions

### Q1: How do you handle blockchain reorganizations?

**Answer:**

"We detect reorgs by comparing parent hashes. When ingesting block N, we check if our stored block N-1's hash matches the new block's parent hash. If not, we've detected a reorg.

Our handling process:

1. **Find common ancestor**: Walk backwards through parent hashes until we find a matching block
2. **Atomic rollback**: Use a database transaction to delete all blocks, transactions, and events after the common ancestor
3. **Resume ingestion**: Continue from common ancestor + 1
4. **Kafka replay**: Reset consumer offset to re-process affected blocks

```sql
BEGIN;
DELETE FROM events WHERE chain_id = 1 AND block_number > 12345;
DELETE FROM transactions WHERE chain_id = 1 AND block_number > 12345;
DELETE FROM blocks WHERE chain_id = 1 AND block_number > 12345;
INSERT INTO reorg_events (...) VALUES (...);
COMMIT;
```

For production, we only mark blocks as 'finalized' after the chain-specific finality threshold - 128 blocks for Ethereum (~26 minutes), 256 blocks for Polygon. Finalized blocks are cached aggressively since they can't reorg."

**Follow-up: What if a reorg is 1000 blocks deep?**

"Deep reorgs are extremely rare on major chains (would require massive 51% attack). For defense:
- Alert on-call engineers for manual review
- Pause ingestion temporarily
- Verify multiple RPC providers agree on the chain state
- Consider the chain compromised and wait for community resolution

On Ethereum post-merge, reorgs deeper than 2-3 epochs would require 1/3 of validators to be slashed, making deep reorgs economically impossible."

---

### Q2: How do you scale the ingester for multiple chains?

**Answer:**

"We deploy separate ingester instances per chain, each with:

1. **Dedicated checkpoint tracking**: `last_indexed_block` stored per chain in database
2. **Chain-specific configuration**: RPC URLs, finality rules, block times
3. **Independent failure domains**: One chain's RPC issues don't affect others
4. **Kafka partition by chain_id**: Each chain publishes to its own partition for ordered processing

```go
// Each ingester goroutine handles one chain
for _, chainConfig := range chains {
    go func(cfg ChainConfig) {
        for {
            block := fetchNextBlock(cfg.ChainID, cfg.RPCURL)
            publishToKafka(block, cfg.ChainID)  // Same chain_id → same partition
            updateCheckpoint(cfg.ChainID, block.Number)
        }
    }(chainConfig)
}
```

This enables:
- **Parallel processing**: All chains ingest simultaneously
- **Independent scaling**: Ethereum might need 3 ingesters, Polygon just 1
- **Chain-specific monitoring**: Alert if Ethereum ingester lags

For RPC rate limits, we use:
- Token bucket rate limiter per provider
- Multiple providers with automatic failover (Infura + Alchemy + Quicknode)
- Connection pooling to reuse TCP connections
- Exponential backoff on 429 responses
- WebSocket subscriptions for new blocks (more efficient than polling)"

---

### Q3: How do you ensure data consistency during high load?

**Answer:**

"We use several strategies:

1. **Database transactions**: All reorg rollbacks use ACID transactions - all deletes succeed or all fail together

2. **Idempotent operations**: All database writes use `ON CONFLICT DO NOTHING` or upserts:
```sql
INSERT INTO events (...) VALUES (...)
ON CONFLICT (tx_hash, log_index) DO NOTHING;
```

3. **Exactly-once semantics**: Kafka consumers commit offsets ONLY after successful database writes
```go
// Write to database first
tx.Exec("INSERT INTO blocks (...) VALUES (...)")
tx.Commit()

// Then commit Kafka offset
consumer.CommitMessages(ctx, msg)
```

4. **Checkpointing**: We persist `last_processed_block` and resume from there on restart

5. **Backpressure handling**: Kafka acts as a buffer between ingestion and processing, preventing overload

6. **Connection pooling**: Limit concurrent database connections to prevent saturation (max 20 connections)"

**Follow-up: What happens if the processor crashes mid-batch?**

"The crash scenario is safe due to offset commit strategy:

1. Processor crashes after writing 7/10 messages to database
2. Kafka offset was NOT committed (we only commit after full batch success)
3. On restart, processor sees last committed offset
4. Re-processes those 7 messages
5. Database INSERT uses `ON CONFLICT DO NOTHING` → No duplicates
6. Continues with remaining 3 messages

This guarantees at-least-once delivery with idempotent writes = effectively exactly-once."

---

### Q4: How would you optimize query performance for address transaction history?

**Answer:**

"Multiple approaches:

**1. Composite indexes** on frequently queried columns:
```sql
CREATE INDEX idx_txs_from_addr ON transactions (from_address, block_number DESC);
CREATE INDEX idx_txs_to_addr ON transactions (to_address, block_number DESC);
```

**2. Cursor-based pagination** instead of offset/limit:
```sql
-- Fast (uses index seek)
SELECT * FROM transactions 
WHERE from_address = '0x123' AND block_number < 18500000 
ORDER BY block_number DESC LIMIT 20;

-- Slow (scans 1M rows to skip them)
SELECT * FROM transactions 
WHERE from_address = '0x123' 
ORDER BY block_number DESC 
OFFSET 1000000 LIMIT 20;
```

**3. Redis caching** for hot addresses:
- Cache recent 100 transactions per address
- TTL 5 minutes for non-finalized blocks
- TTL 24 hours for finalized blocks

**4. Partition pruning**: Always include `chain_id` in WHERE clause:
```sql
-- Fast (only scans transactions_eth partition)
SELECT * FROM transactions WHERE chain_id = 1 AND from_address = '0x123';

-- Slow (scans all chain partitions)
SELECT * FROM transactions WHERE from_address = '0x123';
```

**5. Materialized view** for address summaries:
```sql
CREATE MATERIALIZED VIEW address_stats AS
SELECT 
    address,
    COUNT(*) as tx_count,
    MAX(block_number) as last_active,
    SUM(value) as total_volume
FROM (
    SELECT from_address as address, block_number, value FROM transactions
    UNION ALL
    SELECT to_address as address, block_number, value FROM transactions
) GROUP BY address;

-- Refresh periodically
REFRESH MATERIALIZED VIEW CONCURRENTLY address_stats;
```

This reduces query time from 5 seconds to 50ms for popular addresses."

---

### Q5: How do you handle API rate limiting at scale?

**Answer:**

"We implement multi-tier rate limiting:

**1. Per-IP limiting** (prevent DDoS):
```go
// Token bucket: 100 req/sec, burst 200
ipLimiter := NewTokenBucket(200, 100)
if !ipLimiter.Allow() {
    http.Error(w, "Rate limit exceeded", 429)
    return
}
```

**2. Per-API-key limiting** (tiered pricing):
- Free tier: 10 req/sec
- Basic tier: 50 req/sec
- Pro tier: 200 req/sec
- Enterprise: 1000 req/sec

**3. Distributed rate limiting with Redis**:
```go
// All API servers share same Redis counter
key := fmt.Sprintf("ratelimit:%s:%d", apiKey, time.Now().Unix()/60)
count := redis.Incr(key)
redis.Expire(key, 120)  // 2 minute window
if count > limit {
    return 429
}
```

**4. Response headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 73
X-RateLimit-Reset: 1699564800
Retry-After: 60
```

**Why Redis over in-memory?**

With multiple API servers, in-memory limiters don't share state:
- User makes 100 req/sec to Server 1 → Allowed
- User makes 100 req/sec to Server 2 → Allowed
- Total: 200 req/sec (bypassing 100 req/sec limit!)

Redis provides single source of truth, all servers enforce limits consistently. Trade-off is 1-2ms latency per request, but necessary for correctness."

---

### Q6: Explain your partitioning strategy

**Answer:**

"We partition tables by `chain_id` using LIST partitioning:

```sql
CREATE TABLE blocks (...) PARTITION BY LIST (chain_id);
CREATE TABLE blocks_eth PARTITION OF blocks FOR VALUES IN (1);
CREATE TABLE blocks_polygon PARTITION OF blocks FOR VALUES IN (137);
```

**Why LIST not RANGE or HASH?**

- Chain IDs aren't sequential (Ethereum=1, Polygon=137, Arbitrum=42161)
- RANGE would create wasted partitions for gaps
- HASH would mix chains together (can't isolate per chain)

**Benefits:**

1. **Query performance**: Queries with `chain_id` filter only scan relevant partition (5x faster)
```sql
-- Only scans blocks_eth
SELECT * FROM blocks WHERE chain_id = 1 AND block_number > 18000000;
```

2. **Maintenance**: Can archive old Ethereum data without affecting Polygon
3. **Scalability**: Move hot chains to SSD, cold chains to HDD
4. **Isolation**: One chain's high volume doesn't slow down others

**Critical**: Always include `chain_id` in WHERE clause for partition pruning. Without it, PostgreSQL scans ALL partitions (no performance benefit)."

---

### Q7: How would you handle 1 billion transactions?

**Answer:**

"Current design supports ~100M transactions. For 1B+:

**1. Time-based sub-partitioning**:
```sql
-- Sub-partition Ethereum by month
CREATE TABLE blocks_eth_2024_01 PARTITION OF blocks_eth 
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE blocks_eth_2024_02 PARTITION OF blocks_eth 
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
```

**2. Migrate to TimescaleDB** (PostgreSQL extension):
- Automatic partitioning by time (hypertables)
- Compression (10x space savings)
- Continuous aggregates for analytics
- Drop-in replacement (no code changes)

**3. Archival strategy**:
- **Hot data**: Last 3 months on NVMe SSD (fast queries)
- **Warm data**: 3-12 months on HDD (slower, cheaper)
- **Cold data**: >12 months on S3 (archive, query via Athena)

**4. Horizontal sharding** for address-centric queries:
```
Shard 1: addresses starting 0x0-0x7
Shard 2: addresses starting 0x8-0xf
```

**5. Read replicas**: 
- 1 primary for writes
- 5 read replicas for queries
- Connection pooling (pgBouncer) to distribute load

**6. Denormalized summary tables**:
```sql
-- Pre-compute daily address activity
CREATE TABLE address_daily_stats (
    date DATE,
    address TEXT,
    tx_count INT,
    total_value NUMERIC,
    PRIMARY KEY (date, address)
);
```

This scales to billions of transactions while maintaining sub-second query performance."

---

### Q8: Design an API endpoint for real-time block streaming

**Answer:**

"Three approaches depending on client capabilities:

**Approach 1: WebSocket** (best for browsers):
```javascript
const ws = new WebSocket('wss://api.indexer.com/v1/ws/blocks?chain_id=1');

ws.onmessage = (event) => {
    const block = JSON.parse(event.data);
    updateDashboard(block);
};
```

Server uses Redis Pub/Sub to fan out to all connected clients:
```go
subscriber := redis.Subscribe("new_block:1")
for msg := range subscriber.Channel() {
    conn.WriteJSON(msg.Payload)  // Send to WebSocket client
}
```

**Approach 2: Server-Sent Events (SSE)** (simpler than WebSocket):
```javascript
const eventSource = new EventSource('/v1/chains/1/blocks/stream');
eventSource.onmessage = (event) => {
    const block = JSON.parse(event.data);
    updateDashboard(block);
};
```

**Approach 3: Long polling** (fallback for restricted networks):
```
GET /v1/chains/1/blocks?since=18500000&timeout=30s

// Server holds connection for 30s until new block arrives
// Client immediately polls again (server holds next request)
```

**Production considerations:**

1. **Heartbeat**: Ping every 30s to detect dead connections
2. **Rate limit**: Max 100 concurrent WebSocket connections per IP
3. **Buffer**: Keep last 100 blocks in memory for reconnecting clients
4. **Backpressure**: Return 429 if client can't keep up with block rate
5. **Redis Pub/Sub**: Fan out to all API servers (horizontally scalable)

Choice: Use WebSocket for interactive apps, SSE for dashboards, long polling as fallback."

---

### Q9: How would you design the API for a block explorer?

**Answer:**

"**Core endpoints**:

```
GET /v1/chains                           # List supported chains
GET /v1/chains/{id}/status               # Current block, indexing lag
GET /v1/chains/{id}/blocks               # Recent blocks (paginated)
GET /v1/chains/{id}/blocks/{number}      # Block details + transactions
GET /v1/chains/{id}/transactions/{hash}  # Transaction details + events
GET /v1/addresses/{address}/transactions # Address transaction history
```

**Design principles**:

1. **Versioning**: `/v1/` in path (allows breaking changes in `/v2/`)

2. **Cursor-based pagination** for performance:
```json
{
  "transactions": [...],
  "pagination": {
    "next_cursor": "18499901",
    "has_more": true
  }
}
```

3. **Caching headers** based on finality:
```
// Old blocks (immutable)
Cache-Control: public, max-age=31536000, immutable

// Recent blocks (might reorg)
Cache-Control: public, max-age=60
```

4. **Field selection** to reduce payload:
```
GET /v1/blocks?fields=number,hash,timestamp
```

5. **Filtering**:
```
GET /v1/transactions?status=success&from_block=18000000&to_block=18500000
```

6. **Standard error format**:
```json
{
  "error": {
    "code": "TRANSACTION_NOT_FOUND",
    "message": "Transaction 0x123... not found on chain 1"
  },
  "meta": {
    "request_id": "req_abc123",
    "timestamp": "2024-11-27T10:30:00Z"
  }
}
```

7. **Rate limiting headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 73
X-RateLimit-Reset: 1699564800
```

This design scales to millions of users while maintaining sub-100ms response times."

---

### Q10: How do you ensure exactly-once processing with Kafka?

**Answer:**

"We combine three techniques:

**1. Idempotent database operations**:
```sql
INSERT INTO blocks (chain_id, block_number, ...)
VALUES (1, 18500000, ...)
ON CONFLICT (chain_id, block_number) DO NOTHING;
```
Even if we process the same Kafka message twice, no duplicate data.

**2. Commit offset AFTER database write**:
```go
func processMessage(msg kafka.Message) error {
    // Step 1: Write to database
    tx, _ := db.Begin()
    err := insertBlock(tx, block)
    if err != nil {
        tx.Rollback()
        return err  // Kafka offset NOT committed
    }
    tx.Commit()
    
    // Step 2: Commit Kafka offset only after success
    consumer.CommitMessages(ctx, msg)
    
    return nil
}
```

**3. Store checkpoint in database (same transaction)**:
```go
tx.Exec("INSERT INTO blocks (...) VALUES (...)")
tx.Exec(`
    INSERT INTO kafka_checkpoints (topic, partition, offset)
    VALUES ('blocks', 2, $1)
    ON CONFLICT (topic, partition) 
    DO UPDATE SET offset = EXCLUDED.offset
`, msg.Offset)
tx.Commit()
```

**Failure scenarios**:

- **Database fails**: Transaction rolls back, Kafka offset not committed → Message replayed ✅
- **Commit fails**: Idempotent INSERT handles duplicate gracefully → No duplicate data ✅
- **Consumer crashes**: Restarts from last committed offset → Replays since checkpoint ✅

Result: At-least-once delivery + idempotent writes = effectively exactly-once semantics."

---

### Q11: What's your strategy for handling RPC provider failures?

**Answer:**

"Multi-layered approach:

**1. Multiple providers with automatic failover**:
```go
providers := []string{
    "https://eth-mainnet.g.alchemy.com/v2/...",
    "https://mainnet.infura.io/v3/...",
    "https://rpc.ankr.com/eth",
}

func fetchBlock(blockNum int64) (*Block, error) {
    for _, provider := range providers {
        block, err := tryFetchBlock(provider, blockNum)
        if err == nil {
            return block, nil
        }
        log.Warn("Provider failed, trying next", "provider", provider, "error", err)
    }
    return nil, errors.New("all providers failed")
}
```

**2. Circuit breaker pattern**:
```go
// After 5 consecutive failures, stop trying for 60 seconds
circuitBreaker := NewCircuitBreaker(5, 60*time.Second)

if !circuitBreaker.Allow(provider) {
    // Provider is in "open" state, skip it
    continue
}

block, err := tryFetchBlock(provider, blockNum)
if err != nil {
    circuitBreaker.RecordFailure(provider)
} else {
    circuitBreaker.RecordSuccess(provider)
}
```

**3. Exponential backoff**:
```go
backoff := 1 * time.Second
for retries := 0; retries < 5; retries++ {
    block, err := fetchBlock(blockNum)
    if err == nil {
        return block
    }
    time.Sleep(backoff)
    backoff *= 2  // 1s, 2s, 4s, 8s, 16s
}
```

**4. Provider health monitoring**:
```go
// Check provider latency and success rate
type ProviderStats struct {
    SuccessRate float64
    AvgLatency  time.Duration
}

// Route traffic to healthy providers
func selectProvider(stats map[string]ProviderStats) string {
    // Sort by: success rate > 95% && latency < 500ms
    // Use best provider, fallback to others
}
```

**5. Rate limit compliance**:
```go
// Token bucket per provider
limiters := map[string]*TokenBucket{
    "alchemy": NewTokenBucket(100, 10),  // 10 req/sec
    "infura":  NewTokenBucket(100, 10),
}

if !limiters[provider].Allow() {
    // Skip this provider, it's rate limited
    continue
}
```

This ensures <1 minute recovery time even if a provider goes completely down."

---

### Q12: How do you handle database migrations in production without downtime?

**Answer:**

"We use golang-migrate with a zero-downtime strategy:

**1. Backward-compatible migrations**:
```sql
-- ✅ SAFE: Add nullable column
ALTER TABLE transactions ADD COLUMN input_data TEXT;

-- ❌ UNSAFE: Add NOT NULL column (requires default)
ALTER TABLE transactions ADD COLUMN input_data TEXT NOT NULL;

-- ✅ SAFE: Add column with default, then make NOT NULL
ALTER TABLE transactions ADD COLUMN input_data TEXT DEFAULT '';
-- Deploy code that populates input_data
-- Later migration: ALTER TABLE transactions ALTER COLUMN input_data SET NOT NULL;
```

**2. Multi-phase deployments**:

Phase 1 (expand):
```sql
-- Add new column, keep old column
ALTER TABLE blocks ADD COLUMN timestamp_v2 TIMESTAMPTZ;
```

Phase 2 (migrate data):
```sql
-- Backfill in batches to avoid locking
UPDATE blocks 
SET timestamp_v2 = timestamp::TIMESTAMPTZ 
WHERE timestamp_v2 IS NULL 
LIMIT 10000;
```

Phase 3 (contract):
```sql
-- Drop old column after all code uses new column
ALTER TABLE blocks DROP COLUMN timestamp;
ALTER TABLE blocks RENAME COLUMN timestamp_v2 TO timestamp;
```

**3. Online index creation**:
```sql
-- ❌ BAD: Locks table for minutes
CREATE INDEX idx_tx_hash ON transactions(tx_hash);

-- ✅ GOOD: No lock, can take longer but doesn't block writes
CREATE INDEX CONCURRENTLY idx_tx_hash ON transactions(tx_hash);
```

**4. Migration versioning**:
```
000001_initial_schema.up.sql
000001_initial_schema.down.sql
000002_add_calldata_parsing.up.sql
000002_add_calldata_parsing.down.sql
```

**5. Rollback plan**:
```bash
# Check current version
make migrate-status

# Rollback last migration
make migrate-down

# Force to specific version if needed
make migrate-force VERSION=3
```

This allows deploying schema changes with zero downtime and safe rollback capability."

---

## Related Documentation

- [Database Fundamentals](./Database-Fundamentals.md) - General database theory (ACID, normalization, indexing)
- [PostgreSQL Database](./PostgreSQL-Database.md) - PostgreSQL-specific features and optimization
- [Go Programming](./Go-Programming.md) - Go language patterns and concurrency
- [Message Queues](./Message-Queues.md) - Kafka and Redis deep dive
- [Deployment & Production](./Deployment-Production.md) - CI/CD, monitoring, cost optimization
- [Technical Specification](../docs/TECHNICAL_SPEC.md) - Detailed system architecture

---

*Last updated: 2024-11-27*
