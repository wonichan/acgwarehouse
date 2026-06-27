const API_BASE = '/api/v1'
const TOKEN_KEY = 'acgwarehouse_token'

function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export function isAuthenticated(): boolean {
  return getToken() !== null
}

async function apiCall<T>(
  path: string,
  options?: RequestInit & { skipAuth?: boolean }
): Promise<T> {
  const token = getToken()
  
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  
  if (token && !options?.skipAuth) {
    headers['Authorization'] = `Bearer ${token}`
  }
  
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...headers,
      ...options?.headers as Record<string, string>,
    },
  })
  
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}))
    throw new ApiError(
      errorData.message || `API Error: ${response.status}`,
      response.status,
      errorData.code
    )
  }
  
  return response.json()
}

export class ApiError extends Error {
  status: number
  code?: string
  
  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

async function unwrapResponse<T>(promise: Promise<ApiResponse<T>>): Promise<T> {
  const response = await promise
  return response.data
}

export interface UserResponse {
  id: number
  username: string
  role: string
  created_at: string
}

export interface ImageItem {
  id: number
  filename: string
  cos_key: string
  url: string
  width: number
  height: number
  size: number
  category: string
  avg_score: number
  rating_count: number
  view_count: number
  favorite_count: number
  last_modified: string
  created_at: string
}

export interface ImageDetailResponse extends ImageItem {
  tags: TagResponse[]
}

interface BackendListResponse<T> {
  list: T[]
  total: number
  page: number
  size: number
}

export interface ImageListResponse {
  items: ImageItem[]
  total: number
  page: number
  limit: number
}

function normalizeListResponse<T>(response: BackendListResponse<T>): { items: T[]; total: number; page: number; limit: number } {
  return {
    items: response.list,
    total: response.total,
    page: response.page,
    limit: response.size,
  }
}

export interface TagResponse {
  id: number
  name: string
  category: string
  count: number
}

export interface RankingResponse {
  period: string
  rank: number
  score: number
  bayesian_score: number
  rating_count: number
  favorite_count: number
  view_count: number
  computed_at: string
  image: ImageItem
}

export interface CollectionResponse {
  id: number
  name: string
  description: string
  item_count: number
  created_at: string
}

export interface CollectionDetailResponse extends CollectionResponse {
  items: ImageItem[]
}

export interface ImageQuery {
  page?: number
  limit?: number
  tag?: string
  category?: string
}

export interface SearchQuery {
  keyword?: string
  tags?: string
  minScore?: number
  page?: number
  limit?: number
}

export async function login(username: string, password: string): Promise<{ token: string }> {
  const response = await apiCall<ApiResponse<{ token: string }>>(
    '/users/login',
    {
      method: 'POST',
      body: JSON.stringify({ username, password }),
      skipAuth: true,
    }
  )
  return response.data
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

export async function getImages(params?: ImageQuery): Promise<ImageListResponse> {
  const query = new URLSearchParams()
  if (params?.page) query.set('page', String(params.page))
  if (params?.limit) query.set('limit', String(params.limit))
  if (params?.tag) query.set('tag', params.tag)
  if (params?.category) query.set('category', params.category)
  
  const queryString = query.toString()
  const path = queryString ? `/images?${queryString}` : '/images'
  
  const response = await unwrapResponse(apiCall<ApiResponse<BackendListResponse<ImageItem>>>(path))
  return normalizeListResponse(response)
}

export async function getImage(id: number): Promise<ImageDetailResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<ImageDetailResponse>>(`/images/${id}`)
  )
}

export async function searchImages(params: SearchQuery): Promise<ImageListResponse> {
  const query = new URLSearchParams()
  if (params.keyword) query.set('keyword', params.keyword)
  if (params.tags) query.set('tags', params.tags)
  if (params.minScore) query.set('min_score', String(params.minScore))
  if (params.page) query.set('page', String(params.page))
  if (params.limit) query.set('limit', String(params.limit))
  
  const response = await unwrapResponse(
    apiCall<ApiResponse<BackendListResponse<ImageItem>>>(`/search?${query.toString()}`)
  )
  return normalizeListResponse(response)
}

export async function getTags(): Promise<TagResponse[]> {
  return unwrapResponse(
    apiCall<ApiResponse<TagResponse[]>>('/tags')
  )
}

export async function suggestTags(prefix: string): Promise<TagResponse[]> {
  return unwrapResponse(
    apiCall<ApiResponse<TagResponse[]>>(`/tags/suggest?prefix=${encodeURIComponent(prefix)}`)
  )
}

export async function getRankings(limit?: number): Promise<RankingResponse[]> {
  const query = limit ? `?limit=${limit}` : ''
  const response = await unwrapResponse(
    apiCall<ApiResponse<BackendListResponse<RankingResponse>>>(`/rankings${query}`)
  )
  return response.list
}

export async function getCollections(): Promise<CollectionResponse[]> {
  return unwrapResponse(
    apiCall<ApiResponse<CollectionResponse[]>>('/collections')
  )
}

export async function getCollection(id: number): Promise<CollectionDetailResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<CollectionDetailResponse>>(`/collections/${id}`)
  )
}

export async function createCollection(name: string, description?: string): Promise<CollectionResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<CollectionResponse>>(
      '/collections',
      {
        method: 'POST',
        body: JSON.stringify({ name, description }),
      }
    )
  )
}

export async function addImageToCollection(collectionId: number, imageId: number): Promise<void> {
  await apiCall<ApiResponse<null>>(
    `/collections/${collectionId}/items`,
    {
      method: 'POST',
      body: JSON.stringify({ image_id: imageId }),
    }
  )
}

export async function rateImage(imageId: number, score: number): Promise<void> {
  await apiCall<ApiResponse<null>>(
    `/images/${imageId}/rating`,
    {
      method: 'PUT',
      body: JSON.stringify({ score }),
    }
  )
}

export const api = {
  login,
  register,
  getCurrentUser,
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
}