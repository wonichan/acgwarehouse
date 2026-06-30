import type { ArtItem, CarouselSlide } from '@/types'
import { getRankings } from '@/api/client'
import type { RankingResponse } from '@/api/client'
import { hasDisplayableImageItem } from '@/utils/imagePresentation'

export const COMMUNITY_FOCUS_PERIOD = 'week' as const
export const COMMUNITY_FOCUS_REQUEST_LIMIT = 20
export const COMMUNITY_FOCUS_DISPLAY_LIMIT = 10

const carouselVariants: Array<'rain' | 'character' | 'album'> = ['rain', 'character', 'album']
const carouselTags = ['热榜缓存', '真实数据', '社区精选'] as const

export async function getCommunityFocusSlides(): Promise<CarouselSlide[]> {
  const rankingsData = await getRankings({ period: COMMUNITY_FOCUS_PERIOD, limit: COMMUNITY_FOCUS_REQUEST_LIMIT })
  return rankingsData
    .filter(hasDisplayableRankingImage)
    .slice(0, COMMUNITY_FOCUS_DISPLAY_LIMIT)
    .map(rankingToSlide)
}

function hasDisplayableRankingImage(ranking: RankingResponse): boolean {
  return hasDisplayableImageItem(ranking.image)
}

function rankingToSlide(ranking: RankingResponse, index: number): CarouselSlide {
  const image = ranking.image
  const score = Number.isFinite(ranking.score) ? ranking.score : 0
  return {
    id: String(image.id),
    title: image.filename,
    description: `排名第${ranking.rank}的热门作品，热度分${score.toFixed(1)}，均分${image.avg_score.toFixed(1)}/100`,
    tag: carouselTags[index % carouselTags.length],
    tagType: index === 0 ? 'hot' : 'normal',
    score,
    favorites: ranking.favorite_count,
    artVariant: carouselVariants[index % carouselVariants.length],
    imageUrl: image.url,
  }
}

export function appendArtItems(current: ArtItem[], nextItems: readonly ArtItem[]): ArtItem[] {
  return [...current, ...nextItems]
}
