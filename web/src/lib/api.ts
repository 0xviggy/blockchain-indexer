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
}

export const api = new ApiClient(API_BASE_URL)
