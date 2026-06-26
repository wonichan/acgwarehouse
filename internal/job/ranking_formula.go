package job

import (
	"math"
	"sort"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
)

const defaultGlobalMeanScore = 50

// bayesianScore 计算带先验票数的贝叶斯评分。
func bayesianScore(metric do.RankingMetrics, bayesianC float64) float64 {
	if bayesianC < 0 {
		bayesianC = 0
	}
	denominator := bayesianC + float64(metric.RatingCount)
	if denominator == 0 {
		return 0
	}
	return (bayesianC*defaultGlobalMeanScore + metric.SumScore) / denominator
}

// heatScore 计算热榜总热度分。
func heatScore(metric do.RankingMetrics, bayesian float64, cfg conf.RankingConfig) float64 {
	return cfg.WeightRating*bayesian +
		cfg.WeightFavorite*math.Log1p(float64(metric.FavoriteCount)) +
		cfg.WeightView*math.Log1p(float64(metric.ViewCount))
}

// sortRankingScores 按热度倒序、图片 ID 升序稳定排序并写入名次。
func sortRankingScores(scores []do.RankingScore) {
	sort.SliceStable(scores, func(left int, right int) bool {
		if scores[left].Score == scores[right].Score {
			return scores[left].ImageID < scores[right].ImageID
		}
		return scores[left].Score > scores[right].Score
	})
	for index := range scores {
		scores[index].Rank = index + 1
	}
}

// trimRankingScores 截断热榜缓存大小。
func trimRankingScores(scores []do.RankingScore, topN int) []do.RankingScore {
	if topN < 1 || len(scores) <= topN {
		return scores
	}
	return scores[:topN]
}
