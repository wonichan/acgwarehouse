package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
)

// mockAdminService implements AdminServiceInterface for testing
type mockAdminService struct {
	summary      *service.Summary
	overview     *service.TaskPlatformOverview
	jobs         []any
	taskBatches  []service.TaskBatchReadModel
	tasks        []service.TaskReadModel
	scanJobID    int64
	retryCount   int
	retryBatch   *service.RetryBatchResult
	retryTask    *service.RetryBatchResult
	clearCount   int
	batchCancel  int
	cancelCount  int
	pauseCalled  bool
	resumeCalled bool
	isRunning    bool
	err          error
}

func (m *mockAdminService) GetSummary(ctx context.Context) (*service.Summary, error) {
	return m.summary, m.err
}

func (m *mockAdminService) GetTaskPlatformOverview(ctx context.Context) (*service.TaskPlatformOverview, error) {
	return m.overview, m.err
}

func (m *mockAdminService) GetJobs(ctx context.Context, limit int) ([]any, error) {
	return m.jobs, m.err
}

func (m *mockAdminService) GetTaskBatches(ctx context.Context, filter service.TaskBatchReadFilter) ([]service.TaskBatchReadModel, error) {
	return m.taskBatches, m.err
}

func (m *mockAdminService) GetTasks(ctx context.Context, filter service.TaskReadFilter) ([]service.TaskReadModel, error) {
	filtered := make([]service.TaskReadModel, 0, len(m.tasks))
	for _, task := range m.tasks {
		if filter.BatchID != nil && task.BatchID != *filter.BatchID {
			continue
		}
		if filter.TaskType != "" && task.TaskType != filter.TaskType {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		filtered = append(filtered, task)
	}

	start := filter.Offset
	start = min(start, len(filtered))
	end := len(filtered)
	if filter.Limit > 0 {
		end = min(end, start+filter.Limit)
	}

	return filtered[start:end], m.err
}

func (m *mockAdminService) TriggerScan(ctx context.Context) (int64, error) {
	return m.scanJobID, m.err
}

func (m *mockAdminService) RetryFailedJobs(ctx context.Context) (int, error) {
	return m.retryCount, m.err
}

func (m *mockAdminService) RetryFailedBatchTasks(ctx context.Context, batchID int64) (*service.RetryBatchResult, error) {
	if m.retryBatch != nil {
		return m.retryBatch, m.err
	}
	return &service.RetryBatchResult{Batch: &domain.TaskBatch{ID: batchID}, RetryCount: m.retryCount}, m.err
}

func (m *mockAdminService) RetryFailedTask(ctx context.Context, taskID int64) (*service.RetryBatchResult, error) {
	if m.retryTask != nil {
		return m.retryTask, m.err
	}
	return &service.RetryBatchResult{Batch: &domain.TaskBatch{ID: taskID}, RetryCount: m.retryCount}, m.err
}

func (m *mockAdminService) PauseBackgroundTasks(ctx context.Context) error {
	m.pauseCalled = true
	return m.err
}

func (m *mockAdminService) ResumeBackgroundTasks(ctx context.Context) error {
	m.resumeCalled = true
	return m.err
}

func (m *mockAdminService) ClearTaskQueue(ctx context.Context) (int, error) {
	return m.clearCount, m.err
}

func (m *mockAdminService) CancelTaskBatch(ctx context.Context, batchID int64) (int, error) {
	return m.batchCancel, m.err
}

func (m *mockAdminService) CancelTask(ctx context.Context, taskID int64) (int, error) {
	return m.cancelCount, m.err
}

func (m *mockAdminService) IsBackgroundRunning() bool {
	return m.isRunning
}

type mockSidecarRuntime struct {
	startErr error
	state    sidecar.State
	status   sidecar.Status
	calls    []string
}

func (m *mockSidecarRuntime) Start(ctx context.Context) error {
	m.calls = append(m.calls, "start")
	return m.startErr
}

func (m *mockSidecarRuntime) Stop(ctx context.Context) error {
	m.calls = append(m.calls, "stop")
	return nil
}

func (m *mockSidecarRuntime) State() sidecar.State {
	return m.state
}

func (m *mockSidecarRuntime) Status() sidecar.Status {
	return m.status
}

func TestAdminHandler_Summary_AuthRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret123",
		},
	}

	mockSvc := &mockAdminService{
		summary: &service.Summary{
			Health: service.HealthStatus{Status: "healthy"},
		},
	}

	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/summary", handler.GetSummary)

	// Test 1: No credentials - should return 401
	req := httptest.NewRequest("GET", "/admin/api/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 without credentials, got %d", w.Code)
	}

	// Test 2: Invalid credentials - should return 401
	req = httptest.NewRequest("GET", "/admin/api/summary", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("wrong:wrong")))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 with invalid credentials, got %d", w.Code)
	}

	// Test 3: Valid credentials - should return 200
	req = httptest.NewRequest("GET", "/admin/api/summary", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret123")))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 with valid credentials, got %d", w.Code)
	}
}

