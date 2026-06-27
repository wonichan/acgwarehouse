// 作品卡片数据类型
export interface ArtItem {
  id: string
  title: string
  tags: string[]
  score: number
  favorites: number
  previewVariant: 'default' | 'tall' | 'wide'
  imageUrl?: string
}

// 轮播数据类型
export interface CarouselSlide {
  id: string
  title: string
  description: string
  tag: string
  tagType: 'hot' | 'normal'
  score: number
  favorites: number
  artVariant: 'rain' | 'character' | 'album'
}

// 相册数据类型
export interface Album {
  id: string
  name: string
  tags: string[]
  count: number
  lastUpdated: string
}

// 热榜项数据类型
export interface RankItem {
  id: string
  title: string
  meta: string
  variant: 'primary' | 'secondary'
}

// 搜索结果项数据类型
export interface SearchResult {
  id: string
  title: string
  filename: string
  tags: string[]
  score: number
  status: string
}

// 表单状态类型
export type FormStatus = 'idle' | 'loading' | 'success' | 'error'

// 用户信息类型
export interface UserInfo {
  nickname: string
  avatar: string
  bio: string
  favorites: number
  tags: number
  preferenceTags: string[]
}