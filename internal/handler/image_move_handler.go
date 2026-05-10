package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

type ImageMoveHandler struct {
	svc *service.ImageMoveService
}

func NewImageMoveHandler(svc *service.ImageMoveService) *ImageMoveHandler {
	return &ImageMoveHandler{svc: svc}
}

func (h *ImageMoveHandler) PreviewMove(c *gin.Context) {
	var req domain.ImageMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	preview, err := h.svc.PreviewMove(c.Request.Context(), req)
	if err != nil {
		respondImageMoveError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (h *ImageMoveHandler) ExecuteMove(c *gin.Context) {
	var req domain.ImageMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.svc.ExecuteMove(c.Request.Context(), req)
	if err != nil {
		respondImageMoveError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func respondImageMoveError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrImageMoveInvalidRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrImageMoveTagNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
