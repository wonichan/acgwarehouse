package router_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler"
	"github.com/yachiyo/acgwarehouse/internal/handler/router"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
	"github.com/yachiyo/acgwarehouse/internal/service"
)

type memoryRouterRankingRepository struct {
	result do.RankingListResult
}

func (r *memoryRouterRankingRepository) ListCached(
	_ context.Context,
	period do.RankingPeriod,
	query repository.RankingListQuery,
) (do.RankingListResult, error) {
	r.result.Page = query.Page
	r.result.Size = query.Size
	for index := range r.result.List {
		r.result.List[index].Period = period
	}
	return r.result, nil
}

func Test_RankingRoute_returns_bad_request_when_period_invalid(t *testing.T) {
	// Given
	engine := rankingRouterTestEngine(t, do.RankingListResult{})

	// When
	recorder := ut.PerformRequest(engine.Engine.Engine, consts.MethodGet, "/api/v1/rankings?period=year", nil)

	// Then
	if recorder.Code != consts.StatusBadRequest {
		t.Fatalf("status = %d body=%s, want 400", recorder.Code, recorder.Body.String())
	}
}

func Test_RankingRoute_reads_cached_rankings_when_period_valid(t *testing.T) {
	// Given
	engine := rankingRouterTestEngine(t, do.RankingListResult{
		Total: 1,
		List: []do.Ranking{{
			Rank:       1,
			Score:      51,
			ComputedAt: time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC),
			Image:      do.Image{ID: 7, COSKey: "thumbnails/miku.png", Filename: "miku.png"},
		}},
	})

	// When
	recorder := ut.PerformRequest(engine.Engine.Engine, consts.MethodGet, "/api/v1/rankings?period=week", nil)

	// Then
	if recorder.Code != consts.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	var response handler.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := response.Data.(map[string]interface{})
	if !ok || data["total"].(float64) != 1 {
		t.Fatalf("response = %#v, want cached ranking total", response)
	}
}

func rankingRouterTestEngine(t *testing.T, result do.RankingListResult) *routerTestHarness {
	t.Helper()
	rankingRepo := &memoryRouterRankingRepository{result: result}
	return routerTestEngineWithServices(t, router.Services{
		Ranking: service.NewRankingService(rankingRepo, "https://cdn.example.com"),
	})
}
