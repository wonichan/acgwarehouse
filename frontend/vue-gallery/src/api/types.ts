export type RankingPeriod = 'day' | 'week' | 'month'
export type CollectionVisibility = 'private' | 'public'
export type ImageSort = 'created_at' | 'size' | 'tag'
export type SortOrder = 'asc' | 'desc'

export interface UserResponse {
  readonly id: number
  readonly username: string
  readonly role: string
  readonly created_at: string
}

export interface ImageItem {
  readonly id: number
  readonly filename: string
  readonly cos_key: string
  readonly url: string
  readonly width: number
  readonly height: number
  readonly size: number
  readonly category: string
  readonly avg_score: number
  readonly rating_count: number
  readonly view_count: number
  readonly favorite_count: number
  readonly last_modified: string
  readonly created_at: string
}

export interface ImageDetailResponse {
  readonly image: ImageItem
  readonly tags: readonly string[]
  readonly avg_score: number
  readonly rating_count: number
  readonly favorite_count: number
  readonly my_rating: number | null
  readonly is_collected: boolean
  readonly similar_images: readonly ImageItem[]
}

export interface ImageListResponse {
  readonly items: readonly ImageItem[]
  readonly total: number
  readonly page: number
  readonly limit: number
}

export interface TagResponse {
  readonly id: number
  readonly name: string
  readonly usage_count: number
  readonly created_at: string
  readonly updated_at: string
}

export interface RankingResponse {
  readonly period: RankingPeriod
  readonly rank: number
  readonly score: number
  readonly bayesian_score: number
  readonly rating_count: number
  readonly favorite_count: number
  readonly view_count: number
  readonly computed_at: string
  readonly image: ImageItem
}

export interface CollectionItemResponse {
  readonly collection_id: number
  readonly image_id: number
  readonly created_at: string
}

export interface CollectionResponse {
  readonly id: number
  readonly user_id: number
  readonly name: string
  readonly visibility: CollectionVisibility
  readonly created_at: string
  readonly updated_at?: string
  readonly items: readonly CollectionItemResponse[]
}

export interface CollectionDetailResponse extends CollectionResponse {}

export interface RatingResponse {
  readonly image_id: number
  readonly user_id: number
  readonly score: number
  readonly updated_at: string
}

export interface ImageQuery {
  readonly page?: number
  readonly limit?: number
  readonly tag?: string
  readonly filename?: string
  readonly sort?: ImageSort
  readonly order?: SortOrder
}

export interface SearchQuery {
  readonly keyword?: string
  readonly page?: number
  readonly limit?: number
}

export interface RankingQuery {
  readonly period?: RankingPeriod
  readonly page?: number
  readonly limit?: number
}
