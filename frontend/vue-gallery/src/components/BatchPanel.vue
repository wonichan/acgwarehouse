<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAuth } from '@/composables/useAuth'
import { useSelection } from '@/composables/useSelection'
import { useToast } from '@/composables/useToast'
import AuthRequiredStatus from '@/components/AuthRequiredStatus.vue'
import CollectionPickerPanel from '@/components/CollectionPickerPanel.vue'
import TagPickerPanel from '@/components/TagPickerPanel.vue'

const { selectedCount, selectedIds, clear } = useSelection()
const { show } = useToast()
const { isLoggedIn } = useAuth()

const validImageIds = computed<readonly number[]>(() => {
  const ids: number[] = []
  for (const id of selectedIds) {
    const parsed = Number(id)
    if (Number.isInteger(parsed) && parsed > 0) {
      ids.push(parsed)
    }
  }
  return ids
})

const collectionPickerRef = ref<InstanceType<typeof CollectionPickerPanel> | null>(null)
const tagPickerRef = ref<InstanceType<typeof TagPickerPanel> | null>(null)
const batchActionError = ref<string | null>(null)
const batchAuthRequired = ref<string | null>(null)

function requireLogin(message: string): boolean {
  if (isLoggedIn.value) return false
  batchAuthRequired.value = message
  show(message)
  return true
}

function openCollectionPicker(): void {
  batchActionError.value = null
  batchAuthRequired.value = null
  if (requireLogin('请先登录后再将选中图片加入收藏夹')) return
  if (validImageIds.value.length === 0) {
    batchActionError.value = '没有有效的图片 ID 可供操作'
    return
  }
  collectionPickerRef.value?.open()
}

function openTagPicker(): void {
  batchActionError.value = null
  batchAuthRequired.value = null
  if (requireLogin('请先登录后再批量管理图片标签')) return
  if (validImageIds.value.length === 0) {
    batchActionError.value = '没有有效的图片 ID 可供操作'
    return
  }
  tagPickerRef.value?.toggle()
}

function onSuccess(message: string): void {
  show(message)
  clear()
}

function handleClear(): void {
  batchActionError.value = null
  batchAuthRequired.value = null
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
    <button class="btn btn-secondary btn-small" @click="openCollectionPicker">加入收藏夹</button>
    <button class="btn btn-secondary btn-small" @click="openTagPicker">批量打标签</button>
    <button class="btn btn-secondary btn-small" @click="handleClear">取消选择</button>

    <AuthRequiredStatus v-if="batchAuthRequired" :message="batchAuthRequired" class="batch-status" />
    <div v-if="batchActionError" class="status status--error is-visible batch-status" role="alert">
      {{ batchActionError }}
    </div>

    <CollectionPickerPanel
      v-if="validImageIds.length > 0"
      ref="collectionPickerRef"
      :image-ids="validImageIds"
      @success="onSuccess(`已将 ${validImageIds.length} 张图片添加到收藏夹`)"
    />
    <TagPickerPanel
      v-if="validImageIds.length > 0"
      ref="tagPickerRef"
      :image-ids="validImageIds"
      allow-bulk-remove
      @success="onSuccess($event === '标签已移除' ? `已从 ${validImageIds.length} 张图片移除标签` : `已为 ${validImageIds.length} 张图片添加标签`)"
    />
  </div>
</template>

<style scoped>
.batch-status {
  flex-basis: 100%;
  justify-content: center;
}
</style>
