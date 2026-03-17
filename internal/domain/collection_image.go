package domain

import "time"

// CollectionImage represents the many-to-many relationship between collections and images
type CollectionImage struct {
	CollectionID int64     `json:"collection_id" db:"collection_id"`
	ImageID      int64     `json:"image_id" db:"image_id"`
	AddedAt      time.Time `json:"added_at" db:"added_at"`
}

// TableName returns the database table name for CollectionImage
func (CollectionImage) TableName() string {
	return "collection_images"
}
