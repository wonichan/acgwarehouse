package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

type BatchHandler struct {
	svc *service.BatchService
}

func NewBatchHandler(svc *service.BatchService) *BatchHandler {
	return &BatchHandler{svc: svc}
}

// BatchAddTags handles POST /api/v1/batch/tags/add
func (h *BatchHandler) BatchAddTags(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		ImageIDs []int64 `json:"image_ids" binding:"required"`
		TagIDs   []int64 `json:"tag_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.svc.BatchAddTags(ctx, req.ImageIDs, req.TagIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "tags added successfully",
		"images_updated": count,
	})
}

// BatchRemoveTags handles POST /api/v1/batch/tags/remove
func (h *BatchHandler) BatchRemoveTags(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		ImageIDs []int64 `json:"image_ids" binding:"required"`
		TagIDs   []int64 `json:"tag_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.svc.BatchRemoveTags(ctx, req.ImageIDs, req.TagIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "tags removed successfully",
		"images_updated": count,
	})
}

// BatchMoveToCollection handles POST /api/v1/batch/collections/move
func (h *BatchHandler) BatchMoveToCollection(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		ImageIDs     []int64 `json:"image_ids" binding:"required"`
		CollectionID int64   `json:"collection_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.svc.BatchMoveToCollection(ctx, req.ImageIDs, req.CollectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "images moved successfully",
		"images_moved": count,
	})
}

// BatchRemoveFromCollection handles POST /api/v1/batch/collections/remove
func (h *BatchHandler) BatchRemoveFromCollection(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		ImageIDs     []int64 `json:"image_ids" binding:"required"`
		CollectionID int64   `json:"collection_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.svc.BatchRemoveFromCollection(ctx, req.ImageIDs, req.CollectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "images removed from collection",
		"images_removed": count,
	})
}

// BatchDeleteImages handles POST /api/v1/batch/images/delete
func (h *BatchHandler) BatchDeleteImages(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		ImageIDs []int64 `json:"image_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.svc.BatchDeleteImages(ctx, req.ImageIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "images deleted successfully",
		"images_deleted": count,
	})
}
