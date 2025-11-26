import { useState, useEffect } from 'react'
import { api, type SkippedBlock, type Checkpoint } from './lib/api'
import type { Chain, Transaction } from './types/api'
import './App.css'

type TabType = 'transactions' | 'events' | 'health' | 'logs' | 'skipped' | 'control'

interface LogEntry {
  timestamp: Date
  level: 'info' | 'warning' | 'error' | 'success'
  message: string
}

function App() {
  const [chains, setChains] = useState<Chain[]>([])
  const [selectedChain, setSelectedChain] = useState<number | null>(null)
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [events, setEvents] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [activeTab, setActiveTab] = useState<TabType>('transactions')
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [apiHealth, setApiHealth] = useState<any>(null)
  const [expandedBlocks, setExpandedBlocks] = useState<Set<number>>(new Set())
  const [txLimit, setTxLimit] = useState(500)
  const [minBlock, setMinBlock] = useState<number | null>(null)
  const [maxBlock, setMaxBlock] = useState<number | null>(null)
  const [blockTimestamps, setBlockTimestamps] = useState<Record<number, number>>({})
  const [lastIngestionTime, setLastIngestionTime] = useState<Date | null>(null)
  const [ingesterRunning, setIngesterRunning] = useState<boolean>(false)
  const [ingesterMessage, setIngesterMessage] = useState<string>('')
  const [skippedBlocks, setSkippedBlocks] = useState<SkippedBlock[]>([])
  const [checkpoint, setCheckpoint] = useState<Checkpoint | null>(null)
  const [resetBlockNumber, setResetBlockNumber] = useState<string>('')

  // Load chains on mount
  useEffect(() => {
    loadChains()
  }, [])

  // Auto-refresh data every 5 seconds
  useEffect(() => {
    if (!autoRefresh || !selectedChain) return
    
    const interval = setInterval(() => {
      loadTransactions(selectedChain, txLimit)
      loadEvents(selectedChain)
      loadHealth()
      loadIngesterStatus()
      loadSkippedBlocks(selectedChain)
    }, 5000)
    
    return () => clearInterval(interval)
  }, [autoRefresh, selectedChain, txLimit])

  // Add log helper
  const addLog = (level: LogEntry['level'], message: string) => {
    setLogs(prev => [{
      timestamp: new Date(),
      level,
      message
    }, ...prev].slice(0, 100)) // Keep last 100 logs
  }

  const loadChains = async () => {
    try {
      const data = await api.getChains()
      setChains(data)
      if (data.length > 0 && !selectedChain) {
        const firstChain = data[0].chain_id
        setSelectedChain(firstChain)
        loadTransactions(firstChain, txLimit)
        loadSkippedBlocks(firstChain)
        loadCheckpoint(firstChain)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load chains')
      setLoading(false)
    }
  }

  const loadTransactions = async (chainId: number, limit: number) => {
    try {
      const data = await api.getTransactions(chainId, limit)
      setTransactions(data)
      setLoading(false)
      setError(null)
      
      // Calculate min/max blocks and extract timestamps
      if (data.length > 0) {
        const blocks = data.map(tx => tx.block_number)
        setMinBlock(Math.min(...blocks))
        setMaxBlock(Math.max(...blocks))
        
        // Extract unique block timestamps from created_at
        const timestamps: Record<number, number> = {}
        data.forEach(tx => {
          if (!timestamps[tx.block_number]) {
            timestamps[tx.block_number] = new Date(tx.created_at).getTime()
          }
        })
        setBlockTimestamps(timestamps)
        
        // Update last ingestion time (most recent created_at)
        const mostRecent = data.reduce((latest, tx) => {
          const txTime = new Date(tx.created_at)
          return txTime > latest ? txTime : latest
        }, new Date(0))
        if (mostRecent.getTime() > 0) {
          setLastIngestionTime(mostRecent)
        }
      }
      
      // Check data quality
      const failedCount = data.filter(tx => tx.status === 0).length
      if (failedCount > 0) {
        addLog('success', `Found ${failedCount} failed transactions (Task 1: ✅ Working)`)
      }
      const zeroGasCount = data.filter(tx => tx.gas_used === 0).length
      if (zeroGasCount > data.length * 0.5) {
        addLog('warning', `${zeroGasCount}/${data.length} transactions have zero gas_used (Task 1: ⚠️ Check receipts)`)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load transactions')
      addLog('error', `Failed to load transactions: ${err}`)
      setLoading(false)
    }
  }

  const loadEvents = async (chainId: number) => {
    try {
      const data = await api.getEvents(chainId, 50)
      setEvents(data)
      if (data.length > 0) {
        addLog('success', `Loaded ${data.length} events (Task 2: ✅ Working)`)
      } else {
        addLog('info', 'No events found (Task 2: ⏳ Not implemented yet)')
      }
    } catch (err) {
      addLog('warning', 'Events endpoint not available (Task 2: ⏳ Not implemented yet)')
    }
  }

  const loadHealth = async () => {
    try {
      const data = await api.getHealth()
      setApiHealth(data)
    } catch (err) {
      addLog('error', `Health check failed: ${err}`)
    }
  }
  
  const loadIngesterStatus = async () => {
    try {
      const status = await api.getIngesterStatus()
      setIngesterRunning(status.running)
      setIngesterMessage(status.message)
    } catch {
      setIngesterRunning(false)
      setIngesterMessage('Unable to check ingester status')
    }
  }

  const loadSkippedBlocks = async (chainId: number) => {
    try {
      const data = await api.getSkippedBlocks(chainId)
      setSkippedBlocks(data)
    } catch (err) {
      console.error('Failed to load skipped blocks:', err)
      setSkippedBlocks([])
    }
  }

  const loadCheckpoint = async (chainId: number) => {
    try {
      const data = await api.getCheckpoint(chainId)
      setCheckpoint(data)
    } catch (err) {
      console.error('Failed to load checkpoint:', err)
      setCheckpoint(null)
    }
  }

  const handleResetCheckpoint = async () => {
    if (!selectedChain || !resetBlockNumber) return
    
    const blockNum = parseInt(resetBlockNumber)
    if (isNaN(blockNum) || blockNum < 0) {
      addLog('error', 'Invalid block number')
      return
    }

    try {
      await api.updateCheckpoint(selectedChain, blockNum)
      addLog('success', `Checkpoint reset to block ${blockNum}`)
      loadCheckpoint(selectedChain)
      setResetBlockNumber('')
    } catch (err) {
      addLog('error', `Failed to reset checkpoint: ${err}`)
    }
  }

  const handleClearSkipped = async () => {
    if (!selectedChain) return
    
    if (!confirm('Are you sure you want to clear all skipped blocks? This cannot be undone.')) {
      return
    }

    try {
      const result = await api.clearSkippedBlocks(selectedChain)
      addLog('success', `Cleared ${result.rows_deleted} skipped blocks`)
      loadSkippedBlocks(selectedChain)
    } catch (err) {
      addLog('error', `Failed to clear skipped blocks: ${err}`)
    }
  }

  const handleChainChange = (chainId: number) => {
    setSelectedChain(chainId)
    setLoading(true)
    loadTransactions(chainId, txLimit)
    loadEvents(chainId)
    loadIngesterStatus()
    loadSkippedBlocks(chainId)
    loadCheckpoint(chainId)
    addLog('info', `Switched to chain ${chainId}`)
  }

  // Helper to format relative time
  const formatRelativeTime = (timestamp: number) => {
    const now = Date.now()
    const diff = now - timestamp
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)
    
    if (seconds < 60) return `${seconds}s ago`
    if (minutes < 60) return `${minutes}m ago`
    if (hours < 24) return `${hours}h ago`
    return `${days}d ago`
  }
  
  // Helper to format absolute time
  const formatAbsoluteTime = (timestamp: number) => {
    return new Date(timestamp).toLocaleString()
  }

  // Calculate stats
  const successTxs = transactions.filter(tx => tx.status === 1).length
  const failedTxs = transactions.filter(tx => tx.status === 0).length
  const totalGasUsed = transactions.reduce((sum, tx) => sum + BigInt(tx.gas_used || '0'), 0n)
  const avgGasUsed = transactions.length > 0 ? Number(totalGasUsed / BigInt(transactions.length)) : 0

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100">
      <div className="container mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold mb-2">🔗 Blockchain Indexer</h1>
          <p className="text-gray-400">Data Verification Dashboard</p>
        </div>

        {/* Chain Selector */}
        <div className="mb-6 flex items-center gap-4">
          <label className="text-sm font-medium">Chain:</label>
          <select 
            value={selectedChain || ''} 
            onChange={(e) => handleChainChange(Number(e.target.value))}
            className="bg-gray-800 border border-gray-700 rounded px-3 py-2"
          >
            {chains.map(chain => (
              <option key={chain.chain_id} value={chain.chain_id}>
                {chain.name} (Chain ID: {chain.chain_id})
              </option>
            ))}
          </select>
          
          <label className="flex items-center gap-2 ml-auto">
            <input 
              type="checkbox" 
              checked={autoRefresh} 
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="w-4 h-4"
            />
            <span className="text-sm">Auto-refresh (5s)</span>
          </label>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
          <div className="bg-gray-800 rounded-lg p-4 border border-gray-700">
            <div className="text-sm text-gray-400 mb-1">Total Transactions</div>
            <div className="text-3xl font-bold">{transactions.length}</div>
          </div>
          
          <div className="bg-gray-800 rounded-lg p-4 border border-green-800">
            <div className="text-sm text-gray-400 mb-1">✅ Successful</div>
            <div className="text-3xl font-bold text-green-400">{successTxs}</div>
            <div className="text-xs text-gray-500 mt-1">
              {transactions.length > 0 ? ((successTxs / transactions.length) * 100).toFixed(1) : 0}%
            </div>
          </div>
          
          <div className="bg-gray-800 rounded-lg p-4 border border-red-800">
            <div className="text-sm text-gray-400 mb-1">❌ Failed</div>
            <div className="text-3xl font-bold text-red-400">{failedTxs}</div>
            <div className="text-xs text-gray-500 mt-1">
              {transactions.length > 0 ? ((failedTxs / transactions.length) * 100).toFixed(1) : 0}%
            </div>
          </div>
          
          <div className="bg-gray-800 rounded-lg p-4 border border-blue-800">
            <div className="text-sm text-gray-400 mb-1">⛽ Avg Gas Used</div>
            <div className="text-3xl font-bold text-blue-400">
              {(avgGasUsed / 1000).toFixed(0)}k
            </div>
            <div className="text-xs text-gray-500 mt-1">
              {avgGasUsed > 0 ? '✅ Fetched' : '⚠️ Not fetched'}
            </div>
          </div>
        </div>

        {/* Error Display */}
        {error && (
          <div className="bg-red-900/20 border border-red-800 rounded-lg p-4 mb-6">
            <div className="font-medium text-red-400">⚠️ Error</div>
            <div className="text-sm text-gray-300 mt-1">{error}</div>
          </div>
        )}

        {/* Tab Navigation */}
        <div className="flex gap-2 mb-6 border-b border-gray-700">
          <button
            onClick={() => setActiveTab('transactions')}
            className={`px-4 py-2 font-medium transition-colors ${
              activeTab === 'transactions'
                ? 'text-blue-400 border-b-2 border-blue-400'
                : 'text-gray-400 hover:text-gray-300'
            }`}
          >
            💳 Transactions (Task 1)
          </button>
          <button
            onClick={() => setActiveTab('events')}
            className={`px-4 py-2 font-medium transition-colors ${
              activeTab === 'events'
                ? 'text-blue-400 border-b-2 border-blue-400'
                : 'text-gray-400 hover:text-gray-300'
            }`}
          >
            📋 Events (Task 2)
          </button>
          <button
            onClick={() => setActiveTab('health')}
            className={`px-4 py-2 font-medium transition-colors ${
              activeTab === 'health'
                ? 'text-blue-400 border-b-2 border-blue-400'
                : 'text-gray-400 hover:text-gray-300'
            }`}
          >
            ❤️ Health
          </button>
          <button
            onClick={() => setActiveTab('logs')}
            className={`px-4 py-2 font-medium transition-colors relative ${
              activeTab === 'logs'
                ? 'text-blue-400 border-b-2 border-blue-400'
                : 'text-gray-400 hover:text-gray-300'
            }`}
          >
            📝 Logs
            {logs.filter(l => l.level === 'error').length > 0 && (
              <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
                {logs.filter(l => l.level === 'error').length}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab('skipped')}
            className={`px-4 py-2 font-medium transition-colors relative ${
              activeTab === 'skipped'
                ? 'text-blue-400 border-b-2 border-blue-400'
                : 'text-gray-400 hover:text-gray-300'
            }`}
          >
            ⚠️ Skipped Blocks
            {skippedBlocks.length > 0 && (
              <span className="absolute -top-1 -right-1 bg-yellow-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
                {skippedBlocks.length}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab('control')}
            className={`px-4 py-2 font-medium transition-colors ${
              activeTab === 'control'
                ? 'text-blue-400 border-b-2 border-blue-400'
                : 'text-gray-400 hover:text-gray-300'
            }`}
          >
            ⚙️ Ingester Control
          </button>
        </div>

        {/* Content Panels */}
        {activeTab === 'transactions' && (
        <div className="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-700">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-semibold">Recent Transactions</h2>
                <p className="text-sm text-gray-400 mt-1">
                  Grouped by block • Showing {transactions.length} transactions
                  {minBlock && maxBlock && ` • Blocks ${minBlock} - ${maxBlock}`}
                </p>
              </div>
              <div className="flex items-center gap-4">
                {/* Limit Selector */}
                <div className="flex items-center gap-2">
                  <label className="text-sm text-gray-400">Show:</label>
                  <select
                    value={txLimit}
                    onChange={(e) => {
                      const newLimit = Number(e.target.value)
                      setTxLimit(newLimit)
                      if (selectedChain) loadTransactions(selectedChain, newLimit)
                    }}
                    className="bg-gray-700 text-gray-200 border border-gray-600 rounded px-3 py-1 text-sm"
                  >
                    <option value={100}>100 txs</option>
                    <option value={500}>500 txs</option>
                    <option value={1000}>1000 txs</option>
                  </select>
                </div>
                {/* Refresh Button */}
                <button
                  onClick={() => selectedChain && loadTransactions(selectedChain, txLimit)}
                  className="px-3 py-1 bg-blue-600 hover:bg-blue-700 text-white rounded text-sm transition-colors"
                >
                  🔄 Refresh
                </button>
              </div>
            </div>
          </div>
          
          
          {/* Info Banner */}
          {!loading && transactions.length > 0 && (
            <div className="px-6 py-3 bg-blue-900/20 border-b border-blue-800/30">
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-2 text-sm">
                  <span className="text-blue-400">ℹ️</span>
                  <div>
                    <span className="text-blue-300">Currently showing the latest indexed transactions.</span>
                    <span className="text-gray-400 ml-1">
                      Auto-refresh is {autoRefresh ? 'ON' : 'OFF'}. Click blocks to expand and see individual transactions.
                    </span>
                  </div>
                </div>
                {lastIngestionTime && (
                  <div className="text-sm flex items-center gap-2">
                    <span className={ingesterRunning ? "text-green-400 animate-pulse" : "text-gray-500"}>●</span>
                    <span className="text-gray-400">
                      {ingesterRunning ? 'Ingesting' : 'Last ingested'}: <span className="text-gray-300">{formatRelativeTime(lastIngestionTime.getTime())}</span>
                    </span>
                    {ingesterMessage && (
                      <span className="text-xs text-gray-500" title={ingesterMessage}>ⓘ</span>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}
          
          {loading ? (
            <div className="p-8 text-center text-gray-400">Loading...</div>
          ) : transactions.length === 0 ? (
            <div className="p-8 text-center">
              <div className="text-yellow-400 text-5xl mb-4">⏳</div>
              <div className="text-lg font-medium text-gray-300 mb-2">No transactions indexed yet</div>
              <div className="text-sm text-gray-400">The ingester is starting up or catching up with the blockchain.</div>
            </div>
          ) : (
            <div className="divide-y divide-gray-700">
              {/* Group transactions by block */}
              {Object.entries(
                transactions.reduce((acc, tx) => {
                  if (!acc[tx.block_number]) acc[tx.block_number] = []
                  acc[tx.block_number].push(tx)
                  return acc
                }, {} as Record<number, Transaction[]>)
              )
              .sort(([a], [b]) => Number(b) - Number(a)) // Sort blocks descending
              .map(([blockNum, blockTxs]) => {
                const isExpanded = expandedBlocks.has(Number(blockNum))
                const successCount = blockTxs.filter(tx => tx.status === 1).length
                const failedCount = blockTxs.length - successCount
                
                return (
                  <div key={blockNum} className="border-b border-gray-700 last:border-b-0">
                    {/* Block Header - Clickable */}
                    <button
                      onClick={() => {
                        const newExpanded = new Set(expandedBlocks)
                        if (isExpanded) {
                          newExpanded.delete(Number(blockNum))
                        } else {
                          newExpanded.add(Number(blockNum))
                        }
                        setExpandedBlocks(newExpanded)
                      }}
                      className="w-full px-6 py-4 flex items-center justify-between hover:bg-gray-750 transition-colors"
                    >
                      <div className="flex items-center gap-4">
                        <span className="text-2xl">{isExpanded ? '▼' : '▶'}</span>
                        <div className="text-left">
                          <div className="flex items-center gap-3">
                            <span className="text-lg font-semibold text-blue-400">Block {blockNum}</span>
                            <span className="text-sm text-gray-400">{blockTxs.length} transactions</span>
                            {blockTimestamps[Number(blockNum)] && (
                              <span className="text-xs text-gray-500" title={formatAbsoluteTime(blockTimestamps[Number(blockNum)])}>
                                🕐 {formatRelativeTime(blockTimestamps[Number(blockNum)])}
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-3 mt-1">
                            {successCount > 0 && (
                              <span className="text-xs text-green-400">✓ {successCount} success</span>
                            )}
                            {failedCount > 0 && (
                              <span className="text-xs text-red-400">✗ {failedCount} failed</span>
                            )}
                          </div>
                        </div>
                      </div>
                      <a
                        href={`https://etherscan.io/block/${blockNum}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        onClick={(e) => e.stopPropagation()}
                        className="text-sm text-blue-400 hover:text-blue-300 underline"
                      >
                        View on Etherscan →
                      </a>
                    </button>
                    
                    {/* Transactions Table - Collapsible */}
                    {isExpanded && (
                      <div className="overflow-x-auto bg-gray-850">
                        <table className="w-full">
                          <thead className="bg-gray-900">
                            <tr>
                              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Status</th>
                              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Hash</th>
                              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Index</th>
                              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">From</th>
                              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">To</th>
                              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Gas Used</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-gray-700">
                            {blockTxs
                              .sort((a, b) => a.tx_index - b.tx_index)
                              .map((tx) => (
                                <tr key={tx.tx_hash} className="hover:bg-gray-800">
                                  <td className="px-4 py-3">
                                    {tx.status === 1 ? (
                                      <span className="inline-flex items-center px-2.5 py-0.5 rounded text-xs font-medium bg-green-900/30 text-green-400 border border-green-800">
                                        ✓
                                      </span>
                                    ) : (
                                      <span className="inline-flex items-center px-2.5 py-0.5 rounded text-xs font-medium bg-red-900/30 text-red-400 border border-red-800">
                                        ✗
                                      </span>
                                    )}
                                  </td>
                                  <td className="px-4 py-3 font-mono text-sm">
                                    <a 
                                      href={`https://etherscan.io/tx/${tx.tx_hash}`} 
                                      target="_blank" 
                                      rel="noopener noreferrer"
                                      className="text-blue-400 hover:text-blue-300"
                                    >
                                      {tx.tx_hash.slice(0, 10)}...{tx.tx_hash.slice(-8)}
                                    </a>
                                  </td>
                                  <td className="px-4 py-3 text-sm text-gray-400">{tx.tx_index}</td>
                                  <td className="px-4 py-3 font-mono text-sm text-gray-400">
                                    {tx.from_address.slice(0, 6)}...{tx.from_address.slice(-4)}
                                  </td>
                                  <td className="px-4 py-3 font-mono text-sm text-gray-400">
                                    {tx.to_address ? `${tx.to_address.slice(0, 6)}...${tx.to_address.slice(-4)}` : '(contract)'}
                                  </td>
                                  <td className="px-4 py-3 text-sm">
                                    {tx.gas_used === 0 ? (
                                      <span className="text-yellow-400">⚠️ 0</span>
                                    ) : (
                                      <span className="text-gray-300">
                                        {Number(tx.gas_used).toLocaleString()}
                                      </span>
                                    )}
                                  </td>
                                </tr>
                              ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>
        )}

        {/* Events Tab (Task 2 Verification) */}
        {activeTab === 'events' && (
        <div className="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-700">
            <h2 className="text-xl font-semibold">Smart Contract Events</h2>
            <p className="text-sm text-gray-400 mt-1">Verifies Task 2: Event Log Parsing</p>
          </div>
          
          {events.length === 0 ? (
            <div className="p-8 text-center">
              <div className="text-yellow-400 text-5xl mb-4">⏳</div>
              <div className="text-lg font-medium text-gray-300 mb-2">Events Not Implemented Yet</div>
              <div className="text-sm text-gray-400">
                Task 2 (Event Log Parsing) has not been completed.
                <br />
                Once implemented, you'll see protocol events here (Uniswap swaps, ERC20 transfers, etc.)
              </div>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-900">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Protocol</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Event</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Contract</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Block</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Tx Hash</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-700">
                  {events.map((evt, idx) => (
                    <tr key={idx} className="hover:bg-gray-750">
                      <td className="px-4 py-3">
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded text-xs font-medium bg-purple-900/30 text-purple-400 border border-purple-800">
                          {evt.protocol || 'Unknown'}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm">{evt.event_name || 'Event'}</td>
                      <td className="px-4 py-3 font-mono text-sm text-gray-400">
                        {evt.contract_address?.slice(0, 10)}...
                      </td>
                      <td className="px-4 py-3 text-sm">{evt.block_number}</td>
                      <td className="px-4 py-3 font-mono text-sm">
                        {evt.transaction_hash?.slice(0, 10)}...
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
        )}

        {/* Health Tab */}
        {activeTab === 'health' && (
        <div className="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-700">
            <h2 className="text-xl font-semibold">System Health</h2>
            <p className="text-sm text-gray-400 mt-1">Overall system status and chain sync</p>
          </div>
          
          <div className="p-6 space-y-4">
            {apiHealth ? (
              <>
                <div className="flex items-center justify-between p-4 bg-gray-900 rounded">
                  <span className="text-gray-400">API Status</span>
                  <span className="text-green-400 font-medium">✓ {apiHealth.status || 'OK'}</span>
                </div>
                <div className="flex items-center justify-between p-4 bg-gray-900 rounded">
                  <span className="text-gray-400">Database</span>
                  <span className="text-green-400 font-medium">✓ {apiHealth.database || 'Connected'}</span>
                </div>
                
                {apiHealth.chains && Object.entries(apiHealth.chains).length > 0 && (
                  <div className="mt-4">
                    <div className="text-sm font-medium text-gray-400 mb-2">Chain Status</div>
                    {Object.entries(apiHealth.chains).map(([name, info]: [string, any]) => (
                      <div key={name} className="flex items-center justify-between p-3 bg-gray-900 rounded mb-2">
                        <span className="text-gray-300">{name}</span>
                        <div className="text-right">
                          <div className="text-sm text-gray-400">Block: {info.last_block}</div>
                          <div className="text-xs text-gray-500">{info.last_update}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </>
            ) : (
              <div className="text-center py-8 text-gray-400">
                Loading health data...
              </div>
            )}
          </div>
        </div>
        )}

        {/* Logs Tab */}
        {activeTab === 'logs' && (
        <div className="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-700 flex items-center justify-between">
            <div>
              <h2 className="text-xl font-semibold">Development Logs</h2>
              <p className="text-sm text-gray-400 mt-1">Real-time data quality checks</p>
            </div>
            <button
              onClick={() => setLogs([])}
              className="px-3 py-1 text-sm bg-gray-700 hover:bg-gray-600 rounded"
            >
              Clear
            </button>
          </div>
          
          <div className="p-4 space-y-2 max-h-96 overflow-y-auto">
            {logs.length === 0 ? (
              <div className="text-center py-8 text-gray-400">
                No logs yet. Logs will appear as you interact with the UI.
              </div>
            ) : (
              logs.map((log, idx) => (
                <div
                  key={idx}
                  className={`p-3 rounded text-sm font-mono ${
                    log.level === 'error' ? 'bg-red-900/20 border border-red-800 text-red-400' :
                    log.level === 'warning' ? 'bg-yellow-900/20 border border-yellow-800 text-yellow-400' :
                    log.level === 'success' ? 'bg-green-900/20 border border-green-800 text-green-400' :
                    'bg-gray-900 border border-gray-700 text-gray-400'
                  }`}
                >
                  <span className="text-gray-500">{log.timestamp.toLocaleTimeString()}</span>
                  {' '}
                  <span className="font-bold">
                    {log.level === 'error' ? '❌' : 
                     log.level === 'warning' ? '⚠️' : 
                     log.level === 'success' ? '✅' : 'ℹ️'}
                  </span>
                  {' '}
                  {log.message}
                </div>
              ))
            )}
          </div>
        </div>
        )}

        {activeTab === 'skipped' && (
        <div className="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-700">
            <h2 className="text-xl font-semibold">Skipped Blocks</h2>
            <p className="text-sm text-gray-400 mt-1">
              Blocks that failed to process • Total: {skippedBlocks.length}
            </p>
          </div>
          
          <div className="overflow-x-auto">
            {skippedBlocks.length === 0 ? (
              <div className="text-center py-8 text-gray-400">
                <div className="text-4xl mb-2">✅</div>
                No skipped blocks! All blocks processed successfully.
              </div>
            ) : (
              <table className="w-full">
                <thead className="bg-gray-900 text-gray-400 text-sm">
                  <tr>
                    <th className="px-4 py-3 text-left">Block Number</th>
                    <th className="px-4 py-3 text-left">Reason</th>
                    <th className="px-4 py-3 text-left">Error Message</th>
                    <th className="px-4 py-3 text-left">Skipped At</th>
                    <th className="px-4 py-3 text-left">Retries</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-700">
                  {skippedBlocks.map((block) => (
                    <tr key={block.block_number} className="hover:bg-gray-750">
                      <td className="px-4 py-3">
                        <span className="font-mono text-yellow-400">
                          #{block.block_number}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="px-2 py-1 text-xs bg-orange-900/30 text-orange-400 rounded border border-orange-800">
                          {block.skip_reason}
                        </span>
                      </td>
                      <td className="px-4 py-3 max-w-md">
                        <span className="text-sm text-gray-400 truncate block" title={block.error_message}>
                          {block.error_message}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-400">
                        {new Date(block.skipped_at).toLocaleString()}
                      </td>
                      <td className="px-4 py-3 text-center">
                        {block.retry_count > 0 ? (
                          <span className="px-2 py-1 text-xs bg-red-900/30 text-red-400 rounded border border-red-800">
                            {block.retry_count}
                          </span>
                        ) : (
                          <span className="text-gray-500">-</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
        )}

        {activeTab === 'control' && (
        <div className="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-700">
            <h2 className="text-xl font-semibold">Ingester Control Panel</h2>
            <p className="text-sm text-gray-400 mt-1">
              Manage checkpoint and re-process block ranges
            </p>
          </div>
          
          <div className="p-6 space-y-6">
            {/* Current Status */}
            <div className="bg-gray-900 rounded-lg p-4 border border-gray-700">
              <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                📊 Current Status
              </h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-sm text-gray-400">Chain ID</div>
                  <div className="text-lg font-mono">{selectedChain}</div>
                </div>
                <div>
                  <div className="text-sm text-gray-400">Ingester Status</div>
                  <div className="flex items-center gap-2">
                    <span className={`inline-block w-2 h-2 rounded-full ${ingesterRunning ? 'bg-green-500 animate-pulse' : 'bg-gray-500'}`}></span>
                    <span>{ingesterRunning ? 'Running' : 'Stopped'}</span>
                  </div>
                </div>
                <div>
                  <div className="text-sm text-gray-400">Last Processed Block</div>
                  <div className="text-lg font-mono text-blue-400">
                    {checkpoint ? `#${checkpoint.last_processed_block.toLocaleString()}` : 'Loading...'}
                  </div>
                </div>
                <div>
                  <div className="text-sm text-gray-400">Last Updated</div>
                  <div className="text-sm">
                    {checkpoint ? new Date(checkpoint.updated_at).toLocaleString() : '-'}
                  </div>
                </div>
              </div>
            </div>

            {/* Reset Checkpoint */}
            <div className="bg-gray-900 rounded-lg p-4 border border-gray-700">
              <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                🔄 Reset Checkpoint
              </h3>
              <p className="text-sm text-gray-400 mb-4">
                Set the ingester to start processing from a specific block. Useful for re-indexing historical data or recovering from errors.
              </p>
              <div className="flex gap-3">
                <input
                  type="number"
                  value={resetBlockNumber}
                  onChange={(e) => setResetBlockNumber(e.target.value)}
                  placeholder="Enter block number (e.g., 23883990)"
                  className="flex-1 px-4 py-2 bg-gray-800 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                  min="0"
                />
                <button
                  onClick={handleResetCheckpoint}
                  disabled={!resetBlockNumber}
                  className="px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:cursor-not-allowed rounded font-medium transition-colors"
                >
                  Reset to Block
                </button>
              </div>
              <div className="mt-3 text-xs text-yellow-400 flex items-start gap-2">
                <span>⚠️</span>
                <span>The ingester must be restarted for checkpoint changes to take effect.</span>
              </div>
            </div>

            {/* Quick Actions */}
            <div className="bg-gray-900 rounded-lg p-4 border border-gray-700">
              <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                ⚡ Quick Actions
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <button
                  onClick={() => {
                    const blockNum = checkpoint ? checkpoint.last_processed_block - 1000 : ''
                    setResetBlockNumber(blockNum.toString())
                  }}
                  disabled={!checkpoint}
                  className="px-4 py-2 bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:cursor-not-allowed rounded text-left transition-colors"
                >
                  <div className="font-medium">Reprocess Last 1000 Blocks</div>
                  <div className="text-xs text-gray-400 mt-1">
                    Set checkpoint to {checkpoint ? (checkpoint.last_processed_block - 1000).toLocaleString() : '-'}
                  </div>
                </button>
                
                <button
                  onClick={() => setResetBlockNumber('23883990')}
                  className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded text-left transition-colors"
                >
                  <div className="font-medium">Test Blob Transaction Error</div>
                  <div className="text-xs text-gray-400 mt-1">
                    Process block 23,883,999 (known blob tx issue)
                  </div>
                </button>

                <button
                  onClick={() => setResetBlockNumber('19426587')}
                  className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded text-left transition-colors"
                >
                  <div className="font-medium">Start from Dencun Fork</div>
                  <div className="text-xs text-gray-400 mt-1">
                    First block with EIP-4844 blob transactions
                  </div>
                </button>

                <button
                  onClick={handleClearSkipped}
                  disabled={skippedBlocks.length === 0}
                  className="px-4 py-2 bg-red-900/30 hover:bg-red-900/50 disabled:bg-gray-800 disabled:cursor-not-allowed border border-red-800 rounded text-left transition-colors"
                >
                  <div className="font-medium text-red-400">Clear Skipped Blocks</div>
                  <div className="text-xs text-gray-400 mt-1">
                    Remove all {skippedBlocks.length} skipped blocks from database
                  </div>
                </button>
              </div>
            </div>

            {/* Usage Guide */}
            <div className="bg-blue-900/20 border border-blue-800 rounded-lg p-4">
              <h3 className="text-lg font-semibold mb-2 text-blue-400">💡 How to Use</h3>
              <ul className="text-sm text-gray-300 space-y-2">
                <li><strong>Historical Mode:</strong> Set checkpoint to a past block, restart ingester, it will process forward from that point</li>
                <li><strong>Skip Recovery:</strong> Clear skipped blocks, reset checkpoint before them, restart ingester to retry</li>
                <li><strong>Latest Mode:</strong> Ingester automatically continues from last checkpoint (default behavior)</li>
                <li><strong>Note:</strong> Always restart the ingester after changing the checkpoint for changes to take effect</li>
              </ul>
            </div>
          </div>
        </div>
        )}

        {/* Debug Info */}
        <div className="mt-6 text-xs text-gray-500">
          Last updated: {new Date().toLocaleTimeString()} | 
          API: {import.meta.env.VITE_API_URL || 'http://localhost:8000'}
        </div>
      </div>
    </div>
  )
}

export default App
