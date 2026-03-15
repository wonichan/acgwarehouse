package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

type AITagHandler struct {
	jobManager *worker.Manager
	imageRepo  repository.ImageRepository
	jobRepo    repository.JobRepository
}

func NewAITagHandler(jobManager *worker.Manager, imageRepo repository.ImageRepository, jobRepo repository.JobRepository) *AITagHandler {
	return &AITagHandler{jobManager: jobManager, imageRepo: imageRepo, jobRepo: jobRepo}
}

func (h *AITagHandler) TriggerAITags(c *gin.Context) {
	imageID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	image, err := h.imageRepo.FindByID(imageID)
	if err != nil {
		respondRepoError(c, err)
		return
	}

	payload, err := json.Marshal(worker.AITagPayload{ImageID: imageID, Path: image.Path})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	jobID, err := h.jobManager.AddJob(c.Request.Context(), "ai_tag_generation", string(payload))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "status": "queued"})
}

func (h *AITagHandler) GetAITagStatus(c *gin.Context) {
	imageID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if _, err := h.imageRepo.FindByID(imageID); err != nil {
		respondRepoError(c, err)
		return
	}

	job, err := h.findLatestJobForImage(imageID)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	status := job.Status
	if status == "ready" {
		status = "queued"
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":   job.ID,
		"status":   status,
		"progress": job.Progress,
		"error":    job.Error,
	})
}

func (h *AITagHandler) BatchTriggerAITags(c *gin.Context) {
	var req struct {
		ImageIDs []int64 `json:"image_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	jobIDs := make([]int64, 0, len(req.ImageIDs))
	for _, imageID := range req.ImageIDs {
		image, err := h.imageRepo.FindByID(imageID)
		if err != nil {
			respondRepoError(c, err)
			return
		}
		payload, err := json.Marshal(worker.AITagPayload{ImageID: imageID, Path: image.Path})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		jobID, err := h.jobManager.AddJob(c.Request.Context(), "ai_tag_generation", string(payload))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		jobIDs = append(jobIDs, jobID)
	}

	c.JSON(http.StatusAccepted, gin.H{"job_ids": jobIDs})
}

func (h *AITagHandler) findLatestJobForImage(imageID int64) (*repositoryJobView, error) {
	jobs, err := h.jobRepo.FindByType("ai_tag_generation")
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		var payload worker.AITagPayload
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			continue
		}
		if payload.ImageID == imageID {
			return &repositoryJobView{ID: job.ID, Status: job.Status, Progress: job.Progress, Error: job.Error}, nil
		}
	}
	return nil, sql.ErrNoRows
}

type repositoryJobView struct {
	ID       int64
	Status   string
	Progress float64
	Error    *string
}
