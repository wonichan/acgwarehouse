package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestAutoEnqueueCondition(t *testing.T) {
	t.Parallel()

	t.Run("thumbnail ready image without AI tags is enqueued", func(t *testing.T) {
		t.Parallel()

		env := newAITagAutoSchedulerIntegrationEnv(t)
		image := env.saveImage("eligible.png", "/thumb/eligible-small.jpg")

		queued, err := env.scheduler.ScanAndEnqueue(context.Background())
		if err != nil {
			t.Fatalf("ScanAndEnqueue() error = %v", err)
		}
		if queued != 1 {
			t.Fatalf("queued = %d, want 1", queued)
		}
		if got := env.countPlatformTasks(); got != 1 {
			t.Fatalf("platform task count = %d, want 1", got)
		}
		task := env.mustFindPlatformTaskByImageID(image.ID)
		if task.TaskType != domain.PlatformTaskTypeAITagGeneration {
			t.Fatalf("task.TaskType = %q, want %q", task.TaskType, domain.PlatformTaskTypeAITagGeneration)
		}
		if task.SourceType != domain.TaskBatchSourceImportScan {
			t.Fatalf("task.SourceType = %q, want %q", task.SourceType, domain.TaskBatchSourceImportScan)
		}
		jobs, err := env.jobRepo.FindByPlatformTaskID(task.ID)
		if err != nil {
			t.Fatalf("FindByPlatformTaskID() error = %v", err)
		}
		if len(jobs) != 1 {
			t.Fatalf("len(jobs) = %d, want 1", len(jobs))
		}
	})

	t.Run("image without thumbnail is skipped", func(t *testing.T) {
		t.Parallel()

		env := newAITagAutoSchedulerIntegrationEnv(t)
		env.saveImage("no-thumbnail.png", "")

		queued, err := env.scheduler.ScanAndEnqueue(context.Background())
		if err != nil {
			t.Fatalf("ScanAndEnqueue() error = %v", err)
		}
		if queued != 0 {
			t.Fatalf("queued = %d, want 0", queued)
		}
		if got := env.countPlatformTasks(); got != 0 {
			t.Fatalf("platform task count = %d, want 0", got)
		}
	})

	t.Run("image with AI tags is skipped", func(t *testing.T) {
		t.Parallel()

		env := newAITagAutoSchedulerIntegrationEnv(t)
		image := env.saveImage("already-tagged.png", "/thumb/already-tagged.jpg")
		env.addImageTag(image.ID, "ai")

		queued, err := env.scheduler.ScanAndEnqueue(context.Background())
		if err != nil {
			t.Fatalf("ScanAndEnqueue() error = %v", err)
		}
		if queued != 0 {
			t.Fatalf("queued = %d, want 0", queued)
		}
		if got := env.countPlatformTasks(); got != 0 {
			t.Fatalf("platform task count = %d, want 0", got)
		}
	})

	t.Run("disabled config skips scan", func(t *testing.T) {
		t.Parallel()

		env := newAITagAutoSchedulerIntegrationEnv(t)
		env.scheduler.config.AI.AutoAITagOnImport = false
		env.saveImage("disabled.png", "/thumb/disabled.jpg")

		queued, err := env.scheduler.ScanAndEnqueue(context.Background())
		if err != nil {
			t.Fatalf("ScanAndEnqueue() error = %v", err)
		}
		if queued != 0 {
			t.Fatalf("queued = %d, want 0", queued)
		}
		if got := env.countPlatformTasks(); got != 0 {
			t.Fatalf("platform task count = %d, want 0", got)
		}
	})

	t.Run("existing pending and queued tasks are deduplicated", func(t *testing.T) {
		t.Parallel()

		for _, status := range []string{domain.PlatformTaskStatusPending, domain.PlatformTaskStatusQueued} {
			status := status
			t.Run(status, func(t *testing.T) {
				env := newAITagAutoSchedulerIntegrationEnv(t)
				image := env.saveImage("dedupe-"+status+".png", "/thumb/dedupe-"+status+".jpg")
				env.seedExistingAITask(image, status)

				queued, err := env.scheduler.ScanAndEnqueue(context.Background())
				if err != nil {
					t.Fatalf("ScanAndEnqueue() error = %v", err)
				}
				if queued != 0 {
					t.Fatalf("queued = %d, want 0", queued)
				}
				if got := env.countPlatformTasks(); got != 1 {
					t.Fatalf("platform task count = %d, want 1", got)
				}
			})
		}
	})
}

