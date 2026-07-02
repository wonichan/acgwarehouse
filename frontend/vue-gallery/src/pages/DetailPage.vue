<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiError, getImage, rateImage } from '@/api/client'
import type { ImageDetailResponse, ImageItem } from '@/api/client'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { useZoom } from '@/composables/useZoom'
import AuthRequiredStatus from '@/components/AuthRequiredStatus.vue'
import CollectionPickerPanel from '@/components/CollectionPickerPanel.vue'
import DetailLoadingState from '@/components/DetailLoadingState.vue'
import DetailStatusPanel from '@/components/DetailStatusPanel.vue'
import SimilarImagesPanel from '@/components/SimilarImagesPanel.vue'
import TagPickerPanel from '@/components/TagPickerPanel.vue'

const route = useRoute()
const { zoom, zoomIn, zoomOut, reset } = useZoom()
const { show } = useToast()
const { isLoggedIn } = useAuth()

const detail = ref<ImageDetailResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const selectedScore = ref(50)
const savingRating = ref(false)
const collectionAuthRequired = ref<string | null>(null)
const tagAuthRequired = ref<string | null>(null)
const ratingScoreMin = 0
const ratingScoreMax = 100

const collectionPickerRef = ref<InstanceType<typeof CollectionPickerPanel> | null>(null)
const tagPickerRef = ref<InstanceType<typeof TagPickerPanel> | null>(null)

type DetailRefreshOverrides = Partial<Pick<ImageDetailResponse, 'my_rating' | 'is_collected'>>

const imageId = computed<number | null>(() => {
  const v = route.query['id']
  const raw = typeof v === 'string' ? v : Array.isArray(v) && typeof v[0] === 'string' ? v[0] : null
  if (raw === null) return null
  const parsed = Number(raw)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

const image = computed<ImageItem | null>(() => detail.value?.image ?? null)

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
  return Math.min(ratingScoreMax, Math.max(ratingScoreMin, Math.round(value)))
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

function applyDetailRefreshOverrides(
  response: ImageDetailResponse,
  overrides: DetailRefreshOverrides,
): ImageDetailResponse {
  const current = detail.value
  const myRating = overrides.my_rating ?? response.my_rating ?? current?.my_rating ?? null
  const refreshedCollected = overrides.is_collected ?? response.is_collected
  const isCollected = refreshedCollected || current?.is_collected === true
  return {
    ...response,
    my_rating: myRating,
    is_collected: isCollected,
  }
}

async function refreshDetailSilently(overrides: DetailRefreshOverrides = {}): Promise<void> {
  if (imageId.value === null) return
  try {
    const response = await getImage(imageId.value)
    const nextDetail = applyDetailRefreshOverrides(response, overrides)
    detail.value = nextDetail
    selectedScore.value = clampRatingScore(nextDetail.my_rating ?? nextDetail.avg_score)
  } catch {
    // 静默刷新失败不覆盖已展示数据，也不设 error，用户可继续操作或手动重试
  }
}

async function handleSaveRating(): Promise<void> {
  if (imageId.value === null) return
  if (!isLoggedIn.value) {
    tagAuthRequired.value = '请先登录后再提交评分'
    show(tagAuthRequired.value)
    return
  }
  savingRating.value = true
  try {
    const result = await rateImage(imageId.value, clampRatingScore(selectedScore.value))
    // 乐观更新 my_rating，让"我的评分"标签立即反馈
    if (detail.value) {
      detail.value = { ...detail.value, my_rating: result.score }
    }
    show(`评分已更新为 ${result.score}/100`)
    // 静默刷新 avg_score / rating_count，不闪骨架屏
    await refreshDetailSilently({ my_rating: result.score })
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
  collectionAuthRequired.value = null
  if (!isLoggedIn.value) {
    collectionAuthRequired.value = '请先登录后再选择收藏夹'
    show(collectionAuthRequired.value)
    return
  }
  collectionPickerRef.value?.open()
}

function handleTagging(): void {
  tagAuthRequired.value = null
  if (!isLoggedIn.value) {
    tagAuthRequired.value = '请先登录后再管理标签'
    show(tagAuthRequired.value)
    return
  }
  tagPickerRef.value?.toggle()
}

async function onPickerSuccess(message: string, overrides: DetailRefreshOverrides = {}): Promise<void> {
  collectionAuthRequired.value = null
  tagAuthRequired.value = null
  show(message)
  await refreshDetailSilently(overrides)
}

async function onCollectionSuccess(): Promise<void> {
  await onPickerSuccess('已添加到收藏夹', { is_collected: true })
}

onMounted(() => {
  loadDetail()
})

watch(imageId, () => {
  collectionAuthRequired.value = null
  tagAuthRequired.value = null
  reset()
  loadDetail()
})
</script>

<template>
  <main>
    <section class="section" data-od-id="detail-viewer">
      <DetailStatusPanel v-if="imageId === null" variant="missing-id" />

      <DetailLoadingState v-else-if="loading" />

      <DetailStatusPanel v-else-if="error" variant="error" :message="error" @retry="loadDetail" />

      <div v-else-if="detail && image" class="container detail-stage">
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

        <aside class="stack">
          <div class="panel panel-raised">
            <p class="eyebrow">作品详情</p>
            <h1 class="detail-title">{{ image.filename }}</h1>
            <p class="lead">
              {{ image.width }}×{{ image.height }} · {{ formatBytes(image.size) }} · {{ image.category || '未分类' }} · 浏览 {{ image.view_count }} 次
            </p>
            <div class="kicker-row">
              <span class="tag is-hot">{{ formatScore(detail.avg_score) }}/100</span>
              <span class="tag">{{ detail.rating_count }} 评分</span>
              <span class="tag">{{ detail.favorite_count }} 收藏</span>
              <span v-if="detail.is_collected" class="tag">已收藏</span>
              <span v-if="detail.my_rating !== null" class="tag">我的评分 {{ detail.my_rating }}/100</span>
              <span v-for="tag in detail.tags" :key="tag" class="tag">{{ tag }}</span>
            </div>
            <div class="divider"></div>
            <div class="grid-2">
              <button class="btn btn-primary" type="button" @click="handleFavorite">收藏到相册</button>
              <button class="btn btn-secondary" type="button" @click="show('当前后端未提供下载接口，可打开原图后手动保存')">下载说明</button>
            </div>
            <AuthRequiredStatus v-if="collectionAuthRequired" :message="collectionAuthRequired" />
            <CollectionPickerPanel
              v-if="imageId !== null"
              ref="collectionPickerRef"
              :image-ids="[imageId]"
              @success="onCollectionSuccess"
            />
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
                <input
                  v-model.number="selectedScore"
                  class="input"
                  type="number"
                  :min="ratingScoreMin"
                  :max="ratingScoreMax"
                  step="1"
                  inputmode="numeric"
                  @change="selectedScore = clampRatingScore(selectedScore)"
                />
              </label>
              <button class="btn btn-primary" type="button" :disabled="savingRating" @click="handleSaveRating">
                {{ savingRating ? '保存中...' : '保存评分' }}
              </button>
              <button class="btn btn-secondary" type="button" @click="handleTagging">
                管理标签
              </button>
              <AuthRequiredStatus v-if="tagAuthRequired" :message="tagAuthRequired" />
              <TagPickerPanel
                v-if="imageId !== null"
                ref="tagPickerRef"
                :image-ids="[imageId]"
                :current-tag-names="detail.tags"
                @success="onPickerSuccess($event)"
              />
            </div>
          </div>

          <SimilarImagesPanel :images="detail.similar_images" />
        </aside>
      </div>
    </section>
  </main>

</template>
