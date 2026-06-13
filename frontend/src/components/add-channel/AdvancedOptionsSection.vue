<template>
  <div class="advanced-options-section">
    <v-row dense>
      <!-- 描述 -->
      <v-col cols="12">
        <v-textarea
          :model-value="form.description"
          :label="t('addChannel.descriptionLabel')"
          :hint="t('addChannel.descriptionHint')"
          persistent-hint
          prepend-inner-icon="mdi-text"
          variant="outlined"
          density="comfortable"
          rows="3"
          no-resize
          @update:model-value="updateField('description', $event)"
        />
      </v-col>

      <!-- 跳过 TLS 证书验证 -->
      <v-col cols="12">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-2">
            <v-icon color="warning">mdi-shield-alert</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.skipTlsLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.skipTlsHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.insecureSkipVerify" inset color="warning" hide-details @update:model-value="updateField('insecureSkipVerify', $event)" />
        </div>
      </v-col>

      <!-- 低质量渠道标记 -->
      <v-col cols="12">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-2">
            <v-icon color="info">mdi-speedometer-slow</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.lowQualityLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.lowQualityHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.lowQuality" inset color="info" hide-details @update:model-value="updateField('lowQuality', $event)" />
        </div>
      </v-col>

      <!-- Runtime 运行期策略 -->
      <v-col cols="12">
        <v-card variant="outlined" class="pa-4">
          <div class="text-caption font-weight-bold text-uppercase text-medium-emphasis mb-3">
            <v-icon size="small" color="primary" class="mr-1">mdi-cog-outline</v-icon>
            Runtime 运行期策略
          </div>

          <!-- 余额耗尽自动拉黑 -->
          <div class="d-flex align-center justify-space-between mb-3">
            <div class="d-flex align-center ga-2">
              <v-icon color="warning">mdi-cash-remove</v-icon>
              <div>
                <div class="section-title section-title--soft">{{ t('addChannel.autoBlacklistBalanceLabel') }}</div>
                <div class="text-caption text-medium-emphasis">{{ t('addChannel.autoBlacklistBalanceHint') }}</div>
              </div>
            </div>
            <v-switch :model-value="form.autoBlacklistBalance" inset color="warning" hide-details @update:model-value="updateField('autoBlacklistBalance', $event)" />
          </div>

          <!-- 自动学习429限速 -->
          <div class="d-flex align-center justify-space-between">
            <div class="d-flex align-center ga-2">
              <v-icon color="secondary">mdi-robot</v-icon>
              <div>
                <div class="section-title section-title--soft">{{ t('addChannel.rateLimitAutoFromHeadersLabel') }}</div>
                <div class="text-caption text-medium-emphasis">{{ t('addChannel.rateLimitAutoFromHeadersHint') }}</div>
              </div>
            </div>
            <v-switch :model-value="form.rateLimitAutoFromHeaders" inset color="secondary" hide-details @update:model-value="updateField('rateLimitAutoFromHeaders', $event)" />
          </div>
        </v-card>
      </v-col>

      <!-- Codex Native Tool Passthrough -->
      <v-col v-if="channelType === 'responses'" cols="12">
        <div class="d-flex align-center justify-space-between ga-5">
          <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
            <v-icon color="primary">mdi-cog</v-icon>
            <div style="min-width: 0;">
              <div class="section-title section-title--soft">{{ t('addChannel.codexNativeToolPassthroughLabel') }}</div>
              <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.codexNativeToolPassthroughHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.codexNativeToolPassthrough" inset color="primary" hide-details style="flex-shrink: 0;" @update:model-value="updateField('codexNativeToolPassthrough', $event)" />
        </div>
      </v-col>

      <!-- Codex Tool Compat -->
      <v-col v-if="channelType === 'responses'" cols="12">
        <div class="d-flex align-center justify-space-between ga-5">
          <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
            <v-icon color="primary">mdi-cog</v-icon>
            <div style="min-width: 0;">
              <div class="section-title section-title--soft">{{ t('addChannel.codexToolCompatLabel') }}</div>
              <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.codexToolCompatHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.codexToolCompat" inset color="primary" hide-details style="flex-shrink: 0;" @update:model-value="updateField('codexToolCompat', $event)" />
        </div>
      </v-col>

      <!-- Strip Image Generation Tool -->
      <v-col v-if="channelType === 'responses' || channelType === 'chat'" cols="12">
        <div class="d-flex align-center justify-space-between ga-5">
          <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
            <v-icon color="warning">mdi-filter-remove</v-icon>
            <div style="min-width: 0;">
              <div class="section-title section-title--soft">{{ t('addChannel.stripImageGenerationToolLabel') }}</div>
              <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.stripImageGenerationToolHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.stripImageGenerationTool" inset color="warning" hide-details style="flex-shrink: 0;" @update:model-value="updateField('stripImageGenerationTool', $event)" />
        </div>
      </v-col>

      <!-- Normalize Metadata UserId -->
      <v-col v-if="channelType === 'messages' || channelType === 'responses'" cols="12">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-2">
            <v-icon color="primary">mdi-identifier</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.normalizeMetadataUserIdLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.normalizeMetadataUserIdHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.normalizeMetadataUserId" inset color="primary" hide-details @update:model-value="updateField('normalizeMetadataUserId', $event)" />
        </div>
      </v-col>

      <!-- Strip Billing Header -->
      <v-col v-if="channelType === 'messages'" cols="12">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-2">
            <v-icon color="warning">mdi-tag-off</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.stripBillingHeaderLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.stripBillingHeaderHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.stripBillingHeader" inset color="warning" hide-details @update:model-value="updateField('stripBillingHeader', $event)" />
        </div>
      </v-col>

      <!-- Normalize Nonstandard Chat Roles -->
      <v-col v-if="supportsChatRoleNormalization" cols="12">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-2">
            <v-icon color="primary">mdi-account-switch</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.normalizeNonstandardChatRolesLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.normalizeNonstandardChatRolesHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.normalizeNonstandardChatRoles" inset color="primary" hide-details @update:model-value="updateField('normalizeNonstandardChatRoles', $event)" />
        </div>
      </v-col>

      <!-- Reasoning Param Style -->
      <v-col v-if="supportsOpenAIAdvancedOptions" cols="12">
        <div class="d-flex align-center justify-space-between ga-4">
          <div class="d-flex align-center ga-2">
            <v-icon color="primary">mdi-tune</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.reasoningParamStyleLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.reasoningParamStyleHint') }}</div>
            </div>
          </div>
          <v-select
            :model-value="form.reasoningParamStyle"
            :items="reasoningParamStyleOptions"
            variant="outlined"
            density="comfortable"
            hide-details
            class="channel-config-select"
            eager
            @update:model-value="updateField('reasoningParamStyle', $event)"
            @update:menu="$emit('menu-update', $event)"
          />
        </div>
      </v-col>

      <!-- Inject Dummy Thought Signature (Gemini) -->
      <v-col v-if="(channelType === 'gemini' || channelType === 'messages') && form.serviceType === 'gemini'" cols="12">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-2">
            <v-icon color="secondary">mdi-signature</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.injectDummyThoughtSignatureLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.injectDummyThoughtSignatureHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.injectDummyThoughtSignature" inset color="secondary" hide-details @update:model-value="updateField('injectDummyThoughtSignature', $event)" />
        </div>
      </v-col>

      <!-- Strip Thought Signature (Gemini) -->
      <v-col v-if="form.serviceType === 'gemini' && (channelType === 'gemini' || channelType === 'messages' || channelType === 'chat' || channelType === 'responses')" cols="12">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-2">
            <v-icon color="error">mdi-close-circle</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.stripThoughtSignatureLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.stripThoughtSignatureHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.stripThoughtSignature" inset color="error" hide-details @update:model-value="updateField('stripThoughtSignature', $event)" />
        </div>
      </v-col>

      <!-- Passback Reasoning Content (Claude) -->
      <v-col v-if="(channelType === 'messages' || channelType === 'chat' || channelType === 'responses') && form.serviceType === 'claude'" cols="12">
        <div class="d-flex align-center justify-space-between ga-5">
          <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
            <v-icon color="secondary">mdi-brain</v-icon>
            <div style="min-width: 0;">
              <div class="section-title section-title--soft">{{ t('addChannel.passbackReasoningContentLabel') }}</div>
              <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.passbackReasoningContentHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.passbackReasoningContent" inset color="secondary" hide-details style="flex-shrink: 0;" @update:model-value="updateField('passbackReasoningContent', $event)" />
        </div>
      </v-col>

      <!-- Passback Thinking Blocks (Claude) -->
      <v-col v-if="(channelType === 'messages' || channelType === 'chat' || channelType === 'responses') && form.serviceType === 'claude'" cols="12">
        <div class="d-flex align-center justify-space-between ga-5">
          <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
            <v-icon color="secondary">mdi-head-snowflake</v-icon>
            <div style="min-width: 0;">
              <div class="section-title section-title--soft">{{ t('addChannel.passbackThinkingBlocksLabel') }}</div>
              <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.passbackThinkingBlocksHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.passbackThinkingBlocks" inset color="secondary" hide-details style="flex-shrink: 0;" @update:model-value="updateField('passbackThinkingBlocks', $event)" />
        </div>
      </v-col>

      <!-- Strip Empty Text Blocks (Claude Messages) -->
      <v-col v-if="channelType === 'messages' && form.serviceType === 'claude'" cols="12">
        <div class="d-flex align-center justify-space-between ga-5">
          <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
            <v-icon color="warning">mdi-filter-remove</v-icon>
            <div style="min-width: 0;">
              <div class="section-title section-title--soft">{{ t('addChannel.stripEmptyTextBlocksLabel') }}</div>
              <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.stripEmptyTextBlocksHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.stripEmptyTextBlocks" inset color="warning" hide-details style="flex-shrink: 0;" @update:model-value="updateField('stripEmptyTextBlocks', $event)" />
        </div>
      </v-col>

      <!-- Normalize System Role To TopLevel (Messages) -->
      <v-col v-if="channelType === 'messages'" cols="12">
        <div class="d-flex align-center justify-space-between ga-5">
          <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
            <v-icon color="warning">mdi-arrow-collapse-up</v-icon>
            <div style="min-width: 0;">
              <div class="section-title section-title--soft">{{ t('addChannel.normalizeSystemRoleToTopLevelLabel') }}</div>
              <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.normalizeSystemRoleToTopLevelHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.normalizeSystemRoleToTopLevel" inset color="warning" hide-details style="flex-shrink: 0;" @update:model-value="updateField('normalizeSystemRoleToTopLevel', $event)" />
        </div>
      </v-col>

      <slot name="custom-headers" />

      <!-- 代理 URL -->
      <v-col cols="12">
        <v-text-field
          :model-value="form.proxyUrl"
          :label="t('addChannel.proxyUrlLabel')"
          :placeholder="t('addChannel.proxyUrlPlaceholder')"
          prepend-inner-icon="mdi-shield-lock-outline"
          :hint="t('addChannel.proxyUrlHint')"
          persistent-hint
          clearable
          variant="outlined"
          density="comfortable"
          @update:model-value="updateField('proxyUrl', $event)"
        />
      </v-col>

      <!-- 请求超时 -->
      <v-col cols="12">
        <v-text-field
          :model-value="form.requestTimeoutMs"
          :label="t('addChannel.requestTimeoutMsLabel')"
          :placeholder="t('addChannel.requestTimeoutMsPlaceholder')"
          prepend-inner-icon="mdi-timer-sand"
          :hint="t('addChannel.requestTimeoutMsHint')"
          :rules="[rules.requestTimeoutMs]"
          persistent-hint
          clearable
          variant="outlined"
          density="comfortable"
          type="number"
          min="1"
          step="1000"
          @update:model-value="updateField('requestTimeoutMs', $event)"
        />
      </v-col>

      <slot name="stream-timeout" />

      <!-- 主动限速 -->
      <v-col cols="12">
        <div class="d-flex align-center justify-space-between flex-wrap ga-2 mb-2">
          <span class="section-title">{{ t('addChannel.rateLimitSectionLabel') }}</span>
          <span class="text-caption text-medium-emphasis">{{ t('addChannel.rateLimitSectionHint') }}</span>
        </div>
      </v-col>
      <v-col cols="12" md="4">
        <v-text-field
          :model-value="form.rateLimitRpm"
          :label="t('addChannel.rateLimitRpmLabel')"
          :placeholder="t('addChannel.rateLimitRpmPlaceholder')"
          prepend-inner-icon="mdi-speedometer"
          :hint="t('addChannel.rateLimitRpmHint')"
          persistent-hint
          clearable
          variant="outlined"
          density="comfortable"
          type="number"
          min="1"
          @update:model-value="updateField('rateLimitRpm', $event ? parseInt($event) : null)"
        />
      </v-col>
      <v-col cols="12" md="4">
        <v-text-field
          :model-value="form.rateLimitWindowMinutes"
          :label="t('addChannel.rateLimitWindowMinutesLabel')"
          :placeholder="t('addChannel.rateLimitWindowMinutesPlaceholder')"
          prepend-inner-icon="mdi-clock-outline"
          :hint="t('addChannel.rateLimitWindowMinutesHint')"
          persistent-hint
          clearable
          variant="outlined"
          density="comfortable"
          type="number"
          min="1"
          @update:model-value="updateField('rateLimitWindowMinutes', $event ? parseInt($event) : null)"
        />
      </v-col>
      <v-col cols="12" md="4">
        <v-text-field
          :model-value="form.rateLimitMaxConcurrent"
          :label="t('addChannel.rateLimitMaxConcurrentLabel')"
          :placeholder="t('addChannel.rateLimitMaxConcurrentPlaceholder')"
          prepend-inner-icon="mdi-server-network"
          :hint="t('addChannel.rateLimitMaxConcurrentHint')"
          persistent-hint
          clearable
          variant="outlined"
          density="comfortable"
          type="number"
          min="1"
          @update:model-value="updateField('rateLimitMaxConcurrent', $event ? parseInt($event) : null)"
        />
      </v-col>

      <!-- 路由前缀 -->
      <v-col cols="12">
        <v-text-field
          :model-value="form.routePrefix"
          :label="t('addChannel.routePrefixLabel')"
          :placeholder="t('addChannel.routePrefixPlaceholder')"
          prepend-inner-icon="mdi-routes"
          :hint="t('addChannel.routePrefixHint')"
          persistent-hint
          clearable
          variant="outlined"
          density="comfortable"
          @update:model-value="updateField('routePrefix', $event)"
        />
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from '../../i18n'