func TestAutoSchedulingE2E(t *testing.T) {
	TestAutoEnqueueCondition(t)
}

func TestAutoEnqueueBatchLimit(t *testing.T) {
	t.Parallel()

	env := newAITagAutoSchedulerIntegrationEnv(t)
	for i := 0; i < 150; i++ {
		env.saveImage(time.Now().Format("150405.000000000")+"-batch.png", "/thumb/batch.jpg")
	}

	queued, err := env.scheduler.ScanAndEnqueue(context.Background())
	if err != nil {
		t.Fatalf("first ScanAndEnqueue() error = %v", err)
	}
	if queued != 100 {
		t.Fatalf("first queued = %d, want 100", queued)
	}
	if got := env.countPlatformTasks(); got != 100 {
		t.Fatalf("platform task count after first scan = %d, want 100", got)
	}

	queued, err = env.scheduler.ScanAndEnqueue(context.Background())
	if err != nil {
		t.Fatalf("second ScanAndEnqueue() error = %v", err)
	}
	if queued != 50 {
		t.Fatalf("second queued = %d, want 50", queued)
	}
	if got := env.countPlatformTasks(); got != 150 {
		t.Fatalf("platform task count after second scan = %d, want 150", got)
	}
}

func TestAITagAutoSchedulerScanAndEnqueueCallsFindImagesWithoutAITags(t *testing.T) {
	t.Parallel()

	finder := &fakeAITagImageFinder{
		images: []domain.Image{{ID: 1, Path: "/images/1.png", SourceRoot: "/images", FileSize: 10}},
	}
	platform := &fakeAITagTaskPlatform{
		planResult: &TaskPlatformPlanResult{CreatedTasks: []domain.PlatformTask{{ID: 11, ImageID: 1}}},
	}
	scheduler := NewAITagAutoScheduler(finder, platform, schedulerTestConfig())

	queued, err := scheduler.ScanAndEnqueue(context.Background())
	if err != nil {
		t.Fatalf("ScanAndEnqueue() error = %v", err)
	}
	if queued != 1 {
		t.Fatalf("queued = %d, want 1", queued)
	}
	if finder.lastLimit != 100 {
		t.Fatalf("FindImagesWithoutAITags limit = %d, want 100", finder.lastLimit)
	}
}

