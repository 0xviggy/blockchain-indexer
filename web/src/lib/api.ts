import type { Chain, Block, Transaction, ChainStats, HealthResponse } from '../types/api'

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8000'

class ApiClient {
  private baseUrl: string

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
  }

  private async fetch<T>(endpoint: string): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`)
    if (!response.ok) {
      throw new Error(`API Error: ${response.statusText}`)
    }
    return response.json()
  }

  // Health
  async getHealth(): Promise<HealthResponse> {
    return this.fetch<HealthResponse>('/health')
  }
  
  // Ingester status
  async getIngesterStatus(): Promise<{
    running: boolean
    last_block_time: string | null
    last_block_number: number | null
    blocks_behind: number | null
    message: string
  }> {
    return this.fetch('/ingester/status')
  }

  // Chains
  async getChains(): Promise<Chain[]> {
    return this.fetch<Chain[]>('/api/v1/chains')
  }

  async getChain(chainId: number): Promise<Chain> {
    return this.fetch<Chain>(`/api/v1/chains/${chainId}`)
  }

  async getChainStats(chainId: number): Promise<ChainStats> {
    return this.fetch<ChainStats>(`/api/v1/chains/${chainId}/stats`)
  }

  // Blocks
  async getBlocks(chainId: number, limit = 20): Promise<Block[]> {
    return this.fetch<Block[]>(`/api/v1/chains/${chainId}/blocks?limit=${limit}`)
  }

  async getBlock(chainId: number, blockNumber: number): Promise<Block> {
    return this.fetch<Block>(`/api/v1/chains/${chainId}/blocks/${blockNumber}`)
  }

  // Transactions
  async getTransactions(chainId: number, limit = 20): Promise<Transaction[]> {
    return this.fetch<Transaction[]>(`/api/v1/chains/${chainId}/transactions?limit=${limit}`)
  }

  async getTransaction(chainId: number, hash: string): Promise<Transaction> {
    return this.fetch<Transaction>(`/api/v1/chains/${chainId}/transactions/${hash}`)
  }

  async getBlockTransactions(chainId: number, blockNumber: number): Promise<Transaction[]> {
    return this.fetch<Transaction[]>(`/api/v1/chains/${chainId}/blocks/${blockNumber}/transactions`)
  }

  async getAddressTransactions(address: string, limit = 20): Promise<Transaction[]> {
    return this.fetch<Transaction[]>(`/api/v1/addresses/${address}/transactions?limit=${limit}`)
  }

  // Skipped Blocks
  async getSkippedBlocks(chainId: number, limit = 100): Promise<SkippedBlock[]> {
    return this.fetch<SkippedBlock[]>(`/api/v1/chains/${chainId}/skipped-blocks?limit=${limit}`)
  }

  async clearSkippedBlocks(chainId: number): Promise<{ success: boolean; rows_deleted: number }> {
    const response = await fetch(`${this.baseUrl}/api/v1/chains/${chainId}/skipped-blocks`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      throw new Error(`API Error: ${response.statusText}`)
    }
    return response.json()
  }

  // Ingester Control
  async getCheckpoint(chainId: number): Promise<Checkpoint> {
    return this.fetch<Checkpoint>(`/api/v1/chains/${chainId}/checkpoint`)
  }

  async updateCheckpoint(chainId: number, blockNumber: number): Promise<{ success: boolean }> {
    const response = await fetch(`${this.baseUrl}/api/v1/chains/${chainId}/checkpoint`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ block_number: blockNumber }),
    })
    if (!response.ok) {
      throw new Error(`API Error: ${response.statusText}`)
    }
    return response.json()
  }

  // Events (for Task 2 verification)
  async getEvents(chainId: number, limit = 20): Promise<any[]> {
    try {
      return this.fetch<any[]>(`/api/v1/chains/${chainId}/events?limit=${limit}`)
    } catch (err) {
      // Events endpoint might not exist yet
      return []
    }
  }
}

export interface SkippedBlock {
  chain_id: number
  block_number: number
  skip_reason: string
  error_message: string
  skipped_at: string
  retry_count: number
  last_retry_at?: string
}

export interface Checkpoint {
  chain_id: number
  service_name: string
  last_processed_block: number
  updated_at: string
}

export const api = new ApiClient(API_BASE_URL)
