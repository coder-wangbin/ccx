<template>
  <v-card
    :class="['conversation-card', { 'override-active': hasOverride }]"
    variant="outlined"
    @click="!expanded && $emit('toggleExpand')"
  >
    <v-card-text class="pa-3">
      <!-- Row 1: Status + Kind + ID + Stats -->
      <div class="d-flex align-center ga-2 mb-1">
        <v-icon :color="statusColor" size="10">mdi-circle</v-icon>
        <v-chip :color="kindColor" size="x-small" variant="flat">{{ conversation.kind }}</v-chip>
        <span class="text-caption font-weight-mono text-medium-emphasis">
          <v-tooltip :text="conversation.id" location="top">
            <template #activator="{ props: tp }">
              <span v-bind="tp">{{ conversation.id.slice(0, 12) }}...</span>
            </template>
          </v-tooltip>
        </span>
        <v-spacer />
        <span class="text-caption text-medium-emphasis">{{ conversation.requestCount }}x</span>
        <span class="text-caption text-medium-emphasis">{{ duration }}</span>
      </div>

      <!-- Row 2: Model + Channel chips (collapsed) -->
      <div v-if="!expanded" class="d-flex align-center ga-1 flex-wrap">
        <span class="text-body-2 mr-2">{{ conversation.lastModel }}</span>
        <v-chip
          v-for="(ch, i) in visibleChannels"
          :key="ch.index"
          :color="ch.index === conversation.currentChannel ? 'primary' : undefined"
          :variant="ch.index === conversation.currentChannel ? 'flat' : 'outlined'"
          size="x-small"
          @click.stop="handleQuickOverride(ch)"
        >
          {{ ch.name }}
          <template v-if="ch.index === conversation.currentChannel" #append>
            <v-icon size="10">mdi-check</v-icon>
          </template>
        </v-chip>
        <v-chip
          v-if="hiddenCount > 0"
          size="x-small"
          variant="text"
          @click.stop="$emit('toggleExpand')"
        >
          +{{ hiddenCount }}
        </v-chip>
      </div>

      <!-- Expanded: Override alert -->
      <v-alert
        v-if="expanded && hasOverride"
        type="warning"
        density="compact"
        variant="tonal"
        class="mb-2 mt-2"
      >
        <div class="d-flex align-center">
          <span class="text-caption">Custom sequence active — {{ remainingTime }}</span>
          <v-spacer />
          <v-btn size="x-small" variant="text" @click.stop="$emit('removeOverride', conversation.id)">
            Restore default
          </v-btn>
        </div>
      </v-alert>

      <!-- Expanded: Full channel sequence -->
      <div v-if="expanded" class="mt-3">
        <div class="text-caption text-medium-emphasis mb-1">{{ conversation.lastModel }}</div>
        <div class="channel-sequence">
          <div
            v-for="(ch, i) in channelSequence"
            :key="ch.index"
            :class="['channel-item d-flex align-center pa-1 rounded', { 'demoted': isDemoted(i) }]"
          >
            <v-icon size="16" class="drag-handle mr-1" color="grey">mdi-drag-horizontal-variant</v-icon>
            <span class="text-caption font-weight-medium mr-1">{{ i + 1 }}.</span>
            <span
              class="text-caption flex-grow-1 channel-name"
              @click.stop="handleMoveToTop(ch, i)"
            >
              {{ ch.name }}
            </span>
            <v-chip
              v-if="ch.index === conversation.currentChannel"
              size="x-small"
              color="primary"
              variant="flat"
              class="mr-1"
            >current</v-chip>
            <v-chip
              v-if="ch.status === 'suspended'"
              size="x-small"
              color="error"
              variant="flat"
              class="mr-1"
            >fused</v-chip>
            <v-btn
              icon
              size="x-small"
              variant="text"
              :disabled="i === channelSequence.length - 1"
              @click.stop="handleDemote(i)"
            >
              <v-icon size="14">mdi-arrow-down</v-icon>
            </v-btn>
          </div>
        </div>
        <div class="text-right mt-1">
          <v-btn size="x-small" variant="text" @click.stop="$emit('toggleExpand')">Collapse</v-btn>
        </div>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ConversationInfo, SequenceOverrideInfo, ChannelSequenceEntry } from '@/services/api'

interface ChannelInfo {
  index: number
  name: string
  status: string
}

