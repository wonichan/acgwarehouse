package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

func TestNewInitializesAdminActionsWithWorkerManager(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "app.db")
	cfgPath := writeTestConfig(t, dbPath)

	app, err := New(cfgPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := app.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	jobID, err := app.adminSvc.TriggerScan(context.Background())
	if err != nil {
		t.Fatalf("TriggerScan() error = %v", err)
	}
	if jobID <= 0 {
		t.Fatalf("TriggerScan() jobID = %d, want > 0", jobID)
	}
}

func TestNewInitializesAutoScheduler(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "app.db")
	cfgPath := writeTestConfig(t, dbPath)

	app, err := New(cfgPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := app.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	if app.autoScheduler == nil {
		t.Fatal("expected autoScheduler to be initialized")
	}
}

func TestAutoSchedulerStartStartsOnlyOnceWhenEnabled(t *testing.T) {
	t.Parallel()

	tracker := &schedulerLifecycleTracker{}
	app := newTestLifecycleApp(tracker, &config.Config{AI: config.AIConfig{AutoAITagOnImport: true}})

	app.startAutoScheduler()
	app.startAutoScheduler()

	if tracker.starts != 1 {
		t.Fatalf("starts = %d, want 1", tracker.starts)
	}
}

func TestAutoSchedulerStartSkipsWhenDisabled(t *testing.T) {
	t.Parallel()

	tracker := &schedulerLifecycleTracker{}
	app := newTestLifecycleApp(tracker, &config.Config{AI: config.AIConfig{AutoAITagOnImport: false}})

	app.startAutoScheduler()

	if tracker.starts != 0 {
		t.Fatalf("starts = %d, want 0", tracker.starts)
	}
}

func TestShutdownStopsAutoSchedulerOnlyOnce(t *testing.T) {
	t.Parallel()

	tracker := &schedulerLifecycleTracker{}
	app := newTestLifecycleApp(tracker, &config.Config{AI: config.AIConfig{AutoAITagOnImport: true}})
	app.startAutoScheduler()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := app.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() second call error = %v", err)
	}

	if tracker.stops != 1 {
		t.Fatalf("stops = %d, want 1", tracker.stops)
	}
}

func TestAutoSchedulerConfigReloadRestartsOnRelevantChanges(t *testing.T) {
	t.Parallel()

	factory := &schedulerFactoryTracker{}
	app := &App{
		config:               &config.Config{AI: config.AIConfig{AutoAITagOnImport: true, AutoScanIntervalMinutes: 5, AutoScanBatchSize: 100}},
		newAutoScheduler:     factory.new,
		refillStopCh:         make(chan struct{}),
		autoSchedulerControl: factory.new(&config.Config{AI: config.AIConfig{AutoAITagOnImport: true, AutoScanIntervalMinutes: 5, AutoScanBatchSize: 100}}),
		autoSchedulerStarted: true,
	}

	oldCfg := &config.Config{AI: config.AIConfig{AutoAITagOnImport: true, AutoScanIntervalMinutes: 5, AutoScanBatchSize: 100}}
	newCfg := &config.Config{AI: config.AIConfig{AutoAITagOnImport: true, AutoScanIntervalMinutes: 9, AutoScanBatchSize: 100}}

	app.handleAutoSchedulerConfigChange(oldCfg, newCfg)

	if len(factory.schedulers) != 2 {
		t.Fatalf("scheduler instances = %d, want 2", len(factory.schedulers))
	}
	if factory.schedulers[0].stops != 1 {
		t.Fatalf("old scheduler stops = %d, want 1", factory.schedulers[0].stops)
	}
	if factory.schedulers[1].starts != 1 {
		t.Fatalf("new scheduler starts = %d, want 1", factory.schedulers[1].starts)
	}
}

