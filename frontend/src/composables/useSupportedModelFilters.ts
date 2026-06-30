import { computed, ref } from 'vue'
import {
  filterValidSupportedModelPatterns,
  parseSupportedModelInput,
} from '../utils/add-channel-modal-state'
import { normalizeSelectableString } from '../utils/channelPayload'

type Translator = (key: string) => string
type FormLike = {
  supportedModels: string[]
}

export function useSupportedModelFilters(form: FormLike, t: Translator) {
  const commonSupportedModelFilters = ['claude-*', 'gpt-5*', 'gpt-image-2', 'grok-4*', 'gemini-3*', '!*image*']
  const selectedSupportedModelSet = computed(() => new Set(form.supportedModels))
  const supportedModelsError = ref('')

  const handleSupportedModelsChange = (values: Array<string | { title: string; value: string }>) => {
    const normalizedValues = values
      .map(normalizeSelectableString)
      .flatMap(parseSupportedModelInput)

    const { validPatterns, hasInvalidPatterns } = filterValidSupportedModelPatterns(normalizedValues)
    form.supportedModels = validPatterns
    supportedModelsError.value = hasInvalidPatterns ? t('addChannel.supportedModelsInvalidPattern') : ''
  }

  const isSupportedModelSelected = (filter: string): boolean => {
    return selectedSupportedModelSet.value.has(filter)
  }

  const appendSupportedModelFilter = (filter: string) => {
    if (isSupportedModelSelected(filter)) {
      return
    }
    form.supportedModels.push(filter)
    supportedModelsError.value = ''
  }

  return {
    commonSupportedModelFilters,
    selectedSupportedModelSet,
    supportedModelsError,
    handleSupportedModelsChange,
    isSupportedModelSelected,
    appendSupportedModelFilter,
  }
}
