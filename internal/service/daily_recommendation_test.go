package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
)

type memoryDailyRecommendationRepository struct {
	dates  []string
	limits []int
	result do.DailyRecommendationList
}

func (r *memoryDailyRecommendationRepository) GetOrCreateToday(
	_ context.Context,
	date string,
	limit int,
	_ time.Time,
) (do.DailyRecommendationList, error) {
	r.dates = append(r.dates, date)
	r.limits = append(r.limits, limit)
	r.result.Date = date
	return r.result, nil
}

func Test_DailyRecommendationService_Today_uses_shanghai_natural_date_when_utc_crosses_boundary(t *testing.T) {
	// Given
	repo := &memoryDailyRecommendationRepository{result: do.DailyRecommendationList{Images: []do.Image{}}}
	svc := service.NewDailyRecommendationService(repo, "https://cdn.example.com")
	beforeBeijingMidnight := time.Date(2026, 6, 29, 15, 30, 0, 0, time.UTC)
	afterBeijingMidnight := time.Date(2026, 6, 29, 16, 30, 0, 0, time.UTC)

	// When
	_, firstErr := svc.Today(context.Background(), beforeBeijingMidnight)
	_, secondErr := svc.Today(context.Background(), afterBeijingMidnight)

	// Then
	if firstErr != nil || secondErr != nil {
		t.Fatalf("today: first=%v second=%v", firstErr, secondErr)
	}
	if len(repo.dates) != 2 || repo.dates[0] != "2026-06-29" || repo.dates[1] != "2026-06-30" {
		t.Fatalf("dates = %#v, want Shanghai dates around midnight", repo.dates)
	}
	if repo.limits[0] != 10 || repo.limits[1] != 10 {
		t.Fatalf("limits = %#v, want default limit 10", repo.limits)
	}
}

func Test_DailyRecommendationService_Today_maps_images_to_response_with_cdn_url(t *testing.T) {
	// Given
	repo := &memoryDailyRecommendationRepository{result: do.DailyRecommendationList{Images: []do.Image{{
		ID:            7,
		COSKey:        "thumbnails/miku.png",
		Filename:      "miku.png",
		Size:          123,
		Width:         640,
		Height:        480,
		AvgScore:      88,
		RatingCount:   2,
		FavoriteCount: 3,
		ViewCount:     4,
		LastModified:  fixedServiceDailyTime(),
		CreatedAt:     fixedServiceDailyTime(),
	}}}}
	svc := service.NewDailyRecommendationService(repo, "https://cdn.example.com/")

	// When
	result, err := svc.Today(context.Background(), fixedServiceDailyTime())

	// Then
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	if result.Date != "2026-06-30" || result.Timezone != "Asia/Shanghai" || result.Total != 1 {
		t.Fatalf("result metadata = %#v, want date/timezone/total", result)
	}
	if len(result.List) != 1 || result.List[0].URL != "https://cdn.example.com/thumbnails/miku.png" {
		t.Fatalf("list = %#v, want CDN image URL", result.List)
	}
}

func fixedServiceDailyTime() time.Time {
	return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
}
