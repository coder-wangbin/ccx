<script setup lang="ts">
import { computed } from 'vue'
import type {
  Channel,
  ChannelMetrics,
  ChannelRecentActivity,
} from '@/services/admin-api'
import { useLanguage } from '@/composables/useLanguage'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  Ban,
  CheckCircle2,
  Copy,
  Edit3,
  ExternalLink,
  GripVertical,
  Key,
  MoreVertical,
  Pause,
  Play,
  RotateCcw,
  Sparkles,
  Terminal,
  Trash2,
  XCircle,
  Zap,
} from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  channel: Channel
  metrics?: ChannelMetrics
  activity?: ChannelRecentActivity
  priority?: number
  inactive?: boolean
  supportsCapability?: boolean
  canDelete?: boolean
  canMoveTop?: boolean
  canMoveBottom?: boolean
}>(), {
  priority: 0,
  inactive: false,
  supportsCapability: true,
  canDelete: true,
  canMoveTop: false,
  canMoveBottom: false,
})

const emit = defineEmits<{
  edit: []
  delete: []
  logs: []
  capability: []
  status: []
  resume: []
  promote: []
  moveTop: []
  moveBottom: []
  disable: []
  enable: []
}>()

const { tf } = useLanguage()

const isSuspended = computed(() => props.channel.status === 'suspended')
const isDisabled = computed(() => props.channel.status === 'disabled')

const serviceTypeClass = computed(() => {
  const map: Record<string, string> = {
    openai: 'border-blue-500/25 bg-blue-500/10 text-blue-700 dark:text-blue-300',
    claude: 'border-orange-500/25 bg-orange-500/10 text-orange-700 dark:text-orange-300',
    gemini: 'border-purple-500/25 bg-purple-500/10 text-purple-700 dark:text-purple-300',
    responses: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
  }
  return map[props.channel.serviceType] || map.openai
})

const statusConfig = computed(() => {
  if (isDisabled.value) {
    return {
      label: tf('console.channelStatus.disabled', 'Standby'),
      icon: XCircle,
      class: 'border-rose-500/25 bg-rose-500/10 text-rose-700 dark:text-rose-300',
      dot: 'bg-rose-500',
    }
  }
  if (isSuspended.value) {
    return {
      label: tf('console.channelStatus.suspended', 'Suspended'),
      icon: Pause,
      class: 'border-amber-500/25 bg-amber-500/10 text-amber-700 dark:text-amber-300',
      dot: 'bg-amber-500',
    }
  }
  return {
    label: tf('console.channelStatus.active', 'Active'),
    icon: CheckCircle2,
    class: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
    dot: 'bg-emerald-500',
  }
})

const circuitDisplay = computed(() => {
  const state = props.metrics?.circuitState
  if (state === 'open') {
    return {
      label: tf('console.circuit.open', 'Circuit Open'),
      class: 'border-rose-500/25 bg-rose-500/10 text-rose-700 dark:text-rose-300',
    }
  }
  return null
})

const isPromoted = computed(() => {
  if (!props.channel.promotionUntil) return false
  return new Date(props.channel.promotionUntil).getTime() > Date.now()
})

const keyCount = computed(() => props.channel.apiKeys?.length ?? 0)
const disabledKeyCount = computed(() => props.channel.disabledApiKeys?.length ?? 0)

const successRateDisplay = computed(() => {
  const raw = props.metrics?.successRate
  if (raw === undefined || raw === null) return '—'
  const value = raw <= 1 ? raw * 100 : raw
  return `${value.toFixed(0)}%`
})

const requestDisplay = computed(() => {
  const value = props.metrics?.requestCount ?? 0
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return `${value}`
})

const rpmDisplay = computed(() => {
  const rpm = props.activity?.rpm ?? props.channel.rpm ?? 0
  if (!rpm) return '—'
  return rpm >= 10 ? rpm.toFixed(0) : rpm.toFixed(1)
})

const tpmDisplay = computed(() => {
  const tpm = props.activity?.tpm ?? 0
  if (!tpm) return '—'
  if (tpm >= 1_000_000) return `${(tpm / 1_000_000).toFixed(1)}M`
  if (tpm >= 1_000) return `${(tpm / 1_000).toFixed(1)}K`
  return tpm.toFixed(0)
})

const websiteUrl = computed(() => {
  if (props.channel.website) return props.channel.website
  try {
    const url = new URL(props.channel.baseUrl)
    return `${url.protocol}//${url.host}`
  } catch {
    return ''
  }
})

