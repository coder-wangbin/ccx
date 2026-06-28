<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert } from '@/components/ui/alert'
import { Save, RefreshCw, Zap } from 'lucide-vue-next'
import { useStatus } from '@/composables/useStatus'
import { useLanguage } from '@/composables/useLanguage'
import { streamTimeoutPresets } from '@/utils/stream-timeout-presets'
import { GetAdminAccessKey } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

const { status } = useStatus()
const { t } = useLanguage()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref('')
let messageTimer: ReturnType<typeof setTimeout> | null = null

const activePreset = ref('balanced')
const form = reactive({
  windowSize: 10,
  failureThreshold: 0.5,
  consecutiveFailuresThreshold: 3,
  requestTimeoutMs: 120000,
  responseHeaderTimeoutMs: 60000,
  streamFirstContentTimeoutMs: streamTimeoutPresets.balanced.firstContentMs,
  streamInactivityTimeoutMs: streamTimeoutPresets.balanced.inactivityMs,
  streamToolCallIdleTimeoutMs: streamTimeoutPresets.balanced.toolCallIdleMs,
})

const sliderStyle = (value: number, min: number, max: number) => {
  const percent = ((value - min) / (max - min)) * 100
  return { '--cb-slider-progress': `${Math.min(100, Math.max(0, percent))}%` }
}

// 工具调用 idle 预设按低速 5 TPS 粗估：60/120/300s 分别预留约 300/600/1500 token 的参数生成窗口。
const presets = [
  { key: 'gentle', labelKey: 'env.runtimeCbPresetGentle' as const, windowSize: 20, failureThreshold: 0.70, consecutiveFailuresThreshold: 5, requestTimeoutMs: 300000, responseHeaderTimeoutMs: 120000, streamFirstContentTimeoutMs: streamTimeoutPresets.gentle.firstContentMs, streamInactivityTimeoutMs: streamTimeoutPresets.gentle.inactivityMs, streamToolCallIdleTimeoutMs: streamTimeoutPresets.gentle.toolCallIdleMs },
  { key: 'balanced', labelKey: 'env.runtimeCbPresetBalanced' as const, windowSize: 10, failureThreshold: 0.50, consecutiveFailuresThreshold: 3, requestTimeoutMs: 120000, responseHeaderTimeoutMs: 60000, streamFirstContentTimeoutMs: streamTimeoutPresets.balanced.firstContentMs, streamInactivityTimeoutMs: streamTimeoutPresets.balanced.inactivityMs, streamToolCallIdleTimeoutMs: streamTimeoutPresets.balanced.toolCallIdleMs },
  { key: 'aggressive', labelKey: 'env.runtimeCbPresetAggressive' as const, windowSize: 5, failureThreshold: 0.30, consecutiveFailuresThreshold: 2, requestTimeoutMs: 60000, responseHeaderTimeoutMs: 30000, streamFirstContentTimeoutMs: streamTimeoutPresets.aggressive.firstContentMs, streamInactivityTimeoutMs: streamTimeoutPresets.aggressive.inactivityMs, streamToolCallIdleTimeoutMs: streamTimeoutPresets.aggressive.toolCallIdleMs },
  { key: 'custom', labelKey: 'env.runtimeCbPresetCustom' as const, windowSize: 10, failureThreshold: 0.50, consecutiveFailuresThreshold: 3, requestTimeoutMs: 120000, responseHeaderTimeoutMs: 60000, streamFirstContentTimeoutMs: streamTimeoutPresets.balanced.firstContentMs, streamInactivityTimeoutMs: streamTimeoutPresets.balanced.inactivityMs, streamToolCallIdleTimeoutMs: streamTimeoutPresets.balanced.toolCallIdleMs },
]

// 历史图片轮次限制
const historicalImageLimit = ref(0)

const matchPreset = () => {
  for (const p of presets) {
    if (p.key === 'custom') continue
    if (form.windowSize === p.windowSize && form.failureThreshold === p.failureThreshold && form.consecutiveFailuresThreshold === p.consecutiveFailuresThreshold && form.requestTimeoutMs === p.requestTimeoutMs && form.responseHeaderTimeoutMs === p.responseHeaderTimeoutMs && form.streamFirstContentTimeoutMs === p.streamFirstContentTimeoutMs && form.streamInactivityTimeoutMs === p.streamInactivityTimeoutMs && form.streamToolCallIdleTimeoutMs === p.streamToolCallIdleTimeoutMs) {
      activePreset.value = p.key
      return
    }
  }
  activePreset.value = 'custom'
}

