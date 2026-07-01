<script setup lang="ts">
import type { ArtItem } from '@/types'
import ArtCard from '@/components/ArtCard.vue'

const SKELETON_COUNT = 10

defineProps<{
  readonly items: ArtItem[]
  readonly loading: boolean
  readonly error: string | null
}>()

const emit = defineEmits<{
  retry: []
}>()
</script>

<template>
  <section class="section daily-random-section" data-od-id="daily-random-recommendations">
    <div class="container">
      <div class="panel panel-raised daily-random-panel">
        <div class="daily-random-head">
          <div>
            <p class="eyebrow">北京时间今日更新</p>
            <h2>每日随机推荐</h2>
          </div>
          <div class="daily-random-stats" aria-label="每日推荐摘要">
            <span class="tag is-hot">10 张</span>
            <span class="tag">全站同款</span>
            <span class="tag">公平轮转</span>
          </div>
        </div>

        <div v-if="loading" class="daily-random-grid" aria-label="每日推荐加载中">
          <div
            v-for="index in SKELETON_COUNT"
            :key="index"
            class="daily-random-skeleton"
            :class="{ 'daily-random-skeleton-feature': index === 1 }"
          ></div>
        </div>
        <div v-else-if="error" class="panel daily-random-state" role="status">
          <h3>每日推荐暂时不可用</h3>
          <p class="meta">图库仍可继续浏览，稍后可重试。</p>
          <button class="btn btn-secondary btn-small" type="button" @click="emit('retry')">重试</button>
        </div>
        <div v-else-if="items.length === 0" class="panel daily-random-state">
          <h3>今日还没有可展示作品</h3>
          <p class="meta">当图库里有 active 图片后，这里会展示每日随机推荐。</p>
        </div>
        <div v-else class="daily-random-grid" aria-label="每日随机推荐作品">
          <ArtCard
            v-for="(item, index) in items"
            :key="item.id"
            :item="item"
            :selectable="false"
            :class="{ 'daily-random-feature-card': index === 0 }"
          />
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.daily-random-section { padding-top: 0; }
.daily-random-panel { background: color-mix(in oklab, var(--surface), var(--surface-warm) 8%); }
.daily-random-head { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: var(--space-6); align-items: start; margin-bottom: var(--space-6); }
.daily-random-copy { margin: var(--space-3) 0 0; color: var(--fg-2); max-width: 54ch; }
.daily-random-stats { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: var(--space-2); }
.daily-random-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: var(--space-4); align-items: stretch; }
.daily-random-grid :deep(.art-card) { margin: 0; height: 100%; }
.daily-random-grid :deep(.art-preview) { min-height: 176px; }
.daily-random-feature-card { grid-column: span 2; grid-row: span 2; }
.daily-random-feature-card :deep(.art-preview) { min-height: 392px; }
.daily-random-state { display: grid; gap: var(--space-3); border-color: var(--border-soft); background: var(--surface); }
.daily-random-state .btn { justify-self: start; }
.daily-random-skeleton { min-height: 262px; border: 1px solid var(--border-soft); border-radius: var(--radius-lg); background: linear-gradient(110deg, color-mix(in oklab, var(--surface-warm), var(--surface) 48%), var(--surface), color-mix(in oklab, var(--accent), var(--surface) 86%)); opacity: 0.76; }
.daily-random-skeleton-feature { grid-column: span 2; grid-row: span 2; min-height: 488px; }

@media (max-width: 1180px) {
  .daily-random-head { grid-template-columns: 1fr; }
  .daily-random-stats { justify-content: flex-start; }
  .daily-random-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .daily-random-feature-card,
  .daily-random-skeleton-feature { grid-column: span 2; }
}

@media (max-width: 744px) {
  .daily-random-grid { grid-template-columns: 1fr; }
  .daily-random-feature-card,
  .daily-random-skeleton-feature { grid-column: span 1; grid-row: span 1; }
  .daily-random-feature-card :deep(.art-preview) { min-height: 300px; }
  .daily-random-skeleton-feature { min-height: 300px; }
}
</style>
