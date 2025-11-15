package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ChainConfig holds chain-specific configuration
type ChainConfig struct {
	Name            string
	ChainID         int64
	EnvVar          string
	ExampleRPCURL   string
	RecentBlock     int64 // A recent block for testing
	SupportsTracing bool  // Whether debug_traceTransaction is typically available
	BlockTime       int   // Average block time in seconds
}

// SupportedChains defines all chains we support (or plan to support)
var SupportedChains = []ChainConfig{
	// Tier 1: Priority chains (implement first)
	{
		Name:            "Ethereum",
		ChainID:         1,
		EnvVar:          "ETH_RPC_URL",
		ExampleRPCURL:   "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY",
		RecentBlock:     18500000,
		SupportsTracing: false, // Requires archive node
		BlockTime:       12,
	},
	{
		Name:            "Polygon",
		ChainID:         137,
		EnvVar:          "POLYGON_RPC_URL",
		ExampleRPCURL:   "https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY",
		RecentBlock:     50000000,
		SupportsTracing: false,
		BlockTime:       2,
	},
	{
		Name:            "Arbitrum",
		ChainID:         42161,
		EnvVar:          "ARBITRUM_RPC_URL",
		ExampleRPCURL:   "https://arb-mainnet.g.alchemy.com/v2/YOUR_KEY",
		RecentBlock:     150000000,
		SupportsTracing: false,
		BlockTime:       1,
	},
	{
		Name:            "Optimism",
		ChainID:         10,
		EnvVar:          "OPTIMISM_RPC_URL",
		ExampleRPCURL:   "https://opt-mainnet.g.alchemy.com/v2/YOUR_KEY",
		RecentBlock:     110000000,
		SupportsTracing: false,
		BlockTime:       2,
	},
	{
		Name:            "Base",
		ChainID:         8453,
		EnvVar:          "BASE_RPC_URL",
		ExampleRPCURL:   "https://base-mainnet.g.alchemy.com/v2/YOUR_KEY",
		RecentBlock:     5000000,
		SupportsTracing: false,
		BlockTime:       2,
	},

	// Tier 2: Next priority (high value, easy to add)
	{
		Name:            "BSC",
		ChainID:         56,
		EnvVar:          "BSC_RPC_URL",
		ExampleRPCURL:   "https://bsc-dataseed.binance.org/",
		RecentBlock:     34000000,
		SupportsTracing: false,
		BlockTime:       3,
	},
	{
		Name:            "Avalanche",
		ChainID:         43114,
		EnvVar:          "AVAX_RPC_URL",
		ExampleRPCURL:   "https://api.avax.network/ext/bc/C/rpc",
		RecentBlock:     38000000,
		SupportsTracing: false,
		BlockTime:       2,
	},
}

func main() {
	// Check which chains are configured
	configuredChains := []ChainConfig{}
	for _, chain := range SupportedChains {
		if rpcURL := os.Getenv(chain.EnvVar); rpcURL != "" {
			chain.ExampleRPCURL = rpcURL // Override with actual URL
			configuredChains = append(configuredChains, chain)
		}
	}

	if len(configuredChains) == 0 {
		log.Println("❌ No chain RPC URLs configured")
		log.Println("")
		log.Println("Configure at least one chain:")
		log.Println("")
		log.Println("Tier 1 Chains (Priority):")
		for _, chain := range SupportedChains[:5] {
			log.Printf("  %s: export %s=\"%s\"", chain.Name, chain.EnvVar, chain.ExampleRPCURL)
		}
		log.Println("")
		log.Println("Tier 2 Chains (Future):")
		for _, chain := range SupportedChains[5:] {
			log.Printf("  %s: export %s=\"%s\"", chain.Name, chain.EnvVar, chain.ExampleRPCURL)
		}
		log.Println("")
		log.Println("Get free API keys at:")
		log.Println("  - Alchemy: https://www.alchemy.com/")
		log.Println("  - Infura: https://infura.io/")
		os.Exit(1)
	}

	log.Println("🔍 === MULTI-CHAIN RPC EXPLORATION ===")
	log.Printf("Found %d configured chain(s)\n", len(configuredChains))
	log.Println("")

	ctx := context.Background()

	// Explore each configured chain
	for i, chain := range configuredChains {
		if i > 0 {
			log.Println("\n" + strings.Repeat("=", 80) + "\n")
		}

		log.Printf("🔗 === %s (Chain ID: %d) ===", chain.Name, chain.ChainID)
		log.Println("")

		if err := exploreChain(ctx, chain); err != nil {
			log.Printf("❌ Failed to explore %s: %v\n", chain.Name, err)
		}
	}

	log.Println("\n🎉 === EXPLORATION COMPLETE ===")
	log.Println("Review the output above and update LEARNING_GUIDE.md with findings")
	log.Println("")
	log.Println("Next steps:")
	log.Println("  1. Document any missing function signatures")
	log.Println("  2. Note which RPC methods are supported per chain")
	log.Println("  3. Update migration if needed")
}

