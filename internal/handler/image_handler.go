package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

type ImageHandler struct {
	imageRepo      repository.ImageRepository
	tagRepo        repository.TagRepository
	imageTagRepo   repository.ImageTagRepository
	collectionRepo repository.CollectionRepository
	fileActions    imageFileActionExecutor
	searchSvc      imageSearchWindowProvider
	adminSvc       imageScanTrigger
}

type imageScanTrigger interface {
	TriggerScan(ctx context.Context) (int64, error)
}

type imageSearchWindowProvider interface {
	ViewerWindow(ctx context.Context, opts service.SearchOptions, selectedIndex, limit int) (*service.ViewerWindowResult, error)
}

type viewerWindowRequest struct {
	Source          string               `json:"source"`
	SelectedIndex   int                  `json:"selected_index"`
	SelectedImageID int64                `json:"selected_image_id"`
	Limit           int                  `json:"limit"`
	Snapshot        viewerWindowSnapshot `json:"snapshot"`
}

type viewerWindowSnapshot struct {
	SortBy         string  `json:"sort_by"`
	SortDir        string  `json:"sort_dir"`
	TagIDs         []int64 `json:"tag_ids"`
	HasTags        *bool   `json:"has_tags"`
	HasPendingTags *bool   `json:"has_pending_tags"`
	Query          string  `json:"q"`
	SortOrder      string  `json:"sort_order"`
}

func NewImageHandler(imageRepo repository.ImageRepository, tagRepo repository.TagRepository, imageTagRepo repository.ImageTagRepository, depsOpt ...any) *ImageHandler {
	var adminSvc imageScanTrigger
	var searchSvc imageSearchWindowProvider
	var collectionRepo repository.CollectionRepository
	var fileActions imageFileActionExecutor
	for _, dep := range depsOpt {
		switch v := dep.(type) {
		case imageScanTrigger:
			adminSvc = v
		case imageSearchWindowProvider:
			searchSvc = v
		case repository.CollectionRepository:
			collectionRepo = v
		case imageFileActionExecutor:
			fileActions = v
		}
	}
	if fileActions == nil {
		fileActions = newDefaultImageFileActionExecutor()
	}

	return &ImageHandler{
		imageRepo:      imageRepo,
		tagRepo:        tagRepo,
		imageTagRepo:   imageTagRepo,
		collectionRepo: collectionRepo,
		fileActions:    fileActions,
		searchSvc:      searchSvc,
		adminSvc:       adminSvc,
	}
}

