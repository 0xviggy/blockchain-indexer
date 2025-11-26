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
	client, _ := ethclient.Dial(rpcURL)
	defer client.Close()

	// Test recent blocks
	testBlocks := []int64{23884349, 23884348, 23884347, 23884346, 23884345}
	
	for _, blockNum := range testBlocks {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		block, err := client.BlockByNumber(ctx, big.NewInt(blockNum))
		cancel()
		
		if err != nil {
			log.Printf("Block %d: ❌ ERROR - %v", blockNum, err)
			continue
		}
		
		// Count transaction types
		blobCount := 0
		for _, tx := range block.Transactions() {
			if tx.Type() == 3 {
				blobCount++
			}
		}
		
		if blobCount > 0 {
			log.Printf("Block %d: ✅ SUCCESS - %d transactions (%d blob txs)", blockNum, len(block.Transactions()), blobCount)
		} else {
			log.Printf("Block %d: ✅ SUCCESS - %d transactions (no blob txs)", blockNum, len(block.Transactions()))
		}
	}
}
