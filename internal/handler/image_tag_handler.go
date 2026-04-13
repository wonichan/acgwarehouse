package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

type ImageTagHandler struct {
	imageTagRepo  repository.ImageTagRepository
	tagRepo       repository.TagRepository
	aliasRepo     repository.TagAliasRepository
	imageRepo     repository.ImageRepository
	governanceSvc *service.TagGovernanceService
}

func NewImageTagHandler(
	imageTagRepo repository.ImageTagRepository,
	tagRepo repository.TagRepository,
	aliasRepo repository.TagAliasRepository,
	imageRepo repository.ImageRepository,
	governanceSvc *service.TagGovernanceService,
) *ImageTagHandler {
	return &ImageTagHandler{
		imageTagRepo:  imageTagRepo,
		tagRepo:       tagRepo,
		aliasRepo:     aliasRepo,
		imageRepo:     imageRepo,
		governanceSvc: governanceSvc,
	}
}

func (h *ImageTagHandler) GetImageTags(c *gin.Context) {
	imageID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if _, err := h.imageRepo.FindByID(imageID); err != nil {
		respondRepoError(c, err)
		return
	}

	items, err := h.imageTagRepo.FindByImageID(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"confirmed": make([]gin.H, 0),
		"pending":   make([]gin.H, 0),
		"rejected":  make([]gin.H, 0),
	}
	for _, item := range items {
		tag, err := h.tagRepo.FindByID(c.Request.Context(), item.TagID)
		if err != nil {
			respondRepoError(c, err)
			return
		}
		payload := gin.H{
			"image_id":        item.ImageID,
			"tag_id":          item.TagID,
			"preferred_label": tag.PreferredLabel,
			"level":           tag.Level,
			"parent_id":       tag.ParentID,
			"review_state":    item.ReviewState,
			"confidence":      item.Confidence,
		}
		bucket := item.ReviewState
		if _, ok := response[bucket]; !ok {
			bucket = "pending"
		}
		response[bucket] = append(response[bucket].([]gin.H), payload)
	}

	c.JSON(http.StatusOK, response)
}

func (h *ImageTagHandler) AddImageTag(c *gin.Context) {
	imageID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if _, err := h.imageRepo.FindByID(imageID); err != nil {
		respondRepoError(c, err)
		return
	}

	var req struct {
		TagID    int64  `json:"tag_id"`
		TagLabel string `json:"tag_label"`
		Level    string `json:"level"`
		ParentID *int64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	tagID := req.TagID
	if tagID == 0 {
		label := strings.TrimSpace(req.TagLabel)
		if label == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tag_id or tag_label is required"})
			return
		}
		tag, _, err := resolveOrCreateManualTag(c.Request.Context(), h.tagRepo, h.aliasRepo, manualTagCreateInput{
			PreferredLabel: label,
			Level:          req.Level,
			ParentID:       req.ParentID,
			ReviewState:    "confirmed",
		})
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, service.ErrInvalidHierarchy) {
				status = http.StatusBadRequest
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		tagID = tag.ID
	}

	tag, err := h.tagRepo.FindByID(c.Request.Context(), tagID)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	if err := h.imageTagRepo.Save(c.Request.Context(), &domain.ImageTag{ImageID: imageID, TagID: tagID, ReviewState: "confirmed", Confidence: 1}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"image_id":        imageID,
		"tag_id":          tagID,
		"preferred_label": tag.PreferredLabel,
		"level":           tag.Level,
		"parent_id":       tag.ParentID,
		"review_state":    "confirmed",
	})
}

func (h *ImageTagHandler) RemoveImageTag(c *gin.Context) {
	imageID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	tagID, ok := parseIDParam(c, "tag_id")
	if !ok {
		return
	}

	rowsAffected, err := h.imageTagRepo.Delete(c.Request.Context(), imageID, tagID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "deleted": rowsAffected > 0})
}

func (h *ImageTagHandler) ReviewTag(c *gin.Context) {
	imageID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	tagID, ok := parseIDParam(c, "tag_id")
	if !ok {
		return
	}

	var req struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	state, ok := reviewActionToState(req.Action)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be confirm or reject"})
		return
	}

	// Get current review state to determine if usage count should change
	tags, err := h.imageTagRepo.FindByImageID(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var currentState string
	for _, t := range tags {
		if t.TagID == tagID {
			currentState = t.ReviewState
			break
		}
	}
	if currentState == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "image-tag association not found"})
		return
	}

	if err := h.imageTagRepo.UpdateReviewState(c.Request.Context(), imageID, tagID, state); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "review_state": state})
}

func (h *ImageTagHandler) BatchReview(c *gin.Context) {
	imageID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		TagIDs []int64 `json:"tag_ids"`
		Action string  `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	state, ok := reviewActionToState(req.Action)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be confirm or reject"})
		return
	}

	// Get current states of all tags for this image
	tags, err := h.imageTagRepo.FindByImageID(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	currentStates := make(map[int64]string)
	for _, t := range tags {
		currentStates[t.TagID] = t.ReviewState
	}

	// Apply review state changes
	if err := h.imageTagRepo.BatchUpdateReviewState(c.Request.Context(), imageID, req.TagIDs, state); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "updated_count": len(req.TagIDs)})
}

func reviewActionToState(action string) (string, bool) {
	switch strings.TrimSpace(action) {
	case "confirm":
		return "confirmed", true
	case "reject":
		return "rejected", true
	default:
		return "", false
	}
}

// MergeImageTag handles POST /api/v1/images/:id/tags/:tag_id/merge
// It reassigns a pending AI tag on an image to an existing target tag.
func (h *ImageTagHandler) MergeImageTag(c *gin.Context) {
	imageID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	sourceTagID, ok := parseIDParam(c, "tag_id")
	if !ok {
		return
	}

	var req struct {
		TargetTagID    int64  `json:"target_tag_id"`
		TargetLabel    string `json:"target_label"`
		TargetLevel    string `json:"target_level"`
		TargetParentID *int64 `json:"target_parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	targetTagID := req.TargetTagID
	sourceTag, err := h.tagRepo.FindByID(c.Request.Context(), sourceTagID)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	if targetTagID == 0 {
		label := strings.TrimSpace(req.TargetLabel)
		if label == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target_tag_id or target_label is required"})
			return
		}
		tag, _, err := resolveOrCreateManualTag(c.Request.Context(), h.tagRepo, h.aliasRepo, manualTagCreateInput{
			PreferredLabel: label,
			Level:          req.TargetLevel,
			ParentID:       req.TargetParentID,
			ReviewState:    "confirmed",
		})
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, service.ErrInvalidHierarchy) {
				status = http.StatusBadRequest
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		targetTagID = tag.ID
	}

	// Verify target tag exists
	targetTag, err := h.tagRepo.FindByID(c.Request.Context(), targetTagID)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	if sourceTag.Level != targetTag.Level {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source and target tags must be at the same level"})
		return
	}

	if err := h.imageTagRepo.MergeImageTag(c.Request.Context(), imageID, sourceTagID, targetTagID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                true,
		"image_id":               imageID,
		"source_tag_id":          sourceTagID,
		"target_tag_id":          targetTagID,
		"target_preferred_label": targetTag.PreferredLabel,
	})
}