// ListImages handles GET /api/v1/images with optional tag_ids filtering.
// When tag_ids query param is provided, returns images that have ALL specified tags (AND semantics).
// When has_tags=false is provided, returns images that have NO tags.
// When no filter is provided, returns all images with pagination.
// has_tags=false is mutually exclusive with tag_ids parameter.
func (h *ImageHandler) ListImages(c *gin.Context) {
	ctx := c.Request.Context()
	limit := parsePositiveInt(c.DefaultQuery("limit", "20"), 20)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0)

	// 解析排序参数
	sortBy := c.DefaultQuery("sort_by", "id")
	sortDir := c.DefaultQuery("sort_dir", "desc")

	// 验证排序字段
	validSortFields := map[string]bool{
		"created_at": true,
		"filename":   true,
		"file_size":  true,
		"id":         true,
	}
	if !validSortFields[sortBy] {
		sortBy = "id"
	}

	// 验证排序方向
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	// 解析 has_tags 参数
	hasTagsStr := strings.TrimSpace(c.Query("has_tags"))
	var hasTags *bool
	if hasTagsStr != "" {
		val := strings.ToLower(hasTagsStr) == "true"
		hasTags = &val
	}

	hasPendingTagsStr := strings.TrimSpace(c.Query("has_pending_tags"))
	var hasPendingTags *bool
	if hasPendingTagsStr != "" {
		val := strings.ToLower(hasPendingTagsStr) == "true"
		hasPendingTags = &val
	}

	tagIDsStr := strings.TrimSpace(c.Query("tag_ids"))

	// 验证互斥性：has_tags=false 与 tag_ids 不能同时使用
	if hasTags != nil && !*hasTags && tagIDsStr != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "has_tags=false is incompatible with tag_ids parameter"})
		return
	}
	if hasPendingTags != nil && *hasPendingTags && hasTags != nil && !*hasTags {
		c.JSON(http.StatusBadRequest, gin.H{"error": "has_pending_tags=true is incompatible with has_tags=false"})
		return
	}
	if hasPendingTags != nil && *hasPendingTags && tagIDsStr != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "has_pending_tags=true is incompatible with tag_ids parameter"})
		return
	}

	var images []any
	var total int64

	// 根据 has_tags 参数决定查询方式
	if hasTags != nil && !*hasTags {
		// 查询未打标签的图片
		untaggedImages, err := h.imageRepo.FindUntagged(ctx, limit, offset, sortBy, sortDir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		total, err = h.imageRepo.CountUntagged(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, img := range untaggedImages {
			images = append(images, rewriteImageForRequest(c.Request, img))
		}
	} else if hasPendingTags != nil && *hasPendingTags {
		pendingImages, err := h.imageRepo.FindPendingTags(ctx, limit, offset, sortBy, sortDir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		total, err = h.imageRepo.CountPendingTags(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, img := range pendingImages {
			images = append(images, rewriteImageForRequest(c.Request, img))
		}
	} else if tagIDsStr != "" {
		// Parse comma-separated tag IDs
		tagIDs, err := parseTagIDs(tagIDsStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag_ids format"})
			return
		}

		filteredImages, err := h.imageRepo.FindByTagIDs(ctx, tagIDs, limit, offset, sortBy, sortDir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		total, err = h.imageRepo.CountByTagIDs(ctx, tagIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, img := range filteredImages {
			images = append(images, rewriteImageForRequest(c.Request, img))
		}
	} else {
		// No filter - return all images
		allImages, err := h.imageRepo.FindAll(limit, offset, sortBy, sortDir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		total, err = h.imageRepo.Count()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, img := range allImages {
			images = append(images, rewriteImageForRequest(c.Request, img))
		}
	}

	// Build response with Flutter-compatible structure
	hasMore := offset+len(images) < int(total)
	nextCursor := ""
	if hasMore && len(images) > 0 {
		nextCursor = strconv.Itoa(offset + len(images))
	}

	c.JSON(http.StatusOK, gin.H{
		"images":      images,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
		"total":       total,
	})
}

// GetImage handles GET /api/v1/images/:id
func (h *ImageHandler) GetImage(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	image, err := h.imageRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rewriteImageForRequest(c.Request, *image))
}

func (h *ImageHandler) PermanentDeleteImage(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if h.collectionRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "collection repository not configured"})
		return
	}

	ctx := c.Request.Context()
	image, err := h.imageRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	affectedCollectionIDs, err := h.collectionRepo.FindCollectionIDsByImage(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve related collections"})
		return
	}

	if err := h.fileActions.DeleteSourceAndThumbnails(*image); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete source file or thumbnails"})
		return
	}

	if err := h.imageRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete image record"})
		return
	}

	for _, collectionID := range affectedCollectionIDs {
		if err := h.collectionRepo.ReconcileAfterImageDelete(ctx, collectionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reconcile related collections"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
}

// TriggerImport handles POST /api/v1/images/scan and queues a manual scan job.
func (h *ImageHandler) TriggerImport(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  "import service not configured",
		})
		return
	}

	jobID, err := h.adminSvc.TriggerScan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  "failed to queue import: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status": "queued",
		"job_id": jobID,
	})
}

func (h *ImageHandler) ViewerWindow(c *gin.Context) {
	var req viewerWindowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondViewerRequestError(c, "invalid viewer request")
		return
	}
	if req.SelectedIndex < 0 || req.SelectedImageID <= 0 {
		respondViewerRequestError(c, "selected_index is out of range for the supplied snapshot")
		return
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 10 {
		limit = 10
	}

	source := strings.ToLower(strings.TrimSpace(req.Source))
	switch source {
	case "gallery":
		h.viewerWindowGallery(c, req, limit)
	case "search":
		h.viewerWindowSearch(c, req, limit)
	default:
		respondViewerRequestError(c, "unsupported viewer source")
	}
}

