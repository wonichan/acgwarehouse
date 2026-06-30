package dto

// DailyRecommendationResponse 表示今日每日推荐 HTTP 响应。
type DailyRecommendationResponse struct {
	Date     string          `json:"date"`
	Timezone string          `json:"timezone"`
	Total    int             `json:"total"`
	List     []ImageResponse `json:"list"`
}
