<script setup lang="ts">
import { ref, reactive, computed, watch, onBeforeUnmount } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  AlertCircle,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  ClipboardPaste,
  Loader2,
  RotateCcw,
  Wand2,
  X,
} from 'lucide-vue-next'
import { useConsoleChannels } from '@/composables/useConsoleChannels'
import { useLanguage } from '@/composables/useLanguage'
import { buildChannelPayload } from '@/utils/channel-payload'
import { parseQuickInput } from '@/utils/quick-input-parser'
import type { Channel, DisabledKeyInfo } from '@/services/admin-api'

interface Props {
  channel?: Channel | null
  channelType: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved'): void
}>()

const { tf } = useLanguage()
const { saveChannel, restoreApiKey } = useConsoleChannels()

const isEditMode = computed(() => !!props.channel)
const saving = ref(false)
const restoringKey = ref('')
const error = ref('')
const showAdvanced = ref(false)
const showProtocolOptions = ref(false)
const quickInput = ref('')

const reasoningParamStyleOptions = [
  { label: 'reasoning.effort', value: 'reasoning' },
  { label: 'reasoning_effort', value: 'reasoning_effort' },
  { label: 'thinking (JD/GLM)', value: 'thinking' },
]

const textVerbosityOptions = [
  { label: 'Default', value: '' },
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
]

const form = reactive({
  name: '',
  description: '',
  serviceType: '' as 'openai' | 'claude' | 'gemini' | 'responses' | '',
  baseUrl: '',
  baseUrlsText: '',
  website: '',
  proxyUrl: '',
  requestTimeoutMs: '' as string | number,
  routePrefix: '',
  insecureSkipVerify: false,
  apiKeysText: '',
  customHeadersText: '{}',
  modelMappingText: '{}',
  reasoningMappingText: '{}',
  reasoningParamStyle: 'reasoning' as 'reasoning' | 'reasoning_effort' | 'thinking',
  textVerbosity: '' as 'low' | 'medium' | 'high' | '',
  supportedModelsText: '',
  noVisionModelsText: '',
  visionFallbackModel: '',
  noVision: false,
  passbackReasoningContent: false,
  passbackThinkingBlocks: false,
  fastMode: false,
  lowQuality: false,
  injectDummyThoughtSignature: false,
  stripThoughtSignature: false,
  stripEmptyTextBlocks: false,
  normalizeSystemRoleToTopLevel: false,
  normalizeMetadataUserId: true,
  normalizeNonstandardChatRoles: false,
  autoBlacklistBalance: true,
  codexNativeToolPassthrough: false,
  codexToolCompat: false,
  stripCodexClientTools: false,
})

const disabledApiKeys = computed<DisabledKeyInfo[]>(() => props.channel?.disabledApiKeys ?? [])
const historicalApiKeys = computed(() => props.channel?.historicalApiKeys ?? [])

const quickDetection = computed(() => parseQuickInput(quickInput.value, form.serviceType || undefined))
const detectedBaseUrls = computed(() => quickDetection.value.detectedBaseUrls)
const detectedApiKeys = computed(() => quickDetection.value.detectedApiKeys)
const quickHasDetections = computed(() => detectedBaseUrls.value.length > 0 || detectedApiKeys.value.length > 0 || !!quickDetection.value.detectedServiceType)

function resetForm() {
  form.name = ''
  form.description = ''
  form.serviceType = defaultServiceTypeForChannel()
  form.baseUrl = ''
  form.baseUrlsText = ''
  form.website = ''
  form.proxyUrl = ''
  form.requestTimeoutMs = ''
  form.routePrefix = ''
  form.insecureSkipVerify = false
  form.apiKeysText = ''
  form.customHeadersText = '{}'
  form.modelMappingText = '{}'
  form.reasoningMappingText = '{}'
  form.reasoningParamStyle = 'reasoning'
  form.textVerbosity = ''
  form.supportedModelsText = ''
  form.noVisionModelsText = ''
  form.visionFallbackModel = ''
  form.noVision = false
  form.passbackReasoningContent = false
  form.passbackThinkingBlocks = false
  form.fastMode = false
  form.lowQuality = false
  form.injectDummyThoughtSignature = false
  form.stripThoughtSignature = false
  form.stripEmptyTextBlocks = false
  form.normalizeSystemRoleToTopLevel = false
  form.normalizeMetadataUserId = true
  form.normalizeNonstandardChatRoles = false
  form.autoBlacklistBalance = true
  form.codexNativeToolPassthrough = false
  form.codexToolCompat = false
  form.stripCodexClientTools = false
  quickInput.value = ''
  error.value = ''
  showAdvanced.value = false
  showProtocolOptions.value = false
}

