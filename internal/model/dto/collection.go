package dto

// CollectionCreateRequest 表示创建收藏夹请求。
type CollectionCreateRequest struct {
	Name       string `json:"name" vd:"len($) > 0 && len($) <= 64"`
	Visibility string `json:"visibility" vd:"$ == '' || $ == 'private' || $ == 'public'"`
}

// CollectionUpdateRequest 表示更新收藏夹请求。
type CollectionUpdateRequest struct {
	Name         string `json:"name" vd:"len($) > 0 && len($) <= 64"`
	Visibility   string `json:"visibility" vd:"$ == 'private' || $ == 'public'"`
	CoverImageID *int64 `json:"cover_image_id"`
}

// CollectionItemRequest 表示收藏夹新增图片请求。
type CollectionItemRequest struct {
	ImageID int64 `json:"image_id" vd:"$ > 0"`
}

// CollectionResponse 表示收藏夹响应。
type CollectionResponse struct {
	ID            int64                    `json:"id"`
	UserID        int64                    `json:"user_id"`
	Name          string                   `json:"name"`
	Visibility    string                   `json:"visibility"`
	CreatedAt     string                   `json:"created_at"`
	CoverImageID  int64                    `json:"cover_image_id"`
	CoverImageURL string                   `json:"cover_image_url"`
	Items         []CollectionItemResponse `json:"items,omitempty"`
}

// CollectionItemResponse 表示收藏夹图片条目响应。
type CollectionItemResponse struct {
	CollectionID int64         `json:"collection_id"`
	ImageID      int64         `json:"image_id"`
	CreatedAt    string        `json:"created_at"`
	Image        *ImageResponse `json:"image,omitempty"`
}
