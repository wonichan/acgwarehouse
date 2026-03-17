package domain

import "time"

// Collection represents a user's image collection/folder
type Collection struct {
	ID           int64     `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Description  string    `json:"description" db:"description"`
	CoverImageID *int64    `json:"cover_image_id" db:"cover_image_id"`
	ImageCount   int       `json:"image_count" db:"image_count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// TableName returns the database table name for Collection
func (Collection) TableName() string {
	return "collections"
}
