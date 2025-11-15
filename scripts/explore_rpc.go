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

	// Signature analysis across multiple blocks
	log.Println("\n" + strings.Repeat("=", 80))
	log.Println("\n📊 === SIGNATURE ANALYSIS ACROSS MULTIPLE BLOCKS ===")
	if err := analyzeSignatures(ctx, configuredChains); err != nil {
		log.Printf("⚠️  Signature analysis: %v\n", err)
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
		// Uniswap
		"0x38ed1739": "swapExactTokensForTokens (Uniswap V2)",
		"0x7ff36ab5": "swapExactETHForTokens (Uniswap V2)",
		"0x18cbafe5": "swapExactTokensForETH (Uniswap V2)",
		"0x414bf389": "exactInputSingle (Uniswap V3)",
		"0xc04b8d59": "exactInput (Uniswap V3)",
		"0x3593564c": "execute (Uniswap Universal Router)",
		"0x24856bc3": "execute (Uniswap Universal Router, no deadline)",
		// ERC20
		"0xa9059cbb": "transfer (ERC20)",
		"0x23b872dd": "transferFrom (ERC20)",
		"0x095ea7b3": "approve (ERC20)",
		// ERC721
		"0x42842e0e": "safeTransferFrom (ERC721)",
		// Bridges
		"0x1a0a6e":   "send (LayerZero)",
		"0x3687011a": "deposit (Across Bridge)",
		"0x0f5287b0": "swap (Stargate)",
		// DEX Aggregators
		"0x7c025200": "swap (1inch)",
		"0xa08edebc": "swap (Metamask Router)",
		"0x13d79a0b": "settle (CoW Protocol)",
		// Other DeFi
		"0x3df02124": "exchange (Curve)",
		"0xe8eda9df": "supply (Aave V3)",
		"0x69328dec": "withdraw (Aave V3)",
		"0x30f28b7a": "permit (Permit2)",
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

// analyzeSignatures scans multiple blocks to find unknown function signatures
func analyzeSignatures(ctx context.Context, chains []ChainConfig) error {
	for _, chain := range chains {
		log.Printf("\n🔍 Analyzing signatures for %s...", chain.Name)

		client, err := ethclient.Dial(chain.ExampleRPCURL)
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %w", chain.Name, err)
		}
		defer client.Close()

		// Get latest block number
		header, err := client.HeaderByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get latest header: %w", err)
		}

		latestBlock := header.Number.Int64()
		log.Printf("Latest block: %d", latestBlock)
		log.Printf("Scanning last 10 blocks for signatures...\n")

		// Track signature frequency
		signatureCounts := make(map[string]int)
		unknownSignatures := make(map[string][]string) // sig -> sample tx hashes
		totalTxs := 0
		totalWithCalldata := 0

		// Scan last 10 blocks
		for i := latestBlock - 9; i <= latestBlock; i++ {
			block, err := client.BlockByNumber(ctx, big.NewInt(i))
			if err != nil {
				log.Printf("⚠️  Could not fetch block %d: %v", i, err)
				continue
			}

			for _, tx := range block.Transactions() {
				totalTxs++
				data := tx.Data()

				if len(data) >= 4 {
					totalWithCalldata++
					funcSig := "0x" + hex.EncodeToString(data[:4])
					signatureCounts[funcSig]++

					// Check if unknown
					if !isKnownSignature(funcSig) {
						if txHashes, exists := unknownSignatures[funcSig]; exists {
							// Only keep first 3 examples
							if len(txHashes) < 3 {
								unknownSignatures[funcSig] = append(txHashes, tx.Hash().Hex())
							}
						} else {
							unknownSignatures[funcSig] = []string{tx.Hash().Hex()}
						}
					}
				}
			}
		}

		// Report statistics
		log.Printf("\n📈 Statistics:")
		log.Printf("  Total transactions: %d", totalTxs)
		log.Printf("  With calldata: %d (%.1f%%)", totalWithCalldata, float64(totalWithCalldata)/float64(totalTxs)*100)
		log.Printf("  Unique signatures: %d", len(signatureCounts))
		log.Printf("  Unknown signatures: %d", len(unknownSignatures))

		// Show top 10 most common signatures
		log.Printf("\n🔥 Top 10 Most Common Signatures:")
		type sigFreq struct {
			sig   string
			count int
			known bool
		}
		var frequencies []sigFreq
		for sig, count := range signatureCounts {
			frequencies = append(frequencies, sigFreq{sig, count, isKnownSignature(sig)})
		}
		// Sort by count
		for i := 0; i < len(frequencies); i++ {
			for j := i + 1; j < len(frequencies); j++ {
				if frequencies[j].count > frequencies[i].count {
					frequencies[i], frequencies[j] = frequencies[j], frequencies[i]
				}
			}
		}
		for i := 0; i < 10 && i < len(frequencies); i++ {
			freq := frequencies[i]
			status := "✅"
			if !freq.known {
				status = "❌ UNKNOWN"
			}
			log.Printf("  %d. %s - %d calls %s", i+1, freq.sig, freq.count, status)
			if freq.known {
				log.Printf("      %s", getSignatureName(freq.sig))
			}
		}

		// Show unknown signatures with examples
		if len(unknownSignatures) > 0 {
			log.Printf("\n❓ Unknown Signatures Found (add these to database):")
			log.Println("```sql")
			log.Println("-- Add to protocol_signatures table:")

			count := 0
			for sig, txHashes := range unknownSignatures {
				count++
				if count > 20 { // Limit output
					log.Printf("-- ... and %d more", len(unknownSignatures)-20)
					break
				}

				log.Printf("-- Signature: %s", sig)
				log.Printf("--   Found in: %d transactions", signatureCounts[sig])
				log.Printf("--   Example tx: %s", txHashes[0])
				log.Printf("--   Lookup: https://www.4byte.directory/signatures/?bytes4_signature=%s", sig)
				log.Printf("-- ('YOUR_SIG', 'function_name', 'protocol', 'abi(...)', 'description'),")
				log.Println()
			}
			log.Println("```")
		} else {
			log.Println("\n✅ All signatures are known!")
		}
	}

	return nil
}

