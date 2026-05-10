package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

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

func (h *ImageMoveHandler) CreateMoveJob(c *gin.Context) {
	var req domain.ImageMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	job, err := h.svc.CreateMoveJob(c.Request.Context(), req)
	if err != nil {
		respondImageMoveError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, job)
}

func (h *ImageMoveHandler) GetMoveJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	job, err := h.svc.GetMoveJob(c.Request.Context(), id)
	if err != nil {
		respondImageMoveError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *ImageMoveHandler) CancelMoveJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	job, err := h.svc.CancelMoveJob(c.Request.Context(), id)
	if err != nil {
		respondImageMoveError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *ImageMoveHandler) ListHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	history, err := h.svc.ListMoveHistory(c.Request.Context(), limit)
	if err != nil {
		respondImageMoveError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": history})
}

func respondImageMoveError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrImageMoveInvalidRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrImageMoveTagNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
	case errors.Is(err, sql.ErrNoRows):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
