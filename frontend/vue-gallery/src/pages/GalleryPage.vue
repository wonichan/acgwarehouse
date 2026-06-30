<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import type { ArtItem, CarouselSlide } from '@/types'
import Carousel from '@/components/Carousel.vue'
import ArtCard from '@/components/ArtCard.vue'
import DailyRecommendations from '@/components/DailyRecommendations.vue'
import { getImages, ApiError } from '@/api/client'
import type { ImageQuery, ImageSort, SortOrder } from '@/api/client'
import { useDailyRecommendations } from '@/composables/useDailyRecommendations'
import { appendArtItems, getCommunityFocusSlides } from '@/utils/galleryPresentation'
import { imageToArtItem } from '@/utils/imagePresentation'

const GALLERY_PAGE_SIZE = 20
const INFINITE_SCROLL_ROOT_MARGIN = '480px 0px'

const loading = ref(true)
const loadingMore = ref(false)
const error = ref<string | null>(null)
const nextPageError = ref<string | null>(null)
const dailyRecommendations = useDailyRecommendations()
const artItems = ref<ArtItem[]>([])
const carouselSlides = ref<CarouselSlide[]>([])
const currentPage = ref(1)
const totalItems = ref(0)
const hasMore = ref(false)
const gallerySentinel = ref<HTMLElement | null>(null)
let galleryObserver: IntersectionObserver | null = null

const filters = ['推荐', '最新', '高分参考', '收藏热度'] as const

type GalleryFilter = (typeof filters)[number]

const activeFilter = ref<GalleryFilter>('推荐')

const gallerySortByFilter: Record<GalleryFilter, { readonly sort: ImageSort, readonly order: SortOrder }> = {
  推荐: { sort: 'view_count', order: 'desc' },
  最新: { sort: 'created_at', order: 'desc' },
  高分参考: { sort: 'avg_score', order: 'desc' },
  收藏热度: { sort: 'favorite_count', order: 'desc' },
}

function updatePagination(page: number, total: number): void {
  currentPage.value = page
  totalItems.value = total
  hasMore.value = artItems.value.length < total
}

function galleryImageQuery(page: number): ImageQuery {
  const sort = gallerySortByFilter[activeFilter.value] ?? gallerySortByFilter['推荐']
  return {
    page,
    limit: GALLERY_PAGE_SIZE,
    sort: sort.sort,
    order: sort.order,
  }
}

async function loadNextGalleryPage(): Promise<void> {
  if (loading.value || loadingMore.value || !hasMore.value) return

  loadingMore.value = true
  nextPageError.value = null

  try {
    const nextPage = currentPage.value + 1
    const imagesData = await getImages(galleryImageQuery(nextPage))
    const nextItems = imagesData.items.map(imageToArtItem)
    artItems.value = appendArtItems(artItems.value, nextItems)
    updatePagination(imagesData.page, imagesData.total)
  } catch (e) {
    if (e instanceof ApiError) {
      nextPageError.value = e.message
    } else {
      nextPageError.value = '继续加载失败，请稍后重试'
    }
  } finally {
    loadingMore.value = false
  }
}

function observeGallerySentinel(): void {
  if (!gallerySentinel.value || galleryObserver !== null) return

  galleryObserver = new IntersectionObserver(
    entries => {
      if (entries.some(entry => entry.isIntersecting)) {
        void loadNextGalleryPage()
      }
    },
    { rootMargin: INFINITE_SCROLL_ROOT_MARGIN }
  )
  galleryObserver.observe(gallerySentinel.value)
}

async function loadGallery() {
  loading.value = true
  error.value = null
  nextPageError.value = null
  hasMore.value = false
  dailyRecommendations.reset()

  try {
    dailyRecommendations.loading.value = true
    const dailyResult = dailyRecommendations.load()
    const [imagesData, slides] = await Promise.all([
      getImages(galleryImageQuery(1)),
      getCommunityFocusSlides(),
    ])
    artItems.value = imagesData.items.map(imageToArtItem)
    updatePagination(imagesData.page, imagesData.total)
    carouselSlides.value = slides
    await dailyResult
    dailyRecommendations.loading.value = false
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '加载失败，请刷新重试'
    }
  } finally {
    loading.value = false
    dailyRecommendations.loading.value = false
  }
}

async function handleFilter(filter: GalleryFilter) {
  activeFilter.value = filter
  await loadGallery()
}

