package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/lib/pq"
)

type ChainConfig struct {
	ChainID    int64
	Name       string
	RPCURL     string
	WSUrl      string
	StartBlock int64 // Block to start indexing from
}

type Ingester struct {
	db           *sql.DB
	chains       []ChainConfig
	clients      map[int64]*ethclient.Client
	wsClients    map[int64]*ethclient.Client
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownOnce sync.Once
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting Multi-Chain Blockchain Ingester")

	// Load configuration
	chains := loadChainConfigs()
	if len(chains) == 0 {
		log.Fatal("❌ No chains configured. Set RPC URLs in environment variables.")
	}

	log.Printf("📋 Configured chains: %d", len(chains))
	for _, chain := range chains {
		log.Printf("  - %s (Chain ID: %d)", chain.Name, chain.ChainID)
	}

	// Connect to database
	db, err := connectDB()
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}
	defer db.Close()
	log.Println("✅ Database connected")

	// Initialize ingester
	ctx, cancel := context.WithCancel(context.Background())
	ingester := &Ingester{
		db:        db,
		chains:    chains,
		clients:   make(map[int64]*ethclient.Client),
		wsClients: make(map[int64]*ethclient.Client),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Connect to all chains
	if err := ingester.connectChains(); err != nil {
		log.Fatalf("❌ Failed to connect to chains: %v", err)
	}

	// Ensure chains are in database
	if err := ingester.ensureChainsExist(); err != nil {
		log.Fatalf("❌ Failed to initialize chains in database: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n⚠️  Shutdown signal received, gracefully stopping...")
		ingester.shutdown()
	}()

	// Start ingesting for all chains
	log.Println("🎬 Starting block ingestion for all chains...")
	for _, chain := range chains {
		ingester.wg.Add(1)
		go ingester.ingestChain(chain)
	}

	// Wait for all goroutines
	ingester.wg.Wait()
	log.Println("✅ All chains stopped. Goodbye!")
}

func loadChainConfigs() []ChainConfig {
	var configs []ChainConfig

	// Ethereum
	if rpcURL := os.Getenv("ETH_RPC_URL"); rpcURL != "" {
		configs = append(configs, ChainConfig{
			ChainID:    1,
			Name:       "Ethereum",
			RPCURL:     rpcURL,
			WSUrl:      os.Getenv("ETH_WS_URL"),
			StartBlock: getEnvInt64("ETH_START_BLOCK", 0),
		})
	}

	// Polygon
	if rpcURL := os.Getenv("POLYGON_RPC_URL"); rpcURL != "" {
		configs = append(configs, ChainConfig{
			ChainID:    137,
			Name:       "Polygon",
			RPCURL:     rpcURL,
			WSUrl:      os.Getenv("POLYGON_WS_URL"),
			StartBlock: getEnvInt64("POLYGON_START_BLOCK", 0),
		})
	}

	// Arbitrum
	if rpcURL := os.Getenv("ARBITRUM_RPC_URL"); rpcURL != "" {
		configs = append(configs, ChainConfig{
			ChainID:    42161,
			Name:       "Arbitrum",
			RPCURL:     rpcURL,
			WSUrl:      os.Getenv("ARBITRUM_WS_URL"),
			StartBlock: getEnvInt64("ARBITRUM_START_BLOCK", 0),
		})
	}

	// Optimism
	if rpcURL := os.Getenv("OPTIMISM_RPC_URL"); rpcURL != "" {
		configs = append(configs, ChainConfig{
			ChainID:    10,
			Name:       "Optimism",
			RPCURL:     rpcURL,
			WSUrl:      os.Getenv("OPTIMISM_WS_URL"),
			StartBlock: getEnvInt64("OPTIMISM_START_BLOCK", 0),
		})
	}

	// Base
	if rpcURL := os.Getenv("BASE_RPC_URL"); rpcURL != "" {
		configs = append(configs, ChainConfig{
			ChainID:    8453,
			Name:       "Base",
			RPCURL:     rpcURL,
			WSUrl:      os.Getenv("BASE_WS_URL"),
			StartBlock: getEnvInt64("BASE_START_BLOCK", 0),
		})
	}

	return configs
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		var result int64
		if _, err := fmt.Sscanf(val, "%d", &result); err == nil {
			return result
		}
	}
	return defaultVal
}