function defaultServiceTypeForChannel() {
  if (props.channelType === 'gemini') return 'gemini'
  if (props.channelType === 'responses') return 'responses'
  if (props.channelType === 'messages') return 'claude'
  return 'openai'
}

function populateFromChannel(ch: Channel) {
  form.name = ch.name || ''
  form.description = ch.description || ''
  form.serviceType = ch.serviceType || defaultServiceTypeForChannel()
  form.baseUrl = ch.baseUrl || ''
  form.baseUrlsText = (ch.baseUrls || []).join('\n')
  form.website = ch.website || ''
  form.proxyUrl = ch.proxyUrl || ''
  form.requestTimeoutMs = ch.requestTimeoutMs || ''
  form.routePrefix = ch.routePrefix || ''
  form.insecureSkipVerify = ch.insecureSkipVerify ?? false
  form.apiKeysText = (ch.apiKeys || []).join('\n')
  form.customHeadersText = stringifyJson(ch.customHeaders)
  form.modelMappingText = stringifyJson(ch.modelMapping)
  form.reasoningMappingText = stringifyJson(ch.reasoningMapping)
  form.reasoningParamStyle = ch.reasoningParamStyle || 'reasoning'
  form.textVerbosity = ch.textVerbosity || ''
  form.supportedModelsText = (ch.supportedModels || []).join('\n')
  form.noVisionModelsText = (ch.noVisionModels || []).join('\n')
  form.visionFallbackModel = ch.visionFallbackModel || ''
  form.noVision = ch.noVision ?? false
  form.passbackReasoningContent = ch.passbackReasoningContent ?? false
  form.passbackThinkingBlocks = ch.passbackThinkingBlocks ?? false
  form.fastMode = ch.fastMode ?? false
  form.lowQuality = ch.lowQuality ?? false
  form.injectDummyThoughtSignature = ch.injectDummyThoughtSignature ?? false
  form.stripThoughtSignature = ch.stripThoughtSignature ?? false
  form.stripEmptyTextBlocks = ch.stripEmptyTextBlocks ?? false
  form.normalizeSystemRoleToTopLevel = ch.normalizeSystemRoleToTopLevel ?? false
  form.normalizeMetadataUserId = ch.normalizeMetadataUserId ?? true
  form.normalizeNonstandardChatRoles = ch.normalizeNonstandardChatRoles ?? false
  form.autoBlacklistBalance = ch.autoBlacklistBalance ?? true
  form.codexNativeToolPassthrough = ch.codexNativeToolPassthrough ?? false
  form.codexToolCompat = ch.codexToolCompat ?? ch.stripCodexClientTools ?? false
  form.stripCodexClientTools = ch.stripCodexClientTools ?? ch.codexToolCompat ?? false
}

watch(() => props.channel, (ch) => {
  resetForm()
  if (ch) populateFromChannel(ch)
}, { immediate: true })

const errors = computed(() => {
  const errs: Record<string, string> = {}
  if (!form.name.trim()) errs.name = tf('console.form.nameRequired', '渠道名称必填')
  if (!form.serviceType) errs.serviceType = tf('console.form.serviceTypeRequired', '请选择服务类型')
  if (!form.baseUrl.trim() && !form.baseUrlsText.trim()) errs.baseUrl = tf('console.form.baseUrlRequired', '至少需要一个 Base URL')
  if (String(form.requestTimeoutMs).trim()) {
    const timeout = Number(form.requestTimeoutMs)
    if (!Number.isInteger(timeout) || timeout <= 0) {
      errs.requestTimeoutMs = tf('console.form.requestTimeoutInvalid', '请求超时必须是正整数毫秒')
    }
  }
  return errs
})

const isValid = computed(() => Object.keys(errors.value).length === 0)

function stringifyJson(value?: Record<string, unknown>) {
  if (!value || Object.keys(value).length === 0) return '{}'
  return JSON.stringify(value, null, 2)
}

