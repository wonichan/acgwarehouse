package handler

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/dto"
	"github.com/yachiyo/acgwarehouse/internal/service"
	apperrors "github.com/yachiyo/acgwarehouse/pkg/errors"
)

// TagHandler 处理标签管理请求。
type TagHandler struct {
	tagService *service.TagService
}

// NewTagHandler 创建标签处理器。
func NewTagHandler(tagService *service.TagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

// List 处理标签列表请求。
func (h *TagHandler) List(c context.Context, ctx *app.RequestContext) {
	tags, err := h.tagService.List(c)
	if err != nil {
		writeTagError(c, ctx, err)
		return
	}
	Success(ctx, tagsToResponse(tags))
}

// Create 处理标签创建请求。
func (h *TagHandler) Create(c context.Context, ctx *app.RequestContext) {
	input, ok := bindTagCreate(c, ctx)
	if !ok {
		return
	}
	tag, err := h.tagService.Create(c, do.Tag{Name: input.Name})
	if err != nil {
		writeTagError(c, ctx, err)
		return
	}
	Success(ctx, tagToResponse(tag))
}

// Update 处理管理员标签更新请求。
func (h *TagHandler) Update(c context.Context, ctx *app.RequestContext) {
	id, ok := parseTagIDParam(c, ctx)
	if !ok {
		return
	}
	input, ok := bindTagUpdate(c, ctx)
	if !ok {
		return
	}
	tag, err := h.tagService.Update(c, currentRole(ctx), do.Tag{ID: id, Name: input.Name})
	if err != nil {
		writeTagError(c, ctx, err)
		return
	}
	Success(ctx, tagToResponse(tag))
}

// Delete 处理管理员标签删除请求。
func (h *TagHandler) Delete(c context.Context, ctx *app.RequestContext) {
	id, ok := parseTagIDParam(c, ctx)
	if !ok {
		return
	}
	if err := h.tagService.Delete(c, currentRole(ctx), id); err != nil {
		writeTagError(c, ctx, err)
		return
	}
	Success(ctx, map[string]bool{"deleted": true})
}

// Suggest 处理标签建议请求。
func (h *TagHandler) Suggest(c context.Context, ctx *app.RequestContext) {
	limit := parsePositiveInt(string(ctx.Query("limit")), 20)
	tags, err := h.tagService.Suggest(c, string(ctx.Query("q")), limit)
	if err != nil {
		writeTagError(c, ctx, err)
		return
	}
	Success(ctx, tagsToResponse(tags))
}

// Assign 处理批量打标签请求。
func (h *TagHandler) Assign(c context.Context, ctx *app.RequestContext) {
	input, ok := bindTagBatch(c, ctx)
	if !ok {
		return
	}
	images, err := h.tagService.AssignToImages(c, input.ImageIDs, input.TagIDs)
	if err != nil {
		writeTagError(c, ctx, err)
		return
	}
	Success(ctx, imagesToTagResponse(images))
}

// Unassign 处理批量取消标签请求。
func (h *TagHandler) Unassign(c context.Context, ctx *app.RequestContext) {
	input, ok := bindTagBatch(c, ctx)
	if !ok {
		return
	}
	images, err := h.tagService.UnassignFromImages(c, input.ImageIDs, input.TagIDs)
	if err != nil {
		writeTagError(c, ctx, err)
		return
	}
	Success(ctx, imagesToTagResponse(images))
}

// bindTagCreate 绑定标签创建请求体。
func bindTagCreate(c context.Context, ctx *app.RequestContext) (dto.TagCreateRequest, bool) {
	var input dto.TagCreateRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "参数错误", err)
		return dto.TagCreateRequest{}, false
	}
	return input, true
}

// bindTagUpdate 绑定标签更新请求体。
func bindTagUpdate(c context.Context, ctx *app.RequestContext) (dto.TagUpdateRequest, bool) {
	var input dto.TagUpdateRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "参数错误", err)
		return dto.TagUpdateRequest{}, false
	}
	return input, true
}

// bindTagBatch 绑定批量图片标签请求体。
func bindTagBatch(c context.Context, ctx *app.RequestContext) (dto.ImageTagBatchRequest, bool) {
	var input dto.ImageTagBatchRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "参数错误", err)
		return dto.ImageTagBatchRequest{}, false
	}
	return input, true
}

// parseTagIDParam 解析路径中的标签 ID。
func parseTagIDParam(c context.Context, ctx *app.RequestContext) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id < 1 {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "标签 ID 非法", err)
		return 0, false
	}
	return id, true
}

// currentRole 读取当前登录用户角色。
func currentRole(ctx *app.RequestContext) do.UserRole {
	value, ok := ctx.Get("role")
	if !ok {
		return ""
	}
	role, ok := value.(string)
	if !ok {
		return ""
	}
	return do.UserRole(role)
}

// tagToResponse 将标签领域对象转换为 HTTP 响应 DTO。
func tagToResponse(tag do.Tag) dto.TagResponse {
	return dto.TagResponse{
		ID:         tag.ID,
		Name:       tag.Name,
		UsageCount: tag.UsageCount,
		CreatedAt:  FormatTime(tag.CreatedAt),
		UpdatedAt:  FormatTime(tag.UpdatedAt),
	}
}

// tagsToResponse 将标签领域对象列表转换为 HTTP 响应 DTO 列表。
func tagsToResponse(tags []do.Tag) []dto.TagResponse {
	result := make([]dto.TagResponse, 0, len(tags))
	for _, tag := range tags {
		result = append(result, tagToResponse(tag))
	}
	return result
}

// imagesToTagResponse 将受影响图片转换为打标响应 DTO 列表。
func imagesToTagResponse(images []do.Image) []dto.ImageTagResponse {
	result := make([]dto.ImageTagResponse, 0, len(images))
	for _, image := range images {
		result = append(result, dto.ImageTagResponse{
			Image: imageToTagImageResponse(image),
			Tags:  image.Tags,
		})
	}
	return result
}

// imageToTagImageResponse 将图片领域对象转换为打标响应中的图片 DTO。
func imageToTagImageResponse(image do.Image) dto.ImageResponse {
	return dto.ImageResponse{
		ID:            image.ID,
		COSKey:        image.COSKey,
		Filename:      image.Filename,
		Size:          image.Size,
		LastModified:  FormatTime(image.LastModified),
		Width:         image.Width,
		Height:        image.Height,
		Category:      image.Category,
		AvgScore:      image.AvgScore,
		RatingCount:   image.RatingCount,
		FavoriteCount: image.FavoriteCount,
		ViewCount:     image.ViewCount,
		CreatedAt:     FormatTime(image.CreatedAt),
	}
}

// writeTagError 将标签业务错误映射为统一响应。
func writeTagError(c context.Context, ctx *app.RequestContext, err error) {
	switch {
	case stderrors.Is(err, service.ErrInvalidTagInput):
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "标签参数错误", err)
	case stderrors.Is(err, service.ErrForbidden):
		Fail(c, ctx, consts.StatusForbidden, apperrors.CodeForbidden, "无权操作", err)
	case stderrors.Is(err, service.ErrTagNotFound):
		Fail(c, ctx, consts.StatusNotFound, apperrors.CodeNotFound, "标签不存在", err)
	default:
		Fail(c, ctx, consts.StatusInternalServerError, apperrors.CodeInternal, "系统异常", err)
	}
}
