package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// ============================================================================
// Types
// ============================================================================

type API struct {
	db     *sql.DB
	router *gin.Engine
}

// Response types
type Chain struct {
	ChainID     int64     `json:"chain_id"`
	Name        string    `json:"name"`
	IsActive    bool      `json:"is_active"`
	LastBlock   *int64    `json:"last_block,omitempty"`
	BlockCount  *int64    `json:"block_count,omitempty"`
	TxCount     *int64    `json:"tx_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Block struct {
	ChainID          int64     `json:"chain_id"`
	BlockNumber      int64     `json:"block_number"`
	BlockHash        string    `json:"block_hash"`
	ParentHash       string    `json:"parent_hash"`
	Timestamp        time.Time `json:"timestamp"`
	GasUsed          int64     `json:"gas_used"`
	GasLimit         int64     `json:"gas_limit"`
	BaseFeePerGas    *string   `json:"base_fee_per_gas,omitempty"`
	TransactionCount int       `json:"transaction_count"`
	CreatedAt        time.Time `json:"created_at"`
}

type Transaction struct {
	ChainID     int64     `json:"chain_id"`
	TxHash      string    `json:"tx_hash"`
	BlockNumber int64     `json:"block_number"`
	TxIndex     int       `json:"tx_index"`
	FromAddress string    `json:"from_address"`
	ToAddress   *string   `json:"to_address,omitempty"`
	Value       string    `json:"value"`
	GasLimit    int64     `json:"gas_limit"`
	GasPrice    string    `json:"gas_price"`
	InputData   []byte    `json:"input_data,omitempty"`
	Nonce       int64     `json:"nonce"`
	Status      int       `json:"status"`
	GasUsed     int64     `json:"gas_used"`
	CreatedAt   time.Time `json:"created_at"`
}

type ChainStats struct {
	ChainID         int64    `json:"chain_id"`
	Name            string   `json:"name"`
	TotalBlocks     int64    `json:"total_blocks"`
	TotalTxs        int64    `json:"total_transactions"`
	LastProcessed   *int64   `json:"last_processed_block,omitempty"`
	AvgBlockTime    *float64 `json:"avg_block_time_seconds,omitempty"`
	AvgTxsPerBlock  *float64 `json:"avg_txs_per_block,omitempty"`
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Database  string            `json:"database"`
	Chains    map[string]string `json:"chains"`
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting Blockchain Indexer API")

	// Connect to database
	db, err := connectDB()
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}
	defer db.Close()
	log.Println("✅ Database connected")

	// Initialize API
	api := &API{
		db:     db,
		router: gin.Default(),
	}

	// Setup routes
	api.setupRoutes()

	// Start server
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("🌐 API server starting on port %s", port)
	log.Printf("📖 Documentation: http://localhost:%s/docs", port)
	log.Printf("💚 Health check: http://localhost:%s/health", port)

	if err := api.router.Run(":" + port); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}

// ============================================================================
// Database
// ============================================================================

func connectDB() (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://indexer:password@localhost:5432/indexer?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// ============================================================================
// Routes
// ============================================================================

func (api *API) setupRoutes() {
	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"} // For development; restrict in production
	config.AllowMethods = []string{"GET", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	api.router.Use(cors.New(config))

	// Health check
	api.router.GET("/health", api.handleHealth)

	// API documentation
	api.router.GET("/docs", api.handleDocs)

	// V1 API routes
	v1 := api.router.Group("/api/v1")
	{
		// Chains
		v1.GET("/chains", api.handleGetChains)
		v1.GET("/chains/:chain_id", api.handleGetChain)
		v1.GET("/chains/:chain_id/stats", api.handleGetChainStats)

		// Blocks
		v1.GET("/chains/:chain_id/blocks", api.handleGetBlocks)
		v1.GET("/chains/:chain_id/blocks/:block_number", api.handleGetBlock)

		// Transactions
		v1.GET("/chains/:chain_id/transactions", api.handleGetTransactions)
		v1.GET("/chains/:chain_id/transactions/:tx_hash", api.handleGetTransaction)
		v1.GET("/chains/:chain_id/blocks/:block_number/transactions", api.handleGetBlockTransactions)

		// Address
		v1.GET("/chains/:chain_id/addresses/:address/transactions", api.handleGetAddressTransactions)
	}
}

// ============================================================================
// Health & Docs
// ============================================================================

func (api *API) handleHealth(c *gin.Context) {
	// Check database
	dbStatus := "ok"
	if err := api.db.Ping(); err != nil {
		dbStatus = "error: " + err.Error()
	}

	// Check chains
	chains := make(map[string]string)
	rows, err := api.db.Query(`
		SELECT c.chain_id, c.chain_name, cp.last_processed_block
		FROM chains c
		LEFT JOIN checkpoints cp ON c.chain_id = cp.chain_id AND cp.service_name = 'ingester'
		WHERE c.enabled = true
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var chainID int64
			var name string
			var lastBlock *int64
			if err := rows.Scan(&chainID, &name, &lastBlock); err == nil {
				if lastBlock != nil {
					chains[name] = fmt.Sprintf("block %d", *lastBlock)
				} else {
					chains[name] = "not started"
				}
			}
		}
	}

	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Database:  dbStatus,
		Chains:    chains,
	})
}

