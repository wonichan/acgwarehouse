package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type JobFunc func(ctx context.Context, id int64, payload string) error

// Stats holds runtime statistics about the job manager.
type Stats struct {
	QueueSize   int
	IsRunning   bool
	IsPaused    bool
	WorkerCount int
	JobTypes    map[string]int
}

type Manager struct {
	jobRepo   repository.JobRepository
	handlers  map[string]JobFunc
	queue     chan *domain.AsyncJob
	queueSize int
	stopOnce  sync.Once
	stopCh    chan struct{}
	runningMu sync.Mutex
	running   bool
	pausedMu  sync.Mutex
	paused    bool

	// Worker pool management
	workerCountMu sync.Mutex
	workerCount   int
	workerWg      sync.WaitGroup
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
	return &Manager{
		jobRepo:     jobRepo,
		handlers:    make(map[string]JobFunc),
		queue:       make(chan *domain.AsyncJob, queueSize),
		queueSize:   queueSize,
		workerCount: workerCount,
		stopCh:      make(chan struct{}),
	}
}

func (m *Manager) RegisterHandler(jobType string, handler JobFunc) {
	m.handlers[jobType] = handler
}

// Start launches multiple worker goroutines to process jobs concurrently.
func (m *Manager) Start(ctx context.Context) {
	m.runningMu.Lock()
	if m.running {
		m.runningMu.Unlock()
		return
	}
	m.running = true
	m.runningMu.Unlock()

	// 启动 worker 协程池
	m.workerCountMu.Lock()
	for i := 0; i < m.workerCount; i++ {
		m.workerWg.Add(1)
		go m.workerWithCountCheck(ctx, i)
	}
	workerCount := m.workerCount
	m.workerCountMu.Unlock()

	log.Printf("启动 %d 个 worker 协程处理任务", workerCount)
}

func (m *Manager) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
	m.workerWg.Wait() // 等待所有 worker 退出
	m.runningMu.Lock()
	m.running = false
	m.runningMu.Unlock()
}

// SetWorkerCount dynamically adjusts the number of workers.
// It can add or remove workers at runtime.
// Note: Reducing worker count is a graceful process - excess workers will
// finish their current task before exiting.
func (m *Manager) SetWorkerCount(ctx context.Context, newCount int) {
	if newCount < 1 {
		newCount = 1
	}

	m.workerCountMu.Lock()
	defer m.workerCountMu.Unlock()

	currentCount := m.workerCount
	if newCount == currentCount {
		return
	}

	if newCount > currentCount {
		// Add new workers
		for i := currentCount; i < newCount; i++ {
			m.workerWg.Add(1)
			go m.workerWithCountCheck(ctx, i)
		}
		m.workerCount = newCount
		log.Printf("Worker 数量已调整为 %d (增加 %d)", newCount, newCount-currentCount)
	} else {
		// Reducing workers: just update the count
		// Workers will check and exit if their index >= newCount
		m.workerCount = newCount
		log.Printf("Worker 数量将调整为 %d (减少 %d)，多余的 worker 完成当前任务后退出", newCount, currentCount-newCount)
	}
}

// workerWithCountCheck is a worker that checks if it should still be running.
func (m *Manager) workerWithCountCheck(ctx context.Context, id int) {
	defer m.workerWg.Done()

	for {
		// Check if this worker should still be running
		m.workerCountMu.Lock()
		shouldRun := id < m.workerCount
		m.workerCountMu.Unlock()

		if !shouldRun {
			log.Printf("Worker #%d 退出（数量已减少）", id)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case job := <-m.queue:
			if job == nil {
				continue
			}

			// Check if paused
			m.pausedMu.Lock()
			isPaused := m.paused
			m.pausedMu.Unlock()

			if isPaused {
				// Re-queue the job and wait
				go func(j *domain.AsyncJob) {
					time.Sleep(100 * time.Millisecond)
					select {
					case m.queue <- j:
					default:
						// Queue full, drop job (it's persisted in DB)
					}
				}(job)
				continue
			}
			m.processJob(ctx, job)
		}
	}
}

// GetWorkerCount returns the current number of workers.
func (m *Manager) GetWorkerCount() int {
	m.workerCountMu.Lock()
	defer m.workerCountMu.Unlock()
	return m.workerCount
}

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

	// 非阻塞方式入队列，避免在 worker 执行时死锁
	select {
	case m.queue <- job:
		return job.ID, nil
	default:
		// 队列已满，任务已存入数据库，等待后续加载
		log.Printf("队列已满，任务 #%d 已保存到数据库，等待后续处理", job.ID)
		return job.ID, nil
	}
}

// LoadExistingJob 将已有的任务加载到队列中（不创建新记录）
// 使用非阻塞方式，队列满时返回 false，调用方应跳过该任务
func (m *Manager) LoadExistingJob(job *domain.AsyncJob) bool {
	select {
	case m.queue <- job:
		return true
	default:
		return false
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
		log.Printf("任务 %d 失败: 未找到处理器 %s", job.ID, job.Type)
		return
	}

	started := time.Now()
	job.Status = "running"
	job.StartedAt = &started
	job.Error = nil
	_ = m.jobRepo.Update(job)

	log.Printf("开始执行任务: %s #%d", job.Type, job.ID)

	if err := handler(ctx, job.ID, job.Payload); err != nil {
		errText := err.Error()
		job.Status = "failed"
		job.Error = &errText
		log.Printf("任务 %s #%d 执行失败: %v", job.Type, job.ID, err)
	} else {
		job.Status = "finished"
		job.Progress = 100
		duration := time.Since(started)
		log.Printf("任务 %s #%d 执行完成，耗时: %.2f秒", job.Type, job.ID, duration.Seconds())
	}

	finished := time.Now()
	job.FinishedAt = &finished
	_ = m.jobRepo.Update(job)
}

// Pause stops the worker from processing new jobs while preserving the queue.
func (m *Manager) Pause() {
	m.pausedMu.Lock()
	m.paused = true
	m.pausedMu.Unlock()
}

// Resume allows the worker to continue processing jobs.
func (m *Manager) Resume() {
	m.pausedMu.Lock()
	m.paused = false
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

// QueueSize returns the current number of jobs waiting in the queue.
func (m *Manager) QueueSize() int {
	return len(m.queue)
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

	m.workerCountMu.Lock()
	workerCount := m.workerCount
	m.workerCountMu.Unlock()

	return Stats{
		QueueSize:   len(m.queue),
		IsRunning:   running,
		IsPaused:    paused,
		WorkerCount: workerCount,
		JobTypes:    nil,
	}
}
