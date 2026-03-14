package domain

import "time"

type Tag struct {
	ID              int64     `json:"id"`
	PreferredLabel  string    `json:"preferred_label"`
	Slug            string    `json:"slug"`
	PrimaryCategory string    `json:"primary_category"`
	ReviewState     string    `json:"review_state"`
	TrustScore      float64   `json:"trust_score"`
	UsageCount      int       `json:"usage_count"`
	CreatedAt       time.Time `json:"created_at"`
}
