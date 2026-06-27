<script setup lang="ts">
import { useSelection } from '@/composables/useSelection'
import { useToast } from '@/composables/useToast'

const { selectedCount, clear } = useSelection()
const { show } = useToast()

const handleBatchAction = (action: string) => {
  show(`${selectedCount.value} 张图片已${action}`)
}

const handleClear = () => {
  clear()
  show('已取消选择')
}
</script>

<template>
  <div
    class="batch-panel"
    :class="{ 'is-open': selectedCount > 0 }"
    aria-live="polite"
  >
    <strong><span>{{ selectedCount }}</span> 张已选择</strong>
    <button class="btn btn-secondary btn-small" @click="handleBatchAction('加入收藏夹')">加入收藏夹</button>
    <button class="btn btn-secondary btn-small" @click="handleBatchAction('批量打标签')">批量打标签</button>
    <button class="btn btn-secondary btn-small" @click="handleClear">取消选择</button>
  </div>
</template>