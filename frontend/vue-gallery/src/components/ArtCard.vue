<script setup lang="ts">
import type { ArtItem } from '@/types'
import { useSelection } from '@/composables/useSelection'

const props = withDefaults(defineProps<{
  item: ArtItem
  selectable?: boolean
}>(), {
  selectable: true
})

const { isSelected, toggle } = useSelection()

const handleClick = (event: MouseEvent) => {
  // Don't toggle if clicking on a link or button
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
    <RouterLink :to="`/detail?id=${item.id}`" :aria-label="`查看${item.title}详情`">
      <div class="art-preview" :class="item.previewVariant"></div>
    </RouterLink>
    <button
      v-if="selectable"
      class="select-check"
      aria-label="选择图片"
      @click.stop="toggle(item.id)"
    >✓</button>
    <div class="art-body">
      <p class="art-title">{{ item.title }}</p>
      <div class="art-meta">
        <span v-for="tag in item.tags.slice(0, 3)" :key="tag">#{{ tag }}</span>
        <span>#{{ item.score }} 分</span>
        <span>{{ item.favorites }} 收藏</span>
      </div>
    </div>
  </article>
</template>