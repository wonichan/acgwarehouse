import { ref } from 'vue'
import { ApiError, addImageToCollection, createCollection, getCollections } from '@/api/client'
import type { CollectionResponse, CollectionVisibility } from '@/api/client'

export function useCollectionPicker() {
  const collections = ref<readonly CollectionResponse[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const creating = ref(false)
  const adding = ref(false)

  async function loadCollections(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      collections.value = await getCollections()
    } catch (e) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = '收藏夹加载失败，请稍后重试'
      }
    } finally {
      loading.value = false
    }
  }

  async function createNewCollection(
    name: string,
    visibility: CollectionVisibility
  ): Promise<CollectionResponse | null> {
    creating.value = true
    try {
      const collection = await createCollection(name, visibility)
      collections.value = [...collections.value, collection]
      return collection
    } catch (e) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = '创建收藏夹失败，请稍后重试'
      }
      return null
    } finally {
      creating.value = false
    }
  }

  async function addToCollection(collectionId: number, imageId: number): Promise<boolean> {
    adding.value = true
    try {
      await addImageToCollection(collectionId, imageId)
      return true
    } catch (e) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = '添加到收藏夹失败，请稍后重试'
      }
      return false
    } finally {
      adding.value = false
    }
  }

  return {
    collections,
    loading,
    error,
    creating,
    adding,
    loadCollections,
    createNewCollection,
    addToCollection,
  }
}
