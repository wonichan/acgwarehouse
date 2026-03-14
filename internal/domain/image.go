package domain

import "time"

type Image struct {
	ID         int64     `json:"id"`
	Path       string    `json:"path"`
	Filename   string    `json:"filename"`
	SourceRoot string    `json:"source_root"`
	FileSize   int64     `json:"file_size"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	Format     string    `json:"format"`
	PHash      int64     `json:"phash"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