func TestAITagAutoSchedulerScanAndEnqueueCreatesPlatformTasks(t *testing.T) {
	t.Parallel()

	images := []domain.Image{
		{ID: 1, Path: "/images/1.png", SourceRoot: "/images", FileSize: 10},
		{ID: 2, Path: "/images/2.png", SourceRoot: "/images", FileSize: 20},
	}
	finder := &fakeAITagImageFinder{images: images}
	platform := &fakeAITagTaskPlatform{
		planResult: &TaskPlatformPlanResult{CreatedTasks: []domain.PlatformTask{{ID: 11, ImageID: 1}, {ID: 12, ImageID: 2}}},
	}
	scheduler := NewAITagAutoScheduler(finder, platform, schedulerTestConfig())

	queued, err := scheduler.ScanAndEnqueue(context.Background())
	if err != nil {
		t.Fatalf("ScanAndEnqueue() error = %v", err)
	}
	if queued != 2 {
		t.Fatalf("queued = %d, want 2", queued)
	}
	if platform.planRequest == nil {
		t.Fatal("expected PlanBatch to be called")
	}
	if platform.planRequest.SourceType != domain.TaskBatchSourceImportScan {
		t.Fatalf("SourceType = %q, want %q", platform.planRequest.SourceType, domain.TaskBatchSourceImportScan)
	}
	if len(platform.planRequest.Items) != 2 {
		t.Fatalf("len(plan items) = %d, want 2", len(platform.planRequest.Items))
	}
	if len(platform.queuedPayloads) != 2 {
		t.Fatalf("len(queued payloads) = %d, want 2", len(platform.queuedPayloads))
	}
	var payload struct {
		ImageID int64  `json:"image_id"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal([]byte(platform.queuedPayloads[0]), &payload); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if payload.ImageID != 1 || payload.Path != "/images/1.png" {
		t.Fatalf("queued payload = %+v, want image 1 path", payload)
	}
	if platform.queuedJobType != domain.PlatformTaskTypeAITagGeneration {
		t.Fatalf("queued job type = %q, want %q", platform.queuedJobType, domain.PlatformTaskTypeAITagGeneration)
	}
}

func TestAITagAutoSchedulerScanAndEnqueueUsesConfiguredBatchSize(t *testing.T) {
	t.Parallel()

	cfg := schedulerTestConfig()
	cfg.AI.AutoScanBatchSize = 7
	finder := &fakeAITagImageFinder{}
	platform := &fakeAITagTaskPlatform{planResult: &TaskPlatformPlanResult{}}
	scheduler := NewAITagAutoScheduler(finder, platform, cfg)

	if _, err := scheduler.ScanAndEnqueue(context.Background()); err != nil {
		t.Fatalf("ScanAndEnqueue() error = %v", err)
	}
	if finder.lastLimit != 7 {
		t.Fatalf("FindImagesWithoutAITags limit = %d, want 7", finder.lastLimit)
	}
}

func TestAITagAutoSchedulerStartTriggersPeriodicScan(t *testing.T) {
	t.Parallel()

	finder := &fakeAITagImageFinder{}
	platform := &fakeAITagTaskPlatform{planResult: &TaskPlatformPlanResult{}}
	ticker := newFakeSchedulerTicker()
	scheduler := NewAITagAutoScheduler(finder, platform, schedulerTestConfig())
	scheduler.tickerFactory = func(time.Duration) schedulerTicker { return ticker }

	called := make(chan struct{}, 1)
	scheduler.scanAndEnqueue = func(context.Context) (int, error) {
		called <- struct{}{}
		return 0, nil
	}

	scheduler.Start(context.Background())
	t.Cleanup(scheduler.Stop)

	ticker.tick()

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("expected periodic scan to run")
	}
}

func TestAITagAutoSchedulerStopStopsTicker(t *testing.T) {
	t.Parallel()

	finder := &fakeAITagImageFinder{}
	platform := &fakeAITagTaskPlatform{planResult: &TaskPlatformPlanResult{}}
	ticker := newFakeSchedulerTicker()
	scheduler := NewAITagAutoScheduler(finder, platform, schedulerTestConfig())
	scheduler.tickerFactory = func(time.Duration) schedulerTicker { return ticker }

	var calls int
	var mu sync.Mutex
	scheduler.scanAndEnqueue = func(context.Context) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		return 0, nil
	}

	scheduler.Start(context.Background())
	scheduler.Stop()
	ticker.tick()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !ticker.stopped {
		t.Fatal("expected ticker to be stopped")
	}
	if calls != 0 {
		t.Fatalf("scan calls after Stop = %d, want 0", calls)
	}
}

func TestAITagAutoSchedulerScanAndEnqueueSkipsWhenDisabled(t *testing.T) {
	t.Parallel()

	cfg := schedulerTestConfig()
	cfg.AI.AutoAITagOnImport = false
	finder := &fakeAITagImageFinder{}
	platform := &fakeAITagTaskPlatform{planResult: &TaskPlatformPlanResult{}}
	scheduler := NewAITagAutoScheduler(finder, platform, cfg)

	queued, err := scheduler.ScanAndEnqueue(context.Background())
	if err != nil {
		t.Fatalf("ScanAndEnqueue() error = %v", err)
	}
	if queued != 0 {
		t.Fatalf("queued = %d, want 0", queued)
	}
	if finder.called {
		t.Fatal("expected finder not to be called when disabled")
	}
}

type fakeAITagImageFinder struct {
	images    []domain.Image
	err       error
	called    bool
	lastLimit int
}

func (f *fakeAITagImageFinder) FindImagesWithoutAITags(_ context.Context, limit int) ([]domain.Image, error) {
	f.called = true
	f.lastLimit = limit
	if f.err != nil {
		return nil, f.err
	}
	return append([]domain.Image(nil), f.images...), nil
}

type fakeAITagTaskPlatform struct {
	planResult     *TaskPlatformPlanResult
	planErr        error
	queueErr       error
	planRequest    *TaskPlatformPlanRequest
	queuedJobType  string
	queuedPayloads []string
}

func (f *fakeAITagTaskPlatform) PlanBatch(_ context.Context, req TaskPlatformPlanRequest) (*TaskPlatformPlanResult, error) {
	f.planRequest = &req
	if f.planErr != nil {
		return nil, f.planErr
	}
	if f.planResult == nil {
		return &TaskPlatformPlanResult{}, nil
	}
	copyResult := *f.planResult
	copyResult.CreatedTasks = append([]domain.PlatformTask(nil), f.planResult.CreatedTasks...)
	return &copyResult, nil
}

func (f *fakeAITagTaskPlatform) QueueTask(_ context.Context, _ *domain.PlatformTask, jobType, payload string) (*domain.AsyncJob, error) {
	f.queuedJobType = jobType
	f.queuedPayloads = append(f.queuedPayloads, payload)
	if f.queueErr != nil {
		return nil, f.queueErr
	}
	return &domain.AsyncJob{ID: int64(len(f.queuedPayloads))}, nil
}

type fakeSchedulerTicker struct {
	ch      chan time.Time
	stopped bool
}

func newFakeSchedulerTicker() *fakeSchedulerTicker {
	return &fakeSchedulerTicker{ch: make(chan time.Time, 1)}
}

func (f *fakeSchedulerTicker) C() <-chan time.Time {
	return f.ch
}

func (f *fakeSchedulerTicker) Stop() {
	f.stopped = true
}

func (f *fakeSchedulerTicker) tick() {
	if !f.stopped {
		f.ch <- time.Now()
	}
}

func schedulerTestConfig() *config.Config {
	return &config.Config{AI: config.AIConfig{AutoAITagOnImport: true, AutoScanIntervalMinutes: 5, AutoScanBatchSize: 100}}
}

type aiTagAutoSchedulerIntegrationEnv struct {
	db         *sql.DB
	imageRepo  repository.ImageRepository
	batchRepo  repository.TaskBatchRepository
	taskRepo   repository.PlatformTaskRepository
	jobRepo    repository.JobRepository
	tagRepo    repository.TagRepository
	scheduler  *AITagAutoScheduler
	nextTagSeq int
}

func newAITagAutoSchedulerIntegrationEnv(t *testing.T) *aiTagAutoSchedulerIntegrationEnv {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "ai-tag-auto-scheduler.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	batchRepo := repository.NewTaskBatchRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	jobRepo := repository.NewJobRepository(db)
	tagRepo := repository.NewTagRepository(db)
	taskPlatform := NewTaskPlatformService(batchRepo, taskRepo, jobRepo)

	return &aiTagAutoSchedulerIntegrationEnv{
		db:         db,
		imageRepo:  imageRepo,
		batchRepo:  batchRepo,
		taskRepo:   taskRepo,
		jobRepo:    jobRepo,
		tagRepo:    tagRepo,
		scheduler:  NewAITagAutoScheduler(imageRepo, taskPlatform, schedulerTestConfig()),
		nextTagSeq: 1,
	}
}

func (e *aiTagAutoSchedulerIntegrationEnv) saveImage(filename, thumbnailSmallURL string) *domain.Image {
	e.nextTagSeq++
	image := &domain.Image{
		Path:              "/auto-scheduler/" + filename,
		Filename:          filename,
		SourceRoot:        "/auto-scheduler",
		FileSize:          512 + int64(e.nextTagSeq),
		Width:             128,
		Height:            128,
		Format:            "png",
		PHash:             int64(1000 + e.nextTagSeq),
		ThumbnailSmallUrl: thumbnailSmallURL,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if _, err := e.imageRepo.SaveImage(image); err != nil {
		panic(err)
	}
	return image
}

func (e *aiTagAutoSchedulerIntegrationEnv) addImageTag(imageID int64, source string) {
	tag := &domain.Tag{
		PreferredLabel: "tag-" + time.Now().Format("150405.000000000") + "-" + source,
		Slug:           "tag-" + source + "-" + time.Now().Format("150405000000000"),
		ReviewState:    "confirmed",
	}
	if err := e.tagRepo.Save(context.Background(), tag); err != nil {
		panic(err)
	}
	if _, err := e.db.Exec(`
		INSERT INTO image_tags (image_id, tag_id, source, confidence, review_state)
		VALUES (?, ?, ?, ?, ?)
	`, imageID, tag.ID, source, 0.99, "confirmed"); err != nil {
		panic(err)
	}
	e.nextTagSeq++
}

func (e *aiTagAutoSchedulerIntegrationEnv) seedExistingAITask(image *domain.Image, status string) {
	ctx := context.Background()
	batch := &domain.TaskBatch{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: "existing ai task",
		Status:       domain.TaskBatchStatusPending,
		CreatedAt:    time.Now(),
	}
	if err := e.batchRepo.Create(ctx, batch); err != nil {
		panic(err)
	}
	task := &domain.PlatformTask{
		BatchID:         batch.ID,
		ImageID:         image.ID,
		TaskType:        domain.PlatformTaskTypeAITagGeneration,
		SourceType:      domain.TaskBatchSourceImportScan,
		Status:          status,
		DedupeKey:       BuildImageVersionKey(image) + ":" + domain.PlatformTaskTypeAITagGeneration,
		ImageVersionKey: BuildImageVersionKey(image),
		CreatedAt:       time.Now(),
	}
	if status == domain.PlatformTaskStatusQueued {
		queuedAt := time.Now()
		task.QueuedAt = &queuedAt
	}
	if err := e.taskRepo.Create(ctx, task); err != nil {
		panic(err)
	}
}

func (e *aiTagAutoSchedulerIntegrationEnv) countPlatformTasks() int {
	var count int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM platform_tasks`).Scan(&count); err != nil {
		panic(err)
	}
	return count
}

