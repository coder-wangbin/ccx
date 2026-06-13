<script setup lang="ts">
import type { TabsRootEmits, TabsRootProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { computed } from "vue"
import { reactiveOmit } from "@vueuse/core"
import { TabsRoot, useForwardPropsEmits } from "reka-ui"
import { cn } from "@/lib/utils"

const props = defineProps<TabsRootProps & { class?: HTMLAttributes["class"] }>()
const emits = defineEmits<TabsRootEmits>()

const delegatedProps = reactiveOmit(props, "class")
const forwarded = useForwardPropsEmits(delegatedProps, emits)

// orientation="vertical" 时 TabsRoot 需要 flex-row 才能让左右并排
const orientationClass = computed(() =>
  props.orientation === 'vertical' ? 'flex-row' : 'flex-col'
)
</script>

<template>
  <TabsRoot
    v-slot="slotProps"
    data-slot="tabs"
    v-bind="forwarded"
    :class="cn('flex gap-2', orientationClass, props.class)"
  >
    <slot v-bind="slotProps" />
  </TabsRoot>
</template>