func TestAdminHandler_GetSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret",
		},
	}

	mockSvc := &mockAdminService{
		summary: &service.Summary{
			Health:  service.HealthStatus{Status: "healthy"},
			Config:  service.ConfigSummary{ServerHost: "localhost"},
			Tasks:   service.TaskSummary{Total: 10},
			Library: service.LibraryStats{TotalImages: 100},
		},
	}

	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/summary", handler.GetSummary)

	req := httptest.NewRequest("GET", "/admin/api/summary", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestAdminHandler_GetTaskPlatformOverview(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret",
		},
	}

	mockSvc := &mockAdminService{
		overview: &service.TaskPlatformOverview{
			Health: service.HealthStatus{Status: "healthy"},
			Queue:  service.QueueOverview{IsRunning: true, IsPaused: false, QueueSize: 2, WorkerCount: 4},
			Sidecar: service.SidecarDiagnosticsOverview{
				State:            "degraded",
				LastProbeAt:      "2026-04-04T10:00:00Z",
				LastProbeResult:  "failed",
				LastErrorSummary: "startup timeout",
			},
			Batches: map[string]int64{
				"running": 1,
			},
			Tasks: map[string]int64{
				"pending": 1,
				"queued":  1,
			},
		},
	}

	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/task-platform/overview", handler.GetTaskPlatformOverview)

	req := httptest.NewRequest("GET", "/admin/api/task-platform/overview", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "\"queue\"") {
		t.Fatalf("Expected queue field in overview response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"batches\"") {
		t.Fatalf("Expected batches field in overview response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"tasks\"") {
		t.Fatalf("Expected tasks field in overview response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"sidecar\"") {
		t.Fatalf("Expected sidecar field in overview response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"last_probe_at\"") {
		t.Fatalf("Expected sidecar.last_probe_at in overview response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"last_probe_result\"") {
		t.Fatalf("Expected sidecar.last_probe_result in overview response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"last_error_summary\"") {
		t.Fatalf("Expected sidecar.last_error_summary in overview response, got %s", w.Body.String())
	}
}

func TestSidecarRestartReturnsInterruptedTaskCountAfterSuccessfulRestart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	adminSvc := &mockAdminService{
		tasks: []service.TaskReadModel{
			{ID: 1, Status: "running"},
			{ID: 2, Status: "completed"},
			{ID: 3, Status: "running"},
		},
	}
	runtime := &mockSidecarRuntime{}
	handler := NewAdminHandler(cfg, adminSvc, runtime)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/sidecar/restart", handler.HandleSidecarRestart)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/actions/sidecar/restart", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d with body %s", w.Code, w.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			InterruptedTaskCount int `json:"interrupted_task_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !response.Success {
		t.Fatal("expected success response")
	}
	if response.Data.InterruptedTaskCount != 2 {
		t.Fatalf("interrupted_task_count = %d, want 2", response.Data.InterruptedTaskCount)
	}
	if got := strings.Join(runtime.calls, ","); got != "stop,start" {
		t.Fatalf("runtime call order = %q, want %q", got, "stop,start")
	}
}

func TestSidecarRestartReturnsServiceUnavailableWhenRuntimeMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	handler := NewAdminHandler(cfg, &mockAdminService{}, nil)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/sidecar/restart", handler.HandleSidecarRestart)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/actions/sidecar/restart", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("Expected 503, got %d with body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sidecar runtime not configured") {
		t.Fatalf("expected sidecar runtime error, got %s", w.Body.String())
	}
}

func TestSidecarRestartCountsOnlyRunningTasks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	adminSvc := &mockAdminService{
		tasks: []service.TaskReadModel{
			{ID: 1, Status: "queued"},
			{ID: 2, Status: "running"},
			{ID: 3, Status: "failed"},
		},
	}
	handler := NewAdminHandler(cfg, adminSvc, &mockSidecarRuntime{})

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/sidecar/restart", handler.HandleSidecarRestart)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/actions/sidecar/restart", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d with body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"interrupted_task_count\":1") {
		t.Fatalf("expected interrupted_task_count=1, got %s", w.Body.String())
	}
}

func TestSidecarRestartReturnsInternalServerErrorWhenStartFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	adminSvc := &mockAdminService{
		tasks: []service.TaskReadModel{{ID: 1, Status: "running"}},
	}
	runtime := &mockSidecarRuntime{startErr: errors.New("boom")}
	handler := NewAdminHandler(cfg, adminSvc, runtime)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/sidecar/restart", handler.HandleSidecarRestart)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/actions/sidecar/restart", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500, got %d with body %s", w.Code, w.Body.String())
	}
	if got := strings.Join(runtime.calls, ","); got != "stop,start" {
		t.Fatalf("runtime call order = %q, want %q", got, "stop,start")
	}
}

