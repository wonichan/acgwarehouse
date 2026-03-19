package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type ImageHandler struct {
	imageRepo    repository.ImageRepository
	tagRepo      repository.TagRepository
	imageTagRepo repository.ImageTagRepository
}

func NewImageHandler(imageRepo repository.ImageRepository, tagRepo repository.TagRepository, imageTagRepo repository.ImageTagRepository) *ImageHandler {
	return &ImageHandler{
		imageRepo:    imageRepo,
		tagRepo:      tagRepo,
		imageTagRepo: imageTagRepo,
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

	tagIDsStr := strings.TrimSpace(c.Query("tag_ids"))

	// 验证互斥性：has_tags=false 与 tag_ids 不能同时使用
	if hasTags != nil && !*hasTags && tagIDsStr != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "has_tags=false is incompatible with tag_ids parameter"})
		return
	}

	var images []interface{}
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
			images = append(images, img)
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
			images = append(images, img)
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
			images = append(images, img)
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

	c.JSON(http.StatusOK, image)
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
