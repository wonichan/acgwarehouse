package service_test

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
)

type memoryRatingRepository struct {
	ratings map[[2]int64]do.Rating
	images  map[int64]do.Image
}

func newMemoryRatingRepository() *memoryRatingRepository {
	return &memoryRatingRepository{
		ratings: make(map[[2]int64]do.Rating),
		images: map[int64]do.Image{
			7: {ID: 7, COSKey: "thumbnails/rated.png", Filename: "rated.png", Status: do.ImageStatusActive},
		},
	}
}

func (r *memoryRatingRepository) Upsert(_ context.Context, rating do.Rating) (do.Rating, error) {
	key := [2]int64{rating.UserID, rating.ImageID}
	r.ratings[key] = rating
	r.recomputeAggregate(rating.ImageID)
	return r.ratings[key], nil
}

func (r *memoryRatingRepository) FindImageByID(id int64) do.Image {
	return r.images[id]
}

func (r *memoryRatingRepository) RatingCount() int {
	return len(r.ratings)
}

func (r *memoryRatingRepository) recomputeAggregate(imageID int64) {
	var total int
	var count int64
	for key, rating := range r.ratings {
		if key[1] != imageID {
			continue
		}
		total += rating.Score
		count++
	}
	image := r.images[imageID]
	image.RatingCount = count
	if count > 0 {
		image.AvgScore = float64(total) / float64(count)
	}
	r.images[imageID] = image
}

func Test_RatingService_Rate_accepts_boundary_scores_when_score_is_zero_or_one_hundred(t *testing.T) {
	// Given
	repo := newMemoryRatingRepository()
	svc := service.NewRatingService(repo)

	// When
	zero, zeroErr := svc.Rate(context.Background(), do.Rating{UserID: 3, ImageID: 7, Score: 0})
	hundred, hundredErr := svc.Rate(context.Background(), do.Rating{UserID: 4, ImageID: 7, Score: 100})

	// Then
	if zeroErr != nil || hundredErr != nil {
		t.Fatalf("rate boundary scores: %v %v", zeroErr, hundredErr)
	}
	if zero.Score != 0 || hundred.Score != 100 {
		t.Fatalf("zero=%#v hundred=%#v, want accepted boundary scores", zero, hundred)
	}
}

func Test_RatingService_Rate_returns_validation_error_when_score_out_of_range(t *testing.T) {
	// Given
	repo := newMemoryRatingRepository()
	svc := service.NewRatingService(repo)

	// When
	_, lowErr := svc.Rate(context.Background(), do.Rating{UserID: 3, ImageID: 7, Score: -1})
	_, highErr := svc.Rate(context.Background(), do.Rating{UserID: 3, ImageID: 7, Score: 101})

	// Then
	if !stderrors.Is(lowErr, service.ErrInvalidRatingInput) {
		t.Fatalf("low error = %v, want invalid rating input", lowErr)
	}
	if !stderrors.Is(highErr, service.ErrInvalidRatingInput) {
		t.Fatalf("high error = %v, want invalid rating input", highErr)
	}
	if repo.RatingCount() != 0 {
		t.Fatalf("rating count = %d, want no persisted invalid scores", repo.RatingCount())
	}
}

func Test_RatingService_Rate_overwrites_same_user_rating_and_updates_image_aggregate(t *testing.T) {
	// Given
	repo := newMemoryRatingRepository()
	svc := service.NewRatingService(repo)
	if _, err := svc.Rate(context.Background(), do.Rating{UserID: 3, ImageID: 7, Score: 80}); err != nil {
		t.Fatalf("create first rating: %v", err)
	}

	// When
	updated, err := svc.Rate(context.Background(), do.Rating{UserID: 3, ImageID: 7, Score: 40})

	// Then
	if err != nil {
		t.Fatalf("overwrite rating: %v", err)
	}
	image := repo.FindImageByID(7)
	if updated.Score != 40 || repo.RatingCount() != 1 || image.AvgScore != 40 || image.RatingCount != 1 {
		t.Fatalf("updated=%#v image=%#v rows=%d, want overwritten aggregate", updated, image, repo.RatingCount())
	}
}
