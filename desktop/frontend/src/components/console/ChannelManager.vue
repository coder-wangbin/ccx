<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Alert } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { Plus, Search, Layers, Archive, Loader2, ShieldCheck, ShieldOff, Zap } from 'lucide-vue-next'
import { useConsoleChannels } from '@/composables/useConsoleChannels'
import { useAdminApi } from '@/composables/useAdminApi'
import { useStatus } from '@/composables/useStatus'
import { useLanguage } from '@/composables/useLanguage'
import ChannelCard from '@/components/console/ChannelCard.vue'
import ChannelEditDialog from '@/components/console/ChannelEditDialog.vue'
import ChannelLogsDialog from '@/components/console/ChannelLogsDialog.vue'
import CapabilityTestDialog from '@/components/console/CapabilityTestDialog.vue'
import CircuitBreakerDialog from '@/components/console/CircuitBreakerDialog.vue'
import GlobalStatsChart from '@/components/console/charts/GlobalStatsChart.vue'
import type { ManagedChannelType } from '@/utils/channel-type-api'
import type { Channel, ChannelMetrics, ChannelRecentActivity, GlobalStatsHistoryResponse } from '@/services/admin-api'

interface Props {
  type: ManagedChannelType
}

const props = defineProps<Props>()

const { t, tf } = useLanguage()
const { status } = useStatus()
const {
  activeTab,
  channelsByType,
  dashboardCache,
  refreshChannels,
  deleteChannel,
  setChannelStatus,
  resumeChannel,
  promoteChannel,
  reorderChannels,
} = useConsoleChannels()

watch(() => props.type, (newType) => {
  activeTab.value = newType
  void refreshChannels()
}, { immediate: true })

const channels = computed(() => channelsByType.value[props.type]?.channels || [])
const currentIndex = computed(() => channelsByType.value[props.type]?.current ?? -1)
const currentChannelName = computed(() => {
  const idx = currentIndex.value
  if (idx < 0) return ''
  return channels.value.find(ch => ch.index === idx)?.name || ''
})
const metrics = computed(() => dashboardCache.value[props.type]?.metrics || [])
const activity = computed(() => dashboardCache.value[props.type]?.recentActivity || [])
const stats = computed(() => dashboardCache.value[props.type]?.stats)

const searchQuery = ref('')
const normalizedSearch = computed(() => searchQuery.value.trim().toLowerCase())

const activeChannels = computed(() => {
  const list = channels.value.filter(ch => ch.status !== 'disabled')
  return sortChannels(list).filter(matchesSearch)
})

const inactiveChannels = computed(() => {
  const list = channels.value.filter(ch => ch.status === 'disabled')
  return sortChannels(list).filter(matchesSearch)
})

const activeCount = computed(() => channels.value.filter(ch => ch.status === 'active' || ch.status === undefined || ch.status === '').length)
const failoverCount = computed(() => channels.value.filter(ch => ch.status !== 'disabled').length)
const disabledCount = computed(() => channels.value.filter(ch => ch.status === 'disabled').length)

const metricsMap = computed(() => {
  const map = new Map<number, ChannelMetrics>()
  for (const m of metrics.value) {
    map.set(m.channelIndex, m)
  }
  return map
})

const activityMap = computed(() => {
  const map = new Map<number, ChannelRecentActivity>()
  for (const a of activity.value) {
    map.set(a.channelIndex, a)
  }
  return map
})

const hasLoaded = computed(() => Boolean(stats.value) || metrics.value.length > 0 || channels.value.length > 0)
const isInitialLoading = computed(() => !hasLoaded.value && !actionError.value)
const actionLoading = ref(false)
const actionError = ref('')
const isRefreshing = ref(false)
const showAddDialog = ref(false)
const editingChannel = ref<Channel | null>(null)
const logsChannel = ref<Channel | null>(null)
const capabilityChannel = ref<Channel | null>(null)
const showCapabilityDialog = ref(false)
const draggedIndex = ref<number | null>(null)

// Fuzzy 模式
const fuzzyEnabled = ref(false)
const fuzzyLoading = ref(false)
const fuzzyLoadError = ref(false)
const showCbDialog = ref(false)