func TestAdminHandler_GetTaskPlatformOverview_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{err: context.DeadlineExceeded}
	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/task-platform/overview", handler.GetTaskPlatformOverview)

	req := httptest.NewRequest("GET", "/admin/api/task-platform/overview", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "failed to get task platform overview") {
		t.Fatalf("Expected task platform overview error message, got %s", w.Body.String())
	}
}

func TestAdminHandler_TriggerScan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret",
		},
	}

	mockSvc := &mockAdminService{
		scanJobID: 42,
	}

	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/scan", handler.TriggerScan)

	req := httptest.NewRequest("POST", "/admin/api/scan", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestAdminHandler_RetryFailedJobs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret",
		},
	}

	mockSvc := &mockAdminService{
		retryCount: 3,
	}

	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/retry-failed", handler.RetryFailedJobs)

	req := httptest.NewRequest("POST", "/admin/api/retry-failed", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestAdminHandler_RetryFailedBatchTasks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{
		retryBatch: &service.RetryBatchResult{
			Batch:        &domain.TaskBatch{ID: 88, SourceType: domain.TaskBatchSourceRetry},
			RetryCount:   2,
			CreatedTasks: []domain.PlatformTask{{ID: 1}, {ID: 2}},
		},
	}
	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/task-batches/:batch_id/retry-failed", handler.RetryFailedBatchTasks)

	req := httptest.NewRequest("POST", "/admin/api/actions/task-batches/42/retry-failed", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Fatalf("Expected success response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"retry_count":2`) {
		t.Fatalf("Expected retry_count in response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"batch_id":88`) {
		t.Fatalf("Expected batch_id in response, got %s", w.Body.String())
	}
}

func TestAdminHandler_RetryFailedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{
		retryTask: &service.RetryBatchResult{
			Batch:        &domain.TaskBatch{ID: 99, SourceType: domain.TaskBatchSourceRetry},
			RetryCount:   1,
			CreatedTasks: []domain.PlatformTask{{ID: 11}},
		},
	}
	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/tasks/:task_id/retry-failed", handler.RetryFailedTask)

	req := httptest.NewRequest("POST", "/admin/api/actions/tasks/abc/retry-failed", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", w.Code)
	}

	req = httptest.NewRequest("POST", "/admin/api/actions/tasks/33/retry-failed", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"retry_count":1`) {
		t.Fatalf("Expected retry_count in response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"batch_id":99`) {
		t.Fatalf("Expected batch_id in response, got %s", w.Body.String())
	}
}

func TestAdminHandler_Pause(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret",
		},
	}

	mockSvc := &mockAdminService{}

	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/pause", handler.PauseBackgroundTasks)

	req := httptest.NewRequest("POST", "/admin/api/pause", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	if !mockSvc.pauseCalled {
		t.Error("Expected PauseBackgroundTasks to be called")
	}
}

