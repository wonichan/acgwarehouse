<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const props = defineProps<{
  options: string[]
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { show } = useToast()

const select = (option: string) => {
  emit('update:modelValue', option)
  show(`已切换到${option}`)
}
</script>

<template>
  <div class="segment" role="list" aria-label="内容筛选">
    <button
      v-for="option in options"
      :key="option"
      type="button"
      :class="{ 'is-active': modelValue === option }"
      @click="select(option)"
    >
      {{ option }}
    </button>
  </div>
</template>