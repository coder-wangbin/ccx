<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useSetup } from '@/composables/useSetup'
import Logo from '@/components/layout/Logo.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Shield, Eye, EyeOff, Copy, Check, FolderOpen, Info, Sparkles, RefreshCw } from 'lucide-vue-next'

const { setupKey, setupSaving, setupError, envPath, confirmSetup, regenerateKey } = useSetup()

const inputKey = ref('')
const userEdited = ref(false)
const showKey = ref(false)
const copiedTarget = ref<'key' | 'path' | ''>('')
let copiedTimer: ReturnType<typeof setTimeout> | null = null

// setupKey 异步到达时自动同步到表单（仅在用户未编辑过的前提下）
watch(setupKey, (val) => {
  if (!userEdited.value) {
    inputKey.value = val
  }
}, { immediate: true })

const onInput = () => {
  userEdited.value = true
}

const showCopied = (target: 'key' | 'path') => {
  copiedTarget.value = target
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => { copiedTarget.value = '' }, 2000)
}

const handleCopyKey = async () => {
  if (!inputKey.value) return
  try {
    await navigator.clipboard.writeText(inputKey.value)
    showCopied('key')
  } catch { /* ignore */ }
}

const handleCopyPath = async () => {
  if (!envPath.value) return
  try {
    await navigator.clipboard.writeText(envPath.value)
    showCopied('path')
  } catch { /* ignore */ }
}

const canSubmit = computed(() => inputKey.value.trim().length > 0 && !setupSaving.value)

const handleSubmit = () => {
  if (!canSubmit.value) return
  void confirmSetup(inputKey.value)
}

const handleRegenerate = async () => {
  userEdited.value = false
  await regenerateKey()
  inputKey.value = setupKey.value
}

onMounted(() => {
  if (setupKey.value && !inputKey.value) {
    inputKey.value = setupKey.value
  }
})

onBeforeUnmount(() => {
  if (copiedTimer) clearTimeout(copiedTimer)
})
</script>

<template>
  <div class="fixed inset-0 flex flex-col items-center justify-center bg-[#060a13] text-slate-100 font-sans overflow-y-auto p-6" data-wails-drag>
    <div class="w-full max-w-md">
      <Card class="border-slate-900/60 bg-slate-950/40 backdrop-blur-md">
        <CardHeader class="pb-4 text-center">
          <div class="flex justify-center mb-4">
            <Logo :size="48" />
          </div>
          <CardTitle class="text-lg font-bold tracking-wide">CCX Desktop 初始配置</CardTitle>
          <CardDescription class="text-slate-500 leading-relaxed">
            PROXY_ACCESS_KEY 是 AI Agent 通过 CCX 代理访问上游 API 的身份凭证，所有调用方必须持有此密钥。
          </CardDescription>
        </CardHeader>

        <CardContent class="space-y-5">
          <!-- 密钥输入区 -->
          <div class="space-y-1.5">
            <div class="flex items-center justify-between">
              <Label class="text-xs text-muted-foreground">PROXY_ACCESS_KEY</Label>
              <button
                type="button"
                @click="handleRegenerate"
                :disabled="setupSaving"
                class="text-[10px] text-blue-400/70 hover:text-blue-400 flex items-center gap-1 transition-colors disabled:opacity-50 cursor-pointer"
                title="重新生成随机密钥"
              >
                <Sparkles class="w-3 h-3" />
                <span>重新生成</span>
              </button>
            </div>

            <div class="flex gap-2">
              <Input
                v-model="inputKey"
                :type="showKey ? 'text' : 'password'"
                placeholder="ccx-..."
                class="font-mono text-sm h-9"
                @input="onInput"
              />
              <Button type="button" variant="secondary" size="sm" @click="showKey = !showKey" class="shrink-0 px-2.5" :title="showKey ? '隐藏' : '显示'">
                <EyeOff v-if="showKey" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </Button>
              <Button type="button" variant="outline" size="sm" @click="handleCopyKey" :disabled="!inputKey" class="shrink-0 px-2.5" :title="copiedTarget === 'key' ? '已复制' : '复制密钥'">
                <Check v-if="copiedTarget === 'key'" class="w-4 h-4 text-emerald-400" />
                <Copy v-else class="w-4 h-4" />
              </Button>
            </div>
          </div>

          <!-- 数据目录路径 -->
          <div v-if="envPath" class="flex items-start gap-2 rounded-lg border border-slate-800/50 bg-slate-900/30 px-3 py-2">
            <FolderOpen class="w-3.5 h-3.5 text-slate-500 mt-0.5 shrink-0" />
            <div class="min-w-0 flex-1">
              <p class="text-[10px] text-slate-500 mb-0.5">配置文件路径</p>
              <p class="text-xs font-mono text-slate-400 break-all">{{ envPath }}</p>
            </div>
            <button
              type="button"
              @click="handleCopyPath"
              class="text-slate-500 hover:text-slate-400 transition-colors shrink-0 mt-1 cursor-pointer"
              :title="copiedTarget === 'path' ? '已复制' : '复制路径'"
            >
              <Check v-if="copiedTarget === 'path'" class="w-3 h-3 text-emerald-400" />
              <Copy v-else class="w-3 h-3" />
            </button>
          </div>

          <!-- 错误提示 -->
          <Alert v-if="setupError" variant="destructive" class="text-xs">
            <div class="flex items-center gap-2">
              <span class="font-medium">{{ setupError }}</span>
            </div>
          </Alert>

          <!-- 后续可修改提示 -->
          <div class="flex items-start gap-2 text-[11px] text-slate-600">
            <Info class="w-3.5 h-3.5 shrink-0 mt-px" />
            <span>保存后 CCX 将自动启动。后续可在主界面【环境参数】页继续调整其他配置。</span>
          </div>

          <!-- 提交按钮 -->
          <Button
            type="button"
            :disabled="!canSubmit"
            variant="default"
            class="w-full h-10 text-sm font-semibold cursor-pointer"
            @click="handleSubmit"
          >
            <RefreshCw v-if="setupSaving" class="w-4 h-4 animate-spin" />
            <Shield v-else class="w-4 h-4" />
            {{ setupSaving ? '正在保存并启动...' : '完成配置并启动' }}
          </Button>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