const baseUrlText = computed(() => {
  if (props.channel.baseUrls?.length) return props.channel.baseUrls.join(' · ')
  return props.channel.baseUrl || '—'
})

async function copyChannelInfo() {
  const lines = [
    ...(props.channel.baseUrls?.length ? props.channel.baseUrls : [props.channel.baseUrl]),
    ...(props.channel.apiKeys ?? []),
  ].map(item => item?.trim()).filter(Boolean)
  await navigator.clipboard?.writeText(lines.join('\n'))
}
</script>

<template>
  <div
    :class="[
      'group grid grid-cols-[24px_minmax(150px,1fr)_88px_74px_86px] items-center gap-1 border px-2 py-2 transition-all duration-200 lg:grid-cols-[36px_minmax(170px,1fr)_110px_82px_74px_128px] lg:gap-1.5 xl:grid-cols-[42px_minmax(220px,1fr)_120px_96px_88px_150px] xl:gap-3 xl:px-3 xl:py-2.5',
      'bg-card/75 hover:bg-card dark:bg-card/55 dark:hover:bg-card/80',
      inactive ? 'border-dashed border-border/80 opacity-80' : 'border-border',
      circuitDisplay ? 'ring-1 ring-rose-500/20' : '',
    ]"
  >
    <div class="flex items-center gap-2 text-muted-foreground">
      <GripVertical class="hidden h-4 w-4 cursor-grab opacity-60 group-hover:opacity-100 lg:block" />
      <span class="min-w-5 text-right font-mono text-[11px] font-bold text-foreground/70">
        {{ priority || '—' }}
      </span>
    </div>

    <div class="min-w-0 space-y-1">
      <div class="flex min-w-0 items-center gap-2">
        <span
          class="h-2 w-2 shrink-0 rounded-full shadow-[0_0_8px_currentColor]"
          :class="statusConfig.dot"
        />
        <button
          type="button"
          class="min-w-0 truncate text-left text-sm font-semibold text-foreground underline-offset-4 transition-colors hover:text-primary hover:underline focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
          @click.stop="emit('edit')"
          @keydown.enter.stop="emit('edit')"
          @keydown.space.prevent.stop="emit('edit')"
        >
          {{ channel.name }}
        </button>
        <Badge class="shrink-0 border text-[10px] uppercase" :class="serviceTypeClass">
          {{ channel.serviceType }}
        </Badge>
        <button
          type="button"
          class="inline-flex shrink-0 items-center gap-1 border border-border bg-secondary/50 px-1.5 py-0.5 text-[10px] font-semibold text-muted-foreground transition-colors hover:border-primary/30 hover:text-primary"
          @click.stop="emit('edit')"
        >
          <Key class="h-3 w-3" />
          {{ keyCount }} keys
        </button>
        <span
          v-if="isPromoted"
          class="inline-flex shrink-0 items-center gap-1 border border-purple-500/25 bg-purple-500/10 px-1.5 py-0.5 text-[10px] font-bold text-purple-700 dark:text-purple-300"
        >
          <Sparkles class="h-3 w-3" />
          PROMO
        </span>
        <span
          v-if="circuitDisplay"
          class="inline-flex shrink-0 items-center gap-1 border px-1.5 py-0.5 text-[10px] font-bold"
          :class="circuitDisplay.class"
        >
          <AlertTriangle class="h-3 w-3" />
          {{ circuitDisplay.label }}
        </span>
      </div>
      <div class="truncate font-mono text-[11px] text-muted-foreground" :title="baseUrlText">
        {{ baseUrlText }}
      </div>
      <div v-if="channel.description" class="truncate text-[11px] text-muted-foreground" :title="channel.description">
        {{ channel.description }}
      </div>
    </div>

    <div class="inline-flex items-center justify-center gap-1.5 border px-2 py-1 text-[11px] font-semibold" :class="statusConfig.class">
      <component :is="statusConfig.icon" class="h-3.5 w-3.5" />
      {{ statusConfig.label }}
    </div>

    <div class="space-y-0.5 text-right font-mono">
      <div class="text-[11px] font-bold text-foreground xl:text-xs">{{ rpmDisplay }} / {{ tpmDisplay }}</div>
      <div class="text-[10px] text-muted-foreground">RPM / TPM</div>
    </div>

    <div class="hidden space-y-0.5 text-right font-mono lg:block">
      <div class="text-[11px] font-bold text-foreground xl:text-xs">{{ requestDisplay }}</div>
      <div class="text-[10px] text-muted-foreground">{{ successRateDisplay }}</div>
    </div>

    <div class="flex items-center justify-end gap-1">
      <Button v-if="isDisabled" variant="outline" size="sm" class="px-2 text-xs lg:px-2 xl:px-3" @click="emit('enable')">
        <Play class="h-3.5 w-3.5" />
        <span class="hidden lg:inline">{{ tf('console.actions.enable', 'Enable') }}</span>
      </Button>
      <Button v-else-if="isSuspended" variant="outline" size="sm" class="px-2 text-xs lg:px-2 xl:px-3" @click="emit('status')">
        <Play class="h-3.5 w-3.5" />
        <span class="hidden lg:inline">{{ tf('console.actions.resume', 'Resume') }}</span>
      </Button>
      <Button v-else variant="outline" size="sm" class="px-2 text-xs lg:px-2 xl:px-3" @click="emit('status')">
        <Pause class="h-3.5 w-3.5" />
        <span class="hidden lg:inline">{{ tf('console.actions.suspend', 'Suspend') }}</span>
      </Button>

      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="ghost" size="icon-sm" class="shrink-0">
            <MoreVertical class="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-52">
          <DropdownMenuLabel>{{ tf('console.actions.label', 'Actions') }}</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem @click="emit('edit')">
              <Edit3 class="h-4 w-4" />
              {{ tf('console.actions.edit', 'Edit Channel') }}
            </DropdownMenuItem>
            <DropdownMenuItem v-if="supportsCapability" @click="emit('capability')">
              <Zap class="h-4 w-4" />
              {{ tf('console.actions.capability', 'Capability Test') }}
            </DropdownMenuItem>
            <DropdownMenuItem @click="emit('logs')">
              <Terminal class="h-4 w-4" />
              {{ tf('console.actions.logs', 'View Logs') }}
            </DropdownMenuItem>
            <DropdownMenuItem @click="copyChannelInfo">
              <Copy class="h-4 w-4" />
              {{ tf('console.actions.copy', 'Copy Config') }}
            </DropdownMenuItem>
            <DropdownMenuItem v-if="websiteUrl" as="a" :href="websiteUrl" target="_blank" rel="noopener">
              <ExternalLink class="h-4 w-4" />
              {{ tf('console.actions.website', 'Visit Website') }}
            </DropdownMenuItem>
          </DropdownMenuGroup>

          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem v-if="circuitDisplay" @click="emit('resume')">
              <RotateCcw class="h-4 w-4" />
              {{ tf('console.actions.resetCircuit', 'Reset Circuit Breaker') }}
            </DropdownMenuItem>
            <DropdownMenuItem v-if="!isPromoted && !isDisabled" @click="emit('promote')">
              <Sparkles class="h-4 w-4" />
              {{ tf('console.actions.promote', 'Promote') }}
            </DropdownMenuItem>
            <DropdownMenuItem v-if="canMoveTop" @click="emit('moveTop')">
              <ArrowUp class="h-4 w-4" />
              {{ tf('orchestration.moveTop', 'Move to top') }}
            </DropdownMenuItem>
            <DropdownMenuItem v-if="canMoveBottom" @click="emit('moveBottom')">
              <ArrowDown class="h-4 w-4" />
              {{ tf('orchestration.moveBottom', 'Move to bottom') }}
            </DropdownMenuItem>
            <DropdownMenuItem v-if="!isDisabled" @click="emit('disable')">
              <Ban class="h-4 w-4" />
              {{ tf('console.actions.disable', 'Move to Standby') }}
            </DropdownMenuItem>
          </DropdownMenuGroup>

          <DropdownMenuSeparator />
          <div class="px-2 py-1 text-[10px] text-muted-foreground">
            <Key class="mr-1 inline h-3 w-3" />
            {{ keyCount }} {{ tf('console.keys.active', 'active keys') }}
            <span v-if="disabledKeyCount"> · {{ disabledKeyCount }} {{ tf('console.keys.disabled', 'disabled keys') }}</span>
          </div>
          <DropdownMenuItem variant="destructive" :disabled="!canDelete" @click="canDelete && emit('delete')">
            <Trash2 class="h-4 w-4" />
            {{ tf('console.actions.delete', 'Delete Channel') }}
            <span v-if="!canDelete" class="ml-1 text-[10px] opacity-70">{{ tf('orchestration.keepOne', 'keep one') }}</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </div>
</template>
