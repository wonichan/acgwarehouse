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
	GetJobs(ctx context.Context, limit int) ([]interface{}, error)
	GetTaskBatches(ctx context.Context, filter service.TaskBatchReadFilter) ([]service.TaskBatchReadModel, error)
	GetTasks(ctx context.Context, filter service.TaskReadFilter) ([]service.TaskReadModel, error)
	TriggerScan(ctx context.Context) (int64, error)
	RetryFailedJobs(ctx context.Context) (int, error)
	PauseBackgroundTasks(ctx context.Context) error
	ResumeBackgroundTasks(ctx context.Context) error
	IsBackgroundRunning() bool
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
