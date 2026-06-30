<template>
  <v-dialog :model-value="show" max-width="1200" persistent scrollable @update:model-value="$emit('update:show', $event)">
    <v-card rounded="lg" class="add-channel-dialog channel-editor-dialog">
      <!-- 头部 -->
      <AddChannelHeader
        :is-editing="isEditing"
        :channel-type="props.channelType"
        :no-vision="form.noVision"
        :header-classes="headerClasses"
        :avatar-color="avatarColor"
        :header-icon-style="headerIconStyle"
        :subtitle-classes="subtitleClasses"
        :edit-title="t('addChannel.editTitle')"
        :create-title="t('addChannel.createTitle')"
        :edit-subtitle="t('addChannel.editSubtitle')"
        :create-subtitle="t('addChannel.quickSubtitle')"
        :test-capability-label="t('addChannel.testCapability')"
        :vision-tooltip="form.noVision ? t('channelCard.noVision') : t('channelCard.hasVision')"
        @toggle-no-vision="form.noVision = !form.noVision"
        @test-capability="handleTestCapability"
      />

      <!-- 主体内容 -->
      <v-card-text class="pa-0 channel-editor-body">
        <!-- 左侧导航 + 右侧面板 -->
        <div class="content-row">
          <!-- 左侧垂直导航 -->
          <AddChannelSidebarNav
            :title="t('addChannel.outline')"
            :sections="sections"
            :active-section="activeSection"
            @navigate="scrollToSection"
          />

          <!-- 右侧内容面板 -->
          <v-form ref="formRef" class="content-area" @submit.prevent="handleSubmit">
            <!-- 基本信息 -->
            <section :ref="(el: any) => setSectionRef('basic', el)" data-section-id="basic" class="pa-6 scroll-mt-4">
              <BasicInfoSection
                :form="form"
                :base-urls-text="baseUrlsText"
                :expected-request-urls="expectedRequestUrls"
                :base-url-has-error="baseUrlHasError"
                :service-type-options="serviceTypeOptions"
                :errors="errors"
                :rules="rules"
                @update:form="updateForm"
                @update:base-urls-text="baseUrlsText = $event"
                @menu-update="onMenuUpdate"
              />
            </section>

            <!-- 身份认证 -->
            <section :ref="(el: any) => setSectionRef('auth', el)" data-section-id="auth" class="pa-6 scroll-mt-4">
              <ApiKeyManagementSection
                :api-keys="form.apiKeys"
                :disabled-keys="disabledKeys"
                :key-models-status="keyModelsStatus"
                :is-editing="isEditing"
                :restoring-key="restoringKey"
                :service-type="form.serviceType"
                :channel-id="props.channel?.index"
                @update:api-keys="form.apiKeys = $event"
                @restore-key="restoreDisabledKey"
              />
            </section>

            <!-- 模型重定向（模型映射 + Vision 回退 + 模型过滤） -->
            <section :ref="(el: any) => setSectionRef('redirect', el)" data-section-id="redirect" class="pa-6 scroll-mt-4">
              <ModelMappingSection
                v-if="form.serviceType"
                :mapping-rows="modelMappingRows"
                :source-model-options="sourceModelOptions"
                :target-model-options="targetModelOptions"
                :fetching-models="fetchingModels"
                :source-mapping-error="sourceMappingError"
                :fetch-models-error="fetchModelsError"
                :model-mapping-hint="modelMappingHint"
                :target-model-placeholder="targetModelPlaceholder"
                :show-model-mapping-presets="showModelMappingPresets"
                :show-messages-open-a-i-channel-presets="showMessagesOpenAIChannelPresets"
                :show-claude-channel-presets="showClaudeChannelPresets"
                :show-codex-responses-channel-presets="showCodexResponsesChannelPresets"
                :supports-reasoning-mapping-options="supportsReasoningMappingOptions"
                :reasoning-effort-options="reasoningEffortOptions"
                @update:mapping-rows="modelMappingRows = ($event as any)"
                @sync-upstream="syncUpstreamModels"
                @apply-preset="applyPreset"
                @menu-update="onMenuUpdate"
                @target-edit-start="startMappingTargetEdit"
                @target-edit-end="finishMappingTargetEdit"
              >
                <template #vision-fallback>
                  <div v-if="hasNoVisionRows" class="mt-6">
                    <v-row dense>
                      <v-col cols="12" :md="supportsReasoningMappingOptions ? 8 : 12">
                        <v-combobox
                          v-model="form.visionFallbackModel"
                          :label="t('addChannel.visionFallbackLabel')"
                          :placeholder="t('addChannel.visionFallbackPlaceholder')"
                          :hint="t('addChannel.visionFallbackHint')"
                          :items="targetModelOptions"
                          prepend-inner-icon="mdi-eye"
                          persistent-hint
                          clearable
                          variant="outlined"
                          density="comfortable"
                          eager
                          @focus="startMappingTargetEdit(); ensureTargetModelsLoaded()"
                          @blur="finishMappingTargetEdit"
                          @update:menu="onMenuUpdate"
                        />
                      </v-col>
                      <v-col v-if="supportsReasoningMappingOptions" cols="12" md="4">
                        <v-select
                          v-model="form.visionFallbackReasoningEffort"
                          :label="t('addChannel.visionFallbackReasoningLabel')"
                          :items="reasoningEffortOptions"
                          variant="outlined"
                          density="comfortable"
                          clearable
                          persistent-hint
                          :hint="t('addChannel.visionFallbackReasoningHint')"
                          eager
                          @update:menu="onMenuUpdate"
                        />
                      </v-col>
                    </v-row>
                  </div>
                </template>
              </ModelMappingSection>

              <!-- 模型过滤 -->
              <div class="mt-4">
                <SupportedModelsFilter
                  :model-value="form.supportedModels"
                  :error="supportedModelsError"
                  :common-filters="commonSupportedModelFilters"
                  :selected-filters="Array.from(selectedSupportedModelSet)"
                  @update:model-value="handleSupportedModelsChange($event as any)"
                  @append-filter="appendSupportedModelFilter"
                  @menu-update="onMenuUpdate"
                />
              </div>

              <div v-if="props.channelType !== 'images'" class="mt-6">
                <ModelCapabilitySection
                  v-model:rows="form.modelCapabilityRows"
                  :target-model-options="targetModelOptions"
                  :mapped-target-models="mappedTargetModels"
                  :fetching-models="fetchingModels"
                  :fetch-models-error="fetchModelsError"
                  :error="modelCapabilitiesError"
                  @sync-upstream="syncUpstreamModels"
                  @menu-update="onMenuUpdate"
                />
              </div>
            </section>

            <!-- 高级选项 -->
            <section :ref="(el: any) => setSectionRef('advanced', el)" data-section-id="advanced" class="pa-6 scroll-mt-4">
              <AdvancedOptionsSection
                :form="form"
                :channel-type="props.channelType"
                :supports-chat-role-normalization="supportsChatRoleNormalization"
                :supports-open-a-i-advanced-options="supportsOpenAIAdvancedOptions"
                :reasoning-param-style-options="reasoningParamStyleOptions"
                :text-verbosity-options="textVerbosityOptions"
                :diagnosing="diagnosingCompat"
                @update:form="updateForm"
                @menu-update="onMenuUpdate"
                @diagnose="handleDiagnoseCompat"
              />
            </section>

            <!-- 自定义参数（自定义请求头 + 流式超时） -->
            <section :ref="(el: any) => setSectionRef('custom', el)" data-section-id="custom" class="pa-6 scroll-mt-4">
              <CustomHeadersSection
                :headers="customHeadersArray"
                @update:headers="updateCustomHeaders"
              />

              <div class="mt-6">
                <TransportConfigGroup :form="form" @update:field="(field, value) => updateForm({ [field]: value })" />
              </div>

              <div class="mt-6">
                <StreamTimeoutSection
                  :request-timeout-ms="form.requestTimeoutMs"
                  :response-header-timeout-ms="form.responseHeaderTimeoutMs"
                  :selected-strategy="selectedStreamTimeoutStrategy"
                  :first-content-enabled="form.streamFirstContentTimeoutEnabled"
                  :first-content-ms="form.streamFirstContentTimeoutMs"
                  :inactivity-enabled="form.streamInactivityTimeoutEnabled"
                  :inactivity-ms="form.streamInactivityTimeoutMs"
                  :tool-call-idle-enabled="form.streamToolCallIdleTimeoutEnabled"
                  :tool-call-idle-ms="form.streamToolCallIdleTimeoutMs"
                  @update:request-timeout-ms="form.requestTimeoutMs = $event"
                  @update:response-header-timeout-ms="form.responseHeaderTimeoutMs = $event"
                  @apply-strategy="applyStreamTimeoutStrategy"
                  @update:first-content-ms="form.streamFirstContentTimeoutMs = $event"
                  @update:inactivity-ms="form.streamInactivityTimeoutMs = $event"
                  @update:tool-call-idle-ms="form.streamToolCallIdleTimeoutMs = $event"
                />
              </div>

              <div class="mt-6">
                <RateLimitGroup :form="form" @update:field="(field, value) => updateForm({ [field]: value })" />
              </div>
            </section>
          </v-form>
        </div>
      </v-card-text>

      <!-- 底部按钮 -->
      <v-card-actions class="pa-6 pt-2">
        <v-spacer />
        <v-btn variant="outlined" @click="handleCancel">
          {{ t('app.actions.cancel') }}<span class="shortcut-hint ml-2 text-xs opacity-50">Esc</span>
        </v-btn>
        <v-btn
          color="primary"
          variant="elevated"
          :disabled="!isFormValid"
          :loading="submitting"
          @click="handleSubmit"
        >
          {{ t('app.actions.save') }}<span class="shortcut-hint ml-2 text-xs opacity-50">{{ isMac ? '⌘Enter' : 'Ctrl+Enter' }}</span>
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
// 子组件导入
import AddChannelHeader from './edit-channel/AddChannelHeader.vue'
import AddChannelSidebarNav from './edit-channel/AddChannelSidebarNav.vue'
import BasicInfoSection from './edit-channel/BasicInfoSection.vue'
import ApiKeyManagementSection from './edit-channel/ApiKeyManagementSection.vue'
import ModelMappingSection from './edit-channel/ModelMappingSection.vue'
import ModelCapabilitySection from './edit-channel/ModelCapabilitySection.vue'
import SupportedModelsFilter from './edit-channel/SupportedModelsFilter.vue'
import CustomHeadersSection from './edit-channel/CustomHeadersSection.vue'
import StreamTimeoutSection from './edit-channel/StreamTimeoutSection.vue'
import AdvancedOptionsSection from './edit-channel/AdvancedOptionsSection.vue'
import TransportConfigGroup from './edit-channel/TransportConfigGroup.vue'
import RateLimitGroup from './edit-channel/RateLimitGroup.vue'
import { useEditChannelModal, type EditChannelModalEmits, type EditChannelModalProps } from '../composables/useEditChannelModal'

