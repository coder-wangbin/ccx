<script setup lang="ts">
import Logo from '@/components/layout/Logo.vue'
</script>

<template>
  <div class="fixed inset-0 flex flex-col items-center justify-center bg-[#060a13] text-slate-300 select-none" data-wails-drag>
    <Logo :size="56" />
    <div class="mt-6 flex items-center gap-2 text-xs font-mono tracking-widest text-slate-500 uppercase">
      <span class="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse" />
      <span>正在初始化 CCX 控制台</span>
    </div>
  </div>
</template>
