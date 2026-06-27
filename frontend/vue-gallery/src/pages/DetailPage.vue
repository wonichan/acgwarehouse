<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError, getImage, rateImage } from '@/api/client'
import type { ImageDetailResponse, ImageItem } from '@/api/client'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { useZoom } from '@/composables/useZoom'

const route = useRoute()
const { zoom, zoomIn, zoomOut, reset } = useZoom()
const { show } = useToast()
const { isLoggedIn } = useAuth()

const detail = ref<ImageDetailResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const selectedScore = ref(50)
const savingRating = ref(false)

function firstQueryValue(value: unknown): string | null {
  if (typeof value === 'string') return value
  if (Array.isArray(value)) {
    const first = value[0]
    return typeof first === 'string' ? first : null
  }
  return null
}

const imageId = computed<number | null>(() => {
  const rawValue = firstQueryValue(route.query['id'])
  if (rawValue === null) return null

  const parsed = Number(rawValue)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

const image = computed<ImageItem | null>(() => detail.value?.image ?? null)

const title = computed(() => image.value?.filename ?? '请选择一张作品')
const tags = computed(() => detail.value?.tags ?? [])
const similarImages = computed(() => detail.value?.similar_images ?? [])

function formatScore(value: number): string {
  return Number.isFinite(value) ? value.toFixed(1) : '0.0'
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '未知大小'
  const megabytes = value / 1024 / 1024
  if (megabytes >= 1) return `${megabytes.toFixed(2)} MB`
  return `${(value / 1024).toFixed(1)} KB`
}

function clampRatingScore(value: number | null): number {
  if (value === null || !Number.isFinite(value)) return 50
  return Math.min(100, Math.max(0, Math.round(value)))
}

async function loadDetail(): Promise<void> {
  if (imageId.value === null) {
    detail.value = null
    error.value = null
    loading.value = false
    return
  }

  loading.value = true
  error.value = null

  try {
    const response = await getImage(imageId.value)
    detail.value = response
    selectedScore.value = clampRatingScore(response.my_rating ?? response.avg_score)
  } catch (e) {
    detail.value = null
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '作品详情加载失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

async function handleSaveRating(): Promise<void> {
  if (imageId.value === null) return
  if (!isLoggedIn.value) {
    show('请先登录后再提交评分')
    return
  }

  savingRating.value = true
  try {
    const result = await rateImage(imageId.value, selectedScore.value)
    show(`评分已更新为 ${result.score}/100`)
    await loadDetail()
  } catch (e) {
    if (e instanceof ApiError) {
      show(e.message)
    } else {
      show('评分保存失败，请稍后重试')
    }
  } finally {
    savingRating.value = false
  }
}

function handleFavorite(): void {
  if (!isLoggedIn.value) {
    show('请先登录后再选择收藏夹')
    return
  }
  show('收藏夹选择流程尚未接入，未执行收藏操作')
}

function handleTagging(): void {
  if (!isLoggedIn.value) {
    show('请先登录后再管理标签')
    return
  }
  show('个人标签保存接口尚未接入，未写入标签')
}

function handleDownload(): void {
  show('当前后端未提供下载接口，可打开原图后手动保存')
}

onMounted(() => {
  loadDetail()
})

watch(imageId, () => {
  reset()
  loadDetail()
})
</script>

<template>
  <main>
    <section class="section" data-od-id="detail-viewer">
      <div v-if="imageId === null" class="container">
        <article class="panel panel-raised">
          <p class="eyebrow">作品详情</p>
          <h1>请选择一张作品</h1>
          <p class="lead">详情页需要有效的图片 ID，例如从图库、搜索结果或热榜进入 /detail?id=5149。</p>
          <div class="hero-actions">
            <RouterLink class="btn btn-primary" to="/">返回图库</RouterLink>
            <RouterLink class="btn btn-secondary" to="/search">去搜索作品</RouterLink>
          </div>
        </article>
      </div>

      <div v-else-if="loading" class="container">
        <article class="panel panel-raised">
          <p class="eyebrow">正在加载</p>
          <h1>读取真实作品详情</h1>
          <p class="lead">正在通过 /api/v1/images/{{ imageId }} 获取图片、标签、评分和相似推荐。</p>
        </article>
      </div>

      <div v-else-if="error" class="container">
        <article class="panel panel-raised">
          <p class="eyebrow">加载失败</p>
          <h1>无法展示作品</h1>
          <p class="lead">{{ error }}</p>
          <div class="hero-actions">
            <button class="btn btn-primary" type="button" @click="loadDetail">重试</button>
            <RouterLink class="btn btn-secondary" to="/">返回图库</RouterLink>
          </div>
        </article>
      </div>

      <div v-else-if="detail && image" class="container detail-stage">
        <!-- Viewer Panel -->
        <article class="panel viewer panel-raised" aria-label="图片放大查看器">
          <div class="viewer-art" :style="`--zoom: ${zoom}`" data-viewer-art>
            <img :src="image.url" :alt="image.filename" />
          </div>
          <div class="zoom-controls" aria-label="缩放控制">
            <button class="btn btn-secondary btn-small" type="button" @click="zoomOut">缩小</button>
            <button class="btn btn-primary btn-small" type="button" @click="zoomIn">放大</button>
            <button class="btn btn-secondary btn-small" type="button" @click="reset">复位</button>
          </div>
        </article>

        <!-- Side Panel -->
        <aside class="stack">
          <div class="panel panel-raised">
            <p class="eyebrow">作品详情</p>
            <h1 class="detail-title">{{ title }}</h1>
            <p class="lead">
              {{ image.width }}×{{ image.height }} · {{ formatBytes(image.size) }} · {{ image.category || '未分类' }} · 浏览 {{ image.view_count }} 次
            </p>
            <div class="kicker-row">
              <span class="tag is-hot">{{ formatScore(detail.avg_score) }}/100</span>
              <span class="tag">{{ detail.rating_count }} 评分</span>
              <span class="tag">{{ detail.favorite_count }} 收藏</span>
              <span v-if="detail.is_collected" class="tag">已收藏</span>
              <span v-if="detail.my_rating !== null" class="tag">我的评分 {{ detail.my_rating }}/100</span>
              <span v-for="tag in tags" :key="tag" class="tag">{{ tag }}</span>
            </div>
            <div class="divider"></div>
            <div class="grid-2">
              <button class="btn btn-primary" type="button" @click="handleFavorite">收藏到相册</button>
              <button class="btn btn-secondary" type="button" @click="handleDownload">下载说明</button>
            </div>
          </div>

          <div class="panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">评分与标签</p>
                <h3>按后端百分制评分</h3>
              </div>
              <span class="meta">0-100</span>
            </div>
            <div class="form-grid">
              <label class="field">
                个人评分
                <select class="select" v-model.number="selectedScore">
                  <option :value="100">100 分</option>
                  <option :value="75">75 分</option>
                  <option :value="50">50 分</option>
                  <option :value="25">25 分</option>
                  <option :value="0">0 分</option>
                </select>
              </label>
              <button class="btn btn-primary" type="button" :disabled="savingRating" @click="handleSaveRating">
                {{ savingRating ? '保存中...' : '保存评分' }}
              </button>
              <button class="btn btn-secondary" type="button" @click="handleTagging">标签功能状态</button>
              <p class="meta">标签分配 UI 与个人标签保存接口尚未完整接入，此页不会伪造保存成功。</p>
            </div>
          </div>

          <div class="panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">相似推荐</p>
                <h3>同类作品</h3>
              </div>
              <RouterLink class="btn btn-secondary btn-small" to="/search">更多</RouterLink>
            </div>
            <div v-if="similarImages.length === 0" class="activity-empty">
              <p class="activity-empty__title">暂无相似作品</p>
              <p class="activity-empty__desc">后端返回 similar_images 为空时显示此状态。</p>
            </div>
            <div v-else class="grid-2">
              <RouterLink
                v-for="item in similarImages"
                :key="item.id"
                :to="`/detail?id=${item.id}`"
                :aria-label="`查看${item.filename}详情`"
              >
                <div class="thumb">
                  <img v-if="item.url" :src="item.url" :alt="item.filename" loading="lazy" />
                </div>
              </RouterLink>
            </div>
          </div>
        </aside>
      </div>
    </section>
  </main>
</template>
