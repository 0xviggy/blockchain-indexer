package models

import "time"

// Chain represents a blockchain network
type Chain struct {
	ChainID          int       `json:"chain_id" db:"chain_id"`
	ChainName        string    `json:"chain_name" db:"chain_name"`
	RPCURL           string    `json:"rpc_url" db:"rpc_url"`
	WSURL            string    `json:"ws_url" db:"ws_url"`
	BlockTimeSeconds int       `json:"block_time_seconds" db:"block_time_seconds"`
	FinalityBlocks   int       `json:"finality_blocks" db:"finality_blocks"`
	Enabled          bool      `json:"enabled" db:"enabled"`
	LastIndexedBlock int64     `json:"last_indexed_block" db:"last_indexed_block"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Block represents a blockchain block
type Block struct {
	ChainID          int       `json:"chain_id" db:"chain_id"`
	BlockNumber      int64     `json:"block_number" db:"block_number"`
	BlockHash        string    `json:"block_hash" db:"block_hash"`
	ParentHash       string    `json:"parent_hash" db:"parent_hash"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	Miner            string    `json:"miner" db:"miner"`
	GasUsed          int64     `json:"gas_used" db:"gas_used"`
	GasLimit         int64     `json:"gas_limit" db:"gas_limit"`
	TransactionCount int       `json:"transaction_count" db:"transaction_count"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ChainID     int       `json:"chain_id" db:"chain_id"`
	TxHash      string    `json:"tx_hash" db:"tx_hash"`
	BlockNumber int64     `json:"block_number" db:"block_number"`
	TxIndex     int       `json:"tx_index" db:"tx_index"`
	FromAddress string    `json:"from_address" db:"from_address"`
	ToAddress   *string   `json:"to_address" db:"to_address"`
	Value       string    `json:"value" db:"value"`
	GasPrice    int64     `json:"gas_price" db:"gas_price"`
	GasUsed     int64     `json:"gas_used" db:"gas_used"`
	Input       []byte    `json:"input" db:"input"`
	Status      bool      `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Event represents a blockchain event/log
type Event struct {
	ChainID         int       `json:"chain_id" db:"chain_id"`
	ID              int64     `json:"id" db:"id"`
	TxHash          string    `json:"tx_hash" db:"tx_hash"`
	BlockNumber     int64     `json:"block_number" db:"block_number"`
	LogIndex        int       `json:"log_index" db:"log_index"`
	ContractAddress string    `json:"contract_address" db:"contract_address"`
	EventSignature  string    `json:"event_signature" db:"event_signature"`
	Topic1          *string   `json:"topic1" db:"topic1"`
	Topic2          *string   `json:"topic2" db:"topic2"`
	Topic3          *string   `json:"topic3" db:"topic3"`
	Data            []byte    `json:"data" db:"data"`
	DecodedData     string    `json:"decoded_data" db:"decoded_data"` // JSONB as string
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Checkpoint tracks ingestion progress
type Checkpoint struct {
	ServiceName        string    `json:"service_name" db:"service_name"`
	LastProcessedBlock int64     `json:"last_processed_block" db:"last_processed_block"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// ParsedCalldata represents decoded function calls
type ParsedCalldata struct {
	ChainID           int       `json:"chain_id" db:"chain_id"`
	TxHash            string    `json:"tx_hash" db:"tx_hash"`
	FunctionSignature string    `json:"function_signature" db:"function_signature"`
	FunctionName      string    `json:"function_name" db:"function_name"`
	Protocol          string    `json:"protocol" db:"protocol"`
	DecodedParams     string    `json:"decoded_params" db:"decoded_params"` // JSONB as string
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// InternalTransaction represents contract-to-contract calls
type InternalTransaction struct {
	ChainID         int       `json:"chain_id" db:"chain_id"`
	TxHash          string    `json:"tx_hash" db:"tx_hash"`
	InternalTxIndex int       `json:"internal_tx_index" db:"internal_tx_index"`
	CallType        string    `json:"call_type" db:"call_type"`
	FromAddress     string    `json:"from_address" db:"from_address"`
	ToAddress       *string   `json:"to_address" db:"to_address"`
	Value           string    `json:"value" db:"value"`
	Gas             int64     `json:"gas" db:"gas"`
	GasUsed         int64     `json:"gas_used" db:"gas_used"`
	Input           []byte    `json:"input" db:"input"`
	Output          []byte    `json:"output" db:"output"`
	Success         bool      `json:"success" db:"success"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// RevertReason stores error messages from failed transactions
type RevertReason struct {
	ChainID        int       `json:"chain_id" db:"chain_id"`
	TxHash         string    `json:"tx_hash" db:"tx_hash"`
	RevertReason   *string   `json:"revert_reason" db:"revert_reason"`
	ErrorSignature *string   `json:"error_signature" db:"error_signature"`
	ErrorName      *string   `json:"error_name" db:"error_name"`
	ErrorParams    *string   `json:"error_params" db:"error_params"` // JSONB as string
	ExtractedAt    time.Time `json:"extracted_at" db:"extracted_at"`
}

// ProtocolSignature represents a known function signature
type ProtocolSignature struct {
	Signature    string    `json:"signature" db:"signature"`
	FunctionName string    `json:"function_name" db:"function_name"`
	Protocol     string    `json:"protocol" db:"protocol"`
	ABI          string    `json:"abi" db:"abi"`
	Description  *string   `json:"description" db:"description"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
