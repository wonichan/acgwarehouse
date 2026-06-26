package dto

// TagResponse 表示标签公开响应。
type TagResponse struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	UsageCount int64  `json:"usage_count"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ImageTagResponse 表示打标后受影响图片及标签响应。
type ImageTagResponse struct {
	Image ImageResponse `json:"image"`
	Tags  []string      `json:"tags"`
}

// TagCreateRequest 表示创建标签请求。
type TagCreateRequest struct {
	Name string `json:"name" vd:"len($) >= 1 && len($) <= 64"`
}

// TagUpdateRequest 表示更新标签请求。
type TagUpdateRequest struct {
	Name string `json:"name" vd:"len($) >= 1 && len($) <= 64"`
}

// ImageTagBatchRequest 表示批量打标或取消打标请求。
type ImageTagBatchRequest struct {
	ImageIDs []int64 `json:"image_ids" vd:"len($) > 0"`
	TagIDs   []int64 `json:"tag_ids" vd:"len($) > 0"`
}