func (e *aiTagAutoSchedulerIntegrationEnv) mustFindPlatformTaskByImageID(imageID int64) *domain.PlatformTask {
	task := &domain.PlatformTask{}
	row := e.db.QueryRow(`
		SELECT id, batch_id, image_id, task_type, source_type, status, dedupe_key, image_version_key,
		       latest_async_job_id, skip_reason, error_summary, created_at, queued_at, started_at, finished_at
		FROM platform_tasks WHERE image_id = ?
	`, imageID)
	var latestAsyncJobID sql.NullInt64
	var skipReason sql.NullString
	var errorSummary sql.NullString
	var queuedAt sql.NullTime
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	if err := row.Scan(
		&task.ID,
		&task.BatchID,
		&task.ImageID,
		&task.TaskType,
		&task.SourceType,
		&task.Status,
		&task.DedupeKey,
		&task.ImageVersionKey,
		&latestAsyncJobID,
		&skipReason,
		&errorSummary,
		&task.CreatedAt,
		&queuedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		panic(err)
	}
	if latestAsyncJobID.Valid {
		task.LatestAsyncJobID = &latestAsyncJobID.Int64
	}
	if skipReason.Valid {
		task.SkipReason = &skipReason.String
	}
	if errorSummary.Valid {
		task.ErrorSummary = &errorSummary.String
	}
	if queuedAt.Valid {
		task.QueuedAt = &queuedAt.Time
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		task.FinishedAt = &finishedAt.Time
	}
	return task
}