func exploreChain(ctx context.Context, chain ChainConfig) error {
	// Connect to chain
	client, err := ethclient.Dial(chain.ExampleRPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	log.Printf("✅ Connected to %s RPC", chain.Name)
	log.Println("")

	// Verify chain ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Printf("⚠️  Could not verify chain ID: %v", err)
	} else if chainID.Int64() != chain.ChainID {
		log.Printf("⚠️  Warning: Expected chain ID %d but got %d", chain.ChainID, chainID.Int64())
	} else {
		log.Printf("✅ Chain ID verified: %d", chainID.Int64())
	}
	log.Println("")

	// Get latest block info
	if err := exploreLatestBlock(ctx, client, chain.Name); err != nil {
		log.Printf("❌ Error exploring latest block: %v\n", err)
	}

	// Explore a recent block with transactions
	if err := exploreRecentTransactions(ctx, client, chain); err != nil {
		log.Printf("❌ Error exploring transactions: %v\n", err)
	}

	// Test internal transaction tracing (if supported)
	if chain.SupportsTracing {
		if err := exploreInternalTransactions(ctx, chain.ExampleRPCURL, chain.Name); err != nil {
			log.Printf("⚠️  Internal transaction tracing: %v\n", err)
		}
	} else {
		log.Println("⚠️  Internal transaction tracing: Typically requires archive node")
		log.Println("    Skipping for this exploration")
	}

	return nil
}

func exploreLatestBlock(ctx context.Context, client *ethclient.Client, chainName string) error {
	log.Printf("📦 === LATEST BLOCK EXPLORATION (%s) ===", chainName)

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest header: %w", err)
	}

	log.Printf("Block Number: %d", header.Number.Uint64())
	log.Printf("Block Hash: %s", header.Hash().Hex())
	log.Printf("Parent Hash: %s", header.ParentHash.Hex())
	log.Printf("Timestamp: %d", header.Time)
	log.Printf("Gas Used: %d", header.GasUsed)
	log.Printf("Gas Limit: %d", header.GasLimit)

	// Get full block with transactions
	block, err := client.BlockByNumber(ctx, header.Number)
	if err != nil {
		return fmt.Errorf("failed to get full block: %w", err)
	}

	log.Printf("Transaction Count: %d", len(block.Transactions()))
	log.Printf("Miner: %s", block.Coinbase().Hex())
	log.Println("")

	return nil
}

func exploreRecentTransactions(ctx context.Context, client *ethclient.Client, chain ChainConfig) error {
	log.Printf("💸 === TRANSACTION EXPLORATION (%s) ===", chain.Name)

	// Get a recent block with transactions
	blockNumber := big.NewInt(chain.RecentBlock)
	block, err := client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	log.Printf("Exploring block %d with %d transactions\n", block.Number().Uint64(), len(block.Transactions()))

	// Explore first 3 transactions
	count := 3
	if len(block.Transactions()) < count {
		count = len(block.Transactions())
	}

	for i, tx := range block.Transactions()[:count] {
		log.Printf("\n--- Transaction %d ---", i+1)
		log.Printf("Hash: %s", tx.Hash().Hex())
		log.Printf("From: %s", getSender(tx))
		if tx.To() != nil {
			log.Printf("To: %s", tx.To().Hex())
		} else {
			log.Printf("To: nil (contract creation)")
		}
		log.Printf("Value: %s wei", tx.Value().String())
		log.Printf("Gas: %d", tx.Gas())
		log.Printf("Gas Price: %s wei", tx.GasPrice().String())

		// Analyze input data
		data := tx.Data()
		if len(data) >= 4 {
			funcSig := "0x" + hex.EncodeToString(data[:4])
			log.Printf("Function Signature: %s", funcSig)
			log.Printf("Input Data Length: %d bytes", len(data))

			// Check if it matches known signatures
			checkKnownSignature(funcSig)
		} else if len(data) > 0 {
			log.Printf("Input Data: %s (not a function call)", hex.EncodeToString(data))
		} else {
			log.Printf("Input Data: empty (simple ETH transfer)")
		}

		// Get transaction receipt
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			log.Printf("⚠️  Could not get receipt: %v", err)
			continue
		}

		log.Printf("Status: %d (1=success, 0=failed)", receipt.Status)
		log.Printf("Gas Used: %d", receipt.GasUsed)
		log.Printf("Logs/Events: %d", len(receipt.Logs))

		// Explore events
		if len(receipt.Logs) > 0 {
			log.Printf("\n  First Event:")
			firstLog := receipt.Logs[0]
			log.Printf("    Contract: %s", firstLog.Address.Hex())
			if len(firstLog.Topics) > 0 {
				log.Printf("    Event Signature: %s", firstLog.Topics[0].Hex())
				checkKnownEventSignature(firstLog.Topics[0].Hex())
			}
			log.Printf("    Topics: %d", len(firstLog.Topics))
			log.Printf("    Data: %d bytes", len(firstLog.Data))
		}
	}

	log.Println("")
	return nil
}

