package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/job"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

func Test_RankingRepository_RecomputeCachesDayWeekMonthWindows_whenEventsExist(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	rankingRepo := repository.NewRankingRepository(database.Read, database.Write)
	now := fixedRankingNow()
	image := mustCreateImage(t, imageRepo, "thumbnails/ranking-window.png")
	mustCreateImageEvent(t, imageRepo, do.ImageEvent{ImageID: image.ID, Type: do.ImageEventTypeView, Value: 1, CreatedAt: now.Add(-23 * time.Hour)})
	mustCreateImageEvent(t, imageRepo, do.ImageEvent{ImageID: image.ID, Type: do.ImageEventTypeView, Value: 1, CreatedAt: now.Add(-25 * time.Hour)})

	// When
	mustRecomputeAllRankings(t, rankingRepo, now)
	day, dayErr := rankingRepo.ListCached(context.Background(), do.RankingPeriodDay, repository.RankingListQuery{Page: 1, Size: 10})
	week, weekErr := rankingRepo.ListCached(context.Background(), do.RankingPeriodWeek, repository.RankingListQuery{Page: 1, Size: 10})
	month, monthErr := rankingRepo.ListCached(context.Background(), do.RankingPeriodMonth, repository.RankingListQuery{Page: 1, Size: 10})

	// Then
	if dayErr != nil || weekErr != nil || monthErr != nil {
		t.Fatalf("list rankings: day=%v week=%v month=%v", dayErr, weekErr, monthErr)
	}
	assertSingleRankingViewCount(t, day, 1)
	assertSingleRankingViewCount(t, week, 2)
	assertSingleRankingViewCount(t, month, 2)
}

func Test_RankingRepository_ListCached_usesCachedRows_whenEventsChangeAfterRecompute(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	rankingRepo := repository.NewRankingRepository(database.Read, database.Write)
	now := fixedRankingNow()
	image := mustCreateImage(t, imageRepo, "thumbnails/ranking-cache.png")
	mustCreateImageEvent(t, imageRepo, do.ImageEvent{ImageID: image.ID, Type: do.ImageEventTypeView, Value: 1, CreatedAt: now})
	mustRecomputeAllRankings(t, rankingRepo, now)
	mustCreateImageEvent(t, imageRepo, do.ImageEvent{ImageID: image.ID, Type: do.ImageEventTypeView, Value: 1, CreatedAt: now})

	// When
	cached, err := rankingRepo.ListCached(context.Background(), do.RankingPeriodDay, repository.RankingListQuery{Page: 1, Size: 10})

	// Then
	if err != nil {
		t.Fatalf("list cached rankings: %v", err)
	}
	assertSingleRankingViewCount(t, cached, 1)
}

func Test_RankingRepository_Recompute_excludesSoftDeletedImages_whenImageDeleted(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	rankingRepo := repository.NewRankingRepository(database.Read, database.Write)
	now := fixedRankingNow()
	active := mustCreateImage(t, imageRepo, "thumbnails/ranking-active.png")
	deleted := mustCreateImage(t, imageRepo, "thumbnails/ranking-deleted.png")
	mustCreateImageEvent(t, imageRepo, do.ImageEvent{ImageID: active.ID, Type: do.ImageEventTypeView, Value: 1, CreatedAt: now})
	mustCreateImageEvent(t, imageRepo, do.ImageEvent{ImageID: deleted.ID, Type: do.ImageEventTypeView, Value: 5, CreatedAt: now})
	if err := imageRepo.SoftDelete(context.Background(), deleted.ID, now); err != nil {
		t.Fatalf("soft delete image: %v", err)
	}

	// When
	mustRecomputeAllRankings(t, rankingRepo, now)
	rankings, err := rankingRepo.ListCached(context.Background(), do.RankingPeriodDay, repository.RankingListQuery{Page: 1, Size: 10})

	// Then
	if err != nil {
		t.Fatalf("list cached rankings: %v", err)
	}
	if len(rankings.List) != 1 || rankings.List[0].Image.ID != active.ID {
		t.Fatalf("rankings = %#v, want only active image", rankings.List)
	}
}

