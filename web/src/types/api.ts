export interface Chain {
  id: number
  name: string
  rpc_url: string
  last_block: number
  block_count: number
  tx_count: number
}

export interface Block {
  chain_id: number
  number: number
  hash: string
  parent_hash: string
  timestamp: number
  miner: string
  gas_used: string
  gas_limit: string
  base_fee_per_gas: string | null
  transaction_count: number
}

export interface Transaction {
  chain_id: number
  hash: string
  block_number: number
  tx_index: number
  from_address: string
  to_address: string | null
  value: string
  gas: string
  gas_price: string | null
  max_fee_per_gas: string | null
  max_priority_fee_per_gas: string | null
  input_data: string
  nonce: number
}

export interface ChainStats {
  total_blocks: number
  total_transactions: number
  avg_block_time: number
  avg_tx_per_block: number
}

export interface HealthResponse {
  status: string
  database: string
  chains: Record<string, {
    last_block: number
    last_update: string
  }>
}