const applyPreset = (preset: typeof presets[number]) => {
  if (preset.key === 'custom') return
  form.windowSize = preset.windowSize
  form.failureThreshold = preset.failureThreshold
  form.consecutiveFailuresThreshold = preset.consecutiveFailuresThreshold
  form.requestTimeoutMs = preset.requestTimeoutMs
  form.responseHeaderTimeoutMs = preset.responseHeaderTimeoutMs
  form.streamFirstContentTimeoutMs = preset.streamFirstContentTimeoutMs
  form.streamInactivityTimeoutMs = preset.streamInactivityTimeoutMs
  form.streamToolCallIdleTimeoutMs = preset.streamToolCallIdleTimeoutMs
  activePreset.value = preset.key
}

const onSliderChange = (field: string, event: Event) => {
  const val = Number((event.target as HTMLInputElement).value)
  if (field === 'failureThreshold') {
    form.failureThreshold = Math.round(val * 100) / 100
  } else if (field === 'windowSize') {
    form.windowSize = val
  } else if (field === 'consecutiveFailuresThreshold') {
    form.consecutiveFailuresThreshold = val
  } else if (field === 'requestTimeoutMs') {
    form.requestTimeoutMs = val
  } else if (field === 'responseHeaderTimeoutMs') {
    form.responseHeaderTimeoutMs = val
  } else if (field === 'streamFirstContentTimeoutMs') {
    form.streamFirstContentTimeoutMs = val
  } else if (field === 'streamInactivityTimeoutMs') {
    form.streamInactivityTimeoutMs = val
  } else if (field === 'streamToolCallIdleTimeoutMs') {
    form.streamToolCallIdleTimeoutMs = val
  }
  matchPreset()
}

const clearMessages = () => {
  error.value = ''
  success.value = ''
  if (messageTimer) {
    clearTimeout(messageTimer)
    messageTimer = null
  }
}

const showMessage = (msg: string, type: 'success' | 'error') => {
  clearMessages()
  if (type === 'success') {
    success.value = msg
  } else {
    error.value = msg
  }
  messageTimer = setTimeout(clearMessages, 5000)
}

const buildApiUrl = async (path: string): Promise<string | null> => {
  if (!status.value.url) return null
  return `${status.value.url}${path}`
}

const fetchConfig = async () => {
  const url = await buildApiUrl('/api/settings/circuit-breaker')
  if (!url) return

  loading.value = true
  clearMessages()
  try {
    const adminKey = await GetAdminAccessKey()
    const resp = await fetch(url, {
      headers: { 'x-api-key': adminKey },
    })
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const data = await resp.json()
    form.windowSize = data.windowSize ?? 10
    form.failureThreshold = data.failureThreshold ?? 0.5
    form.consecutiveFailuresThreshold = data.consecutiveFailuresThreshold ?? 3
    form.requestTimeoutMs = data.requestTimeoutMs && data.requestTimeoutMs >= 1000 ? data.requestTimeoutMs : 120000
    form.responseHeaderTimeoutMs = data.responseHeaderTimeoutMs && data.responseHeaderTimeoutMs >= 1000 ? data.responseHeaderTimeoutMs : 60000
    form.streamFirstContentTimeoutMs = data.streamFirstContentTimeoutMs && data.streamFirstContentTimeoutMs >= 5000 ? data.streamFirstContentTimeoutMs : 60000
    form.streamInactivityTimeoutMs = data.streamInactivityTimeoutMs && data.streamInactivityTimeoutMs >= 1000 ? data.streamInactivityTimeoutMs : 60000
    form.streamToolCallIdleTimeoutMs = data.streamToolCallIdleTimeoutMs && data.streamToolCallIdleTimeoutMs >= 30000 ? data.streamToolCallIdleTimeoutMs : 180000
    matchPreset()
  } catch (e) {
    showMessage(t('env.runtimeCbLoadFailed', { error: e instanceof Error ? e.message : String(e) }), 'error')
  } finally {
    loading.value = false
  }
}

