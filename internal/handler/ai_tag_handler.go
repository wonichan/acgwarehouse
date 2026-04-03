package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

type AITagHandler struct {
	jobManager      *worker.Manager
	imageRepo       repository.ImageRepository
	jobRepo         repository.JobRepository
	taskRepo        repository.PlatformTaskRepository
	taskPlatformSvc *service.TaskPlatformService
	configProvider  func() *config.Config
}

func NewAITagHandler(jobManager *worker.Manager, imageRepo repository.ImageRepository, jobRepo repository.JobRepository, taskRepo repository.PlatformTaskRepository, taskPlatformSvc *service.TaskPlatformService, configProvider func() *config.Config) *AITagHandler {
	return &AITagHandler{jobManager: jobManager, imageRepo: imageRepo, jobRepo: jobRepo, taskRepo: taskRepo, taskPlatformSvc: taskPlatformSvc, configProvider: configProvider}
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

	// 解析可选的自定义提示词
	var req struct {
		Prompt string `json:"prompt"`
	}
	// 忽略解析错误，因为 prompt 是可选的
	_ = c.ShouldBindJSON(&req)

	result, err := h.enqueueAITagBatch(c.Request.Context(), domain.TaskBatchSourceManualSingle, []*domain.Image{image}, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, result)
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
	} else if status == "finished" {
		status = "completed"
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":   job.ID,
		"status":   status,
		"progress": job.Progress,
		"error":    job.Error,
	})
}

// GetDefaultPrompt 返回默认的 AI 标签生成提示词
func (h *AITagHandler) GetDefaultPrompt(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"default_prompt": worker.GetDefaultTagPrompt(),
	})
}

func (h *AITagHandler) BatchTriggerAITags(c *gin.Context) {
	var req struct {
		ImageIDs []int64 `json:"image_ids"`
		Prompt   string  `json:"prompt"` // 可选的自定义提示词
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	images := make([]*domain.Image, 0, len(req.ImageIDs))
	for _, imageID := range req.ImageIDs {
		image, err := h.imageRepo.FindByID(imageID)
		if err != nil {
			respondRepoError(c, err)
			return
		}
		images = append(images, image)
	}

	result, err := h.enqueueAITagBatch(c.Request.Context(), domain.TaskBatchSourceManualBatch, images, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, result)
}

type aiTagBatchResponse struct {
	BatchID         int64   `json:"batch_id"`
	SourceType      string  `json:"source_type"`
	Status          string  `json:"status"`
	CreatedTasks    int     `json:"created_tasks"`
	SkippedTasks    int     `json:"skipped_tasks"`
	PlatformTaskIDs []int64 `json:"platform_task_ids,omitempty"`
	JobIDs          []int64 `json:"job_ids,omitempty"`
}

func (h *AITagHandler) enqueueAITagBatch(ctx context.Context, sourceType string, images []*domain.Image, prompt string) (*aiTagBatchResponse, error) {
	if h.taskPlatformSvc == nil {
		return nil, sql.ErrConnDone
	}
	items := make([]service.TaskPlatformPlanItem, 0, len(images))
	roots := make([]string, 0, len(images))
	for _, image := range images {
		items = append(items, service.TaskPlatformPlanItem{
			ImageID:          image.ID,
			ImageVersionKey:  service.BuildImageVersionKey(image),
			SourceDescriptor: image.Path,
		})
		roots = append(roots, image.SourceRoot)
	}
	plan, err := h.taskPlatformSvc.PlanBatch(ctx, service.TaskPlatformPlanRequest{
		SourceType:   sourceType,
		SummaryLabel: service.BuildTaskBatchSummaryLabel(sourceType, roots, len(items)),
		SourceRoots:  roots,
		TaskTypes:    []string{domain.PlatformTaskTypeAITagGeneration},
		Items:        items,
	})
	if err != nil {
		return nil, err
	}
	response := &aiTagBatchResponse{
		BatchID:      plan.Batch.ID,
		SourceType:   sourceType,
		Status:       "skipped",
		CreatedTasks: len(plan.CreatedTasks),
		SkippedTasks: len(items) - len(plan.CreatedTasks),
	}
	for _, task := range plan.CreatedTasks {
		image, err := h.imageRepo.FindByID(task.ImageID)
		if err != nil {
			return nil, err
		}
		payload, err := json.Marshal(worker.AITagPayload{ImageID: task.ImageID, Path: service.ResolveAITagImagePath(image), Prompt: prompt})
		if err != nil {
			return nil, err
		}
		job, err := h.taskPlatformSvc.QueueTask(ctx, &task, domain.PlatformTaskTypeAITagGeneration, string(payload))
		if err != nil {
			return nil, err
		}
		response.PlatformTaskIDs = append(response.PlatformTaskIDs, task.ID)
		response.JobIDs = append(response.JobIDs, job.ID)
		if h.jobManager != nil && h.jobManager.QueuedByType(domain.PlatformTaskTypeAITagGeneration) < service.ResolveAITagQueueLimit(h.currentConfig()) {
			h.jobManager.LoadExistingJob(job)
		}
	}
	if len(response.JobIDs) > 0 {
		response.Status = "queued"
	}
	return response, nil
}

func (h *AITagHandler) findLatestJobForImage(imageID int64) (*repositoryJobView, error) {
	if h.taskRepo != nil {
		tasks, err := h.taskRepo.ListByImageAndTypes(context.Background(), imageID, []string{domain.PlatformTaskTypeAITagGeneration})
		if err == nil && len(tasks) > 0 {
			latest := tasks[len(tasks)-1]
			if latest.LatestAsyncJobID != nil {
				job, err := h.jobRepo.FindByID(*latest.LatestAsyncJobID)
				if err == nil {
					return &repositoryJobView{ID: job.ID, Status: job.Status, Progress: job.Progress, Error: job.Error}, nil
				}
			}
		}
	}
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

func (h *AITagHandler) currentConfig() *config.Config {
	if h == nil || h.configProvider == nil {
		return nil
	}
	return h.configProvider()
}

type repositoryJobView struct {
	ID       int64
	Status   string
	Progress float64
	Error    *string
}
