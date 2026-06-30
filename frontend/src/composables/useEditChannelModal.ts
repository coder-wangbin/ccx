import { ref, reactive, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useTheme } from 'vuetify'
import type { Channel } from '../services/api'
import { ApiService } from '../services/api'
import { supportsAdvancedChannelOptions, supportsReasoningMapping } from '../utils/channelAdvancedOptions'
import {
  buildChannelPayload,
  createModelCapabilityRow,
  modelCapabilitiesToRows,
  modelCapabilityRowsToRecord,
  normalizeSelectableString,
  resolveBuiltinUpstreamModelCapability,
  type ModelCapabilityRow,
} from '../utils/channelPayload'
import {
  resolveChannelWatcherAction,
  syncBaseUrlsFormState,
  filterValidSupportedModelPatterns,
  parseSupportedModelInput,
} from '../utils/add-channel-modal-state'
import { streamTimeoutPresets } from '../utils/streamTimeoutPresets'
import { useI18n } from '../i18n'
import { useChannelEditorFormDerived } from './useChannelEditorFormDerived'
import { useChannelEditorHeaderState } from './useChannelEditorHeaderState'
import { useDialogMenuWorkaround } from './useDialogMenuWorkaround'
import { useDisabledApiKeys } from './useDisabledApiKeys'
import { useEditChannelPresets } from './useEditChannelPresets'
import { useEditChannelSectionNav } from './useEditChannelSectionNav'
import { useTargetModelFetch } from './useTargetModelFetch'
import { useStreamTimeoutStrategy } from './useStreamTimeoutStrategy'
import { useSupportedModelFilters } from './useSupportedModelFilters'
import { useEditChannelOptions } from '../utils/editChannelOptions'
import { isValidUrl, normalizeModelCapabilities } from '../utils/editChannelHelpers'
import { createHandleTestCapability } from '../utils/editChannelPayload'

export interface EditChannelModalProps {
  show: boolean
  channel?: Channel | null
  channelType?: 'messages' | 'chat' | 'responses' | 'gemini' | 'images'
}

export type EditChannelModalEmits = {
  'update:show': [value: boolean]
  save: [channel: Omit<Channel, 'index' | 'latency' | 'status'>, options?: { isQuickAdd?: boolean; triggerCapabilityTest?: boolean }]
  testCapability: [channelId: number]
  error: [message: string]
  success: [message: string]
}

type EditChannelModalEmit = <K extends keyof EditChannelModalEmits>(event: K, ...args: EditChannelModalEmits[K]) => void
type ResolvedEditChannelModalProps = Readonly<EditChannelModalProps & { channelType: NonNullable<EditChannelModalProps['channelType']> }>

