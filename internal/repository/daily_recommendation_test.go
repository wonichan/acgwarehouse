package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

func Test_DailyRecommendationRepository_GetOrCreateToday_returns_stable_ten_images_when_same_date(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	dailyRepo := repository.NewDailyRecommendationRepositoryWithShuffler(
		database.Read,
		database.Write,
		repository.IdentityDailyRecommendationShuffler,
	)
	mustCreateDailyImages(t, imageRepo, 12)

	// When
	first, firstErr := dailyRepo.GetOrCreateToday(context.Background(), "2026-06-30", 10, fixedDailyNow())
	second, secondErr := dailyRepo.GetOrCreateToday(context.Background(), "2026-06-30", 10, fixedDailyNow().Add(time.Hour))

	// Then
	if firstErr != nil || secondErr != nil {
		t.Fatalf("get daily recommendations: first=%v second=%v", firstErr, secondErr)
	}
	if len(first.Images) != 10 || len(second.Images) != 10 {
		t.Fatalf("lengths = %d/%d, want 10/10", len(first.Images), len(second.Images))
	}
	assertImageIDsEqual(t, imageIDs(first.Images), imageIDs(second.Images))
	assertUniqueImageIDs(t, first.Images)
}

func Test_DailyRecommendationRepository_GetOrCreateToday_advances_cycle_when_pool_large(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	dailyRepo := repository.NewDailyRecommendationRepositoryWithShuffler(
		database.Read,
		database.Write,
		repository.IdentityDailyRecommendationShuffler,
	)
	mustCreateDailyImages(t, imageRepo, 25)

	// When
	dayOne, dayOneErr := dailyRepo.GetOrCreateToday(context.Background(), "2026-06-30", 10, fixedDailyNow())
	dayTwo, dayTwoErr := dailyRepo.GetOrCreateToday(
		context.Background(),
		"2026-07-01",
		10,
		fixedDailyNow().AddDate(0, 0, 1),
	)
	dayThree, dayThreeErr := dailyRepo.GetOrCreateToday(
		context.Background(),
		"2026-07-02",
		10,
		fixedDailyNow().AddDate(0, 0, 2),
	)

	// Then
	if dayOneErr != nil || dayTwoErr != nil || dayThreeErr != nil {
		t.Fatalf("get daily recommendations: day1=%v day2=%v day3=%v", dayOneErr, dayTwoErr, dayThreeErr)
	}
	assertImageIDsEqual(t, imageIDs(dayOne.Images), dailyRange(1, 10))
	assertImageIDsEqual(t, imageIDs(dayTwo.Images), dailyRange(11, 20))
	assertImageIDsEqual(t, imageIDs(dayThree.Images), append(dailyRange(21, 25), dailyRange(1, 5)...))
}

func Test_DailyRecommendationRepository_GetOrCreateToday_returns_small_pool_without_duplicates(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	dailyRepo := repository.NewDailyRecommendationRepositoryWithShuffler(
		database.Read,
		database.Write,
		repository.IdentityDailyRecommendationShuffler,
	)
	mustCreateDailyImages(t, imageRepo, 3)

	// When
	result, err := dailyRepo.GetOrCreateToday(context.Background(), "2026-06-30", 10, fixedDailyNow())

	// Then
	if err != nil {
		t.Fatalf("get daily recommendations: %v", err)
	}
	assertImageIDsEqual(t, imageIDs(result.Images), dailyRange(1, 3))
	assertUniqueImageIDs(t, result.Images)
}

func Test_DailyRecommendationRepository_GetOrCreateToday_filters_soft_deleted_today_result_and_refills(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	dailyRepo := repository.NewDailyRecommendationRepositoryWithShuffler(
		database.Read,
		database.Write,
		repository.IdentityDailyRecommendationShuffler,
	)
	images := mustCreateDailyImages(t, imageRepo, 12)
	first, err := dailyRepo.GetOrCreateToday(context.Background(), "2026-06-30", 10, fixedDailyNow())
	if err != nil {
		t.Fatalf("create daily recommendations: %v", err)
	}
	if err := imageRepo.SoftDelete(context.Background(), first.Images[0].ID, fixedDailyNow()); err != nil {
		t.Fatalf("soft delete recommended image: %v", err)
	}

	// When
	result, err := dailyRepo.GetOrCreateToday(context.Background(), "2026-06-30", 10, fixedDailyNow().Add(time.Hour))

	// Then
	if err != nil {
		t.Fatalf("refill daily recommendations: %v", err)
	}
	if len(result.Images) != 10 {
		t.Fatalf("images = %#v, want refilled 10 images", result.Images)
	}
	for _, image := range result.Images {
		if image.ID == images[0].ID {
			t.Fatalf("images = %#v, want soft-deleted image removed", result.Images)
		}
	}
	if result.Images[len(result.Images)-1].ID != images[10].ID {
		t.Fatalf("last image id = %d, want next pool image %d", result.Images[len(result.Images)-1].ID, images[10].ID)
	}
}

