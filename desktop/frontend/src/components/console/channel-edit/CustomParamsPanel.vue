<script setup lang="ts">
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Clock, Globe } from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'

interface FormData {
  proxyUrl: string
  routePrefix: string
  requestTimeoutMs: string | number
  responseHeaderTimeoutMs: string | number
  rateLimitRpm: string | number
  rateLimitWindowMinutes: string | number
  rateLimitMaxConcurrent: string | number
}

defineProps<{
  form: FormData
}>()

const emit = defineEmits<{
  'update:form': [value: Partial<FormData>]
}>()

const { t } = useLanguage()

function updateField<K extends keyof FormData>(key: K, value: FormData[K]) {
  emit('update:form', { [key]: value } as Partial<FormData>)
}
</script>

<template>
  <div class="space-y-6">
    <!-- Transport 代理路由网络 -->
    <section class="p-4 rounded-lg border border-border/60 bg-gradient-to-br from-background/60 to-background/40 shadow-sm backdrop-blur-sm space-y-3">
      <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
        <Globe class="h-3 w-3" />
        {{ t('channelEditor.transport.title') }}
      </div>
      <div class="grid gap-2">
        <div class="space-y-1">
          <Label class="text-[9px] font-bold text-muted-foreground">{{ t('channelEditor.transport.proxyUrl.label') }}</Label>
          <Input
            :model-value="form.proxyUrl"
            class="h-8 w-full font-mono text-xs"
            :placeholder="t('channelEditor.transport.proxyUrl.placeholder')"
            @update:model-value="(val) => updateField('proxyUrl', val as string)"
          />
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.transport.proxyUrl.hint') }}</p>
        </div>
        <div class="space-y-1">
          <Label class="text-[9px] font-bold text-muted-foreground">{{ t('channelEditor.transport.requestTimeout.label') }}</Label>
          <Input
            :model-value="form.requestTimeoutMs"
            type="number"
            min="1000"
            max="300000"
            step="1000"
            class="h-8 w-full text-xs"
            :placeholder="t('channelEditor.transport.requestTimeout.placeholder')"
            @update:model-value="(val) => updateField('requestTimeoutMs', val)"
          />
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.transport.requestTimeout.hint') }}</p>
        </div>
        <div class="space-y-1">
          <Label class="text-[9px] font-bold text-muted-foreground">{{ t('channelEditor.transport.responseHeaderTimeout.label') }}</Label>
          <Input
            :model-value="form.responseHeaderTimeoutMs"
            type="number"
            min="1000"
            max="300000"
            step="1000"
            class="h-8 w-full text-xs"
            :placeholder="t('channelEditor.transport.responseHeaderTimeout.placeholder')"
            @update:model-value="(val) => updateField('responseHeaderTimeoutMs', val)"
          />
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.transport.responseHeaderTimeout.hint') }}</p>
        </div>
        <div class="space-y-1">
          <Label class="text-[9px] font-bold text-muted-foreground">{{ t('channelEditor.transport.routePrefix.label') }}</Label>
          <Input
            :model-value="form.routePrefix"
            class="h-8 w-full font-mono text-xs"
            :placeholder="t('channelEditor.transport.routePrefix.placeholder')"
            @update:model-value="(val) => updateField('routePrefix', val as string)"
          />
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.transport.routePrefix.hint') }}</p>
        </div>
      </div>
    </section>

    <!-- Rate Limit -->
    <section class="p-4 rounded-lg border border-border/60 bg-gradient-to-br from-background/60 to-background/40 shadow-sm backdrop-blur-sm space-y-3">
      <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
        <Clock class="h-3 w-3" />
        {{ t('channelEditor.rateLimit.title') }}
      </div>
      <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.rateLimit.section.hint') }}</p>
      <div class="grid gap-3 md:grid-cols-3">
        <div class="space-y-1">
          <Label class="text-[10px] font-medium text-muted-foreground/80">{{ t('channelEditor.rateLimit.rpm.label') }}</Label>
          <Input
            :model-value="form.rateLimitRpm"
            type="number"
            class="h-9 text-xs"
            :placeholder="t('channelEditor.rateLimit.rpm.placeholder')"
            @update:model-value="updateField('rateLimitRpm', $event)"
          />
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.rateLimit.rpm.hint') }}</p>
        </div>
        <div class="space-y-1">
          <Label class="text-[10px] font-medium text-muted-foreground/80">{{ t('channelEditor.rateLimit.window.label') }}</Label>
          <Input
            :model-value="form.rateLimitWindowMinutes"
            type="number"
            class="h-9 text-xs"
            :placeholder="t('channelEditor.rateLimit.window.placeholder')"
            @update:model-value="updateField('rateLimitWindowMinutes', $event)"
          />
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.rateLimit.window.hint') }}</p>
        </div>
        <div class="space-y-1">
          <Label class="text-[10px] font-medium text-muted-foreground/80">{{ t('channelEditor.rateLimit.maxConcurrent.label') }}</Label>
          <Input
            :model-value="form.rateLimitMaxConcurrent"
            type="number"
            class="h-9 text-xs"
            :placeholder="t('channelEditor.rateLimit.maxConcurrent.placeholder')"
            @update:model-value="updateField('rateLimitMaxConcurrent', $event)"
          />
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.rateLimit.maxConcurrent.hint') }}</p>
        </div>
      </div>
    </section>
  </div>
</template>
