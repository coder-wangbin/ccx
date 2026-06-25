type TimestampedPoint = {
  timestamp: string
}

export type ChartDuration = '1h' | '6h' | '24h' | 'today' | '7d' | '30d' | '90d' | '180d' | '365d' | 'thisyear'

const DENSE_SAMPLING_INTERVALS = [
  { maxSpanMs: 6 * 60 * 60 * 1000, interval: '1m', intervalMs: 60 * 1000 },
  { maxSpanMs: 24 * 60 * 60 * 1000, interval: '5m', intervalMs: 5 * 60 * 1000 },
  { maxSpanMs: 7 * 24 * 60 * 60 * 1000, interval: '15m', intervalMs: 15 * 60 * 1000 },
  { maxSpanMs: 30 * 24 * 60 * 60 * 1000, interval: '1h', intervalMs: 60 * 60 * 1000 },
  { maxSpanMs: 90 * 24 * 60 * 60 * 1000, interval: '4h', intervalMs: 4 * 60 * 60 * 1000 }
] as const

const LONG_DURATIONS = new Set<ChartDuration>(['7d', '30d', '90d', '180d', '365d', 'thisyear'])

export function isLongChartDuration(duration: ChartDuration): boolean {
  return LONG_DURATIONS.has(duration)
}

export function effectiveChartIntervalMs(defaultIntervalSeconds?: number, adaptiveInterval?: string): number | undefined {
  return parseIntervalMs(adaptiveInterval) ?? (defaultIntervalSeconds && defaultIntervalSeconds > 0 ? defaultIntervalSeconds * 1000 : undefined)
}

export function selectDenseSamplingInterval<T extends TimestampedPoint>(
  duration: ChartDuration,
  points: T[],
  hasVisibleData: (_point: T) => boolean,
  currentIntervalSeconds?: number
): string | undefined {
  if (!isLongChartDuration(duration)) return undefined

  const firstTimestamp = getFirstVisibleTimestamp(points, hasVisibleData)
  if (firstTimestamp === undefined) return undefined

  const dataSpanMs = Date.now() - firstTimestamp
  const currentIntervalMs = currentIntervalSeconds && currentIntervalSeconds > 0
    ? currentIntervalSeconds * 1000
    : undefined

  const match = DENSE_SAMPLING_INTERVALS.find(item => dataSpanMs <= item.maxSpanMs)
  if (!match) return undefined
  if (currentIntervalMs !== undefined && match.intervalMs >= currentIntervalMs) return undefined

  return match.interval
}

function getFirstVisibleTimestamp<T extends TimestampedPoint>(
  points: T[],
  hasVisibleData: (_point: T) => boolean
): number | undefined {
  let earliest = Infinity
  points.forEach(point => {
    if (!hasVisibleData(point)) return
    const timestamp = new Date(point.timestamp).getTime()
    if (Number.isFinite(timestamp) && timestamp < earliest) {
      earliest = timestamp
    }
  })
  return earliest === Infinity ? undefined : earliest
}

function parseIntervalMs(interval?: string): number | undefined {
  if (!interval) return undefined
  const match = /^(\d+)(m|h)$/.exec(interval)
  if (!match) return undefined
  const amount = Number(match[1])
  if (!Number.isFinite(amount) || amount <= 0) return undefined
  return match[2] === 'm'
    ? amount * 60 * 1000
    : amount * 60 * 60 * 1000
}
