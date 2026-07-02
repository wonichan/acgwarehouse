package handler

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
	apperrors "github.com/yachiyo/acgwarehouse/pkg/errors"
)

// RankingHandler 处理热榜查询请求。
type RankingHandler struct {
	rankingService *service.RankingService
}

// NewRankingHandler 创建热榜处理器。
func NewRankingHandler(rankingService *service.RankingService) *RankingHandler {
	return &RankingHandler{rankingService: rankingService}
}

// List 处理热榜缓存查询请求。
func (h *RankingHandler) List(c context.Context, ctx *app.RequestContext) {
	page := parseRankingPageQuery(ctx)
	result, err := h.rankingService.List(c, service.RankingQuery{
		Period: string(ctx.Query("period")),
		Page:   page.Page,
		Size:   page.Size,
	})
	if err != nil {
		writeRankingError(c, ctx, err)
		return
	}
	Success(ctx, NewListResponse(result.Total, PageQuery{Page: result.Page, Size: result.Size}, result.List))
}

// parseRankingPageQuery 从热榜请求中解析按周期约定的分页参数。
func parseRankingPageQuery(ctx *app.RequestContext) PageQuery {
	page := ParsePageQuery(ctx)
	if strings.TrimSpace(string(ctx.Query("size"))) != "" {
		return page
	}
	page.Size = do.RankingPeriod(strings.ToLower(strings.TrimSpace(string(ctx.Query("period"))))).DefaultSize()
	return page
}

// writeRankingError 将热榜业务错误映射为统一响应。
func writeRankingError(c context.Context, ctx *app.RequestContext, err error) {
	if stderrors.Is(err, service.ErrInvalidRankingPeriod) {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "榜单周期参数错误", err)
		return
	}
	Fail(c, ctx, consts.StatusInternalServerError, apperrors.CodeInternal, "系统异常", err)
}
