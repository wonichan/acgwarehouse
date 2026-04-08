package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	r.GET("/api/v1/duplicates/tasks/:task_id", h.GetDuplicateTaskStatus)
	r.GET("/api/v1/duplicates/tasks/:task_id/events", h.StreamDuplicateTaskEvents)
	r.GET("/api/v1/duplicates", h.ListDuplicates)
	r.GET("/api/v1/duplicates/:id", h.GetDuplicate)
	r.DELETE("/api/v1/duplicates/:id", h.DeleteDuplicate)

	return r, db
}

func createAsyncMockSidecar(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/duplicates/detect":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-handler", "status": "pending", "progress": 0})
		case "/compute/duplicates/tasks/task-handler":
			_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "task-handler", "status": "completed", "progress": 100, "message": "completed"})
		case "/compute/duplicates/tasks/task-handler/result":
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
}

func TestDuplicateHandler_DetectDuplicates_SidecarUnavailable(t *testing.T) {
	mockSidecar := createAsyncMockSidecar(t)
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
}

func TestDuplicateHandler_DetectDuplicates_ReturnsTaskImmediately(t *testing.T) {
	mockSidecar := createAsyncMockSidecar(t)
	defer mockSidecar.Close()

	readyRuntime := createReadyRuntime(t)
	r, _ := setupDuplicateHandlerTest(t, readyRuntime, mockSidecar.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", bytes.NewBufferString(`{"threshold":40}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["task_id"] == "" {
		t.Fatalf("task_id empty in payload: %v", payload)
	}
	if payload["status"] != "queued" {
		t.Fatalf("status = %v, want queued", payload["status"])
	}
}

func TestDuplicateHandler_GetDuplicateTaskStatus_AndNotFound(t *testing.T) {
	mockSidecar := createAsyncMockSidecar(t)
	defer mockSidecar.Close()

	readyRuntime := createReadyRuntime(t)
	r, _ := setupDuplicateHandlerTest(t, readyRuntime, mockSidecar.URL)

	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", bytes.NewBufferString(`{"threshold":40}`))
	reqCreate.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wCreate, reqCreate)

	var createPayload map[string]any
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createPayload)
	taskID, _ := createPayload["task_id"].(string)

	wStatus := httptest.NewRecorder()
	r.ServeHTTP(wStatus, httptest.NewRequest(http.MethodGet, "/api/v1/duplicates/tasks/"+taskID, nil))
	if wStatus.Code != http.StatusOK {
		t.Fatalf("status endpoint = %d, want 200, body=%s", wStatus.Code, wStatus.Body.String())
	}

	wNotFound := httptest.NewRecorder()
	r.ServeHTTP(wNotFound, httptest.NewRequest(http.MethodGet, "/api/v1/duplicates/tasks/missing-task", nil))
	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("missing task status = %d, want 404", wNotFound.Code)
	}
}

func TestDuplicateHandler_StreamDuplicateTaskEvents(t *testing.T) {
	mockSidecar := createAsyncMockSidecar(t)
	defer mockSidecar.Close()

	readyRuntime := createReadyRuntime(t)
	r, _ := setupDuplicateHandlerTest(t, readyRuntime, mockSidecar.URL)

	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/duplicates/detect", bytes.NewBufferString(`{"threshold":40}`))
	reqCreate.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wCreate, reqCreate)

	var createPayload map[string]any
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createPayload)
	taskID, _ := createPayload["task_id"].(string)

	server := httptest.NewServer(r)
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(server.URL + "/api/v1/duplicates/tasks/" + taskID + "/events")
	if err != nil {
		t.Fatalf("open SSE stream: %v", err)
	}
	defer resp.Body.Close()

	if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", resp.Header.Get("Content-Type"))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		t.Fatalf("read sse body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "event: progress") {
		t.Fatalf("SSE body missing progress event: %s", text)
	}
	if !strings.Contains(text, "event: terminal") {
		t.Fatalf("SSE body missing terminal event: %s", text)
	}
}

func TestDuplicateHandler_StreamDuplicateTaskEvents_NotFound(t *testing.T) {
	mockSidecar := createAsyncMockSidecar(t)
	defer mockSidecar.Close()

	readyRuntime := createReadyRuntime(t)
	r, _ := setupDuplicateHandlerTest(t, readyRuntime, mockSidecar.URL)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/duplicates/tasks/missing/events", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDuplicateHandler_ListGetDeleteStillWork(t *testing.T) {
	mockSidecar := createAsyncMockSidecar(t)
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

	time.Sleep(300 * time.Millisecond)

	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, httptest.NewRequest(http.MethodGet, "/api/v1/duplicates", nil))
	if wList.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", wList.Code, wList.Body.String())
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