// 用量统计
const statsData = ref<GlobalStatsHistoryResponse | null>(null)
const statsLoading = ref(false)
const statsDuration = ref('6h')

async function loadGlobalStats() {
  statsLoading.value = true
  try {
    const adminApi = useAdminApi()
    const typeMap: Record<ManagedChannelType, string> = {
      messages: 'messages',
      chat: 'chat',
      responses: 'responses',
      gemini: 'gemini',
      images: 'images'
    }
    const apiPath = typeMap[props.type]
    statsData.value = await adminApi.get<GlobalStatsHistoryResponse>(`/api/${apiPath}/global/stats/history?duration=${statsDuration.value}`)
  } catch {
    statsData.value = null
  } finally {
    statsLoading.value = false
  }
}

async function loadFuzzyMode() {
  fuzzyLoadError.value = false
  try {
    const adminApi = useAdminApi()
    const data = await adminApi.get<{ fuzzyModeEnabled: boolean }>('/api/settings/fuzzy-mode')
    fuzzyEnabled.value = data.fuzzyModeEnabled
  } catch {
    fuzzyLoadError.value = true
  }
}

async function toggleFuzzyMode() {
  if (fuzzyLoadError.value) {
    await loadFuzzyMode()
    return
  }
  fuzzyLoading.value = true
  try {
    const adminApi = useAdminApi()
    await adminApi.put('/api/settings/fuzzy-mode', { enabled: !fuzzyEnabled.value })
    fuzzyEnabled.value = !fuzzyEnabled.value
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  } finally {
    fuzzyLoading.value = false
  }
}

function clearActionError() {
  actionError.value = ''
}

function sortChannels(source: Channel[]) {
  return [...source].sort((a, b) => {
    const priorityDiff = (a.priority ?? a.index) - (b.priority ?? b.index)
    if (priorityDiff !== 0) return priorityDiff
    return a.index - b.index
  })
}

function matchesSearch(channel: Channel) {
  if (!normalizedSearch.value) return true
  const q = normalizedSearch.value
  return (
    channel.name?.toLowerCase().includes(q) ||
    channel.description?.toLowerCase().includes(q) ||
    channel.serviceType?.toLowerCase().includes(q) ||
    channel.baseUrl?.toLowerCase().includes(q) ||
    channel.baseUrls?.some(url => url.toLowerCase().includes(q))
  )
}

async function refreshCurrentChannels() {
  clearActionError()
  isRefreshing.value = true
  try {
    await refreshChannels()
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  } finally {
    isRefreshing.value = false
  }
}


function canDeleteChannel(channel: Channel) {
  const activeCount = activeChannels.value.filter(ch => ch.status === 'active' || ch.status === undefined || ch.status === '').length
  const isActive = channel.status === 'active' || channel.status === undefined || channel.status === ''
  return !(isActive && activeCount <= 1)
}

async function handleDelete(channel: Channel) {
  clearActionError()
  if (!canDeleteChannel(channel)) {
    actionError.value = tf('orchestration.deleteActiveGuard', '至少保留一个活跃渠道')
    return
  }

  actionLoading.value = true
  try {
    await deleteChannel(channel.index)
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  } finally {
    actionLoading.value = false
  }
}

async function handleStatusToggle(channelId: number, currentStatus: string) {
  clearActionError()
  const newStatus = currentStatus === 'active' || !currentStatus ? 'suspended' : 'active'
  try {
    await setChannelStatus(channelId, newStatus as 'active' | 'suspended')
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  }
}

async function handleDisable(channelId: number) {
  clearActionError()
  try {
    await setChannelStatus(channelId, 'disabled')
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  }
}

async function handleEnable(channelId: number) {
  clearActionError()
  try {
    await setChannelStatus(channelId, 'active')
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  }
}

async function handleResume(channelId: number) {
  clearActionError()
  try {
    await resumeChannel(channelId)
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  }
}

function isBreakerManagedChannel(channel: Channel) {
  const channelMetrics = metricsMap.value.get(channel.index)
  return channel.status === 'suspended' || channelMetrics?.circuitState === 'open'
}

