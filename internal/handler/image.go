package handler

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/service"
	apperrors "github.com/yachiyo/acgwarehouse/pkg/errors"
)

// ImageHandler 处理图片查询、搜索和生命周期请求。
type ImageHandler struct {
	imageService *service.ImageService
}

// NewImageHandler 创建图片处理器。
func NewImageHandler(imageService *service.ImageService) *ImageHandler {
	return &ImageHandler{imageService: imageService}
}

// List 处理图片列表查询请求。
func (h *ImageHandler) List(c context.Context, ctx *app.RequestContext) {
	page := ParsePageQuery(ctx)
	result, err := h.imageService.List(c, service.ImageQuery{
		Filename: string(ctx.Query("filename")),
		Tag:      string(ctx.Query("tag")),
		Page:     page.Page,
		Size:     page.Size,
		Sort:     page.Sort,
		Order:    page.Order,
	})
	if err != nil {
		writeImageError(c, ctx, err)
		return
	}
	Success(ctx, NewListResponse(result.Total, PageQuery{Page: result.Page, Size: result.Size}, result.List))
}

// Detail 处理图片详情查询请求。
func (h *ImageHandler) Detail(c context.Context, ctx *app.RequestContext) {
	id, ok := parseIDParam(c, ctx)
	if !ok {
		return
	}
	detail, err := h.imageService.Detail(c, id, currentUserID(ctx))
	if err != nil {
		writeImageError(c, ctx, err)
		return
	}
	Success(ctx, detail)
}

// Search 处理图片全文搜索请求。
func (h *ImageHandler) Search(c context.Context, ctx *app.RequestContext) {
	page := ParsePageQuery(ctx)
	result, err := h.imageService.Search(c, service.SearchQuery{
		Text: string(ctx.Query("q")),
		Page: page.Page,
		Size: page.Size,
	})
	if err != nil {
		writeImageError(c, ctx, err)
		return
	}
	Success(ctx, NewListResponse(result.Total, PageQuery{Page: result.Page, Size: result.Size}, result.List))
}

// SoftDelete 处理管理员软删除图片请求。
func (h *ImageHandler) SoftDelete(c context.Context, ctx *app.RequestContext) {
	id, ok := parseIDParam(c, ctx)
	if !ok {
		return
	}
	if err := h.imageService.SoftDelete(c, id); err != nil {
		writeImageError(c, ctx, err)
		return
	}
	Success(ctx, map[string]bool{"deleted": true})
}

// Restore 处理管理员恢复图片请求。
func (h *ImageHandler) Restore(c context.Context, ctx *app.RequestContext) {
	id, ok := parseIDParam(c, ctx)
	if !ok {
		return
	}
	if _, err := h.imageService.Restore(c, id); err != nil {
		writeImageError(c, ctx, err)
		return
	}
	Success(ctx, map[string]bool{"restored": true})
}

// parseIDParam 解析路径中的图片 ID。
func parseIDParam(c context.Context, ctx *app.RequestContext) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id < 1 {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "图片 ID 非法", err)
		return 0, false
	}
	return id, true
}

// currentUserID 读取可选登录用户 ID。
func currentUserID(ctx *app.RequestContext) int64 {
	value, ok := ctx.Get("user_id")
	if !ok {
		return 0
	}
	id, ok := value.(int64)
	if !ok {
		return 0
	}
	return id
}

// writeImageError 将图片业务错误映射为统一响应。
func writeImageError(c context.Context, ctx *app.RequestContext, err error) {
	if stderrors.Is(err, service.ErrImageNotFound) {
		Fail(c, ctx, consts.StatusNotFound, apperrors.CodeNotFound, "图片不存在", err)
		return
	}
	Fail(c, ctx, consts.StatusInternalServerError, apperrors.CodeInternal, "系统异常", err)
}
