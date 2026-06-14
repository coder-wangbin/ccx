<script setup lang="ts">
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useLanguage } from '@/composables/useLanguage'

interface FormData {
  fastMode: boolean
  textVerbosity: 'low' | 'medium' | 'high' | ''
  visionFallbackModel: string
  noVision: boolean
  historicalImageTurnLimit: number
  passbackReasoningContent: boolean
  passbackThinkingBlocks: boolean
  lowQuality: boolean
  injectDummyThoughtSignature: boolean
  stripThoughtSignature: boolean
  stripEmptyTextBlocks: boolean
  normalizeSystemRoleToTopLevel: boolean
  normalizeMetadataUserId: boolean
  stripBillingHeader: boolean
  normalizeNonstandardChatRoles: boolean
  autoBlacklistBalance: boolean
  codexNativeToolPassthrough: boolean
  codexToolCompat: boolean
  stripCodexClientTools: boolean
  stripImageGenerationTool: boolean
  reasoningParamStyle: string
  serviceType: string
  proxyUrl: string
  routePrefix: string
  rateLimitRpm: string | number
  rateLimitWindowMinutes: string | number
  rateLimitMaxConcurrent: string | number
  rateLimitAutoFromHeaders: boolean
  requestTimeoutMs: string | number
}

const props = defineProps<{
  form: FormData
  channelType: string
  textVerbosityOptions: Array<{ label: string; value: string }>
  supportsOpenAIAdvanced: boolean
  supportsOpenAIAdvancedOptions: boolean
  supportsChatRoleNormalization: boolean
  DEFAULT_SELECT_VALUE: string
}>()

const emit = defineEmits<{
  'update:form': [value: Partial<FormData>]
}>()

const { tf } = useLanguage()

function updateField<K extends keyof FormData>(key: K, value: FormData[K]) {
  emit('update:form', { [key]: value } as Partial<FormData>)
}

function toSelectValue(value: string): string {
  return value === '' ? props.DEFAULT_SELECT_VALUE : value
}

function fromSelectValue(value: string): string {
  return value === props.DEFAULT_SELECT_VALUE ? '' : value
}

</script>

