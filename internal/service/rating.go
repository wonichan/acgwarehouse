package service

import (
	"context"

	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
)

var (
	// ErrInvalidRatingInput 表示评分输入非法。
	ErrInvalidRatingInput = pkgerrors.New("service: invalid rating input")
)

// RatingRepository 定义评分服务依赖的仓储能力。
type RatingRepository interface {
	Upsert(ctx context.Context, rating do.Rating) (do.Rating, error)
}

// RatingService 提供图片评分能力。
type RatingService struct {
	repo RatingRepository
}

// NewRatingService 创建评分服务。
func NewRatingService(repo RatingRepository) *RatingService {
	return &RatingService{repo: repo}
}

// Rate 校验并写入用户对图片的评分。
func (s *RatingService) Rate(ctx context.Context, rating do.Rating) (do.Rating, error) {
	if err := validateRating(rating); err != nil {
		return do.Rating{}, err
	}
	stored, err := s.repo.Upsert(ctx, rating)
	if err != nil {
		return do.Rating{}, pkgerrors.WithMessage(err, "upsert rating")
	}
	return stored, nil
}

// validateRating 校验评分业务输入。
func validateRating(rating do.Rating) error {
	if rating.UserID < 1 {
		return pkgerrors.WithMessage(ErrInvalidRatingInput, "validate rating user")
	}
	if rating.ImageID < 1 {
		return pkgerrors.WithMessage(ErrInvalidRatingInput, "validate rating image")
	}
	if rating.Score < 0 || rating.Score > 100 {
		return pkgerrors.WithMessage(ErrInvalidRatingInput, "validate rating score")
	}
	return nil
}
