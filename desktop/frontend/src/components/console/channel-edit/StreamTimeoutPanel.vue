<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { useLanguage } from '@/composables/useLanguage'
import { streamTimeoutPresets } from '@/utils/stream-timeout-presets'

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

const { t } = useLanguage()

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

function applyInheritStrategy() {
  emit('update:form', {
    streamFirstContentTimeoutEnabled: false,
    streamInactivityTimeoutEnabled: false,
    streamToolCallIdleTimeoutEnabled: false,
  } as Partial<FormData>)
}

const selectedStrategy = computed(() => {
  if (
    !props.form.streamFirstContentTimeoutEnabled &&
    !props.form.streamInactivityTimeoutEnabled &&
    !props.form.streamToolCallIdleTimeoutEnabled
  ) {
    return 'inherit'
  }
  for (const [key, preset] of Object.entries(streamTimeoutPresets)) {
    if (
      props.form.streamFirstContentTimeoutEnabled &&
      props.form.streamInactivityTimeoutEnabled &&
      props.form.streamToolCallIdleTimeoutEnabled &&
      props.form.streamFirstContentTimeoutMs === preset.firstContentMs &&
      props.form.streamInactivityTimeoutMs === preset.inactivityMs &&
      props.form.streamToolCallIdleTimeoutMs === preset.toolCallIdleMs
    ) {
      return key
    }
  }
  return 'custom'
})
</script>

<template>
  <div class="rounded-xl border border-border/60 bg-card/40 p-4 shadow-xs space-y-4">
    <div class="flex items-start justify-between gap-3 flex-wrap">
      <div>
        <div class="text-[10px] font-bold uppercase tracking-wider text-primary">
          {{ t('addChannel.streamTimeoutStrategyLabel') }}
        </div>
        <div class="text-[10px] leading-4 text-muted-foreground">
          {{ selectedStrategy === 'inherit' ? t('addChannel.streamTimeoutInheritHint') : t('addChannel.streamTimeoutOverrideHint') }}
        </div>
      </div>
      <div class="flex gap-1 flex-wrap">
        <Button
          type="button"
          size="sm"
          variant="outline"
          class="h-6 px-2 text-[10px]"
          :class="selectedStrategy === 'inherit' ? 'border-primary/40 text-primary' : ''"
          @click="applyInheritStrategy"
        >
          {{ t('addChannel.streamTimeoutStrategyInherit') }}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          class="h-6 px-2 text-[10px]"
          :class="selectedStrategy === 'gentle' ? 'border-primary/40 text-primary' : ''"
          @click="applyStreamTimeoutPreset('gentle')"
        >
          {{ t('channelEditor.streamTimeout.preset.gentle') }}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          class="h-6 px-2 text-[10px]"
          :class="selectedStrategy === 'balanced' ? 'border-primary/40 text-primary' : ''"
          @click="applyStreamTimeoutPreset('balanced')"
        >
          {{ t('channelEditor.streamTimeout.preset.balanced') }}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          class="h-6 px-2 text-[10px]"
          :class="selectedStrategy === 'aggressive' ? 'border-primary/40 text-primary' : ''"
          @click="applyStreamTimeoutPreset('aggressive')"
        >
          {{ t('channelEditor.streamTimeout.preset.aggressive') }}
        </Button>
      </div>
    </div>

    <div class="overflow-hidden rounded-xl border border-border/60 bg-background/60">
      <div class="grid gap-0 md:grid-cols-3">
        <div class="space-y-2.5 p-4" :class="{ 'opacity-50': !form.streamFirstContentTimeoutEnabled }">
          <div class="flex items-center justify-between gap-2">
            <span class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/70">
              {{ t('addChannel.streamFirstContentTimeoutLabel') }}
            </span>
            <span class="font-mono text-xs font-semibold text-primary">
              {{ form.streamFirstContentTimeoutMs / 1000 }}s
            </span>
          </div>
          <input
            :value="form.streamFirstContentTimeoutMs"
            type="range"
            min="5000"
            max="300000"
            step="1000"
            class="h-1 w-full cursor-pointer appearance-none rounded-lg bg-muted accent-primary"
            :disabled="!form.streamFirstContentTimeoutEnabled"
            @input="updateField('streamFirstContentTimeoutMs', Number(($event.target as HTMLInputElement).value))"
          />
          <div class="flex justify-between text-[10px] text-muted-foreground/70">
            <span>5s</span>
            <span>300s</span>
          </div>
        </div>

        <div class="space-y-2.5 border-t border-border/60 p-4 md:border-l md:border-t-0" :class="{ 'opacity-50': !form.streamInactivityTimeoutEnabled }">
          <div class="flex items-center justify-between gap-2">
            <span class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/70">
              {{ t('addChannel.streamInactivityTimeoutLabel') }}
            </span>
            <span class="font-mono text-xs font-semibold text-primary">
              {{ form.streamInactivityTimeoutMs / 1000 }}s
            </span>
          </div>
          <input
            :value="form.streamInactivityTimeoutMs"
            type="range"
            min="1000"
            max="180000"
            step="1000"
            class="h-1 w-full cursor-pointer appearance-none rounded-lg bg-muted accent-primary"
            :disabled="!form.streamInactivityTimeoutEnabled"
            @input="updateField('streamInactivityTimeoutMs', Number(($event.target as HTMLInputElement).value))"
          />
          <div class="flex justify-between text-[10px] text-muted-foreground/70">
            <span>1s</span>
            <span>180s</span>
          </div>
        </div>

        <div class="space-y-2.5 border-t border-border/60 p-4 md:border-l md:border-t-0" :class="{ 'opacity-50': !form.streamToolCallIdleTimeoutEnabled }">
          <div class="flex items-center justify-between gap-2">
            <span class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/70">
              {{ t('addChannel.streamToolCallIdleTimeoutLabel') }}
            </span>
            <span class="font-mono text-xs font-semibold text-primary">
              {{ form.streamToolCallIdleTimeoutMs / 1000 }}s
            </span>
          </div>
          <input
            :value="form.streamToolCallIdleTimeoutMs"
            type="range"
            min="30000"
            max="300000"
            step="1000"
            class="h-1 w-full cursor-pointer appearance-none rounded-lg bg-muted accent-primary"
            :disabled="!form.streamToolCallIdleTimeoutEnabled"
            @input="updateField('streamToolCallIdleTimeoutMs', Number(($event.target as HTMLInputElement).value))"
          />
          <div class="flex justify-between text-[10px] text-muted-foreground/70">
            <span>30s</span>
            <span>300s</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
