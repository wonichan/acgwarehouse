package job_test

import (
	"context"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/job"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
)

type recordingRankingRepository struct {
	metrics  []do.RankingMetrics
	replaced map[do.RankingPeriod][]do.RankingScore
}

func (r *recordingRankingRepository) AggregateMetrics(
	_ context.Context,
	_ do.RankingPeriod,
	_ time.Time,
) ([]do.RankingMetrics, error) {
	return r.metrics, nil
}

func (r *recordingRankingRepository) ReplacePeriod(
	_ context.Context,
	period do.RankingPeriod,
	scores []do.RankingScore,
) error {
	r.replaced[period] = append([]do.RankingScore(nil), scores...)
	return nil
}

func Test_RankingJob_RecomputeAll_applies_period_image_count_limits(t *testing.T) {
	// Given
	repo := &recordingRankingRepository{
		metrics:  rankingMetrics(120),
		replaced: make(map[do.RankingPeriod][]do.RankingScore),
	}
	rankingJob := job.NewRankingJob(repo, conf.RankingConfig{WeightRating: 1})

	// When
	if err := rankingJob.RecomputeAll(context.Background(), time.Date(2026, 7, 2, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("recompute all rankings: %v", err)
	}

	// Then
	wantLimits := map[do.RankingPeriod]int{
		do.RankingPeriodDay:   20,
		do.RankingPeriodWeek:  50,
		do.RankingPeriodMonth: 100,
	}
	for period, wantLimit := range wantLimits {
		if gotLimit := len(repo.replaced[period]); gotLimit != wantLimit {
			t.Fatalf("%s ranking scores = %d, want %d", period, gotLimit, wantLimit)
		}
	}
}

func rankingMetrics(count int) []do.RankingMetrics {
	metrics := make([]do.RankingMetrics, 0, count)
	for imageID := 1; imageID <= count; imageID++ {
		metrics = append(metrics, do.RankingMetrics{
			ImageID:     int64(imageID),
			SumScore:    float64(imageID),
			RatingCount: 1,
		})
	}
	return metrics
}
