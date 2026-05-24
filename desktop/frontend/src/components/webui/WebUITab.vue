<script setup lang="ts">
import { computed, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Globe, RefreshCw } from 'lucide-vue-next'
import type { DesktopStatus } from '@/types'
import { GetProxyAccessKey, OpenWebUIInBrowser } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

const props = defineProps<{
  status: DesktopStatus
  loading: boolean
}>()

const iframeRef = ref<HTMLIFrameElement | null>(null)

const iframeSrc = computed(() => {
  if (!props.status.url) return ''
  const url = new URL(props.status.url.replace('http://127.0.0.1:', 'http://localhost:'))
  url.searchParams.set('ccx_desktop', '1')
  return url.toString()
})

const postProxyAccessKey = async () => {
  if (!iframeRef.value?.contentWindow || !iframeSrc.value) return
  try {
    const accessKey = await GetProxyAccessKey()
    const targetOrigin = new URL(iframeSrc.value).origin
    iframeRef.value.contentWindow.postMessage(
      { type: 'ccx-desktop-auth', accessKey },
      targetOrigin,
    )
  } catch {
    // Web UI 仍可手动输入 access key
  }
}

const refreshIframe = () => {
  if (!iframeRef.value) return
  iframeRef.value.src = iframeRef.value.src
}

const openInBrowser = async () => {
  try {
    await OpenWebUIInBrowser()
  } catch {
    // handled by parent
  }
}
</script>

<template>
  <div>
    <div v-if="status.running && iframeSrc">
      <div class="flex justify-end mb-2">
        <Button size="sm" variant="secondary" @click="refreshIframe">
          <RefreshCw class="w-3.5 h-3.5 mr-1.5" />
          刷新
        </Button>
      </div>
      <div class="rounded-lg overflow-hidden border border-border" style="min-height: 620px">
      <iframe
        ref="iframeRef"
        :src="iframeSrc"
        class="w-full border-0"
        style="min-height: 620px; background: white"
        title="CCX Web UI"
        @load="postProxyAccessKey"
      />
      </div>
    </div>
    <Card v-else>
      <CardContent class="flex flex-col items-start gap-4 py-8">
        <p class="text-sm text-muted-foreground">CCX 服务尚未启动，无法显示 Web UI。</p>
        <Button size="sm" :disabled="loading" @click="openInBrowser">
          <Globe class="w-4 h-4 mr-1.5" />
          浏览器打开
        </Button>
      </CardContent>
    </Card>
  </div>
</template>
