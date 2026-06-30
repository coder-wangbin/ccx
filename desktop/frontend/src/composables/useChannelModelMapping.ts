import { computed, nextTick, reactive, ref, watch, type ComputedRef } from 'vue'
import {
  createModelCapabilityRow,
  resolveBuiltinUpstreamModelCapability,
  type ModelCapabilityRow,
} from '@/utils/channel-payload'
import type { Channel } from '@/services/admin-api'

export type ReasoningEffort = 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'

export interface ModelMappingRow {
  id: number
  source: string
  target: string
  reasoning: ReasoningEffort | ''
  noVision: boolean
}

type FormLike = {
  modelCapabilityRows: ModelCapabilityRow[]
  visionFallbackModel: string
  visionFallbackReasoningEffort: ReasoningEffort | ''
}

type ChannelModelMappingOptions = {
  form: FormLike
  getSourceMappingError: () => string
  nextRowId: () => number
  supportsReasoningMappingOptions: ComputedRef<boolean>
}

export function useChannelModelMapping(options: ChannelModelMappingOptions) {
  const modelMappingRows = ref<ModelMappingRow[]>([])
  const modelCapabilityRows = ref<ModelCapabilityRow[]>([])
  const incompleteMappedTargetSuffix = /[._:/-]$/
  const isMappingTargetEditing = ref(false)
  const hasPendingModelCapabilitySync = ref(false)
  const newModelMapping = reactive<ModelMappingRow>({ id: 0, source: '', target: '', reasoning: '', noVision: false })

  const isCompleteMappedTargetModel = (model: string) => !!model && !incompleteMappedTargetSuffix.test(model)

  const mappedTargetModels = computed(() => {
    const seen = new Set<string>()
    const models = [
      ...modelMappingRows.value.map(row => row.target.trim()),
      options.form.visionFallbackModel.trim(),
    ]

    return models.filter(model => {
      const key = model.toLowerCase()
      if (!isCompleteMappedTargetModel(model) || seen.has(key)) return false
      seen.add(key)
      return true
    })
  })

  function modelMappingFromChannel(ch: Channel) {
    const mapping = ch.modelMapping || {}
    const reasoning = ch.reasoningMapping || {}
    const noVision = new Set(ch.noVisionModels || [])
    return Object.entries(mapping).map(([source, target]) => ({
      id: options.nextRowId(),
      source,
      target,
      reasoning: (reasoning[source] || '') as ModelMappingRow['reasoning'],
      noVision: noVision.has(target),
    }))
  }

  function addModelMappingRow() {
    if (!newModelMapping.source.trim() || !newModelMapping.target.trim() || options.getSourceMappingError()) return
    const target = newModelMapping.target.trim()
    modelMappingRows.value.push({
      id: options.nextRowId(),
      source: newModelMapping.source.trim(),
      target,
      reasoning: newModelMapping.reasoning || '',
      noVision: findNoVisionForTarget(target) ?? newModelMapping.noVision,
    })
    newModelMapping.source = ''
    newModelMapping.target = ''
    newModelMapping.reasoning = ''
    newModelMapping.noVision = false
    finishMappingTargetEdit()
  }

  function removeModelMappingRow(id: number) {
    const index = modelMappingRows.value.findIndex(row => row.id === id)
    if (index >= 0) modelMappingRows.value.splice(index, 1)
  }

  function getModelMappingAsObject(): Record<string, string> {
    const result: Record<string, string> = {}
    for (const row of modelMappingRows.value) {
      if (row.source && row.target) result[row.source] = row.target
    }
    return result
  }

  function getReasoningMappingAsObject(): Record<string, ReasoningEffort> {
    const result: Record<string, ReasoningEffort> = {}
    for (const row of modelMappingRows.value) {
      if (row.source && row.target && row.reasoning) {
        result[row.source] = row.reasoning
      }
    }
    return result
  }

  function applyVisionFallbackReasoning(payload: Partial<Channel>) {
    const fallbackModel = options.form.visionFallbackModel.trim()
    if (!options.supportsReasoningMappingOptions.value || !fallbackModel) {
      return
    }

    const reasoningMapping = { ...(payload.reasoningMapping || {}) }
    if (options.form.visionFallbackReasoningEffort) {
      reasoningMapping[fallbackModel] = options.form.visionFallbackReasoningEffort
    } else if (!modelMappingRows.value.some(row => row.source === fallbackModel && row.reasoning)) {
      delete reasoningMapping[fallbackModel]
    }
    payload.reasoningMapping = reasoningMapping
  }

  function getNoVisionModelsFromRows(): string[] {
    return [...new Set(
      modelMappingRows.value
        .filter(row => row.noVision && row.target.trim())
        .map(row => row.target.trim())
    )]
  }

  function normalizeTargetKey(target: string): string {
    return target.trim()
  }

  function findNoVisionForTarget(target: string): boolean | undefined {
    const targetKey = normalizeTargetKey(target)
    const matched = modelMappingRows.value.find(row => normalizeTargetKey(row.target) === targetKey)
    return matched?.noVision
  }

  function setNoVisionForTarget(target: string, noVision: boolean) {
    const targetKey = normalizeTargetKey(target)
    if (!targetKey) return
    modelMappingRows.value.forEach(row => {
      if (normalizeTargetKey(row.target) === targetKey) {
        row.noVision = noVision
      }
    })
  }

  function updateMappingRow(id: number, field: keyof ModelMappingRow, value: any) {
    const row = modelMappingRows.value.find(r => r.id === id)
    if (!row) return

    if (field === 'noVision') {
      setNoVisionForTarget(row.target, value)
    } else if (field === 'target') {
      const target = String(value).trim()
      const existingNoVision = findNoVisionForTarget(target)
      row.target = target
      row.noVision = existingNoVision ?? row.noVision
      setNoVisionForTarget(target, row.noVision)
    } else {
      ;(row as any)[field] = value
    }
  }

  function updateModelCapabilityRows(rows: ModelCapabilityRow[]) {
    modelCapabilityRows.value = rows
    options.form.modelCapabilityRows = rows
  }

  function syncModelCapabilitiesFromMapping() {
    const existingModels = new Set(
      modelCapabilityRows.value
        .map(row => row.model.trim())
        .map(model => model.toLowerCase())
        .filter(Boolean)
    )
    const rowsToAdd = mappedTargetModels.value
      .filter(isCompleteMappedTargetModel)
      .filter(model => !existingModels.has(model.toLowerCase()))
      .map(model => {
        const builtin = resolveBuiltinUpstreamModelCapability(model)
        return createModelCapabilityRow(
          options.nextRowId(),
          model,
          builtin?.capability,
          builtin ? 'builtin' : 'custom',
          builtin?.pattern || '',
        )
      })
    if (!rowsToAdd.length) return
    updateModelCapabilityRows([...modelCapabilityRows.value, ...rowsToAdd])
  }

  function syncModelCapabilitiesFromMappingWhenIdle() {
    if (isMappingTargetEditing.value) {
      hasPendingModelCapabilitySync.value = true
      return
    }
    hasPendingModelCapabilitySync.value = false
    syncModelCapabilitiesFromMapping()
  }

  function startMappingTargetEdit() {
    isMappingTargetEditing.value = true
  }

  function finishMappingTargetEdit() {
    if (!isMappingTargetEditing.value) return
    isMappingTargetEditing.value = false
    if (!hasPendingModelCapabilitySync.value) return
    hasPendingModelCapabilitySync.value = false
    nextTick(syncModelCapabilitiesFromMapping)
  }

  watch(mappedTargetModels, () => {
    syncModelCapabilitiesFromMappingWhenIdle()
  })

  return {
    modelMappingRows,
    modelCapabilityRows,
    mappedTargetModels,
    newModelMapping,
    modelMappingFromChannel,
    addModelMappingRow,
    removeModelMappingRow,
    getModelMappingAsObject,
    getReasoningMappingAsObject,
    applyVisionFallbackReasoning,
    getNoVisionModelsFromRows,
    updateMappingRow,
    updateModelCapabilityRows,
    syncModelCapabilitiesFromMapping,
    startMappingTargetEdit,
    finishMappingTargetEdit,
  }
}