const props = withDefaults(defineProps<EditChannelModalProps>(), {
  channelType: 'messages',
})

const emit = defineEmits<EditChannelModalEmits>()

const {
  formRef,
  activeSection,
  sections,
  baseUrlHasError,
  onMenuUpdate,
  serviceTypeOptions,
  sourceModelOptions,
  modelMappingHint,
  targetModelPlaceholder,
  reasoningEffortOptions,
  reasoningParamStyleOptions,
  textVerbosityOptions,
  supportsOpenAIAdvancedOptions,
  supportsReasoningMappingOptions,
  supportsChatRoleNormalization,
  showModelMappingPresets,
  showMessagesOpenAIChannelPresets,
  showClaudeChannelPresets,
  showCodexResponsesChannelPresets,
  form,
  baseUrlsText,
  modelMappingRows,
  hasNoVisionRows,
  mappedTargetModels,
  sourceMappingError,
  targetModelOptions,
  fetchingModels,
  fetchModelsError,
  keyModelsStatus,
  errors,
  rules,
  isEditing,
  isMac,
  selectedStreamTimeoutStrategy,
  applyStreamTimeoutStrategy,
  commonSupportedModelFilters,
  selectedSupportedModelSet,
  supportedModelsError,
  modelCapabilitiesError,
  startMappingTargetEdit,
  finishMappingTargetEdit,
  headerClasses,
  avatarColor,
  headerIconStyle,
  subtitleClasses,
  isFormValid,
  handleSupportedModelsChange,
  restoringKey,
  submitting,
  disabledKeys,
  expectedRequestUrls,
  customHeadersArray,
  updateCustomHeaders,
  restoreDisabledKey,
  appendSupportedModelFilter,
  ensureTargetModelsLoaded,
  updateForm,
  syncUpstreamModels,
  applyPreset,
  handleSubmit,
  handleCancel,
  handleTestCapability,
  diagnosingCompat,
  handleDiagnoseCompat,
  scrollToSection,
  setSectionRef,
  t,
} = useEditChannelModal(props, emit)
</script>

<style scoped src="./edit-channel/edit-channel-modal.css"></style>
