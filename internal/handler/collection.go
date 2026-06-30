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

// CollectionHandler 处理收藏夹请求。
type CollectionHandler struct {
	collectionService *service.CollectionService
}

// NewCollectionHandler 创建收藏夹处理器。
func NewCollectionHandler(collectionService *service.CollectionService) *CollectionHandler {
	return &CollectionHandler{collectionService: collectionService}
}

// ListMine 处理当前用户收藏夹列表请求。
func (h *CollectionHandler) ListMine(c context.Context, ctx *app.RequestContext) {
	collections, err := h.collectionService.ListByOwner(c, currentUserID(ctx))
	if err != nil {
		writeCollectionError(c, ctx, err)
		return
	}
	Success(ctx, collectionsToResponse(collections))
}

// Create 处理收藏夹创建请求。
func (h *CollectionHandler) Create(c context.Context, ctx *app.RequestContext) {
	input, ok := bindCollectionCreate(c, ctx)
	if !ok {
		return
	}
	collection, err := h.collectionService.Create(c, do.Collection{
		UserID:     currentUserID(ctx),
		Name:       input.Name,
		Visibility: do.CollectionVisibility(input.Visibility),
	})
	if err != nil {
		writeCollectionError(c, ctx, err)
		return
	}
	Success(ctx, collectionToResponse(collection))
}

// Update 处理 owner 收藏夹更新请求。
func (h *CollectionHandler) Update(c context.Context, ctx *app.RequestContext) {
	id, ok := parseCollectionIDParam(c, ctx)
	if !ok {
		return
	}
	input, ok := bindCollectionUpdate(c, ctx)
	if !ok {
		return
	}
	coverImageID := int64(0)
	coverImageIDSet := input.CoverImageID != nil
	if coverImageIDSet {
		coverImageID = *input.CoverImageID
	}
	collection, err := h.collectionService.Update(c, do.Collection{
		ID:              id,
		UserID:          currentUserID(ctx),
		Name:            input.Name,
		Visibility:      do.CollectionVisibility(input.Visibility),
		CoverImageID:    coverImageID,
		CoverImageIDSet: coverImageIDSet,
	})
	if err != nil {
		writeCollectionError(c, ctx, err)
		return
	}
	Success(ctx, collectionToResponse(collection))
}

// Delete 处理 owner 收藏夹删除请求。
func (h *CollectionHandler) Delete(c context.Context, ctx *app.RequestContext) {
	id, ok := parseCollectionIDParam(c, ctx)
	if !ok {
		return
	}
	if err := h.collectionService.Delete(c, id, currentUserID(ctx)); err != nil {
		writeCollectionError(c, ctx, err)
		return
	}
	Success(ctx, map[string]bool{"deleted": true})
}

// Detail 处理收藏夹详情请求，公开收藏夹允许匿名访问。
func (h *CollectionHandler) Detail(c context.Context, ctx *app.RequestContext) {
	id, ok := parseCollectionIDParam(c, ctx)
	if !ok {
		return
	}
	collection, err := h.collectionService.FindVisible(c, id, currentUserID(ctx))
	if err != nil {
		writeCollectionError(c, ctx, err)
		return
	}
	Success(ctx, collectionToResponse(collection))
}

// AddItem 处理 owner 新增收藏夹图片请求。
func (h *CollectionHandler) AddItem(c context.Context, ctx *app.RequestContext) {
	id, ok := parseCollectionIDParam(c, ctx)
	if !ok {
		return
	}
	input, ok := bindCollectionItem(c, ctx)
	if !ok {
		return
	}
	item, err := h.collectionService.AddItem(c, id, currentUserID(ctx), input.ImageID)
	if err != nil {
		writeCollectionError(c, ctx, err)
		return
	}
	Success(ctx, collectionItemToResponse(item))
}

// RemoveItem 处理 owner 移除收藏夹图片请求。
func (h *CollectionHandler) RemoveItem(c context.Context, ctx *app.RequestContext) {
	id, ok := parseCollectionIDParam(c, ctx)
	if !ok {
		return
	}
	imageID, ok := parseImageIDParam(c, ctx)
	if !ok {
		return
	}
	if err := h.collectionService.RemoveItem(c, id, currentUserID(ctx), imageID); err != nil {
		writeCollectionError(c, ctx, err)
		return
	}
	Success(ctx, map[string]bool{"removed": true})
}

// bindCollectionCreate 绑定收藏夹创建请求体。
func bindCollectionCreate(c context.Context, ctx *app.RequestContext) (dto.CollectionCreateRequest, bool) {
	var input dto.CollectionCreateRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "收藏夹参数错误", err)
		return dto.CollectionCreateRequest{}, false
	}
	return input, true
}

