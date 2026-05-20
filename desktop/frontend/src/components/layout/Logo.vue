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
  <div class="relative shrink-0 flex items-center justify-center select-none" :style="sizeStyle">
    <svg
      viewBox="0 0 100 100"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      class="w-full h-full"
    >
      <defs>
        <!-- 主流光渐变线 -->
        <linearGradient id="ccx-logo-grad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#3b82f6" />   <!-- CCX Blue -->
          <stop offset="50%" stop-color="#6366f1" />  <!-- Indigo -->
          <stop offset="100%" stop-color="#10b981" /> <!-- Emerald Green -->
        </linearGradient>

        <!-- 极细微高斯发光滤镜 -->
        <filter id="ccx-glow" x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation="3" result="blur" />
          <feComposite in="SourceGraphic" in2="blur" operator="over" />
        </filter>
      </defs>

      <!-- 1. 外部高科技数据循环轨道 (Orbit) -->
      <circle
        cx="50"
        cy="50"
        r="44"
        stroke="url(#ccx-logo-grad)"
        stroke-width="1.5"
        stroke-dasharray="12 8 4 8"
        class="opacity-25"
        :class="{ 'animate-orbit-rotate': animated }"
      />

      <!-- 2. "C" 字型左翼分流弧线 -->
      <path
        d="M 32 24 C 18 32, 18 68, 32 76"
        stroke="url(#ccx-logo-grad)"
        stroke-width="5"
        stroke-linecap="round"
        class="opacity-80"
      />

      <!-- 3. "X" 字型右翼路由交叉交叉射束 -->
      <path
        d="M 72 24 L 50 50 L 72 76"
        stroke="url(#ccx-logo-grad)"
        stroke-width="5"
        stroke-linecap="round"
        class="opacity-80"
      />

      <!-- 4. X 的核心反向贯穿路径 -->
      <path
        d="M 50 50 L 36 36"
        stroke="url(#ccx-logo-grad)"
        stroke-width="5"
        stroke-linecap="round"
        class="opacity-85"
      />
      <path
        d="M 50 50 L 36 64"
        stroke="url(#ccx-logo-grad)"
        stroke-width="5"
        stroke-linecap="round"
        class="opacity-85"
      />

      <!-- 5. 核心高能 AI 智能调度路由核 (Glowing Core Node) -->
      <g :class="{ 'animate-core-pulse': animated }" filter="url(#ccx-glow)">
        <!-- 外层发光晕 -->
        <circle
          cx="50"
          cy="50"
          r="9"
          fill="url(#ccx-logo-grad)"
          class="opacity-40"
        />
        <!-- 内层核心实心点 -->
        <circle
          cx="50"
          cy="50"
          r="5.5"
          fill="#ffffff"
        />
      </g>
    </svg>
  </div>
</template>

<style scoped>
/* 外部虚线轨道极速缓慢旋转 */
@keyframes orbit-spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
.animate-orbit-rotate {
  transform-origin: 50px 50px;
  animation: orbit-spin 25s linear infinite;
}

/* 核心路由节点仿生呼吸脉冲 */
@keyframes core-pulse {
  0%, 100% {
    transform: scale(0.92);
    transform-origin: 50px 50px;
    opacity: 0.85;
  }
  50% {
    transform: scale(1.08);
    transform-origin: 50px 50px;
    opacity: 1;
  }
}
.animate-core-pulse {
  animation: core-pulse 2.5s infinite ease-in-out;
}
</style>
