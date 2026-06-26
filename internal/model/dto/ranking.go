package dto

// RankingResponse 表示热榜单项响应。
type RankingResponse struct {
	Period        string        `json:"period"`
	Rank          int           `json:"rank"`
	Score         float64       `json:"score"`
	BayesianScore float64       `json:"bayesian_score"`
	RatingCount   int64         `json:"rating_count"`
	FavoriteCount int64         `json:"favorite_count"`
	ViewCount     int64         `json:"view_count"`
	ComputedAt    string        `json:"computed_at"`
	Image         ImageResponse `json:"image"`
}
