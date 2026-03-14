package domain

import "time"

type TagObservation struct {
	ID            int64     `json:"id"`
	ImageID       int64     `json:"image_id"`
	RawText       string    `json:"raw_text"`
	Confidence    float64   `json:"confidence"`
	EvidenceType  string    `json:"evidence_type"`
	Provider      string    `json:"provider"`
	ModelName     string    `json:"model_name"`
	PromptVersion string    `json:"prompt_version"`
	CreatedAt     time.Time `json:"created_at"`
}
