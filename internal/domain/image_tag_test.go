package domain

import "testing"

func TestImageTagPKReturnsCompositeKey(t *testing.T) {
	t.Parallel()

	sourceObservationID := int64(7)
	imageTag := &ImageTag{
		ImageID:             11,
		TagID:               22,
		SourceObservationID: &sourceObservationID,
		Confidence:          0.85,
		ReviewState:         "pending",
	}

	imageID, tagID := imageTag.PK()
	if imageID != 11 || tagID != 22 {
		t.Fatalf("PK() = (%d, %d), want (11, 22)", imageID, tagID)
	}
}