func isKnownSignature(sig string) bool {
	signatures := map[string]bool{
		// Uniswap
		"0x38ed1739": true, "0x7ff36ab5": true, "0x18cbafe5": true,
		"0x414bf389": true, "0xc04b8d59": true, "0x3593564c": true, "0x24856bc3": true,
		"0x791ac947": true, "0x3d0e3ec5": true,
		// ERC20/ERC721 (0x23b872dd is transferFrom for both)
		"0xa9059cbb": true, "0x23b872dd": true, "0x095ea7b3": true,
		"0x42842e0e": true, "0xb88d4fde": true,
		// Bridges
		"0x1a0a6e": true, "0x3687011a": true, "0x0f5287b0": true,
		// DEX Aggregators
		"0x7c025200": true, "0xa08edebc": true, "0x13d79a0b": true,
		// Other DeFi
		"0x3df02124": true, "0xe8eda9df": true, "0x69328dec": true, "0x30f28b7a": true,
		// Forwarders & Staking (high-volume from scan)
		"0x78e111f6": true, "0x122067ed": true, "0x88ffe867": true, "0x6fadcf72": true,
	}
	return signatures[sig]
}

func getSignatureName(sig string) string {
	signatures := map[string]string{
		"0x38ed1739": "swapExactTokensForTokens (Uniswap V2)",
		"0x7ff36ab5": "swapExactETHForTokens (Uniswap V2)",
		"0x18cbafe5": "swapExactTokensForETH (Uniswap V2)",
		"0x414bf389": "exactInputSingle (Uniswap V3)",
		"0xc04b8d59": "exactInput (Uniswap V3)",
		"0x3593564c": "execute (Uniswap Universal Router)",
		"0x24856bc3": "execute (Uniswap Universal Router)",
		"0xa9059cbb": "transfer (ERC20)",
		"0x23b872dd": "transferFrom (ERC20/ERC721)",
		"0x095ea7b3": "approve (ERC20)",
		"0x42842e0e": "safeTransferFrom (ERC721)",
		"0xb88d4fde": "safeTransferFrom (ERC721 with data)",
		"0x1a0a6e":   "send (LayerZero)",
		"0x3687011a": "deposit (Across Bridge)",
		"0x0f5287b0": "swap (Stargate)",
		"0x7c025200": "swap (1inch)",
		"0xa08edebc": "swap (Metamask Router)",
		"0x13d79a0b": "settle (CoW Protocol)",
		"0x3df02124": "exchange (Curve)",
		"0xe8eda9df": "supply (Aave V3)",
		"0x69328dec": "withdraw (Aave V3)",
		"0x30f28b7a": "permit (Permit2)",
		"0x78e111f6": "executeFFsYo (Meta-tx Forwarder)",
		"0x122067ed": "unknown_swap (Aggregator)",
		"0x88ffe867": "pledge (Staking)",
		"0x6fadcf72": "forward (Forwarder)",
		"0x791ac947": "swapExactTokensForETHSupportingFeeOnTransferTokens (Uniswap V2)",
		"0x3d0e3ec5": "swapExactTokensForETHSupportingFeeOnTransferTokens (Custom DEX)",
	}
	if name, ok := signatures[sig]; ok {
		return name
	}
	return "Unknown"
}
