package service

import (
	"context"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/dto"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

var (
	// ErrInvalidRankingPeriod 表示热榜周期参数非法。
	ErrInvalidRankingPeriod = pkgerrors.New("service: invalid ranking period")
)

// RepositoryRankingQuery 定义热榜仓储列表查询条件。
type RepositoryRankingQuery = repository.RankingListQuery

// RankingQuery 定义热榜查询条件。
type RankingQuery struct {
	Period string
	Page   int
	Size   int
}

// RankingListResult 定义热榜列表业务结果。
type RankingListResult struct {
	Total int64
	Page  int
	Size  int
	List  []dto.RankingResponse
}

// RankingRepository 定义热榜服务依赖的仓储能力。
type RankingRepository interface {
	ListCached(ctx context.Context, period do.RankingPeriod, query RepositoryRankingQuery) (do.RankingListResult, error)
}

// RankingService 提供热榜缓存查询能力。
type RankingService struct {
	repo    RankingRepository
	cosBase string
}

// NewRankingService 创建热榜服务。
func NewRankingService(repo RankingRepository, cosBase string) *RankingService {
	return &RankingService{repo: repo, cosBase: strings.TrimRight(cosBase, "/")}
}

// List 查询指定周期的热榜缓存。
func (s *RankingService) List(ctx context.Context, query RankingQuery) (RankingListResult, error) {
	period, err := parseRankingPeriod(query.Period)
	if err != nil {
		return RankingListResult{}, err
	}
	result, err := s.repo.ListCached(ctx, period, RepositoryRankingQuery{Page: query.Page, Size: query.Size})
	if err != nil {
		return RankingListResult{}, pkgerrors.WithMessage(err, "list cached rankings")
	}
	return RankingListResult{Total: result.Total, Page: result.Page, Size: result.Size, List: s.rankingsToResponse(result.List)}, nil
}

// parseRankingPeriod 解析热榜周期，空值默认 day。
func parseRankingPeriod(raw string) (do.RankingPeriod, error) {
	period := do.RankingPeriod(strings.ToLower(strings.TrimSpace(raw)))
	if period == "" {
		return do.RankingPeriodDay, nil
	}
	if !period.IsValid() {
		return "", pkgerrors.WithMessage(ErrInvalidRankingPeriod, "parse ranking period")
	}
	return period, nil
}

// rankingsToResponse 将热榜领域对象列表转换为 HTTP DTO 列表。
func (s *RankingService) rankingsToResponse(rankings []do.Ranking) []dto.RankingResponse {
	result := make([]dto.RankingResponse, 0, len(rankings))
	for _, ranking := range rankings {
		result = append(result, s.rankingToResponse(ranking))
	}
	return result
}

// rankingToResponse 将热榜领域对象转换为 HTTP DTO。
func (s *RankingService) rankingToResponse(ranking do.Ranking) dto.RankingResponse {
	return dto.RankingResponse{
		Period:        string(ranking.Period),
		Rank:          ranking.Rank,
		Score:         ranking.Score,
		BayesianScore: ranking.BayesianScore,
		RatingCount:   ranking.RatingCount,
		FavoriteCount: ranking.FavoriteCount,
		ViewCount:     ranking.ViewCount,
		ComputedAt:    formatAPITime(ranking.ComputedAt),
		Image:         s.imageToResponse(ranking.Image),
	}
}

// imageToResponse 将热榜图片领域对象转换为公开响应 DTO。
func (s *RankingService) imageToResponse(image do.Image) dto.ImageResponse {
	return dto.ImageResponse{
		ID:            image.ID,
		COSKey:        image.COSKey,
		Filename:      image.Filename,
		URL:           s.imageURL(image.COSKey),
		Size:          image.Size,
		LastModified:  formatAPITime(image.LastModified),
		Width:         image.Width,
		Height:        image.Height,
		Category:      image.Category,
		AvgScore:      image.AvgScore,
		RatingCount:   image.RatingCount,
		FavoriteCount: image.FavoriteCount,
		ViewCount:     image.ViewCount,
		CreatedAt:     formatAPITime(image.CreatedAt),
	}
}

// imageURL 拼接热榜图片 COS 公开访问地址。
func (s *RankingService) imageURL(cosKey string) string {
	if s.cosBase == "" {
		return cosKey
	}
	return s.cosBase + "/" + strings.TrimLeft(cosKey, "/")
}
