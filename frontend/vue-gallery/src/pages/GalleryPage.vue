<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { ArtItem, CarouselSlide } from '@/types'
import Carousel from '@/components/Carousel.vue'
import ArtCard from '@/components/ArtCard.vue'
import { getImages, getRankings, ApiError } from '@/api/client'
import type { ImageItem, RankingResponse } from '@/api/client'

// Loading and error states
const loading = ref(true)
const error = ref<string | null>(null)

// Real data from API
const artItems = ref<ArtItem[]>([])
const carouselSlides = ref<CarouselSlide[]>([])

// Filter state
const activeFilter = ref('推荐')
const filters = ['推荐', '最新', '高分', '可商用标记']

// Convert API image to ArtItem for display
function imageToArtItem(img: ImageItem): ArtItem {
  // Determine preview variant based on aspect ratio
  let previewVariant: 'default' | 'tall' | 'wide' = 'default'
  if (img.width && img.height) {
    const ratio = img.width / img.height
    if (ratio < 0.8) previewVariant = 'tall'
    else if (ratio > 1.3) previewVariant = 'wide'
  }
  
  const score = Number.isFinite(img.avg_score) ? img.avg_score : 0
  const category = img.category || '未分类'
  
  return {
    id: String(img.id),
    title: img.filename,
    tags: [category, `${score.toFixed(1)}分`],
    score,
    favorites: img.favorite_count,
    previewVariant,
    imageUrl: img.url,
  }
}

// Convert ranking to carousel slide
function rankingToSlide(ranking: RankingResponse, index: number): CarouselSlide {
  const variants: Array<'rain' | 'character' | 'album'> = ['rain', 'character', 'album']
  const tags = ['热度上升', '场景焦点', '头像精选']
  
  const image = ranking.image
  const score = Number.isFinite(ranking.score) ? ranking.score : 0
  
  return {
    id: String(image.id),
    title: image.filename,
    description: `排名第${ranking.rank}的热门作品，热度分${score.toFixed(1)}`,
    tag: tags[index % tags.length],
    tagType: index === 0 ? 'hot' : 'normal',
    score,
    favorites: ranking.favorite_count,
    artVariant: variants[index % variants.length],
  }
}

// Load gallery data
async function loadGallery() {
  loading.value = true
  error.value = null
  
  try {
    // Load images and rankings in parallel
    const [imagesData, rankingsData] = await Promise.all([
      getImages({ limit: 20 }),
      getRankings(3),
    ])
    
    // Convert to display format
    artItems.value = imagesData.items.map(imageToArtItem)
    carouselSlides.value = rankingsData.map((r, i) => rankingToSlide(r, i))
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '加载失败，请刷新重试'
    }
    console.error('Failed to load gallery:', e)
  } finally {
    loading.value = false
  }
}

// Filter handler
async function handleFilter(filter: string) {
  activeFilter.value = filter
  // In future, call API with different sort params
  await loadGallery()
}

onMounted(() => {
  loadGallery()
})
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
        <Carousel v-if="carouselSlides.length > 0" :slides="carouselSlides" />
        <aside v-else class="panel panel-raised community-carousel-panel">
          <p class="eyebrow">本周社区焦点</p>
          <h3>正在加载社区精选</h3>
          <p class="meta">热榜数据加载完成后会自动显示。</p>
        </aside>
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
              @click="handleFilter(filter)"
            >
              {{ filter }}
            </button>
          </div>
          <div class="row">
            <span class="meta">{{ loading ? '加载中...' : `共 ${artItems.length} 张作品` }}</span>
            <RouterLink class="btn btn-secondary btn-small" to="/detail">查看示例详情</RouterLink>
          </div>
        </div>

        <!-- Error state -->
        <div v-if="error" class="panel">
          <p class="meta">{{ error }}</p>
          <button class="btn btn-secondary" @click="loadGallery">重试</button>
        </div>

        <!-- Gallery grid -->
        <div v-else class="masonry" aria-label="图库瀑布流">
          <ArtCard v-for="item in artItems" :key="item.id" :item="item" />
        </div>
      </div>
    </section>
  </main>
</template>