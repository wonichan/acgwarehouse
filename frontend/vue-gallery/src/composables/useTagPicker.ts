import { ref } from 'vue'
import { ApiError, assignTagsToImages, createTag, getTags, unassignTagsFromImages } from '@/api/client'
import type { ImageTagResponse, TagResponse } from '@/api/client'

export function useTagPicker() {
  const tags = ref<readonly TagResponse[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const creating = ref(false)
  const assigning = ref(false)
  const unassigning = ref(false)

  async function loadTags(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      tags.value = await getTags()
    } catch (e) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = '标签加载失败，请稍后重试'
      }
    } finally {
      loading.value = false
    }
  }

  async function createNewTag(name: string): Promise<TagResponse | null> {
    creating.value = true
    try {
      const tag = await createTag(name)
      tags.value = [...tags.value, tag]
      return tag
    } catch (e) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = '创建标签失败，请稍后重试'
      }
      return null
    } finally {
      creating.value = false
    }
  }

  async function assignToImages(
    imageIds: readonly number[],
    tagIds: readonly number[]
  ): Promise<readonly ImageTagResponse[] | null> {
    assigning.value = true
    try {
      return await assignTagsToImages(imageIds, tagIds)
    } catch (e) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = '标签添加失败，请稍后重试'
      }
      return null
    } finally {
      assigning.value = false
    }
  }

  async function unassignFromImages(
    imageIds: readonly number[],
    tagIds: readonly number[]
  ): Promise<readonly ImageTagResponse[] | null> {
    unassigning.value = true
    try {
      return await unassignTagsFromImages(imageIds, tagIds)
    } catch (e) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = '标签移除失败，请稍后重试'
      }
      return null
    } finally {
      unassigning.value = false
    }
  }

  return {
    tags,
    loading,
    error,
    creating,
    assigning,
    unassigning,
    loadTags,
    createNewTag,
    assignToImages,
    unassignFromImages,
  }
}
