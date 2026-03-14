package domain

import "time"

type Collection struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CoverImageID *int64    `json:"cover_image_id"`
	ImageCount   int       `json:"image_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