async function loadInitialGallery(): Promise<void> {
  await loadGallery()
  await nextTick()
  observeGallerySentinel()
}

onMounted(() => {
  void loadInitialGallery()
})

onBeforeUnmount(() => {
  galleryObserver?.disconnect()
  galleryObserver = null
})
</script>

<template>
  <main>
    <section class="section hero" data-od-id="gallery-hero">
      <div class="container hero-grid">
        <div>
          <p class="eyebrow">社区图库 · 瀑布流浏览</p>
          <h1>发现值得收藏的二次元灵感图。</h1>
          <p class="lead">按真实后端图片、评分和热榜缓存浏览社区投稿，把喜欢的作品快速加入后续收藏夹流程。</p>
          <div class="hero-actions">
            <RouterLink class="btn btn-primary" to="/search">开始智能搜索</RouterLink>
            <RouterLink class="btn btn-secondary" to="/trending">查看今日热榜</RouterLink>
          </div>
          <div class="kicker-row" aria-label="热门标签">
            <span class="pill is-active">全部</span>
            <span class="pill">真实图片</span>
            <span class="pill">热榜缓存</span>
            <span class="pill">百分制评分</span>
            <span class="pill">高评分</span>
          </div>
        </div>
        <Carousel v-if="carouselSlides.length > 0" :slides="carouselSlides" />
        <aside v-else class="panel panel-raised community-carousel-panel community-carousel-empty">
          <p class="eyebrow">本周社区焦点</p>
          <template v-if="loading">
            <h3>正在加载本周精选</h3>
            <p class="meta">正在读取本周热榜作品，完成后会自动展示前 10 张有效图片。</p>
          </template>
          <template v-else-if="error">
            <h3>社区焦点加载失败</h3>
            <p class="meta">{{ error }}</p>
            <button class="btn btn-secondary btn-small" type="button" @click="loadGallery">重试</button>
          </template>
          <template v-else>
            <h3>本周暂无焦点作品</h3>
            <p class="meta">社区活动积累后会自动生成每周精选。</p>
            <RouterLink class="btn btn-secondary btn-small" to="/trending">查看今日热榜</RouterLink>
          </template>
        </aside>
      </div>
    </section>

    <DailyRecommendations
      :items="dailyRecommendations.items.value"
      :loading="dailyRecommendations.loading.value"
      :error="dailyRecommendations.error.value"
      @retry="dailyRecommendations.load"
    />

    <section class="section" data-od-id="gallery-feed">
      <div class="container">
        <div class="panel toolbar">
          <div class="row" role="list" aria-label="内容筛选">
            <button
              v-for="filter in filters"
              :key="filter"
              class="btn"
              type="button"
              :class="filter === activeFilter ? 'btn-primary btn-small' : 'btn-secondary btn-small'"
              @click="handleFilter(filter)"
            >
              {{ filter }}
            </button>
          </div>
          <div class="row">
            <span class="meta">{{ loading ? '加载中...' : `已展示 ${artItems.length} / ${totalItems} 张作品` }}</span>
            <RouterLink v-if="artItems.length > 0" class="btn btn-secondary btn-small" :to="`/detail?id=${artItems[0]?.id}`">查看最新详情</RouterLink>
          </div>
        </div>

        <div v-if="error" class="panel">
          <p class="meta">{{ error }}</p>
          <button class="btn btn-secondary" type="button" @click="loadGallery">重试</button>
        </div>

        <template v-else>
          <div class="masonry" aria-label="图库瀑布流">
            <ArtCard v-for="item in artItems" :key="item.id" :item="item" />
          </div>
          <div ref="gallerySentinel" class="panel" aria-live="polite">
            <p v-if="loadingMore" class="meta">正在加载更多作品...</p>
            <template v-else-if="nextPageError">
              <p class="meta">{{ nextPageError }}</p>
              <button class="btn btn-secondary btn-small" type="button" @click="loadNextGalleryPage">重试加载更多</button>
            </template>
            <p v-else-if="hasMore" class="meta">继续向下滚动会自动加载更多作品。</p>
            <p v-else-if="artItems.length > 0" class="meta">已加载全部作品。</p>
            <p v-else class="meta">暂无可展示作品。</p>
          </div>
        </template>
      </div>
    </section>
  </main>
</template>
