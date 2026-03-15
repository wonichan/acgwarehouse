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
// When no tag_ids is provided, returns all images with pagination.
func (h *ImageHandler) ListImages(c *gin.Context) {
	ctx := c.Request.Context()
	limit := parsePositiveInt(c.DefaultQuery("limit", "20"), 20)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0)

	tagIDsStr := strings.TrimSpace(c.Query("tag_ids"))
	var images []interface{}
	var total int64

	if tagIDsStr != "" {
		// Parse comma-separated tag IDs
		tagIDs, err := parseTagIDs(tagIDsStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag_ids format"})
			return
		}

		filteredImages, err := h.imageRepo.FindByTagIDs(ctx, tagIDs, limit, offset)
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
		allImages, err := h.imageRepo.FindAll(limit, offset)
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
