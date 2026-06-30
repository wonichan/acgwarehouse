package do

import "time"

const (
	// ImageStatusActive 表示图片可公开展示。
	ImageStatusActive ImageStatus = "active"
	// ImageStatusDeleted 表示图片已软删除。
	ImageStatusDeleted ImageStatus = "deleted"
	// ImageEventTypeView 表示图片详情浏览事件。
	ImageEventTypeView ImageEventType = "view"
	// ImageEventTypeRating 表示图片评分事件。
	ImageEventTypeRating ImageEventType = "rating"
	// ImageEventTypeFavorite 表示图片收藏事件。
	ImageEventTypeFavorite ImageEventType = "favorite"
)

// ImageStatus 定义图片生命周期状态。
type ImageStatus string

// ImageEventType 定义图片行为事件类型。
type ImageEventType string

// ImageSearchQuery 定义图片全文搜索条件。
type ImageSearchQuery struct {
	Text string
	Page int
	Size int
}

// ImageSearchResult 定义图片全文搜索结果。
type ImageSearchResult struct {
	IDs   []int64
	Total int64
}

// Image 表示图片领域对象。
type Image struct {
	ID            int64
	COSKey        string
	Filename      string
	Size          int64
	LastModified  time.Time
	Width         int
	Height        int
	Category      string
	AvgScore      float64
	RatingCount   int64
	FavoriteCount int64
	ViewCount     int64
	Status        ImageStatus
	DeletedAt     time.Time
	CreatedAt     time.Time
	Tags          []string
	URL           string
}

// ImageEvent 表示图片行为事件领域对象。
type ImageEvent struct {
	ID        int64
	ImageID   int64
	UserID    int64
	Type      ImageEventType
	Value     int
	CreatedAt time.Time
}

// IsActive 判断图片是否处于可展示状态。
func (i Image) IsActive() bool {
	return i.Status == "" || i.Status == ImageStatusActive
}
