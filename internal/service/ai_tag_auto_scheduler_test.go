package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

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