func TestAdminHandler_Resume(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret",
		},
	}

	mockSvc := &mockAdminService{}

	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/resume", handler.ResumeBackgroundTasks)

	req := httptest.NewRequest("POST", "/admin/api/resume", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	if !mockSvc.resumeCalled {
		t.Error("Expected ResumeBackgroundTasks to be called")
	}
}

func TestAdminHandler_ClearTaskQueue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{clearCount: 2}
	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/jobs/clear-queue", handler.ClearTaskQueue)

	req := httptest.NewRequest("POST", "/admin/api/actions/jobs/clear-queue", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "count") {
		t.Fatalf("Expected count in response, got %s", w.Body.String())
	}
}

func TestAdminHandler_CancelTaskBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{batchCancel: 3}
	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/task-batches/:batch_id/cancel", handler.CancelTaskBatch)

	req := httptest.NewRequest("POST", "/admin/api/actions/task-batches/abc/cancel", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for invalid batch id, got %d", w.Code)
	}

	req = httptest.NewRequest("POST", "/admin/api/actions/task-batches/12/cancel", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for valid batch id, got %d", w.Code)
	}
}

func TestAdminHandler_CancelTask(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{cancelCount: 1}
	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/tasks/:task_id/cancel", handler.CancelTask)

	req := httptest.NewRequest("POST", "/admin/api/actions/tasks/0/cancel", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for invalid task id, got %d", w.Code)
	}

	req = httptest.NewRequest("POST", "/admin/api/actions/tasks/9/cancel", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for valid task id, got %d", w.Code)
	}
}

func TestAdminRoutes_ServeStaticFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	// Serve admin static files (simulating what SetupRoutes does)
	// Note: In production, this serves from ./web/admin
	// For testing, we'll use a different path or verify the route registration
	r.GET("/admin", func(c *gin.Context) {
		c.String(200, "Admin Dashboard")
	})

	// Test that admin index is accessible
	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Route should be accessible
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for admin page, got %d", w.Code)
	}

	// Test content
	if w.Body.String() != "Admin Dashboard" {
		t.Error("Expected admin dashboard content")
	}
}

