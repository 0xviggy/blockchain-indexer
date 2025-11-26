export interface Chain {
  chain_id: number
  name: string
  is_active: boolean
  last_block: number | null
  block_count: number | null
  tx_count: number | null
  created_at: string
  updated_at: string
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
  tx_hash: string
  block_number: number
  tx_index: number
  from_address: string
  to_address: string | null
  value: string
  gas_limit: number
  gas_price: string
  input_data?: string
  nonce: number
  status: number
  gas_used: number
  created_at: string
}

export interface ChainStats {
  total_blocks: number
  total_transactions: number
  avg_block_time: number
  avg_tx_per_block: number
}

export interface Event {
  chain_id: number
  transaction_hash: string
  log_index: number
  contract_address: string
  event_signature: string
  protocol: string
  topic1: string | null
  topic2: string | null
  topic3: string | null
  data: string
  block_number: number
}

export interface HealthResponse {
  status: string
  database: string
  chains: Record<string, {
    last_block: number
    last_update: string
  }>
}
