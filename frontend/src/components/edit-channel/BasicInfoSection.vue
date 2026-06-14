<template>
  <div class="basic-info-section">
    <v-row>
      <!-- 渠道名称 -->
      <v-col cols="12" md="6">
        <v-text-field
          :model-value="form.name"
          :label="t('channelEditor.basic.name.label')"
          :placeholder="t('channelEditor.basic.name.placeholder')"
          prepend-inner-icon="mdi-tag"
          variant="outlined"
          density="comfortable"
          :rules="[rules.required]"
          required
          :error-messages="errors.name"
          @update:model-value="updateField('name', $event)"
        />
      </v-col>

      <!-- 服务类型 -->
      <v-col cols="12" md="6">
        <v-select
          :model-value="form.serviceType"
          :label="t('channelEditor.basic.serviceType.label')"
          :items="serviceTypeOptions"
          prepend-inner-icon="mdi-cog"
          variant="outlined"
          density="comfortable"
          :rules="[rules.required]"
          required
          :error-messages="errors.serviceType"
          eager
          @update:model-value="updateField('serviceType', $event)"
          @update:menu="$emit('menu-update', $event)"
        />
      </v-col>

      <!-- Base URL -->
      <v-col cols="12">
        <v-textarea
          :model-value="baseUrlsText"
          :label="t('channelEditor.basic.baseUrl.label')"
          :placeholder="t('channelEditor.basic.baseUrl.placeholder')"
          prepend-inner-icon="mdi-web"
          variant="outlined"
          density="comfortable"
          rows="3"
          no-resize
          :rules="[rules.required, rules.baseUrls]"
          required
          :error-messages="errors.baseUrl"
          hide-details="auto"
          @update:model-value="$emit('update:baseUrlsText', $event)"
        />
        <!-- 预期请求提示 -->
        <div v-show="expectedRequestUrls.length > 0 && !baseUrlHasError" class="base-url-hint">
          <div v-for="(item, index) in expectedRequestUrls" :key="index" class="expected-request-item">
            <span class="text-caption text-medium-emphasis">
              {{ t('addChannel.expectedRequest') }} {{ item.expectedUrl }}
            </span>
          </div>
        </div>
      </v-col>

      <!-- 官网/控制台 -->
      <v-col cols="12">
        <v-text-field
          :model-value="form.website"
          :label="t('channelEditor.basic.website.label')"
          :placeholder="t('channelEditor.basic.website.placeholder')"
          prepend-inner-icon="mdi-open-in-new"
          variant="outlined"
          density="comfortable"
          type="url"
          :rules="[rules.urlOptional]"
          :error-messages="errors.website"
          @update:model-value="updateField('website', $event)"
        />
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '../../i18n'

interface FormData {
  name: string
  serviceType: string
  website: string
}

interface Props {
  form: FormData
  baseUrlsText: string
  expectedRequestUrls: Array<{ expectedUrl: string }>
  baseUrlHasError: boolean
  serviceTypeOptions: Array<{ title: string; value: string }>
  errors: Record<string, string>
  rules: Record<string, (v: any) => boolean | string>
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:form': [Partial<FormData>]
  'update:baseUrlsText': [string]
  'menu-update': [boolean]
}>()

const { t } = useI18n()

const updateField = (field: keyof FormData, value: any) => {
  emit('update:form', { [field]: value })
}
</script>

<style scoped>
.base-url-hint {
  margin-top: 8px;
  padding: 8px 12px;
  background: rgba(var(--v-theme-surface-variant), 0.3);
  border-radius: 4px;
}

.expected-request-item {
  margin: 2px 0;
}
</style>
