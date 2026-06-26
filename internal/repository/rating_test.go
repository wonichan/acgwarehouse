package repository_test

import (
	"context"
	"testing"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

func Test_RatingRepository_Upsert_creates_rating_and_updates_image_aggregate(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	ratingRepo := repository.NewRatingRepository(database.Read, database.Write)
	image := mustCreateImage(t, imageRepo, "thumbnails/rating-create.png")

	// When
	rating, err := ratingRepo.Upsert(context.Background(), do.Rating{UserID: 7, ImageID: image.ID, Score: 80})

	// Then
	if err != nil {
		t.Fatalf("upsert rating: %v", err)
	}
	if rating.UserID != 7 || rating.ImageID != image.ID || rating.Score != 80 || rating.UpdatedAt.IsZero() {
		t.Fatalf("rating = %#v, want created rating", rating)
	}
	stored, err := imageRepo.FindActiveByID(context.Background(), image.ID)
	if err != nil {
		t.Fatalf("find rated image: %v", err)
	}
	if stored.AvgScore != 80 || stored.RatingCount != 1 {
		t.Fatalf("image aggregate = avg %.2f count %d, want 80/1", stored.AvgScore, stored.RatingCount)
	}
}

func Test_RatingRepository_Upsert_overwrites_existing_rating_without_duplicate_count(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	ratingRepo := repository.NewRatingRepository(database.Read, database.Write)
	image := mustCreateImage(t, imageRepo, "thumbnails/rating-overwrite.png")
	if _, err := ratingRepo.Upsert(context.Background(), do.Rating{UserID: 7, ImageID: image.ID, Score: 80}); err != nil {
		t.Fatalf("create first rating: %v", err)
	}

	// When
	rating, err := ratingRepo.Upsert(context.Background(), do.Rating{UserID: 7, ImageID: image.ID, Score: 20})

	// Then
	if err != nil {
		t.Fatalf("overwrite rating: %v", err)
	}
	if rating.Score != 20 {
		t.Fatalf("score = %d, want overwritten score 20", rating.Score)
	}
	stored, err := imageRepo.FindActiveByID(context.Background(), image.ID)
	if err != nil {
		t.Fatalf("find rated image: %v", err)
	}
	if stored.AvgScore != 20 || stored.RatingCount != 1 {
		t.Fatalf("image aggregate = avg %.2f count %d, want 20/1", stored.AvgScore, stored.RatingCount)
	}
	var count int64
	if err := database.Read.Model(&po.Rating{}).Where("image_id = ?", image.ID).Count(&count).Error; err != nil {
		t.Fatalf("count ratings: %v", err)
	}
	if count != 1 {
		t.Fatalf("rating rows = %d, want 1", count)
	}
}

func Test_RatingRepository_Upsert_recomputes_average_across_users(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	ratingRepo := repository.NewRatingRepository(database.Read, database.Write)
	image := mustCreateImage(t, imageRepo, "thumbnails/rating-average.png")
	if _, err := ratingRepo.Upsert(context.Background(), do.Rating{UserID: 7, ImageID: image.ID, Score: 80}); err != nil {
		t.Fatalf("create first rating: %v", err)
	}

	// When
	_, err := ratingRepo.Upsert(context.Background(), do.Rating{UserID: 8, ImageID: image.ID, Score: 40})

	// Then
	if err != nil {
		t.Fatalf("create second rating: %v", err)
	}
	stored, err := imageRepo.FindActiveByID(context.Background(), image.ID)
	if err != nil {
		t.Fatalf("find rated image: %v", err)
	}
	if stored.AvgScore != 60 || stored.RatingCount != 2 {
		t.Fatalf("image aggregate = avg %.2f count %d, want 60/2", stored.AvgScore, stored.RatingCount)
	}
}

func Test_RatingRepository_Upsert_records_rating_event(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	ratingRepo := repository.NewRatingRepository(database.Read, database.Write)
	image := mustCreateImage(t, imageRepo, "thumbnails/rating-event.png")

	// When
	_, err := ratingRepo.Upsert(context.Background(), do.Rating{UserID: 7, ImageID: image.ID, Score: 95})

	// Then
	if err != nil {
		t.Fatalf("upsert rating: %v", err)
	}
	var events []po.ImageEvent
	if err := database.Read.Where("image_id = ?", image.ID).Find(&events).Error; err != nil {
		t.Fatalf("list rating events: %v", err)
	}
	if len(events) != 1 || events[0].UserID == nil || *events[0].UserID != 7 {
		t.Fatalf("events = %#v, want one event for user 7", events)
	}
	if events[0].Type != string(do.ImageEventTypeRating) || events[0].Value != 95 {
		t.Fatalf("event = %#v, want rating value 95", events[0])
	}
}
