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
	QueueSize int
	IsRunning bool
	IsPaused  bool
	JobTypes  map[string]int
}

type Manager struct {
	jobRepo   repository.JobRepository
	handlers  map[string]JobFunc
	queue     chan *domain.AsyncJob
	stopOnce  sync.Once
	stopCh    chan struct{}
	runningMu sync.Mutex
	running   bool
	pausedMu  sync.Mutex
	paused    bool
}

func NewManager(jobRepo repository.JobRepository) *Manager {
	return &Manager{
		jobRepo:  jobRepo,
		handlers: make(map[string]JobFunc),
		queue:    make(chan *domain.AsyncJob, 128),
		stopCh:   make(chan struct{}),
	}
}

func (m *Manager) RegisterHandler(jobType string, handler JobFunc) {
	m.handlers[jobType] = handler
}

func (m *Manager) Start(ctx context.Context) {
	m.runningMu.Lock()
	if m.running {
		m.runningMu.Unlock()
		return
	}
	m.running = true
	m.runningMu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-m.stopCh:
				return
			case job := <-m.queue:
				if job != nil {
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
	}()
}

func (m *Manager) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
	m.runningMu.Lock()
	m.running = false
	m.runningMu.Unlock()
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

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-m.stopCh:
		return 0, context.Canceled
	case m.queue <- job:
		return job.ID, nil
	}
}

// LoadExistingJob 将已有的任务加载到队列中（不创建新记录）
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

// GetStats returns runtime statistics about the job manager.
func (m *Manager) GetStats() Stats {
	m.runningMu.Lock()
	running := m.running
	m.runningMu.Unlock()

	m.pausedMu.Lock()
	paused := m.paused
	m.pausedMu.Unlock()

	return Stats{
		QueueSize: len(m.queue),
		IsRunning: running,
		IsPaused:  paused,
		JobTypes:  nil, // Not tracking job type counts for now
	}
}
