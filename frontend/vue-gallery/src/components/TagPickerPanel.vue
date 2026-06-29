<script setup lang="ts">
import { computed, ref } from 'vue'
import { useTagPicker } from '@/composables/useTagPicker'
import type { TagResponse } from '@/api/client'

const props = withDefaults(defineProps<{
  imageIds: readonly number[]
  currentTagNames?: readonly string[]
  allowBulkRemove?: boolean
}>(), {
  currentTagNames: () => [],
  allowBulkRemove: false,
})

const emit = defineEmits<{
  success: [message: string]
  close: []
}>()

const {
  tags: allTags,
  loading,
  error: pickerError,
  creating,
  assigning,
  unassigning,
  loadTags,
  createNewTag,
  assignToImages,
  unassignFromImages,
} = useTagPicker()

const visible = ref(false)
const tagSearch = ref('')
const newTagName = ref('')
const selectedTagIdsToAdd = ref<Set<number>>(new Set())
const selectedTagIdsToRemove = ref<Set<number>>(new Set())

const removeModeEnabled = computed(() => props.allowBulkRemove || props.currentTagNames.length > 0)

const currentTagIds = computed<Set<number>>(() => {
  if (props.allowBulkRemove) return new Set()
  const nameToId = new Map<string, number>()
  for (const tag of allTags.value) nameToId.set(tag.name, tag.id)
  const ids = new Set<number>()
  for (const name of props.currentTagNames) {
    const id = nameToId.get(name)
    if (id !== undefined) ids.add(id)
  }
  return ids
})

const currentTags = computed<readonly TagResponse[]>(() => {
  if (props.allowBulkRemove) return allTags.value
  const nameToTag = new Map<string, TagResponse>()
  for (const tag of allTags.value) nameToTag.set(tag.name, tag)
  const result: TagResponse[] = []
  for (const name of props.currentTagNames) {
    const tag = nameToTag.get(name)
    if (tag) result.push(tag)
  }
  return result
})

const availableTags = computed<readonly TagResponse[]>(() => {
  const current = props.allowBulkRemove ? new Set<number>() : currentTagIds.value
  return allTags.value.filter((tag) => !current.has(tag.id))
})

const filteredAvailableTags = computed<readonly TagResponse[]>(() => {
  const q = tagSearch.value.trim().toLowerCase()
  if (!q) return availableTags.value
  return availableTags.value.filter((tag) => tag.name.toLowerCase().includes(q))
})

async function open(): Promise<void> {
  if (allTags.value.length === 0 && !loading.value) await loadTags()
}

function toggle(): void {
  visible.value = !visible.value
  if (visible.value) {
    open()
  } else {
    emit('close')
  }
}

function close(): void {
  visible.value = false
  emit('close')
}

function toggleTagSelection(set: Set<number>, id: number): void {
  if (set.has(id)) set.delete(id)
  else set.add(id)
}

async function handleAddTags(): Promise<void> {
  const tagIds = Array.from(selectedTagIdsToAdd.value)
  if (tagIds.length === 0) return
  const result = await assignToImages(props.imageIds, tagIds)
  if (result) {
    selectedTagIdsToAdd.value.clear()
    close()
    emit('success', '标签已添加')
  }
}

async function handleRemoveTags(): Promise<void> {
  const tagIds = Array.from(selectedTagIdsToRemove.value)
  if (tagIds.length === 0) return
  const result = await unassignFromImages(props.imageIds, tagIds)
  if (result) {
    selectedTagIdsToRemove.value.clear()
    close()
    emit('success', '标签已移除')
  }
}

async function handleCreateAndAddTag(): Promise<void> {
  const name = newTagName.value.trim()
  if (!name) return
  const tag = await createNewTag(name)
  if (!tag) return
  const result = await assignToImages(props.imageIds, [tag.id])
  if (result) {
    newTagName.value = ''
    close()
    emit('success', '标签已创建并添加')
  }
}

defineExpose({ toggle })
</script>

<template>
  <div v-if="visible" class="panel panel-raised picker-panel">
    <p class="eyebrow">标签管理</p>

    <div v-if="loading" class="status status--loading is-visible" role="status">
      <span class="status-dot"></span>
      正在加载标签...
    </div>
    <div v-else-if="pickerError" class="status status--error is-visible" role="alert">
      {{ pickerError }}
      <button class="btn btn-secondary btn-small" type="button" @click="loadTags">重试</button>
    </div>
    <div v-else>
      <div v-if="removeModeEnabled">
        <p class="meta">{{ allowBulkRemove ? '可移除标签（选择标签后会从所有选中图片移除）' : '当前标签（点击移除）' }}</p>
        <div v-if="currentTags.length === 0" class="activity-empty picker-empty">
          <p class="activity-empty__desc">暂无可移除标签。</p>
        </div>
        <div v-else class="kicker-row">
          <button
            v-for="tag in currentTags"
            :key="tag.id"
            type="button"
            class="tag"
            :class="{ 'is-hot': selectedTagIdsToRemove.has(tag.id) }"
            @click="toggleTagSelection(selectedTagIdsToRemove, tag.id)"
          >
            {{ tag.name }}
          </button>
        </div>
        <button class="btn btn-secondary btn-small" type="button" :disabled="unassigning || selectedTagIdsToRemove.size === 0" @click="handleRemoveTags">
          {{ unassigning ? '移除中...' : '移除选中标签' }}
        </button>
        <div class="divider"></div>
      </div>

      <p class="meta">可选标签（点击添加）</p>
      <input class="input" v-model="tagSearch" placeholder="搜索标签..." aria-label="搜索标签" />
      <div v-if="filteredAvailableTags.length === 0" class="activity-empty picker-empty">
        <p class="activity-empty__desc">没有更多可选标签。</p>
      </div>
      <div v-else class="kicker-row">
        <button
          v-for="tag in filteredAvailableTags"
          :key="tag.id"
          type="button"
          class="tag"
          :class="{ 'is-hot': selectedTagIdsToAdd.has(tag.id) }"
          @click="toggleTagSelection(selectedTagIdsToAdd, tag.id)"
        >
          {{ tag.name }}
        </button>
      </div>
      <button class="btn btn-primary btn-small" type="button" :disabled="assigning || selectedTagIdsToAdd.size === 0" @click="handleAddTags">
        {{ assigning ? '添加中...' : '添加选中标签' }}
      </button>

      <div class="divider"></div>

      <div class="form-grid">
        <label class="field">
          新建标签
          <input class="input" v-model="newTagName" placeholder="输入新标签名称" :disabled="creating" />
        </label>
        <button class="btn btn-secondary btn-small" type="button" :disabled="creating || !newTagName.trim()" @click="handleCreateAndAddTag">
          {{ creating ? '创建中...' : '创建并添加' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>.picker-panel { margin-top: var(--space-4); }.picker-empty { margin-top: var(--space-3); }</style>