func TestAutoSchedulerConfigReloadStopsWhenDisabled(t *testing.T) {
	t.Parallel()

	factory := &schedulerFactoryTracker{}
	app := &App{
		config:               &config.Config{AI: config.AIConfig{AutoAITagOnImport: true, AutoScanIntervalMinutes: 5, AutoScanBatchSize: 100}},
		newAutoScheduler:     factory.new,
		refillStopCh:         make(chan struct{}),
		autoSchedulerControl: factory.new(&config.Config{AI: config.AIConfig{AutoAITagOnImport: true, AutoScanIntervalMinutes: 5, AutoScanBatchSize: 100}}),
		autoSchedulerStarted: true,
	}

	oldCfg := &config.Config{AI: config.AIConfig{AutoAITagOnImport: true, AutoScanIntervalMinutes: 5, AutoScanBatchSize: 100}}
	newCfg := &config.Config{AI: config.AIConfig{AutoAITagOnImport: false, AutoScanIntervalMinutes: 5, AutoScanBatchSize: 100}}

	app.handleAutoSchedulerConfigChange(oldCfg, newCfg)

	if len(factory.schedulers) != 1 {
		t.Fatalf("scheduler instances = %d, want 1", len(factory.schedulers))
	}
	if factory.schedulers[0].stops != 1 {
		t.Fatalf("scheduler stops = %d, want 1", factory.schedulers[0].stops)
	}
	if app.autoSchedulerStarted {
		t.Fatal("expected scheduler to be marked stopped")
	}
}

