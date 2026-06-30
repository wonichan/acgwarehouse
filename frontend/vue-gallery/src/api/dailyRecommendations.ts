import { apiCall, unwrapResponse } from './transport'
import type { ApiResponse } from './transport'
import type { DailyRecommendationListResponse } from './types'

export async function getDailyRecommendations(): Promise<DailyRecommendationListResponse> {
  return unwrapResponse(
    apiCall<ApiResponse<DailyRecommendationListResponse>>('/daily-recommendations')
  )
}
