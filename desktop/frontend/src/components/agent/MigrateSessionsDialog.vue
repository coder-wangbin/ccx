<script setup lang="ts">
import { computed, watch, onBeforeUnmount } from 'vue'
import { AlertTriangle, CheckCircle2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { useLanguage } from '@/composables/useLanguage'
import type { MigrateCodexSessionsResult } from '@/types'

const { t } = useLanguage()

const props = defineProps<{
  open: boolean
  loading: boolean
  targetProvider: string
  result: MigrateCodexSessionsResult | null
  error?: string
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const canClose = computed(() => !props.loading)
const hasResult = computed(() => props.result !== null)

const handleKeydown = (e: KeyboardEvent) => {
  if (!props.open) return
  if (e.key === 'Escape' && canClose.value) {
    e.preventDefault()
    emit('cancel')
  } else if (e.key === 'Enter' && !props.loading && !hasResult.value) {
    e.preventDefault()
    emit('confirm')
  }
}

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    window.addEventListener('keydown', handleKeydown)
  } else {
    window.removeEventListener('keydown', handleKeydown)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
        @click.self="canClose && emit('cancel')"
      >
        <div class="w-[min(560px,90vw)] overflow-hidden rounded-2xl border border-border bg-card shadow-2xl">
          <div class="border-b border-border px-6 py-4">
            <div class="flex items-baseline justify-between">
              <h2 class="text-lg font-semibold text-foreground">{{ t('agent.migrateConfirmTitle') }}</h2>
              <div class="text-xs text-muted-foreground">Codex</div>
            </div>
          </div>

          <div class="space-y-4 px-6 py-4">
            <div class="flex items-start gap-2 rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-xs text-yellow-700 dark:text-yellow-300">
              <AlertTriangle class="h-4 w-4 shrink-0 mt-0.5" />
              <p>{{ t('agent.migrateConfirmDesc') }}</p>
            </div>

            <div class="rounded-lg border border-border bg-secondary/30 px-4 py-3">
              <div class="text-xs text-muted-foreground">{{ t('agent.migrateTarget') }}</div>
              <code class="text-sm font-semibold text-foreground">{{ targetProvider }}</code>
            </div>

            <div v-if="loading" class="py-6 text-center text-sm text-muted-foreground">
              {{ t('agent.migrateRunning') }}
            </div>

            <div v-else-if="error" class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive-foreground">
              {{ error }}
            </div>

            <div v-else-if="result" class="space-y-3">
              <div class="flex items-start gap-2 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700 dark:text-emerald-300">
                <CheckCircle2 class="h-4 w-4 shrink-0 mt-0.5" />
                <p>{{ t('agent.migrateResultSuccess', { migrated: String(result.migratedFiles), total: String(result.totalFiles) }) }}</p>
              </div>
              <div class="grid grid-cols-2 gap-2 text-sm">
                <div class="rounded-lg bg-secondary/40 px-3 py-2">
                  <div class="text-xs text-muted-foreground">{{ t('agent.migrateResultSkippedLabel') }}</div>
                  <div class="font-semibold">{{ result.skippedFiles }}</div>
                </div>
                <div class="rounded-lg bg-secondary/40 px-3 py-2">
                  <div class="text-xs text-muted-foreground">{{ t('agent.migrateResultFailedLabel') }}</div>
                  <div class="font-semibold">{{ result.failedFiles }}</div>
                </div>
              </div>
              <div class="rounded-lg bg-secondary/40 px-3 py-2 text-sm">
                <template v-if="result.sqliteSkipped">
                  <div class="font-medium">{{ t('agent.migrateSQLiteSkipped') }}</div>
                  <div v-if="result.sqliteError" class="mt-1 break-all text-xs text-muted-foreground">{{ result.sqliteError }}</div>
                </template>
                <template v-else>
                  {{ t('agent.migrateSQLiteUpdated', { count: String(result.sqliteRowsUpdated) }) }}
                </template>
              </div>
            </div>
          </div>

          <div class="flex justify-end gap-2 border-t border-border px-6 py-4">
            <Button variant="ghost" size="sm" :disabled="loading" @click="emit('cancel')">
              {{ result ? t('agent.migrateClose') : t('agent.diffCancel') }} <span class="ml-1.5 text-xs opacity-60">Esc</span>
            </Button>
            <Button v-if="!result" size="sm" :disabled="loading" @click="emit('confirm')">
              {{ t('agent.migrateConfirm') }} <span class="ml-1.5 text-xs opacity-60">Enter</span>
            </Button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
