<template>
  <v-card variant="outlined" class="pa-4">
    <div class="text-caption font-weight-bold text-uppercase text-medium-emphasis mb-3">
      <v-icon size="small" color="primary" class="mr-1">mdi-format-align-justify</v-icon>
      Compatibility 协议规范化
    </div>

    <div class="d-flex flex-column ga-3">
      <!-- Codex Native Tool Passthrough -->
      <div v-if="channelType === 'responses'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="primary">mdi-cog</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.codexNativeToolPassthroughLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.codexNativeToolPassthroughHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.codexNativeToolPassthrough" inset color="primary" hide-details @update:model-value="updateField('codexNativeToolPassthrough', $event)" />
      </div>

      <!-- Codex Tool Compat -->
      <div v-if="channelType === 'responses'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="primary">mdi-cog</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.codexToolCompatLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.codexToolCompatHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.codexToolCompat" inset color="primary" hide-details @update:model-value="updateField('codexToolCompat', $event)" />
      </div>

      <!-- Strip Image Generation Tool -->
      <div v-if="channelType === 'responses' || channelType === 'chat'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="warning">mdi-filter-remove</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.stripImageGenerationToolLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.stripImageGenerationToolHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.stripImageGenerationTool" inset color="warning" hide-details @update:model-value="updateField('stripImageGenerationTool', $event)" />
      </div>

      <!-- Normalize System Role To TopLevel -->
      <div v-if="channelType === 'messages'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="warning">mdi-arrow-collapse-up</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.normalizeSystemRoleToTopLevelLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.normalizeSystemRoleToTopLevelHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.normalizeSystemRoleToTopLevel" inset color="warning" hide-details @update:model-value="updateField('normalizeSystemRoleToTopLevel', $event)" />
      </div>

      <!-- Normalize Metadata UserId -->
      <div v-if="channelType === 'messages' || channelType === 'responses'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="primary">mdi-identifier</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.normalizeMetadataUserIdLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.normalizeMetadataUserIdHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.normalizeMetadataUserId" inset color="primary" hide-details @update:model-value="updateField('normalizeMetadataUserId', $event)" />
      </div>

      <!-- Strip Billing Header -->
      <div v-if="channelType === 'messages'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="warning">mdi-tag-off</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.stripBillingHeaderLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.stripBillingHeaderHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.stripBillingHeader" inset color="warning" hide-details @update:model-value="updateField('stripBillingHeader', $event)" />
      </div>

      <!-- Normalize Nonstandard Chat Roles -->
      <div v-if="supportsChatRoleNormalization" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="primary">mdi-account-switch</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.normalizeNonstandardChatRolesLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.normalizeNonstandardChatRolesHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.normalizeNonstandardChatRoles" inset color="primary" hide-details @update:model-value="updateField('normalizeNonstandardChatRoles', $event)" />
      </div>

      <!-- Reasoning Param Style -->
      <div v-if="supportsOpenAIAdvancedOptions" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2 flex-1">
          <v-icon color="primary">mdi-tune</v-icon>
          <div class="flex-1">
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
          style="max-width: 200px;"
          eager
          @update:model-value="updateField('reasoningParamStyle', $event)"
          @update:menu="$emit('menu-update', $event)"
        />
      </div>

      <!-- Fast Mode -->
      <div v-if="supportsOpenAIAdvancedOptions" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="primary">mdi-fast-forward</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.fastModeLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.fastModeHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.fastMode" inset color="primary" hide-details @update:model-value="updateField('fastMode', $event)" />
      </div>

      <!-- Inject Dummy Thought Signature (Gemini) -->
      <div v-if="(channelType === 'gemini' || channelType === 'messages') && form.serviceType === 'gemini'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="secondary">mdi-signature</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.injectDummyThoughtSignatureLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.injectDummyThoughtSignatureHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.injectDummyThoughtSignature" inset color="secondary" hide-details @update:model-value="updateField('injectDummyThoughtSignature', $event)" />
      </div>

      <!-- Strip Thought Signature (Gemini) -->
      <div v-if="form.serviceType === 'gemini' && (channelType === 'gemini' || channelType === 'messages' || channelType === 'chat' || channelType === 'responses')" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="error">mdi-close-circle</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.stripThoughtSignatureLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.stripThoughtSignatureHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.stripThoughtSignature" inset color="error" hide-details @update:model-value="updateField('stripThoughtSignature', $event)" />
      </div>

      <!-- Passback Reasoning Content (Claude) -->
      <div v-if="(channelType === 'messages' || channelType === 'chat' || channelType === 'responses') && form.serviceType === 'claude'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="secondary">mdi-brain</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.passbackReasoningContentLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.passbackReasoningContentHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.passbackReasoningContent" inset color="secondary" hide-details @update:model-value="updateField('passbackReasoningContent', $event)" />
      </div>

      <!-- Passback Thinking Blocks (Claude) -->
      <div v-if="(channelType === 'messages' || channelType === 'chat' || channelType === 'responses') && form.serviceType === 'claude'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="secondary">mdi-head-snowflake</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.passbackThinkingBlocksLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.passbackThinkingBlocksHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.passbackThinkingBlocks" inset color="secondary" hide-details @update:model-value="updateField('passbackThinkingBlocks', $event)" />
      </div>

      <!-- Strip Empty Text Blocks (Claude) -->
      <div v-if="channelType === 'messages' && form.serviceType === 'claude'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2">
          <v-icon color="warning">mdi-filter-remove</v-icon>
          <div>
            <div class="section-title section-title--soft">{{ t('addChannel.stripEmptyTextBlocksLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.stripEmptyTextBlocksHint') }}</div>
          </div>
        </div>
        <v-switch :model-value="form.stripEmptyTextBlocks" inset color="warning" hide-details @update:model-value="updateField('stripEmptyTextBlocks', $event)" />
      </div>

      <!-- Historical Image Turn Limit -->
      <div v-if="channelType === 'messages' || channelType === 'chat'" class="d-flex align-center justify-space-between">
        <div class="d-flex align-center ga-2 flex-1">
          <v-icon color="primary">mdi-image-multiple</v-icon>
          <div class="flex-1">
            <div class="section-title section-title--soft">{{ t('addChannel.historicalImageTurnLimitLabel') }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('addChannel.historicalImageTurnLimitHint') }}</div>
          </div>
        </div>
        <v-text-field
          :model-value="form.historicalImageTurnLimit"
          type="number"
          min="0"
          variant="outlined"
          density="comfortable"
          hide-details
          style="max-width: 120px;"
          @update:model-value="updateField('historicalImageTurnLimit', Number($event))"
        />
      </div>
    </div>
  </v-card>