func (api *API) handleDocs(c *gin.Context) {
	docs := `<!DOCTYPE html>
<html>
<head>
	<title>Blockchain Indexer API</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; max-width: 1200px; margin: 40px auto; padding: 0 20px; }
		h1 { color: #2c3e50; }
		h2 { color: #34495e; border-bottom: 2px solid #3498db; padding-bottom: 10px; margin-top: 40px; }
		.endpoint { background: #f8f9fa; padding: 15px; margin: 10px 0; border-left: 4px solid #3498db; }
		.method { display: inline-block; padding: 4px 8px; border-radius: 4px; font-weight: bold; margin-right: 10px; }
		.get { background: #61affe; color: white; }
		code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
		pre { background: #282c34; color: #abb2bf; padding: 15px; border-radius: 5px; overflow-x: auto; }
	</style>
</head>
<body>
	<h1>🚀 Blockchain Indexer API</h1>
	<p>Multi-chain blockchain data API supporting Ethereum, Polygon, Arbitrum, Optimism, and Base.</p>

	<h2>Base URL</h2>
	<code>http://localhost:8000/api/v1</code>

	<h2>Endpoints</h2>

	<div class="endpoint">
		<span class="method get">GET</span>
		<strong>/chains</strong>
		<p>List all supported chains with statistics</p>
		<pre>curl http://localhost:8000/api/v1/chains</pre>
	</div>

	<div class="endpoint">
		<span class="method get">GET</span>
		<strong>/chains/:chain_id</strong>
		<p>Get details for a specific chain (1=Ethereum, 137=Polygon, etc.)</p>
		<pre>curl http://localhost:8000/api/v1/chains/1</pre>
	</div>

	<div class="endpoint">
		<span class="method get">GET</span>
		<strong>/chains/:chain_id/stats</strong>
		<p>Get statistics for a chain (total blocks, transactions, averages)</p>
		<pre>curl http://localhost:8000/api/v1/chains/1/stats</pre>
	</div>

	<div class="endpoint">
		<span class="method get">GET</span>
		<strong>/chains/:chain_id/blocks</strong>
		<p>Get recent blocks. Query params: <code>limit</code> (default 20, max 100)</p>
		<pre>curl http://localhost:8000/api/v1/chains/1/blocks?limit=10</pre>
	</div>

	<div class="endpoint">
		<span class="method get">GET</span>
		<strong>/chains/:chain_id/blocks/:block_number</strong>
		<p>Get a specific block by number</p>
		<pre>curl http://localhost:8000/api/v1/chains/1/blocks/18500000</pre>
	</div>

	<div class="endpoint">
		<span class="method get">GET</span>
		<strong>/chains/:chain_id/blocks/:block_number/transactions</strong>
		<p>Get all transactions in a block</p>
		<pre>curl http://localhost:8000/api/v1/chains/1/blocks/18500000/transactions</pre>
	</div>

	<div class="endpoint">
		<span class="method get">GET</span>
		<strong>/chains/:chain_id/transactions/:tx_hash</strong>
		<p>Get a specific transaction by hash</p>
		<pre>curl http://localhost:8000/api/v1/chains/1/transactions/0x123...</pre>
	</div>

	<div class="endpoint">
		<span class="method get">GET</span>
		<strong>/chains/:chain_id/addresses/:address/transactions</strong>
		<p>Get transactions for an address. Query params: <code>limit</code> (default 20, max 100)</p>
		<pre>curl http://localhost:8000/api/v1/chains/1/addresses/0xabc.../transactions?limit=50</pre>
	</div>

	<h2>Chain IDs</h2>
	<ul>
		<li><strong>1</strong> - Ethereum Mainnet</li>
		<li><strong>137</strong> - Polygon</li>
		<li><strong>42161</strong> - Arbitrum One</li>
		<li><strong>10</strong> - Optimism</li>
		<li><strong>8453</strong> - Base</li>
	</ul>

	<h2>Response Format</h2>
	<p>All responses are JSON. Errors return appropriate HTTP status codes with error messages.</p>
	<pre>{
  "error": "Not found",
  "message": "Block 123 not found on chain 1"
}</pre>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docs))
}

// ============================================================================
// Chain Handlers
// ============================================================================

func (api *API) handleGetChains(c *gin.Context) {
	query := `
		SELECT 
			c.chain_id, c.chain_name, c.enabled,
			cp.last_processed_block,
			COUNT(DISTINCT b.block_number) as block_count,
			COUNT(t.tx_hash) as tx_count,
			c.created_at, c.updated_at
		FROM chains c
		LEFT JOIN blocks b ON c.chain_id = b.chain_id
		LEFT JOIN transactions t ON c.chain_id = t.chain_id
		LEFT JOIN checkpoints cp ON c.chain_id = cp.chain_id AND cp.service_name = 'ingester'
		WHERE c.enabled = true
		GROUP BY c.chain_id, c.chain_name, c.enabled, cp.last_processed_block, c.created_at, c.updated_at
		ORDER BY c.chain_id
	`

	rows, err := api.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed", "message": err.Error()})
		return
	}
	defer rows.Close()

	chains := []Chain{}
	for rows.Next() {
		var chain Chain
		err := rows.Scan(
			&chain.ChainID, &chain.Name, &chain.IsActive,
			&chain.LastBlock, &chain.BlockCount, &chain.TxCount,
			&chain.CreatedAt, &chain.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning chain: %v", err)
			continue
		}
		chains = append(chains, chain)
	}

	c.JSON(http.StatusOK, chains)
}

func (api *API) handleGetChain(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	var chain Chain
	err = api.db.QueryRow(`
		SELECT chain_id, chain_name, enabled, created_at, updated_at
		FROM chains
		WHERE chain_id = $1
	`, chainID).Scan(&chain.ChainID, &chain.Name, &chain.IsActive, &chain.CreatedAt, &chain.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chain)
}

func (api *API) handleGetChainStats(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	var stats ChainStats
	err = api.db.QueryRow(`
		SELECT 
			c.chain_id,
			c.chain_name,
			COUNT(DISTINCT b.block_number) as total_blocks,
			COUNT(t.tx_hash) as total_txs,
			cp.last_processed_block,
			AVG(EXTRACT(EPOCH FROM (b2.timestamp - b.timestamp))) as avg_block_time,
			AVG(b.transaction_count) as avg_txs_per_block
		FROM chains c
		LEFT JOIN blocks b ON c.chain_id = b.chain_id
		LEFT JOIN blocks b2 ON c.chain_id = b2.chain_id AND b2.block_number = b.block_number + 1
		LEFT JOIN transactions t ON c.chain_id = t.chain_id
		LEFT JOIN checkpoints cp ON c.chain_id = cp.chain_id AND cp.service_name = 'ingester'
		WHERE c.chain_id = $1
		GROUP BY c.chain_id, c.chain_name, cp.last_processed_block
	`, chainID).Scan(
		&stats.ChainID, &stats.Name, &stats.TotalBlocks, &stats.TotalTxs,
		&stats.LastProcessed, &stats.AvgBlockTime, &stats.AvgTxsPerBlock,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ============================================================================
// Block Handlers
// ============================================================================

func (api *API) handleGetBlocks(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	rows, err := api.db.Query(`
		SELECT block_number, block_hash, parent_hash, timestamp, gas_used, gas_limit,
			base_fee_per_gas, transaction_count, created_at
		FROM blocks
		WHERE chain_id = $1
		ORDER BY block_number DESC
		LIMIT $2
	`, chainID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": err.Error()})
		return
	}
	defer rows.Close()

	blocks := []Block{}
	for rows.Next() {
		var block Block
		block.ChainID = chainID
		err := rows.Scan(
			&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp,
			&block.GasUsed, &block.GasLimit, &block.BaseFeePerGas, &block.TransactionCount,
			&block.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning block: %v", err)
			continue
		}
		blocks = append(blocks, block)
	}

	c.JSON(http.StatusOK, blocks)
}

func (api *API) handleGetBlock(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	blockNumber, err := strconv.ParseInt(c.Param("block_number"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block number"})
		return
	}

	var block Block
	block.ChainID = chainID
	err = api.db.QueryRow(`
		SELECT block_number, block_hash, parent_hash, timestamp, gas_used, gas_limit,
			base_fee_per_gas, transaction_count, created_at
		FROM blocks
		WHERE chain_id = $1 AND block_number = $2
	`, chainID, blockNumber).Scan(
		&block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp,
		&block.GasUsed, &block.GasLimit, &block.BaseFeePerGas, &block.TransactionCount,
		&block.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, block)
}

// ============================================================================
// Transaction Handlers
// ============================================================================

func (api *API) handleGetTransactions(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	rows, err := api.db.Query(`
		SELECT tx_hash, block_number, tx_index, from_address, to_address, value,
			gas_limit, gas_price, nonce, status, gas_used, created_at
		FROM transactions
		WHERE chain_id = $1
		ORDER BY block_number DESC, tx_index
		LIMIT $2
	`, chainID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": err.Error()})
		return
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var tx Transaction
		tx.ChainID = chainID
		err := rows.Scan(
			&tx.TxHash, &tx.BlockNumber, &tx.TxIndex, &tx.FromAddress, &tx.ToAddress,
			&tx.Value, &tx.GasLimit, &tx.GasPrice, &tx.Nonce, &tx.Status, &tx.GasUsed,
			&tx.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning transaction: %v", err)
			continue
		}
		transactions = append(transactions, tx)
	}

	c.JSON(http.StatusOK, transactions)
}

func (api *API) handleGetTransaction(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	txHash := c.Param("tx_hash")

	var tx Transaction
	tx.ChainID = chainID
	err = api.db.QueryRow(`
		SELECT tx_hash, block_number, tx_index, from_address, to_address, value,
			gas_limit, gas_price, input_data, nonce, status, gas_used, created_at
		FROM transactions
		WHERE chain_id = $1 AND tx_hash = $2
	`, chainID, txHash).Scan(
		&tx.TxHash, &tx.BlockNumber, &tx.TxIndex, &tx.FromAddress, &tx.ToAddress,
		&tx.Value, &tx.GasLimit, &tx.GasPrice, &tx.InputData, &tx.Nonce,
		&tx.Status, &tx.GasUsed, &tx.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tx)
}

func (api *API) handleGetBlockTransactions(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	blockNumber, err := strconv.ParseInt(c.Param("block_number"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block number"})
		return
	}

	rows, err := api.db.Query(`
		SELECT tx_hash, block_number, tx_index, from_address, to_address, value,
			gas_limit, gas_price, nonce, status, gas_used, created_at
		FROM transactions
		WHERE chain_id = $1 AND block_number = $2
		ORDER BY tx_index
	`, chainID, blockNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": err.Error()})
		return
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var tx Transaction
		tx.ChainID = chainID
		err := rows.Scan(
			&tx.TxHash, &tx.BlockNumber, &tx.TxIndex, &tx.FromAddress, &tx.ToAddress,
			&tx.Value, &tx.GasLimit, &tx.GasPrice, &tx.Nonce, &tx.Status, &tx.GasUsed,
			&tx.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning transaction: %v", err)
			continue
		}
		transactions = append(transactions, tx)
	}

	c.JSON(http.StatusOK, transactions)
}

func (api *API) handleGetAddressTransactions(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	address := c.Param("address")

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	rows, err := api.db.Query(`
		SELECT tx_hash, block_number, tx_index, from_address, to_address, value,
			gas_limit, gas_price, nonce, status, gas_used, created_at
		FROM transactions
		WHERE chain_id = $1 AND (from_address = $2 OR to_address = $2)
		ORDER BY block_number DESC, tx_index
		LIMIT $3
	`, chainID, address, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": err.Error()})
		return
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var tx Transaction
		tx.ChainID = chainID
		err := rows.Scan(
			&tx.TxHash, &tx.BlockNumber, &tx.TxIndex, &tx.FromAddress, &tx.ToAddress,
			&tx.Value, &tx.GasLimit, &tx.GasPrice, &tx.Nonce, &tx.Status, &tx.GasUsed,
			&tx.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning transaction: %v", err)
			continue
		}
		transactions = append(transactions, tx)
	}

	c.JSON(http.StatusOK, transactions)
}
