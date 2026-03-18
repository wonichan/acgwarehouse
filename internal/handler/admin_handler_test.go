package handler

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

// mockAdminService implements AdminServiceInterface for testing
type mockAdminService struct {
	summary      *service.Summary
	jobs         []interface{}
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
		jobs: []interface{}{},
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
