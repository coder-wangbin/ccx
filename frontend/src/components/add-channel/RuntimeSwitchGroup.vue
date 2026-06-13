<template>
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
</template>

<script setup lang="ts">
import { useI18n } from '../../i18n'

interface FormData {
  autoBlacklistBalance: boolean
  rateLimitAutoFromHeaders: boolean
}

interface Props {
  form: FormData
}

defineProps<Props>()

const emit = defineEmits<{
  'update:field': [field: keyof FormData, value: unknown]
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
</style>
