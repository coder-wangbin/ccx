<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  size?: number | string
  animated?: boolean
}>(), {
  size: 24,
  animated: true
})

const sizeStyle = computed(() => {
  const s = typeof props.size === 'number' ? `${props.size}px` : props.size
  return { width: s, height: s }
})
</script>

<template>
  <div class="ccx-desktop-logo relative shrink-0 flex items-center justify-center select-none" :style="sizeStyle">
    <svg
      viewBox="0 0 100 100"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      class="w-full h-full"
      aria-hidden="true"
    >
      <defs>
        <!-- 终端网关流光渐变线：由 CSS 变量按明暗主题切换 -->
        <linearGradient id="desktop-gateway-flow" x1="18%" y1="28%" x2="82%" y2="72%">
          <stop offset="0%" stop-color="var(--logo-flow-start)" />
          <stop offset="48%" stop-color="var(--logo-flow-mid)" />
          <stop offset="100%" stop-color="var(--logo-flow-end)" />
        </linearGradient>

        <!-- 玻璃面板渐变：亮色用浅底，暗色用深底 -->
        <linearGradient id="desktop-gateway-panel" x1="15%" y1="20%" x2="85%" y2="82%">
          <stop offset="0%" stop-color="var(--logo-panel-start)" stop-opacity="0.95" />
          <stop offset="52%" stop-color="var(--logo-panel-mid)" stop-opacity="0.92" />
          <stop offset="100%" stop-color="var(--logo-panel-end)" stop-opacity="0.95" />
        </linearGradient>

        <radialGradient id="desktop-gateway-bg" cx="70%" cy="70%" r="86%">
          <stop offset="0%" stop-color="var(--logo-bg-start)" />
          <stop offset="40%" stop-color="var(--logo-bg-mid)" />
          <stop offset="100%" stop-color="var(--logo-bg-end)" />
        </radialGradient>

        <filter id="desktop-gateway-glow" x="-28%" y="-28%" width="156%" height="156%">
          <feGaussianBlur stdDeviation="2.2" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>

      <!-- 1. Sidebar 主题自适应圆角底 -->
      <rect x="5" y="5" width="90" height="90" rx="22" fill="url(#desktop-gateway-bg)" />
      <rect x="7.5" y="7.5" width="85" height="85" rx="20" fill="none" stroke="var(--logo-border)" stroke-width="0.9" opacity="0.42" />

      <!-- 2. 玻璃终端窗口 -->
      <rect x="15" y="20" width="70" height="62" rx="14" fill="url(#desktop-gateway-panel)" stroke="var(--logo-panel-border)" stroke-width="1.4" opacity="0.98" />
      <path d="M 19 32 H 81" stroke="var(--logo-divider)" stroke-width="0.8" opacity="0.22" />
      <circle cx="25" cy="26.5" r="2.3" fill="var(--logo-dot-green)" />
      <circle cx="32" cy="26.5" r="2.3" fill="var(--logo-dot-blue)" opacity="0.82" />
      <circle cx="39" cy="26.5" r="2.3" fill="var(--logo-dot-indigo)" opacity="0.82" />

      <!-- 3. 终端网关提示符与 X 路由束 -->
      <g filter="url(#desktop-gateway-glow)" stroke-linecap="round" stroke-linejoin="round">
        <path d="M 28 39 L 42 51 L 28 63" stroke="url(#desktop-gateway-flow)" stroke-width="8" />
        <path d="M 52 38 L 73 64" stroke="url(#desktop-gateway-flow)" stroke-width="8" />
        <path d="M 73 38 L 52 64" stroke="url(#desktop-gateway-flow)" stroke-width="8" />
      </g>

      <!-- 4. 底部网关状态线与在线节点 -->
      <path d="M 22 74 H 50" stroke="var(--logo-status-green)" stroke-width="2.6" stroke-linecap="round" opacity="0.56" />
      <path d="M 55 74 H 68" stroke="var(--logo-status-blue)" stroke-width="2.6" stroke-linecap="round" opacity="0.46" />
      <g :class="{ 'animate-gateway-pulse': animated }">
        <circle cx="76" cy="74" r="2.4" fill="var(--logo-online)" />
        <circle cx="76" cy="74" r="5.5" stroke="var(--logo-online)" stroke-width="1.1" opacity="0.28" />
      </g>
    </svg>
  </div>
</template>

<style scoped>
.ccx-desktop-logo {
  /* 亮色主题为默认值；暗色主题通过继承自 .dark 的全局变量覆盖 */
  --logo-bg-start: var(--desktop-logo-bg-start, #ccfbf1);
  --logo-bg-mid: var(--desktop-logo-bg-mid, #dbeafe);
  --logo-bg-end: var(--desktop-logo-bg-end, #f8fafc);
  --logo-border: var(--desktop-logo-border, #2563eb);
  --logo-panel-start: var(--desktop-logo-panel-start, #ffffff);
  --logo-panel-mid: var(--desktop-logo-panel-mid, #eff6ff);
  --logo-panel-end: var(--desktop-logo-panel-end, #ecfdf5);
  --logo-panel-border: var(--desktop-logo-panel-border, #0284c7);
  --logo-divider: var(--desktop-logo-divider, #0f766e);
  --logo-flow-start: var(--desktop-logo-flow-start, #0369a1);
  --logo-flow-mid: var(--desktop-logo-flow-mid, #2563eb);
  --logo-flow-end: var(--desktop-logo-flow-end, #059669);
  --logo-dot-green: var(--desktop-logo-dot-green, #059669);
  --logo-dot-blue: var(--desktop-logo-dot-blue, #0284c7);
  --logo-dot-indigo: var(--desktop-logo-dot-indigo, #4f46e5);
  --logo-status-green: var(--desktop-logo-status-green, #047857);
  --logo-status-blue: var(--desktop-logo-status-blue, #0369a1);
  --logo-online: var(--desktop-logo-online, #0d9488);
}

:global(.dark) {
  /* 暗色主题：深底 + 蓝靛绿霓虹，呼应 App 图标 */
  --desktop-logo-bg-start: #064e3b;
  --desktop-logo-bg-mid: #082f49;
  --desktop-logo-bg-end: #020617;
  --desktop-logo-border: #93c5fd;
  --desktop-logo-panel-start: #102a56;
  --desktop-logo-panel-mid: #06142a;
  --desktop-logo-panel-end: #042f2e;
  --desktop-logo-panel-border: #93c5fd;
  --desktop-logo-divider: #bae6fd;
  --desktop-logo-flow-start: #38bdf8;
  --desktop-logo-flow-mid: #6366f1;
  --desktop-logo-flow-end: #10b981;
  --desktop-logo-dot-green: #10b981;
  --desktop-logo-dot-blue: #38bdf8;
  --desktop-logo-dot-indigo: #6366f1;
  --desktop-logo-status-green: #10b981;
  --desktop-logo-status-blue: #38bdf8;
  --desktop-logo-online: #5eead4;
}

/* 在线网关节点呼吸脉冲 */
@keyframes gateway-pulse {
  0%, 100% {
    transform: scale(0.92);
    transform-origin: 76px 74px;
    opacity: 0.82;
  }
  50% {
    transform: scale(1.12);
    transform-origin: 76px 74px;
    opacity: 1;
  }
}

.animate-gateway-pulse {
  animation: gateway-pulse 2.4s infinite ease-in-out;
}
</style>
