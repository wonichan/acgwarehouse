package dto

// ImageResponse 表示图片公开元数据响应。
type ImageResponse struct {
	ID            int64   `json:"id"`
	COSKey        string  `json:"cos_key"`
	Filename      string  `json:"filename"`
	URL           string  `json:"url"`
	Size          int64   `json:"size"`
	LastModified  string  `json:"last_modified"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	Category      string  `json:"category"`
	AvgScore      float64 `json:"avg_score"`
	RatingCount   int64   `json:"rating_count"`
	FavoriteCount int64   `json:"favorite_count"`
	ViewCount     int64   `json:"view_count"`
	CreatedAt     string  `json:"created_at"`
}

// ImageDetailResponse 表示图片详情聚合响应。
type ImageDetailResponse struct {
	Image         ImageResponse   `json:"image"`
	Tags          []string        `json:"tags"`
	AvgScore      float64         `json:"avg_score"`
	RatingCount   int64           `json:"rating_count"`
	FavoriteCount int64           `json:"favorite_count"`
	MyRating      *int            `json:"my_rating"`
	IsCollected   bool            `json:"is_collected"`
	SimilarImages []ImageResponse `json:"similar_images"`
}
