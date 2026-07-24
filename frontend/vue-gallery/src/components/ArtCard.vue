<script setup lang="ts">
import { computed } from 'vue'
import { Check } from 'lucide-vue-next'
import type { ArtItem } from '@/types'
import { useSelection } from '@/composables/useSelection'
import AppIcon from '@/components/AppIcon.vue'

const props = withDefaults(defineProps<{
  item: ArtItem
  selectable?: boolean
}>(), {
  selectable: true
})

const { isSelected, toggle } = useSelection()

const imageWidth = computed(() => (
  props.item.imageWidth !== undefined && props.item.imageWidth > 0 ? props.item.imageWidth : undefined
))
const imageHeight = computed(() => (
  props.item.imageHeight !== undefined && props.item.imageHeight > 0 ? props.item.imageHeight : undefined
))
const previewStyle = computed(() => {
  if (imageWidth.value === undefined || imageHeight.value === undefined) return undefined
  return { aspectRatio: `${imageWidth.value} / ${imageHeight.value}` }
})

const handleClick = (event: MouseEvent) => {
  if ((event.target as HTMLElement).closest('a, button')) return
  if (props.selectable) {
    toggle(props.item.id)
  }
}
</script>

<template>
  <article
    class="art-card"
    :class="{ 'is-selected': isSelected(item.id) }"
    :data-selectable="selectable"
    :data-id="item.id"
    @click="handleClick"
  >
    <RouterLink class="art-card-link" :to="`/detail?id=${item.id}`" :aria-label="`查看${item.title}详情`">
      <div
        v-if="item.imageUrl"
        class="art-preview"
        :class="[item.previewVariant, { 'has-stable-ratio': previewStyle !== undefined }]"
        :style="previewStyle"
      >
        <img
          :src="item.imageUrl"
          :alt="item.title"
          :width="imageWidth"
          :height="imageHeight"
          loading="lazy"
        />
      </div>
      <div
        v-else
        class="art-preview"
        :class="[item.previewVariant, { 'has-stable-ratio': previewStyle !== undefined }]"
        :style="previewStyle"
      ></div>
    </RouterLink>
    <button
      v-if="selectable"
      class="select-check"
      type="button"
      aria-label="选择图片"
      :aria-pressed="isSelected(item.id)"
      @click.stop="toggle(item.id)"
    >
      <AppIcon :icon="Check" :size="16" />
    </button>
    <div class="art-body">
      <p class="art-title">{{ item.title }}</p>
      <div class="art-meta">
        <span v-for="tag in item.tags.slice(0, 3)" :key="tag">#{{ tag }}</span>
        <span class="art-meta-score">#{{ item.score }}/100</span>
        <span>{{ item.favorites }} 收藏</span>
      </div>
    </div>
  </article>
</template>

<style scoped>
.art-card-link {
  display: block;
  color: inherit;
}

.art-title {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.art-meta-score {
  color: var(--accent);
  font-variant-numeric: tabular-nums;
}

.art-card :deep(.select-check) {
  color: var(--surface);
}

.art-card :deep(.select-check .app-icon) {
  color: inherit;
}

@media (hover: hover) {
  .art-card:hover .art-meta {
    color: var(--fg-2);
  }
}
</style>
