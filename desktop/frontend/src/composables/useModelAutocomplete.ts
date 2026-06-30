import { computed, ref, type ComputedRef, type Ref } from 'vue'

type ModelMappingRow = {
  source: string
  target: string
}

type NewModelMapping = {
  source: string
  target: string
}

type Options = {
  finishMappingTargetEdit: () => void
  modelMappingRows: Ref<ModelMappingRow[]>
  newModelMapping: NewModelMapping
  sourceModelOptions: ComputedRef<string[]>
  targetModelDatalist: ComputedRef<string[]>
}

export function useModelAutocomplete(options: Options) {
  const showTargetSuggestions = ref(false)
  const activeTargetInputId = ref<string | null>(null)
  const targetInputFilter = ref('')

  const showSourceSuggestions = ref(false)
  const activeSourceInputId = ref<string | null>(null)
  const sourceInputFilter = ref('')

  function getTargetModelWindow(index: number): string[] {
    const models = options.targetModelDatalist.value
    const limit = 20
    const before = 8
    const maxStart = Math.max(models.length - limit, 0)
    const start = Math.min(Math.max(index - before, 0), maxStart)
    return models.slice(start, start + limit)
  }

  function getFilteredTargetModels(filter: string): string[] {
    const models = options.targetModelDatalist.value
    const value = filter.trim()
    if (!value) return models.slice(0, 20)

    const lower = value.toLowerCase()
    const exactIndex = models.findIndex(m => m.toLowerCase() === lower)
    if (exactIndex >= 0) return getTargetModelWindow(exactIndex)

    const filtered = models.filter(m => m.toLowerCase().includes(lower))
    if (filtered.length === 1) {
      const index = models.findIndex(m => m === filtered[0])
      if (index >= 0) return getTargetModelWindow(index)
    }

    return filtered.slice(0, 20)
  }

  function getSourceModelWindow(index: number, models: string[]): string[] {
    const limit = 80
    const before = 30
    const maxStart = Math.max(models.length - limit, 0)
    const start = Math.min(Math.max(index - before, 0), maxStart)
    return models.slice(start, start + limit)
  }

  function getFilteredSourceModels(filter: string): string[] {
    const models = options.sourceModelOptions.value
    const value = filter.trim()
    if (!value) return models.slice(0, 80)

    const lower = value.toLowerCase()
    const filtered = models.filter(m => m.toLowerCase().includes(lower))
    if (filtered.length === 1 && filtered[0].toLowerCase() === lower) {
      const index = models.findIndex(m => m === filtered[0])
      if (index >= 0) return getSourceModelWindow(index, models)
    }

    return filtered.slice(0, 80)
  }

  const filteredTargetModels = computed(() => getFilteredTargetModels(targetInputFilter.value))
  const filteredSourceModels = computed(() => getFilteredSourceModels(sourceInputFilter.value))

  function showTargetDropdown(inputId: string, currentValue: string) {
    activeTargetInputId.value = inputId
    targetInputFilter.value = currentValue
    showTargetSuggestions.value = options.targetModelDatalist.value.length > 0
  }

  function hideTargetDropdown() {
    showTargetSuggestions.value = false
    activeTargetInputId.value = null
    options.finishMappingTargetEdit()
  }

  function showSourceDropdown(inputId: string, currentValue: string) {
    activeSourceInputId.value = inputId
    sourceInputFilter.value = currentValue
    showSourceSuggestions.value = true
  }

  function hideSourceDropdown() {
    showSourceSuggestions.value = false
    activeSourceInputId.value = null
  }

  function handlePointerDown(e: PointerEvent) {
    const target = e.target as Element | null
    if (target?.closest('[data-target-model-picker]') || target?.closest('[data-source-model-picker]')) return
    hideTargetDropdown()
    hideSourceDropdown()
  }

  function selectSourceModel(inputId: string, model: string) {
    if (inputId === 'new-source') {
      options.newModelMapping.source = model
    }
    showSourceSuggestions.value = false
    activeSourceInputId.value = null
  }

  function selectTargetModel(inputId: string, model: string) {
    if (inputId === 'new') {
      options.newModelMapping.target = model
    } else if (inputId.startsWith('row-')) {
      const index = parseInt(inputId.slice(4), 10)
      if (!Number.isNaN(index) && options.modelMappingRows.value[index]) {
        options.modelMappingRows.value[index].target = model
      }
    }
    showTargetSuggestions.value = false
    activeTargetInputId.value = null
    options.finishMappingTargetEdit()
  }

  return {
    showTargetSuggestions,
    activeTargetInputId,
    filteredTargetModels,
    showSourceSuggestions,
    activeSourceInputId,
    filteredSourceModels,
    showTargetDropdown,
    hideTargetDropdown,
    handlePointerDown,
    showSourceDropdown,
    hideSourceDropdown,
    selectSourceModel,
    selectTargetModel,
  }
}
