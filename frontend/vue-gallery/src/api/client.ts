import { apiCall, unwrapResponse } from './transport'
import { getDailyRecommendations } from './dailyRecommendations'
import {
  assignTagsToImages,
  createTag,
  getTags,
  suggestTags,
  unassignTagsFromImages,
} from './tags'
import type {
  CollectionDetailResponse,
  CollectionItemResponse,
  CollectionResponse,
  CollectionVisibility,
  ImageDetailResponse,
  ImageItem,
  ImageListResponse,
  ImageQuery,
  MonthlyCheckInsResponse,
  RankingQuery,
  RankingResponse,
  RatingResponse,
  SearchQuery,
  UserPasswordUpdateRequest,
  UserProfileUpdateRequest,
  UserResponse,
} from './types'
import type { ApiResponse } from './transport'
export { ApiError, clearToken, isAuthenticated, setToken } from './transport'
export { getDailyRecommendations } from './dailyRecommendations'
export {
  assignTagsToImages,
  createTag,
  getTags,
  suggestTags,
  unassignTagsFromImages,
} from './tags'
export type {
  CollectionDetailResponse,
  CollectionItemResponse,
  CollectionResponse,
  CollectionVisibility,
  DailyRecommendationListResponse,
  ImageDetailResponse,
  ImageItem,
  ImageListResponse,
  ImageQuery,
  ImageSort,
  ImageTagResponse,
  MonthlyCheckInsResponse,
  RankingPeriod,
  RankingQuery,
  RankingResponse,
  RatingResponse,
  SearchQuery,
  SortOrder,
  TagResponse,
  UserPasswordUpdateRequest,
  UserProfileUpdateRequest,
  UserResponse,
} from './types'

interface BackendListResponse<T> {
  readonly list: readonly T[]
  readonly total: number
  readonly page: number
  readonly size: number
}

interface BackendCollectionResponse extends Omit<CollectionResponse, 'items'> {
  readonly items?: readonly CollectionItemResponse[]
}

function normalizeListResponse<T>(response: BackendListResponse<T>): { readonly items: readonly T[]; readonly total: number; readonly page: number; readonly limit: number } {
  return {
    items: response.list,
    total: response.total,
    page: response.page,
    limit: response.size,
  }
}

function normalizeCollectionResponse(response: BackendCollectionResponse): CollectionResponse {
  return {
    ...response,
    items: response.items ?? [],
  }
}

function appendNumberParam(query: URLSearchParams, key: string, value: number | undefined): void {
  if (value !== undefined) query.set(key, String(value))
}

function appendStringParam(query: URLSearchParams, key: string, value: string | undefined): void {
  if (value !== undefined && value.length > 0) query.set(key, value)
}

function queryPath(path: string, query: URLSearchParams): string {
  const queryString = query.toString()
  return queryString.length > 0 ? `${path}?${queryString}` : path
}

export async function login(username: string, password: string): Promise<{ readonly token: string }> {
  return unwrapResponse(
    apiCall<ApiResponse<{ readonly token: string }>>(
      '/users/login',
      {
        method: 'POST',
        body: JSON.stringify({ username, password }),
        skipAuth: true,
      }
    )
  )
}

export async function register(username: string, password: string): Promise<UserResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<UserResponse>>(
      '/users/register',
      {
        method: 'POST',
        body: JSON.stringify({ username, password }),
        skipAuth: true,
      }
    )
  )
}

export async function getCurrentUser(): Promise<UserResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<UserResponse>>('/users/me')
  )
}

export async function updateCurrentUserProfile(input: UserProfileUpdateRequest): Promise<UserResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<UserResponse>>(
      '/users/me',
      { method: 'PUT', body: JSON.stringify(input) }
    )
  )
}

export async function changeCurrentUserPassword(input: UserPasswordUpdateRequest): Promise<void> {
  await unwrapResponse(
    apiCall<ApiResponse<null>>(
      '/users/password',
      { method: 'PUT', body: JSON.stringify(input) }
    )
  )
}

export async function getMonthlyCheckIns(year: number, month: number): Promise<MonthlyCheckInsResponse> {
  const query = new URLSearchParams()
  query.set('year', String(year))
  query.set('month', String(month))
  return unwrapResponse(
    apiCall<ApiResponse<MonthlyCheckInsResponse>>(queryPath('/users/me/check-ins', query))
  )
}

