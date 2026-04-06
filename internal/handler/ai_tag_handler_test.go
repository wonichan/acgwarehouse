package handler

import (
	"bytes"
	"context"
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
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

func TestAITagTriggerCreatesManualSingleBatch(t *testing.T) {
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
		BatchID        int64   `json:"batch_id"`
		SourceType     string  `json:"source_type"`
		Status         string  `json:"status"`
		CreatedTasks   int     `json:"created_tasks"`
		SkippedTasks   int     `json:"skipped_tasks"`
		PlatformTaskID []int64 `json:"platform_task_ids"`
		JobIDs         []int64 `json:"job_ids"`
	}
	decodeAIJSONBody(t, w.Body.Bytes(), &resp)

	if resp.BatchID == 0 {
		t.Fatal("expected batch id to be assigned")
	}
	if resp.SourceType != domain.TaskBatchSourceManualSingle {
		t.Fatalf("source_type = %q, want %q", resp.SourceType, domain.TaskBatchSourceManualSingle)
	}
	if resp.Status != "queued" {
		t.Fatalf("status = %q, want queued", resp.Status)
	}
	if resp.CreatedTasks != 1 || resp.SkippedTasks != 0 {
		t.Fatalf("created/skipped = %d/%d, want 1/0", resp.CreatedTasks, resp.SkippedTasks)
	}
	if len(resp.PlatformTaskID) != 1 || resp.PlatformTaskID[0] == 0 {
		t.Fatalf("platform_task_ids = %+v, want one task id", resp.PlatformTaskID)
	}
	if len(resp.JobIDs) != 1 || resp.JobIDs[0] == 0 {
		t.Fatalf("job_ids = %+v, want one async job", resp.JobIDs)
	}
}

