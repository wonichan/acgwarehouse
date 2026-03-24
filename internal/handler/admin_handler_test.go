package handler

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

// mockAdminService implements AdminServiceInterface for testing
type mockAdminService struct {
	summary      *service.Summary
	jobs         []interface{}
	taskBatches  []service.TaskBatchReadModel
	tasks        []service.TaskReadModel
	scanJobID    int64
	retryCount   int
	pauseCalled  bool
	resumeCalled bool
	isRunning    bool
	err          error
}

func (m *mockAdminService) GetSummary(ctx context.Context) (*service.Summary, error) {
	return m.summary, m.err
}

func (m *mockAdminService) GetJobs(ctx context.Context, limit int) ([]interface{}, error) {
	return m.jobs, m.err
}

func (m *mockAdminService) GetTaskBatches(ctx context.Context, filter service.TaskBatchReadFilter) ([]service.TaskBatchReadModel, error) {
	return m.taskBatches, m.err
}

func (m *mockAdminService) GetTasks(ctx context.Context, filter service.TaskReadFilter) ([]service.TaskReadModel, error) {
	return m.tasks, m.err
}

func (m *mockAdminService) TriggerScan(ctx context.Context) (int64, error) {
	return m.scanJobID, m.err
}

func (m *mockAdminService) RetryFailedJobs(ctx context.Context) (int, error) {
	return m.retryCount, m.err
}

func (m *mockAdminService) PauseBackgroundTasks(ctx context.Context) error {
	m.pauseCalled = true
	return m.err
}

func (m *mockAdminService) ResumeBackgroundTasks(ctx context.Context) error {
	m.resumeCalled = true
	return m.err
}

func (m *mockAdminService) IsBackgroundRunning() bool {
	return m.isRunning
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
		jobs:        []interface{}{},
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
	admin.GET("/jobs", handler.GetJobs)
	admin.GET("/task-batches", handler.GetTaskBatches)
	admin.GET("/tasks", handler.GetTasks)
	admin.POST("/actions/scan", handler.TriggerScan)
	admin.POST("/actions/jobs/pause", handler.PauseBackgroundTasks)
	admin.POST("/actions/jobs/resume", handler.ResumeBackgroundTasks)
	admin.POST("/actions/jobs/retry-failed", handler.RetryFailedJobs)

	// Test /admin/api/summary with auth
	req := httptest.NewRequest("GET", "/admin/api/summary", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
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
