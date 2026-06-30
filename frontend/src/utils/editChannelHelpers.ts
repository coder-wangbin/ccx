import type { Channel } from '../services/api'

export function isValidUrl(url: string): boolean {
  try {
    new URL(url)
    return true
  } catch {
    return false
  }
}

export function normalizeModelCapabilities(record: Channel['modelCapabilities'] = {}): Channel['modelCapabilities'] {
  return Object.fromEntries(Object.entries(record).sort(([a], [b]) => a.localeCompare(b)))
}