const fetchHistoricalImageLimit = async () => {
  const url = await buildApiUrl('/api/settings/historical-image-turn-limit')
  if (!url) return

  try {
    const adminKey = await GetAdminAccessKey()
    const resp = await fetch(url, {
      headers: { 'x-api-key': adminKey },
    })
    if (resp.ok) {
      const data = await resp.json()
      historicalImageLimit.value = data.historicalImageTurnLimit ?? 0
    }
  } catch {
    // 非关键功能，静默忽略
  }
}

const saveConfig = async () => {
  const cbUrl = await buildApiUrl('/api/settings/circuit-breaker')
  const imgUrl = await buildApiUrl('/api/settings/historical-image-turn-limit')
  if (!cbUrl) {
    showMessage(t('env.runtimeCbNoBackend'), 'error')
    return
  }

  saving.value = true
  clearMessages()
  try {
    const adminKey = await GetAdminAccessKey()
    const promises: Promise<Response>[] = [
      fetch(cbUrl, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', 'x-api-key': adminKey },
        body: JSON.stringify({
          windowSize: form.windowSize,
          failureThreshold: form.failureThreshold,
          consecutiveFailuresThreshold: form.consecutiveFailuresThreshold,
          requestTimeoutMs: form.requestTimeoutMs,
          responseHeaderTimeoutMs: form.responseHeaderTimeoutMs,
          streamFirstContentTimeoutMs: form.streamFirstContentTimeoutMs,
          streamInactivityTimeoutMs: form.streamInactivityTimeoutMs,
          streamToolCallIdleTimeoutMs: form.streamToolCallIdleTimeoutMs,
        }),
      }),
    ]
    if (imgUrl) {
      promises.push(
        fetch(imgUrl, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json', 'x-api-key': adminKey },
          body: JSON.stringify({ limit: historicalImageLimit.value }),
        }),
      )
    }
    const results = await Promise.all(promises)
    for (const resp of results) {
      if (!resp.ok) {
        const body = await resp.json().catch(() => ({}))
        throw new Error(body.error || `HTTP ${resp.status}`)
      }
    }
    showMessage(t('env.runtimeCbSaved'), 'success')
  } catch (e) {
    showMessage(t('env.runtimeCbSaveFailed', { error: e instanceof Error ? e.message : String(e) }), 'error')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  if (status.value.running) {
    fetchConfig()
    fetchHistoricalImageLimit()
  }
})
</script>