func connectDB() (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://indexer:password@localhost:5432/indexer?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func (ing *Ingester) connectChains() error {
	for _, chain := range ing.chains {
		// Connect HTTP client
		client, err := ethclient.Dial(chain.RPCURL)
		if err != nil {
			return fmt.Errorf("failed to connect to %s RPC: %w", chain.Name, err)
		}
		ing.clients[chain.ChainID] = client

		// Verify chain ID
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		chainID, err := client.ChainID(ctx)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to get chain ID for %s: %w", chain.Name, err)
		}
		if chainID.Int64() != chain.ChainID {
			return fmt.Errorf("chain ID mismatch for %s: expected %d, got %d",
				chain.Name, chain.ChainID, chainID.Int64())
		}

		log.Printf("✅ Connected to %s (Chain ID: %d)", chain.Name, chain.ChainID)

		// Connect WebSocket if available
		if chain.WSUrl != "" {
			wsClient, err := ethclient.Dial(chain.WSUrl)
			if err != nil {
				log.Printf("⚠️  WebSocket connection failed for %s, will use HTTP polling: %v",
					chain.Name, err)
			} else {
				ing.wsClients[chain.ChainID] = wsClient
				log.Printf("✅ WebSocket connected for %s", chain.Name)
			}
		}
	}
	return nil
}

func (ing *Ingester) ensureChainsExist() error {
	query := `
		INSERT INTO chains (chain_id, chain_name, rpc_url, ws_url, block_time_seconds, enabled)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (chain_id) DO UPDATE
		SET chain_name = EXCLUDED.chain_name, 
		    rpc_url = EXCLUDED.rpc_url,
		    ws_url = EXCLUDED.ws_url,
		    enabled = true, 
		    updated_at = NOW()
	`

	for _, chain := range ing.chains {
		// Default block times for common chains
		blockTime := 12 // Default for Ethereum
		if chain.ChainID == 137 {
			blockTime = 2 // Polygon
		} else if chain.ChainID == 42161 || chain.ChainID == 10 || chain.ChainID == 8453 {
			blockTime = 2 // Arbitrum, Optimism, Base
		}
		
		wsURL := sql.NullString{String: chain.WSUrl, Valid: chain.WSUrl != ""}
		if _, err := ing.db.Exec(query, chain.ChainID, chain.Name, chain.RPCURL, wsURL, blockTime); err != nil {
			return fmt.Errorf("failed to insert chain %s: %w", chain.Name, err)
		}
	}

	log.Println("✅ All chains initialized in database")
	return nil
}

func (ing *Ingester) ingestChain(chain ChainConfig) {
	defer ing.wg.Done()

	log.Printf("🔄 [%s] Starting ingestion...", chain.Name)

	client := ing.clients[chain.ChainID]

	// Get last processed block from checkpoint
	lastBlock, err := ing.getLastProcessedBlock(chain.ChainID)
	if err != nil {
		log.Printf("❌ [%s] Failed to get checkpoint: %v", chain.Name, err)
		return
	}

	// If no checkpoint and StartBlock specified, use it
	if lastBlock == 0 && chain.StartBlock > 0 {
		lastBlock = chain.StartBlock - 1
		log.Printf("📍 [%s] Starting from configured block: %d", chain.Name, chain.StartBlock)
	}

	// If still 0, get current block and start from there
	if lastBlock == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		header, err := client.HeaderByNumber(ctx, nil)
		cancel()
		if err != nil {
			log.Printf("❌ [%s] Failed to get latest block: %v", chain.Name, err)
			return
		}
		lastBlock = header.Number.Int64() - 1
		log.Printf("📍 [%s] Starting from current block: %d", chain.Name, lastBlock+1)
	}

	// Use WebSocket subscription if available, otherwise poll
	wsClient := ing.wsClients[chain.ChainID]
	if wsClient != nil {
		ing.subscribeToBlocks(chain, wsClient, lastBlock)
	} else {
		ing.pollBlocks(chain, client, lastBlock)
	}
}

