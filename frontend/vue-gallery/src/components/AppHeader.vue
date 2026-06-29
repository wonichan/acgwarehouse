<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { LocationQueryValue } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const route = useRoute()
const router = useRouter()
const { user, isLoggedIn } = useAuth()
const quickSearchKeyword = ref('')

const currentPath = computed(() => route.path)
const accountLabel = computed(() => {
  if (!isLoggedIn.value) return '登录'
  return user.value?.nickname || user.value?.username || '我的'
})

const navItems = [
  { path: '/', label: '图库' },
  { path: '/search', label: '搜索' },
  { path: '/trending', label: '热榜' },
  { path: '/collections', label: '收藏夹' },
  { path: '/account', label: '我的' }
] as const

function queryKeyword(value: LocationQueryValue | LocationQueryValue[]): string {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string') ?? ''
  }
  return value ?? ''
}

watch(
  () => [route.path, route.query.q] as const,
  ([path, value]) => {
    if (path === '/search') {
      quickSearchKeyword.value = queryKeyword(value).trim()
    }
  },
  { immediate: true }
)

async function handleQuickSearch(): Promise<void> {
  const q = quickSearchKeyword.value.trim()
  await router.push(q.length > 0 ? { path: '/search', query: { q } } : { path: '/search' })
}
</script>

<template>
  <header class="topnav">
    <div class="container nav-inner">
      <RouterLink class="brand" to="/" aria-label="ACGWarehouse 首页">
        <span class="brand-mark">AW</span>
        <span>ACGWarehouse</span>
      </RouterLink>
      <nav class="nav-links" aria-label="主导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          class="nav-link"
          :class="{ 'is-active': currentPath === item.path }"
          :to="item.path"
          :aria-current="currentPath === item.path ? 'page' : undefined"
        >
          {{ item.label }}
        </RouterLink>
      </nav>
      <form class="nav-actions" role="search" @submit.prevent="handleQuickSearch">
        <input v-model="quickSearchKeyword" class="search-mini" aria-label="快速搜索" placeholder="搜索标签、文件名" />
        <RouterLink class="btn btn-secondary btn-small" to="/account">{{ accountLabel }}</RouterLink>
      </form>
    </div>
  </header>
</template>
