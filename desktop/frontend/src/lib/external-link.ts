import { Browser } from '@wailsio/runtime'

export const providerConsoleLinks: Record<string, string> = {
  deepseek: 'https://platform.deepseek.com/usage',
  mimo: 'https://platform.xiaomimimo.com/console/balance',
  compshare: 'https://console.compshare.cn/light-gpu/model-manage',
  runapi: 'https://runapi.co/console',
  kimi: 'https://platform.moonshot.cn/console/account',
  glm: 'https://open.bigmodel.cn/coding-plan/personal/overview',
  minimax: 'https://platform.minimaxi.com/user-center/payment/balance',
  dashscope: 'https://bailian.console.aliyun.com/cn-beijing?tab=model#/api-key',
  'opencode-zen': 'https://opencode.ai/',
  'opencode-go': 'https://opencode.ai/',
  openai: 'https://platform.openai.com',
}

export const providerPromotionLinks: Record<string, string> = {
  compshare: 'https://www.compshare.cn/?ytag=GPU_YY_git_ccx',
  runapi: 'https://runapi.co/register?aff=CqQO',
}

export function openExternalLink(url: string) {
  return Browser.OpenURL(url)
}

export function openProviderConsole(provider: string) {
  const url = providerConsoleLinks[provider]
  if (!url) return Promise.resolve()
  return openExternalLink(url)
}

export function openProviderPromotion(provider: string) {
  const url = providerPromotionLinks[provider]
  if (!url) return Promise.resolve()
  return openExternalLink(url)
}
