import { apiCall, unwrapResponse } from './transport'
import type {
  CollectionDetailResponse,
  CollectionResponse,
  CollectionVisibility,
  ImageDetailResponse,
  ImageItem,
  ImageListResponse,
  ImageQuery,
  ImageTagResponse,
  RankingQuery,
  RankingResponse,
  RatingResponse,
  SearchQuery,
  TagResponse,
  UserPasswordUpdateRequest,
  UserProfileUpdateRequest,
  UserResponse,
} from './types'
import type { ApiResponse } from './transport'
export { ApiError, clearToken, isAuthenticated, setToken } from './transport'
export type {
  CollectionDetailResponse,
  CollectionItemResponse,
  CollectionResponse,
  CollectionVisibility,
  ImageDetailResponse,
  ImageItem,
  ImageListResponse,
  ImageQuery,
  ImageSort,
  ImageTagResponse,
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

function normalizeListResponse<T>(response: BackendListResponse<T>): { readonly items: readonly T[]; readonly total: number; readonly page: number; readonly limit: number } {
  return {
    items: response.list,
    total: response.total,
    page: response.page,
    limit: response.size,
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

export async function getTags(): Promise<readonly TagResponse[]> {
  return unwrapResponse(apiCall<ApiResponse<readonly TagResponse[]>>('/tags'))
}

export async function suggestTags(queryText: string, limit?: number): Promise<readonly TagResponse[]> {
  const query = new URLSearchParams()
  appendStringParam(query, 'q', queryText)
  appendNumberParam(query, 'limit', limit)

  return unwrapResponse(
    apiCall<ApiResponse<readonly TagResponse[]>>(queryPath('/tags/suggest', query))
  )
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
  return unwrapResponse(apiCall<ApiResponse<readonly CollectionResponse[]>>('/collections'))
}

export async function getCollection(id: number): Promise<CollectionDetailResponse> {
  return unwrapResponse(apiCall<ApiResponse<CollectionDetailResponse>>(`/collections/${id}`))
}

export async function createCollection(
  name: string,
  visibility: CollectionVisibility = 'private'
): Promise<CollectionResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<CollectionResponse>>(
      '/collections',
      { method: 'POST', body: JSON.stringify({ name, visibility }) }
    )
  )
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

export async function createTag(name: string): Promise<TagResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<TagResponse>>(
      '/tags',
      { method: 'POST', body: JSON.stringify({ name }) }
    )
  )
}

export async function assignTagsToImages(
  imageIds: readonly number[],
  tagIds: readonly number[]
): Promise<readonly ImageTagResponse[]> {
  return unwrapResponse(
    apiCall<ApiResponse<readonly ImageTagResponse[]>>(
      '/images/tags',
      { method: 'POST', body: JSON.stringify({ image_ids: imageIds, tag_ids: tagIds }) }
    )
  )
}

export async function unassignTagsFromImages(
  imageIds: readonly number[],
  tagIds: readonly number[]
): Promise<readonly ImageTagResponse[]> {
  return unwrapResponse(
    apiCall<ApiResponse<readonly ImageTagResponse[]>>(
      '/images/tags',
      { method: 'DELETE', body: JSON.stringify({ image_ids: imageIds, tag_ids: tagIds }) }
    )
  )
}

export const api = {
  login,
  register,
  getCurrentUser,
  updateCurrentUserProfile,
  changeCurrentUserPassword,
  getImages,
  getImage,
  searchImages,
  getTags,
  suggestTags,
  getRankings,
  getCollections,
  getCollection,
  createCollection,
  addImageToCollection,
  rateImage,
  createTag,
  assignTagsToImages,
  unassignTagsFromImages,
}
