package domain

import "time"

const (
	TagLevelRoot   = "root"
	TagLevelParent = "parent"
	TagLevelChild  = "child"
)

type Tag struct {
	ID              int64     `json:"id"`
	PreferredLabel  string    `json:"preferred_label"`
	Slug            string    `json:"slug"`
	Level           string    `json:"level"`
	ParentID        *int64    `json:"parent_id"`
	PrimaryCategory string    `json:"primary_category"`
	ReviewState     string    `json:"review_state"`
	TrustScore      float64   `json:"trust_score"`
	UsageCount      int       `json:"usage_count"`
	CreatedAt       time.Time `json:"created_at"`
}