function parseJsonObject<T extends Record<string, unknown>>(text: string, label: string): T {
  const trimmed = text.trim()
  if (!trimmed) return {} as T
  const parsed = JSON.parse(trimmed)
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error(`${label} must be a JSON object`)
  }
  return parsed as T
}

function parseLines(text: string) {
  return text
    .split('\n')
    .map(s => s.trim())
    .filter(Boolean)
}

function mergeLineText(currentText: string, additions: string[]) {
  const merged = [...parseLines(currentText)]
  for (const addition of additions.map(item => item.trim()).filter(Boolean)) {
    if (!merged.includes(addition)) merged.push(addition)
  }
  return merged.join('\n')
}

function applyQuickInput() {
  const result = quickDetection.value
  if (result.detectedServiceType) form.serviceType = result.detectedServiceType
  if (result.detectedBaseUrl) form.baseUrl = result.detectedBaseUrl
  if (result.detectedBaseUrls.length > 1) form.baseUrlsText = result.detectedBaseUrls.join('\n')
  if (result.detectedApiKeys.length) form.apiKeysText = mergeLineText(form.apiKeysText, result.detectedApiKeys)
  if (!form.name.trim() && result.detectedServiceType) {
    form.name = `${props.channelType}-${result.detectedServiceType}-channel`
  }
}

function handleQuickPaste(text: string) {
  const result = parseQuickInput(text, form.serviceType || undefined)
  if (result.detectedBaseUrl) form.baseUrl = result.detectedBaseUrl
  if (result.detectedBaseUrls.length > 1) form.baseUrlsText = result.detectedBaseUrls.join('\n')
  if (result.detectedApiKeys.length) form.apiKeysText = mergeLineText(form.apiKeysText, result.detectedApiKeys)
  if (result.detectedServiceType && !form.serviceType) form.serviceType = result.detectedServiceType
}

async function handleRestoreKey(key: string) {
  if (!props.channel) return
  restoringKey.value = key
  error.value = ''
  try {
    await restoreApiKey(props.channel.index, key)
    emit('saved')
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    restoringKey.value = ''
  }
}

async function handleSubmit() {
  if (!isValid.value) return
  saving.value = true
  error.value = ''

  try {
    const customHeaders = parseJsonObject<Record<string, string>>(form.customHeadersText, 'Custom headers')
    const modelMapping = parseJsonObject<Record<string, string>>(form.modelMappingText, 'Model mapping')
    const reasoningMapping = parseJsonObject<Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>>(
      form.reasoningMappingText,
      'Reasoning mapping',
    )

    const payload = buildChannelPayload({
      name: form.name,
      serviceType: form.serviceType,
      baseUrl: form.baseUrl,
      baseUrls: parseLines(form.baseUrlsText),
      website: form.website,
      insecureSkipVerify: form.insecureSkipVerify,
      lowQuality: form.lowQuality,
      injectDummyThoughtSignature: form.injectDummyThoughtSignature,
      stripThoughtSignature: form.stripThoughtSignature,
      passbackReasoningContent: form.passbackReasoningContent,
      passbackThinkingBlocks: form.passbackThinkingBlocks,
      description: form.description,
      apiKeys: parseLines(form.apiKeysText),
      modelMapping,
      reasoningMapping,
      reasoningParamStyle: form.reasoningParamStyle,
      textVerbosity: form.textVerbosity,
      fastMode: form.fastMode,
      customHeaders,
      proxyUrl: form.proxyUrl,
      requestTimeoutMs: form.requestTimeoutMs,
      routePrefix: form.routePrefix,
      supportedModels: parseLines(form.supportedModelsText),
      autoBlacklistBalance: form.autoBlacklistBalance,
      normalizeMetadataUserId: form.normalizeMetadataUserId,
      stripEmptyTextBlocks: form.stripEmptyTextBlocks,
      normalizeSystemRoleToTopLevel: form.normalizeSystemRoleToTopLevel,
      codexNativeToolPassthrough: form.codexNativeToolPassthrough,
      codexToolCompat: form.codexToolCompat,
      normalizeNonstandardChatRoles: form.normalizeNonstandardChatRoles,
      stripCodexClientTools: form.stripCodexClientTools,
      noVision: form.noVision,
      noVisionModels: parseLines(form.noVisionModelsText),
      visionFallbackModel: form.visionFallbackModel,
    })

    if (isEditMode.value && props.channel?.requestTimeoutMs && !String(form.requestTimeoutMs ?? '').trim()) {
      payload.requestTimeoutMs = 0
    }

    await saveChannel(payload, props.channel?.index ?? null)
    emit('saved')
    emit('close')
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}

// Keyboard shortcuts: Esc 取消，Enter 保存（与 Agent 配置卡片一致）
const handleGlobalKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    e.preventDefault()
    emit('close')
  } else if (e.key === 'Enter' && !e.shiftKey && !e.metaKey && !e.ctrlKey) {
    // 如果焦点在 textarea 内，Enter 换行不触发提交
    const tag = (e.target as HTMLElement)?.tagName
    if (tag === 'TEXTAREA') return
    e.preventDefault()
    void handleSubmit()
  }
}

