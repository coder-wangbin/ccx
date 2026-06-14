<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { useLanguage } from '@/composables/useLanguage'

interface FormData {
  streamFirstContentTimeoutEnabled: boolean
  streamFirstContentTimeoutMs: number
  streamInactivityTimeoutEnabled: boolean
  streamInactivityTimeoutMs: number
  streamToolCallIdleTimeoutEnabled: boolean
  streamToolCallIdleTimeoutMs: number
}

const props = defineProps<{
  form: FormData
}>()

const emit = defineEmits<{
  'update:form': [value: Partial<FormData>]
}>()

const { tf } = useLanguage()

const streamTimeoutPresets = {
  gentle: { firstContentMs: 90000, inactivityMs: 90000, toolCallIdleMs: 300000 },
  balanced: { firstContentMs: 60000, inactivityMs: 60000, toolCallIdleMs: 180000 },
  aggressive: { firstContentMs: 30000, inactivityMs: 30000, toolCallIdleMs: 60000 },
} as const

function updateField<K extends keyof FormData>(key: K, value: FormData[K]) {
  emit('update:form', { [key]: value } as Partial<FormData>)
}

function applyStreamTimeoutPreset(presetKey: 'gentle' | 'balanced' | 'aggressive') {
  const preset = streamTimeoutPresets[presetKey]
  emit('update:form', {
    streamFirstContentTimeoutEnabled: true,
    streamFirstContentTimeoutMs: preset.firstContentMs,
    streamInactivityTimeoutEnabled: true,
    streamInactivityTimeoutMs: preset.inactivityMs,
    streamToolCallIdleTimeoutEnabled: true,
    streamToolCallIdleTimeoutMs: preset.toolCallIdleMs,
  } as Partial<FormData>)
}
</script>

<template>
  <div class="rounded-xl border border-border/60 bg-card/40 p-4 shadow-xs space-y-3">
    <div class="flex items-center justify-between">
      <div class="text-[10px] font-bold uppercase tracking-wider text-primary">{{ tf('console.form.streamTimeoutTitle', '流式断流超时控制') }}</div>
      <div class="flex gap-1">
        <Button
          size="sm"
          variant="outline"
          class="h-6 px-2 text-[10px]"
          @click="applyStreamTimeoutPreset('gentle')"
        >
          {{ tf('console.form.presetGentle', '温和') }}
        </Button>
        <Button
          size="sm"
          variant="outline"
          class="h-6 px-2 text-[10px]"
          @click="applyStreamTimeoutPreset('balanced')"
        >
          {{ tf('console.form.presetBalanced', '均衡') }}
        </Button>
        <Button
          size="sm"
          variant="outline"
          class="h-6 px-2 text-[10px]"
          @click="applyStreamTimeoutPreset('aggressive')"
        >
          {{ tf('console.form.presetAggressive', '激进') }}
        </Button>
      </div>
    </div>
    <div class="grid gap-3 md:grid-cols-3">
      <!-- 首字等待 -->
      <div class="border border-border/60 bg-background/60 p-3 rounded-xl space-y-2.5">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <Label class="text-xs font-semibold block">{{ tf('console.form.firstByteWait', '首字等待') }}</Label>
            <span class="text-[9px] text-muted-foreground leading-none">{{ tf('console.form.firstByteWaitHint', '未响应则自动断开') }}</span>
          </div>
          <Switch
            :model-value="form.streamFirstContentTimeoutEnabled"
            @update:model-value="updateField('streamFirstContentTimeoutEnabled', $event)"
          />
        </div>
        <div class="space-y-1" :class="{ 'opacity-50 pointer-events-none': !form.streamFirstContentTimeoutEnabled }">
          <div class="flex items-center justify-between text-[10px] font-mono font-medium text-muted-foreground">
            <span>{{ tf('console.form.timeoutThreshold', '超时阈值:') }}</span>
            <span class="text-primary font-bold">{{ (form.streamFirstContentTimeoutMs / 1000) }}s</span>
          </div>
          <input
            :value="form.streamFirstContentTimeoutMs"
            type="range"
            min="5000"
            max="300000"
            step="1000"
            class="w-full accent-primary h-1 bg-muted rounded-lg appearance-none cursor-pointer"
            :disabled="!form.streamFirstContentTimeoutEnabled"
            @input="updateField('streamFirstContentTimeoutMs', Number(($event.target as HTMLInputElement).value))"
          />
        </div>
      </div>

      <!-- 首字后断流 -->
      <div class="border border-border/60 bg-background/60 p-3 rounded-xl space-y-2.5">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <Label class="text-xs font-semibold block">{{ tf('console.form.idleAfterFirstByte', '首字后断流') }}</Label>
            <span class="text-[9px] text-muted-foreground leading-none">{{ tf('console.form.idleAfterFirstByteHint', '生成中途卡顿超时') }}</span>
          </div>
          <Switch
            :model-value="form.streamInactivityTimeoutEnabled"
            @update:model-value="updateField('streamInactivityTimeoutEnabled', $event)"
          />
        </div>
        <div class="space-y-1" :class="{ 'opacity-50 pointer-events-none': !form.streamInactivityTimeoutEnabled }">
          <div class="flex items-center justify-between text-[10px] font-mono font-medium text-muted-foreground">
            <span>{{ tf('console.form.timeoutThreshold', '超时阈值:') }}</span>
            <span class="text-primary font-bold">{{ (form.streamInactivityTimeoutMs / 1000) }}s</span>
          </div>
          <input
            :value="form.streamInactivityTimeoutMs"
            type="range"
            min="1000"
            max="180000"
            step="1000"
            class="w-full accent-primary h-1 bg-muted rounded-lg appearance-none cursor-pointer"
            :disabled="!form.streamInactivityTimeoutEnabled"
            @input="updateField('streamInactivityTimeoutMs', Number(($event.target as HTMLInputElement).value))"
          />
        </div>
      </div>

      <!-- 工具调用空闲 -->
      <div class="border border-border/60 bg-background/60 p-3 rounded-xl space-y-2.5">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <Label class="text-xs font-semibold block">{{ tf('console.form.toolCallIdle', '工具调用空闲') }}</Label>
            <span class="text-[9px] text-muted-foreground leading-none">{{ tf('console.form.toolCallIdleHint', 'FunctionCall 延迟') }}</span>
          </div>
          <Switch
            :model-value="form.streamToolCallIdleTimeoutEnabled"
            @update:model-value="updateField('streamToolCallIdleTimeoutEnabled', $event)"
          />
        </div>
        <div class="space-y-1" :class="{ 'opacity-50 pointer-events-none': !form.streamToolCallIdleTimeoutEnabled }">
          <div class="flex items-center justify-between text-[10px] font-mono font-medium text-muted-foreground">
            <span>{{ tf('console.form.timeoutThreshold', '超时阈值:') }}</span>
            <span class="text-primary font-bold">{{ (form.streamToolCallIdleTimeoutMs / 1000) }}s</span>
          </div>
          <input
            :value="form.streamToolCallIdleTimeoutMs"
            type="range"
            min="1000"
            max="180000"
            step="1000"
            class="w-full accent-primary h-1 bg-muted rounded-lg appearance-none cursor-pointer"
            :disabled="!form.streamToolCallIdleTimeoutEnabled"
            @input="updateField('streamToolCallIdleTimeoutMs', Number(($event.target as HTMLInputElement).value))"
          />
        </div>
      </div>
    </div>
  </div>
</template>
