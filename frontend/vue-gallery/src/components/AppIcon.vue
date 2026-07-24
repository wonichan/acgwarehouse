<script setup lang="ts">
import { computed, type Component } from 'vue'

const props = withDefaults(defineProps<{
  icon: Component
  size?: number
  strokeWidth?: number
  label?: string
}>(), {
  size: 20,
  strokeWidth: 2,
})

const a11y = computed(() => {
  if (props.label) {
    return { role: 'img' as const, 'aria-label': props.label }
  }
  return { 'aria-hidden': true as const }
})
</script>

<template>
  <component
    :is="icon"
    class="app-icon"
    :size="size"
    :stroke-width="strokeWidth"
    v-bind="a11y"
  />
</template>

<style scoped>
.app-icon {
  flex-shrink: 0;
  display: block;
}
</style>
