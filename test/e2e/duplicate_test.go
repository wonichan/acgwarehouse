package e2e

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/handler"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
)

func setupDuplicateTestServer(t *testing.T) (*gin.Engine, *sql.DB) {
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	mockSidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/duplicates/detect":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-e2e", "status": "pending", "progress": 0})
		case "/compute/duplicates/tasks/task-e2e":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-e2e", "status": "completed", "progress": 100})
		case "/compute/duplicates/tasks/task-e2e/result":
			_ = json.NewEncoder(w).Encode(map[string]any{"groups": []any{}, "total_images": 3, "total_groups": 0, "skipped_images": []any{}, "computation_time_ms": 1})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(mockSidecar.Close)
	duplicateSvc := service.NewDuplicateService(imageRepo, duplicateRepo, sidecar.NewSidecarClient(mockSidecar.URL), nil)

	r := gin.New()
	api := r.Group("/api/v1")

	duplicateHandler := handler.NewDuplicateHandler(duplicateSvc, nil)
	api.POST("/duplicates/detect", duplicateHandler.DetectDuplicates)
	api.GET("/duplicates/tasks/:task_id", duplicateHandler.GetDuplicateTaskStatus)
	api.GET("/duplicates/tasks/:task_id/events", duplicateHandler.StreamDuplicateTaskEvents)
	api.GET("/duplicates", duplicateHandler.ListDuplicates)
	api.GET("/duplicates/:id", duplicateHandler.GetDuplicate)
	api.DELETE("/duplicates/:id", duplicateHandler.DeleteDuplicate)

	return r, db
}

func TestE2E_DuplicateDetection(t *testing.T) {
	r, db := setupDuplicateTestServer(t)
	defer db.Close()

	// Create test images with same hash (simulating duplicates)
	for i := 0; i < 3; i++ {
		_, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, phash, created_at, updated_at)
			VALUES (?, ?, '', 1000, 100, 100, 'jpg', 123456789, datetime('now'), datetime('now'))
		`, "/images/test"+string(rune('0'+i))+".jpg", "test"+string(rune('0'+i))+".jpg")
		if err != nil {
			t.Fatalf("Failed to create test image: %v", err)
		}
	}

	// Test 1: Trigger duplicate detection
	t.Run("DetectDuplicates", func(t *testing.T) {
		body := bytes.NewBufferString(`{"threshold": 10}`)
		req := httptest.NewRequest("POST", "/api/v1/duplicates/detect", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		if resp["task_id"] == nil {
			t.Error("Expected task_id in response")
		}

		taskID, _ := resp["task_id"].(string)
		if taskID == "" {
			t.Fatal("empty task_id")
		}

		statusReq := httptest.NewRequest("GET", "/api/v1/duplicates/tasks/"+taskID, nil)
		statusW := httptest.NewRecorder()
		r.ServeHTTP(statusW, statusReq)
		if statusW.Code != http.StatusOK {
			t.Errorf("Expected status endpoint 200, got %d: %s", statusW.Code, statusW.Body.String())
		}

		waitForDuplicateTaskTerminalE2E(t, r, taskID)
	})

	// Test 2: List duplicate groups
	t.Run("ListDuplicates", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/duplicates", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}

		groups, ok := resp["groups"].([]interface{})
		if !ok {
			t.Error("Expected groups array in response")
		}

		if len(groups) == 0 {
			t.Log("No duplicate groups found (expected if images have different hashes)")
		}
	})

	// Test 3: Get duplicate group detail
	t.Run("GetDuplicate", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/duplicates/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// May return 404 if no groups exist
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Test 4: Delete duplicate group
	t.Run("DeleteDuplicate", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/duplicates/999", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// May return 404 if group doesn't exist
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func waitForDuplicateTaskTerminalE2E(t *testing.T, router *gin.Engine, taskID string) {
	t.Helper()

	const maxAttempts = 40
	for i := 0; i < maxAttempts; i++ {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/duplicates/tasks/%s", taskID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("task status = %d, want 200, body=%s", w.Code, w.Body.String())
		}

		var payload map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal task status: %v", err)
		}
		status, _ := payload["status"].(string)
		if status == "completed" {
			return
		}
		if status == "failed" {
			t.Fatalf("duplicate task failed: %v", payload)
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("duplicate task did not reach terminal status: task_id=%s", taskID)
}

func TestE2E_DuplicateThreshold(t *testing.T) {
	r, db := setupDuplicateTestServer(t)
	defer db.Close()

	// Test threshold validation
	t.Run("InvalidThreshold", func(t *testing.T) {
		body := bytes.NewBufferString(`{"threshold": -1}`)
		req := httptest.NewRequest("POST", "/api/v1/duplicates/detect", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Should handle invalid threshold
		if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
			t.Errorf("Unexpected status: %d", w.Code)
		}
	})

	t.Run("ValidThreshold", func(t *testing.T) {
		body := bytes.NewBufferString(`{"threshold": 5}`)
		req := httptest.NewRequest("POST", "/api/v1/duplicates/detect", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}
