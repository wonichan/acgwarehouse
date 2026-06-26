package do

import "time"

const (
	// ImageStatusActive 表示图片可公开展示。
	ImageStatusActive ImageStatus = "active"
	// ImageStatusDeleted 表示图片已软删除。
	ImageStatusDeleted ImageStatus = "deleted"
)

// ImageStatus 定义图片生命周期状态。
type ImageStatus string

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
}

// IsActive 判断图片是否处于可展示状态。
func (i Image) IsActive() bool {
	return i.Status == "" || i.Status == ImageStatusActive
}

// NormalizeForCreate 补齐图片创建时的默认字段。
func (i Image) NormalizeForCreate(now time.Time) Image {
	if i.Status == "" {
		i.Status = ImageStatusActive
	}
	if i.CreatedAt.IsZero() {
		i.CreatedAt = now.UTC()
	}
	if !i.LastModified.IsZero() {
		i.LastModified = i.LastModified.UTC()
	}
	return i
}
