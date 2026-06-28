export interface StreamTimeoutPreset {
  firstContentMs: number
  inactivityMs: number
  toolCallIdleMs: number
}

export const streamTimeoutPresets: Record<'gentle' | 'balanced' | 'aggressive', StreamTimeoutPreset> = {
  gentle: { firstContentMs: 90000, inactivityMs: 90000, toolCallIdleMs: 300000 },
  balanced: { firstContentMs: 60000, inactivityMs: 60000, toolCallIdleMs: 180000 },
  aggressive: { firstContentMs: 30000, inactivityMs: 30000, toolCallIdleMs: 60000 },
}

export const defaultStreamTimeouts = streamTimeoutPresets.balanced