<template>
  <Card>
    <CardHeader class="pb-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <CardTitle class="text-base flex items-center gap-2">
            <Zap class="w-4 h-4" />
            {{ t('env.runtimeCbTitle') }}
          </CardTitle>
          <p class="text-xs text-muted-foreground mt-1">{{ t('env.runtimeCbDesc') }}</p>
        </div>
        <div class="flex gap-2">
          <Button size="sm" variant="ghost" :disabled="loading || !status.running" @click="fetchConfig">
            <RefreshCw class="w-4 h-4 mr-1.5" :class="{ 'animate-spin': loading }" />
            {{ t('env.refresh') }}
          </Button>
          <Button size="sm" :disabled="saving || !status.running" @click="saveConfig">
            <Save class="w-4 h-4 mr-1.5" :class="{ 'animate-spin': saving }" />
            {{ saving ? t('env.saving') : t('env.save') }}
          </Button>
        </div>
      </div>

      <Alert v-if="!status.running" variant="default" class="mt-3">
        <p class="text-sm">{{ t('env.runtimeCbServiceStopped') }}</p>
      </Alert>
      <Alert v-if="error" variant="destructive" class="mt-3">
        <p class="text-sm">{{ error }}</p>
      </Alert>
      <Alert v-if="success" variant="default" class="mt-3">
        <p class="text-sm text-green-600">{{ success }}</p>
      </Alert>
    </CardHeader>

    <CardContent class="space-y-4">
      <!-- Sliders - 三列并排：基础参数 -->
      <div class="flex mb-4">
        <!-- 滑动窗口大小 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbWindowSize') }}</span>
            <span class="text-xs font-medium">{{ form.windowSize }}</span>
          </div>
          <div class="cb-slider-shell" :style="sliderStyle(form.windowSize, 3, 100)">
            <input
              type="range"
              :value="form.windowSize"
              :min="3"
              :max="100"
              step="1"
              class="cb-slider-input"
              :disabled="!status.running"
              :aria-label="t('env.runtimeCbWindowSize')"
              @input="onSliderChange('windowSize', $event)"
            />
            <div class="cb-slider-visual" aria-hidden="true">
              <div class="cb-slider-track">
                <div class="cb-slider-fill" />
              </div>
              <div class="cb-slider-thumb" />
            </div>
          </div>
          <div class="flex justify-between text-xs text-muted-foreground"><span>3</span><span>100</span></div>
        </div>

        <div class="w-px bg-border mx-1 self-stretch" />

        <!-- 失败率阈值 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbFailureThreshold') }}</span>
            <span class="text-xs font-medium">{{ form.failureThreshold.toFixed(2) }}</span>
          </div>
          <div class="cb-slider-shell" :style="sliderStyle(form.failureThreshold, 0.01, 1)">
            <input
              type="range"
              :value="form.failureThreshold"
              :min="0.01"
              :max="1"
              step="0.01"
              class="cb-slider-input"
              :disabled="!status.running"
              :aria-label="t('env.runtimeCbFailureThreshold')"
              @input="onSliderChange('failureThreshold', $event)"
            />
            <div class="cb-slider-visual" aria-hidden="true">
              <div class="cb-slider-track">
                <div class="cb-slider-fill" />
              </div>
              <div class="cb-slider-thumb" />
            </div>
          </div>
          <div class="flex justify-between text-xs text-muted-foreground"><span>0.01</span><span>1.00</span></div>
        </div>

        <div class="w-px bg-border mx-1 self-stretch" />

        <!-- 连续失败阈值 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbConsecutiveFailures') }}</span>
            <span class="text-xs font-medium">{{ form.consecutiveFailuresThreshold }}</span>
          </div>
          <div class="cb-slider-shell" :style="sliderStyle(form.consecutiveFailuresThreshold, 1, 100)">
            <input
              type="range"
              :value="form.consecutiveFailuresThreshold"
              :min="1"
              :max="100"
              step="1"
              class="cb-slider-input"
              :disabled="!status.running"
              :aria-label="t('env.runtimeCbConsecutiveFailures')"
              @input="onSliderChange('consecutiveFailuresThreshold', $event)"
            />
            <div class="cb-slider-visual" aria-hidden="true">
              <div class="cb-slider-track">
                <div class="cb-slider-fill" />
              </div>
              <div class="cb-slider-thumb" />
            </div>
          </div>
          <div class="flex justify-between text-xs text-muted-foreground"><span>1</span><span>100</span></div>
        </div>
      </div>

      <!-- Sliders - 请求生命周期超时 -->
      <div class="flex mb-4">
        <!-- 非流式请求超时 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbRequestTimeout') }}</span>
            <span class="text-xs font-medium">{{ (form.requestTimeoutMs / 1000) + 's' }}</span>
          </div>
          <div class="cb-slider-shell" :style="sliderStyle(form.requestTimeoutMs, 1000, 300000)">
            <input
              type="range"
              :value="form.requestTimeoutMs"
              :min="1000"
              :max="300000"
              step="1000"
              class="cb-slider-input"
              :disabled="!status.running"
              :aria-label="t('env.runtimeCbRequestTimeout')"
              @input="onSliderChange('requestTimeoutMs', $event)"
            />
            <div class="cb-slider-visual" aria-hidden="true">
              <div class="cb-slider-track">
                <div class="cb-slider-fill" />
              </div>
              <div class="cb-slider-thumb" />
            </div>
          </div>
          <div class="flex justify-between text-xs text-muted-foreground"><span>1s</span><span>300s</span></div>
        </div>

        <div class="w-px bg-border mx-1 self-stretch" />

        <!-- 响应头等待超时 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbResponseHeaderTimeout') }}</span>
            <span class="text-xs font-medium">{{ (form.responseHeaderTimeoutMs / 1000) + 's' }}</span>
          </div>
          <div class="cb-slider-shell" :style="sliderStyle(form.responseHeaderTimeoutMs, 1000, 300000)">
            <input
              type="range"
              :value="form.responseHeaderTimeoutMs"
              :min="1000"
              :max="300000"
              step="1000"
              class="cb-slider-input"
              :disabled="!status.running"
              :aria-label="t('env.runtimeCbResponseHeaderTimeout')"
              @input="onSliderChange('responseHeaderTimeoutMs', $event)"
            />
            <div class="cb-slider-visual" aria-hidden="true">
              <div class="cb-slider-track">
                <div class="cb-slider-fill" />
              </div>
              <div class="cb-slider-thumb" />
            </div>
          </div>
          <div class="flex justify-between text-xs text-muted-foreground"><span>1s</span><span>300s</span></div>
        </div>
      </div>

      <!-- Sliders - 流式健康检测超时 -->
      <div class="flex mb-4">
        <!-- 首字等待超时 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbStreamFirstContentTimeout') }}</span>
            <span class="text-xs font-medium">{{ (form.streamFirstContentTimeoutMs / 1000) + 's' }}</span>
          </div>
          <div class="cb-slider-shell" :style="sliderStyle(form.streamFirstContentTimeoutMs, 5000, 300000)">
            <input
              type="range"
              :value="form.streamFirstContentTimeoutMs"
              :min="5000"
              :max="300000"
              step="1000"
              class="cb-slider-input"
              :disabled="!status.running"
              :aria-label="t('env.runtimeCbStreamFirstContentTimeout')"
              @input="onSliderChange('streamFirstContentTimeoutMs', $event)"
            />
            <div class="cb-slider-visual" aria-hidden="true">
              <div class="cb-slider-track">
                <div class="cb-slider-fill" />
              </div>
              <div class="cb-slider-thumb" />
            </div>
          </div>
          <div class="flex justify-between text-xs text-muted-foreground"><span>5s</span><span>300s</span></div>
        </div>

        <div class="w-px bg-border mx-1 self-stretch" />

        <!-- 首字后断流超时 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbStreamInactivityTimeout') }}</span>
            <span class="text-xs font-medium">{{ (form.streamInactivityTimeoutMs / 1000) + 's' }}</span>
          </div>
          <div class="cb-slider-shell" :style="sliderStyle(form.streamInactivityTimeoutMs, 1000, 180000)">
            <input
              type="range"
              :value="form.streamInactivityTimeoutMs"
              :min="1000"
              :max="180000"
              step="1000"
              class="cb-slider-input"
              :disabled="!status.running"
              :aria-label="t('env.runtimeCbStreamInactivityTimeout')"
              @input="onSliderChange('streamInactivityTimeoutMs', $event)"
            />
            <div class="cb-slider-visual" aria-hidden="true">
              <div class="cb-slider-track">
                <div class="cb-slider-fill" />
              </div>
              <div class="cb-slider-thumb" />
            </div>
          </div>
          <div class="flex justify-between text-xs text-muted-foreground"><span>1s</span><span>180s</span></div>
        </div>

        <div class="w-px bg-border mx-1 self-stretch" />

        <!-- 工具调用空闲超时 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbStreamToolCallIdleTimeout') }}</span>
            <span class="text-xs font-medium">{{ (form.streamToolCallIdleTimeoutMs / 1000) + 's' }}</span>
          </div>
          <div class="cb-slider-shell" :style="sliderStyle(form.streamToolCallIdleTimeoutMs, 30000, 300000)">
            <input
              type="range"
              :value="form.streamToolCallIdleTimeoutMs"
              :min="30000"
              :max="300000"
              step="1000"
              class="cb-slider-input"
              :disabled="!status.running"
              :aria-label="t('env.runtimeCbStreamToolCallIdleTimeout')"
              @input="onSliderChange('streamToolCallIdleTimeoutMs', $event)"
            />
            <div class="cb-slider-visual" aria-hidden="true">
              <div class="cb-slider-track">
                <div class="cb-slider-fill" />
              </div>
              <div class="cb-slider-thumb" />
            </div>
          </div>
          <div class="flex justify-between text-xs text-muted-foreground"><span>30s</span><span>300s</span></div>
        </div>
      </div>

      <!-- Preset buttons -->
      <div class="flex gap-2">
        <Button
          v-for="p in presets"
          :key="p.key"
          size="sm"
          :variant="activePreset === p.key ? 'default' : 'outline'"
          :disabled="!status.running"
          @click="applyPreset(p)"
        >
          {{ t(p.labelKey) }}
        </Button>
      </div>

      <!-- 历史图片轮次限制 -->
      <div class="border-t border-border pt-4 mt-4">
        <div class="flex items-center justify-between mb-2">
          <div>
            <p class="text-sm font-medium">{{ t('env.historicalImageTurnLimitTitle') }}</p>
            <p class="text-xs text-muted-foreground mt-0.5">{{ t('env.historicalImageTurnLimitHint') }}</p>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs text-muted-foreground">{{ t('env.historicalImageTurnLimitLabel') }}</span>
            <input
              v-model.number="historicalImageLimit"
              type="number"
              min="0"
              class="w-20 h-8 rounded border border-input bg-background px-2 text-sm text-center"
              :disabled="!status.running"
            />
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