export async function getImages(params?: ImageQuery): Promise<ImageListResponse> {
  const query = new URLSearchParams()
  appendNumberParam(query, 'page', params?.page)
  appendNumberParam(query, 'size', params?.limit)
  appendStringParam(query, 'tag', params?.tag)
  appendStringParam(query, 'filename', params?.filename)
  appendStringParam(query, 'sort', params?.sort)
  appendStringParam(query, 'order', params?.order)

  const response = await unwrapResponse(
    apiCall<ApiResponse<BackendListResponse<ImageItem>>>(queryPath('/images', query))
  )
  return normalizeListResponse(response)
}

export async function getImage(id: number): Promise<ImageDetailResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<ImageDetailResponse>>(`/images/${id}`)
  )
}

export async function searchImages(params: SearchQuery): Promise<ImageListResponse> {
  const query = new URLSearchParams()
  appendStringParam(query, 'q', params.keyword)
  appendNumberParam(query, 'page', params.page)
  appendNumberParam(query, 'size', params.limit)

  const response = await unwrapResponse(
    apiCall<ApiResponse<BackendListResponse<ImageItem>>>(queryPath('/search', query))
  )
  return normalizeListResponse(response)
}

export async function getRankings(params?: RankingQuery): Promise<readonly RankingResponse[]> {
  const query = new URLSearchParams()
  appendStringParam(query, 'period', params?.period ?? 'day')
  appendNumberParam(query, 'page', params?.page)
  appendNumberParam(query, 'size', params?.limit)

  const response = await unwrapResponse(
    apiCall<ApiResponse<BackendListResponse<RankingResponse>>>(queryPath('/rankings', query))
  )
  return response.list
}

export async function getCollections(): Promise<readonly CollectionResponse[]> {
  const response = await unwrapResponse(apiCall<ApiResponse<readonly BackendCollectionResponse[]>>('/collections'))
  return response.map(normalizeCollectionResponse)
}

export async function getCollection(id: number): Promise<CollectionDetailResponse> {
  const response = await unwrapResponse(apiCall<ApiResponse<BackendCollectionResponse>>(`/collections/${id}`))
  return normalizeCollectionResponse(response)
}

export async function createCollection(
  name: string,
  visibility: CollectionVisibility = 'private'
): Promise<CollectionResponse> {
  const response = await unwrapResponse(
    apiCall<ApiResponse<BackendCollectionResponse>>(
      '/collections',
      { method: 'POST', body: JSON.stringify({ name, visibility }) }
    )
  )
  return normalizeCollectionResponse(response)
}

export interface CollectionUpdateInput {
  readonly name: string
  readonly visibility: CollectionVisibility
  readonly cover_image_id?: number
}

export async function updateCollection(id: number, input: CollectionUpdateInput): Promise<CollectionResponse> {
  const response = await unwrapResponse(
    apiCall<ApiResponse<BackendCollectionResponse>>(
      `/collections/${id}`,
      { method: 'PUT', body: JSON.stringify(input) }
    )
  )
  return normalizeCollectionResponse(response)
}

export async function addImageToCollection(collectionId: number, imageId: number): Promise<void> {
  await unwrapResponse(
    apiCall<ApiResponse<null>>(
      `/collections/${collectionId}/items`,
      { method: 'POST', body: JSON.stringify({ image_id: imageId }) }
    )
  )
}

export async function rateImage(imageId: number, score: number): Promise<RatingResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<RatingResponse>>(
      `/images/${imageId}/rating`,
      { method: 'PUT', body: JSON.stringify({ score }) }
    )
  )
}

export const api = {
  login,
  register,
  getCurrentUser,
  updateCurrentUserProfile,
  changeCurrentUserPassword,
  getMonthlyCheckIns,
  getImages,
  getImage,
  searchImages,
  getRankings,
  getDailyRecommendations,
  getCollections,
  getCollection,
  createCollection,
  addImageToCollection,
  rateImage,
  getTags,
  suggestTags,
  createTag,
  assignTagsToImages,
  unassignTagsFromImages,
}