func exploreInternalTransactions(ctx context.Context, rpcURL string, chainName string) error {
	log.Printf("🔍 === INTERNAL TRANSACTION TRACING (%s) ===", chainName)

	// Use raw RPC client for debug/trace methods
	rpcClient, err := rpc.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer rpcClient.Close()

	// Example transaction hash - replace with a real one that has internal txs
	// This is a known Uniswap transaction with internal calls
	txHash := "0x5c504ed432cb51138bcf09aa5e8a410dd4a1e204ef84bfed1be16dfba1b22060"

	log.Printf("⚠️  Testing trace methods (may not be supported by all RPC providers)")
	log.Printf("Attempting to trace tx: %s", txHash)
	log.Println("")

	// Try debug_traceTransaction
	var traceResult interface{}
	err = rpcClient.CallContext(ctx, &traceResult, "debug_traceTransaction", txHash, map[string]interface{}{
		"tracer": "callTracer",
	})

	if err != nil {
		log.Printf("❌ debug_traceTransaction not supported or tx not found: %v", err)
		log.Printf("   Note: This requires an archive node with debug APIs enabled")
		log.Printf("   Alternatives: Use trace_transaction or parity_trace")
		log.Println("")
	} else {
		log.Printf("✅ debug_traceTransaction works!")
		traceJSON, _ := json.MarshalIndent(traceResult, "", "  ")
		log.Printf("Trace result (first 500 chars):\n%s...", limitString(string(traceJSON), 500))
		log.Println("")
	}

	// Try trace_transaction (OpenEthereum/Erigon)
	var traceResult2 interface{}
	err = rpcClient.CallContext(ctx, &traceResult2, "trace_transaction", txHash)
	if err != nil {
		log.Printf("❌ trace_transaction not supported: %v", err)
		log.Println("")
	} else {
		log.Printf("✅ trace_transaction works!")
		log.Println("")
	}

	return nil
}

func getSender(tx *types.Transaction) string {
	// Note: This requires the chain ID to recover sender
	// For production, use types.Sender() with proper signer
	from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		return "unknown"
	}
	return from.Hex()
}

func checkKnownSignature(sig string) {
	signatures := map[string]string{
		"0x38ed1739": "swapExactTokensForTokens (Uniswap V2)",
		"0x7ff36ab5": "swapExactETHForTokens (Uniswap V2)",
		"0x18cbafe5": "swapExactTokensForETH (Uniswap V2)",
		"0x414bf389": "exactInputSingle (Uniswap V3)",
		"0xc04b8d59": "exactInput (Uniswap V3)",
		"0xa9059cbb": "transfer (ERC20)",
		"0x23b872dd": "transferFrom (ERC20)",
		"0x095ea7b3": "approve (ERC20)",
		"0x42842e0e": "safeTransferFrom (ERC721)",
		"0x1a0a6e":   "send (LayerZero)",
		"0x3687011a": "deposit (Across Bridge)",
		"0x0f5287b0": "swap (Stargate)",
		"0x7c025200": "swap (1inch)",
		"0x3df02124": "exchange (Curve)",
		"0xe8eda9df": "supply (Aave V3)",
		"0x69328dec": "withdraw (Aave V3)",
	}

	if name, ok := signatures[sig]; ok {
		log.Printf("    ✅ KNOWN: %s", name)
	} else {
		log.Printf("    ❓ Unknown signature")
		log.Printf("       Look up at: https://www.4byte.directory/signatures/?bytes4_signature=%s", sig)
	}
}

func checkKnownEventSignature(sig string) {
	events := map[string]string{
		"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef": "Transfer (ERC20/ERC721)",
		"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925": "Approval (ERC20)",
		"0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62": "Swap (Uniswap V2)",
		"0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67": "Swap (Uniswap V3)",
	}

	if name, ok := events[sig]; ok {
		log.Printf("       ✅ KNOWN EVENT: %s", name)
	} else {
		log.Printf("       ❓ Unknown event signature")
	}
}

func limitString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