interface FormData {
  description: string
  insecureSkipVerify: boolean
  lowQuality: boolean
  autoBlacklistBalance: boolean
  codexNativeToolPassthrough?: boolean
  codexToolCompat?: boolean
  stripImageGenerationTool?: boolean
  normalizeMetadataUserId?: boolean
  stripBillingHeader?: boolean
  normalizeNonstandardChatRoles?: boolean
  reasoningParamStyle?: string
  injectDummyThoughtSignature?: boolean
  stripThoughtSignature?: boolean
  passbackReasoningContent?: boolean
  passbackThinkingBlocks?: boolean
  stripEmptyTextBlocks?: boolean
  normalizeSystemRoleToTopLevel?: boolean
  proxyUrl: string
  requestTimeoutMs: string | number | null
  rateLimitRpm: string | number | null
  rateLimitWindowMinutes: string | number | null
  rateLimitMaxConcurrent: string | number | null
  rateLimitAutoFromHeaders: boolean
  routePrefix?: string
  serviceType: string
}

interface Props {
  form: FormData
  channelType: string
  supportsChatRoleNormalization: boolean
  supportsOpenAIAdvancedOptions: boolean
  reasoningParamStyleOptions: Array<{ title: string; value: string }>
  rules: Record<string, (v: any) => boolean | string>
}

defineProps<Props>()

const emit = defineEmits<{
  'update:form': [Partial<FormData>]
  'menu-update': [boolean]
}>()

const { t } = useI18n()

const updateField = (field: keyof FormData, value: any) => {
  emit('update:form', { [field]: value })
}
</script>

<style scoped>
.section-title--soft {
  font-weight: 500;
  font-size: 0.875rem;
}

.channel-config-select {
  max-width: 200px;
}

.rate-limit-card {
  background: rgba(var(--v-theme-surface-variant), 0.3);
}
</style>
