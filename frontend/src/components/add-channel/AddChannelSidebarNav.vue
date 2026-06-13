<template>
  <nav class="nav-sidebar">
    <div class="nav-sidebar-title">{{ title }}</div>
    <button
      v-for="item in sections"
      :key="item.id"
      class="nav-item"
      :class="{ 'nav-item--active': activeSection === item.id }"
      @click="$emit('navigate', item.id)"
    >
      <v-icon size="20" class="nav-item-icon">{{ item.icon }}</v-icon>
      {{ item.label }}
    </button>
  </nav>
</template>

<script setup lang="ts">
interface NavSection {
  id: string
  icon: string
  label: string
}

interface Props {
  title?: string
  sections: NavSection[]
  activeSection?: string
}

withDefaults(defineProps<Props>(), {
  title: '配置大纲',
  activeSection: '',
})

defineEmits<{
  navigate: [sectionId: string]
}>()
</script>
