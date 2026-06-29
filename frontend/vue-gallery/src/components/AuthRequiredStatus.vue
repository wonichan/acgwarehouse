<script setup lang="ts">
defineProps<{
  message: string
}>()
</script>

<template>
  <div class="status status--error is-visible auth-status" role="alert">
    <span>{{ message }}</span>
    <RouterLink class="btn btn-secondary btn-small" to="/account">去登录</RouterLink>
  </div>
</template>

<style scoped>
.auth-status {
  margin-top: var(--space-4);
  justify-content: center;
}
</style>
