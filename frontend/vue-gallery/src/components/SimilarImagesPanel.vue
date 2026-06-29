<script setup lang="ts">
import type { ImageItem } from '@/api/client'

defineProps<{
  images: readonly ImageItem[]
}>()
</script>

<template>
  <div class="panel">
    <div class="panel-head">
      <div>
        <p class="eyebrow">相似推荐</p>
        <h3>同类作品</h3>
      </div>
      <RouterLink class="btn btn-secondary btn-small" to="/search">更多</RouterLink>
    </div>
    <div v-if="images.length === 0" class="activity-empty">
      <p class="activity-empty__title">暂无相似作品</p>
      <p class="activity-empty__desc">后端返回 similar_images 为空时显示此状态。</p>
    </div>
    <div v-else class="grid-2">
      <RouterLink
        v-for="item in images"
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
</template>
