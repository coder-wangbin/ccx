import { computed } from 'vue'
import { streamTimeoutPresets } from '../utils/streamTimeoutPresets'

type StreamTimeoutPresetKey = 'gentle' | 'balanced' | 'aggressive' | 'custom'
type FormLike = {
  streamFirstContentTimeoutEnabled: boolean
  streamFirstContentTimeoutMs: number
  streamInactivityTimeoutEnabled: boolean
  streamInactivityTimeoutMs: number
  streamToolCallIdleTimeoutEnabled: boolean
  streamToolCallIdleTimeoutMs: number
}

export function useStreamTimeoutStrategy(form: FormLike) {
  const selectedStreamTimeoutStrategy = computed(() => {
    if (!form.streamFirstContentTimeoutEnabled && !form.streamInactivityTimeoutEnabled && !form.streamToolCallIdleTimeoutEnabled) {
      return 'inherit'
    }
    for (const [key, preset] of Object.entries(streamTimeoutPresets) as Array<[StreamTimeoutPresetKey, { firstContentMs: number; inactivityMs: number; toolCallIdleMs: number }]>) {
      if (
        form.streamFirstContentTimeoutEnabled
        && form.streamInactivityTimeoutEnabled
        && form.streamToolCallIdleTimeoutEnabled
        && form.streamFirstContentTimeoutMs === preset.firstContentMs
        && form.streamInactivityTimeoutMs === preset.inactivityMs
        && form.streamToolCallIdleTimeoutMs === preset.toolCallIdleMs
      ) {
        return key
      }
    }
    return 'custom'
  })

  const applyStreamTimeoutStrategy = (strategy: string | null) => {
    if (!strategy) return
    if (strategy === 'inherit') {
      form.streamFirstContentTimeoutEnabled = false
      form.streamInactivityTimeoutEnabled = false
      form.streamToolCallIdleTimeoutEnabled = false
      return
    }

    const preset = streamTimeoutPresets[strategy as keyof typeof streamTimeoutPresets]
    if (!preset) return
    form.streamFirstContentTimeoutEnabled = true
    form.streamFirstContentTimeoutMs = preset.firstContentMs
    form.streamInactivityTimeoutEnabled = true
    form.streamInactivityTimeoutMs = preset.inactivityMs
    form.streamToolCallIdleTimeoutEnabled = true
    form.streamToolCallIdleTimeoutMs = preset.toolCallIdleMs
  }

  return {
    selectedStreamTimeoutStrategy,
    applyStreamTimeoutStrategy,
  }
}
