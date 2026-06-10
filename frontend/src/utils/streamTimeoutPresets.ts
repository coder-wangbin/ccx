// 全局与渠道级共用的流式超时预设（三套固定值）
export const streamTimeoutPresets = {
  gentle: { firstContentMs: 90000, inactivityMs: 90000, toolCallIdleMs: 300000 },
  balanced: { firstContentMs: 60000, inactivityMs: 60000, toolCallIdleMs: 180000 },
  aggressive: { firstContentMs: 30000, inactivityMs: 30000, toolCallIdleMs: 60000 },
} as const

export type StreamTimeoutPresetKey = 'gentle' | 'balanced' | 'aggressive'
