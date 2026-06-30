<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import type { ComponentPublicInstance } from 'vue'
import ArtCard from '@/components/ArtCard.vue'
import { useMasonryLayout } from '@/composables/useMasonryLayout'
import type { ArtItem } from '@/types'

const props = defineProps<{
  readonly items: readonly ArtItem[]
  readonly coverImageId: number
  readonly isOwner: boolean
  readonly settingCover: number | null
}>()

const emit = defineEmits<{
  'set-cover': [imageId: number]
}>()

const sourceItems = computed(() => props.items)
const { masonry, columnCount, columns, rebuild, observe } = useMasonryLayout(sourceItems)

function setMasonryElement(element: Element | ComponentPublicInstance | null): void {
  masonry.value = element instanceof HTMLElement ? element : null
}

function emitSetCover(item: ArtItem): void {
  emit('set-cover', Number(item.id))
}

onMounted(() => {
  rebuild()
  observe()
})

watch(sourceItems, () => {
  rebuild()
})
</script>

<template>
  <div
    :ref="setMasonryElement"
    class="masonry"
    :style="{ '--masonry-columns': String(columnCount) }"
  >
    <div v-for="(column, columnIndex) in columns" :key="columnIndex" class="masonry-column">
      <div v-for="item in column" :key="item.id" class="masonry-item">
        <div v-if="isOwner" class="masonry-item__actions">
          <button
            v-if="coverImageId === Number(item.id)"
            class="cover-badge"
            type="button"
            disabled
          >
            当前封面
          </button>
          <button
            v-else
            class="cover-btn"
            type="button"
            :disabled="settingCover === Number(item.id)"
            @click.stop="emitSetCover(item)"
          >
            {{ settingCover === Number(item.id) ? '设置中...' : '设为封面' }}
          </button>
        </div>
        <ArtCard :item="item" :selectable="false" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.masonry {
  display: grid;
  grid-template-columns: repeat(var(--masonry-columns), 1fr);
  gap: var(--space-4);
  align-items: start;
}

.masonry-column {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.masonry-item {
  position: relative;
}

.masonry-item__actions {
  position: absolute;
  top: var(--space-2);
  right: var(--space-2);
  z-index: 2;
  opacity: 0;
  transition: opacity var(--motion-fast) var(--ease-standard);
}

.masonry-item:hover .masonry-item__actions,
.masonry-item:focus-within .masonry-item__actions {
  opacity: 1;
}

.cover-btn,
.cover-badge {
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
}

.cover-btn {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--fg);
  cursor: pointer;
  backdrop-filter: blur(8px);
  transition:
    background var(--motion-fast) var(--ease-standard),
    border-color var(--motion-fast) var(--ease-standard),
    transform var(--motion-fast) var(--ease-standard);
}

.cover-btn:hover:not(:disabled) {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--accent-on);
}

.cover-btn:active:not(:disabled) {
  transform: scale(0.97);
}

.cover-btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.cover-badge {
  border: 1px solid var(--accent);
  background: var(--accent);
  color: var(--accent-on);
  cursor: default;
}

@media (max-width: 560px) {
  .masonry {
    grid-template-columns: 1fr;
  }

  .masonry-item__actions {
    opacity: 1;
  }
}
</style>
