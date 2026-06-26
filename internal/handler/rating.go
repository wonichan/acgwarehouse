package handler

import (
	"context"
	stderrors "errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/dto"
	"github.com/yachiyo/acgwarehouse/internal/service"
	apperrors "github.com/yachiyo/acgwarehouse/pkg/errors"
)

// RatingHandler 处理图片评分请求。
type RatingHandler struct {
	ratingService *service.RatingService
}

// NewRatingHandler 创建评分处理器。
func NewRatingHandler(ratingService *service.RatingService) *RatingHandler {
	return &RatingHandler{ratingService: ratingService}
}

// Rate 处理用户对图片评分请求。
func (h *RatingHandler) Rate(c context.Context, ctx *app.RequestContext) {
	imageID, ok := parseIDParam(c, ctx)
	if !ok {
		return
	}
	input, ok := bindRating(c, ctx)
	if !ok {
		return
	}
	rating, err := h.ratingService.Rate(c, do.Rating{
		UserID:  currentUserID(ctx),
		ImageID: imageID,
		Score:   input.Score,
	})
	if err != nil {
		writeRatingError(c, ctx, err)
		return
	}
	Success(ctx, ratingToResponse(rating))
}

// bindRating 绑定图片评分请求体。
func bindRating(c context.Context, ctx *app.RequestContext) (dto.RatingRequest, bool) {
	var input dto.RatingRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "评分参数错误", err)
		return dto.RatingRequest{}, false
	}
	return input, true
}

// ratingToResponse 将评分领域对象转换为 HTTP 响应 DTO。
func ratingToResponse(rating do.Rating) dto.RatingResponse {
	return dto.RatingResponse{
		ImageID:   rating.ImageID,
		UserID:    rating.UserID,
		Score:     rating.Score,
		UpdatedAt: FormatTime(rating.UpdatedAt),
	}
}

// writeRatingError 将评分业务错误映射为统一响应。
func writeRatingError(c context.Context, ctx *app.RequestContext, err error) {
	switch {
	case stderrors.Is(err, service.ErrInvalidRatingInput):
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "评分参数错误", err)
	case stderrors.Is(err, service.ErrImageNotFound):
		Fail(c, ctx, consts.StatusNotFound, apperrors.CodeNotFound, "图片不存在", err)
	default:
		Fail(c, ctx, consts.StatusInternalServerError, apperrors.CodeInternal, "系统异常", err)
	}
}