<style scoped>
.cb-slider-shell {
  --cb-slider-progress: 0%;
  --cb-slider-track-bg: color-mix(in srgb, var(--color-input) 76%, var(--color-muted) 24%);
  --cb-slider-fill-bg: color-mix(in srgb, var(--color-primary) 72%, transparent);
  --cb-slider-thumb-ring: color-mix(in srgb, var(--color-primary) 18%, var(--color-border));
  --cb-slider-thumb-hover-ring: color-mix(in srgb, var(--color-primary) 42%, var(--color-border));
  --cb-slider-thumb-hover-glow: color-mix(in srgb, var(--color-primary) 14%, transparent);
  --cb-slider-thumb-active-ring: color-mix(in srgb, var(--color-primary) 52%, var(--color-border));
  --cb-slider-thumb-active-glow: color-mix(in srgb, var(--color-primary) 16%, transparent);
  position: relative;
  width: 100%;
  height: 28px;
}
.cb-slider-input {
  -webkit-appearance: none;
  appearance: none;
  position: absolute;
  inset: 0;
  z-index: 2;
  width: 100%;
  height: 28px;
  margin: 0;
  background: transparent;
  opacity: 0;
  outline: none;
  cursor: pointer;
}
.cb-slider-input::-webkit-slider-runnable-track {
  height: 28px;
  background: transparent;
}
.cb-slider-input::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 26px;
  height: 28px;
  background: transparent;
  border: 0;
}
.cb-slider-input::-moz-range-track {
  height: 28px;
  background: transparent;
  border: 0;
}
.cb-slider-input::-moz-range-thumb {
  width: 26px;
  height: 28px;
  background: transparent;
  border: 0;
}
.cb-slider-input:disabled {
  cursor: not-allowed;
}
.cb-slider-visual {
  position: absolute;
  top: 0;
  right: 13px;
  bottom: 0;
  left: 13px;
  pointer-events: none;
}
.cb-slider-track {
  position: absolute;
  top: 50%;
  right: 0;
  left: 0;
  height: 4px;
  transform: translateY(-50%);
  border-radius: 999px;
  background: var(--cb-slider-track-bg);
  box-shadow: inset 0 1px 1px rgb(15 23 42 / 0.08);
}
.cb-slider-fill {
  width: var(--cb-slider-progress);
  height: 100%;
  border-radius: inherit;
  background: var(--cb-slider-fill-bg);
}
.cb-slider-thumb {
  position: absolute;
  top: 50%;
  left: var(--cb-slider-progress);
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--color-primary);
  cursor: pointer;
  border: 3px solid var(--color-background);
  box-shadow:
    0 0 0 1px var(--cb-slider-thumb-ring),
    0 2px 5px rgb(15 23 42 / 0.16);
  transform: translate(-50%, -50%);
  transition: all 0.15s cubic-bezier(0.4, 0, 0.2, 1);
}
.cb-slider-input:hover + .cb-slider-visual .cb-slider-thumb,
.cb-slider-input:focus-visible + .cb-slider-visual .cb-slider-thumb {
  box-shadow:
    0 0 0 1px var(--cb-slider-thumb-hover-ring),
    0 0 0 4px var(--cb-slider-thumb-hover-glow),
    0 3px 7px rgb(15 23 42 / 0.18);
  transform: translate(-50%, -50%) scale(1.1);
}
.cb-slider-input:active + .cb-slider-visual .cb-slider-thumb {
  box-shadow:
    0 0 0 1px var(--cb-slider-thumb-active-ring),
    0 0 0 5px var(--cb-slider-thumb-active-glow),
    0 2px 5px rgb(15 23 42 / 0.16);
  transform: translate(-50%, -50%) scale(1.04);
}
.cb-slider-input:disabled + .cb-slider-visual {
  opacity: 0.5;
}
</style>
