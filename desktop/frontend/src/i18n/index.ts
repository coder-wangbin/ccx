export { defaultLocale, languageOptions, messages } from './messages'
export type { MessageKey, SupportedLocale } from './messages'
export { applyDocumentLanguage, normalizeLocale, resolveInitialLocale, translate } from './core'
export { useLanguage as useI18n } from '@/composables/useLanguage'