async function handlePromote(channel: Channel, duration: number) {
  clearActionError()
  try {
    if (isBreakerManagedChannel(channel)) {
      await resumeChannel(channel.index)
    }
    await promoteChannel(channel.index, duration)
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  }
}

async function handleMoveTop(channelId: number) {
  const ordered = activeChannels.value.map(channel => channel.index)
  const index = ordered.indexOf(channelId)
  if (index <= 0) return
  ordered.splice(index, 1)
  ordered.unshift(channelId)
  await handleReorder(ordered)
}

async function handleMoveBottom(channelId: number) {
  const ordered = activeChannels.value.map(channel => channel.index)
  const index = ordered.indexOf(channelId)
  if (index < 0 || index >= ordered.length - 1) return
  ordered.splice(index, 1)
  ordered.push(channelId)
  await handleReorder(ordered)
}

function handleEdit(channel: Channel) {
  editingChannel.value = channel
  showAddDialog.value = true
}

function handleAdd() {
  editingChannel.value = null
  showAddDialog.value = true
}

function handleSaved() {
  showAddDialog.value = false
  refreshCurrentChannels()
}

function handleEditTestCapability(channel: Channel) {
  showAddDialog.value = false
  void refreshCurrentChannels()
  capabilityChannel.value = channel
  showCapabilityDialog.value = true
}

function handleLogs(channel: Channel) {
  logsChannel.value = channel
}

function handleCapability(channel: Channel) {
  capabilityChannel.value = channel
  showCapabilityDialog.value = true
}

function closeCapabilityDialog() {
  showCapabilityDialog.value = false
  capabilityChannel.value = null
}

async function handleReorder(newOrder: number[]) {
  clearActionError()
  try {
    await reorderChannels(newOrder)
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  }
}

function onDragStart(e: DragEvent, channelIndex: number) {
  draggedIndex.value = channelIndex
  e.dataTransfer?.setData('text/plain', String(channelIndex))
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
}

function onDrop(e: DragEvent, targetIndex: number) {
  e.preventDefault()
  const sourceIndex = draggedIndex.value
  if (sourceIndex === null || sourceIndex === targetIndex || normalizedSearch.value) return

  const currentChannels = [...activeChannels.value]
  const sourcePos = currentChannels.findIndex(c => c.index === sourceIndex)
  const targetPos = currentChannels.findIndex(c => c.index === targetIndex)
  if (sourcePos === -1 || targetPos === -1) return

  const [moved] = currentChannels.splice(sourcePos, 1)
  currentChannels.splice(targetPos, 0, moved)
  void handleReorder(currentChannels.map(c => c.index))
  draggedIndex.value = null
}

function onDragEnd() {
  draggedIndex.value = null
}

onMounted(() => {
  activeTab.value = props.type
})

// 服务状态变化时自动加载 Fuzzy 模式和统计数据
watch(() => status.value.running, (running) => {
  if (running) {
    loadFuzzyMode()
    loadGlobalStats()
  }
}, { immediate: true })

// 类型切换时重新加载统计
watch(() => props.type, () => {
  if (status.value.running) {
    loadGlobalStats()
  }
})
</script>