export function useEditChannelModal(props: ResolvedEditChannelModalProps, emit: EditChannelModalEmit) {
const { t } = useI18n()
  const apiService = new ApiService()
  
  // 主题
  const theme = useTheme()
  
  // 表单引用
  const formRef = ref()
  
  const defaultServiceTypeValueFallback = (): 'openai' | 'gemini' | 'claude' | 'responses' | 'copilot' => {
    if (props.channelType === 'chat') return 'openai'
    if (props.channelType === 'gemini') return 'gemini'
    if (props.channelType === 'responses') return 'responses'
    return 'claude'
  }

  const defaultNormalizeMetadataUserId = () => props.channelType !== 'responses'
  
  // 详细表单预期请求 URL 预览（防止输入时抖动）
  const formBaseUrlPreview = ref('')
  let formBaseUrlPreviewTimer: number | null = null
  
  const {
    activeSection,
    sections,
    scrollToSection,
    setSectionRef,
    attachScrollListener,
    detachScrollListener,
  } = useEditChannelSectionNav(t)

  const { isAnySelectMenuOpen, suppressDialogEscapeUntil, onMenuUpdate } = useDialogMenuWorkaround()

  const supportsOpenAIAdvancedOptions = computed(() => supportsAdvancedChannelOptions(form.serviceType))
  const supportsReasoningMappingOptions = computed(() => supportsReasoningMapping(form.serviceType))
  const supportsChatRoleNormalization = computed(() => {
    return props.channelType === 'chat' || (props.channelType === 'responses' && form.serviceType === 'openai')
  })
  
  // 模型优先级排序规则（索引越小优先级越高）
  // 表单数据：balanced 预设值作为渠道级默认回退值
  const defaultStreamTimeouts = { ...streamTimeoutPresets.balanced }
  
  const form = reactive({
    name: '',
    serviceType: '' as 'openai' | 'gemini' | 'claude' | 'responses' | 'copilot' | '',
    authHeader: 'auto' as 'auto' | 'bearer' | 'x-api-key' | '',
    baseUrl: '',
    baseUrls: [] as string[],
    website: '',
    insecureSkipVerify: false,
    lowQuality: false,
    injectDummyThoughtSignature: false,
    stripThoughtSignature: false,
    passbackReasoningContent: false,
    passbackThinkingBlocks: false,
    stripEmptyTextBlocks: false,
    normalizeSystemRoleToTopLevel: false,
    description: '',
    apiKeys: [] as string[],
    apiKeyConfigs: undefined as Channel['apiKeyConfigs'],
    modelMapping: {} as Record<string, string>,
    modelCapabilitiesText: '',
    modelCapabilityRows: [] as ModelCapabilityRow[],
    defaultContextWindowTokens: null as string | number | null,
    defaultMaxOutputTokens: null as string | number | null,
    allowUnknownContext: false,
    reasoningMapping: {} as Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>,
    reasoningParamStyle: 'reasoning' as 'reasoning' | 'reasoning_effort' | 'thinking',
    textVerbosity: '' as 'low' | 'medium' | 'high' | '',
    fastMode: false,
    customHeaders: {} as Record<string, string>,
    proxyUrl: '',
    requestTimeoutMs: null as string | number | null,
    responseHeaderTimeoutMs: null as string | number | null,
    streamFirstContentTimeoutEnabled: false,
    streamFirstContentTimeoutMs: defaultStreamTimeouts.firstContentMs as number,
    streamInactivityTimeoutEnabled: false,
    streamInactivityTimeoutMs: defaultStreamTimeouts.inactivityMs as number,
    streamToolCallIdleTimeoutEnabled: false,
    streamToolCallIdleTimeoutMs: defaultStreamTimeouts.toolCallIdleMs as number,
    rateLimitRpm: null as string | number | null,
    rateLimitWindowMinutes: null as string | number | null,
    rateLimitMaxConcurrent: null as string | number | null,
    rateLimitAutoFromHeaders: true,
    routePrefix: '',
    supportedModels: [] as string[],
    autoBlacklistBalance: true,
    normalizeMetadataUserId: defaultNormalizeMetadataUserId(),
    stripBillingHeader: false,
    codexNativeToolPassthrough: false,
    codexToolCompat: false,
    normalizeNonstandardChatRoles: false,
    stripCodexClientTools: false,
    stripImageGenerationTool: false,
    convertImageUrlToB64Json: false,
    noVision: false,
    noVisionModels: [] as string[],
    visionFallbackModel: '',
    visionFallbackReasoningEffort: '' as 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max' | '',
    historicalImageTurnLimit: 0,
  })

  const channelTypeRef = computed(() => props.channelType)
  const {
    serviceTypeOptions,
    sourceModelOptions,
    modelMappingHint,
    targetModelPlaceholder,
    reasoningEffortOptions,
    reasoningParamStyleOptions,
    textVerbosityOptions,
  } = useEditChannelOptions(channelTypeRef, form, t)
  
  // 多 BaseURL 文本输入（独立变量，保留用户输入的换行）
  const baseUrlsText = ref('')
  
  // 监听 baseUrlsText 变化，同步到 form（去重等效 URL）
  watch(baseUrlsText, val => {
    const { baseUrl, baseUrls } = syncBaseUrlsFormState(val, form.serviceType)
    form.baseUrl = baseUrl
    form.baseUrls = baseUrls
  })
  
  watch(() => form.serviceType, () => {
    const { baseUrl, baseUrls } = syncBaseUrlsFormState(baseUrlsText.value, form.serviceType)
    form.baseUrl = baseUrl
    form.baseUrls = baseUrls
  })
  
  // 模型映射行数据结构（改用数组存储，支持直接编辑）
  interface ModelMappingRow {
    id: number
    source: string
    target: string
    reasoning: '' | 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'
    noVision: boolean
  }
  
  let rowIdCounter = 0
  const modelMappingRows = ref<ModelMappingRow[]>([])
  let capabilityRowIdCounter = 0
  const nextCapabilityRowId = () => ++capabilityRowIdCounter
  
  const incompleteMappedTargetSuffix = /[._:/-]$/
  const isCompleteMappedTargetModel = (model: string) => !!model && !incompleteMappedTargetSuffix.test(model)
  const hasNoVisionRows = computed(() => modelMappingRows.value.some(row => row.noVision && row.target.trim()))
  const mappedTargetModels = computed(() => {
    const seen = new Set<string>()
    const models = [
      ...modelMappingRows.value.map(row => normalizeSelectableString(row.target).trim()),
      normalizeSelectableString(form.visionFallbackModel).trim(),
    ]
  
    return models.filter(model => {
      const key = model.toLowerCase()
      if (!isCompleteMappedTargetModel(model) || seen.has(key)) return false
      seen.add(key)
      return true
    })
  })
  const isMappingTargetEditing = ref(false)
  const hasPendingModelCapabilitySync = ref(false)
  
  function resetTransientUiState() {
    sourceMappingError.value = ''
    resetRestoredKeys()
    errors.name = ''
    errors.serviceType = ''
    errors.baseUrl = ''
    errors.website = ''
    formBaseUrlPreview.value = ''
  }
  
  // 源模型名验证错误
  const sourceMappingError = ref('')
  
  // 表单验证错误
  const errors = reactive({
    name: '',
    serviceType: '',
    baseUrl: '',
    website: ''
  })
  
  // 验证规则
  const rules = {
    required: (value: string) => !!value || t('addChannel.fieldRequired'),
    url: (value: string) => {
      try {
        new URL(value)
        return true
      } catch {
        return t('addChannel.invalidUrl')
      }
    },
    urlOptional: (value: string) => {
      if (!value) return true
      try {
        new URL(value)
        return true
      } catch {
        return t('addChannel.invalidUrl')
      }
    },
    baseUrls: (value: string) => {
      if (!value) return t('addChannel.fieldRequired')
      const urls = value
        .split('\n')
        .map(s => s.trim())
        .filter(Boolean)
      if (urls.length === 0) return t('addChannel.atLeastOneUrl')
      for (const url of urls) {
        try {
          new URL(url)
        } catch {
          return t('addChannel.invalidUrlValue', { url })
        }
      }
      return true
    },
    requestTimeoutMs: (value: string | number | null) => {
      if (value === null || value === undefined || value === '') return true
      const timeout = Number(value)
      return (Number.isInteger(timeout) && timeout >= 1000 && timeout <= 300000) || t('addChannel.requestTimeoutMsInvalid')
    },
    responseHeaderTimeoutMs: (value: string | number | null) => {
      if (value === null || value === undefined || value === '') return true
      const timeout = Number(value)
      return (Number.isInteger(timeout) && timeout >= 1000 && timeout <= 300000) || t('addChannel.responseHeaderTimeoutMsInvalid')
    }
  }
  
  // 计算属性
  const dialogMode = ref<'create' | 'edit'>('create')
  const isEditing = computed(() => dialogMode.value === 'edit')
  const isMac = computed(() => typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform))
  const hasDisabledKeysAvailable = computed(() => visibleDisabledKeys.value.length > 0)
  const hasConfigurableKeys = computed(() => form.apiKeys.length > 0 || (isEditing.value && hasDisabledKeysAvailable.value))
  
  const { selectedStreamTimeoutStrategy, applyStreamTimeoutStrategy } = useStreamTimeoutStrategy(form)

  const {
    commonSupportedModelFilters,
    selectedSupportedModelSet,
    supportedModelsError,
    handleSupportedModelsChange,
    appendSupportedModelFilter,
  } = useSupportedModelFilters(form, t)

  const modelCapabilitiesError = computed(() => {
    return modelCapabilityRowsToRecord(form.modelCapabilityRows) === null
      ? t('addChannel.modelCapabilitiesRowsInvalid')
      : ''
  })
  
  const syncModelCapabilitiesFromMapping = () => {
    const existingModels = new Set(
      form.modelCapabilityRows
        .map(row => normalizeSelectableString(row.model).trim().toLowerCase())
        .filter(Boolean)
    )
    const rowsToAdd = mappedTargetModels.value
      .filter(isCompleteMappedTargetModel)
      .filter(model => !existingModels.has(model.toLowerCase()))
      .map(model => {
        const builtin = resolveBuiltinUpstreamModelCapability(model)
        return createModelCapabilityRow(
          nextCapabilityRowId(),
          model,
          builtin?.capability,
          builtin ? 'builtin' : 'custom',
          builtin?.pattern || '',
        )
      })
    if (!rowsToAdd.length) return
    form.modelCapabilityRows = [...form.modelCapabilityRows, ...rowsToAdd]
  }
  
  const syncModelCapabilitiesFromMappingWhenIdle = () => {
    if (isMappingTargetEditing.value) {
      hasPendingModelCapabilitySync.value = true
      return
    }
    hasPendingModelCapabilitySync.value = false
    syncModelCapabilitiesFromMapping()
  }
  
  const startMappingTargetEdit = () => {
    isMappingTargetEditing.value = true
  }
  
  const finishMappingTargetEdit = () => {
    if (!isMappingTargetEditing.value) return
    isMappingTargetEditing.value = false
    if (!hasPendingModelCapabilitySync.value) return
    hasPendingModelCapabilitySync.value = false
    nextTick(syncModelCapabilitiesFromMapping)
  }
  
  const { headerClasses, avatarColor, headerIconStyle, subtitleClasses } = useChannelEditorHeaderState(theme)

  const isFormValid = computed(() => {
    return (
      form.name.trim() && form.serviceType && form.baseUrl.trim() && isValidUrl(form.baseUrl) && hasConfigurableKeys.value && !modelCapabilitiesError.value
    )
  })
  
  const buildSubmitPayload = () => {
    const payload = buildChannelPayload(form)
    applyVisionFallbackReasoning(payload)
    if (!form.streamFirstContentTimeoutEnabled) {
      delete payload.streamFirstContentTimeoutMs
      if (isEditing.value && props.channel?.streamFirstContentTimeoutMs) {
        payload.streamFirstContentTimeoutMs = 0
      }
    }
    if (!form.streamInactivityTimeoutEnabled) {
      delete payload.streamInactivityTimeoutMs
      if (isEditing.value && props.channel?.streamInactivityTimeoutMs) {
        payload.streamInactivityTimeoutMs = 0
      }
    }
    if (!form.streamToolCallIdleTimeoutEnabled) {
      delete payload.streamToolCallIdleTimeoutMs
      if (isEditing.value && props.channel?.streamToolCallIdleTimeoutMs) {
        payload.streamToolCallIdleTimeoutMs = 0
      }
    }
    if (isEditing.value && props.channel?.requestTimeoutMs && !payload.requestTimeoutMs) {
      payload.requestTimeoutMs = 0
    }
    if (isEditing.value && props.channel?.responseHeaderTimeoutMs && !payload.responseHeaderTimeoutMs) {
      payload.responseHeaderTimeoutMs = 0
    }
    if (isEditing.value && props.channel?.rateLimitRpm && !payload.rateLimitRpm) {
      payload.rateLimitRpm = 0
    }
    if (isEditing.value && props.channel?.rateLimitWindowMinutes && !payload.rateLimitWindowMinutes) {
      payload.rateLimitWindowMinutes = 0
    }
    if (isEditing.value && props.channel?.rateLimitMaxConcurrent && !payload.rateLimitMaxConcurrent) {
      payload.rateLimitMaxConcurrent = 0
    }
    return payload
  }
  
  const applyVisionFallbackReasoning = (payload: Partial<Channel>) => {
    const fallbackModel = normalizeSelectableString(form.visionFallbackModel).trim()
    if (!supportsReasoningMappingOptions.value || !fallbackModel) {
      return
    }
  
    const reasoningMapping = { ...(payload.reasoningMapping || {}) }
    if (form.visionFallbackReasoningEffort) {
      reasoningMapping[fallbackModel] = form.visionFallbackReasoningEffort
    } else if (!modelMappingRows.value.some(row => row.source === fallbackModel && row.reasoning)) {
      delete reasoningMapping[fallbackModel]
    }
    payload.reasoningMapping = reasoningMapping
  }
  
  // 表单操作
  const resetForm = () => {
    resetTransientUiState()
    form.name = ''
    form.serviceType = props.channelType === 'images' ? 'openai' : ''
    form.authHeader = 'auto'
    form.baseUrl = ''
    form.baseUrls = []
    form.website = ''
    form.insecureSkipVerify = false
    form.lowQuality = false
    form.injectDummyThoughtSignature = false
    form.stripThoughtSignature = false
    form.passbackReasoningContent = false
    form.passbackThinkingBlocks = false
    form.stripEmptyTextBlocks = false
    form.normalizeSystemRoleToTopLevel = false
    form.description = ''
    form.apiKeys = []
    form.apiKeyConfigs = undefined
    form.modelMapping = {}
    form.modelCapabilitiesText = ''
    form.modelCapabilityRows = []
    form.defaultContextWindowTokens = null
    form.defaultMaxOutputTokens = null
    form.allowUnknownContext = false
    form.reasoningMapping = {}
  
    // 清空模型映射行
    modelMappingRows.value = []
  
    form.reasoningParamStyle = 'reasoning'
    form.textVerbosity = ''
    form.fastMode = false
    form.customHeaders = {}
    form.proxyUrl = ''
    form.requestTimeoutMs = null
    form.responseHeaderTimeoutMs = null
    form.streamFirstContentTimeoutEnabled = false
    form.streamFirstContentTimeoutMs = defaultStreamTimeouts.firstContentMs
    form.streamInactivityTimeoutEnabled = false
    form.streamInactivityTimeoutMs = defaultStreamTimeouts.inactivityMs
    form.streamToolCallIdleTimeoutEnabled = false
    form.streamToolCallIdleTimeoutMs = defaultStreamTimeouts.toolCallIdleMs
    form.rateLimitRpm = null
    form.rateLimitWindowMinutes = null
    form.rateLimitMaxConcurrent = null
    form.rateLimitAutoFromHeaders = true
    form.routePrefix = ''
    form.supportedModels = []
    supportedModelsError.value = ''
    form.autoBlacklistBalance = true
    form.normalizeMetadataUserId = defaultNormalizeMetadataUserId()
    form.stripBillingHeader = false
    form.codexNativeToolPassthrough = false
    form.codexToolCompat = false
    form.normalizeNonstandardChatRoles = false
    form.stripCodexClientTools = false
    form.stripImageGenerationTool = false
    form.convertImageUrlToB64Json = false
    form.noVision = false
    form.noVisionModels = []
    form.visionFallbackModel = ''
    form.visionFallbackReasoningEffort = ''
    form.historicalImageTurnLimit = 0
  
    // 重置 baseUrlsText
    baseUrlsText.value = ''
  
    // 清空模型缓存和状态
    resetTargetModelOptions()
    fetchingModels.value = false
    fetchModelsError.value = ''
    keyModelsStatus.value.clear()
  
    }
  
  const loadChannelData = (channel: Channel) => {
    resetTransientUiState()
    form.name = channel.name
    form.serviceType = props.channelType === 'images' ? 'openai' : channel.serviceType
    form.authHeader = channel.authHeader || 'auto'
    form.baseUrl = channel.baseUrl
    form.baseUrls = channel.baseUrls || []
    form.website = channel.website || ''
    form.insecureSkipVerify = !!channel.insecureSkipVerify
    form.lowQuality = !!channel.lowQuality
    form.injectDummyThoughtSignature = !!channel.injectDummyThoughtSignature
    form.stripThoughtSignature = !!channel.stripThoughtSignature
    form.passbackReasoningContent = !!channel.passbackReasoningContent
    form.passbackThinkingBlocks = !!channel.passbackThinkingBlocks
    form.stripEmptyTextBlocks = !!channel.stripEmptyTextBlocks
    form.normalizeSystemRoleToTopLevel = !!channel.normalizeSystemRoleToTopLevel
    form.description = channel.description || ''
  
    // 同步 baseUrlsText（优先使用 baseUrls，否则使用 baseUrl），保留用户显式配置的原始 URL 形式
    const rawUrls = channel.baseUrls && channel.baseUrls.length > 0
      ? channel.baseUrls
      : (channel.baseUrl ? [channel.baseUrl] : [])
    baseUrlsText.value = rawUrls.join('\n')
  
    // 直接存储原始密钥，不需要映射关系
    form.apiKeys = [...channel.apiKeys]
    form.apiKeyConfigs = channel.apiKeyConfigs
      ? channel.apiKeyConfigs.map(cfg => ({
          ...cfg,
          models: cfg.models ? [...cfg.models] : undefined,
        }))
      : undefined
  
    form.modelMapping = { ...(channel.modelMapping || {}) }
    form.modelCapabilitiesText = Object.keys(channel.modelCapabilities || {}).length > 0
      ? JSON.stringify(normalizeModelCapabilities(channel.modelCapabilities), null, 2)
      : ''
    form.modelCapabilityRows = modelCapabilitiesToRows(channel.modelCapabilities || {}, nextCapabilityRowId)
    form.defaultContextWindowTokens = channel.defaultCapability?.contextWindowTokens || null
    form.defaultMaxOutputTokens = channel.defaultCapability?.maxOutputTokens || null
    form.allowUnknownContext = !!channel.allowUnknownContext
    form.reasoningMapping = { ...(channel.reasoningMapping || {}) }
  
    // 加载模型映射行
    loadModelMappingRows(channel)
  
    form.reasoningParamStyle = channel.reasoningParamStyle || 'reasoning'
    form.textVerbosity = channel.textVerbosity || ''
    form.fastMode = !!channel.fastMode
    form.customHeaders = { ...(channel.customHeaders || {}) }
    form.proxyUrl = channel.proxyUrl || ''
    form.requestTimeoutMs = channel.requestTimeoutMs || null
    form.responseHeaderTimeoutMs = channel.responseHeaderTimeoutMs || null
    form.streamFirstContentTimeoutEnabled = !!(channel.streamFirstContentTimeoutMs && channel.streamFirstContentTimeoutMs > 0)
    form.streamFirstContentTimeoutMs = channel.streamFirstContentTimeoutMs && channel.streamFirstContentTimeoutMs > 0 ? channel.streamFirstContentTimeoutMs : defaultStreamTimeouts.firstContentMs
    form.streamInactivityTimeoutEnabled = !!(channel.streamInactivityTimeoutMs && channel.streamInactivityTimeoutMs > 0)
    form.streamInactivityTimeoutMs = channel.streamInactivityTimeoutMs && channel.streamInactivityTimeoutMs > 0 ? channel.streamInactivityTimeoutMs : defaultStreamTimeouts.inactivityMs
    form.streamToolCallIdleTimeoutEnabled = !!(channel.streamToolCallIdleTimeoutMs && channel.streamToolCallIdleTimeoutMs >= 30000)
    form.streamToolCallIdleTimeoutMs = channel.streamToolCallIdleTimeoutMs && channel.streamToolCallIdleTimeoutMs >= 30000 ? channel.streamToolCallIdleTimeoutMs : defaultStreamTimeouts.toolCallIdleMs
    form.rateLimitRpm = (channel.rateLimitRpm && channel.rateLimitRpm > 0) ? channel.rateLimitRpm : null
    form.rateLimitWindowMinutes = (channel.rateLimitWindowMinutes && channel.rateLimitWindowMinutes > 0) ? channel.rateLimitWindowMinutes : null
    form.rateLimitMaxConcurrent = (channel.rateLimitMaxConcurrent && channel.rateLimitMaxConcurrent > 0) ? channel.rateLimitMaxConcurrent : null
    form.rateLimitAutoFromHeaders = channel.rateLimitAutoFromHeaders !== false
    form.routePrefix = channel.routePrefix || ''
    const { validPatterns, hasInvalidPatterns } = filterValidSupportedModelPatterns(channel.supportedModels || [])
    form.supportedModels = validPatterns
    supportedModelsError.value = hasInvalidPatterns ? t('addChannel.supportedModelsInvalidPattern') : ''
    form.autoBlacklistBalance = channel.autoBlacklistBalance ?? true
    form.normalizeMetadataUserId = channel.normalizeMetadataUserId ?? true
    form.stripBillingHeader = channel.stripBillingHeader ?? false
    form.codexNativeToolPassthrough = !!channel.codexNativeToolPassthrough
    form.codexToolCompat = channel.codexToolCompat ?? channel.stripCodexClientTools ?? false
    form.normalizeNonstandardChatRoles = !!channel.normalizeNonstandardChatRoles
    form.stripCodexClientTools = channel.codexToolCompat ?? channel.stripCodexClientTools ?? false
    form.stripImageGenerationTool = !!channel.stripImageGenerationTool
    form.convertImageUrlToB64Json = !!channel.convertImageUrlToB64Json
    form.noVision = !!channel.noVision
    form.noVisionModels = [...(channel.noVisionModels || [])]
    form.visionFallbackModel = channel.visionFallbackModel || ''
    form.visionFallbackReasoningEffort = (channel.reasoningMapping?.[form.visionFallbackModel] || '') as 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max' | ''
    form.historicalImageTurnLimit = channel.historicalImageTurnLimit ?? 0
  
    // 立即同步 baseUrl 到预览变量，避免等待 debounce
    formBaseUrlPreview.value = channel.baseUrl
  
    // 清空模型缓存和状态（切换渠道时重置）
    resetTargetModelOptions()
    fetchingModels.value = false
    fetchModelsError.value = ''
    keyModelsStatus.value.clear()
  
    // 如果有模型映射配置，主动预加载模型列表
    if (channel.modelMapping && Object.keys(channel.modelMapping).length > 0) {
      nextTick(() => {
        fetchTargetModels()
      })
    }
  }
  
  const {
    restoringKey,
    disabledKeys,
    visibleDisabledKeys,
    resetRestoredKeys,
    restoreDisabledKey,
  } = useDisabledApiKeys({
    apiService,
    channel: computed(() => props.channel),
    channelType: channelTypeRef,
    emitError: message => emit('error', message),
    form,
  })

  // 提交状态
  const submitting = ref(false)

  const {
    targetModelOptions,
    resetTargetModelOptions,
    fetchingModels,
    fetchModelsError,
    keyModelsStatus,
    ensureTargetModelsLoaded,
    fetchTargetModels,
  } = useTargetModelFetch({
    apiService,
    channel: computed(() => props.channel),
    channelType: channelTypeRef,
    defaultServiceType: defaultServiceTypeValueFallback,
    form,
    isEditing,
    t,
    visibleDisabledKeys,
  })
  
  const {
    baseUrlHasError,
    expectedRequestUrls,
    customHeadersArray,
    updateCustomHeaders,
  } = useChannelEditorFormDerived(channelTypeRef, form, baseUrlsText)

  // 将 modelMappingRows 转换为 form.modelMapping 对象（保存时使用）
  const syncModelMappingToForm = () => {
    form.modelMapping = {}
    form.reasoningMapping = {}
    form.noVisionModels = []
    const noVisionModels = new Set<string>()
  
    modelMappingRows.value.forEach(row => {
      if (row.source && row.target) {
        form.modelMapping[row.source] = row.target
        if (row.reasoning) {
          form.reasoningMapping[row.source] = row.reasoning
        }
        if (row.noVision) {
          noVisionModels.add(row.target)
        }
      }
    })
  
    form.noVisionModels = [...noVisionModels]
  }
  
  // 从渠道数据初始化 modelMappingRows
  const loadModelMappingRows = (channel: Channel) => {
    const mapping = channel.modelMapping || {}
    const reasoning = channel.reasoningMapping || {}
    const noVisionSet = new Set(channel.noVisionModels || [])
  
    modelMappingRows.value = Object.entries(mapping).map(([source, target]) => ({
      id: ++rowIdCounter,
      source,
      target,
      reasoning: (reasoning[source] || '') as ModelMappingRow['reasoning'],
      noVision: noVisionSet.has(target)
    }))
  }
  
  const syncModelMappingRowsFromForm = () => {
    const noVisionSet = new Set(form.noVisionModels || [])
  
    modelMappingRows.value = Object.entries(form.modelMapping || {}).map(([source, target]) => ({
      id: ++rowIdCounter,
      source,
      target,
      reasoning: (form.reasoningMapping[source] || '') as ModelMappingRow['reasoning'],
      noVision: noVisionSet.has(target)
    }))
  }

  const {
    showModelMappingPresets,
    showMessagesOpenAIChannelPresets,
    showClaudeChannelPresets,
    showCodexResponsesChannelPresets,
    applyPreset,
  } = useEditChannelPresets({
    channelType: channelTypeRef,
    form,
    supportsOpenAIAdvancedOptions,
    syncModelMappingRowsFromForm,
  })
  
  // 辅助函数：更新表单字段
  const updateForm = (partial: Record<string, any>) => {
    Object.assign(form, partial)
  }
  
  // 辅助函数：同步上游模型
  const syncUpstreamModels = () => {
    fetchTargetModels()
  }
  
  const handleSubmit = async () => {
    if (!formRef.value) return
  
    syncModelCapabilitiesFromMapping()
  
    const { valid } = await formRef.value.validate()
    if (!valid) return
    if (modelCapabilitiesError.value) return
  
    // 将模型映射行同步到 form 对象
    syncModelMappingToForm()
  
    const channelData = buildSubmitPayload()
  
    emit('save', channelData)
  }
  
  const handleCancel = () => {
    emit('update:show', false)
    resetForm()
  }
  
  const handleTestCapability = createHandleTestCapability({
    buildSubmitPayload,
    channel: computed(() => props.channel),
    emitSave: (channelData, options) => emit('save', channelData, options),
    emitTestCapability: channelId => emit('testCapability', channelId),
    formRef,
    modelCapabilitiesError,
    syncModelCapabilitiesFromMapping,
    syncModelMappingToForm,
  })

  const diagnosingCompat = ref(false)
  
  const handleDiagnoseCompat = async () => {
    if (props.channel?.index === undefined || props.channel?.index === null) return
    if (props.channelType === 'images') return
  
    diagnosingCompat.value = true
    try {
      const type = props.channelType as 'messages' | 'chat' | 'responses' | 'gemini'
      const result = await apiService.diagnoseChannelCompat(type, props.channel.index)
      const applied: string[] = []
      for (const [key, val] of Object.entries(result.recommendations)) {
        if (val !== undefined && (form as Record<string, unknown>)[key] !== val) {
          updateForm({ [key]: val })
          applied.push(key)
        }
      }
      if (result.urlRecommendations?.recommended) {
        const current = result.urlRecommendations.current
        const recommended = result.urlRecommendations.recommended
        const lines = baseUrlsText.value.split('\n').map(line => line.trim()).filter(Boolean)
        const nextLines = lines.length > 0
          ? lines.map((line, index) => (index === 0 || line === current) ? recommended : line)
          : [recommended]
        baseUrlsText.value = Array.from(new Set(nextLines)).join('\n')
        applied.push('baseUrl')
      }
      emit('success', applied.length
        ? t('channelEditor.compat.diagnoseApplied', { count: applied.length })
        : t('channelEditor.compat.diagnoseNoChange'))
    } catch (e) {
      emit('error', e instanceof Error ? e.message : t('channelEditor.compat.diagnoseFailed'))
    } finally {
      diagnosingCompat.value = false
    }
  }
  
  // 监听props变化
  watch(
    () => props.show,
    newShow => {
      if (newShow) {
        dialogMode.value = props.channel ? 'edit' : 'create'
        resetRestoredKeys()
  
        if (dialogMode.value === 'edit' && props.channel) {
          // 编辑模式：使用完整表单
          loadChannelData(props.channel)
        } else {
          // 添加模式：固定使用快速添加
          resetForm()
        }
  
        // dialog 渲染完成后绑定滚动监听，同步左侧导航高亮
        nextTick(() => attachScrollListener())
      } else {
        detachScrollListener()
      }
    }
  )
  
  watch(
    () => props.channel,
    (newChannel, oldChannel) => {
      const action = resolveChannelWatcherAction({
        show: props.show,
        newChannel,
        oldChannel,
      })
  
      if (action === 'load-edit-channel' && newChannel) {
        dialogMode.value = 'edit'
        loadChannelData(newChannel)
        return
      }
  
      if (action === 'reset-new-form') {
        dialogMode.value = 'create'
        resetForm()
      }
    }
  )
  
  watch(
    () => form.baseUrl,
    value => {
      if (formBaseUrlPreviewTimer !== null) {
        window.clearTimeout(formBaseUrlPreviewTimer)
      }
      formBaseUrlPreviewTimer = window.setTimeout(() => {
        formBaseUrlPreview.value = value
      }, 200)
    },
    { immediate: true }
  )
  
  watch(
    mappedTargetModels,
    () => {
      syncModelCapabilitiesFromMappingWhenIdle()
    }
  )
  
  watch(
    () => JSON.stringify({
      baseUrl: form.baseUrl,
      baseUrls: form.baseUrls,
      apiKeys: form.apiKeys,
      proxyUrl: form.proxyUrl,
      insecureSkipVerify: form.insecureSkipVerify,
      customHeaders: form.customHeaders,
      authHeader: form.authHeader,
      serviceType: form.serviceType,
      routePrefix: form.routePrefix,
    }),
    () => {
      resetTargetModelOptions()
      keyModelsStatus.value.clear()
      fetchModelsError.value = ''
    }
  )
  
  // ESC键监听 & Cmd/Ctrl+Enter 确认
  const handleKeydown = (event: Event) => {
    const keyboardEvent = event as KeyboardEvent
    if (!props.show) return
  
    if (keyboardEvent.key === 'Escape') {
      if (isAnySelectMenuOpen.value || Date.now() < suppressDialogEscapeUntil.value) {
        keyboardEvent.preventDefault()
        keyboardEvent.stopPropagation()
        return
      }
      keyboardEvent.preventDefault()
      handleCancel()
      return
    }
  
    // Cmd/Ctrl+Enter 确认提交
    if (keyboardEvent.key === 'Enter' && (keyboardEvent.metaKey || keyboardEvent.ctrlKey) && !keyboardEvent.shiftKey) {
      keyboardEvent.preventDefault()
      handleSubmit()
    }
  }
  
  onMounted(() => {
    document.addEventListener('keydown', handleKeydown)
  })
  
  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
    detachScrollListener()
    if (formBaseUrlPreviewTimer !== null) {
      window.clearTimeout(formBaseUrlPreviewTimer)
    }
  })

  return {
    formRef,
    activeSection,
    sections,
    baseUrlHasError,
    onMenuUpdate,
    serviceTypeOptions,
    sourceModelOptions,
    modelMappingHint,
    targetModelPlaceholder,
    reasoningEffortOptions,
    reasoningParamStyleOptions,
    textVerbosityOptions,
    supportsOpenAIAdvancedOptions,
    supportsReasoningMappingOptions,
    supportsChatRoleNormalization,
    showModelMappingPresets,
    showMessagesOpenAIChannelPresets,
    showClaudeChannelPresets,
    showCodexResponsesChannelPresets,
    form,
    baseUrlsText,
    modelMappingRows,
    hasNoVisionRows,
    mappedTargetModels,
    sourceMappingError,
    targetModelOptions,
    fetchingModels,
    fetchModelsError,
    keyModelsStatus,
    errors,
    rules,
    isEditing,
    isMac,
    selectedStreamTimeoutStrategy,
    applyStreamTimeoutStrategy,
    commonSupportedModelFilters,
    selectedSupportedModelSet,
    supportedModelsError,
    modelCapabilitiesError,
    startMappingTargetEdit,
    finishMappingTargetEdit,
    headerClasses,
    avatarColor,
    headerIconStyle,
    subtitleClasses,
    isFormValid,
    handleSupportedModelsChange,
    restoringKey,
    submitting,
    disabledKeys,
    expectedRequestUrls,
    customHeadersArray,
    updateCustomHeaders,
    restoreDisabledKey,
    appendSupportedModelFilter,
    ensureTargetModelsLoaded,
    updateForm,
    syncUpstreamModels,
    applyPreset,
    handleSubmit,
    handleCancel,
    handleTestCapability,
    diagnosingCompat,
    handleDiagnoseCompat,
    scrollToSection,
    setSectionRef,
    t,
  }
}