func Test_DailyRecommendationRepository_GetOrCreateToday_adds_restored_active_image_to_current_pool(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	dailyRepo := repository.NewDailyRecommendationRepositoryWithShuffler(
		database.Read,
		database.Write,
		repository.IdentityDailyRecommendationShuffler,
	)
	images := mustCreateDailyImages(t, imageRepo, 10)
	restored, err := imageRepo.UpsertByCOSKey(context.Background(), do.Image{
		COSKey:       "thumbnails/daily-restored.png",
		Filename:     "daily-restored.png",
		Status:       do.ImageStatusDeleted,
		DeletedAt:    fixedDailyNow(),
		LastModified: fixedDailyNow(),
	})
	if err != nil {
		t.Fatalf("create deleted image: %v", err)
	}
	if _, err := dailyRepo.GetOrCreateToday(context.Background(), "2026-06-30", 10, fixedDailyNow()); err != nil {
		t.Fatalf("create first daily recommendations: %v", err)
	}
	if _, err := imageRepo.Restore(context.Background(), restored.ID); err != nil {
		t.Fatalf("restore image: %v", err)
	}

	// When
	result, err := dailyRepo.GetOrCreateToday(context.Background(), "2026-07-01", 10, fixedDailyNow().AddDate(0, 0, 1))

	// Then
	if err != nil {
		t.Fatalf("get next daily recommendations: %v", err)
	}
	wantIDs := append([]int64{restored.ID}, imageIDs(images[:9])...)
	assertImageIDsEqual(t, imageIDs(result.Images), wantIDs)
}

func Test_DailyRecommendationRepository_GetOrCreateToday_refills_same_day_without_reusing_existing_rows(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	shuffleCalls := 0
	dailyRepo := repository.NewDailyRecommendationRepositoryWithShuffler(
		database.Read,
		database.Write,
		func(ids []int64) {
			shuffleCalls++
			if shuffleCalls == 2 {
				copy(ids, []int64{1, 2, 3, 4, 5, 11, 12, 13, 14, 15, 6, 7, 8, 9, 10})
			}
		},
	)
	images := mustCreateDailyImages(t, imageRepo, 15)
	if _, err := dailyRepo.GetOrCreateToday(context.Background(), "2026-06-30", 10, fixedDailyNow()); err != nil {
		t.Fatalf("create first daily recommendations: %v", err)
	}
	dayTwo, err := dailyRepo.GetOrCreateToday(context.Background(), "2026-07-01", 10, fixedDailyNow().AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("create mixed-cycle daily recommendations: %v", err)
	}
	for _, image := range dayTwo.Images[5:] {
		if err := imageRepo.SoftDelete(context.Background(), image.ID, fixedDailyNow()); err != nil {
			t.Fatalf("soft delete cycle-two image %d: %v", image.ID, err)
		}
	}

	// When
	result, err := dailyRepo.GetOrCreateToday(
		context.Background(),
		"2026-07-01",
		10,
		fixedDailyNow().AddDate(0, 0, 1).Add(time.Hour),
	)

	// Then
	if err != nil {
		t.Fatalf("refill same-day recommendations: %v", err)
	}
	wantIDs := append(imageIDs(images[10:15]), imageIDs(images[5:10])...)
	assertImageIDsEqual(t, imageIDs(result.Images), wantIDs)
}

func Test_DailyRecommendationRepository_AutoMigrate_creates_daily_tables(t *testing.T) {
	// Given
	database := openTestDatabase(t)

	// When
	err := database.Read.Create(&po.DailyRecommendationState{Key: "test", Cycle: 1, UpdatedAt: fixedDailyNow()}).Error

	// Then
	if err != nil {
		t.Fatalf("insert daily recommendation state: %v", err)
	}
}
