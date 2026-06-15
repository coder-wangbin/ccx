<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Plus, Trash2 } from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'

interface HeaderRow {
  id: number
  key: string
  value: string
}

const props = defineProps<{
  headerRows: HeaderRow[]
  newHeader: HeaderRow
}>()

const emit = defineEmits<{
  'update:newHeader': [value: Partial<HeaderRow>]
  'addHeaderRow': []
  'removeHeaderRow': [id: number]
  'updateHeaderRow': [id: number, field: 'key' | 'value', value: string]
}>()

const { tf } = useLanguage()
</script>

<template>
  <section class="space-y-4 rounded-xl border border-border/60 bg-card/40 p-5 shadow-xs">
    <div class="border-b border-border/40 pb-2">
      <h4 class="text-xs font-bold uppercase tracking-wider text-primary">
        {{ tf('addChannel.customHeadersLabel', '自定义请求头 (可选)') }}
      </h4>
      <p class="mt-1 text-[10px] leading-4 text-muted-foreground">
        {{ tf('addChannel.customHeadersHint', '添加或覆盖发送到上游的 HTTP 请求头') }}
      </p>
    </div>

    <div v-if="headerRows.length" class="space-y-2">
      <div
        v-for="row in headerRows"
        :key="row.id"
        class="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-xs shadow-2xs transition-colors hover:bg-background"
      >
        <div class="min-w-0 flex-1 truncate">
          <code class="font-mono text-foreground">{{ row.key }}</code>
          <span class="mx-1 text-muted-foreground">:</span>
          <span class="text-muted-foreground break-all">{{ row.value }}</span>
        </div>
        <Button
          type="button"
          size="icon-sm"
          variant="ghost"
          class="h-8 w-8 shrink-0 text-destructive hover:bg-destructive/10"
          @click="emit('removeHeaderRow', row.id)"
        >
          <Trash2 class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <div class="grid gap-2 md:grid-cols-[1fr_2fr_auto] md:items-end">
      <div class="space-y-1">
        <Label class="text-xs font-semibold text-muted-foreground">
          {{ tf('addChannel.headerNameLabel', 'Header 名称') }}
        </Label>
        <Input
          :model-value="newHeader.key"
          class="h-9 w-full font-mono text-xs"
          :placeholder="tf('addChannel.headerNameLabel', 'Header 名称')"
          @update:model-value="(val) => emit('update:newHeader', { key: val as string })"
          @keydown.enter.prevent="emit('addHeaderRow')"
        />
      </div>
      <div class="space-y-1">
        <Label class="text-xs font-semibold text-muted-foreground">
          {{ tf('addChannel.headerValueLabel', '请求头值') }}
        </Label>
        <Input
          :model-value="newHeader.value"
          class="h-9 w-full font-mono text-xs"
          :placeholder="tf('addChannel.headerValueLabel', '请求头值')"
          @update:model-value="(val) => emit('update:newHeader', { value: val as string })"
          @keydown.enter.prevent="emit('addHeaderRow')"
        />
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        class="h-9 justify-self-start px-3.5 shadow-3xs md:justify-self-auto"
        :disabled="!newHeader.key.trim() || !newHeader.value.trim()"
        @click="emit('addHeaderRow')"
      >
        <Plus class="h-4 w-4" />
      </Button>
    </div>
  </section>
</template>