// bindCollectionUpdate 绑定收藏夹更新请求体。
func bindCollectionUpdate(c context.Context, ctx *app.RequestContext) (dto.CollectionUpdateRequest, bool) {
	var input dto.CollectionUpdateRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "收藏夹参数错误", err)
		return dto.CollectionUpdateRequest{}, false
	}
	return input, true
}

// bindCollectionItem 绑定收藏夹图片条目请求体。
func bindCollectionItem(c context.Context, ctx *app.RequestContext) (dto.CollectionItemRequest, bool) {
	var input dto.CollectionItemRequest
	if err := ctx.BindAndValidate(&input); err != nil {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "收藏图片参数错误", err)
		return dto.CollectionItemRequest{}, false
	}
	return input, true
}

// parseCollectionIDParam 解析路径中的收藏夹 ID。
func parseCollectionIDParam(c context.Context, ctx *app.RequestContext) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id < 1 {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "收藏夹 ID 非法", err)
		return 0, false
	}
	return id, true
}

// parseImageIDParam 解析路径中的图片 ID。
func parseImageIDParam(c context.Context, ctx *app.RequestContext) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("imageId"), 10, 64)
	if err != nil || id < 1 {
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "图片 ID 非法", err)
		return 0, false
	}
	return id, true
}

// collectionsToResponse 将收藏夹列表转换为 HTTP 响应 DTO 列表。
func collectionsToResponse(collections []do.Collection) []dto.CollectionResponse {
	result := make([]dto.CollectionResponse, 0, len(collections))
	for _, collection := range collections {
		result = append(result, collectionToResponse(collection))
	}
	return result
}

// collectionToResponse 将收藏夹领域对象转换为 HTTP 响应 DTO。
func collectionToResponse(collection do.Collection) dto.CollectionResponse {
	return dto.CollectionResponse{
		ID:            collection.ID,
		UserID:        collection.UserID,
		Name:          collection.Name,
		Visibility:    string(collection.Visibility),
		CreatedAt:     FormatTime(collection.CreatedAt),
		CoverImageID:  collection.CoverImageID,
		CoverImageURL: collection.CoverImageURL,
		Items:         collectionItemsToResponse(collection.Items),
	}
}

// collectionItemsToResponse 将收藏夹条目列表转换为 HTTP 响应 DTO 列表。
func collectionItemsToResponse(items []do.CollectionItem) []dto.CollectionItemResponse {
	result := make([]dto.CollectionItemResponse, 0, len(items))
	for _, item := range items {
		result = append(result, collectionItemToResponse(item))
	}
	return result
}

// collectionItemToResponse 将收藏夹条目领域对象转换为 HTTP 响应 DTO。
func collectionItemToResponse(item do.CollectionItem) dto.CollectionItemResponse {
	resp := dto.CollectionItemResponse{
		CollectionID: item.CollectionID,
		ImageID:      item.ImageID,
		CreatedAt:    FormatTime(item.CreatedAt),
	}
	if item.Image.ID > 0 {
		resp.Image = imageToCollectionItemResponse(item.Image)
	}
	return resp
}

// imageToCollectionItemResponse 将收藏条目图片领域对象转换为 HTTP 响应 DTO。
func imageToCollectionItemResponse(image do.Image) *dto.ImageResponse {
	return &dto.ImageResponse{
		ID:            image.ID,
		COSKey:        image.COSKey,
		Filename:      image.Filename,
		URL:           image.URL,
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

// writeCollectionError 将收藏业务错误映射为统一响应。
func writeCollectionError(c context.Context, ctx *app.RequestContext, err error) {
	switch {
	case stderrors.Is(err, service.ErrInvalidCollectionInput):
		Fail(c, ctx, consts.StatusBadRequest, apperrors.CodeInvalidParam, "收藏夹参数错误", err)
	case stderrors.Is(err, service.ErrForbidden):
		Fail(c, ctx, consts.StatusForbidden, apperrors.CodeForbidden, "无权操作", err)
	case stderrors.Is(err, service.ErrCollectionNotFound):
		Fail(c, ctx, consts.StatusNotFound, apperrors.CodeNotFound, "收藏夹不存在", err)
	case stderrors.Is(err, service.ErrImageNotFound):
		Fail(c, ctx, consts.StatusNotFound, apperrors.CodeNotFound, "图片不存在", err)
	default:
		Fail(c, ctx, consts.StatusInternalServerError, apperrors.CodeInternal, "系统异常", err)
	}
}