func (ing *Ingester) subscribeToBlocks(chain ChainConfig, client *ethclient.Client, startBlock int64) {
	log.Printf("📡 [%s] Using WebSocket subscription", chain.Name)

	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(ing.ctx, headers)
	if err != nil {
		log.Printf("❌ [%s] Failed to subscribe to blocks: %v", chain.Name, err)
		// Fallback to polling
		ing.pollBlocks(chain, ing.clients[chain.ChainID], startBlock)
		return
	}
	defer sub.Unsubscribe()

	// Catch up with missed blocks first
	if err := ing.catchUpBlocks(chain, startBlock); err != nil {
		log.Printf("⚠️  [%s] Catch-up failed: %v", chain.Name, err)
	}

	log.Printf("✅ [%s] Subscribed to new blocks", chain.Name)

	for {
		select {
		case <-ing.ctx.Done():
			log.Printf("🛑 [%s] Stopping WebSocket subscription", chain.Name)
			return
		case err := <-sub.Err():
			log.Printf("❌ [%s] Subscription error: %v", chain.Name, err)
			// Attempt to reconnect
			time.Sleep(5 * time.Second)
			lastBlock, _ := ing.getLastProcessedBlock(chain.ChainID)
			ing.subscribeToBlocks(chain, client, lastBlock)
			return
		case header := <-headers:
			blockNum := header.Number.Int64()
			if err := ing.processBlock(chain, blockNum); err != nil {
				log.Printf("❌ [%s] Failed to process block %d: %v",
					chain.Name, blockNum, err)
			} else {
				log.Printf("✅ [%s] Block %d processed",
					chain.Name, blockNum)
			}
		}
	}
}

func (ing *Ingester) pollBlocks(chain ChainConfig, client *ethclient.Client, startBlock int64) {
	log.Printf("🔄 [%s] Using HTTP polling mode", chain.Name)

	currentBlock := startBlock + 1
	ticker := time.NewTicker(12 * time.Second) // Ethereum block time
	defer ticker.Stop()

	for {
		select {
		case <-ing.ctx.Done():
			log.Printf("🛑 [%s] Stopping block polling", chain.Name)
			return
		case <-ticker.C:
			// Get latest block
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			header, err := client.HeaderByNumber(ctx, nil)
			cancel()
			if err != nil {
				log.Printf("❌ [%s] Failed to get latest block: %v", chain.Name, err)
				continue
			}

			latestBlock := header.Number.Int64()

			// Process all blocks up to latest
			for currentBlock <= latestBlock {
				if err := ing.processBlock(chain, currentBlock); err != nil {
					log.Printf("❌ [%s] Failed to process block %d: %v",
						chain.Name, currentBlock, err)
					time.Sleep(1 * time.Second)
					continue
				}
				log.Printf("✅ [%s] Block %d processed", chain.Name, currentBlock)
				currentBlock++
			}
		}
	}
}

func (ing *Ingester) catchUpBlocks(chain ChainConfig, startBlock int64) error {
	client := ing.clients[chain.ChainID]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	header, err := client.HeaderByNumber(ctx, nil)
	cancel()
	if err != nil {
		return err
	}

	latestBlock := header.Number.Int64()
	if startBlock >= latestBlock {
		return nil
	}

	log.Printf("⏳ [%s] Catching up from block %d to %d (%d blocks)",
		chain.Name, startBlock+1, latestBlock, latestBlock-startBlock)

	for blockNum := startBlock + 1; blockNum <= latestBlock; blockNum++ {
		select {
		case <-ing.ctx.Done():
			return fmt.Errorf("shutdown during catch-up")
		default:
			if err := ing.processBlock(chain, blockNum); err != nil {
				log.Printf("❌ [%s] Catch-up failed at block %d: %v", chain.Name, blockNum, err)
				return err
			}
			if blockNum%10 == 0 {
				log.Printf("⏳ [%s] Catch-up progress: %d/%d", chain.Name, blockNum, latestBlock)
			}
		}
	}

	log.Printf("✅ [%s] Catch-up complete", chain.Name)
	return nil
}