func Test_RankingRepository_Recompute_ordersByFormula_whenRatingBeatsViews(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	rankingRepo := repository.NewRankingRepository(database.Read, database.Write)
	now := fixedRankingNow()
	rated := mustCreateImage(t, imageRepo, "thumbnails/ranking-rated.png")
	viewed := mustCreateImage(t, imageRepo, "thumbnails/ranking-viewed.png")
	mustCreateImageEvent(t, imageRepo, do.ImageEvent{ImageID: rated.ID, UserID: 7, Type: do.ImageEventTypeRating, Value: 100, CreatedAt: now})
	mustCreateImageEvent(t, imageRepo, do.ImageEvent{ImageID: viewed.ID, Type: do.ImageEventTypeView, Value: 1, CreatedAt: now})

	// When
	mustRecomputeAllRankings(t, rankingRepo, now)
	rankings, err := rankingRepo.ListCached(context.Background(), do.RankingPeriodDay, repository.RankingListQuery{Page: 1, Size: 10})

	// Then
	if err != nil {
		t.Fatalf("list cached rankings: %v", err)
	}
	if len(rankings.List) != 2 || rankings.List[0].Image.ID != rated.ID {
		t.Fatalf("rankings = %#v, want high rating first", rankings.List)
	}
}

func Test_RankingJob_RecomputeAll_writesAllPeriods_whenRunOnce(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	rankingRepo := repository.NewRankingRepository(database.Read, database.Write)
	now := fixedRankingNow()
	image := mustCreateImage(t, imageRepo, "thumbnails/ranking-job.png")
	mustCreateImageEvent(t, imageRepo, do.ImageEvent{ImageID: image.ID, Type: do.ImageEventTypeView, Value: 1, CreatedAt: now})
	rankingJob := job.NewRankingJob(rankingRepo, testRankingConfig())

	// When
	err := rankingJob.RecomputeAll(context.Background(), now)

	// Then
	if err != nil {
		t.Fatalf("recompute all rankings: %v", err)
	}
	for _, period := range do.RankingPeriods() {
		rankings, listErr := rankingRepo.ListCached(context.Background(), period, repository.RankingListQuery{Page: 1, Size: 10})
		if listErr != nil {
			t.Fatalf("list %s ranking: %v", period, listErr)
		}
		assertSingleRankingViewCount(t, rankings, 1)
	}
}

func mustRecomputeAllRankings(t *testing.T, repo *repository.RankingRepository, now time.Time) {
	t.Helper()
	rankingJob := job.NewRankingJob(repo, testRankingConfig())
	if err := rankingJob.RecomputeAll(context.Background(), now); err != nil {
		t.Fatalf("recompute rankings: %v", err)
	}
}

func mustCreateImageEvent(t *testing.T, repo *repository.ImageRepository, event do.ImageEvent) {
	t.Helper()
	if event.Value == 0 {
		event.Value = 1
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = fixedRankingNow()
	}
	if err := repo.CreateImageEvents(context.Background(), []do.ImageEvent{event}); err != nil {
		t.Fatalf("create image event: %v", err)
	}
}

func assertSingleRankingViewCount(t *testing.T, rankings do.RankingListResult, want int64) {
	t.Helper()
	if len(rankings.List) != 1 {
		t.Fatalf("rankings = %#v, want one row", rankings.List)
	}
	if rankings.List[0].ViewCount != want {
		t.Fatalf("view count = %d, want %d", rankings.List[0].ViewCount, want)
	}
}

func testRankingConfig() conf.RankingConfig {
	return conf.RankingConfig{
		WeightRating:      1,
		WeightFavorite:    1,
		WeightView:        1,
		BayesianC:         10,
		RecomputeInterval: 10 * time.Minute,
	}
}

func fixedRankingNow() time.Time {
	return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
}