<template>
  <div class="space-y-6">
    <!-- 生成参数 -->
    <section v-if="supportsOpenAIAdvanced || channelType === 'responses' || channelType === 'chat'" class="space-y-3 rounded-xl border border-border/60 bg-card/40 p-5 shadow-xs">
      <h4 class="text-xs font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
        {{ tf('console.form.generationParams', '生成参数') }}
      </h4>

      <div v-if="supportsOpenAIAdvanced" class="grid gap-4 md:grid-cols-2 bg-background/30 p-3 rounded-lg border border-border/40">
        <div class="flex items-center justify-between p-2 rounded-md hover:bg-accent/40 transition-colors">
          <div class="space-y-0.5">
            <Label class="text-xs font-semibold">{{ tf('console.form.fastMode', '快速模式') }}</Label>
            <p class="text-[10px] text-muted-foreground">{{ tf('console.form.fastModeHint', '优先选取低延迟的轻量边缘路由链路') }}</p>
          </div>
          <Switch :model-value="form.fastMode" @update:model-value="updateField('fastMode', $event)" />
        </div>

        <div class="space-y-1 p-1">
          <Label class="text-[10px] font-bold text-muted-foreground uppercase">{{ tf('console.form.textVerbosityStyle', 'Text Verbosity Style') }}</Label>
          <Select :model-value="toSelectValue(form.textVerbosity)" @update:model-value="(val) => updateField('textVerbosity', fromSelectValue(val as string) as any)">
            <SelectTrigger class="h-9 w-full">
              <SelectValue :placeholder="tf('console.form.selectDefault', '默认')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="opt in textVerbosityOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <!-- Vision 控制 -->
      <div v-if="['messages', 'chat'].includes(channelType)" class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-3">
        <div class="text-[10px] font-bold uppercase tracking-wider text-primary/80 border-b border-border/30 pb-1">
          {{ tf('console.form.visionTitle', '视觉控制') }}
        </div>
        <div class="space-y-3">
          <div class="flex flex-row-reverse items-center justify-between gap-3">
            <Switch :model-value="form.noVision" @update:model-value="updateField('noVision', $event)" class="shrink-0" />
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ tf('console.form.noVision', '跳过含图请求') }}</Label>
              <p class="text-[10px] text-muted-foreground">
                {{ tf('console.form.noVisionHint', '启用后，包含图片的请求将跳过此渠道并 failover 到下一个渠道') }}
              </p>
            </div>
          </div>
          <div class="space-y-1">
            <Label class="text-[10px] font-bold text-muted-foreground">
              {{ tf('console.form.historicalImageTurnLimit', '历史图片轮次限制') }}
            </Label>
            <Input
              :model-value="form.historicalImageTurnLimit"
              type="number"
              min="0"
              class="h-8 text-xs"
              placeholder="0"
              @update:model-value="updateField('historicalImageTurnLimit', Number($event))"
            />
            <p class="text-[10px] leading-4 text-muted-foreground">
              {{ tf('console.form.historicalImageTurnLimitHint', '0 = 继承全局；后端会对 >0 的值应用最低 3 约束') }}
            </p>
          </div>
        </div>
      </div>
    </section>

    <!-- 高级扩展选项 -->
    <section class="space-y-6 rounded-xl border border-border/60 bg-card/40 p-4 shadow-xs">
      <h4 class="text-xs font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
        {{ tf('console.form.advancedOptions', '高级扩展选项') }}
      </h4>

      <div class="space-y-2.5">
        <!-- Runtime 运行期策略 -->
        <div class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-2.5">
        <div class="text-[10px] font-bold uppercase tracking-wider text-primary/80 border-b border-border/30 pb-1">
          {{ tf('console.form.runtimeTitle', '运行期策略') }}
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <Label class="text-xs font-medium">{{ tf('console.form.autoBlacklistBalanceLabel', '自动熔断/黑名单余额异常 Key') }}</Label>
            <Switch :model-value="form.autoBlacklistBalance" @update:model-value="updateField('autoBlacklistBalance', $event)" />
          </div>
          <div class="flex items-center justify-between">
            <div class="space-y-0.5">
              <Label class="text-xs font-medium">{{ tf('console.form.rateLimitAutoLabel', '自动学习上游限速') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">
                {{ tf('console.form.rateLimitAutoHint', '解析 Retry-After / x-ratelimit-* 响应头动态调整 cooldown') }}
              </p>
            </div>
            <Switch :model-value="form.rateLimitAutoFromHeaders" @update:model-value="updateField('rateLimitAutoFromHeaders', $event)" />
          </div>
        </div>
      </div>

      <!-- 协议规范化 -->
      <div class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-2.5">
      <div class="text-[10px] font-bold uppercase tracking-wider text-primary/80 border-b border-border/30 pb-1">
        {{ tf('console.form.compatibilityTitle', '协议规范化') }}
      </div>
      <div class="space-y-2">
        <div v-if="channelType === 'responses'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.codexNativeToolPassthrough', 'Codex 原生工具透传') }}</Label>
          <Switch :model-value="form.codexNativeToolPassthrough" @update:model-value="updateField('codexNativeToolPassthrough', $event)" />
        </div>
        <div v-if="channelType === 'responses'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.codexToolCompat', 'Codex 工具兼容模式') }}</Label>
          <Switch :model-value="form.codexToolCompat" @update:model-value="updateField('codexToolCompat', $event)" />
        </div>
        <div v-if="channelType === 'responses' || channelType === 'chat'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.stripImageGenerationTool', '过滤图像生成工具') }}</Label>
          <Switch :model-value="form.stripImageGenerationTool" @update:model-value="updateField('stripImageGenerationTool', $event)" />
        </div>
        <div v-if="channelType === 'messages'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.normalizeSystemRole', '规范化 System 角色域') }}</Label>
          <Switch :model-value="form.normalizeSystemRoleToTopLevel" @update:model-value="updateField('normalizeSystemRoleToTopLevel', $event)" />
        </div>
        <div v-if="['messages','responses'].includes(channelType)" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.normalizeUserId', '平铺扁平化用户 ID') }}</Label>
          <Switch :model-value="form.normalizeMetadataUserId" @update:model-value="updateField('normalizeMetadataUserId', $event)" />
        </div>
        <div v-if="channelType === 'messages'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.stripBillingHeader', '抽离并剔除 CCH 计费尾缀') }}</Label>
          <Switch :model-value="form.stripBillingHeader" @update:model-value="updateField('stripBillingHeader', $event)" />
        </div>
        <div v-if="supportsChatRoleNormalization" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.normalizeNonstandardChatRoles', '规范化非标准聊天角色') }}</Label>
          <Switch :model-value="form.normalizeNonstandardChatRoles" @update:model-value="updateField('normalizeNonstandardChatRoles', $event)" />
        </div>
        <div v-if="supportsOpenAIAdvancedOptions" class="flex items-center justify-between">
          <div class="flex-1">
            <Label class="text-xs font-medium">{{ tf('console.form.reasoningParamStyle', 'Reasoning 参数样式') }}</Label>
          </div>
          <Select
            :model-value="form.reasoningParamStyle || 'default'"
            @update:model-value="(val) => updateField('reasoningParamStyle', val === 'default' ? '' : String(val))"
          >
            <SelectTrigger class="h-8 w-[140px] text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="default">{{ tf('console.form.selectDefaultLabel', '默认') }}</SelectItem>
              <SelectItem value="reasoning_effort">reasoning_effort</SelectItem>
              <SelectItem value="developer_message">developer_message</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="(channelType === 'gemini' || channelType === 'messages') && form.serviceType === 'gemini'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.injectDummyThoughtSignature', '注入 Gemini 思考标记') }}</Label>
          <Switch :model-value="form.injectDummyThoughtSignature" @update:model-value="updateField('injectDummyThoughtSignature', $event)" />
        </div>
        <div v-if="form.serviceType === 'gemini'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.stripThoughtSignature', '剥离 Gemini 思考标记') }}</Label>
          <Switch :model-value="form.stripThoughtSignature" @update:model-value="updateField('stripThoughtSignature', $event)" />
        </div>
        <div v-if="(channelType === 'messages' || channelType === 'chat' || channelType === 'responses') && form.serviceType === 'claude'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.passbackReasoningContent', '回传 Claude 推理内容') }}</Label>
          <Switch :model-value="form.passbackReasoningContent" @update:model-value="updateField('passbackReasoningContent', $event)" />
        </div>
        <div v-if="(channelType === 'messages' || channelType === 'chat' || channelType === 'responses') && form.serviceType === 'claude'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.passbackThinkingBlocks', '回传 Claude 思考块') }}</Label>
          <Switch :model-value="form.passbackThinkingBlocks" @update:model-value="updateField('passbackThinkingBlocks', $event)" />
        </div>
        <div v-if="channelType === 'messages' && form.serviceType === 'claude'" class="flex items-center justify-between">
          <Label class="text-xs font-medium">{{ tf('console.form.stripEmptyTextBlocks', '剥离空文本块') }}</Label>
          <Switch :model-value="form.stripEmptyTextBlocks" @update:model-value="updateField('stripEmptyTextBlocks', $event)" />
        </div>
      </div>
    </div>
      </div>

      <!-- Transport 代理路由网络 -->
      <div class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-3">
        <div class="text-[10px] font-bold uppercase tracking-wider text-primary/80 border-b border-border/30 pb-1">
          {{ tf('console.form.transportTitle', '代理路由网络') }}
        </div>
        <div class="grid grid-cols-3 gap-2">
          <div class="space-y-1">
            <Label class="text-[9px] font-bold text-muted-foreground">{{ tf('console.form.proxyUrlLabel', '代理通道 URL') }}</Label>
            <Input
              :model-value="form.proxyUrl"
              class="h-8 w-full font-mono text-xs"
              placeholder="socks5://..."
              @update:model-value="(val) => updateField('proxyUrl', val as string)"
            />
          </div>
          <div class="space-y-1">
            <Label class="text-[9px] font-bold text-muted-foreground">{{ tf('console.form.requestTimeoutLabel', '请求超时 (ms)') }}</Label>
            <Input
              :model-value="form.requestTimeoutMs"
              type="number"
              class="h-8 w-full text-xs"
              placeholder="60000"
              @update:model-value="(val) => updateField('requestTimeoutMs', val)"
            />
          </div>
          <div class="space-y-1">
            <Label class="text-[9px] font-bold text-muted-foreground">{{ tf('console.form.routePrefixLabel', '接口路由前缀') }}</Label>
            <Input
              :model-value="form.routePrefix"
              class="h-8 w-full font-mono text-xs"
              placeholder="kimi"
              @update:model-value="(val) => updateField('routePrefix', val as string)"
            />
          </div>
        </div>
      </div>

      <!-- Rate Limit -->
      <div class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-3">
        <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
          {{ tf('console.form.rateLimitTitle', '上游主动限速流量阀') }}
        </div>
        <div class="grid grid-cols-3 gap-3">
          <div class="space-y-1">
            <Label class="text-[10px] font-medium text-muted-foreground/80">{{ tf('console.form.rpmLabel', '每分钟最大请求量 (RPM)') }}</Label>
            <Input
              :model-value="form.rateLimitRpm"
              type="number"
              class="h-9 text-xs"
              :placeholder="tf('console.form.rpmPlaceholder', '不设限制')"
              @update:model-value="updateField('rateLimitRpm', $event)"
            />
          </div>
          <div class="space-y-1">
            <Label class="text-[10px] font-medium text-muted-foreground/80">{{ tf('console.form.windowLabel', '检测窗口滑动时间 (s)') }}</Label>
            <Input
              :model-value="form.rateLimitWindowMinutes"
              type="number"
              class="h-9 text-xs"
              :placeholder="tf('console.form.windowPlaceholder', '60 秒')"
              @update:model-value="updateField('rateLimitWindowMinutes', $event)"
            />
          </div>
          <div class="space-y-1">
            <Label class="text-[10px] font-medium text-muted-foreground/80">{{ tf('console.form.maxConcurrentLabel', '全双工最大并发数限制') }}</Label>
            <Input
              :model-value="form.rateLimitMaxConcurrent"
              type="number"
              class="h-9 text-xs"
              :placeholder="tf('console.form.maxConcurrentPlaceholder', '不设限制')"
              @update:model-value="updateField('rateLimitMaxConcurrent', $event)"
            />
          </div>
        </div>
      </div>

    </section>
  </div>
</template>
