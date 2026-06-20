import { computed, ref, watch, markRaw, type Ref } from 'vue'
import type { ChannelRecentActivity, ActivitySegment } from '../services/api'
import { expandSparseSegments } from '../services/api-helpers'

/**
 * Activity 可视化相关的状态和计算逻辑。
 * 从 ChannelOrchestration.vue 抽出，降低单文件行数。
 */
export function useChannelActivity(recentActivity: Ref<ChannelRecentActivity[]>, activityTick?: Ref<number>) {
  const activityMap = computed(() => {
    const map = new Map<number, ChannelRecentActivity>()
    for (const a of recentActivity.value) {
      map.set(a.channelIndex, a)
    }
    return map
  })

  const maxRequestsHistory = ref(new Map<number, { max: number; updatedAt: number }>())
  const DECAY_HALF_LIFE = 5 * 60 * 1000  // Half-life: 5 minutes
  const MIN_MAX_REQUESTS = 1  // Minimum baseline value to avoid division by zero

  const getDecayedMax = (record: { max: number; updatedAt: number }, now: number): number => {
    const elapsed = now - record.updatedAt
    const decayFactor = Math.pow(0.5, elapsed / DECAY_HALF_LIFE)
    return Math.max(MIN_MAX_REQUESTS, record.max * decayFactor)
  }

  watch(activityMap, (newMap) => {
    const now = Date.now()
    for (const [channelIndex, activity] of newMap.entries()) {
      const segments = expandSparseSegments(activity)
      if (segments.length === 0) continue

      const currentMax = Math.max(...segments.map(s => s.requestCount), 0)

      const record = maxRequestsHistory.value.get(channelIndex)
      if (!record) {
        if (currentMax > 0) {
          maxRequestsHistory.value.set(channelIndex, { max: currentMax, updatedAt: now })
        }
        continue
      }

      const decayedMax = getDecayedMax(record, now)
      if (currentMax >= decayedMax) {
        maxRequestsHistory.value.set(channelIndex, { max: currentMax, updatedAt: now })
      } else {
        maxRequestsHistory.value.set(channelIndex, { max: decayedMax, updatedAt: now })
      }
    }
    // Clean up stale entries
    for (const key of maxRequestsHistory.value.keys()) {
      if (!newMap.has(key)) {
        maxRequestsHistory.value.delete(key)
      }
    }
  })

  const getChannelActivity = (channelIndex: number): ChannelRecentActivity | undefined => {
    return activityMap.value.get(channelIndex)
  }

  type ActivityBar = { x: number; y: number; width: number; height: number; radius: number; g: number; v: 0 | 1 }

  const activityBarsPersistentCache = new Map<number, { segments: ActivitySegment[], bars: ActivityBar[] }>()

  const activityBarsCache = computed(() => {
    const cache = new Map<number, ActivityBar[]>()
    void activityTick?.value

    for (const [channelIndex, activity] of activityMap.value.entries()) {
      if (!activity) {
        cache.set(channelIndex, [])
        continue
      }

      const existing = activityBarsPersistentCache.get(channelIndex)
      const segments = expandSparseSegments(activity, existing?.segments)
      const numSegments = segments.length

      if (numSegments === 0) {
        cache.set(channelIndex, [])
        continue
      }

      const barWidth = 150 / numSegments
      const barGap = barWidth * 0.2
      const actualBarWidth = barWidth - barGap

      const now = Date.now()
      const record = maxRequestsHistory.value.get(channelIndex)
      const currentMax = Math.max(...segments.map(s => s.requestCount), 1)
      const maxRequests = record ? Math.max(getDecayedMax(record, now), currentMax) : currentMax

      let bars: ActivityBar[]
      if (existing && existing.bars.length === numSegments) {
        bars = existing.bars
      } else {
        bars = new Array(numSegments)
        for (let i = 0; i < numSegments; i++) {
          bars[i] = { x: 0, y: 0, width: 0, height: 0, radius: 0, g: 0, v: 0 }
        }
      }

      for (let i = 0; i < numSegments; i++) {
        const segment = segments[i]
        const requests = segment.requestCount

        if (requests <= 0) {
          bars[i].v = 0
          bars[i].height = 0
          continue
        }

        const successCount = requests - segment.failureCount
        const successRate = (successCount / requests) * 100
        let g: number
        if (successRate < 5) g = 6
        else if (successRate < 20) g = 5
        else if (successRate < 40) g = 4
        else if (successRate < 60) g = 3
        else if (successRate < 80) g = 2
        else if (successRate < 95) g = 1
        else g = 0

        const heightPercent = requests / maxRequests
        const height = Math.max(heightPercent * 85, 2)

        bars[i].v = 1
        bars[i].x = i * barWidth + barGap / 2
        bars[i].y = 100 - height
        bars[i].width = actualBarWidth
        bars[i].height = height
        bars[i].radius = Math.min(actualBarWidth / 2, 1.5)
        bars[i].g = g
      }

      activityBarsPersistentCache.set(channelIndex, { segments, bars: markRaw(bars) })
      cache.set(channelIndex, bars)
    }

    for (const key of activityBarsPersistentCache.keys()) {
      if (!activityMap.value.has(key)) {
        activityBarsPersistentCache.delete(key)
      }
    }

    return cache
  })

  const getActivityBars = (channelIndex: number): ActivityBar[] => {
    return activityBarsCache.value.get(channelIndex) || []
  }

  const getActivityPath = (channelIndex: number): string => {
    const activity = getChannelActivity(channelIndex)
    if (!activity) return ''
    void activityTick?.value

    const segments = expandSparseSegments(activity)
    const numSegments = segments.length
    if (numSegments === 0) return ''

    const maxRequests = Math.max(...segments.map(s => s.requestCount), 1)
    const windowSize = 5
    const smoothedData: number[] = []

    for (let i = 0; i < numSegments; i++) {
      const start = Math.max(0, i - Math.floor(windowSize / 2))
      const end = Math.min(numSegments, i + Math.ceil(windowSize / 2))
      let sum = 0
      let count = 0

      for (let j = start; j < end; j++) {
        sum += segments[j].requestCount
        count++
      }

      smoothedData.push(count > 0 ? sum / count : 0)
    }

    const points: { x: number; y: number }[] = []
    for (let i = 0; i < numSegments; i++) {
      points.push({ x: i, y: 100 - (smoothedData[i] / maxRequests * 85) })
    }
    if (points.length < 2) return ''

    return catmullRomToPath(points)
  }

  function catmullRomToPath(points: { x: number; y: number }[]): string {
    if (points.length < 2) return ''
    const parts: string[] = [`M ${points[0].x} ${points[0].y}`]
    const tension = 0.3
    for (let i = 0; i < points.length - 1; i++) {
      const p0 = points[Math.max(0, i - 1)]
      const p1 = points[i]
      const p2 = points[i + 1]
      const p3 = points[Math.min(points.length - 1, i + 2)]
      const cp1x = p1.x + (p2.x - p0.x) * tension / 6
      const cp1y = p1.y + (p2.y - p0.y) * tension / 6
      const cp2x = p2.x - (p3.x - p1.x) * tension / 6
      const cp2y = p2.y - (p3.y - p1.y) * tension / 6
      parts.push(`C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${p2.x} ${p2.y}`)
    }
    return parts.join(' ')
  }

  const _getActivityAreaPath = (channelIndex: number): string => {
    const linePath = getActivityPath(channelIndex)
    if (!linePath) return ''

    const activity = getChannelActivity(channelIndex)
    if (!activity) return ''

    const segments = expandSparseSegments(activity)
    const numSegments = segments.length
    if (numSegments === 0) return ''

    return `${linePath} L ${numSegments - 1} 100 L 0 100 Z`
  }

  const _getActivityGradient = (channelIndex: number): string => {
    const activity = getChannelActivity(channelIndex)
    if (!activity) return 'transparent'
    void activityTick?.value

    const segments = expandSparseSegments(activity)
    const numSegments = segments.length
    if (numSegments === 0) return 'transparent'

    if (!segments.some(seg => seg.requestCount > 0)) return 'transparent'

    const segmentColors: string[] = []
    for (let i = 0; i < numSegments; i++) {
      const seg = segments[i]

      if (seg.requestCount === 0) {
        segmentColors.push('transparent')
        continue
      }

      if (seg.failureCount > 0) {
        const failureRatio = seg.failureCount / seg.requestCount
        if (failureRatio >= 0.5) {
          const intensity = Math.min(0.5, 0.2 + seg.requestCount * 0.01)
          segmentColors.push(`rgba(239, 68, 68, ${intensity})`)
        } else {
          const intensity = Math.min(0.4, 0.15 + seg.requestCount * 0.008)
          segmentColors.push(`rgba(251, 146, 60, ${intensity})`)
        }
        continue
      }

      if (seg.requestCount >= 20) segmentColors.push('rgba(22, 163, 74, 0.65)')
      else if (seg.requestCount >= 15) segmentColors.push('rgba(22, 163, 74, 0.55)')
      else if (seg.requestCount >= 10) segmentColors.push('rgba(34, 197, 94, 0.50)')
      else if (seg.requestCount >= 6) segmentColors.push('rgba(34, 197, 94, 0.42)')
      else if (seg.requestCount >= 3) segmentColors.push('rgba(74, 222, 128, 0.38)')
      else segmentColors.push('rgba(74, 222, 128, 0.30)')
    }

    const stops = segmentColors.map((color, i) => {
      const start = (i / numSegments * 100).toFixed(3)
      const end = ((i + 1) / numSegments * 100).toFixed(3)
      return `${color} ${start}%, ${color} ${end}%`
    }).join(', ')

    return `linear-gradient(to right, ${stops})`
  }

  const formatRPM = (channelIndex: number): string => {
    const activity = getChannelActivity(channelIndex)
    if (!activity || !activity.rpm) return '--'
    if (activity.rpm >= 10) return activity.rpm.toFixed(0)
    return activity.rpm.toFixed(1)
  }

  const formatTPM = (channelIndex: number): string => {
    const activity = getChannelActivity(channelIndex)
    if (!activity || !activity.tpm) return '--'
    if (activity.tpm >= 1000000) return `${(activity.tpm / 1000000).toFixed(1)}M`
    if (activity.tpm >= 1000) return `${(activity.tpm / 1000).toFixed(1)}K`
    return activity.tpm.toFixed(0)
  }

  const hasActivityData = (channelIndex: number): boolean => {
    const activity = getChannelActivity(channelIndex)
    if (!activity) return false
    return activity.rpm > 0 || activity.tpm > 0
  }

  return {
    activityMap,
    getChannelActivity,
    getActivityBars,
    getActivityPath,
    _getActivityAreaPath,
    _getActivityGradient,
    formatRPM,
    formatTPM,
    hasActivityData,
  }
}
