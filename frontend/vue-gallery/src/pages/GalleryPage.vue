<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, onActivated, onDeactivated, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import type { ArtItem, CarouselSlide } from '@/types'
import Carousel from '@/components/Carousel.vue'
import ArtCard from '@/components/ArtCard.vue'
import DailyRecommendations from '@/components/DailyRecommendations.vue'
import { getImages, ApiError } from '@/api/client'
import type { ImageQuery, ImageSort, SortOrder } from '@/api/client'
import { useDailyRecommendations } from '@/composables/useDailyRecommendations'
import { appendArtItems, getCommunityFocusSlides } from '@/utils/galleryPresentation'
import { imageToArtItem } from '@/utils/imagePresentation'

defineOptions({ name: 'GalleryPage' })

const GALLERY_PAGE_SIZE = 20
const INFINITE_SCROLL_ROOT_MARGIN = '480px 0px'
const MASONRY_MIN_COLUMN_WIDTH = 240
const MASONRY_MAX_COLUMNS = 4
const MASONRY_COLUMN_GAP = 16
const ART_CARD_BODY_ESTIMATED_HEIGHT = 104
const ART_CARD_FALLBACK_HEIGHTS: Record<ArtItem['previewVariant'], number> = {
  default: 210,
  tall: 320,
  wide: 180,
}

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
const galleryMasonry = ref<HTMLElement | null>(null)
const gallerySentinel = ref<HTMLElement | null>(null)
const masonryColumnCount = ref(1)
const masonryColumns = ref<ArtItem[][]>([[]])
const masonryColumnHeights = ref<number[]>([0])
let galleryObserver: IntersectionObserver | null = null
let masonryResizeObserver: ResizeObserver | null = null
let initialized = false

const filters = ['推荐', '最新', '高分参考', '收藏热度'] as const

type GalleryFilter = (typeof filters)[number]

const activeFilter = ref<GalleryFilter>('推荐')

const gallerySortByFilter: Record<GalleryFilter, { readonly sort: ImageSort, readonly order: SortOrder }> = {
  推荐: { sort: 'view_count', order: 'desc' },
  最新: { sort: 'created_at', order: 'desc' },
  高分参考: { sort: 'avg_score', order: 'desc' },
  收藏热度: { sort: 'favorite_count', order: 'desc' },
}

const route = useRoute()

// activeTag 从路由 query 读取，支持 /?tag=<tag> 深链。
const activeTag = computed<string>(() => {
  const raw = route.query['tag']
  return typeof raw === 'string' ? raw : ''
})

// lastLoadedTag 记录上次加载所用 tag，用于 KeepAlive 恢复时检测 tag 是否变化。
const lastLoadedTag = ref<string>('')

function updatePagination(page: number, total: number): void {
  currentPage.value = page
  totalItems.value = total
  hasMore.value = artItems.value.length < total
}

function emptyMasonryColumns(count: number): ArtItem[][] {
  return Array.from({ length: count }, () => [])
}

function columnWidthFor(count: number): number {
  const containerWidth = galleryMasonry.value?.clientWidth ?? MASONRY_MIN_COLUMN_WIDTH
  const totalGap = Math.max(0, count - 1) * MASONRY_COLUMN_GAP
  return Math.max(MASONRY_MIN_COLUMN_WIDTH, (containerWidth - totalGap) / count)
}

function estimateArtItemHeight(item: ArtItem, columnWidth: number): number {
  if (item.imageWidth !== undefined && item.imageHeight !== undefined && item.imageWidth > 0 && item.imageHeight > 0) {
    return (columnWidth * item.imageHeight / item.imageWidth) + ART_CARD_BODY_ESTIMATED_HEIGHT
  }
  return ART_CARD_FALLBACK_HEIGHTS[item.previewVariant] + ART_CARD_BODY_ESTIMATED_HEIGHT
}

function shortestColumnIndex(heights: readonly number[]): number {
  return heights.reduce((shortest, height, index) => height < heights[shortest] ? index : shortest, 0)
}

function appendMasonryItems(nextItems: readonly ArtItem[]): void {
  const columnCount = masonryColumnCount.value
  const nextColumns = masonryColumns.value.length === columnCount
    ? masonryColumns.value.map(column => [...column])
    : emptyMasonryColumns(columnCount)
  const nextHeights = masonryColumnHeights.value.length === columnCount
    ? [...masonryColumnHeights.value]
    : Array.from({ length: columnCount }, () => 0)
  const itemColumnWidth = columnWidthFor(columnCount)

  for (const item of nextItems) {
    const targetColumn = shortestColumnIndex(nextHeights)
    nextColumns[targetColumn].push(item)
    nextHeights[targetColumn] += estimateArtItemHeight(item, itemColumnWidth) + MASONRY_COLUMN_GAP
  }

  masonryColumns.value = nextColumns
  masonryColumnHeights.value = nextHeights
}