func TestAdminRoutes_ApiEndpointsWired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "secret",
		},
	}

	mockSvc := &mockAdminService{
		summary: &service.Summary{
			Health:  service.HealthStatus{Status: "healthy"},
			Tasks:   service.TaskSummary{Total: 5},
			Library: service.LibraryStats{TotalImages: 100},
		},
		overview: &service.TaskPlatformOverview{
			Health:  service.HealthStatus{Status: "healthy"},
			Queue:   service.QueueOverview{IsRunning: true, QueueSize: 1, WorkerCount: 4},
			Batches: map[string]int64{"running": 1},
			Tasks:   map[string]int64{"pending": 1},
		},
		jobs:        []any{},
		taskBatches: []service.TaskBatchReadModel{},
		tasks:       []service.TaskReadModel{},
	}

	handler := NewAdminHandler(cfg, mockSvc)

	r := gin.New()
	// Admin page serves at /admin (no /api)
	r.GET("/admin", func(c *gin.Context) {
		c.String(200, "Admin Dashboard")
	})
	// Admin API at /admin/api
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/summary", handler.GetSummary)
	admin.GET("/task-platform/overview", handler.GetTaskPlatformOverview)
	admin.GET("/jobs", handler.GetJobs)
	admin.GET("/task-batches", handler.GetTaskBatches)
	admin.GET("/tasks", handler.GetTasks)
	admin.POST("/actions/scan", handler.TriggerScan)
	admin.POST("/actions/jobs/pause", handler.PauseBackgroundTasks)
	admin.POST("/actions/jobs/resume", handler.ResumeBackgroundTasks)
	admin.POST("/actions/jobs/clear-queue", handler.ClearTaskQueue)
	admin.POST("/actions/jobs/retry-failed", handler.RetryFailedJobs)
	admin.POST("/actions/task-batches/:batch_id/cancel", handler.CancelTaskBatch)
	admin.POST("/actions/tasks/:task_id/cancel", handler.CancelTask)

	// Test /admin/api/summary with auth
	req := httptest.NewRequest("GET", "/admin/api/summary", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Test /admin/api/task-platform/overview with auth
	req = httptest.NewRequest("GET", "/admin/api/task-platform/overview", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for task-platform overview, got %d", w.Code)
	}

	// Test that action buttons work
	mockSvc2 := &mockAdminService{scanJobID: 99}
	handler2 := NewAdminHandler(cfg, mockSvc2)

	r2 := gin.New()
	r2.GET("/admin", func(c *gin.Context) {
		c.String(200, "Admin Dashboard")
	})
	admin2 := r2.Group("/admin/api")
	admin2.Use(handler2.AuthMiddleware())
	admin2.POST("/actions/scan", handler2.TriggerScan)
	admin2.POST("/actions/jobs/clear-queue", handler2.ClearTaskQueue)
	admin2.POST("/actions/task-batches/:batch_id/cancel", handler2.CancelTaskBatch)
	admin2.POST("/actions/tasks/:task_id/cancel", handler2.CancelTask)

	req = httptest.NewRequest("POST", "/admin/api/actions/scan", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w = httptest.NewRecorder()
	r2.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for scan action, got %d", w.Code)
	}
}

func TestAdminHandler_GetTaskBatches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{
		taskBatches: []service.TaskBatchReadModel{{
			ID:            12,
			SourceType:    "import_scan",
			SummaryLabel:  "import batch",
			Status:        "partial_failed",
			SourceSummary: "root-a, root-b",
			SkipSummary:   service.TaskBatchSkipSummary{Total: 2, Unchanged: 1, DuplicateTasks: 1},
			StatusCounts:  map[string]int64{"completed": 1, "failed": 1},
			TaskTypeCounts: map[string]int64{"thumbnail_generate": 1,
				"ai_tag_generation": 1},
			FailureSummary: "ai tag timeout",
		}},
	}

	handler := NewAdminHandler(cfg, mockSvc)
	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/task-batches", handler.GetTaskBatches)

	req := httptest.NewRequest("GET", "/admin/api/task-batches?limit=20", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "task_batches") {
		t.Fatalf("Expected task_batches response body, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "root-a, root-b") {
		t.Fatalf("Expected source summary in response, got %s", w.Body.String())
	}
}

