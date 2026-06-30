package service

import (
	"context"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/dto"
)

const (
	dailyRecommendationLimit    = 10
	dailyRecommendationTimezone = "Asia/Shanghai"
	dailyRecommendationDate     = "2006-01-02"
)

// DailyRecommendationRepository 定义每日推荐服务依赖的仓储能力。
type DailyRecommendationRepository interface {
	GetOrCreateToday(ctx context.Context, date string, limit int, nowUTC time.Time) (do.DailyRecommendationList, error)
}

// DailyRecommendationService 提供全站每日推荐读取能力。
type DailyRecommendationService struct {
	repo    DailyRecommendationRepository
	cosBase string
}

// NewDailyRecommendationService 创建每日推荐服务。
func NewDailyRecommendationService(repo DailyRecommendationRepository, cosBase string) *DailyRecommendationService {
	return &DailyRecommendationService{repo: repo, cosBase: strings.TrimRight(cosBase, "/")}
}

// Today 返回北京时间自然日的全站每日推荐。
func (s *DailyRecommendationService) Today(
	ctx context.Context,
	now time.Time,
) (dto.DailyRecommendationResponse, error) {
	location, err := time.LoadLocation(dailyRecommendationTimezone)
	if err != nil {
		return dto.DailyRecommendationResponse{}, pkgerrors.WithMessage(err, "load daily recommendation timezone")
	}
	date := now.In(location).Format(dailyRecommendationDate)
	result, err := s.repo.GetOrCreateToday(ctx, date, dailyRecommendationLimit, now.UTC())
	if err != nil {
		return dto.DailyRecommendationResponse{}, pkgerrors.WithMessage(err, "get today daily recommendations")
	}
	return dto.DailyRecommendationResponse{
		Date:     result.Date,
		Timezone: dailyRecommendationTimezone,
		Total:    len(result.Images),
		List:     s.imagesToResponse(result.Images),
	}, nil
}

func (s *DailyRecommendationService) imagesToResponse(images []do.Image) []dto.ImageResponse {
	result := make([]dto.ImageResponse, 0, len(images))
	for _, image := range images {
		result = append(result, s.imageToResponse(image))
	}
	return result
}

func (s *DailyRecommendationService) imageToResponse(image do.Image) dto.ImageResponse {
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

func (s *DailyRecommendationService) imageURL(cosKey string) string {
	if s.cosBase == "" {
		return cosKey
	}
	return s.cosBase + "/" + strings.TrimLeft(cosKey, "/")
}
