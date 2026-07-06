<script setup lang="ts">
import { computed } from 'vue'
import type { ImageItem } from '@/api/client'

const props = defineProps<{
  images: readonly ImageItem[]
  moreLinkTag?: string
}>()

const moreLink = computed<string>(() => {
  const tag = props.moreLinkTag?.trim()
  if (tag && tag.length > 0) {
    return `/?tag=${encodeURIComponent(tag)}`
  }
  return '/search'
})
</script>

<template>
  <div class="panel">
    <div class="panel-head">
      <div>
        <p class="eyebrow">相似推荐</p>
        <h3>同类作品</h3>
      </div>
      <RouterLink class="btn btn-secondary btn-small" :to="moreLink">更多</RouterLink>
    </div>
    <div v-if="images.length === 0" class="activity-empty">
      <p class="activity-empty__title">暂无相似作品</p>
      <p class="activity-empty__desc">还没有相关的作品推荐</p>
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