</template>

<script setup lang="ts">
import { useI18n } from '../../i18n'

interface FormData {
  serviceType: string
  codexNativeToolPassthrough?: boolean
  codexToolCompat?: boolean
  stripImageGenerationTool?: boolean
  normalizeSystemRoleToTopLevel?: boolean
  normalizeMetadataUserId?: boolean
  stripBillingHeader?: boolean
  normalizeNonstandardChatRoles?: boolean
  reasoningParamStyle?: string
  fastMode?: boolean
  injectDummyThoughtSignature?: boolean
  stripThoughtSignature?: boolean
  passbackReasoningContent?: boolean
  passbackThinkingBlocks?: boolean
  stripEmptyTextBlocks?: boolean
  historicalImageTurnLimit?: number
}

interface Props {
  form: FormData
  channelType: string
  supportsChatRoleNormalization: boolean
  supportsOpenAIAdvancedOptions: boolean
  reasoningParamStyleOptions: Array<{ title: string; value: string }>
}

defineProps<Props>()

const emit = defineEmits<{
  'update:field': [field: keyof FormData, value: unknown]
  'menu-update': [open: boolean]
}>()

const { t } = useI18n()

const updateField = (field: keyof FormData, value: unknown) => {
  emit('update:field', field, value)
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
</style>
