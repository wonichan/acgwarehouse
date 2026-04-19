package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type JobFunc func(ctx context.Context, id int64, payload string) error

type queuedJob struct {
	ctx context.Context
	job *domain.AsyncJob
}

// Stats holds runtime statistics about the job manager.
type Stats struct {
	QueueSize   int
	IsRunning   bool
	IsPaused    bool
	WorkerCount int
	JobTypes    map[string]int
}

type Manager struct {
	jobRepo     repository.JobRepository
	handlers    map[string]JobFunc
	queue       chan *queuedJob
	queueSize   int
	queueTypeMu sync.Mutex
	queueTypes  map[string]int

	// Worker pool (ants)
	pool        *ants.Pool
	poolMu      sync.RWMutex
	stopCh      chan struct{}
	dispatchWg  sync.WaitGroup
	taskWg      sync.WaitGroup
	running     bool
	runningMu   sync.Mutex
	paused      bool
	pausedMu    sync.Mutex
	pauseNotify chan struct{}
	pendingCnt  int32

	// 配置：worker 数量
	workerCount atomic.Int32
}

// DefaultWorkerCount is the default number of workers.
const DefaultWorkerCount = 4

// DefaultQueueSize is the default queue buffer size.
const DefaultQueueSize = 512

func NewManager(jobRepo repository.JobRepository) *Manager {
	return NewManagerWithConfig(jobRepo, DefaultWorkerCount, DefaultQueueSize)
}

// NewManagerWithWorkers creates a Manager with specified worker count (uses default queue size).
// Deprecated: Use NewManagerWithConfig instead.
func NewManagerWithWorkers(jobRepo repository.JobRepository, workerCount int) *Manager {
	return NewManagerWithConfig(jobRepo, workerCount, DefaultQueueSize)
}

// NewManagerWithConfig creates a Manager with specified worker count and queue size.
func NewManagerWithConfig(jobRepo repository.JobRepository, workerCount, queueSize int) *Manager {
	if workerCount < 1 {
		workerCount = DefaultWorkerCount
	}
	if queueSize < 1 {
		queueSize = DefaultQueueSize
	}
	m := &Manager{
		jobRepo:     jobRepo,
		handlers:    make(map[string]JobFunc),
		queue:       make(chan *queuedJob, queueSize),
		queueSize:   queueSize,
		queueTypes:  make(map[string]int),
		pauseNotify: make(chan struct{}),
		stopCh:      make(chan struct{}),
	}
	m.workerCount.Store(int32(workerCount))
	return m
}

func (m *Manager) RegisterHandler(jobType string, handler JobFunc) {
	m.handlers[jobType] = handler
}

