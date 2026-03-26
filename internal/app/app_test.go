package app

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
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

type schedulerLifecycleTracker struct {
	mu     sync.Mutex
	starts int
	stops  int
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
	configYAML := []byte("server:\n  host: 127.0.0.1\n  port: 0\n  env: test\ndatabase:\n  type: sqlite\n  path: " + dbPath + "\nstorage:\n  scan_roots: []\nai: {}\ncos: {}\nadmin:\n  username: \"\"\n  password: \"\"\nworker_pool:\n  worker_count: 1\n  queue_size: 8\n  refill_interval_seconds: 1\n  refill_threshold: 0.5\n")
	if err := os.WriteFile(cfgPath, configYAML, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return cfgPath
}