func TestRefillReadyJobsRespectsBatchSize(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML := []byte("server:\n  host: 127.0.0.1\n  port: 0\n  env: test\ndatabase:\n  type: sqlite\n  path: \":memory:\"\nstorage:\n  scan_roots: []\nai: {}\ncos: {}\nadmin:\n  username: \"\"\n  password: \"\"\nworker_pool:\n  worker_count: 1\n  queue_size: 8\n  refill_interval_seconds: 1\n  refill_threshold: 0.5\n  refill_batch_size: 2\n")
	if err := os.WriteFile(cfgPath, configYAML, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	reloader, err := config.NewReloader(cfgPath)
	if err != nil {
		t.Fatalf("NewReloader() error = %v", err)
	}

	repo := &refillTestJobRepo{jobs: []domain.AsyncJob{
		{ID: 1, Status: "ready"},
		{ID: 2, Status: "ready"},
		{ID: 3, Status: "ready"},
		{ID: 4, Status: "ready"},
	}}
	manager := worker.NewManagerWithConfig(repo, 1, 8)
	app := &App{
		cfgReloader:  reloader,
		jobRepo:      repo,
		jobManager:   manager,
		refillStopCh: make(chan struct{}),
	}

	loaded := app.refillReadyJobs()
	if loaded != 2 {
		t.Fatalf("refillReadyJobs() loaded = %d, want 2", loaded)
	}
	if manager.QueueSize() != 2 {
		t.Fatalf("QueueSize() = %d, want 2", manager.QueueSize())
	}
	if repo.findByStatusCalls != 1 {
		t.Fatalf("FindByStatus calls = %d, want 1", repo.findByStatusCalls)
	}
}

func TestRefillReadyJobsSkipsAITasksBeyondQueueLimitAndLoadsThumbnailJobs(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML := []byte("server:\n  host: 127.0.0.1\n  port: 0\n  env: test\ndatabase:\n  type: sqlite\n  path: \":memory:\"\nstorage:\n  scan_roots: []\nai:\n  auto_scan_batch_size: 1\ncos: {}\nadmin:\n  username: \"\"\n  password: \"\"\nworker_pool:\n  worker_count: 1\n  queue_size: 8\n  refill_interval_seconds: 1\n  refill_threshold: 0.5\n  refill_batch_size: 3\n")
	if err := os.WriteFile(cfgPath, configYAML, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	reloader, err := config.NewReloader(cfgPath)
	if err != nil {
		t.Fatalf("NewReloader() error = %v", err)
	}

	repo := &refillTestJobRepo{jobs: []domain.AsyncJob{
		{ID: 1, Type: domain.PlatformTaskTypeAITagGeneration, Status: "ready"},
		{ID: 2, Type: domain.PlatformTaskTypeAITagGeneration, Status: "ready"},
		{ID: 3, Type: domain.PlatformTaskTypeThumbnailGenerate, Status: "ready"},
	}}
	manager := worker.NewManagerWithConfig(repo, 1, 8)
	app := &App{
		cfgReloader:  reloader,
		jobRepo:      repo,
		jobManager:   manager,
		refillStopCh: make(chan struct{}),
	}

	loaded := app.refillReadyJobs()
	if loaded != 2 {
		t.Fatalf("refillReadyJobs() loaded = %d, want 2", loaded)
	}
	if got := manager.QueuedByType(domain.PlatformTaskTypeAITagGeneration); got != 1 {
		t.Fatalf("QueuedByType(ai_tag_generation) = %d, want 1", got)
	}
	if got := manager.QueuedByType(domain.PlatformTaskTypeThumbnailGenerate); got != 1 {
		t.Fatalf("QueuedByType(thumbnail_generate) = %d, want 1", got)
	}
	if manager.QueueSize() != 2 {
		t.Fatalf("QueueSize() = %d, want 2", manager.QueueSize())
	}
}

func TestPrepareSidecarStartupMarksDegradedWhenSidecarFails(t *testing.T) {
	t.Parallel()

	runtime := &sidecarRuntimeTracker{startErr: errors.New("startup timeout")}
	app := &App{sidecarRuntime: runtime, refillStopCh: make(chan struct{})}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := app.prepareSidecarStartup(ctx); err != nil {
		t.Fatalf("prepareSidecarStartup() error = %v", err)
	}
	if app.sidecarMode != sidecarModeDegraded {
		t.Fatalf("sidecarMode = %q, want %q", app.sidecarMode, sidecarModeDegraded)
	}
	if app.fullyReady {
		t.Fatal("fullyReady = true, want false when sidecar startup fails")
	}
}

func TestPrepareSidecarStartupMarksFullyReadyWhenSidecarReady(t *testing.T) {
	t.Parallel()

	runtime := &sidecarRuntimeTracker{state: sidecar.StateReady}
	app := &App{sidecarRuntime: runtime, refillStopCh: make(chan struct{})}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := app.prepareSidecarStartup(ctx); err != nil {
		t.Fatalf("prepareSidecarStartup() error = %v", err)
	}
	if app.sidecarMode != sidecarModeReady {
		t.Fatalf("sidecarMode = %q, want %q", app.sidecarMode, sidecarModeReady)
	}
	if !app.fullyReady {
		t.Fatal("fullyReady = false, want true when Go and sidecar are ready")
	}
}

func TestShutdownStopsSidecarRuntimeOnlyOnce(t *testing.T) {
	t.Parallel()

	runtime := &sidecarRuntimeTracker{state: sidecar.StateReady}
	app := &App{sidecarRuntime: runtime, refillStopCh: make(chan struct{})}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := app.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() second call error = %v", err)
	}
	if runtime.stops != 1 {
		t.Fatalf("sidecar Stop calls = %d, want 1", runtime.stops)
	}
}

func TestAppAdminOverviewReportsDegradedWhenSidecarStartupFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "app.db")
	cfgPath := writeTestConfig(t, dbPath)

	app, err := New(cfgPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := app.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	tracker := &sidecarRuntimeTracker{startErr: errors.New("startup timeout")}
	app.sidecarRuntime = tracker

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.prepareSidecarStartup(ctx); err != nil {
		t.Fatalf("prepareSidecarStartup() error = %v", err)
	}

	overview, err := app.adminSvc.GetTaskPlatformOverview(context.Background())
	if err != nil {
		t.Fatalf("GetTaskPlatformOverview() error = %v", err)
	}

	if overview.Sidecar.State != "degraded" {
		t.Fatalf("sidecar.state = %q, want degraded", overview.Sidecar.State)
	}
	if overview.Sidecar.LastProbeResult != "failed" {
		t.Fatalf("sidecar.last_probe_result = %q, want failed", overview.Sidecar.LastProbeResult)
	}
	if overview.Sidecar.LastErrorSummary == "" {
		t.Fatal("expected non-empty sidecar.last_error_summary")
	}
}

func TestAppAdminOverviewUpdatesWhenSidecarCrashesAfterStartup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "app.db")
	cfgPath := writeTestConfig(t, dbPath)

	app, err := New(cfgPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := app.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	tracker := &sidecarRuntimeTracker{state: sidecar.StateReady}
	app.sidecarRuntime = tracker

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.prepareSidecarStartup(ctx); err != nil {
		t.Fatalf("prepareSidecarStartup() error = %v", err)
	}

	tracker.setStateAndError(sidecar.StateDegraded, "process exited")

	overview, err := app.adminSvc.GetTaskPlatformOverview(context.Background())
	if err != nil {
		t.Fatalf("GetTaskPlatformOverview() error = %v", err)
	}

	if overview.Sidecar.State != "degraded" {
		t.Fatalf("sidecar.state = %q, want degraded", overview.Sidecar.State)
	}
	if overview.Sidecar.LastProbeResult != "failed" {
		t.Fatalf("sidecar.last_probe_result = %q, want failed", overview.Sidecar.LastProbeResult)
	}
	if overview.Sidecar.LastErrorSummary != "process exited" {
		t.Fatalf("sidecar.last_error_summary = %q, want process exited", overview.Sidecar.LastErrorSummary)
	}
}

