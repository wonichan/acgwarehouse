package job

import (
	"context"
	"sync"
	"time"

	pkgerrors "github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

const defaultRankingTopN = 100

// RankingRepository 定义热榜任务依赖的仓储能力。
type RankingRepository interface {
	AggregateMetrics(ctx context.Context, period do.RankingPeriod, from time.Time) ([]do.RankingMetrics, error)
	ReplacePeriod(ctx context.Context, period do.RankingPeriod, scores []do.RankingScore) error
}

// RankingJob 定时预计算热榜缓存。
type RankingJob struct {
	repo     RankingRepository
	cfg      conf.RankingConfig
	topN     int
	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

// NewRankingJob 创建热榜预计算任务。
func NewRankingJob(repo RankingRepository, cfg conf.RankingConfig) *RankingJob {
	return &RankingJob{
		repo: repo,
		cfg:  cfg,
		topN: defaultRankingTopN,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
}

// Start 启动热榜定时预计算循环。
func (j *RankingJob) Start(ctx context.Context) {
	if j == nil {
		return
	}
	go j.run(ctx)
}

// Stop 停止热榜定时预计算循环。
func (j *RankingJob) Stop(ctx context.Context) error {
	if j == nil {
		return nil
	}
	j.stopOnce.Do(func() {
		close(j.stop)
	})
	select {
	case <-j.done:
		return nil
	case <-ctx.Done():
		return pkgerrors.WithMessage(ctx.Err(), "stop ranking job")
	}
}

// RecomputeAll 重新计算全部热榜周期。
func (j *RankingJob) RecomputeAll(ctx context.Context, now time.Time) error {
	for _, period := range do.RankingPeriods() {
		if err := j.Recompute(ctx, period, now); err != nil {
			return pkgerrors.WithMessage(err, "recompute ranking period")
		}
	}
	return nil
}

// Recompute 重新计算指定热榜周期。
func (j *RankingJob) Recompute(ctx context.Context, period do.RankingPeriod, now time.Time) error {
	if !period.IsValid() {
		return pkgerrors.New("job: invalid ranking period")
	}
	metrics, err := j.repo.AggregateMetrics(ctx, period, now.UTC().Add(-period.Window()))
	if err != nil {
		return pkgerrors.WithMessage(err, "aggregate ranking metrics")
	}
	scores := j.rankScores(period, metrics, now.UTC())
	if err := j.repo.ReplacePeriod(ctx, period, scores); err != nil {
		return pkgerrors.WithMessage(err, "replace ranking period")
	}
	return nil
}

// run 按固定间隔刷新热榜缓存。
func (j *RankingJob) run(ctx context.Context) {
	defer close(j.done)
	if err := j.RecomputeAll(ctx, time.Now().UTC()); err != nil {
		logger.Warn(ctx, "recompute rankings failed", zap.Error(err))
	}
	ticker := time.NewTicker(j.cfg.RecomputeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := j.RecomputeAll(ctx, time.Now().UTC()); err != nil {
				logger.Warn(ctx, "recompute rankings failed", zap.Error(err))
			}
		case <-j.stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

// rankScores 根据热度公式生成排名分值。
func (j *RankingJob) rankScores(period do.RankingPeriod, metrics []do.RankingMetrics, computedAt time.Time) []do.RankingScore {
	scores := make([]do.RankingScore, 0, len(metrics))
	for _, metric := range metrics {
		bayesianScore := bayesianScore(metric, j.cfg.BayesianC)
		scores = append(scores, do.RankingScore{
			Period:        period,
			ImageID:       metric.ImageID,
			Score:         heatScore(metric, bayesianScore, j.cfg),
			BayesianScore: bayesianScore,
			RatingCount:   metric.RatingCount,
			FavoriteCount: metric.FavoriteCount,
			ViewCount:     metric.ViewCount,
			ComputedAt:    computedAt,
		})
	}
	sortRankingScores(scores)
	return trimRankingScores(scores, j.topN)
}
