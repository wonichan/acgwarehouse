package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

// AdminServiceInterface defines the interface for admin service operations.
// This allows for both real implementations and mocks for testing.
type AdminServiceInterface interface {
	GetSummary(ctx context.Context) (*service.Summary, error)
	GetTaskPlatformOverview(ctx context.Context) (*service.TaskPlatformOverview, error)
	GetJobs(ctx context.Context, limit int) ([]interface{}, error)
	GetTaskBatches(ctx context.Context, filter service.TaskBatchReadFilter) ([]service.TaskBatchReadModel, error)
	GetTasks(ctx context.Context, filter service.TaskReadFilter) ([]service.TaskReadModel, error)
	TriggerScan(ctx context.Context) (int64, error)
	RetryFailedJobs(ctx context.Context) (int, error)
	RetryFailedBatchTasks(ctx context.Context, batchID int64) (*service.RetryBatchResult, error)
	RetryFailedTask(ctx context.Context, taskID int64) (*service.RetryBatchResult, error)
	PauseBackgroundTasks(ctx context.Context) error
	ResumeBackgroundTasks(ctx context.Context) error
	ClearTaskQueue(ctx context.Context) (int, error)
	CancelTaskBatch(ctx context.Context, batchID int64) (int, error)
	CancelTask(ctx context.Context, taskID int64) (int, error)
	IsBackgroundRunning() bool
}

// GetTaskPlatformOverview returns the Phase 13 platform overview contract.
// GET /admin/api/task-platform/overview
func (h *AdminHandler) GetTaskPlatformOverview(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}

	overview, err := h.adminSvc.GetTaskPlatformOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get task platform overview: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, overview)
}

// AdminHandler handles admin dashboard HTTP endpoints.
type AdminHandler struct {
	cfg      *config.Config
	adminSvc AdminServiceInterface
}

// NewAdminHandler creates a new AdminHandler.
// Parameters: cfg - config for Basic Auth, adminSvc - the admin service interface
func NewAdminHandler(cfg *config.Config, adminSvc AdminServiceInterface) *AdminHandler {
	return &AdminHandler{
		cfg:      cfg,
		adminSvc: adminSvc,
	}
}

