<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const { message, isOpen } = useToast()
</script>

<template>
  <div
    class="toast"
    :class="{ 'is-open': isOpen }"
    role="status"
    aria-live="polite"
    aria-atomic="true"
  >
    {{ message }}
  </div>
</template>