watch(() => props.channel, (ch) => {
  if (ch) {
    window.addEventListener('keydown', handleGlobalKeydown)
  } else {
    window.removeEventListener('keydown', handleGlobalKeydown)
  }
}, { immediate: true })

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="true"
        class="fixed inset-0 z-50 flex items-center justify-center"
      >
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="emit('close')" />

        <div class="relative z-10 flex max-h-[90vh] w-[94vw] max-w-6xl flex-col border border-border bg-card shadow-2xl">
          <div class="flex shrink-0 items-start justify-between border-b border-border p-4">
            <div class="space-y-1">
              <div class="text-xs font-bold uppercase tracking-[0.18em] text-primary">
                {{ channelType }} CHANNEL
              </div>
              <h3 class="text-lg font-semibold">
                {{ isEditMode
                  ? tf('console.form.editChannel', '编辑渠道')
                  : tf('console.form.addChannel', '添加渠道')
                }}
              </h3>
              <p class="text-xs text-muted-foreground">
                {{ isEditMode
                  ? tf('addChannel.editSubtitle', '修改渠道配置信息')
                  : tf('addChannel.fullSubtitle', '配置渠道、密钥、模型与高级协议选项')
                }}
              </p>
            </div>
            <Button variant="ghost" size="icon-sm" @click="emit('close')">
              <X class="h-4 w-4" />
            </Button>
          </div>

          <ScrollArea class="min-h-0 flex-1">
            <form class="grid gap-5 p-4 lg:grid-cols-[1fr_1fr]" @submit.prevent="handleSubmit">
              <div v-if="error" class="lg:col-span-2 border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
                {{ error }}
              </div>

              <section v-if="!isEditMode" class="space-y-3 border border-primary/20 bg-primary/5 p-4 lg:col-span-2">
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <h4 class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-primary">
                      <ClipboardPaste class="h-3.5 w-3.5" />
                      {{ tf('addChannel.quickMode', '快速粘贴') }}
                    </h4>
                    <p class="mt-1 text-xs text-muted-foreground">
                      {{ tf('addChannel.quickHint', '粘贴 Base URL、API Key 或完整配置片段，自动识别并填入表单。') }}
                    </p>
                  </div>
                  <Button type="button" size="sm" :disabled="!quickHasDetections" @click="applyQuickInput">
                    <Wand2 class="h-3.5 w-3.5" />
                    {{ tf('addChannel.applyDetected', '应用识别结果') }}
                  </Button>
                </div>
                <Textarea
                  v-model="quickInput"
                  rows="4"
                  class="font-mono text-xs"
                  placeholder="https://api.example.com/v1&#10;sk-..."
                  @paste="handleQuickPaste(($event.clipboardData?.getData('text/plain') || ''))"
                />
                <div class="grid gap-2 md:grid-cols-3">
                  <div class="border border-border bg-background/70 p-2 text-xs">
                    <div class="mb-1 flex items-center gap-1.5 font-semibold">
                      <CheckCircle2 v-if="detectedBaseUrls.length" class="h-3.5 w-3.5 text-emerald-500" />
                      <AlertCircle v-else class="h-3.5 w-3.5 text-muted-foreground" />
                      Base URLs
                    </div>
                    <p class="truncate text-muted-foreground">
                      {{ detectedBaseUrls.length ? detectedBaseUrls.join(' · ') : tf('addChannel.noneDetected', '未识别') }}
                    </p>
                  </div>
                  <div class="border border-border bg-background/70 p-2 text-xs">
                    <div class="mb-1 flex items-center gap-1.5 font-semibold">
                      <CheckCircle2 v-if="detectedApiKeys.length" class="h-3.5 w-3.5 text-emerald-500" />
                      <AlertCircle v-else class="h-3.5 w-3.5 text-muted-foreground" />
                      API Keys
                    </div>
                    <p class="text-muted-foreground">
                      {{ detectedApiKeys.length ? `${detectedApiKeys.length} ${tf('console.keys.active', 'active keys')}` : tf('addChannel.noneDetected', '未识别') }}
                    </p>
                  </div>
                  <div class="border border-border bg-background/70 p-2 text-xs">
                    <div class="mb-1 flex items-center gap-1.5 font-semibold">
                      <CheckCircle2 v-if="quickDetection.detectedServiceType" class="h-3.5 w-3.5 text-emerald-500" />
                      <AlertCircle v-else class="h-3.5 w-3.5 text-muted-foreground" />
                      Service Type
                    </div>
                    <p class="text-muted-foreground">
                      {{ quickDetection.detectedServiceType || tf('addChannel.noneDetected', '未识别') }}
                    </p>
                  </div>
                </div>
              </section>

              <section class="space-y-3 border border-border bg-background/40 p-4">
                <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {{ tf('console.form.basicInfo', '基础信息') }}
                </h4>
                <div class="grid grid-cols-2 gap-3">
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.name', '名称') }} *</Label>
                    <Input v-model="form.name" :class="{ 'border-destructive': errors.name }" />
                    <p v-if="errors.name" class="text-[10px] text-destructive">{{ errors.name }}</p>
                  </div>
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.serviceType', '服务类型') }} *</Label>
                    <Select v-model="form.serviceType">
                      <SelectTrigger :class="{ 'border-destructive': errors.serviceType }">
                        <SelectValue :placeholder="tf('console.form.selectServiceType', '选择服务类型')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="claude">Claude</SelectItem>
                        <SelectItem value="openai">OpenAI</SelectItem>
                        <SelectItem value="gemini">Gemini</SelectItem>
                        <SelectItem value="responses">Responses</SelectItem>
                      </SelectContent>
                    </Select>
                    <p v-if="errors.serviceType" class="text-[10px] text-destructive">{{ errors.serviceType }}</p>
                  </div>
                </div>
                <div class="space-y-1.5">
                  <Label>{{ tf('console.form.description', '描述') }}</Label>
                  <Textarea v-model="form.description" rows="2" />
                </div>
                <div class="grid grid-cols-2 gap-3">
                  <div class="space-y-1.5">
                    <Label>Website</Label>
                    <Input v-model="form.website" placeholder="https://example.com" />
                  </div>
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.requestTimeoutMs', '请求超时（ms）') }}</Label>
                    <Input v-model="form.requestTimeoutMs" type="number" placeholder="60000" :class="{ 'border-destructive': errors.requestTimeoutMs }" />
                    <p v-if="errors.requestTimeoutMs" class="text-[10px] text-destructive">{{ errors.requestTimeoutMs }}</p>
                  </div>
                </div>
              </section>

              <section class="space-y-3 border border-border bg-background/40 p-4">
                <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {{ tf('console.form.connection', '连接') }}
                </h4>
                <div class="space-y-1.5">
                  <Label>{{ tf('console.form.baseUrl', 'Base URL') }} *</Label>
                  <Input
                    v-model="form.baseUrl"
                    placeholder="https://api.example.com"
                    :class="{ 'border-destructive': errors.baseUrl }"
                    @paste="handleQuickPaste(($event.clipboardData?.getData('text/plain') || ''))"
                  />
                  <p v-if="errors.baseUrl" class="text-[10px] text-destructive">{{ errors.baseUrl }}</p>
                </div>
                <div class="space-y-1.5">
                  <Label>{{ tf('console.form.additionalUrls', '额外 URL（每行一个）') }}</Label>
                  <Textarea v-model="form.baseUrlsText" rows="3" placeholder="https://backup.example.com" />
                </div>
                <div class="grid grid-cols-2 gap-3">
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.proxyUrl', '代理 URL') }}</Label>
                    <Input v-model="form.proxyUrl" placeholder="socks5://..." />
                  </div>
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.routePrefix', '路由前缀') }}</Label>
                    <Input v-model="form.routePrefix" placeholder="kimi" />
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <Switch v-model="form.insecureSkipVerify" />
                  <Label class="text-xs">{{ tf('console.form.insecureSkipVerify', '跳过 TLS 验证') }}</Label>
                </div>
              </section>

              <section class="space-y-3 border border-border bg-background/40 p-4">
                <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {{ tf('console.form.authentication', '认证') }}
                </h4>
                <div class="space-y-1.5">
                  <Label>{{ tf('console.form.apiKeys', 'API Keys（每行一个）') }}</Label>
                  <Textarea
                    v-model="form.apiKeysText"
                    rows="4"
                    placeholder="sk-xxx&#10;sk-yyy"
                    class="font-mono text-xs"
                    @paste="handleQuickPaste(($event.clipboardData?.getData('text/plain') || ''))"
                  />
                </div>
                <div v-if="disabledApiKeys.length" class="space-y-2 border border-amber-500/20 bg-amber-500/10 p-2">
                  <div class="text-[10px] font-bold uppercase tracking-wider text-amber-700 dark:text-amber-300">
                    {{ tf('console.form.disabledKeys', 'Disabled keys') }}
                  </div>
                  <div v-for="item in disabledApiKeys" :key="item.key" class="flex items-center justify-between gap-2 text-xs">
                    <span class="min-w-0 truncate font-mono" :title="item.message || item.reason">{{ item.key }}</span>
                    <Button size="sm" variant="outline" :disabled="restoringKey === item.key" @click="handleRestoreKey(item.key)">
                      <Loader2 v-if="restoringKey === item.key" class="h-3 w-3 animate-spin" />
                      <RotateCcw v-else class="h-3 w-3" />
                      {{ tf('console.form.restoreKey', 'Restore') }}
                    </Button>
                  </div>
                </div>
                <div v-if="historicalApiKeys.length" class="text-xs text-muted-foreground">
                  {{ historicalApiKeys.length }} {{ tf('console.form.historicalKeys', 'historical keys recorded') }}
                </div>
              </section>

              <section class="space-y-3 border border-border bg-background/40 p-4">
                <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {{ tf('console.form.models', '模型') }}
                </h4>
                <div class="space-y-1.5">
                  <Label>{{ tf('console.form.modelMapping', '模型映射（JSON）') }}</Label>
                  <Textarea v-model="form.modelMappingText" rows="4" class="font-mono text-xs" />
                </div>
                <div class="space-y-1.5">
                  <Label>Reasoning Mapping（JSON）</Label>
                  <Textarea v-model="form.reasoningMappingText" rows="3" class="font-mono text-xs" />
                </div>
                <div class="space-y-1.5">
                  <Label>{{ tf('console.form.supportedModels', '支持的模型（每行一个，留空=全部）') }}</Label>
                  <Textarea v-model="form.supportedModelsText" rows="3" placeholder="gpt-4*&#10;claude-3*" class="font-mono text-xs" />
                </div>
              </section>

              <section class="space-y-3 border border-border bg-background/40 p-4 lg:col-span-2">
                <button
                  type="button"
                  class="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground transition-colors hover:text-foreground"
                  @click="showProtocolOptions = !showProtocolOptions"
                >
                  <ChevronDown v-if="!showProtocolOptions" class="h-3.5 w-3.5" />
                  <ChevronUp v-else class="h-3.5 w-3.5" />
                  {{ tf('console.form.protocolOptions', '协议与模型高级选项') }}
                </button>
                <div v-if="showProtocolOptions" class="grid gap-4 lg:grid-cols-3">
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.reasoningParamStyle', 'Reasoning 参数风格') }}</Label>
                    <Select v-model="form.reasoningParamStyle">
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="item in reasoningParamStyleOptions" :key="item.value" :value="item.value">
                          {{ item.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.textVerbosity', 'Text verbosity') }}</Label>
                    <Select v-model="form.textVerbosity">
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="item in textVerbosityOptions" :key="item.value || 'default'" :value="item.value">
                          {{ item.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div class="space-y-1.5">
                    <Label>Vision fallback model</Label>
                    <Input v-model="form.visionFallbackModel" placeholder="mimo-v2.5" />
                  </div>
                  <div class="space-y-1.5 lg:col-span-3">
                    <Label>No vision models（每行一个）</Label>
                    <Textarea v-model="form.noVisionModelsText" rows="2" class="font-mono text-xs" />
                  </div>
                </div>
              </section>

              <section class="space-y-3 border border-border bg-background/40 p-4 lg:col-span-2">
                <button
                  type="button"
                  class="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground transition-colors hover:text-foreground"
                  @click="showAdvanced = !showAdvanced"
                >
                  <ChevronDown v-if="!showAdvanced" class="h-3.5 w-3.5" />
                  <ChevronUp v-else class="h-3.5 w-3.5" />
                  {{ tf('console.form.advancedFlags', '高级选项') }}
                </button>
                <div v-if="showAdvanced" class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                  <div
                    v-for="flag in [
                      { key: 'noVision', label: tf('console.form.noVision', '禁用视觉') },
                      { key: 'passbackReasoningContent', label: tf('console.form.passbackReasoning', '回传推理内容') },
                      { key: 'passbackThinkingBlocks', label: tf('console.form.passbackThinking', '回传思考块') },
                      { key: 'fastMode', label: tf('console.form.fastMode', '快速模式') },
                      { key: 'lowQuality', label: tf('console.form.lowQuality', '低质量标记') },
                      { key: 'injectDummyThoughtSignature', label: tf('console.form.injectDummySignature', '注入假思考签名') },
                      { key: 'stripThoughtSignature', label: tf('console.form.stripThoughtSignature', '移除思考签名') },
                      { key: 'stripEmptyTextBlocks', label: tf('console.form.stripEmptyBlocks', '移除空文本块') },
                      { key: 'normalizeSystemRoleToTopLevel', label: tf('console.form.normalizeSystem', '规范化系统角色') },
                      { key: 'normalizeMetadataUserId', label: tf('console.form.normalizeUserId', '规范化用户 ID') },
                      { key: 'normalizeNonstandardChatRoles', label: tf('console.form.normalizeChatRoles', '规范化 Chat 角色') },
                      { key: 'autoBlacklistBalance', label: tf('console.form.autoBlacklist', '自动黑名单余额异常 Key') },
                      { key: 'codexNativeToolPassthrough', label: tf('console.form.codexNativeTools', 'Codex 原生工具透传') },
                      { key: 'codexToolCompat', label: tf('console.form.codexCompat', 'Codex 工具兼容') },
                      { key: 'stripCodexClientTools', label: tf('console.form.stripCodexTools', '移除 Codex 客户端工具') },
                    ]"
                    :key="flag.key"
                    class="flex items-center gap-2"
                  >
                    <Switch :model-value="(form as any)[flag.key]" @update:model-value="(v: boolean) => (form as any)[flag.key] = v" />
                    <Label class="text-xs">{{ flag.label }}</Label>
                  </div>
                </div>
              </section>

              <section class="space-y-3 border border-border bg-background/40 p-4 lg:col-span-2">
                <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {{ tf('console.form.customHeaders', '自定义 Headers（JSON）') }}
                </h4>
                <Textarea v-model="form.customHeadersText" rows="4" class="font-mono text-xs" />
              </section>
            </form>
          </ScrollArea>

          <div class="flex shrink-0 items-center justify-end gap-2 border-t border-border bg-card p-4">
            <span class="mr-auto text-[10px] text-muted-foreground">
              <kbd class="rounded border border-border px-1 py-0.5 font-mono">Esc</kbd> {{ tf('console.form.cancel', '取消') }}
              <kbd class="ml-2 rounded border border-border px-1 py-0.5 font-mono">Enter</kbd> {{ isEditMode ? tf('console.form.save', '保存') : tf('console.form.create', '创建') }}
            </span>
            <Button variant="ghost" @click="emit('close')">
              {{ tf('common.cancel', '取消') }}
            </Button>
            <Button :disabled="!isValid || saving" @click="handleSubmit">
              <Loader2 v-if="saving" class="mr-2 h-4 w-4 animate-spin" />
              {{ isEditMode
                ? tf('console.form.save', '保存')
                : tf('console.form.create', '创建')
              }}
            </Button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
