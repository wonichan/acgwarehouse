import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'gallery',
    component: () => import('@/pages/GalleryPage.vue'),
    meta: { title: 'ACGWarehouse · 图库首页' }
  },
  {
    path: '/detail',
    name: 'detail',
    component: () => import('@/pages/DetailPage.vue'),
    meta: { title: 'ACGWarehouse · 图片详情' }
  },
  {
    path: '/search',
    name: 'search',
    component: () => import('@/pages/SearchPage.vue'),
    meta: { title: 'ACGWarehouse · 智能搜索' }
  },
  {
    path: '/trending',
    name: 'trending',
    component: () => import('@/pages/TrendingPage.vue'),
    meta: { title: 'ACGWarehouse · 热榜' }
  },
  {
    path: '/collections',
    name: 'collections',
    component: () => import('@/pages/CollectionsPage.vue'),
    meta: { title: 'ACGWarehouse · 收藏夹' }
  },
  {
    path: '/collections/:id',
    name: 'collection-detail',
    component: () => import('@/pages/CollectionDetailPage.vue'),
    meta: { title: 'ACGWarehouse · 收藏夹详情' }
  },
  {
    path: '/account',
    name: 'account',
    component: () => import('@/pages/AccountPage.vue'),
    meta: { title: 'ACGWarehouse · 账户中心' }
  }
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})