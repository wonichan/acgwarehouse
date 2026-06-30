package router_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/handler"
	"github.com/yachiyo/acgwarehouse/internal/handler/router"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
)

type memoryRouterDailyRecommendationRepository struct {
	result do.DailyRecommendationList
	err    error
}

func (r *memoryRouterDailyRecommendationRepository) GetOrCreateToday(
	_ context.Context,
	date string,
	_ int,
	_ time.Time,
) (do.DailyRecommendationList, error) {
	if r.err != nil {
		return do.DailyRecommendationList{}, r.err
	}
	r.result.Date = date
	return r.result, nil
}

func Test_DailyRecommendationRoute_returns_today_recommendations_when_available(t *testing.T) {
	// Given
	engine := dailyRecommendationRouterTestEngine(t, do.DailyRecommendationList{Images: []do.Image{{
		ID:           7,
		COSKey:       "thumbnails/miku.png",
		Filename:     "miku.png",
		Size:         100,
		Width:        640,
		Height:       480,
		CreatedAt:    fixedRouterDailyTime(),
		LastModified: fixedRouterDailyTime(),
	}}}, nil)

	// When
	recorder := ut.PerformRequest(engine.Engine.Engine, consts.MethodGet, "/api/v1/daily-recommendations", nil)

	// Then
	if recorder.Code != consts.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	var response handler.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := response.Data.(map[string]interface{})
	if !ok || data["timezone"] != "Asia/Shanghai" || data["total"].(float64) != 1 {
		t.Fatalf("response = %#v, want daily recommendation shape", response)
	}
}

func Test_DailyRecommendationRoute_returns_internal_error_when_service_fails(t *testing.T) {
	// Given
	engine := dailyRecommendationRouterTestEngine(t, do.DailyRecommendationList{}, pkgerrors.New("boom"))

	// When
	recorder := ut.PerformRequest(engine.Engine.Engine, consts.MethodGet, "/api/v1/daily-recommendations", nil)

	// Then
	if recorder.Code != consts.StatusInternalServerError {
		t.Fatalf("status = %d body=%s, want 500", recorder.Code, recorder.Body.String())
	}
}

func dailyRecommendationRouterTestEngine(
	t *testing.T,
	result do.DailyRecommendationList,
	err error,
) *routerTestHarness {
	t.Helper()
	repo := &memoryRouterDailyRecommendationRepository{result: result, err: err}
	return routerTestEngineWithServices(t, router.Services{
		DailyRecommendation: service.NewDailyRecommendationService(repo, "https://cdn.example.com"),
	})
}

func fixedRouterDailyTime() time.Time {
	return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
}
