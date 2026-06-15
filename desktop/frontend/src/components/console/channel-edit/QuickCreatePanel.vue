<script setup lang="ts">
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AlertCircle, CheckCircle2 } from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'

defineProps<{
  quickInput: string
  serviceType: string
  serviceTypeOptions: Array<{ label: string; value: string }>
  detectedServiceType: string | null
  detectedBaseUrls: string[]
  detectedApiKeys: string[]
  userSelectedServiceType: boolean
  expectedRequestUrls: Array<{ baseUrl: string; expectedUrl: string }>
}>()

const emit = defineEmits<{
  (e: 'update:quick-input', value: string): void
  (e: 'update:service-type', value: string): void
  (e: 'quick-paste', text: string): void
}>()

const { tf } = useLanguage()
</script>

<template>
  <section class="space-y-3 rounded-xl border border-primary/20 bg-primary/5 p-4">
    <div class="grid gap-2 rounded-lg border border-border bg-background/70 p-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
      <div class="space-y-1.5">
        <div class="text-xs font-semibold text-muted-foreground">
          {{ tf('channelEditor.basic.serviceType.label', '上游类型') }}
        </div>
        <Select :model-value="serviceType" @update:model-value="(val) => emit('update:service-type', String(val))">
          <SelectTrigger class="h-9 bg-background">
            <SelectValue :placeholder="tf('channelEditor.basic.serviceType.placeholder', '选择服务类型')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="opt in serviceTypeOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="rounded-md border border-border/60 bg-muted/40 px-2 py-1.5 text-[10px] text-muted-foreground">
        <span v-if="userSelectedServiceType">
          {{ tf('addChannel.serviceTypeManual', '手动选择') }}
        </span>
        <span v-else-if="detectedServiceType">
          {{ tf('addChannel.serviceTypeDetected', '已自动识别') }}
        </span>
        <span v-else>
          {{ tf('addChannel.serviceTypeDefault', '使用默认类型') }}
        </span>
      </div>
    </div>

    <Textarea
      :model-value="quickInput"
      rows="10"
      class="!field-sizing-none min-h-[14rem] font-mono text-xs"
      :placeholder="tf('addChannel.quickInputPlaceholder', '粘贴配置片段，自动识别 Base URL 和 API Key（支持多行）')"
      @update:model-value="(val) => emit('update:quick-input', val as string)"
      @paste="emit('quick-paste', $event.clipboardData?.getData('text/plain') || '')"
    />

    <div class="grid gap-2 md:grid-cols-2">
      <div class="rounded-lg border border-border bg-background/70 p-2 text-xs">
        <div class="mb-1 flex items-center gap-1.5 font-semibold">
          <CheckCircle2 v-if="detectedBaseUrls.length" class="h-3.5 w-3.5 text-emerald-500" />
          <AlertCircle v-else class="h-3.5 w-3.5 text-muted-foreground" />
          Base URLs
        </div>
        <template v-if="detectedBaseUrls.length">
          <div v-for="item in expectedRequestUrls" :key="item.baseUrl" class="mb-1">
            <p class="truncate text-[11px] text-emerald-600">{{ item.baseUrl }}</p>
            <p class="truncate text-[10px] text-muted-foreground/70">
              {{ tf('addChannel.expectedRequest', '预期请求') }} {{ item.expectedUrl }}
            </p>
          </div>
        </template>
        <p v-else class="truncate text-muted-foreground">
          {{ tf('addChannel.noneDetected', '未识别') }}
        </p>
      </div>

      <div class="rounded-lg border border-border bg-background/70 p-2 text-xs">
        <div class="mb-1 flex items-center gap-1.5 font-semibold">
          <CheckCircle2 v-if="detectedApiKeys.length" class="h-3.5 w-3.5 text-emerald-500" />
          <AlertCircle v-else class="h-3.5 w-3.5 text-muted-foreground" />
          {{ tf('channelEditor.auth.keys.label', 'API Keys') }}
        </div>
        <p class="text-muted-foreground">
          {{ detectedApiKeys.length ? `${detectedApiKeys.length} ${tf('channelCard.configuredKeys', 'active keys')}` : tf('addChannel.noneDetected', '未识别') }}
        </p>
      </div>
    </div>
  </section>
</template>
