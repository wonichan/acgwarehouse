<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { searchImages, ApiError } from '@/api/client'
import type { ImageItem } from '@/api/client'

const { show } = useToast()

// Form state
const keyword = ref('')
const tags = ref('')
const scoreRange = ref('不限评分')

// Results state
const loading = ref(false)
const error = ref<string | null>(null)
const results = ref<ImageItem[]>([])
const total = ref(0)
const searchSummary = ref('请输入搜索条件')

// Search handler
async function handleSearch() {
  loading.value = true
  error.value = null
  
  try {
    // Parse score range
    let minScore = 0
    if (scoreRange.value === '4.5 分以上') minScore = 4.5
    else if (scoreRange.value === '4.0 - 4.5 分') minScore = 4.0
    
    const searchData = await searchImages({
      keyword: keyword.value || undefined,
      tags: tags.value || undefined,
      minScore: minScore > 0 ? minScore : undefined,
      limit: 20,
    })
    
    results.value = searchData.items
    total.value = searchData.total
    
    searchSummary.value = keyword.value 
      ? `正在展示「${keyword.value}」相关结果，共 ${total.value} 张作品`
      : `正在展示 ${total.value} 张作品`
    
    show('筛选条件已应用')
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

// Initial load (optional: show all images)
onMounted(() => {
  // Optionally load initial results
  // handleSearch()
})
</script>

<template>
  <main>
    <!-- Hero Section -->
    <section class="section hero" data-od-id="search-hero">
      <div class="container hero-grid">
        <div>
          <p class="eyebrow">智能搜索 · 文件名 / 标签 / 评分</p>
          <h1>把模糊灵感变成可筛选结果。</h1>
          <p class="lead">输入文件名片段、标签组合或评分区间，快速定位适合收藏、打标和下载的作品。</p>
        </div>
        <div class="panel panel-raised form-grid">
          <label class="field">
            关键词
            <input class="input" v-model="keyword" placeholder="输入文件名关键词..." />
          </label>
          <label class="field">
            标签
            <input class="input" v-model="tags" placeholder="多个标签用逗号分隔" />
          </label>
          <label class="field">
            评分范围
            <select class="select" v-model="scoreRange">
              <option>不限评分</option>
              <option>4.5 分以上</option>
              <option>4.0 - 4.5 分</option>
            </select>
          </label>
          <button class="btn btn-primary" @click="handleSearch" :disabled="loading">
            {{ loading ? '搜索中...' : '应用筛选' }}
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
                <p class="eyebrow">筛选结果</p>
                <h2>相关作品</h2>
              </div>
              <span class="meta" data-search-summary>{{ searchSummary }}</span>
            </div>
          </div>
          
          <!-- Error state -->
          <div v-if="error" class="panel">
            <p class="meta">{{ error }}</p>
            <button class="btn btn-secondary" @click="handleSearch">重试</button>
          </div>
          
          <!-- Empty state -->
          <div v-else-if="results.length === 0 && !loading" class="panel">
            <p class="meta">暂无结果，请调整搜索条件</p>
          </div>
          
          <!-- Results list -->
          <div v-else class="results-list">
            <article v-for="item in results" :key="item.id" class="result-row">
              <div class="thumb"></div>
              <div>
                <h3>{{ item.filename }}</h3>
                <p class="meta">分类：{{ item.category }} · 尺寸：{{ item.width }}x{{ item.height }}</p>
                <div class="kicker-row">
                  <span v-if="item.avg_rating >= 4.5" class="tag is-hot">{{ item.avg_rating.toFixed(1) }} 分</span>
                  <span v-else class="tag">{{ item.avg_rating.toFixed(1) }} 分</span>
                  <span class="tag">{{ item.view_count }} 次浏览</span>
                </div>
              </div>
              <RouterLink class="btn btn-secondary btn-small" :to="`/detail?id=${item.id}`">查看</RouterLink>
            </article>
          </div>
        </div>
        <aside class="panel">
          <p class="eyebrow">搜索建议</p>
          <h3>组合标签更稳定</h3>
          <p class="meta">建议用「主题 + 光线 + 角色姿态」组合，例如：雨景、逆光、侧身。评分排序适合找高质量参考，最新排序适合追社区更新。</p>
          <div class="divider"></div>
          <div class="kicker-row">
            <span class="pill">雨景</span>
            <span class="pill">暖光</span>
            <span class="pill">头像</span>
            <span class="pill">场景</span>
          </div>
        </aside>
      </div>
    </section>
  </main>
</template>