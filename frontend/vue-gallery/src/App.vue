<script setup lang="ts">
import AppHeader from '@/components/AppHeader.vue'
import AppFooter from '@/components/AppFooter.vue'
import Toast from '@/components/Toast.vue'
import BatchPanel from '@/components/BatchPanel.vue'
</script>

<template>
  <div class="shell">
    <AppHeader />
    <RouterView v-slot="{ Component }">
      <KeepAlive :include="['GalleryPage']">
        <component :is="Component" />
      </KeepAlive>
    </RouterView>
    <BatchPanel />
    <Toast />
    <AppFooter />
  </div>
</template>