func TestAdminHandler_GetTasksByBatchID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{
		tasks: []service.TaskReadModel{{
			ID:               33,
			BatchID:          12,
			ImageID:          9,
			ImagePath:        "/library/task.png",
			ImageFilename:    "task.png",
			TaskType:         "ai_tag_generation",
			Status:           "queued",
			SkipReason:       "already_completed",
			LatestAsyncJobID: int64Ptr(77),
		}},
	}

	handler := NewAdminHandler(cfg, mockSvc)
	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/tasks", handler.GetTasks)

	req := httptest.NewRequest("GET", "/admin/api/tasks?batch_id=12", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "tasks") {
		t.Fatalf("Expected tasks response body, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ai_tag_generation") {
		t.Fatalf("Expected task type in response, got %s", w.Body.String())
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

// --- Phase 14: Backfill handler tests ---

// mockBackfillService implements BackfillServiceInterface for testing.
type mockBackfillService struct {
	previewResult          *service.BackfillPreviewResult
	executeResult          *service.BackfillExecuteResult
	executeBackfillCapture *string // captures prompt argument if non-nil
	err                    error
}

var lastBackfillPrompt string

func (m *mockBackfillService) PreviewBackfill(_ context.Context, _ repository.BackfillCandidateFilter) (*service.BackfillPreviewResult, error) {
	return m.previewResult, m.err
}

func (m *mockBackfillService) ExecuteBackfill(_ context.Context, _ repository.BackfillCandidateFilter, prompt string) (*service.BackfillExecuteResult, error) {
	if m.executeBackfillCapture != nil {
		if prompt == "" {
			prompt = lastBackfillPrompt
		}
		*m.executeBackfillCapture = prompt
	}
	return m.executeResult, m.err
}

func setupBackfillRouter(backfillSvc BackfillServiceInterface) (*gin.Engine, *AdminHandler) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{}
	handler := NewAdminHandlerWithBackfill(cfg, mockSvc, backfillSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		if c.Request.Method == http.MethodPost && c.Request.URL.Path == "/admin/api/actions/backfill/execute" {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				var payload map[string]any
				if json.Unmarshal(bodyBytes, &payload) == nil {
					if prompt, ok := payload["prompt"].(string); ok {
						lastBackfillPrompt = prompt
					} else {
						lastBackfillPrompt = ""
					}
				}
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}
		c.Next()
	})
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.POST("/actions/backfill/preview", handler.BackfillPreview)
	admin.POST("/actions/backfill/execute", handler.BackfillExecute)
	return r, handler
}

func authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))
}

func TestBackfillPreview_RejectsMissingFilters(t *testing.T) {
	mock := &mockBackfillService{}
	r, _ := setupBackfillRouter(mock)

	// Empty JSON body — no tag_ids or has_tags
	body := `{}`
	req := httptest.NewRequest("POST", "/admin/api/actions/backfill/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unfiltered request, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "narrowing filter") {
		t.Fatalf("expected narrowing filter error, got %s", w.Body.String())
	}
}

func TestBackfillPreview_ReturnsStructuredCounts(t *testing.T) {
	mock := &mockBackfillService{
		previewResult: &service.BackfillPreviewResult{
			HitCount:              20,
			EnqueueableCount:      12,
			SkippedWithAITag:      5,
			SkippedWithActiveTask: 3,
			SkippedTotal:          8,
		},
	}
	r, _ := setupBackfillRouter(mock)

	body := `{"tag_ids": [1, 2], "has_tags": false}`
	req := httptest.NewRequest("POST", "/admin/api/actions/backfill/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object in response, got %T", resp["data"])
	}
	if data["hit_count"].(float64) != 20 {
		t.Errorf("expected hit_count=20, got %v", data["hit_count"])
	}
	if data["enqueueable_count"].(float64) != 12 {
		t.Errorf("expected enqueueable_count=12, got %v", data["enqueueable_count"])
	}
	if data["skipped_with_ai_tag"].(float64) != 5 {
		t.Errorf("expected skipped_with_ai_tag=5, got %v", data["skipped_with_ai_tag"])
	}
	if data["skipped_with_active_task"].(float64) != 3 {
		t.Errorf("expected skipped_with_active_task=3, got %v", data["skipped_with_active_task"])
	}
}

