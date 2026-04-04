package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
)

type fakeProcess struct{}

func (fakeProcess) Kill() error { return nil }
func (fakeProcess) Wait() error { return nil }

func createReadyRuntime(t *testing.T) *sidecar.Runtime {
	t.Helper()
	runtime := sidecar.NewRuntime(sidecar.RuntimeConfig{
		StartupTimeout: 200 * time.Millisecond,
		ProbeInterval:  10 * time.Millisecond,
		CommandFactory: func(context.Context) (sidecar.Process, error) {
			return fakeProcess{}, nil
		},
		Probe: func(context.Context) error { return nil },
		ShutdownProbe: func(context.Context) error {
			return nil
		},
	})
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("runtime.Start() error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Stop(context.Background()) })
	return runtime
}

func setupDuplicateHandlerTest(t *testing.T, runtime *sidecar.Runtime, sidecarServerURL string) (*gin.Engine, *sql.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tmpFile, err := os.CreateTemp("", "duplicate_handler_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, err := sql.Open("sqlite3", tmpPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	now := time.Now()
	for i := 0; i < 3; i++ {
		_, err := db.Exec(`
			INSERT INTO images (path, filename, source_root, file_size, width, height, format, phash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			"/test/img"+string(rune('0'+i))+".jpg",
			"img.jpg",
			"/test",
			1024+int64(i),
			100+i,
			120+i,
			"jpg",
			int64(0),
			now,
			now,
		)
		if err != nil {
			t.Fatalf("insert image: %v", err)
		}
	}

	imageRepo := repository.NewImageRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	duplicateSvc := service.NewDuplicateService(imageRepo, duplicateRepo, sidecar.NewSidecarClient(sidecarServerURL), runtime)
	h := NewDuplicateHandler(duplicateSvc, runtime)

	r := gin.New()
	r.POST("/api/v1/duplicates/detect", h.DetectDuplicates)
	r.GET("/api/v1/duplicates", h.ListDuplicates)
	r.GET("/api/v1/duplicates/:id", h.GetDuplicate)
	r.DELETE("/api/v1/duplicates/:id", h.DeleteDuplicate)

	return r, db
}

func TestDuplicateHandler_DetectDuplicates_SidecarUnavailable(t *testing.T) {
	mockSidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "never-used", "status": "pending"})
	}))
	defer mockSidecar.Close()

	notReadyRuntime := sidecar.NewRuntime(sidecar.RuntimeConfig{})
	r, _ := setupDuplicateHandlerTest(t, notReadyRuntime, mockSidecar.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", bytes.NewBufferString(`{"threshold":40}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503, body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("计算服务不可用")) {
		t.Fatalf("body = %s, want contains 计算服务不可用", w.Body.String())
	}
}

func TestDuplicateHandler_DetectDuplicates_ReadyWithThresholdRules(t *testing.T) {
	var capturedThreshold int
	mockSidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/duplicates/detect":
			var req struct {
				Threshold int `json:"threshold"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			capturedThreshold = req.Threshold
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-handler", "status": "pending", "progress": 0})
		case "/compute/duplicates/tasks/task-handler":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-handler", "status": "completed", "progress": 100})
		case "/compute/duplicates/tasks/task-handler/result":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"groups": []map[string]any{{
					"group_id":       0,
					"recommended_id": 1,
					"members": []map[string]any{{
						"image_id":               1,
						"sha256":                 "sha",
						"phash":                  "phashhex",
						"distance":               0,
						"is_recommended":         true,
						"recommendation_score":   88.8,
						"recommendation_reasons": []map[string]any{{"factor": "resolution", "value": "100x120", "score": 10.0, "weight": 0.5}},
					}},
				}},
				"total_images":        3,
				"total_groups":        1,
				"skipped_images":      []any{},
				"computation_time_ms": 10,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockSidecar.Close()

	readyRuntime := createReadyRuntime(t)
	r, _ := setupDuplicateHandlerTest(t, readyRuntime, mockSidecar.URL)

	// default threshold -> 40
	req := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if capturedThreshold != 40 {
		t.Fatalf("default threshold = %d, want 40", capturedThreshold)
	}

	// max threshold -> 256
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", bytes.NewBufferString(`{"threshold":999}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w2.Code, w2.Body.String())
	}
	if capturedThreshold != 256 {
		t.Fatalf("max threshold = %d, want 256", capturedThreshold)
	}
}

func TestDuplicateHandler_ListGetDeleteAndStructuredRationale(t *testing.T) {
	mockSidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/duplicates/detect":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-list", "status": "pending", "progress": 0})
		case "/compute/duplicates/tasks/task-list":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-list", "status": "completed", "progress": 100})
		case "/compute/duplicates/tasks/task-list/result":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"groups": []map[string]any{{
					"group_id":       0,
					"recommended_id": 1,
					"members": []map[string]any{
						{"image_id": 1, "sha256": "sha1", "phash": "phash1", "distance": 0, "is_recommended": true, "recommendation_score": 90.0, "recommendation_reasons": []map[string]any{{"factor": "resolution", "value": "100x120", "score": 10.0, "weight": 0.5}}},
						{"image_id": 2, "sha256": "sha2", "phash": "phash2", "distance": 3, "is_recommended": false, "recommendation_score": 70.0, "recommendation_reasons": []map[string]any{{"factor": "size", "value": "1025", "score": 8.0, "weight": 0.3}}},
					},
				}},
				"total_images":        3,
				"total_groups":        1,
				"skipped_images":      []any{},
				"computation_time_ms": 10,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockSidecar.Close()

	readyRuntime := createReadyRuntime(t)
	r, _ := setupDuplicateHandlerTest(t, readyRuntime, mockSidecar.URL)

	wDetect := httptest.NewRecorder()
	reqDetect := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", bytes.NewBufferString(`{"threshold":40}`))
	reqDetect.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wDetect, reqDetect)
	if wDetect.Code != http.StatusOK {
		t.Fatalf("detect status = %d, body=%s", wDetect.Code, wDetect.Body.String())
	}

	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, httptest.NewRequest(http.MethodGet, "/api/v1/duplicates", nil))
	if wList.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", wList.Code, wList.Body.String())
	}
	var listResp map[string]any
	if err := json.Unmarshal(wList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	groups := listResp["groups"].([]any)
	firstGroup := groups[0].(map[string]any)
	firstImage := firstGroup["images"].([]any)[0].(map[string]any)
	if _, ok := firstImage["recommendation_rationale"].([]any); !ok {
		t.Fatalf("recommendation_rationale should be JSON array, got %#v", firstImage["recommendation_rationale"])
	}

	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, httptest.NewRequest(http.MethodGet, "/api/v1/duplicates/1", nil))
	if wGet.Code != http.StatusOK {
		t.Fatalf("get status = %d, body=%s", wGet.Code, wGet.Body.String())
	}

	wDelete := httptest.NewRecorder()
	r.ServeHTTP(wDelete, httptest.NewRequest(http.MethodDelete, "/api/v1/duplicates/1", nil))
	if wDelete.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body=%s", wDelete.Code, wDelete.Body.String())
	}
}
