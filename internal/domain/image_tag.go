package domain

const (
	ImageTagSourceAI     = "ai"
	ImageTagSourceManual = "manual"
)

const (
	ReviewStatePending   = "pending"
	ReviewStateConfirmed = "confirmed"
	ReviewStateRejected  = "rejected"
)

type ImageTag struct {
	ImageID             int64   `json:"image_id"`
	TagID               int64   `json:"tag_id"`
	Source              string  `json:"source"`
	SourceObservationID *int64  `json:"source_observation_id"`
	Confidence          float64 `json:"confidence"`
	ReviewState         string  `json:"review_state"`
}

func (it *ImageTag) PK() (int64, int64) {
	return it.ImageID, it.TagID
}
