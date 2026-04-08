package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
)

// DuplicateHandler 重复检测 API 处理器
type DuplicateHandler struct {
	duplicateService *service.DuplicateService
	sidecarRuntime   *sidecar.Runtime
}

// NewDuplicateHandler 创建重复检测处理器实例
func NewDuplicateHandler(duplicateService *service.DuplicateService, sidecarRuntime *sidecar.Runtime) *DuplicateHandler {
	return &DuplicateHandler{
		duplicateService: duplicateService,
		sidecarRuntime:   sidecarRuntime,
	}
}

// DetectRequest 检测请求
type DetectRequest struct {
	Threshold int `json:"threshold"` // 汉明距离阈值，默认 40
}

// DetectResponse 检测响应
type DetectResponse struct {
	TaskID    string  `json:"task_id"`
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	Processed int     `json:"processed"`
	Total     int     `json:"total"`
	Message   string  `json:"message"`
}

type DuplicateTaskStatusResponse struct {
	TaskID      string  `json:"task_id"`
	Status      string  `json:"status"`
	Progress    float64 `json:"progress"`
	Processed   int     `json:"processed"`
	Total       int     `json:"total"`
	Message     string  `json:"message"`
	Error       string  `json:"error,omitempty"`
	GroupsFound int     `json:"groups_found,omitempty"`
}

// ListResponse 列表响应
type ListResponse struct {
	Groups  []domain.DuplicateGroupWithImages `json:"groups"`
	Total   int64                             `json:"total"`
	HasMore bool                              `json:"has_more"`
}

// DetectDuplicates POST /api/v1/duplicates/detect
// 触发重复检测
func (h *DuplicateHandler) DetectDuplicates(c *gin.Context) {
	if h.sidecarRuntime != nil && h.sidecarRuntime.State() != sidecar.StateReady {
		status := h.sidecarRuntime.Status()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "计算服务不可用，请检查 Python 侧车状态",
			"state":   string(status.State),
			"details": status.LastError,
		})
		return
	}

	var req DetectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 使用默认值，不强制要求请求体
		req.Threshold = 40
	}

	if req.Threshold <= 0 {
		req.Threshold = 40
	}
	if req.Threshold > 256 {
		req.Threshold = 256 // 256-bit pHash 最大汉明距离
	}

	task, err := h.duplicateService.StartDetectDuplicatesTask(service.DetectOptions{
		Threshold: req.Threshold,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, DetectResponse{
		TaskID:    task.TaskID,
		Status:    task.Status,
		Progress:  task.Progress,
		Processed: task.Processed,
		Total:     task.Total,
		Message:   task.Message,
	})
}

// GetDuplicateTaskStatus GET /api/v1/duplicates/tasks/:task_id
func (h *DuplicateHandler) GetDuplicateTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	task, ok := h.duplicateService.GetDuplicateTask(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, DuplicateTaskStatusResponse{
		TaskID:      task.TaskID,
		Status:      task.Status,
		Progress:    task.Progress,
		Processed:   task.Processed,
		Total:       task.Total,
		Message:     task.Message,
		Error:       task.Error,
		GroupsFound: task.GroupsFound,
	})
}

// StreamDuplicateTaskEvents GET /api/v1/duplicates/tasks/:task_id/events
func (h *DuplicateHandler) StreamDuplicateTaskEvents(c *gin.Context) {
	taskID := c.Param("task_id")
	updates, unsubscribe, ok := h.duplicateService.SubscribeDuplicateTask(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	defer unsubscribe()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	writeEvent := func(event string, payload any) {
		b, _ := json.Marshal(payload)
		_, _ = fmt.Fprintf(c.Writer, "event: %s\n", event)
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", string(b))
		flusher.Flush()
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			writeEvent("heartbeat", gin.H{"task_id": taskID, "ts": time.Now().UTC().Format(time.RFC3339Nano)})
		case task, ok := <-updates:
			if !ok {
				return
			}
			payload := DuplicateTaskStatusResponse{
				TaskID:      task.TaskID,
				Status:      task.Status,
				Progress:    task.Progress,
				Processed:   task.Processed,
				Total:       task.Total,
				Message:     task.Message,
				Error:       task.Error,
				GroupsFound: task.GroupsFound,
			}
			writeEvent("progress", payload)
			if task.Status == service.DuplicateTaskStatusCompleted || task.Status == service.DuplicateTaskStatusFailed {
				writeEvent("terminal", payload)
				return
			}
		}
	}
}

// ListDuplicates GET /api/v1/duplicates
// 获取重复组列表
func (h *DuplicateHandler) ListDuplicates(c *gin.Context) {
	limit := parsePositiveInt(c.DefaultQuery("limit", "20"), 20)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0)

	groups, total, err := h.duplicateService.GetDuplicateGroups(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	hasMore := offset+len(groups) < int(total)

	c.JSON(http.StatusOK, ListResponse{
		Groups:  rewriteDuplicateGroupsForRequest(c.Request, groups),
		Total:   total,
		HasMore: hasMore,
	})
}

// GetDuplicate GET /api/v1/duplicates/:id
// 获取单个重复组详情
func (h *DuplicateHandler) GetDuplicate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid group id",
		})
		return
	}

	group, err := h.duplicateService.GetDuplicateGroup(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "group not found",
		})
		return
	}

	c.JSON(http.StatusOK, rewriteDuplicateGroupForRequest(c.Request, *group))
}

// DeleteDuplicate DELETE /api/v1/duplicates/:id
// 删除重复组记录
func (h *DuplicateHandler) DeleteDuplicate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid group id",
		})
		return
	}

	if err := h.duplicateService.DeleteDuplicateGroup(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Group deleted",
	})
}
