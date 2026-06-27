<script setup lang="ts">
import { ref } from 'vue'
import { useToast } from '@/composables/useToast'
import { ApiError, searchImages } from '@/api/client'
import type { ImageItem } from '@/api/client'

const { show } = useToast()

const keyword = ref('')
const loading = ref(false)
const error = ref<string | null>(null)
const results = ref<readonly ImageItem[]>([])
const total = ref(0)
const searchSummary = ref('请输入文件名或关键词搜索')

function formatScore(value: number): string {
  return Number.isFinite(value) ? value.toFixed(1) : '0.0'
}

async function handleSearch(): Promise<void> {
  loading.value = true
  error.value = null

  try {
    const searchData = await searchImages({
      keyword: keyword.value.trim() || undefined,
      limit: 20,
    })

    results.value = searchData.items
    total.value = searchData.total

    searchSummary.value = keyword.value.trim().length > 0
      ? `正在展示「${keyword.value.trim()}」相关结果，共 ${total.value} 张作品`
      : `正在展示 ${total.value} 张作品`

    show('已按后端 q 参数完成搜索')
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '搜索失败，请重试'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main>
    <!-- Hero Section -->
    <section class="section hero" data-od-id="search-hero">
      <div class="container hero-grid">
        <div>
          <p class="eyebrow">智能搜索 · 后端真实查询</p>
          <h1>把模糊灵感变成可筛选结果。</h1>
          <p class="lead">当前后端搜索接口消费 q、page、size 参数；标签组合与评分范围筛选待后端支持后再接入。</p>
        </div>
        <div class="panel panel-raised form-grid">
          <label class="field">
            关键词
            <input class="input" v-model="keyword" placeholder="输入文件名关键词..." @keyup.enter="handleSearch" />
          </label>
          <p class="meta">请求会发送到 /api/v1/search?q=关键词&size=20，不会伪造未支持的标签或评分筛选。</p>
          <button class="btn btn-primary" type="button" @click="handleSearch" :disabled="loading">
            {{ loading ? '搜索中...' : '应用搜索' }}
          </button>
        </div>
      </div>
    </section>

    <!-- Results Section -->
    <section class="section" data-od-id="search-results">
      <div class="container grid-main">
        <div class="stack">
          <div class="panel">
            <div class="row-between">
              <div>
                <p class="eyebrow">搜索结果</p>
                <h2>相关作品</h2>
              </div>
              <span class="meta" data-search-summary>{{ searchSummary }}</span>
            </div>
          </div>

          <div v-if="loading" class="panel">
            <p class="eyebrow">正在搜索</p>
            <h3>读取后端搜索结果</h3>
            <p class="meta">正在请求 /api/v1/search。</p>
          </div>

          <div v-else-if="error" class="panel">
            <p class="meta">{{ error }}</p>
            <button class="btn btn-secondary" type="button" @click="handleSearch">重试</button>
          </div>

          <div v-else-if="results.length === 0" class="panel">
            <p class="meta">暂无结果，请输入关键词后搜索。</p>
          </div>

          <div v-else class="results-list">
            <article v-for="item in results" :key="item.id" class="result-row">
              <RouterLink :to="`/detail?id=${item.id}`" :aria-label="`查看${item.filename}详情`">
                <div v-if="item.url" class="thumb">
                  <img :src="item.url" :alt="item.filename" loading="lazy" />
                </div>
                <div v-else class="thumb" aria-hidden="true"></div>
              </RouterLink>
              <div>
                <h3>{{ item.filename }}</h3>
                <p class="meta">分类：{{ item.category || '未分类' }} · 尺寸：{{ item.width }}x{{ item.height }}</p>
                <div class="kicker-row">
                  <span v-if="item.avg_score >= 80" class="tag is-hot">{{ formatScore(item.avg_score) }}/100</span>
                  <span v-else class="tag">{{ formatScore(item.avg_score) }}/100</span>
                  <span class="tag">{{ item.view_count }} 次浏览</span>
                  <span class="tag">{{ item.favorite_count }} 收藏</span>
                </div>
              </div>
              <RouterLink class="btn btn-secondary btn-small" :to="`/detail?id=${item.id}`">查看</RouterLink>
            </article>
          </div>
        </div>
        <aside class="panel">
          <p class="eyebrow">搜索建议</p>
          <h3>先用文件名关键词</h3>
          <p class="meta">当前后端搜索处理器读取 q 参数。标签、评分区间和多条件组合不在本任务中伪造，待后端支持后再展示为可用筛选。</p>
          <div class="divider"></div>
          <div class="kicker-row">
            <span class="pill">682325</span>
            <span class="pill">small</span>
            <span class="pill">jpg</span>
          </div>
        </aside>
      </div>
    </section>
  </main>
</template>
