import { apiCall, unwrapResponse } from './transport'
import type { ApiResponse } from './transport'
import type {
  ImageTagResponse,
  TagResponse,
} from './types'

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
