// preset-messages.ts
// 渠道预设/Plan/Target 文案的 i18n 翻译表（仅 EN）。
// zh-CN 直接 fallback 到 Go 后端 channelpreset/preset.go 中的原中文，
// 避免在前端重复维护一份中文表。
//
// Key 命名约定：
//   channel.preset.{providerId}.label
//   channel.preset.{providerId}.description
//   channel.preset.{providerId}.plan.{planId}.label
//   channel.preset.{providerId}.plan.{planId}.description
//   channel.preset.{providerId}.target.{type}.description  // provider 级覆盖（仅在需要差异化时定义）
//   channel.target.{type}.label                             // 共用 target 标签
//   channel.target.{type}.description                       // 共用 target 描述（默认值）

import type { SupportedLocale } from './messages'

export const presetMessages: Record<SupportedLocale, Record<string, string>> = {
  en: {
    // 共用 target
    'channel.target.messages.label': 'Messages native',
    'channel.target.chat.label': 'Chat passthrough',
    'channel.target.responses.label': 'Codex Responses',
    'channel.target.messages.description': 'Claude Code direct, or CCX messages channel',
    'channel.target.chat.description': 'OpenAI Chat protocol, for Chat clients',
    'channel.target.responses.description': 'OpenAI Responses protocol, for Codex',

    // DeepSeek
    'channel.preset.deepseek.description':
      'Messages native passthrough, Codex Responses, and Chat passthrough — three usage modes.',
    'channel.preset.deepseek.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.deepseek.plan.openai-chat.description': 'Common Chat / Responses endpoint',

    // MiMo
    'channel.preset.mimo.description':
      'Messages native, Codex Responses, and Chat passthrough; includes pay-as-you-go and Token Plan endpoints.',
    'channel.preset.mimo.plan.anthropic.label': 'Pay-as-you-go (Anthropic)',
    'channel.preset.mimo.plan.anthropic.description': 'Messages native endpoint',
    'channel.preset.mimo.plan.openai-chat.label': 'Pay-as-you-go (OpenAI)',
    'channel.preset.mimo.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    'channel.preset.mimo.plan.token-cn.label': 'Token Plan (China)',
    'channel.preset.mimo.plan.token-cn.description': 'China subscription plan',
    'channel.preset.mimo.plan.token-sgp.label': 'Token Plan (Singapore)',
    'channel.preset.mimo.plan.token-sgp.description': 'Singapore subscription plan',
    'channel.preset.mimo.plan.token-ams.label': 'Token Plan (Europe)',
    'channel.preset.mimo.plan.token-ams.description': 'Europe subscription plan',
    'channel.preset.mimo.plan.token-cn-anthropic.label': 'Token Plan (China · Anthropic)',
    'channel.preset.mimo.plan.token-cn-anthropic.description':
      'China subscription plan (Anthropic endpoint)',
    'channel.preset.mimo.plan.token-sgp-anthropic.label': 'Token Plan (Singapore · Anthropic)',
    'channel.preset.mimo.plan.token-sgp-anthropic.description':
      'Singapore subscription plan (Anthropic endpoint)',
    'channel.preset.mimo.plan.token-ams-anthropic.label': 'Token Plan (Europe · Anthropic)',
    'channel.preset.mimo.plan.token-ams-anthropic.description':
      'Europe subscription plan (Anthropic endpoint)',

    // Compshare
    'channel.preset.compshare.label': 'Youyun Zhisuan Plans',
    'channel.preset.compshare.description':
      "Youyun Zhisuan is UCloud's AI cloud platform, offering cost-effective domestic AI model Agent Plan packages by monthly subscription or pay-as-you-go, starting from 49 CNY/month. It also provides stable access to official overseas models, supports Claude Code, Codex, and API integrations, and offers enterprise-grade high concurrency, 24/7 technical support, and self-service invoicing. Users who register through the promotion link can receive a free 5 CNY platform trial credit.",
    'channel.preset.compshare.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.compshare.plan.openai-chat.description':
      'OpenAI Chat / Responses-compatible endpoint',

    // RunAPI
    'channel.preset.runapi.label': 'RunAPI',
    'channel.preset.runapi.description':
      "RunAPI is an efficient and stable API platform—an alternative to OpenRouter. A single API Key gives you access to 150+ leading models, including OpenAI, Claude, Gemini, DeepSeek, Grok, and more, at prices as low as 10% of the original (up to 90% off), with exceptional stability. It's seamlessly compatible with tools like Claude Code, OpenClaw, and others. RunAPI offers an exclusive perk for CCX users: register and contact an administrator to claim ¥7 in free credit.",
    'channel.preset.runapi.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.runapi.plan.openai-chat.description':
      'OpenAI-compatible endpoint',
    'channel.preset.runapi.target.responses.plan.openai-chat.label': 'Responses native',
    'channel.preset.runapi.target.responses.plan.openai-chat.description': 'Native Responses endpoint for Codex',

    // Kimi
    'channel.preset.kimi.description':
      'Messages native passthrough, Codex Responses, and Chat passthrough — three usage modes.',
    'channel.preset.kimi.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.kimi.plan.openai-chat.description': 'Moonshot OpenAI-compatible endpoint',

    // GLM
    'channel.preset.glm.description':
      'Messages native passthrough, Codex Responses, and Chat passthrough — three usage modes.',
    'channel.preset.glm.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.glm.plan.coding.description': 'Zhipu Coding plan endpoint',
    'channel.preset.glm.plan.openai-chat.label': 'OpenAI-compatible (general)',
    'channel.preset.glm.plan.openai-chat.description': 'Zhipu general OpenAI-compatible endpoint',

    // MiniMax
    'channel.preset.minimax.description':
      'Messages native passthrough, Codex Responses, and Chat passthrough — three usage modes.',
    'channel.preset.minimax.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.minimax.plan.openai-chat.description': 'MiniMax OpenAI-compatible endpoint',

    // DashScope
    'channel.preset.dashscope.label': 'Alibaba Cloud DashScope',
    'channel.preset.dashscope.description':
      'Messages native passthrough, Codex Responses, and Chat passthrough — three usage modes.',
    'channel.preset.dashscope.plan.anthropic.label': 'Pay-as-you-go (Anthropic)',
    'channel.preset.dashscope.plan.anthropic.description': 'Messages native endpoint',
    'channel.preset.dashscope.plan.openai-chat.label': 'Pay-as-you-go (OpenAI)',
    'channel.preset.dashscope.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    'channel.preset.dashscope.plan.coding-anthropic.description':
      'Subscription plan Messages endpoint',
    'channel.preset.dashscope.plan.coding-openai-chat.description':
      'Subscription plan OpenAI-compatible endpoint',
    'channel.preset.dashscope.plan.token-plan-anthropic.description':
      'Token Plan Messages endpoint',
    'channel.preset.dashscope.plan.token-plan-openai-chat.description':
      'Token Plan OpenAI-compatible endpoint',

    // OpenCode Zen
    'channel.preset.opencode-zen.description':
      'Pay-as-you-go curated-model gateway, supports Messages, Chat, and Responses protocols.',
    'channel.preset.opencode-zen.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.opencode-zen.plan.openai-chat.description':
      'OpenCode Zen OpenAI-compatible endpoint',

    // OpenCode Go
    'channel.preset.opencode-go.description':
      'Low-cost open-source coding model subscription (from $5/month), supports Messages, Chat, and Responses protocols.',
    'channel.preset.opencode-go.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.opencode-go.plan.openai-chat.description': 'OpenCode Go OpenAI-compatible endpoint',
  },
  'zh-CN': {
    // 留空：所有 key 都通过 translateOrFallback 回退到 Go preset 中的原中文。
  },
}
