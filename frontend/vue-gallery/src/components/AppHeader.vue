<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

const currentPath = computed(() => route.path)

const navItems = [
  { path: '/', label: '图库' },
  { path: '/search', label: '搜索' },
  { path: '/trending', label: '热榜' },
  { path: '/collections', label: '收藏夹' },
  { path: '/account', label: '我的' }
]
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
      <div class="nav-actions">
        <input class="search-mini" aria-label="快速搜索" placeholder="搜索标签、文件名" />
        <RouterLink class="btn btn-secondary btn-small" to="/account">登录</RouterLink>
      </div>
    </div>
  </header>
</template>