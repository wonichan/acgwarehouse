<script setup lang="ts">
import { ref } from 'vue'
import type { ArtItem, CarouselSlide } from '@/types'
import Carousel from '@/components/Carousel.vue'
import ArtCard from '@/components/ArtCard.vue'

// Mock data for carousel
const carouselSlides: CarouselSlide[] = [
  {
    id: 'sakura-rain',
    title: '樱雨街角合集',
    description: '雨景、制服与夜色标签正在快速增长，适合收藏氛围向角色参考。',
    tag: '热度上升',
    tagType: 'hot',
    score: 4.8,
    favorites: 12,
    artVariant: 'rain'
  },
  {
    id: 'blue-station',
    title: '蓝色车站候车室',
    description: '社区本周高频收藏的静态场景，适合整理通勤、雨夜与室内光源参考。',
    tag: '场景焦点',
    tagType: 'normal',
    score: 4.6,
    favorites: 8,
    artVariant: 'character'
  },
  {
    id: 'orange-ear',
    title: '橙发猫耳头像设定',
    description: '高评分头像参考集中在发色、饰品和表情差分，方便快速加入收藏夹。',
    tag: '头像精选',
    tagType: 'normal',
    score: 4.9,
    favorites: 21,
    artVariant: 'album'
  }
]

// Mock data for gallery
const artItems: ArtItem[] = [
  { id: 'sakura-rain', title: '樱雨街角的夜间补光', tags: ['#雨景', '#4.8分'], score: 4.8, favorites: 12, previewVariant: 'tall' },
  { id: 'blue-station', title: '蓝色车站候车室', tags: ['#场景', '#4.6分'], score: 4.6, favorites: 8, previewVariant: 'wide' },
  { id: 'orange-ear', title: '橙发猫耳头像设定', tags: ['#头像', '#4.9分'], score: 4.9, favorites: 21, previewVariant: 'default' },
  { id: 'paper-sky', title: '纸飞机与黄昏天空', tags: ['#暖色', '#4.4分'], score: 4.4, favorites: 6, previewVariant: 'tall' },
  { id: 'library', title: '图书馆靠窗座位', tags: ['#室内', '#4.7分'], score: 4.7, favorites: 14, previewVariant: 'default' },
  { id: 'summer', title: '夏日社团海报底图', tags: ['#海报', '#4.5分'], score: 4.5, favorites: 9, previewVariant: 'wide' },
  { id: 'coffee', title: '咖啡店窗边回头瞬间', tags: ['#角色', '#4.8分'], score: 4.8, favorites: 18, previewVariant: 'tall' },
  { id: 'star', title: '星轨背景练习稿', tags: ['#背景', '#4.3分'], score: 4.3, favorites: 5, previewVariant: 'default' }
]

const activeFilter = ref('推荐')

const filters = ['推荐', '最新', '高分', '可商用标记']
</script>

<template>
  <main>
    <!-- Hero Section -->
    <section class="section hero" data-od-id="gallery-hero">
      <div class="container hero-grid">
        <div>
          <p class="eyebrow">社区图库 · 瀑布流浏览</p>
          <h1>发现值得收藏的二次元灵感图。</h1>
          <p class="lead">按标签、评分和热度浏览社区投稿，把喜欢的作品快速加入收藏夹，后续再统一打标整理。</p>
          <div class="hero-actions">
            <RouterLink class="btn btn-primary" to="/search">开始智能搜索</RouterLink>
            <RouterLink class="btn btn-secondary" to="/trending">查看今日热榜</RouterLink>
          </div>
          <div class="kicker-row" aria-label="热门标签">
            <span class="pill is-active">全部</span>
            <span class="pill">角色设计</span>
            <span class="pill">场景</span>
            <span class="pill">头像</span>
            <span class="pill">高评分</span>
          </div>
        </div>
        <Carousel :slides="carouselSlides" />
      </div>
    </section>

    <!-- Gallery Section -->
    <section class="section" data-od-id="gallery-feed">
      <div class="container">
        <div class="panel toolbar">
          <div class="row" role="list" aria-label="内容筛选">
            <button
              v-for="filter in filters"
              :key="filter"
              class="btn"
              :class="filter === activeFilter ? 'btn-primary btn-small' : 'btn-secondary btn-small'"
              @click="activeFilter = filter"
            >
              {{ filter }}
            </button>
          </div>
          <div class="row">
            <span class="meta">点击卡片进入批量选择</span>
            <RouterLink class="btn btn-secondary btn-small" to="/detail">查看示例详情</RouterLink>
          </div>
        </div>

        <div class="masonry" aria-label="图库瀑布流">
          <ArtCard v-for="item in artItems" :key="item.id" :item="item" />
        </div>
      </div>
    </section>
  </main>
</template>