const props = defineProps<{
  conversation: ConversationInfo
  override?: SequenceOverrideInfo
  availableChannels: ChannelInfo[]
  expanded: boolean
}>()

const emit = defineEmits<{
  toggleExpand: []
  setOverride: [convId: string, sequence: ChannelSequenceEntry[]]
  removeOverride: [convId: string]
}>()

const MAX_VISIBLE = 3

const hasOverride = computed(() => !!props.override)

const statusColor = computed(() => {
  switch (props.conversation.status) {
    case 'streaming': return 'success'
    case 'active': return 'info'
    default: return 'grey'
  }
})

const kindColor = computed(() => {
  switch (props.conversation.kind) {
    case 'messages': return 'purple'
    case 'chat': return 'blue'
    case 'responses': return 'teal'
    case 'gemini': return 'orange'
    case 'images': return 'pink'
    default: return 'grey'
  }
})

const duration = computed(() => {
  const start = new Date(props.conversation.createdAt).getTime()
  const now = Date.now()
  const mins = Math.floor((now - start) / 60000)
  if (mins < 1) return '<1m'
  if (mins < 60) return `${mins}m`
  return `${Math.floor(mins / 60)}h${mins % 60}m`
})

const remainingTime = computed(() => {
  if (!props.override) return ''
  const expires = new Date(props.override.expiresAt).getTime()
  const now = Date.now()
  const remaining = Math.max(0, expires - now)
  const mins = Math.floor(remaining / 60000)
  const secs = Math.floor((remaining % 60000) / 1000)
  return `${mins}:${secs.toString().padStart(2, '0')}`
})

const channelSequence = computed((): ChannelInfo[] => {
  if (props.override?.sequence) {
    return props.override.sequence.map(entry => {
      const ch = props.availableChannels.find(c => c.index === entry.channelIndex)
      return {
        index: entry.channelIndex,
        name: entry.channelName || ch?.name || `Channel ${entry.channelIndex}`,
        status: ch?.status || 'active'
      }
    })
  }
  return props.availableChannels.filter(ch => ch.status !== 'disabled')
})

const visibleChannels = computed(() => channelSequence.value.slice(0, MAX_VISIBLE))
const hiddenCount = computed(() => Math.max(0, channelSequence.value.length - MAX_VISIBLE))

function isDemoted(index: number): boolean {
  if (!props.override) return false
  return index >= channelSequence.value.length - 1
}

function buildSequence(channels: ChannelInfo[]): ChannelSequenceEntry[] {
  return channels.map(ch => ({ channelIndex: ch.index, channelName: ch.name }))
}

function handleQuickOverride(ch: ChannelInfo) {
  if (ch.index === conversation.currentChannel && !hasOverride.value) return
  const current = [...channelSequence.value]
  const idx = current.findIndex(c => c.index === ch.index)
  if (idx <= 0) return
  const [item] = current.splice(idx, 1)
  current.unshift(item)
  emit('setOverride', props.conversation.id, buildSequence(current))
}

function handleMoveToTop(ch: ChannelInfo, currentIdx: number) {
  if (currentIdx === 0) return
  const current = [...channelSequence.value]
  const [item] = current.splice(currentIdx, 1)
  current.unshift(item)
  emit('setOverride', props.conversation.id, buildSequence(current))
}

function handleDemote(index: number) {
  const current = [...channelSequence.value]
  if (index >= current.length - 1) return
  const [item] = current.splice(index, 1)
  current.push(item)
  emit('setOverride', props.conversation.id, buildSequence(current))
}

const conversation = props.conversation
</script>

<style scoped>
.conversation-card {
  cursor: pointer;
  transition: border-color 0.2s;
}
.conversation-card:hover {
  border-color: rgb(var(--v-theme-primary));
}
.conversation-card.override-active {
  border-color: rgb(var(--v-theme-warning));
  border-width: 2px;
}
.font-weight-mono {
  font-family: monospace;
}
.channel-sequence {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
  overflow: hidden;
}
.channel-item {
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.channel-item:last-child {
  border-bottom: none;
}
.channel-item.demoted {
  opacity: 0.5;
}
.channel-name {
  cursor: pointer;
}
.channel-name:hover {
  text-decoration: underline;
  color: rgb(var(--v-theme-primary));
}
.drag-handle {
  cursor: grab;
}
</style>