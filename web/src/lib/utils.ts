import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatHash(hash: string, start = 6, end = 4): string {
  if (!hash) return ''
  if (hash.length <= start + end) return hash
  return `${hash.slice(0, start)}...${hash.slice(-end)}`
}

export function formatNumber(num: number | string): string {
  return Number(num).toLocaleString()
}

export function formatTimestamp(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleString()
}

export function formatWei(wei: string): string {
  const eth = Number(wei) / 1e18
  return `${eth.toFixed(6)} ETH`
}
