<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { LocationQueryValue } from 'vue-router'
import {
  Bookmark,
  Flame,
  Menu,
  Search,
  UserRound,
  X,
  Images,
} from 'lucide-vue-next'
import { useAuth } from '@/composables/useAuth'
import AppIcon from '@/components/AppIcon.vue'

const route = useRoute()
const router = useRouter()
const { user, isLoggedIn } = useAuth()
const quickSearchKeyword = ref('')
const menuOpen = ref(false)

const currentPath = computed(() => route.path)
const accountLabel = computed(() => {
  if (!isLoggedIn.value) return '登录'
  return user.value?.nickname || user.value?.username || '我的'
})

const navItems = [
  { path: '/', label: '图库', icon: Images },
  { path: '/search', label: '搜索', icon: Search },
  { path: '/trending', label: '热榜', icon: Flame },
  { path: '/collections', label: '收藏夹', icon: Bookmark },
  { path: '/account', label: '我的', icon: UserRound },
] as const

function queryKeyword(value: LocationQueryValue | LocationQueryValue[]): string {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string') ?? ''
  }
  return value ?? ''
}

function isNavActive(path: string): boolean {
  if (path === '/') return currentPath.value === '/'
  return currentPath.value === path || currentPath.value.startsWith(`${path}/`)
}

function closeMenu(): void {
  menuOpen.value = false
}

function toggleMenu(): void {
  menuOpen.value = !menuOpen.value
}

function onDocumentKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && menuOpen.value) {
    closeMenu()
  }
}

watch(
  () => [route.path, route.query.q] as const,
  ([path, value]) => {
    closeMenu()
    if (path === '/search') {
      quickSearchKeyword.value = queryKeyword(value).trim()
    }
  },
  { immediate: true },
)

watch(menuOpen, (open) => {
  if (open) {
    document.addEventListener('keydown', onDocumentKeydown)
    return
  }
  document.removeEventListener('keydown', onDocumentKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onDocumentKeydown)
})

async function handleQuickSearch(): Promise<void> {
  const q = quickSearchKeyword.value.trim()
  closeMenu()
  await router.push(q.length > 0 ? { path: '/search', query: { q } } : { path: '/search' })
}
</script>

<template>
  <header class="topnav">
    <div class="container nav-inner">
      <RouterLink class="brand" to="/" aria-label="ACGWarehouse 首页" @click="closeMenu">
        <span class="brand-mark">AW</span>
        <span>ACGWarehouse</span>
      </RouterLink>

      <nav class="nav-links" aria-label="主导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          class="nav-link"
          :class="{ 'is-active': isNavActive(item.path) }"
          :to="item.path"
          :aria-current="isNavActive(item.path) ? 'page' : undefined"
        >
          <AppIcon :icon="item.icon" :size="16" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="nav-actions">
        <form class="nav-search" role="search" @submit.prevent="handleQuickSearch">
          <label class="nav-search-field">
            <AppIcon :icon="Search" :size="16" />
            <span class="sr-only">快速搜索</span>
            <input
              v-model="quickSearchKeyword"
              class="search-mini"
              placeholder="搜索标签、文件名"
            />
          </label>
        </form>
        <RouterLink class="btn btn-secondary btn-small nav-account" to="/account" @click="closeMenu">
          <AppIcon :icon="UserRound" :size="16" />
          <span>{{ accountLabel }}</span>
        </RouterLink>
        <button
          class="btn btn-secondary btn-small nav-menu-toggle"
          type="button"
          :aria-expanded="menuOpen"
          aria-controls="mobile-nav-panel"
          :aria-label="menuOpen ? '关闭导航菜单' : '打开导航菜单'"
          @click="toggleMenu"
        >
          <AppIcon :icon="menuOpen ? X : Menu" :size="18" />
        </button>
      </div>
    </div>

    <div
      id="mobile-nav-panel"
      class="mobile-nav"
      :class="{ 'is-open': menuOpen }"
      :hidden="!menuOpen"
      :inert="!menuOpen || undefined"
    >
      <div class="container mobile-nav-inner">
        <form class="mobile-search" role="search" @submit.prevent="handleQuickSearch">
          <label class="nav-search-field nav-search-field--block">
            <AppIcon :icon="Search" :size="16" />
            <span class="sr-only">快速搜索</span>
            <input
              v-model="quickSearchKeyword"
              class="search-mini search-mini--mobile"
              placeholder="搜索标签、文件名"
            />
          </label>
        </form>
        <nav class="mobile-nav-links" aria-label="移动主导航">
          <RouterLink
            v-for="item in navItems"
            :key="`mobile-${item.path}`"
            class="mobile-nav-link"
            :class="{ 'is-active': isNavActive(item.path) }"
            :to="item.path"
            :aria-current="isNavActive(item.path) ? 'page' : undefined"
            @click="closeMenu"
          >
            <AppIcon :icon="item.icon" :size="18" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </nav>
      </div>
    </div>
  </header>
</template>
