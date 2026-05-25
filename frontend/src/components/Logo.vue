<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  size?: number | string
  animated?: boolean
}>(), {
  size: 32,
  animated: false
})

const sizeStyle = computed(() => {
  const s = typeof props.size === 'number' ? `${props.size}px` : props.size
  return { width: s, height: s }
})
</script>

<template>
  <div class="ccx-logo-container" :style="sizeStyle">
    <svg
      viewBox="0 0 100 100"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      class="ccx-logo-svg"
    >
      <defs>
        <!-- 主流光渐变线 -->
        <linearGradient id="web-logo-grad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#3b82f6" />   <!-- CCX Blue -->
          <stop offset="50%" stop-color="#6366f1" />  <!-- Indigo -->
          <stop offset="100%" stop-color="#10b981" /> <!-- Emerald Green -->
        </linearGradient>

        <!-- 高斯发光滤镜 -->
        <filter id="web-glow" x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation="3" result="blur" />
          <feComposite in="SourceGraphic" in2="blur" operator="over" />
        </filter>
      </defs>

      <!-- 1. 外部数据循环轨道 - 缩收半径至 38px 防止物理贴边裁剪 -->
      <circle
        cx="50"
        cy="50"
        r="38"
        stroke="url(#web-logo-grad)"
        stroke-width="2.2"
        stroke-dasharray="10 6 3 6"
        class="ccx-orbit"
        :class="{ 'animate-spin-slow': animated }"
      />

      <!-- 2. "C" 字型分流弧线 -->
      <path
        d="M 32 24 C 18 32, 18 68, 32 76"
        stroke="url(#web-logo-grad)"
        stroke-width="5.5"
        stroke-linecap="round"
        class="ccx-path"
      />

      <!-- 3. "X" 字型路由交叉射束 -->
      <path
        d="M 72 24 L 50 50 L 72 76"
        stroke="url(#web-logo-grad)"
        stroke-width="5.5"
        stroke-linecap="round"
        class="ccx-path"
      />

      <!-- 4. 反向贯穿路径 -->
      <path
        d="M 50 50 L 36 36"
        stroke="url(#web-logo-grad)"
        stroke-width="5.5"
        stroke-linecap="round"
        class="ccx-path"
      />
      <path
        d="M 50 50 L 36 64"
        stroke="url(#web-logo-grad)"
        stroke-width="5.5"
        stroke-linecap="round"
        class="ccx-path"
      />

      <!-- 5. 核心高能 AI 路由核 -->
      <g :class="{ 'animate-pulse-slow': animated }" filter="url(#web-glow)">
        <circle
          cx="50"
          cy="50"
          r="9"
          fill="url(#web-logo-grad)"
          class="ccx-core-glow"
        />
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
.ccx-logo-container {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.ccx-logo-svg {
  width: 100%;
  height: 100%;
}

.ccx-orbit {
  opacity: 0.55; /* 上调透明度，确保在明色（黄油白底）与暗色应用栏下都具备清晰的轮廓辨识度 */
}

.ccx-path {
  opacity: 0.9;
}

.ccx-core-glow {
  opacity: 0.45;
}

/* 外部轨道旋转动效 */
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.animate-spin-slow {
  transform-origin: 50px 50px;
  animation: spin 22s linear infinite;
}

/* 核心呼吸灯动效 */
@keyframes pulse {
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

.animate-pulse-slow {
  animation: pulse 2.2s infinite ease-in-out;
}
</style>
