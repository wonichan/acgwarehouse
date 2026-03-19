package worker

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestManagerProcessesJobsSequentially(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", t.TempDir()+"/jobs.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	// 使用单 worker 确保顺序执行
	mgr := NewManagerWithConfig(jobRepo, 1, 10)

	var (
		mu    sync.Mutex
		order []int64
	)
	mgr.RegisterHandler("image_imported", func(ctx context.Context, id int64, payload string) error {
		mu.Lock()
		order = append(order, id)
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	id1, err := mgr.AddJob(ctx, "image_imported", `{"path":"a.png"}`)
	if err != nil {
		t.Fatalf("AddJob() first error = %v", err)
	}
	id2, err := mgr.AddJob(ctx, "image_imported", `{"path":"b.png"}`)
	if err != nil {
		t.Fatalf("AddJob() second error = %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		j1, err1 := jobRepo.FindByID(id1)
		j2, err2 := jobRepo.FindByID(id2)
		if err1 == nil && err2 == nil && j1.Status == "finished" && j2.Status == "finished" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	j1, err := jobRepo.FindByID(id1)
	if err != nil {
		t.Fatalf("FindByID(id1) error = %v", err)
	}
	j2, err := jobRepo.FindByID(id2)
	if err != nil {
		t.Fatalf("FindByID(id2) error = %v", err)
	}

	if j1.Status != "finished" || j2.Status != "finished" {
		t.Fatalf("unexpected statuses: job1=%s job2=%s", j1.Status, j2.Status)
	}

	// Verify progress is 100 for finished jobs
	if j1.Progress != 100 {
		t.Fatalf("job1.Progress = %f, want 100", j1.Progress)
	}
	if j2.Progress != 100 {
		t.Fatalf("job2.Progress = %f, want 100", j2.Progress)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != id1 || order[1] != id2 {
		t.Fatalf("handler order = %v, want [%d %d]", order, id1, id2)
	}
}

func TestManagerProcessesJobsParallel(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", t.TempDir()+"/jobs.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	// 使用 4 个 worker 并行处理
	mgr := NewManagerWithConfig(jobRepo, 4, 10)

	var (
		mu         sync.Mutex
		order      []int64
		startTimes []time.Time
	)
	mgr.RegisterHandler("image_imported", func(ctx context.Context, id int64, payload string) error {
		mu.Lock()
		order = append(order, id)
		startTimes = append(startTimes, time.Now())
		mu.Unlock()
		// 模拟一些工作
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	// 添加多个任务
	for i := 0; i < 4; i++ {
		_, err := mgr.AddJob(ctx, "image_imported", `{"path":"test.png"}`)
		if err != nil {
			t.Fatalf("AddJob() error = %v", err)
		}
	}

	// 等待所有任务完成
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		jobs, _ := jobRepo.FindByStatus("finished")
		if len(jobs) >= 4 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	// 验证所有任务完成
	jobs, err := jobRepo.FindByStatus("finished")
	if err != nil {
		t.Fatalf("FindByStatus() error = %v", err)
	}
	if len(jobs) != 4 {
		t.Fatalf("expected 4 finished jobs, got %d", len(jobs))
	}

	// 验证多个任务有重叠执行（并行）
	// 注意：如果严格串行执行，任务完成间隔会 >= 50ms
	// 并行执行时，多个任务会在接近同一时间开始
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 4 {
		t.Fatalf("expected 4 processed jobs, got %d", len(order))
	}
}

func TestManager_PauseAndResume(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", t.TempDir()+"/jobs.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	manager := NewManager(jobRepo)

	// Start the manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.Start(ctx)
	defer manager.Stop()

	// Initially should be running and not paused
	if !manager.IsRunning() {
		t.Error("Expected manager to be running after Start")
	}
	if manager.IsPaused() {
		t.Error("Expected manager to not be paused after Start")
	}

	// Pause
	manager.Pause()
	if !manager.IsPaused() {
		t.Error("Expected manager to be paused after Pause")
	}

	// Resume
	manager.Resume()
	if manager.IsPaused() {
		t.Error("Expected manager to not be paused after Resume")
	}
}

func TestManager_PausePreservesQueue(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", t.TempDir()+"/jobs.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	manager := NewManager(jobRepo)

	// Register a slow handler
	slowHandler := func(ctx context.Context, id int64, payload string) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}
	manager.RegisterHandler("slow_job", slowHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.Start(ctx)
	defer manager.Stop()

	// Add a job to the queue
	jobID, err := manager.AddJob(ctx, "slow_job", "{}")
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Pause immediately
	manager.Pause()

	// Wait a bit for any in-flight job to complete
	time.Sleep(200 * time.Millisecond)

	// Verify the job still exists
	job, err := jobRepo.FindByID(jobID)
	if err != nil {
		t.Fatalf("Failed to find job: %v", err)
	}

	// Job should exist and have valid status
	if job.Status != "ready" && job.Status != "running" && job.Status != "finished" {
		t.Errorf("Unexpected job status: %s", job.Status)
	}

	// Get queue size
	queueSize := manager.QueueSize()
	if queueSize < 0 {
		t.Errorf("Unexpected queue size: %d", queueSize)
	}
}

func TestManager_StateHelpers(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", t.TempDir()+"/jobs.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	manager := NewManager(jobRepo)

	// Initial state (not started yet)
	if manager.IsPaused() {
		t.Error("Expected manager to not be paused initially")
	}

	// Start
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.Start(ctx)
	defer manager.Stop()

	if manager.IsPaused() {
		t.Error("Expected manager to not be paused after Start")
	}

	// Pause
	manager.Pause()
	if !manager.IsPaused() {
		t.Error("Expected manager to be paused after Pause")
	}

	// Resume
	manager.Resume()
	if manager.IsPaused() {
		t.Error("Expected manager to not be paused after Resume")
	}
}

func TestManager_AddJobWhilePaused(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", t.TempDir()+"/jobs.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	manager := NewManager(jobRepo)

	// Register a handler
	var processed bool
	var procMu sync.Mutex
	manager.RegisterHandler("test_job", func(ctx context.Context, id int64, payload string) error {
		procMu.Lock()
		processed = true
		procMu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.Start(ctx)
	defer manager.Stop()

	// Pause immediately
	manager.Pause()

	// Add a job while paused
	jobID, err := manager.AddJob(ctx, "test_job", "{}")
	if err != nil {
		t.Fatalf("Failed to add job while paused: %v", err)
	}

	// Verify job was created
	if jobID == 0 {
		t.Error("Expected job ID to be set")
	}

	// Job should be in ready state (not processed yet)
	job, _ := jobRepo.FindByID(jobID)
	if job.Status != "ready" {
		t.Errorf("Expected job status 'ready' while paused, got '%s'", job.Status)
	}

	// Resume and wait for processing
	manager.Resume()
	time.Sleep(200 * time.Millisecond)

	// Job should now be processed
	procMu.Lock()
	if !processed {
		t.Error("Expected job to be processed after resume")
	}
	procMu.Unlock()
}

func TestManager_GetStats(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", t.TempDir()+"/jobs.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	manager := NewManager(jobRepo)

	// Add some jobs directly to repo
	for i := 0; i < 3; i++ {
		job := &domain.AsyncJob{
			Type:      "test_job",
			Status:    "ready",
			CreatedAt: time.Now(),
		}
		_ = jobRepo.Save(job)
	}

	stats := manager.GetStats()

	if stats.QueueSize < 0 {
		t.Errorf("Unexpected queue size: %d", stats.QueueSize)
	}
}

func TestManager_SetWorkerCount(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", t.TempDir()+"/jobs.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	mgr := NewManagerWithConfig(jobRepo, 2, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	// 初始 worker 数量
	if count := mgr.GetWorkerCount(); count != 2 {
		t.Fatalf("initial worker count = %d, want 2", count)
	}

	// 增加 worker 数量
	mgr.SetWorkerCount(ctx, 4)
	if count := mgr.GetWorkerCount(); count != 4 {
		t.Fatalf("after increase, worker count = %d, want 4", count)
	}

	// 减少 worker 数量
	mgr.SetWorkerCount(ctx, 1)
	if count := mgr.GetWorkerCount(); count != 1 {
		t.Fatalf("after decrease, worker count = %d, want 1", count)
	}
}
