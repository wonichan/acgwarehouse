package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

func TestAITagTriggerQueuesJob(t *testing.T) {
	t.Parallel()

	router, _ := newAITagHandlerTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/1/ai-tags", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp struct {
		JobID  int64  `json:"job_id"`
		Status string `json:"status"`
	}
	decodeAIJSONBody(t, w.Body.Bytes(), &resp)

	if resp.JobID == 0 {
		t.Fatal("expected job id to be assigned")
	}
	if resp.Status != "queued" {
		t.Fatalf("status = %q, want queued", resp.Status)
	}
}

func TestAITagGetStatusReturnsCurrentJobState(t *testing.T) {
	t.Parallel()

	router, repos := newAITagHandlerTestRouter(t)
	jobID, err := repos.manager.AddJob(t.Context(), "ai_tag_generation", `{"image_id":1,"path":"/images/1.png"}`)
	if err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}
	job, err := repos.jobRepo.FindByID(jobID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	job.Status = "running"
	job.Progress = 0.5
	if err := repos.jobRepo.Update(job); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/1/ai-tags/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		JobID    int64   `json:"job_id"`
		Status   string  `json:"status"`
		Progress float64 `json:"progress"`
	}
	decodeAIJSONBody(t, w.Body.Bytes(), &resp)

	if resp.JobID != jobID {
		t.Fatalf("job_id = %d, want %d", resp.JobID, jobID)
	}
	if resp.Status != "running" {
		t.Fatalf("status = %q, want running", resp.Status)
	}
	if resp.Progress != 0.5 {
		t.Fatalf("progress = %f, want 0.5", resp.Progress)
	}
}

type aiTagHandlerTestRepos struct {
	jobRepo repository.JobRepository
	manager *worker.Manager
	db      *sql.DB
}

func newAITagHandlerTestRouter(t *testing.T) (*gin.Engine, *aiTagHandlerTestRepos) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "ai-tag-handler.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES (1, '/images/1.png', '1.png', '/images', 100, 100, 100, 'png', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("seed image: %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	manager := worker.NewManager(jobRepo)
	imageRepo := repository.NewImageRepository(db)
	h := NewAITagHandler(manager, imageRepo, jobRepo)

	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/images/:id/ai-tags", h.TriggerAITags)
	api.GET("/images/:id/ai-tags/status", h.GetAITagStatus)
	api.POST("/images/batch-ai-tags", h.BatchTriggerAITags)

	return router, &aiTagHandlerTestRepos{jobRepo: jobRepo, manager: manager, db: db}
}

func decodeAIJSONBody(t *testing.T, body []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, string(body))
	}
}
