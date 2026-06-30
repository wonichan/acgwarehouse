<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import CollectionMasonryGrid from '@/components/CollectionMasonryGrid.vue'
import { ApiError, getCollection, updateCollection } from '@/api/client'
import type { CollectionResponse, ImageItem } from '@/api/client'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { imageToArtItem } from '@/utils/imagePresentation'
import type { ArtItem } from '@/types'

const route = useRoute()
const router = useRouter()
const { show } = useToast()
const { isLoggedIn, user } = useAuth()

const collection = ref<CollectionResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const settingCover = ref<number | null>(null)

const collectionId = computed<number | null>(() => {
  const raw = route.params['id']
  if (typeof raw !== 'string') return null
  const parsed = Number(raw)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

const artItems = computed<readonly ArtItem[]>(() => {
  if (!collection.value) return []
  return collection.value.items
    .map(item => item.image)
    .filter((image): image is ImageItem => image !== undefined && image.id > 0)
    .map(imageToArtItem)
})

const isOwner = computed<boolean>(() => {
  if (!isLoggedIn.value || !collection.value || !user.value) return false
  return collection.value.user_id === user.value.id
})

function formatVisibility(visibility: CollectionResponse['visibility']): string {
  switch (visibility) {
    case 'private':
      return '私有'
    case 'public':
      return '公开'
  }
}

function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString('zh-CN')
}

async function loadDetail(): Promise<void> {
  if (collectionId.value === null) {
    collection.value = null
    error.value = '收藏夹 ID 非法'
    loading.value = false
    return
  }

  loading.value = true
  error.value = null

  try {
    collection.value = await getCollection(collectionId.value)
  } catch (e) {
    collection.value = null
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '收藏夹详情加载失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

async function handleSetCover(imageId: number): Promise<void> {
  if (!collection.value || !isOwner.value) return

  settingCover.value = imageId
  try {
    const updated = await updateCollection(collection.value.id, {
      name: collection.value.name,
      visibility: collection.value.visibility,
      cover_image_id: imageId,
    })
    collection.value = updated
    show('封面已更新')
  } catch (e) {
    if (e instanceof ApiError) {
      show(e.message)
    } else {
      show('封面更新失败，请稍后重试')
    }
  } finally {
    settingCover.value = null
  }
}

watch(
  collectionId,
  () => {
    loadDetail()
  },
  { immediate: true }
)
</script>

<template>
  <main>
    <section class="section" data-od-id="collection-detail">
      <div class="container">
        <div v-if="loading" class="panel">
          <p class="eyebrow">正在加载</p>
          <h2>读取收藏夹详情</h2>
          <p class="meta">正在请求 /api/v1/collections/{{ collectionId }}。</p>
        </div>

        <div v-else-if="error" class="panel">
          <p class="eyebrow">加载失败</p>
          <h2>无法读取收藏夹</h2>
          <p class="meta">{{ error }}</p>
          <div class="divider"></div>
          <div class="hero-actions">
            <button class="btn btn-secondary" type="button" @click="loadDetail">重试</button>
            <RouterLink class="btn btn-primary" to="/collections">返回收藏夹列表</RouterLink>
          </div>
        </div>

        <template v-else-if="collection">
          <div class="panel-head">
            <div>
              <button class="back-link" type="button" @click="router.push('/collections')">
                ← 返回收藏夹
              </button>
              <p class="eyebrow">{{ formatVisibility(collection.visibility) }} · 收藏夹</p>
              <h1>{{ collection.name }}</h1>
              <p class="meta">
                {{ collection.items.length }} 张作品 · 创建于 {{ formatDate(collection.created_at) }}
              </p>
            </div>
            <div v-if="isOwner" class="hero-actions">
              <span class="hint">悬停图片卡片可设为封面</span>
            </div>
          </div>

          <div v-if="artItems.length === 0" class="panel">
            <p class="eyebrow">暂无收藏</p>
            <h3>这个收藏夹还没有图片</h3>
            <p class="meta">在图片详情页点击收藏按钮，将图片加入此收藏夹。</p>
          </div>

          <CollectionMasonryGrid
            v-else
            :items="artItems"
            :cover-image-id="collection.cover_image_id"
            :is-owner="isOwner"
            :setting-cover="settingCover"
            @set-cover="handleSetCover"
          />
        </template>
      </div>
    </section>
  </main>
</template>

<style scoped>
.back-link {
  background: none;
  border: none;
  color: var(--muted);
  padding: 0;
  margin-bottom: var(--space-2);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: color var(--motion-fast) var(--ease-standard);
}

.back-link:hover {
  color: var(--accent);
}

.hint {
  color: var(--muted);
  font-size: var(--text-sm);
}
</style>