func (h *ImageHandler) viewerWindowGallery(c *gin.Context, req viewerWindowRequest, limit int) {
	ctx := c.Request.Context()
	sortBy, sortDir := normalizeGallerySort(req.Snapshot.SortBy, req.Snapshot.SortDir)
	if req.Snapshot.HasTags != nil && !*req.Snapshot.HasTags && len(req.Snapshot.TagIDs) > 0 {
		respondViewerRequestError(c, "selected_index is out of range for the supplied snapshot")
		return
	}
	if req.Snapshot.HasPendingTags != nil && *req.Snapshot.HasPendingTags {
		if req.Snapshot.HasTags != nil && !*req.Snapshot.HasTags {
			respondViewerRequestError(c, "selected_index is out of range for the supplied snapshot")
			return
		}
		if len(req.Snapshot.TagIDs) > 0 {
			respondViewerRequestError(c, "selected_index is out of range for the supplied snapshot")
			return
		}
	}

	var (
		total int64
		items []any
		err   error
	)

	switch {
	case req.Snapshot.HasTags != nil && !*req.Snapshot.HasTags:
		total, err = h.imageRepo.CountUntagged(ctx)
	case req.Snapshot.HasPendingTags != nil && *req.Snapshot.HasPendingTags:
		total, err = h.imageRepo.CountPendingTags(ctx)
	case len(req.Snapshot.TagIDs) > 0:
		total, err = h.imageRepo.CountByTagIDs(ctx, req.Snapshot.TagIDs)
	default:
		total, err = h.imageRepo.Count()
	}
	if err != nil {
		respondViewerServerError(c)
		return
	}
	if int64(req.SelectedIndex) >= total {
		respondViewerRequestError(c, "selected_index is out of range for the supplied snapshot")
		return
	}

	windowStart := serviceViewerWindowStart(req.SelectedIndex, limit, int(total))
	switch {
	case req.Snapshot.HasTags != nil && !*req.Snapshot.HasTags:
		untagged, err := h.imageRepo.FindUntagged(ctx, limit, windowStart, sortBy, sortDir)
		if err != nil {
			respondViewerServerError(c)
			return
		}
		for _, image := range untagged {
			items = append(items, image)
		}
	case req.Snapshot.HasPendingTags != nil && *req.Snapshot.HasPendingTags:
		pending, err := h.imageRepo.FindPendingTags(ctx, limit, windowStart, sortBy, sortDir)
		if err != nil {
			respondViewerServerError(c)
			return
		}
		for _, image := range pending {
			items = append(items, image)
		}
	case len(req.Snapshot.TagIDs) > 0:
		filtered, err := h.imageRepo.FindByTagIDs(ctx, req.Snapshot.TagIDs, limit, windowStart, sortBy, sortDir)
		if err != nil {
			respondViewerServerError(c)
			return
		}
		for _, image := range filtered {
			items = append(items, image)
		}
	default:
		all, err := h.imageRepo.FindAll(limit, windowStart, sortBy, sortDir)
		if err != nil {
			respondViewerServerError(c)
			return
		}
		for _, image := range all {
			items = append(items, image)
		}
	}

	selectedIndexInWindow := req.SelectedIndex - windowStart
	if selectedIndexInWindow < 0 || selectedIndexInWindow >= len(items) {
		respondViewerRequestError(c, "selected_index is out of range for the supplied snapshot")
		return
	}
	selectedItem, ok := items[selectedIndexInWindow].(domain.Image)
	if !ok {
		respondViewerServerError(c)
		return
	}
	if selectedItem.ID != req.SelectedImageID {
		respondViewerSnapshotDrift(c)
		return
	}

	respondViewerWindow(c, items, windowStart, req.SelectedIndex, selectedIndexInWindow, total)
}

func (h *ImageHandler) viewerWindowSearch(c *gin.Context, req viewerWindowRequest, limit int) {
	if h.searchSvc == nil {
		respondViewerServerError(c)
		return
	}
	result, err := h.searchSvc.ViewerWindow(c.Request.Context(), service.SearchOptions{
		Query:     strings.TrimSpace(req.Snapshot.Query),
		TagIDs:    req.Snapshot.TagIDs,
		SortBy:    req.Snapshot.SortBy,
		SortOrder: req.Snapshot.SortOrder,
	}, req.SelectedIndex, limit)
	if err != nil {
		if errors.Is(err, service.ErrViewerRequestOutOfRange) {
			respondViewerRequestError(c, err.Error())
			return
		}
		respondViewerServerError(c)
		return
	}
	if result.SelectedIndexInWindow < 0 || result.SelectedIndexInWindow >= len(result.Images) {
		respondViewerRequestError(c, "selected_index is out of range for the supplied snapshot")
		return
	}
	if result.Images[result.SelectedIndexInWindow].ID != req.SelectedImageID {
		respondViewerSnapshotDrift(c)
		return
	}
	items := make([]any, 0, len(result.Images))
	for _, image := range result.Images {
		items = append(items, image)
	}
	respondViewerWindow(c, items, result.WindowStart, result.SelectedIndex, result.SelectedIndexInWindow, result.Total)
}

func normalizeGallerySort(sortBy, sortDir string) (string, string) {
	validSortFields := map[string]bool{"created_at": true, "filename": true, "file_size": true, "id": true}
	if !validSortFields[sortBy] {
		sortBy = "id"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}
	return sortBy, sortDir
}

func serviceViewerWindowStart(selectedIndex, limit, total int) int {
	start := selectedIndex - limit/2
	start = max(start, 0)
	maxStart := total - limit
	maxStart = max(maxStart, 0)
	if start > maxStart {
		start = maxStart
	}
	return start
}

func respondViewerWindow(c *gin.Context, items []any, windowStart, selectedIndex, selectedIndexInWindow int, total int64) {
	c.JSON(http.StatusOK, gin.H{
		"items":                    rewriteViewerItemsForRequest(c.Request, items),
		"window_start_index":       windowStart,
		"selected_index":           selectedIndex,
		"selected_index_in_window": selectedIndexInWindow,
		"total":                    total,
		"has_previous":             selectedIndex > 0,
		"has_next":                 int64(selectedIndex+1) < total,
	})
}

func respondViewerRequestError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_viewer_request", "message": message})
}

func respondViewerSnapshotDrift(c *gin.Context) {
	c.JSON(http.StatusConflict, gin.H{"error": "viewer_snapshot_drift", "message": "selected_image_id no longer matches the supplied selected_index"})
}

func respondViewerServerError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "viewer_window_failed", "message": "failed to load viewer window"})
}

func parseTagIDs(s string) ([]int64, error) {
	parts := strings.Split(s, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, errors.New("invalid tag id: " + part)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