// Start launches the ants pool to process jobs concurrently.
func (m *Manager) Start(ctx context.Context) {
	m.runningMu.Lock()
	if m.running {
		m.runningMu.Unlock()
		return
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.runningMu.Unlock()

	// 创建 ants 协程池（带优化选项）
	pool, err := ants.NewPool(
		m.GetWorkerCount(),
		// 防止任务 panic 导致整个 pool 崩溃
		ants.WithPanicHandler(func(i interface{}) {
			logger.Errorf("[ANTS PANIC] task panicked: %v", i)
		}),
		// 空闲 goroutine 回收时间（10分钟）
		ants.WithExpiryDuration(10*time.Minute),
	)
	if err != nil {
		logger.Errorf("创建 ants 池失败: %v", err)
		m.runningMu.Lock()
		m.running = false
		m.runningMu.Unlock()
		return
	}

	m.poolMu.Lock()
	m.pool = pool
	m.poolMu.Unlock()

	m.dispatchWg.Add(1)
	go m.dispatchLoop()

	logger.Infof("启动 %d 个 worker 协程处理任务 (ants pool)", m.GetWorkerCount())
}

// Stop releases the ants pool and waits for running tasks to complete.
func (m *Manager) Stop() {
	// 关闭 stopCh
	select {
	case <-m.stopCh:
		// already closed
	default:
		close(m.stopCh)
	}
	m.dispatchWg.Wait()

	// 释放 ants 池
	m.poolMu.Lock()
	if m.pool != nil {
		m.taskWg.Wait()
		m.pool.Release()
		m.pool = nil
	}
	m.poolMu.Unlock()

	m.runningMu.Lock()
	m.running = false
	m.runningMu.Unlock()
}

// SetWorkerCount dynamically adjusts the number of workers using ants.Tune.
func (m *Manager) SetWorkerCount(ctx context.Context, newCount int) {
	if newCount < 1 {
		newCount = 1
	}

	m.poolMu.Lock()
	pool := m.pool
	m.poolMu.Unlock()

	if pool == nil {
		m.workerCount.Store(int32(newCount))
		logger.Infof("Worker 调优目标已更新为 %d (pool 未启动)", newCount)
		return
	}

	currentCount := m.GetWorkerCount()
	if newCount == currentCount {
		return
	}

	// 使用 ants.Tune 动态调整池大小
	pool.Tune(newCount)

	m.workerCount.Store(int32(newCount))
	logger.Infof("Worker 调优目标已更新为 %d", newCount)
}

// GetWorkerCount returns the current configured worker target.
func (m *Manager) GetWorkerCount() int {
	// 返回当前保存的 worker 目标值，不表示实时活跃并发数。
	return int(m.workerCount.Load())
}

// workerWithCountCheck removed - using ants pool instead

func (m *Manager) AddJob(ctx context.Context, jobType, payload string) (int64, error) {
	job := &domain.AsyncJob{
		Type:      jobType,
		Status:    "ready",
		Payload:   payload,
		Progress:  0,
		CreatedAt: time.Now(),
	}
	if err := m.jobRepo.Save(job); err != nil {
		return 0, err
	}

	if err := m.enqueueJob(context.Background(), job); err != nil {
		logger.Infof("任务 #%d 已持久化，但当前未入队: %v", job.ID, err)
		return job.ID, nil
	}

	return job.ID, nil
}

// submitJob 提交任务到 ants pool 执行
func (m *Manager) submitJob(ctx context.Context, job *domain.AsyncJob) error {
	m.poolMu.RLock()
	pool := m.pool
	m.poolMu.RUnlock()

	if pool == nil {
		return errors.New("worker pool is not running")
	}

	m.taskWg.Add(1)
	err := pool.Submit(func() {
		defer m.taskWg.Done()
		m.processJob(ctx, job)
	})
	if err != nil {
		m.taskWg.Done()
		return err
	}

	return nil
}

// LoadExistingJob 将已有的任务加载到队列中（不创建新记录）
// 使用非阻塞方式，队列满时返回 false，调用方应跳过该任务
func (m *Manager) LoadExistingJob(job *domain.AsyncJob) bool {
	return m.enqueueJob(context.Background(), job) == nil
}

func (m *Manager) enqueueJob(ctx context.Context, job *domain.AsyncJob) error {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case m.queue <- &queuedJob{ctx: ctx, job: job}:
		m.enqueueType(job.Type)
		return nil
	default:
		return errors.New("job queue is full")
	}
}