func (ing *Ingester) processBlock(chain ChainConfig, blockNum int64) error {
	client := ing.clients[chain.ChainID]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch block with transactions
	block, err := client.BlockByNumber(ctx, big.NewInt(blockNum))
	if err != nil {
		return fmt.Errorf("failed to fetch block: %w", err)
	}

	// Start transaction
	tx, err := ing.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert block
	if err := ing.insertBlock(tx, chain.ChainID, block); err != nil {
		return fmt.Errorf("failed to insert block: %w", err)
	}

	// Insert transactions with receipts
	for txIndex, ethTx := range block.Transactions() {
		if err := ing.insertTransaction(tx, chain.ChainID, block, ethTx, uint(txIndex), client, ctx); err != nil {
			return fmt.Errorf("failed to insert transaction %s: %w", ethTx.Hash().Hex(), err)
		}
	}

	// Update checkpoint
	if err := ing.updateCheckpoint(tx, chain.ChainID, blockNum); err != nil {
		return fmt.Errorf("failed to update checkpoint: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (ing *Ingester) insertBlock(tx *sql.Tx, chainID int64, block *types.Block) error {
	query := `
		INSERT INTO blocks (
			chain_id, block_number, block_hash, parent_hash, timestamp,
			gas_used, gas_limit, base_fee_per_gas, transaction_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (chain_id, block_number) DO UPDATE
		SET block_hash = EXCLUDED.block_hash,
		    parent_hash = EXCLUDED.parent_hash,
		    timestamp = EXCLUDED.timestamp,
		    gas_used = EXCLUDED.gas_used,
		    gas_limit = EXCLUDED.gas_limit,
		    base_fee_per_gas = EXCLUDED.base_fee_per_gas,
		    transaction_count = EXCLUDED.transaction_count,
		    updated_at = NOW()
	`

	baseFee := "0"
	if block.BaseFee() != nil {
		baseFee = block.BaseFee().String()
	}

	_, err := tx.Exec(query,
		chainID,
		block.Number().Int64(),
		block.Hash().Hex(),
		block.ParentHash().Hex(),
		time.Unix(int64(block.Time()), 0),
		block.GasUsed(),
		block.GasLimit(),
		baseFee,
		len(block.Transactions()),
	)

	return err
}

func (ing *Ingester) insertTransaction(tx *sql.Tx, chainID int64, block *types.Block, ethTx *types.Transaction, txIndex uint, client *ethclient.Client, ctx context.Context) error {
	query := `
		INSERT INTO transactions (
			chain_id, tx_hash, block_number, tx_index, from_address, to_address,
			value, gas_limit, gas_price, input_data, nonce, status, gas_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (chain_id, tx_hash) DO NOTHING
	`

	// Get transaction sender
	from, err := types.Sender(types.LatestSignerForChainID(big.NewInt(chainID)), ethTx)
	if err != nil {
		return fmt.Errorf("failed to get sender: %w", err)
	}

	toAddr := ""
	if ethTx.To() != nil {
		toAddr = ethTx.To().Hex()
	}

	// Fetch transaction receipt to get actual status and gas used
	receipt, err := client.TransactionReceipt(ctx, ethTx.Hash())
	if err != nil {
		// Log warning but continue with unknown status
		log.Printf("⚠️  Failed to get receipt for tx %s: %v (using status=1, gas_used=0)", ethTx.Hash().Hex(), err)
		// Use defaults if receipt fetch fails
		receipt = &types.Receipt{
			Status:           1, // Assume success
			GasUsed:          0,
			TransactionIndex: txIndex,
		}
	}

	_, err = tx.Exec(query,
		chainID,
		ethTx.Hash().Hex(),
		block.Number().Int64(),
		receipt.TransactionIndex,
		from.Hex(),
		toAddr,
		ethTx.Value().String(),
		ethTx.Gas(),
		ethTx.GasPrice().String(),
		ethTx.Data(),
		ethTx.Nonce(),
		receipt.Status,  // Actual status from receipt (0=fail, 1=success)
		receipt.GasUsed, // Actual gas consumed
	)

	return err
}

func (ing *Ingester) getLastProcessedBlock(chainID int64) (int64, error) {
	query := `SELECT last_processed_block FROM checkpoints WHERE chain_id = $1`

	var lastBlock int64
	err := ing.db.QueryRow(query, chainID).Scan(&lastBlock)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return lastBlock, nil
}

func (ing *Ingester) updateCheckpoint(tx *sql.Tx, chainID int64, blockNum int64) error {
	query := `
		INSERT INTO checkpoints (chain_id, service_name, last_processed_block)
		VALUES ($1, 'ingester', $2)
		ON CONFLICT (chain_id, service_name)
		DO UPDATE SET last_processed_block = EXCLUDED.last_processed_block,
		              updated_at = NOW()
	`

	_, err := tx.Exec(query, chainID, blockNum)
	return err
}

func (ing *Ingester) shutdown() {
	ing.shutdownOnce.Do(func() {
		ing.cancel()

		// Close all connections
		for chainID, client := range ing.clients {
			client.Close()
			log.Printf("🔌 Closed HTTP client for chain %d", chainID)
		}
		for chainID, client := range ing.wsClients {
			client.Close()
			log.Printf("🔌 Closed WebSocket client for chain %d", chainID)
		}
	})
}