// AuthMiddleware returns a Basic Auth middleware for the admin routes.
// It checks the username and password against the config.
func (h *AdminHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth if admin is not configured (for development)
		if h.cfg.Admin.Username == "" && h.cfg.Admin.Password == "" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Header("WWW-Authenticate", `Basic realm="admin"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Expected format: "Basic base64(username:password)"
		if !strings.HasPrefix(authHeader, "Basic ") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		encoded := strings.TrimPrefix(authHeader, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		username := parts[0]
		password := parts[1]

		if username != h.cfg.Admin.Username || password != h.cfg.Admin.Password {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}

// GetSummary returns the aggregated status summary.
// GET /admin/api/summary
func (h *AdminHandler) GetSummary(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}

	summary, err := h.adminSvc.GetSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get summary: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetJobs returns recent jobs.
// GET /admin/api/jobs
func (h *AdminHandler) GetJobs(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		// Simple validation - check if it's a valid number
		valid := true
		for _, ch := range l {
			if ch < '0' || ch > '9' {
				valid = false
				break
			}
		}
		if valid && len(l) > 0 {
			// Use default if parsing fails
			var n int
			_, _ = fmt.Sscanf(l, "%d", &n)
			if n > 0 && n <= 1000 {
				limit = n
			}
		}
	}

	jobs, err := h.adminSvc.GetJobs(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get jobs: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// GetTaskBatches returns aggregated task batch read models.
// GET /admin/api/task-batches
func (h *AdminHandler) GetTaskBatches(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}

	filter := service.TaskBatchReadFilter{
		SourceType: c.Query("source_type"),
		Status:     c.Query("status"),
		Limit:      parsePositiveIntWithCap(c.Query("limit"), 50, 1000),
	}
	batches, err := h.adminSvc.GetTaskBatches(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get task batches: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"task_batches": batches})
}

// GetTasks returns task details, optionally filtered by batch.
// GET /admin/api/tasks
func (h *AdminHandler) GetTasks(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}

	filter := service.TaskReadFilter{
		TaskType: c.Query("task_type"),
		Status:   c.Query("status"),
		Limit:    parsePositiveIntWithCap(c.Query("limit"), 50, 1000),
	}
	if batchIDText := c.Query("batch_id"); batchIDText != "" {
		batchID, err := strconv.ParseInt(batchIDText, 10, 64)
		if err != nil || batchID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid batch_id"})
			return
		}
		filter.BatchID = &batchID
	}

	tasks, err := h.adminSvc.GetTasks(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get tasks: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func parsePositiveIntWithCap(raw string, fallback, max int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func parseRequiredPositiveInt(c *gin.Context, key string) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(c.Param(key)), 10, 64)
	if err != nil || value <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid " + key})
		return 0, false
	}
	return value, true
}

// TriggerScan triggers a manual scan job.
// POST /admin/api/actions/scan
func (h *AdminHandler) TriggerScan(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}

	jobID, err := h.adminSvc.TriggerScan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to trigger scan: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "scan triggered",
		"data":    gin.H{"job_id": jobID},
	})
}

// PauseBackgroundTasks pauses job processing.
// POST /admin/api/actions/jobs/pause
func (h *AdminHandler) PauseBackgroundTasks(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}

	err := h.adminSvc.PauseBackgroundTasks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to pause jobs: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "jobs paused",
	})
}

// ResumeBackgroundTasks resumes job processing.
// POST /admin/api/actions/jobs/resume
func (h *AdminHandler) ResumeBackgroundTasks(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}

	err := h.adminSvc.ResumeBackgroundTasks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to resume jobs: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "jobs resumed",
	})
}

// ClearTaskQueue clears pending/queued platform tasks.
// POST /admin/api/actions/jobs/clear-queue
func (h *AdminHandler) ClearTaskQueue(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}
	count, err := h.adminSvc.ClearTaskQueue(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to clear queue: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("cleared %d queued tasks", count), "data": gin.H{"count": count}})
}

// CancelTaskBatch cancels a platform task batch.
// POST /admin/api/actions/task-batches/:batch_id/cancel
func (h *AdminHandler) CancelTaskBatch(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}
	batchID, ok := parseRequiredPositiveInt(c, "batch_id")
	if !ok {
		return
	}
	count, err := h.adminSvc.CancelTaskBatch(c.Request.Context(), batchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to cancel batch: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("cancelled %d tasks in batch", count), "data": gin.H{"count": count}})
}

// CancelTask cancels a single platform task.
// POST /admin/api/actions/tasks/:task_id/cancel
func (h *AdminHandler) CancelTask(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}
	taskID, ok := parseRequiredPositiveInt(c, "task_id")
	if !ok {
		return
	}
	count, err := h.adminSvc.CancelTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to cancel task: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("cancelled %d task", count), "data": gin.H{"count": count}})
}

// RetryFailedJobs retries all failed jobs.
// POST /admin/api/actions/jobs/retry-failed
func (h *AdminHandler) RetryFailedJobs(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}

	count, err := h.adminSvc.RetryFailedJobs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to retry failed jobs: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "retry initiated",
		"data":    gin.H{"count": count},
	})
}

// RetryFailedBatchTasks retries failed tasks in a batch by creating a new batch.
// POST /admin/api/actions/task-batches/:batch_id/retry-failed
func (h *AdminHandler) RetryFailedBatchTasks(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}
	batchID, ok := parseRequiredPositiveInt(c, "batch_id")
	if !ok {
		return
	}
	result, err := h.adminSvc.RetryFailedBatchTasks(c.Request.Context(), batchID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("retry created %d tasks in batch %d", result.RetryCount, result.Batch.ID),
		"data": gin.H{
			"retry_count": result.RetryCount,
			"batch_id":    result.Batch.ID,
			"batch":       result.Batch,
			"tasks":       result.CreatedTasks,
		},
	})
}

// RetryFailedTask retries a single failed task by creating a new batch.
// POST /admin/api/actions/tasks/:task_id/retry-failed
func (h *AdminHandler) RetryFailedTask(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin service not configured"})
		return
	}
	taskID, ok := parseRequiredPositiveInt(c, "task_id")
	if !ok {
		return
	}
	result, err := h.adminSvc.RetryFailedTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("retry created %d task in batch %d", result.RetryCount, result.Batch.ID),
		"data": gin.H{
			"retry_count": result.RetryCount,
			"batch_id":    result.Batch.ID,
			"batch":       result.Batch,
			"tasks":       result.CreatedTasks,
		},
	})
}
