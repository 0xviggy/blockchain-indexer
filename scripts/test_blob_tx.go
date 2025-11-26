package main

import (
	"context"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL environment variable not set")
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()

	// Test block 23883999 - known to contain blob transactions
	blockNum := int64(23883999)
	
	log.Printf("Testing block %d with go-ethereum v1.16.7...", blockNum)
	log.Printf("RPC URL: %s", rpcURL[:50]+"...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	block, err := client.BlockByNumber(ctx, big.NewInt(blockNum))
	if err != nil {
		log.Printf("❌ FAILED: %v", err)
		log.Printf("Error type: %T", err)
		log.Printf("Error message contains 'transaction type not supported': %v", 
			err.Error() == "transaction type not supported" || 
			err.Error() == "transaction type 3 not supported")
		os.Exit(1)
	}

	log.Printf("✅ SUCCESS: Block %d fetched successfully!", blockNum)
	log.Printf("Block hash: %s", block.Hash().Hex())
	log.Printf("Transactions: %d", len(block.Transactions()))
	log.Printf("Gas used: %d", block.GasUsed())
	log.Printf("Timestamp: %s", time.Unix(int64(block.Time()), 0).Format(time.RFC3339))

	// Check transaction types
	txTypes := make(map[uint8]int)
	for _, tx := range block.Transactions() {
		txTypes[tx.Type()]++
	}
	
	log.Printf("\nTransaction type breakdown:")
	for txType, count := range txTypes {
		typeName := "Unknown"
		switch txType {
		case 0:
			typeName = "Legacy"
		case 1:
			typeName = "EIP-2930 (Access List)"
		case 2:
			typeName = "EIP-1559 (Dynamic Fee)"
		case 3:
			typeName = "EIP-4844 (Blob)"
		}
		log.Printf("  Type %d (%s): %d transactions", txType, typeName, count)
	}

	// Test fetching a receipt
	if len(block.Transactions()) > 0 {
		firstTx := block.Transactions()[0]
		log.Printf("\nTesting receipt fetch for first transaction...")
		receiptCtx, receiptCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer receiptCancel()
		
		receipt, err := client.TransactionReceipt(receiptCtx, firstTx.Hash())
		if err != nil {
			log.Printf("❌ Receipt fetch failed: %v", err)
		} else {
			log.Printf("✅ Receipt fetched successfully")
			log.Printf("  Status: %d", receipt.Status)
			log.Printf("  Gas used: %d", receipt.GasUsed)
		}
	}
}