function rebuildMasonryColumns(items: readonly ArtItem[] = artItems.value): void {
  const columnCount = masonryColumnCount.value
  masonryColumns.value = emptyMasonryColumns(columnCount)
  masonryColumnHeights.value = Array.from({ length: columnCount }, () => 0)
  appendMasonryItems(items)
}

function calculateMasonryColumnCount(width: number): number {
  if (width <= 0) return masonryColumnCount.value
  const nextCount = Math.floor((width + MASONRY_COLUMN_GAP) / (MASONRY_MIN_COLUMN_WIDTH + MASONRY_COLUMN_GAP))
  return Math.max(1, Math.min(MASONRY_MAX_COLUMNS, nextCount))
}

function updateMasonryColumnCount(width: number): void {
  const nextColumnCount = calculateMasonryColumnCount(width)
  if (nextColumnCount === masonryColumnCount.value) return
  masonryColumnCount.value = nextColumnCount
  rebuildMasonryColumns()
}

function observeMasonryContainer(): void {
  if (!galleryMasonry.value || masonryResizeObserver !== null) return

  updateMasonryColumnCount(galleryMasonry.value.clientWidth)
  if (typeof ResizeObserver === 'undefined') return

  masonryResizeObserver = new ResizeObserver(entries => {
    const width = entries[0]?.contentRect.width ?? galleryMasonry.value?.clientWidth ?? 0
    updateMasonryColumnCount(width)
  })
  masonryResizeObserver.observe(galleryMasonry.value)
}

function galleryImageQuery(page: number): ImageQuery {
  const sort = gallerySortByFilter[activeFilter.value] ?? gallerySortByFilter['推荐']
  const tag = activeTag.value.trim()
  return {
    page,
    limit: GALLERY_PAGE_SIZE,
    sort: sort.sort,
    order: sort.order,
    ...(tag.length > 0 ? { tag } : {}),
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
    appendMasonryItems(nextItems)
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

async function loadGallery(): Promise<void> {
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
    const firstPageItems = imagesData.items.map(imageToArtItem)
    artItems.value = firstPageItems
    rebuildMasonryColumns(firstPageItems)
    updatePagination(imagesData.page, imagesData.total)
    lastLoadedTag.value = activeTag.value.trim()
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

async function handleFilter(filter: GalleryFilter): Promise<void> {
  activeFilter.value = filter
  await loadGallery()
}

async function loadInitialGallery(): Promise<void> {
  await loadGallery()
  await nextTick()
  observeMasonryContainer()
  observeGallerySentinel()
}

onMounted(() => {
  void loadInitialGallery().then(() => { initialized = true })
})

// 路由 tag 变化时重新加载图库，支持 /?tag=a → /?tag=b 导航。
watch(() => route.query['tag'], () => {
  void loadInitialGallery()
})

onActivated(() => {
  if (!initialized) return
  // KeepAlive 恢复：若 tag 在 deactivated 期间变化（watcher 被暂停），需重新加载。
  if (activeTag.value.trim() !== lastLoadedTag.value) {
    void loadInitialGallery()
    return
  }
  nextTick(() => {
    observeMasonryContainer()
    observeGallerySentinel()
  })
})

onDeactivated(() => {
  // KeepAlive 缓存：断开观察者，保留数据与 DOM
  galleryObserver?.disconnect()
  galleryObserver = null
  masonryResizeObserver?.disconnect()
  masonryResizeObserver = null
})

onBeforeUnmount(() => {
  galleryObserver?.disconnect()
  galleryObserver = null
  masonryResizeObserver?.disconnect()
  masonryResizeObserver = null
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
          <div
            ref="galleryMasonry"
            class="masonry"
            aria-label="图库瀑布流"
            :style="{ '--masonry-columns': String(masonryColumnCount) }"
          >
            <div
              v-for="(column, columnIndex) in masonryColumns"
              :key="`masonry-column-${masonryColumnCount}-${columnIndex}`"
              class="masonry-column"
            >
              <ArtCard v-for="item in column" :key="item.id" :item="item" />
            </div>
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
