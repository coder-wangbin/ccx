import { ref, type Ref } from 'vue'
import {
  COPILOT_OAUTH_DEVICE_CODE_PATH,
  COPILOT_OAUTH_TOKEN_PATH,
  type CopilotDeviceCodeResponse,
  type CopilotTokenResponse,
} from '@/services/admin-api'
import { useAdminApi } from '@/composables/useAdminApi'
import { openExternalLink } from '@/lib/external-link'

type Translator = (key: string, params?: Record<string, string>) => string

export function useCopilotOAuth(existingApiKeys: Ref<string[]>, t: Translator) {
  const adminApi = useAdminApi()
  const copilotOAuthLoading = ref(false)
  const copilotPolling = ref(false)
  const copilotOAuthError = ref('')
  const copilotOAuthSuccess = ref(false)
  const copilotUserCode = ref('')
  const copilotVerificationUri = ref('')
  const copilotDeviceCode = ref('')
  const copilotUserCodeCopied = ref(false)
  let copilotPollTimer: ReturnType<typeof setTimeout> | null = null
  let copilotCopyTimer: ReturnType<typeof setTimeout> | null = null

  function clearCopilotPollTimer() {
    if (copilotPollTimer !== null) {
      clearTimeout(copilotPollTimer)
      copilotPollTimer = null
    }
  }

  function clearCopilotCopyTimer() {
    if (copilotCopyTimer !== null) {
      clearTimeout(copilotCopyTimer)
      copilotCopyTimer = null
    }
  }

  function clearCopilotAuthorizationCode() {
    copilotDeviceCode.value = ''
    copilotUserCode.value = ''
    copilotVerificationUri.value = ''
    copilotUserCodeCopied.value = false
    clearCopilotCopyTimer()
  }

  async function copyCopilotUserCode() {
    const userCode = copilotUserCode.value.trim()
    if (!userCode) return
    try {
      await navigator.clipboard.writeText(userCode)
      clearCopilotCopyTimer()
      copilotUserCodeCopied.value = true
      copilotCopyTimer = setTimeout(() => {
        copilotUserCodeCopied.value = false
        copilotCopyTimer = null
      }, 1200)
    } catch {
      // clipboard 不可用时静默
    }
  }

  async function pollCopilotToken(intervalSeconds: number) {
    if (!copilotDeviceCode.value) return
    copilotPolling.value = true
    try {
      const token = await adminApi.post<CopilotTokenResponse>(COPILOT_OAUTH_TOKEN_PATH, { deviceCode: copilotDeviceCode.value })
      if (token.accessToken) {
        if (!existingApiKeys.value.includes(token.accessToken)) existingApiKeys.value.push(token.accessToken)
        copilotOAuthSuccess.value = true
        copilotOAuthError.value = ''
        copilotPolling.value = false
        copilotOAuthLoading.value = false
        clearCopilotPollTimer()
        clearCopilotAuthorizationCode()
        return
      }
      if (token.error === 'expired_token') {
        copilotOAuthError.value = t('copilotOAuth.expired')
        copilotPolling.value = false
        copilotOAuthLoading.value = false
        clearCopilotPollTimer()
        return
      }
      if (token.error && token.error !== 'authorization_pending') {
        copilotOAuthError.value = token.errorDescription || token.error
        copilotPolling.value = false
        copilotOAuthLoading.value = false
        clearCopilotPollTimer()
        return
      }
    } catch (err) {
      copilotOAuthError.value = err instanceof Error ? err.message : String(err)
      copilotPolling.value = false
      copilotOAuthLoading.value = false
      clearCopilotPollTimer()
      return
    }
    copilotPollTimer = setTimeout(() => pollCopilotToken(intervalSeconds), Math.max(intervalSeconds, 5) * 1000)
  }

  async function startCopilotOAuth() {
    clearCopilotPollTimer()
    clearCopilotAuthorizationCode()
    copilotOAuthLoading.value = true
    copilotOAuthError.value = ''
    copilotOAuthSuccess.value = false
    try {
      const device = await adminApi.post<CopilotDeviceCodeResponse>(COPILOT_OAUTH_DEVICE_CODE_PATH)
      copilotDeviceCode.value = device.deviceCode
      copilotUserCode.value = device.userCode
      copilotVerificationUri.value = device.verificationUri
      await openCopilotAuthorization()
      await pollCopilotToken(device.interval || 5)
    } catch (err) {
      copilotOAuthError.value = err instanceof Error ? err.message : String(err)
      copilotOAuthLoading.value = false
      copilotPolling.value = false
    }
  }

  async function openCopilotAuthorization() {
    if (!copilotVerificationUri.value) return
    try {
      await openExternalLink(copilotVerificationUri.value)
    } catch (err) {
      copilotOAuthError.value = err instanceof Error ? err.message : String(err)
    }
  }

  return {
    adminApi,
    copilotOAuthLoading,
    copilotPolling,
    copilotOAuthError,
    copilotOAuthSuccess,
    copilotUserCode,
    copilotUserCodeCopied,
    clearCopilotPollTimer,
    clearCopilotCopyTimer,
    copyCopilotUserCode,
    startCopilotOAuth,
    openCopilotAuthorization,
  }
}
