package dto

// RatingRequest 表示图片评分请求。
type RatingRequest struct {
	Score int `json:"score" vd:"$ >= 0 && $ <= 100"`
}

// RatingResponse 表示图片评分响应。
type RatingResponse struct {
	ImageID   int64  `json:"image_id"`
	UserID    int64  `json:"user_id"`
	Score     int    `json:"score"`
	UpdatedAt string `json:"updated_at"`
}
