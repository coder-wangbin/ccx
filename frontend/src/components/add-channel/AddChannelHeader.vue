<template>
  <v-card-title class="d-flex align-center ga-3 pa-6" :class="headerClasses">
    <v-avatar :color="avatarColor" variant="flat" size="40">
      <v-icon :style="headerIconStyle" size="20">{{ isEditing ? 'mdi-pencil' : 'mdi-plus' }}</v-icon>
    </v-avatar>

    <div class="flex-grow-1 modal-header-text">
      <div class="modal-title">
        {{ isEditing ? editTitle : createTitle }}
      </div>
      <div class="modal-subtitle" :class="subtitleClasses">
        {{ isEditing ? editSubtitle : createSubtitle }}
      </div>
    </div>

    <div v-if="isEditing && channelType !== 'images'" class="header-capability-actions">
      <v-tooltip location="bottom" :text="visionTooltip" :open-delay="150" content-class="key-tooltip">
        <template #activator="{ props: tip }">
          <v-btn
            v-bind="tip"
            :color="noVision ? 'warning' : undefined"
            :variant="noVision ? 'tonal' : 'text'"
            size="small"
            icon
            rounded="lg"
            class="mr-2"
            @click="$emit('toggle-no-vision')"
          >
            <v-icon size="18">{{ noVision ? 'mdi-eye-off' : 'mdi-eye' }}</v-icon>
          </v-btn>
        </template>
      </v-tooltip>

      <v-btn
        color="success"
        variant="flat"
        size="small"
        prepend-icon="mdi-test-tube"
        class="capability-test-btn"
        @click="$emit('test-capability')"
      >
        {{ testCapabilityLabel }}
      </v-btn>
    </div>
  </v-card-title>
</template>

<script setup lang="ts">
import type { StyleValue } from 'vue'

type ClassBinding = string | Record<string, boolean> | Array<string | Record<string, boolean>>

interface Props {
  isEditing: boolean
  channelType?: 'messages' | 'chat' | 'responses' | 'gemini' | 'images'
  noVision?: boolean
  headerClasses?: string | Record<string, boolean> | Array<string | Record<string, boolean>>
  avatarColor?: string
  headerIconStyle?: Record<string, string>
  subtitleClasses?: string | Record<string, boolean> | Array<string | Record<string, boolean>>
  editTitle?: string
  createTitle?: string
  editSubtitle?: string
  createSubtitle?: string
  testCapabilityLabel?: string
  visionTooltip?: string
}

withDefaults(defineProps<Props>(), {
  channelType: 'messages',
  noVision: false,
  avatarColor: 'primary',
})

defineEmits<{
  'toggle-no-vision': []
  'test-capability': []
}>()
</script>
