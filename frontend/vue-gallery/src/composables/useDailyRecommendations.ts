import { ref } from 'vue'
import type { Ref } from 'vue'
import type { ArtItem } from '@/types'
import { ApiError } from '@/api/client'
import { getDailyRecommendations } from '@/api/dailyRecommendations'
import { hasDisplayableImageItem, imageToArtItem } from '@/utils/imagePresentation'

const DAILY_RECOMMENDATION_DISPLAY_LIMIT = 10

type DailyRecommendationResult =
  | { readonly kind: 'success'; readonly items: ArtItem[] }
  | { readonly kind: 'api-error'; readonly message: string }
  | { readonly kind: 'unknown-error' }

export interface DailyRecommendationsState {
  readonly items: Ref<ArtItem[]>
  readonly loading: Ref<boolean>
  readonly error: Ref<string | null>
  readonly load: () => Promise<void>
  readonly reset: () => void
}

export function useDailyRecommendations(): DailyRecommendationsState {
  const items = ref<ArtItem[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function load(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      applyResult(await getDailyRecommendationResult())
    } finally {
      loading.value = false
    }
  }

  function reset(): void {
    error.value = null
    items.value = []
  }

  function applyResult(result: DailyRecommendationResult): void {
    switch (result.kind) {
      case 'success':
        items.value = result.items
        return
      case 'api-error':
        items.value = []
        error.value = result.message
        return
      case 'unknown-error':
        items.value = []
        error.value = '每日推荐暂时不可用'
        return
      default:
        return assertNeverDailyRecommendationResult(result)
    }
  }

  return { items, loading, error, load, reset }
}

async function getDailyRecommendationResult(): Promise<DailyRecommendationResult> {
  try {
    const dailyData = await getDailyRecommendations()
    return {
      kind: 'success',
      items: dailyData.list
        .filter(hasDisplayableImageItem)
        .slice(0, DAILY_RECOMMENDATION_DISPLAY_LIMIT)
        .map(imageToArtItem),
    }
  } catch (e) {
    if (e instanceof ApiError) {
      return { kind: 'api-error', message: e.message }
    }
    return { kind: 'unknown-error' }
  }
}

function assertNeverDailyRecommendationResult(result: never): never {
  throw new Error(`Unexpected daily recommendation result: ${JSON.stringify(result)}`)
}
