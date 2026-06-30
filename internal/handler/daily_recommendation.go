package handler

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/service"
	apperrors "github.com/yachiyo/acgwarehouse/pkg/errors"
)

// DailyRecommendationHandler 处理每日推荐查询请求。
type DailyRecommendationHandler struct {
	dailyService *service.DailyRecommendationService
}

// NewDailyRecommendationHandler 创建每日推荐处理器。
func NewDailyRecommendationHandler(dailyService *service.DailyRecommendationService) *DailyRecommendationHandler {
	return &DailyRecommendationHandler{dailyService: dailyService}
}

// Today 处理今日每日推荐查询请求。
func (h *DailyRecommendationHandler) Today(c context.Context, ctx *app.RequestContext) {
	result, err := h.dailyService.Today(c, time.Now().UTC())
	if err != nil {
		Fail(c, ctx, consts.StatusInternalServerError, apperrors.CodeInternal, "系统异常", err)
		return
	}
	Success(ctx, result)
}