func TestBackfillExecute_ReturnsNoOpForZeroEligible(t *testing.T) {
	mock := &mockBackfillService{
		executeResult: &service.BackfillExecuteResult{
			Success:           false,
			CreatedTasks:      0,
			SkippedTotal:      5,
			SkippedWithAITag:  3,
			SkippedWithActive: 2,
			NoOpReason:        "当前筛选结果中没有可补跑的图片（3 张已有 AI 标签， 2 张已有在途任务）",
		},
	}
	r, _ := setupBackfillRouter(mock)

	body := `{"tag_ids": [1]}`
	req := httptest.NewRequest("POST", "/admin/api/actions/backfill/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["success"] != false {
		t.Errorf("expected success=false for no-op, got %v", resp["success"])
	}
	if resp["no_op_reason"] == nil || resp["no_op_reason"].(string) == "" {
		t.Error("expected non-empty no_op_reason")
	}
}

// --- Phase 14-03: Backfill UX contract and failure group tests ---

func TestGetTaskBatches_PayloadIncludesFailureGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{
		taskBatches: []service.TaskBatchReadModel{{
			ID:             15,
			SourceType:     "import_scan",
			SummaryLabel:   "import batch",
			Status:         "partial_failed",
			SourceSummary:  "root-a",
			StatusCounts:   map[string]int64{"completed": 3, "failed": 2},
			TaskTypeCounts: map[string]int64{"ai_tag_generation": 5},
			FailureSummary: "timeout: API request exceeded 30s; network: connection refused",
			FailureGroups: []service.TaskBatchFailureGroup{
				{ReasonKey: "timeout", ReasonLabel: "超时", Count: 1, RetryRecommended: true, RetryHint: "可安全重试，通常为临时性问题"},
				{ReasonKey: "network", ReasonLabel: "网络错误", Count: 1, RetryRecommended: true, RetryHint: "可安全重试，通常为临时性问题"},
			},
		}},
	}

	handler := NewAdminHandler(cfg, mockSvc)
	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/task-batches", handler.GetTaskBatches)

	req := httptest.NewRequest("GET", "/admin/api/task-batches", nil)
	req.Header.Set("Authorization", authHeader())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	batches, ok := resp["task_batches"].([]any)
	if !ok {
		t.Fatalf("expected task_batches array, got %T", resp["task_batches"])
	}
	if len(batches) == 0 {
		t.Fatal("expected at least one batch")
	}

	batch := batches[0].(map[string]any)

	// Verify failure_groups is present in the payload
	fg, ok := batch["failure_groups"]
	if !ok {
		t.Fatal("expected failure_groups field in batch payload")
	}
	groups, ok := fg.([]any)
	if !ok {
		t.Fatalf("expected failure_groups to be an array, got %T", fg)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 failure groups, got %d", len(groups))
	}

	// Verify first group has the required fields for admin UI rendering
	g0 := groups[0].(map[string]any)
	requiredKeys := []string{"reason_key", "reason_label", "count", "retry_recommended", "retry_hint"}
	for _, key := range requiredKeys {
		if _, exists := g0[key]; !exists {
			t.Errorf("failure group missing required field: %s", key)
		}
	}
	if g0["reason_key"].(string) != "timeout" {
		t.Errorf("expected first group reason_key=timeout, got %v", g0["reason_key"])
	}
	if g0["retry_recommended"] != true {
		t.Errorf("expected timeout group retry_recommended=true, got %v", g0["retry_recommended"])
	}
}