func TestAITagTriggerUsesLargeThumbnailURLWhenAvailable(t *testing.T) {
	t.Parallel()

	router, repos := newAITagHandlerTestRouter(t)
	if _, err := repos.db.Exec(`UPDATE images SET thumbnail_large_url = ? WHERE id = 1`, "https://cos.local/thumbnails/1-large.jpg"); err != nil {
		t.Fatalf("seed large thumbnail url: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/1/ai-tags", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp struct {
		JobIDs []int64 `json:"job_ids"`
	}
	decodeAIJSONBody(t, w.Body.Bytes(), &resp)
	if len(resp.JobIDs) != 1 {
		t.Fatalf("job_ids = %+v, want one job id", resp.JobIDs)
	}

	job, err := repos.jobRepo.FindByID(resp.JobIDs[0])
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	var payload worker.AITagPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if payload.Path != "https://cos.local/thumbnails/1-large.jpg" {
		t.Fatalf("payload.Path = %q, want large thumbnail url", payload.Path)
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
	job.Progress = 50
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
	if resp.Progress != 50 {
		t.Fatalf("progress = %f, want 50", resp.Progress)
	}
}

func TestBatchAITagTriggerCreatesManualBatchAndSkipsDuplicateQueue(t *testing.T) {
	t.Parallel()

	router, repos := newAITagHandlerTestRouter(t)
	ctx := context.Background()
	image2 := repos.mustFindImage(t, 2)

	seed, err := repos.taskPlatformSvc.PlanBatch(ctx, service.TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceManualSingle,
		SummaryLabel: "seed duplicate ai task",
		TaskTypes:    []string{domain.PlatformTaskTypeAITagGeneration},
		Items: []service.TaskPlatformPlanItem{{
			ImageID:          1,
			ImageVersionKey:  service.BuildImageVersionKey(repos.mustFindImage(t, 1)),
			SourceDescriptor: "/images/1.png",
		}},
	})
	if err != nil {
		t.Fatalf("PlanBatch(seed) error = %v", err)
	}
	if len(seed.CreatedTasks) != 1 {
		t.Fatalf("seed created tasks = %d, want 1", len(seed.CreatedTasks))
	}
	payload, err := json.Marshal(worker.AITagPayload{ImageID: 1, Path: "/images/1.png"})
	if err != nil {
		t.Fatalf("json.Marshal(seed payload) error = %v", err)
	}
	if _, err := repos.taskPlatformSvc.QueueTask(ctx, &seed.CreatedTasks[0], domain.PlatformTaskTypeAITagGeneration, string(payload)); err != nil {
		t.Fatalf("QueueTask(seed) error = %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"image_ids": []int64{1, image2.ID},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/batch-ai-tags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp struct {
		BatchID        int64   `json:"batch_id"`
		SourceType     string  `json:"source_type"`
		CreatedTasks   int     `json:"created_tasks"`
		SkippedTasks   int     `json:"skipped_tasks"`
		PlatformTaskID []int64 `json:"platform_task_ids"`
		JobIDs         []int64 `json:"job_ids"`
	}
	decodeAIJSONBody(t, w.Body.Bytes(), &resp)

	if resp.BatchID == 0 {
		t.Fatal("expected batch id to be assigned")
	}
	if resp.SourceType != domain.TaskBatchSourceManualBatch {
		t.Fatalf("source_type = %q, want %q", resp.SourceType, domain.TaskBatchSourceManualBatch)
	}
	if resp.CreatedTasks != 1 || resp.SkippedTasks != 1 {
		t.Fatalf("created/skipped = %d/%d, want 1/1", resp.CreatedTasks, resp.SkippedTasks)
	}
	if len(resp.PlatformTaskID) != 1 || len(resp.JobIDs) != 1 {
		t.Fatalf("platform_task_ids/job_ids = %+v/%+v, want one new queued task", resp.PlatformTaskID, resp.JobIDs)
	}
}

func TestBatchAITagTriggerSupportsTagFilterWithoutImageIDs(t *testing.T) {
	t.Parallel()

	router, repos := newAITagHandlerTestRouter(t)
	now := time.Now()
	if _, err := repos.db.Exec(`
		INSERT INTO tags (id, preferred_label, slug, review_state, trust_score, usage_count, created_at)
		VALUES (10, 'heroine', 'heroine', 'confirmed', 1.0, 0, ?)
	`, now); err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	if _, err := repos.db.Exec(`
		INSERT INTO image_tags (image_id, tag_id, source, confidence, review_state)
		VALUES (2, 10, 'manual', 1.0, 'confirmed')
	`); err != nil {
		t.Fatalf("seed image tag: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"tag_ids":  []int64{10},
		"sort_by":  "id",
		"sort_dir": "asc",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/batch-ai-tags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp struct {
		CreatedTasks int     `json:"created_tasks"`
		SkippedTasks int     `json:"skipped_tasks"`
		JobIDs       []int64 `json:"job_ids"`
	}
	decodeAIJSONBody(t, w.Body.Bytes(), &resp)

	if resp.CreatedTasks != 1 || resp.SkippedTasks != 0 {
		t.Fatalf("created/skipped = %d/%d, want 1/0", resp.CreatedTasks, resp.SkippedTasks)
	}
	if len(resp.JobIDs) != 1 {
		t.Fatalf("job_ids = %+v, want one queued job", resp.JobIDs)
	}
}

func TestAITagTriggerDoesNotLoadJobImmediatelyWhenAIQueueLimitReached(t *testing.T) {
	t.Parallel()

	router, repos := newAITagHandlerTestRouterWithConfig(t, &config.Config{AI: config.AIConfig{AutoScanBatchSize: 1}})
	_, err := repos.manager.AddJob(t.Context(), domain.PlatformTaskTypeAITagGeneration, `{"image_id":999,"path":"/images/existing.png"}`)
	if err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/1/ai-tags", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp struct {
		JobIDs []int64 `json:"job_ids"`
	}
	decodeAIJSONBody(t, w.Body.Bytes(), &resp)
	if len(resp.JobIDs) != 1 {
		t.Fatalf("job_ids = %+v, want one job id", resp.JobIDs)
	}
	if got := repos.manager.QueuedByType(domain.PlatformTaskTypeAITagGeneration); got != 1 {
		t.Fatalf("QueuedByType(ai_tag_generation) = %d, want 1", got)
	}
	if repos.manager.QueueSize() != 1 {
		t.Fatalf("QueueSize() = %d, want 1", repos.manager.QueueSize())
	}
}

type aiTagHandlerTestRepos struct {
	db              *sql.DB
	jobRepo         repository.JobRepository
	taskRepo        repository.PlatformTaskRepository
	batchRepo       repository.TaskBatchRepository
	taskPlatformSvc *service.TaskPlatformService
	manager         *worker.Manager
}

func newAITagHandlerTestRouter(t *testing.T) (*gin.Engine, *aiTagHandlerTestRepos) {
	t.Helper()
	return newAITagHandlerTestRouterWithConfig(t, &config.Config{AI: config.AIConfig{AutoScanBatchSize: 100}})
}

func newAITagHandlerTestRouterWithConfig(t *testing.T, cfg *config.Config) (*gin.Engine, *aiTagHandlerTestRepos) {
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
		VALUES
			(1, '/images/1.png', '1.png', '/images', 100, 100, 100, 'png', ?, ?),
			(2, '/images/2.png', '2.png', '/images', 120, 120, 120, 'png', ?, ?)
	`, now, now, now, now)
	if err != nil {
		t.Fatalf("seed image: %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	batchRepo := repository.NewTaskBatchRepository(db)
	taskPlatformSvc := service.NewTaskPlatformService(batchRepo, taskRepo, jobRepo)
	manager := worker.NewManager(jobRepo)
	imageRepo := repository.NewImageRepository(db)
	h := NewAITagHandler(manager, imageRepo, jobRepo, taskRepo, taskPlatformSvc, func() *config.Config { return cfg })

	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/images/:id/ai-tags", h.TriggerAITags)
	api.GET("/images/:id/ai-tags/status", h.GetAITagStatus)
	api.POST("/images/batch-ai-tags", h.BatchTriggerAITags)

	return router, &aiTagHandlerTestRepos{
		db:              db,
		jobRepo:         jobRepo,
		taskRepo:        taskRepo,
		batchRepo:       batchRepo,
		taskPlatformSvc: taskPlatformSvc,
		manager:         manager,
	}
}

func (r *aiTagHandlerTestRepos) mustFindImage(t *testing.T, imageID int64) *domain.Image {
	t.Helper()
	image, err := repository.NewImageRepository(r.db).FindByID(imageID)
	if err != nil {
		t.Fatalf("FindByID(%d) error = %v", imageID, err)
	}
	return image
}

func decodeAIJSONBody(t *testing.T, body []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, string(body))
	}
}