func (m *Manager) dispatchLoop() {
	defer m.dispatchWg.Done()

	var pending *queuedJob
	for {
		if pending == nil {
			select {
			case <-m.stopCh:
				atomic.StoreInt32(&m.pendingCnt, 0)
				return
			case pending = <-m.queue:
				atomic.StoreInt32(&m.pendingCnt, 1)
			}
		}

		if m.IsPaused() {
			pauseNotify := m.getPauseNotify()
			select {
			case <-m.stopCh:
				atomic.StoreInt32(&m.pendingCnt, 0)
				return
			case <-pauseNotify:
			}
			continue
		}

		if err := m.submitJob(pending.ctx, pending.job); err != nil {
			logger.Errorf("提交任务到 pool 失败，稍后重试: %v", err)
			select {
			case <-m.stopCh:
				atomic.StoreInt32(&m.pendingCnt, 0)
				return
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}

		m.dequeueType(pending.job.Type)
		pending = nil
		atomic.StoreInt32(&m.pendingCnt, 0)
	}
}

func (m *Manager) QueuedByType(jobType string) int {
	m.queueTypeMu.Lock()
	defer m.queueTypeMu.Unlock()
	return m.queueTypes[jobType]
}

func (m *Manager) enqueueType(jobType string) {
	if jobType == "" {
		return
	}
	m.queueTypeMu.Lock()
	m.queueTypes[jobType]++
	m.queueTypeMu.Unlock()
}

func (m *Manager) dequeueType(jobType string) {
	if jobType == "" {
		return
	}
	m.queueTypeMu.Lock()
	defer m.queueTypeMu.Unlock()
	if m.queueTypes[jobType] > 0 {
		m.queueTypes[jobType]--
	}
}

func (m *Manager) processJob(ctx context.Context, job *domain.AsyncJob) {
	handler, ok := m.handlers[job.Type]
	if !ok {
		errText := "no handler registered"
		job.Status = "failed"
		job.Error = &errText
		finished := time.Now()
		job.FinishedAt = &finished
		_ = m.jobRepo.Update(job)
		logger.Errorf("任务 %d 失败: 未找到处理器 %s", job.ID, job.Type)
		return
	}

	started := time.Now()
	job.Status = "running"
	job.StartedAt = &started
	job.Error = nil
	_ = m.jobRepo.Update(job)

	logger.Infof("开始执行任务: %s #%d", job.Type, job.ID)

	if err := handler(ctx, job.ID, job.Payload); err != nil {
		errText := err.Error()
		job.Status = "failed"
		job.Error = &errText
		logger.Errorf("任务 %s #%d 执行失败: %v", job.Type, job.ID, err)
	} else {
		job.Status = "finished"
		job.Progress = 100
		duration := time.Since(started)
		logger.Infof("任务 %s #%d 执行完成，耗时: %.2f秒", job.Type, job.ID, duration.Seconds())
	}

	finished := time.Now()
	job.FinishedAt = &finished
	_ = m.jobRepo.Update(job)
}

// Pause stops the worker from processing new jobs while preserving the queue.
func (m *Manager) Pause() {
	m.pausedMu.Lock()
	if m.paused {
		m.pausedMu.Unlock()
		return
	}
	m.paused = true
	m.signalPauseLocked()
	m.pausedMu.Unlock()
}

// Resume allows the worker to continue processing jobs.
func (m *Manager) Resume() {
	m.pausedMu.Lock()
	if !m.paused {
		m.pausedMu.Unlock()
		return
	}
	m.paused = false
	m.signalPauseLocked()
	m.pausedMu.Unlock()
}

// IsRunning returns true if the manager has been started and not stopped.
func (m *Manager) IsRunning() bool {
	m.runningMu.Lock()
	defer m.runningMu.Unlock()
	return m.running
}

// IsPaused returns true if the manager is paused.
func (m *Manager) IsPaused() bool {
	m.pausedMu.Lock()
	defer m.pausedMu.Unlock()
	return m.paused
}

func (m *Manager) getPauseNotify() <-chan struct{} {
	m.pausedMu.Lock()
	defer m.pausedMu.Unlock()
	return m.pauseNotify
}

func (m *Manager) signalPauseLocked() {
	close(m.pauseNotify)
	m.pauseNotify = make(chan struct{})
}

// QueueSize returns the current number of jobs waiting in the queue.
func (m *Manager) QueueSize() int {
	return len(m.queue) + int(atomic.LoadInt32(&m.pendingCnt))
}

// QueueCapacity returns the maximum queue size.
func (m *Manager) QueueCapacity() int {
	return m.queueSize
}

// GetStats returns runtime statistics about the job manager.
func (m *Manager) GetStats() Stats {
	m.runningMu.Lock()
	running := m.running
	m.runningMu.Unlock()

	m.pausedMu.Lock()
	paused := m.paused
	m.pausedMu.Unlock()

	workerCount := m.GetWorkerCount()

	// 获取队列类型统计
	m.queueTypeMu.Lock()
	jobTypes := make(map[string]int)
	for k, v := range m.queueTypes {
		jobTypes[k] = v
	}
	m.queueTypeMu.Unlock()

	return Stats{
		QueueSize:   m.QueueSize(),
		IsRunning:   running,
		IsPaused:    paused,
		WorkerCount: workerCount,
		JobTypes:    jobTypes,
	}
}
