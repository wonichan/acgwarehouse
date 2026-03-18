package domain

import "time"

// Thumbnail represents a generated thumbnail image
type Thumbnail struct {
	Data   []byte
	Width  int
	Height int
	Size   string
}

type Image struct {
	ID                int64     `json:"id"`
	Path              string    `json:"path"`
	Filename          string    `json:"filename"`
	SourceRoot        string    `json:"source_root"`
	FileSize          int64     `json:"file_size"`
	Width             int       `json:"width"`
	Height            int       `json:"height"`
	Format            string    `json:"format"`
	PHash             int64     `json:"phash"`
	ThumbnailSmallUrl string    `json:"thumbnail_small_url"`
	ThumbnailLargeUrl string    `json:"thumbnail_large_url"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