<template>
  <div class="flex min-h-full flex-col gap-4">
    <Alert v-if="actionError" variant="destructive" class="shrink-0">
      <p class="text-sm">{{ actionError }}</p>
      <template #action>
        <Button variant="ghost" size="sm" @click="clearActionError">
          {{ t('common.cancel') }}
        </Button>
      </template>
    </Alert>

    <div class="border border-border bg-card/75 p-3 shadow-sm dark:bg-card/55">
      <div class="flex flex-wrap items-center gap-3">
        <div class="relative min-w-[220px] flex-1 max-w-md">
          <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            v-model="searchQuery"
            :placeholder="tf('console.searchChannels', '搜索频道...')"
            class="pl-9"
          />
        </div>
        <!-- 批量测速按钮：桌面端暂不展示，放不下 -->
        <Button size="sm" @click="handleAdd">
          <Plus class="h-3.5 w-3.5" />
          {{ tf('console.addChannel', '添加频道') }}
        </Button>
        <div class="flex-1" />
        <Button
          size="sm"
          variant="outline"
          class="h-7 text-xs"
          :class="{ 'border-amber-500/40 text-amber-600 dark:text-amber-400': fuzzyEnabled }"
          :disabled="fuzzyLoading"
          :title="fuzzyLoadError ? t('console.fuzzyLoadFailed') : (fuzzyEnabled ? t('console.fuzzyEnabled') : t('console.fuzzyDisabled'))"
          @click="toggleFuzzyMode"
        >
          <ShieldCheck v-if="fuzzyEnabled" class="h-3 w-3 mr-1" />
          <ShieldOff v-else class="h-3 w-3 mr-1" />
          Fuzzy
          <Loader2 v-if="fuzzyLoading" class="h-3 w-3 ml-1 animate-spin" />
        </Button>
        <Button size="sm" variant="outline" class="h-7 text-xs" @click="showCbDialog = true">
          <Zap class="h-3 w-3 mr-1" />
          TB
        </Button>
      </div>

      <div class="mt-3 grid grid-cols-2 gap-2 md:grid-cols-4">
        <div class="border border-border bg-background/60 px-3 py-2">
          <div class="text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">Active</div>
          <div class="font-mono text-lg font-bold text-emerald-600 dark:text-emerald-300">{{ activeCount }}</div>
        </div>
        <div class="border border-border bg-background/60 px-3 py-2">
          <div class="text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">Failover</div>
          <div class="font-mono text-lg font-bold text-blue-600 dark:text-blue-300">{{ failoverCount }}</div>
        </div>
        <div class="border border-border bg-background/60 px-3 py-2">
          <div class="text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">Disabled</div>
          <div class="font-mono text-lg font-bold text-rose-600 dark:text-rose-300">{{ disabledCount }}</div>
        </div>
        <div class="border border-border bg-background/60 px-3 py-2">
          <div class="text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">Mode</div>
          <div class="truncate text-sm font-semibold text-foreground">
            {{ stats?.multiChannelMode ? tf('console.mode.multi', 'Multi-channel') : tf('console.mode.single', 'Single-channel') }}
          </div>
        </div>
      </div>

      <!-- 用量统计图表 -->
      <div v-if="statsData && !statsLoading" class="mt-3 border border-border bg-background/60">
        <GlobalStatsChart
          :data="statsData.dataPoints"
          :summary="statsData.summary"
          :model-data-points="statsData.modelDataPoints"
          :duration="statsDuration"
          compact
        />
      </div>
      <div v-else-if="statsLoading" class="mt-3 border border-border bg-background/60 p-4 flex items-center justify-center">
        <Loader2 class="h-4 w-4 animate-spin text-muted-foreground" />
      </div>
    </div>

    <div v-if="isInitialLoading" class="space-y-2">
      <Skeleton v-for="i in 5" :key="i" class="h-16 w-full rounded-none" />
    </div>

    <div v-else-if="activeChannels.length === 0 && inactiveChannels.length === 0" class="border border-dashed border-border bg-card/50 py-12 text-center">
      <p class="text-sm text-muted-foreground">
        {{ searchQuery
          ? tf('console.noSearchResults', '没有匹配的频道')
          : tf('console.noChannels', '暂无频道，点击上方按钮添加')
        }}
      </p>
    </div>

    <div v-else class="relative space-y-4" :class="{ 'pointer-events-none select-none': actionLoading }">
      <div v-if="actionLoading" class="absolute inset-0 z-10 flex items-center justify-center bg-background/60 backdrop-blur-[1px]">
        <div class="flex items-center gap-2 border border-border bg-card px-3 py-1.5 text-xs text-muted-foreground shadow-sm">
          <Loader2 class="h-3.5 w-3.5 animate-spin" />
          {{ tf('console.actions.deleting', '正在删除...') }}
        </div>
      </div>
      <section class="border border-border bg-card/50">
        <div class="flex items-center justify-between border-b border-border bg-secondary/40 px-3 py-2">
          <div class="flex items-center gap-2">
            <Layers class="h-4 w-4 text-primary" />
            <span class="text-xs font-bold uppercase tracking-[0.18em] text-foreground">
              {{ tf('console.pool.active', 'Failover Sequence') }}
            </span>
          </div>
          <span class="font-mono text-[11px] text-muted-foreground">
            {{ tf('console.pool.current', 'Current channel') }}{{ currentChannelName ? ` ${currentChannelName}` : '' }}{{ currentIndex >= 0 ? ` (#${currentIndex + 1})` : ' —' }}
          </span>
        </div>
        <div class="divide-y divide-border">
          <div
            v-for="(channel, index) in activeChannels"
            :key="channel.index"
            :draggable="!normalizedSearch"
            :class="{ 'opacity-50': draggedIndex === channel.index }"
            @dragstart="onDragStart($event, channel.index)"
            @dragover="onDragOver"
            @drop="onDrop($event, channel.index)"
            @dragend="onDragEnd"
          >
            <ChannelCard
              :channel="channel"
              :metrics="metricsMap.get(channel.index)"
              :activity="activityMap.get(channel.index)"
              :priority="index + 1"
              :supports-capability="type !== 'images'"
              :can-delete="canDeleteChannel(channel)"
              :can-move-top="index > 0 && !normalizedSearch"
              :can-move-bottom="index < activeChannels.length - 1 && !normalizedSearch"
              @edit="handleEdit(channel)"
              @delete="handleDelete(channel)"
              @status="handleStatusToggle(channel.index, channel.status || 'active')"
              @disable="handleDisable(channel.index)"
              @enable="handleEnable(channel.index)"
              @resume="handleResume(channel.index)"
              @promote="handlePromote(channel, 300)"
              @move-top="handleMoveTop(channel.index)"
              @move-bottom="handleMoveBottom(channel.index)"
              @logs="handleLogs(channel)"
              @capability="handleCapability(channel)"
            />
          </div>
        </div>
      </section>

      <section v-if="inactiveChannels.length" class="border border-dashed border-border bg-muted/20">
        <div class="flex items-center gap-2 border-b border-border px-3 py-2">
          <Archive class="h-4 w-4 text-muted-foreground" />
          <span class="text-xs font-bold uppercase tracking-[0.18em] text-muted-foreground">
            {{ tf('console.pool.inactive', 'Standby Pool') }}
          </span>
        </div>
        <div class="divide-y divide-border">
          <ChannelCard
            v-for="(channel, index) in inactiveChannels"
            :key="channel.index"
            :channel="channel"
            :metrics="metricsMap.get(channel.index)"
            :activity="activityMap.get(channel.index)"
            :priority="index + 1"
            :supports-capability="type !== 'images'"
            inactive
            :can-delete="canDeleteChannel(channel)"
            @edit="handleEdit(channel)"
            @delete="handleDelete(channel)"
            @status="handleStatusToggle(channel.index, channel.status || 'disabled')"
            @disable="handleDisable(channel.index)"
            @enable="handleEnable(channel.index)"
            @resume="handleResume(channel.index)"
            @promote="handlePromote(channel, 300)"
            @logs="handleLogs(channel)"
            @capability="handleCapability(channel)"
          />
        </div>
      </section>
    </div>

    <ChannelEditDialog
      v-if="showAddDialog"
      :channel="editingChannel"
      :channel-type="type"
      @close="showAddDialog = false"
      @saved="handleSaved"
      @test-capability="handleEditTestCapability"
    />

    <ChannelLogsDialog
      :open="!!logsChannel"
      :channel-type="type"
      :channel-id="logsChannel?.index ?? -1"
      :channel-name="logsChannel?.name ?? ''"
      @close="logsChannel = null"
    />

    <CapabilityTestDialog
      v-if="capabilityChannel"
      :key="`${type}-${capabilityChannel.index}`"
      :open="showCapabilityDialog"
      :channel-type="type"
      :channel-id="capabilityChannel.index"
      :channel-name="capabilityChannel.name ?? ''"
      @close="closeCapabilityDialog"
    />

    <CircuitBreakerDialog
      :open="showCbDialog"
      @close="showCbDialog = false"
    />
  </div>
</template>
