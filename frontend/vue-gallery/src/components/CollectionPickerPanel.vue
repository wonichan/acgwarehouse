<script setup lang="ts">
import { ref } from 'vue'
import { useCollectionPicker } from '@/composables/useCollectionPicker'
import type { CollectionVisibility } from '@/api/client'

const props = defineProps<{
  imageIds: readonly number[]
}>()

const emit = defineEmits<{
  success: []
  close: []
}>()

const {
  collections,
  loading,
  error: pickerError,
  creating,
  adding,
  loadCollections,
  createNewCollection,
  addToCollection,
} = useCollectionPicker()

const visible = ref(false)
const selectedCollectionId = ref<number | null>(null)
const newCollectionName = ref('')
const newCollectionVisibility = ref<CollectionVisibility>('private')

async function open(): Promise<void> {
  if (collections.value.length === 0 && !loading.value) {
    await loadCollections()
  }
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

async function handleAdd(): Promise<void> {
  if (selectedCollectionId.value === null) return
  const collectionId = selectedCollectionId.value
  const results = await Promise.all(props.imageIds.map((id) => addToCollection(collectionId, id)))
  if (results.every((result) => result)) {
    selectedCollectionId.value = null
    close()
    emit('success')
  }
}

async function handleCreateAndAdd(): Promise<void> {
  const name = newCollectionName.value.trim()
  if (!name) return
  const collection = await createNewCollection(name, newCollectionVisibility.value)
  if (!collection) return
  const results = await Promise.all(props.imageIds.map((id) => addToCollection(collection.id, id)))
  if (results.every((result) => result)) {
    newCollectionName.value = ''
    selectedCollectionId.value = null
    close()
    emit('success')
  }
}

defineExpose({ toggle })
</script>

<template>
  <div v-if="visible" class="panel panel-raised picker-panel">
    <p class="eyebrow">选择收藏夹</p>

    <div v-if="loading" class="status status--loading is-visible" role="status">
      <span class="status-dot"></span>
      正在加载收藏夹...
    </div>
    <div v-else-if="pickerError" class="status status--error is-visible" role="alert">
      {{ pickerError }}
      <button class="btn btn-secondary btn-small" type="button" @click="loadCollections">重试</button>
    </div>
    <div v-else-if="collections.length === 0" class="activity-empty">
      <p class="activity-empty__title">暂无收藏夹</p>
      <p class="activity-empty__desc">创建一个新收藏夹来保存作品。</p>
    </div>
    <div v-else class="stack picker-list">
      <label
        v-for="collection in collections"
        :key="collection.id"
        class="row picker-row"
        :class="{ 'is-selected': selectedCollectionId === collection.id }"
      >
        <input
          type="radio"
          :value="collection.id"
          v-model.number="selectedCollectionId"
          class="sr-only"
        />
        <span class="tag">{{ collection.name }}</span>
        <span class="meta">{{ collection.visibility === 'public' ? '公开' : '私有' }} · {{ collection.items.length }} 张</span>
      </label>
    </div>

    <div class="divider"></div>

    <div class="form-grid">
      <label class="field">
        新建收藏夹
        <input class="input" v-model="newCollectionName" placeholder="输入收藏夹名称" :disabled="creating" />
      </label>
      <label class="field">
        可见性
        <select class="select" v-model="newCollectionVisibility" :disabled="creating">
          <option value="private">私有</option>
          <option value="public">公开</option>
        </select>
      </label>
      <div class="grid-2">
        <button
          class="btn btn-primary"
          type="button"
          :disabled="adding || creating || selectedCollectionId === null"
          @click="handleAdd"
        >
          {{ adding ? '添加中...' : '添加' }}
        </button>
        <button
          class="btn btn-secondary"
          type="button"
          :disabled="creating || !newCollectionName.trim()"
          @click="handleCreateAndAdd"
        >
          {{ creating ? '创建中...' : '创建并添加' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.picker-panel { margin-top: var(--space-4); }
.picker-list { gap: var(--space-2); }
.picker-row {
  cursor: pointer;
  padding: var(--space-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  transition:
    border-color var(--motion-fast) var(--ease-standard),
    background var(--motion-fast) var(--ease-standard);
}
.picker-row.is-selected {
  border-color: var(--accent);
  background: color-mix(in oklab, var(--accent), transparent 92%);
}
</style>