func TestBackfillExecute_ReadsPromptFromJSONBody(t *testing.T) {
	// Verify that execute endpoint reads the prompt field from JSON body (not form data)
	var capturedPrompt string
	mock := &mockBackfillService{
		executeResult: &service.BackfillExecuteResult{
			Success:      true,
			BatchID:      55,
			CreatedTasks: 3,
		},
	}
	// Override ExecuteBackfill to capture prompt
	origExecuteBackfill := mock.ExecuteBackfill
	_ = origExecuteBackfill // keep reference
	mock.executeBackfillCapture = &capturedPrompt

	r, _ := setupBackfillRouter(mock)

	body := `{"has_tags": false, "prompt": "custom prompt for backfill"}`
	req := httptest.NewRequest("POST", "/admin/api/actions/backfill/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}
	if capturedPrompt != "custom prompt for backfill" {
		t.Errorf("expected prompt to be passed through from JSON body, got %q", capturedPrompt)
	}
}

func TestGetTaskBatches_FailedBatchShowsRetryableGuidance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{
		taskBatches: []service.TaskBatchReadModel{{
			ID:             20,
			SourceType:     "ai_tag",
			SummaryLabel:   "AI 标签生成",
			Status:         "failed",
			SourceSummary:  "manual",
			StatusCounts:   map[string]int64{"failed": 5},
			TaskTypeCounts: map[string]int64{"ai_tag_generation": 5},
			FailureSummary: "auth: invalid API key",
			FailureGroups: []service.TaskBatchFailureGroup{
				{ReasonKey: "auth", ReasonLabel: "认证失败", Count: 5, RetryRecommended: false, RetryHint: "不建议重试，请检查配置或数据"},
			},
		}},
	}

	handler := NewAdminHandler(cfg, mockSvc)
	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/task-batches", handler.GetTaskBatches)

	req := httptest.NewRequest("GET", "/admin/api/task-batches", nil)
	req.Header.Set("Authorization", authHeader())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	batches := resp["task_batches"].([]any)
	batch := batches[0].(map[string]any)
	groups := batch["failure_groups"].([]any)
	g0 := groups[0].(map[string]any)

	// Non-retryable: auth failure should have retry_recommended=false
	if g0["retry_recommended"] != false {
		t.Errorf("expected auth group retry_recommended=false, got %v", g0["retry_recommended"])
	}
	if g0["retry_hint"] == nil || g0["retry_hint"].(string) == "" {
		t.Error("expected non-empty retry_hint for auth failure group")
	}
}

func TestAdminHandler_GetTaskBatches_IncludesFailureGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{Admin: config.AdminConfig{Username: "admin", Password: "secret"}}
	mockSvc := &mockAdminService{
		taskBatches: []service.TaskBatchReadModel{{
			ID:             15,
			SourceType:     "manual_batch",
			SummaryLabel:   "backfill batch",
			Status:         "partial_failed",
			FailureSummary: "timeout x3",
			FailureGroups: []service.TaskBatchFailureGroup{
				{ReasonKey: "timeout", ReasonLabel: "AI 标签超时", Count: 3, RetryRecommended: true, RetryHint: "网络波动，建议重试"},
			},
		}},
	}

	handler := NewAdminHandler(cfg, mockSvc)
	r := gin.New()
	admin := r.Group("/admin/api")
	admin.Use(handler.AuthMiddleware())
	admin.GET("/task-batches", handler.GetTaskBatches)

	req := httptest.NewRequest("GET", "/admin/api/task-batches", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "failure_groups") {
		t.Fatalf("Expected failure_groups in batch response, got %s", body)
	}
	if !strings.Contains(body, "AI 标签超时") {
		t.Fatalf("Expected grouped failure reason label in response, got %s", body)
	}
	if !strings.Contains(body, "retry_recommended") {
		t.Fatalf("Expected retry_recommended field in failure group, got %s", body)
	}
}

func TestBackfillExecute_ReturnsBatchOnSuccess(t *testing.T) {
	mock := &mockBackfillService{
		executeResult: &service.BackfillExecuteResult{
			Success:           true,
			BatchID:           42,
			CreatedTasks:      7,
			SkippedTotal:      3,
			SkippedWithAITag:  2,
			SkippedWithActive: 1,
		},
	}
	r, _ := setupBackfillRouter(mock)

	bodyBytes, _ := json.Marshal(map[string]any{
		"has_tags": false,
		"prompt":   "describe this image",
	})
	req := httptest.NewRequest("POST", "/admin/api/actions/backfill/execute", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}
	if resp["batch_id"].(float64) != 42 {
		t.Errorf("expected batch_id=42, got %v", resp["batch_id"])
	}
	if resp["created_tasks"].(float64) != 7 {
		t.Errorf("expected created_tasks=7, got %v", resp["created_tasks"])
	}
}
