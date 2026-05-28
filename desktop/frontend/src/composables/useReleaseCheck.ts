import { ref, onMounted, onBeforeUnmount } from 'vue'
import { CheckLatestRelease } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'
import type { ReleaseCheckResult } from '@bindings/github.com/BenedictKing/ccx/desktop/models'

// 4 小时与 Go 端 releasesCacheTTL 对齐：组件层面也只需 4h 触发一次。
// 进程内即便多个组件订阅，调用 CheckLatestRelease(false) 时后端缓存会立刻返回，
// 不会真正打 GitHub。
const POLL_INTERVAL_MS = 4 * 60 * 60 * 1000
// 启动后稍稍延后再发起首次检查，避免和服务初始化、健康检查等竞争。
const INITIAL_DELAY_MS = 8 * 1000

const releaseInfo = ref<ReleaseCheckResult | null>(null)
let timer: ReturnType<typeof setInterval> | null = null
let initialTimeout: ReturnType<typeof setTimeout> | null = null
let mountedCount = 0

async function pollOnce(force: boolean) {
  try {
    const result = await CheckLatestRelease(force)
    releaseInfo.value = result
  } catch {
    // 忽略：失败由后端记录日志，前端保持原状
  }
}

export function useReleaseCheck() {
  onMounted(() => {
    mountedCount += 1
    if (mountedCount > 1) return // 已有其他实例驱动轮询，复用结果

    initialTimeout = setTimeout(() => {
      pollOnce(false)
      timer = setInterval(() => pollOnce(false), POLL_INTERVAL_MS)
    }, INITIAL_DELAY_MS)
  })

  onBeforeUnmount(() => {
    mountedCount = Math.max(0, mountedCount - 1)
    if (mountedCount === 0) {
      if (initialTimeout) clearTimeout(initialTimeout)
      if (timer) clearInterval(timer)
      initialTimeout = null
      timer = null
    }
  })

  return {
    releaseInfo,
    refresh: () => pollOnce(true),
  }
}
