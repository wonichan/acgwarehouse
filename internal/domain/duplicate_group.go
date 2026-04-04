package domain

import (
	"encoding/json"
	"time"
)

// DuplicateGroup 表示一组重复/相似的图片
type DuplicateGroup struct {
	ID                  int64     `json:"id"`
	RecommendedImageID  int64     `json:"recommended_image_id"` // 推荐保留的图片（分辨率最高）
	SimilarityThreshold int       `json:"similarity_threshold"` // 使用的汉明距离阈值
	CreatedAt           time.Time `json:"created_at"`
}

// DuplicateRelation 表示图片与重复组的关系
type DuplicateRelation struct {
	GroupID                 int64           `json:"group_id"`
	ImageID                 int64           `json:"image_id"`
	IsRecommended           bool            `json:"is_recommended"`
	FileHash                string          `json:"file_hash"` // SHA256 文件哈希
	PHashDistance           int             `json:"phash_distance"`
	RecommendationScore     float64         `json:"recommendation_score"`
	RecommendationRationale json.RawMessage `json:"recommendation_rationale"`
}

// DuplicateGroupWithImages 包含重复组及其关联图片的完整信息
type DuplicateGroupWithImages struct {
	Group  DuplicateGroup   `json:"group"`
	Images []DuplicateImage `json:"images"`
}

// DuplicateImage 重复组中的图片信息
type DuplicateImage struct {
	ID                      int64           `json:"id"`
	Path                    string          `json:"path"`
	Filename                string          `json:"filename"`
	SourceRoot              string          `json:"source_root"`
	Width                   int             `json:"width"`
	Height                  int             `json:"height"`
	FileSize                int64           `json:"file_size"`
	Format                  string          `json:"format"`
	PHash                   int64           `json:"phash"`
	PHashHex                string          `json:"phash_hex"`
	ThumbnailSmallUrl       string          `json:"thumbnail_small_url"`
	ThumbnailLargeUrl       string          `json:"thumbnail_large_url"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
	IsRecommended           bool            `json:"is_recommended"`
	FileHash                string          `json:"file_hash"`
	PHashDistance           int             `json:"phash_distance"`
	RecommendationScore     float64         `json:"recommendation_score"`
	RecommendationRationale json.RawMessage `json:"recommendation_rationale"`
}
