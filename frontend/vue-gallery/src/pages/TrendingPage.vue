<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import SegmentedControl from '@/components/SegmentedControl.vue'
import { ApiError, getRankings } from '@/api/client'
import type { RankingPeriod, RankingResponse } from '@/api/client'

const periodLabels: string[] = ['每日', '每周', '每月']
type PeriodLabel = '每日' | '每周' | '每月'

const activePeriod = ref<PeriodLabel>('每日')
const rankings = ref<readonly RankingResponse[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

const backendPeriod = computed<RankingPeriod>(() => {
  switch (activePeriod.value) {
    case '每日':
      return 'day'
    case '每周':
      return 'week'
    case '每月':
      return 'month'
  }
})

function formatScore(value: number): string {
  return Number.isFinite(value) ? value.toFixed(1) : '0.0'
}

function imageTitle(ranking: RankingResponse): string {
  return ranking.image.filename || `作品 #${ranking.image.id}`
}

async function loadRankings(): Promise<void> {
  loading.value = true
  error.value = null

  try {
    rankings.value = await getRankings({ period: backendPeriod.value, limit: 20 })
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '热榜加载失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadRankings()
})

watch(activePeriod, () => {
  loadRankings()
})
</script>

<template>
  <main>
    <!-- Hero Section -->
    <section class="section hero" data-od-id="trending-hero">
      <div class="container row-between">
        <div>
          <p class="eyebrow">社区热榜</p>
          <h1>每天、每周、每月的收藏风向。</h1>
          <p class="lead">热榜基于社区收藏、评分和近期互动热度排序，适合快速发现高质量参考图。</p>
        </div>
        <SegmentedControl :options="periodLabels" v-model="activePeriod" />
      </div>
    </section>

    <!-- Ranking Section -->
    <section class="section" data-od-id="trending-list">
      <div class="container grid-main">
        <div class="rank-list" aria-live="polite">
          <div v-if="loading" class="panel">
            <p class="eyebrow">正在同步</p>
            <h3>正在读取 {{ activePeriod }}热榜</h3>
            <p class="meta">请求会通过 Vite 代理访问 /api/v1/rankings。</p>
          </div>

          <div v-else-if="error" class="panel">
            <p class="eyebrow">加载失败</p>
            <h3>暂时无法读取热榜</h3>
            <p class="meta">{{ error }}</p>
            <div class="divider"></div>
            <button class="btn btn-secondary btn-small" type="button" @click="loadRankings">重试</button>
          </div>

          <div v-else-if="rankings.length === 0" class="panel">
            <p class="eyebrow">暂无数据</p>
            <h3>{{ activePeriod }}热榜还没有作品</h3>
            <p class="meta">后端 ranking 缓存为空时会显示此状态，而不是使用静态示例。</p>
          </div>

          <article v-for="ranking in rankings" v-else :key="`${ranking.period}-${ranking.rank}-${ranking.image.id}`" class="rank-row">
            <RouterLink :to="`/detail?id=${ranking.image.id}`" :aria-label="`查看${imageTitle(ranking)}详情`">
              <div v-if="ranking.image.url" class="thumb">
                <img :src="ranking.image.url" :alt="imageTitle(ranking)" loading="lazy" />
              </div>
              <div v-else class="thumb" aria-hidden="true"></div>
            </RouterLink>
            <div>
              <h3>{{ imageTitle(ranking) }}</h3>
              <p class="meta">
                热度 {{ formatScore(ranking.score) }} · 均分 {{ formatScore(ranking.image.avg_score) }}/100 · {{ ranking.favorite_count }} 收藏 · {{ ranking.view_count }} 浏览 · {{ ranking.rating_count }} 评分
              </p>
              <div class="kicker-row">
                <span class="tag is-hot">#{{ ranking.rank }}</span>
                <span class="tag">{{ ranking.period }}</span>
                <span class="tag">贝叶斯 {{ formatScore(ranking.bayesian_score) }}</span>
              </div>
            </div>
            <RouterLink class="btn btn-primary btn-small" :to="`/detail?id=${ranking.image.id}`">查看作品</RouterLink>
          </article>
        </div>
        <aside class="panel panel-raised">
          <p class="eyebrow">榜单规则</p>
          <h3>社区互动优先</h3>
          <p class="meta">热度由收藏、评分和近期浏览共同构成。切换周期会重新请求后端 day / week / month ranking 缓存。</p>
          <div class="divider"></div>
          <div class="stack">
            <div class="row-between">
              <span>当前周期</span>
              <strong class="num">{{ backendPeriod }}</strong>
            </div>
            <div class="row-between">
              <span>返回数量</span>
              <strong class="num">{{ rankings.length }}</strong>
            </div>
            <div class="row-between">
              <span>评分单位</span>
              <strong class="num">100</strong>
            </div>
          </div>
        </aside>
      </div>
    </section>
  </main>
</template>