type schedulerLifecycleTracker struct {
	mu     sync.Mutex
	starts int
	stops  int
}

type sidecarRuntimeTracker struct {
	mu       sync.Mutex
	startErr error
	state    sidecar.State
	starts   int
	stops    int
}

func (s *sidecarRuntimeTracker) Start(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.starts++
	if s.startErr != nil {
		s.state = sidecar.StateDegraded
		return s.startErr
	}
	s.state = sidecar.StateReady
	return nil
}

func (s *sidecarRuntimeTracker) Stop(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stops++
	s.state = sidecar.StateStopped
	return nil
}

func (s *sidecarRuntimeTracker) State() sidecar.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *sidecarRuntimeTracker) Status() sidecar.Status {
	return sidecar.Status{State: s.State()}
}

func (s *sidecarRuntimeTracker) setStateAndError(state sidecar.State, summary string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
	s.startErr = errors.New(summary)
}

type schedulerFactoryTracker struct {
	mu         sync.Mutex
	schedulers []*schedulerLifecycleTracker
}

func (s *schedulerFactoryTracker) new(*config.Config) autoSchedulerLifecycle {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracker := &schedulerLifecycleTracker{}
	s.schedulers = append(s.schedulers, tracker)
	return tracker
}

func (s *schedulerLifecycleTracker) Start(context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.starts++
}

func (s *schedulerLifecycleTracker) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stops++
}

func newTestLifecycleApp(tracker autoSchedulerLifecycle, cfg *config.Config) *App {
	return &App{
		config:               cfg,
		refillStopCh:         make(chan struct{}),
		autoSchedulerControl: tracker,
	}
}

func writeTestConfig(t *testing.T, dbPath string) string {
	t.Helper()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML := []byte("server:\n  host: 127.0.0.1\n  port: 0\n  env: test\ndatabase:\n  type: sqlite\n  path: " + dbPath + "\nstorage:\n  scan_roots: []\nai: {}\ncos: {}\nadmin:\n  username: \"\"\n  password: \"\"\nworker_pool:\n  worker_count: 1\n  queue_size: 8\n  refill_interval_seconds: 1\n  refill_threshold: 0.5\n  refill_batch_size: 0\n")
	if err := os.WriteFile(cfgPath, configYAML, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return cfgPath
}

type refillTestJobRepo struct {
	jobs              []domain.AsyncJob
	findByStatusCalls int
}

func (r *refillTestJobRepo) Save(*domain.AsyncJob) error                           { return nil }
func (r *refillTestJobRepo) FindByID(int64) (*domain.AsyncJob, error)              { return nil, os.ErrNotExist }
func (r *refillTestJobRepo) FindByPlatformTaskID(int64) ([]domain.AsyncJob, error) { return nil, nil }
func (r *refillTestJobRepo) FindByStatus(status string) ([]domain.AsyncJob, error) {
	r.findByStatusCalls++
	if status != "ready" {
		return []domain.AsyncJob{}, nil
	}
	result := make([]domain.AsyncJob, len(r.jobs))
	copy(result, r.jobs)
	return result, nil
}
func (r *refillTestJobRepo) FindByType(string) ([]domain.AsyncJob, error) { return nil, nil }
func (r *refillTestJobRepo) FindByTypeAndStatus(string, string) ([]domain.AsyncJob, error) {
	return nil, nil
}
func (r *refillTestJobRepo) Update(*domain.AsyncJob) error             { return nil }
func (r *refillTestJobRepo) FindRecent(int) ([]domain.AsyncJob, error) { return nil, nil }
func (r *refillTestJobRepo) FindFailed() ([]domain.AsyncJob, error)    { return nil, nil }
func (r *refillTestJobRepo) UpdateStatus(int64, string, *string) error { return nil }
func (r *refillTestJobRepo) CountByStatus(string) (int64, error)       { return 0, nil }
func (r *refillTestJobRepo) ResetRunningToReady() (int64, error)       { return 0, nil }
