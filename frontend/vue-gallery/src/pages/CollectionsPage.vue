<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ApiError, createCollection, getCollections } from '@/api/client'
import type { CollectionResponse } from '@/api/client'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'

const { show } = useToast()
const { isLoggedIn, loading: authLoading } = useAuth()

const albumName = ref('雨夜角色参考')
const albumVisibility = ref<CollectionResponse['visibility']>('private')
const collections = ref<readonly CollectionResponse[]>([])
const loading = ref(false)
const creating = ref(false)
const error = ref<string | null>(null)

const totalItems = computed(() => collections.value.reduce((sum, collection) => sum + collection.items.length, 0))

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

function formatDateTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function loadCollections(): Promise<void> {
  if (!isLoggedIn.value) {
    collections.value = []
    error.value = null
    loading.value = false
    return
  }

  loading.value = true
  error.value = null

  try {
    collections.value = await getCollections()
  } catch (e) {
    collections.value = []
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '收藏夹加载失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

async function handleCreateAlbum(): Promise<void> {
  const name = albumName.value.trim()
  if (!isLoggedIn.value) {
    show('请先登录后再创建收藏夹')
    return
  }
  if (name.length === 0) {
    show('请输入收藏夹名称')
    return
  }

  creating.value = true
  try {
    await createCollection(name, albumVisibility.value)
    show('收藏夹已创建')
    albumName.value = ''
    await loadCollections()
  } catch (e) {
    if (e instanceof ApiError) {
      show(e.message)
    } else {
      show('创建收藏夹失败，请稍后重试')
    }
  } finally {
    creating.value = false
  }
}

function handleBatchUnavailable(): void {
  if (!isLoggedIn.value) {
    show('请先登录后再管理收藏夹')
    return
  }
  show('批量下载和批量打标后端流程尚未接入，未执行操作')
}

onMounted(() => {
  loadCollections()
})

watch(isLoggedIn, () => {
  loadCollections()
})
</script>

<template>
  <main>
    <!-- Hero Section -->
    <section class="section hero" data-od-id="collections-hero">
      <div class="container hero-grid">
        <div>
          <p class="eyebrow">收藏夹 / 相册管理</p>
          <h1>把灵感按用途整理成相册。</h1>
          <p class="lead">创建相册、批量收藏、统一打标签，让社区浏览变成可复用的个人素材库。</p>
        </div>
        <div class="panel panel-raised form-grid">
          <label class="field">
            相册名称
            <input class="input" v-model="albumName" :disabled="!isLoggedIn || creating" />
          </label>
          <label class="field">
            可见性
            <select class="select" v-model="albumVisibility" :disabled="!isLoggedIn || creating">
              <option value="private">私有</option>
              <option value="public">公开</option>
            </select>
          </label>
          <p class="meta">新建收藏夹按后端契约创建为所选可见性；后端当前不接收描述或默认标签字段。</p>
          <button class="btn btn-primary" type="button" :disabled="!isLoggedIn || creating" @click="handleCreateAlbum">
            {{ creating ? '创建中...' : '创建相册' }}
          </button>
        </div>
      </div>
    </section>

    <!-- Albums Section -->
    <section class="section" data-od-id="album-grid">
      <div class="container">
        <div class="panel-head">
          <div>
            <p class="eyebrow">我的相册</p>
            <h2>管理收藏内容</h2>
          </div>
          <span class="meta">真实收藏夹 {{ collections.length }} 个 · 收藏条目 {{ totalItems }} 个</span>
        </div>

        <div v-if="authLoading" class="panel">
          <p class="eyebrow">认证状态</p>
          <h3>正在检查登录状态</h3>
          <p class="meta">收藏夹接口需要 Bearer token。</p>
        </div>

        <div v-else-if="!isLoggedIn" class="panel panel-raised">
          <p class="eyebrow">需要登录</p>
          <h3>登录后查看真实收藏夹</h3>
          <p class="meta">未登录访问 /api/v1/collections 会返回 401。这里不再展示 mock 相册或作品。</p>
          <div class="hero-actions">
            <RouterLink class="btn btn-primary" to="/account">去登录</RouterLink>
            <RouterLink class="btn btn-secondary" to="/">继续浏览图库</RouterLink>
          </div>
        </div>

        <div v-else-if="loading" class="panel">
          <p class="eyebrow">正在同步</p>
          <h3>读取收藏夹列表</h3>
          <p class="meta">正在请求 /api/v1/collections。</p>
        </div>

        <div v-else-if="error" class="panel">
          <p class="eyebrow">加载失败</p>
          <h3>无法读取收藏夹</h3>
          <p class="meta">{{ error }}</p>
          <div class="divider"></div>
          <button class="btn btn-secondary" type="button" @click="loadCollections">重试</button>
        </div>

        <div v-else-if="collections.length === 0" class="panel">
          <p class="eyebrow">暂无收藏夹</p>
          <h3>还没有创建收藏夹</h3>
          <p class="meta">创建后会显示后端返回的收藏夹元数据和 items.length 统计。</p>
        </div>

        <div v-else class="album-grid">
          <article v-for="collection in collections" :key="collection.id" class="card album-card">
            <div class="album-cover album-cover--meta">
              <strong class="num">{{ collection.items.length }}</strong>
              <span>真实收藏条目</span>
            </div>
            <h3>{{ collection.name }}</h3>
            <p class="meta">
              {{ collection.items.length }} 张作品 · {{ formatVisibility(collection.visibility) }} · 创建于 {{ formatDate(collection.created_at) }}
            </p>
            <div class="kicker-row">
              <span class="tag">ID {{ collection.id }}</span>
              <span class="tag">Owner {{ collection.user_id }}</span>
              <span class="tag">{{ collection.visibility }}</span>
            </div>
            <div v-if="collection.items.length > 0" class="collection-items">
              <RouterLink
                v-for="item in collection.items"
                :key="`${item.collection_id}-${item.image_id}`"
                class="collection-item"
                :to="`/detail?id=${item.image_id}`"
              >
                <span class="collection-item__id">Image {{ item.image_id }}</span>
                <span class="collection-item__time">{{ formatDateTime(item.created_at) }}</span>
              </RouterLink>
            </div>
            <p v-else class="meta">这个收藏夹还没有真实收藏条目。</p>
          </article>
        </div>
      </div>
    </section>

    <!-- Batch Section -->
    <section class="section" data-od-id="collection-batch">
      <div class="container grid-main">
        <div class="panel panel-raised">
          <p class="eyebrow">收藏内容</p>
          <h3>展示真实收藏条目</h3>
          <p class="meta">收藏夹列表现在展示后端返回的真实 collection item，并按 image_id 跳转到作品详情；不使用 mock 图片卡片。</p>
        </div>
        <aside class="panel">
          <p class="eyebrow">批量操作面板</p>
          <h3>真实接口优先</h3>
          <p class="meta">批量下载与批量打标签缺少完整后端/UI 流程，因此不会伪造成功提示。</p>
          <div class="divider"></div>
          <button class="btn btn-secondary" type="button" @click="handleBatchUnavailable">查看功能状态</button>
        </aside>
      </div>
    </section>
  </main>
</template>

<style scoped>
.collection-items {
  display: grid;
  gap: var(--space-2);
  margin-top: var(--space-3);
}

.collection-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: inherit;
  text-decoration: none;
  transition:
    border-color var(--motion-fast) var(--ease-standard),
    background var(--motion-fast) var(--ease-standard);
}

.collection-item:hover {
  border-color: var(--accent);
  background: color-mix(in oklab, var(--accent), transparent 94%);
}

.collection-item__id {
  min-width: 0;
  overflow-wrap: anywhere;
  font-weight: 700;
}

.collection-item__time {
  flex: 0 0 auto;
  color: var(--muted);
  font-size: var(--text-sm);
}

@media (max-width: 560px) {
  .collection-item {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
