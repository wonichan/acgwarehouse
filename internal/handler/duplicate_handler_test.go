package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

func setupDuplicateHandlerTest(t *testing.T) (*gin.Engine, *DuplicateHandler, *sql.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// 创建临时数据库
	tmpFile, err := os.CreateTemp("", "duplicate_handler_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpPath) })

	db, err := sql.Open("sqlite3", tmpPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// 创建服务
	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	hashService := service.NewHashService()
	duplicateService := service.NewDuplicateService(imageRepo, duplicateRepo, hashService)

	// 创建 handler
	handler := NewDuplicateHandler(duplicateService)

	// 创建路由
	r := gin.New()
	return r, handler, db
}

func insertTestImagesForHandler(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now()

	images := []struct {
		path  string
		pHash int64
	}{
		{"/test/a.jpg", 0x1234567890ABCDEF},
		{"/test/b.jpg", 0x1234567890ABCDEF},
		{"/test/c.jpg", 0x1234567890ABCDEF + 1},
	}

	for _, img := range images {
		_, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, phash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			img.path,
			img.path[len(img.path)-5:],
			"/test",
			1024,
			100,
			100,
			"jpg",
			img.pHash,
			now,
			now,
		)
		if err != nil {
			t.Fatalf("Failed to insert test image: %v", err)
		}
	}
}

func TestDuplicateHandler_DetectDuplicates(t *testing.T) {
	r, handler, db := setupDuplicateHandlerTest(t)
	insertTestImagesForHandler(t, db)

	// 注册路由
	r.POST("/api/v1/duplicates/detect", handler.DetectDuplicates)

	// Test 1: POST /api/v1/duplicates/detect 触发检测并返回检测状态
	body := bytes.NewBufferString(`{"threshold": 10}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DetectResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Message != "Detection completed" {
		t.Errorf("Expected message 'Detection completed', got '%s'", resp.Message)
	}

	t.Logf("Detection response: %+v", resp)
}

func TestDuplicateHandler_DetectDuplicates_DefaultThreshold(t *testing.T) {
	r, handler, db := setupDuplicateHandlerTest(t)
	insertTestImagesForHandler(t, db)

	r.POST("/api/v1/duplicates/detect", handler.DetectDuplicates)

	// 不发送请求体，应该使用默认阈值
	req := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDuplicateHandler_ListDuplicates(t *testing.T) {
	r, handler, db := setupDuplicateHandlerTest(t)
	insertTestImagesForHandler(t, db)

	// 先执行检测
	duplicateRepo := repository.NewDuplicateRepository(db)
	imageRepo := repository.NewImageRepository(db)
	hashService := service.NewHashService()
	duplicateService := service.NewDuplicateService(imageRepo, duplicateRepo, hashService)
	_, err := duplicateService.DetectDuplicates(nil, service.DetectOptions{Threshold: 10})
	if err != nil {
		t.Fatalf("DetectDuplicates failed: %v", err)
	}

	// 注册路由
	r.GET("/api/v1/duplicates", handler.ListDuplicates)

	// Test 2: GET /api/v1/duplicates 返回重复组列表
	req := httptest.NewRequest(http.MethodGet, "/api/v1/duplicates?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	t.Logf("List response: total=%d, groups=%d", resp.Total, len(resp.Groups))
}

func TestDuplicateHandler_GetDuplicate(t *testing.T) {
	r, handler, db := setupDuplicateHandlerTest(t)
	insertTestImagesForHandler(t, db)

	// 先执行检测
	duplicateRepo := repository.NewDuplicateRepository(db)
	imageRepo := repository.NewImageRepository(db)
	hashService := service.NewHashService()
	duplicateService := service.NewDuplicateService(imageRepo, duplicateRepo, hashService)
	_, err := duplicateService.DetectDuplicates(nil, service.DetectOptions{Threshold: 10})
	if err != nil {
		t.Fatalf("DetectDuplicates failed: %v", err)
	}

	// 获取组 ID
	groups, _, err := duplicateService.GetDuplicateGroups(10, 0)
	if err != nil || len(groups) == 0 {
		t.Fatalf("No groups found: %v", err)
	}
	groupID := groups[0].Group.ID

	// 注册路由
	r.GET("/api/v1/duplicates/:id", handler.GetDuplicate)

	// Test 3: GET /api/v1/duplicates/:id 返回单个重复组详情
	req := httptest.NewRequest(http.MethodGet, "/api/v1/duplicates/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.DuplicateGroupWithImages
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	t.Logf("Get response: group_id=%d, images=%d", resp.Group.ID, len(resp.Images))

	_ = groupID
}

func TestDuplicateHandler_GetDuplicate_NotFound(t *testing.T) {
	r, handler, _ := setupDuplicateHandlerTest(t)

	r.GET("/api/v1/duplicates/:id", handler.GetDuplicate)

	// 请求不存在的组
	req := httptest.NewRequest(http.MethodGet, "/api/v1/duplicates/99999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDuplicateHandler_DeleteDuplicate(t *testing.T) {
	r, handler, db := setupDuplicateHandlerTest(t)
	insertTestImagesForHandler(t, db)

	// 先执行检测
	duplicateRepo := repository.NewDuplicateRepository(db)
	imageRepo := repository.NewImageRepository(db)
	hashService := service.NewHashService()
	duplicateService := service.NewDuplicateService(imageRepo, duplicateRepo, hashService)
	_, err := duplicateService.DetectDuplicates(nil, service.DetectOptions{Threshold: 10})
	if err != nil {
		t.Fatalf("DetectDuplicates failed: %v", err)
	}

	// 获取组 ID
	groups, _, err := duplicateService.GetDuplicateGroups(10, 0)
	if err != nil || len(groups) == 0 {
		t.Fatalf("No groups found: %v", err)
	}
	groupID := groups[0].Group.ID

	// 注册路由
	r.DELETE("/api/v1/duplicates/:id", handler.DeleteDuplicate)

	// Test 4: DELETE /api/v1/duplicates/:id 删除重复组记录
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/duplicates/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// 验证已删除
	_, err = duplicateService.GetDuplicateGroup(groupID)
	if err == nil {
		t.Error("Expected error when getting deleted group")
	}
}

func TestDuplicateHandler_ThresholdValidation(t *testing.T) {
	r, handler, db := setupDuplicateHandlerTest(t)
	insertTestImagesForHandler(t, db)

	r.POST("/api/v1/duplicates/detect", handler.DetectDuplicates)

	// Test 5: 查询参数 threshold 可设置检测阈值
	tests := []struct {
		threshold int
		expected  int
	}{
		{0, http.StatusOK},   // 使用默认值
		{10, http.StatusOK},  // 正常值
		{64, http.StatusOK},  // 最大值
		{100, http.StatusOK}, // 超过最大值会被截断到 64
	}

	for _, tt := range tests {
		body := bytes.NewBufferString(`{"threshold": ` + string(rune('0'+tt.threshold)) + `}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != tt.expected {
			t.Errorf("Threshold %d: expected status %d, got %d", tt.threshold, tt.expected, w.Code)
		}
	}
}
