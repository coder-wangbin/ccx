<template>
  <div class="conversation-dashboard">
    <!-- 过滤栏 -->
    <div class="d-flex align-center mb-4 flex-wrap ga-2">
      <v-chip-group v-model="kindFilter" mandatory>
        <v-chip value="" variant="elevated" size="small">All</v-chip>
        <v-chip value="messages" variant="elevated" size="small">Messages</v-chip>
        <v-chip value="chat" variant="elevated" size="small">Chat</v-chip>
        <v-chip value="responses" variant="elevated" size="small">Responses</v-chip>
        <v-chip value="gemini" variant="elevated" size="small">Gemini</v-chip>
        <v-chip value="images" variant="elevated" size="small">Images</v-chip>
      </v-chip-group>
      <v-spacer />
      <span class="text-caption text-medium-emphasis">
        Active: {{ filteredConversations.length }}
        <span v-if="overrideCount > 0" class="ml-2 text-warning">Override: {{ overrideCount }}</span>
      </span>
    </div>

    <!-- Loading -->
    <div v-if="loading && !conversations.length" class="d-flex justify-center py-12">
      <v-progress-circular indeterminate color="primary" />
    </div>

    <!-- Empty -->
    <v-card v-else-if="!filteredConversations.length" variant="outlined" class="text-center pa-12">
      <v-icon size="48" color="grey">mdi-chat-outline</v-icon>
      <div class="text-body-1 mt-4 text-medium-emphasis">
        No active sessions. Sessions will appear when requests pass through the gateway.
      </div>
    </v-card>

    <!-- Conversation cards -->
    <v-row v-else>
      <v-col v-for="conv in filteredConversations" :key="conv.id" cols="12" md="6">
        <ConversationCard
          :conversation="conv"
          :override="overrides[conv.id]"
          :available-channels="getChannelsForKind(conv.kind)"
          :expanded="expandedCards.has(conv.id)"
          @toggle-expand="toggleExpand(conv.id)"
          @set-override="handleSetOverride"
          @remove-override="handleRemoveOverride"
        />
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { api, type ConversationInfo, type SequenceOverrideInfo, type ChannelSequenceEntry } from '@/services/api'
import { useGlobalTick } from '@/composables/useGlobalTick'
import { useChannelStore } from '@/stores/channel'
import ConversationCard from './ConversationCard.vue'

const channelStore = useChannelStore()
const loading = ref(true)
const conversations = ref<ConversationInfo[]>([])
const overrides = ref<Record<string, SequenceOverrideInfo>>({})
const kindFilter = ref('')
const expandedCards = ref(new Set<string>())

const filteredConversations = computed(() => {
  const filter = kindFilter.value
  if (!filter) return conversations.value
  return conversations.value.filter(c => c.kind === filter)
})

const overrideCount = computed(() => Object.keys(overrides.value).length)

function getChannelsForKind(kind: string): { index: number; name: string; status: string }[] {
  const data = getChannelDataForKind(kind)
  if (!data?.channels) return []
  return data.channels.map((ch: any) => ({
    index: ch.index ?? 0,
    name: ch.name || `Channel ${ch.index}`,
    status: ch.status || 'active'
  }))
}

function getChannelDataForKind(kind: string) {
  switch (kind) {
    case 'messages': return channelStore.currentChannelsData
    case 'chat': return (channelStore as any).chatChannelsData
    case 'responses': return (channelStore as any).responsesChannelsData
    case 'gemini': return (channelStore as any).geminiChannelsData
    case 'images': return (channelStore as any).imagesChannelsData
    default: return channelStore.currentChannelsData
  }
}

async function fetchConversations() {
  try {
    const resp = await api.getConversations(kindFilter.value || undefined)
    conversations.value = resp.conversations || []
    overrides.value = resp.overrides || {}
  } catch (e) {
    console.error('[ConversationDashboard] fetch error:', e)
  } finally {
    loading.value = false
  }
}

function toggleExpand(id: string) {
  if (expandedCards.value.has(id)) {
    expandedCards.value.delete(id)
  } else {
    expandedCards.value.add(id)
  }
}

async function handleSetOverride(convId: string, sequence: ChannelSequenceEntry[]) {
  try {
    await api.setConversationOverride(convId, sequence)
    await fetchConversations()
  } catch (e) {
    console.error('[ConversationDashboard] set override error:', e)
  }
}

async function handleRemoveOverride(convId: string) {
  try {
    await api.removeConversationOverride(convId)
    await fetchConversations()
  } catch (e) {
    console.error('[ConversationDashboard] remove override error:', e)
  }
}

// Polling
const tick = useGlobalTick(3000, 'ConversationDashboard')
tick.onTick(() => fetchConversations())
fetchConversations()
</script>

<style scoped>
.conversation-dashboard {
  max-width: 1400px;
  margin: 0 auto;
}
</style>