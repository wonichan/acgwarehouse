package worker

import (
	"context"
	"sync"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type JobFunc func(ctx context.Context, id int64, payload string) error

type Manager struct {
	jobRepo   repository.JobRepository
	handlers  map[string]JobFunc
	queue     chan *domain.AsyncJob
	stopOnce  sync.Once
	stopCh    chan struct{}
	runningMu sync.Mutex
	running   bool
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

func (m *Manager) processJob(ctx context.Context, job *domain.AsyncJob) {
	handler, ok := m.handlers[job.Type]
	if !ok {
		errText := "no handler registered"
		job.Status = "failed"
		job.Error = &errText
		finished := time.Now()
		job.FinishedAt = &finished
		_ = m.jobRepo.Update(job)
		return
	}

	started := time.Now()
	job.Status = "running"
	job.StartedAt = &started
	job.Error = nil
	_ = m.jobRepo.Update(job)

	if err := handler(ctx, job.ID, job.Payload); err != nil {
		errText := err.Error()
		job.Status = "failed"
		job.Error = &errText
	} else {
		job.Status = "finished"
		job.Progress = 1
	}

	finished := time.Now()
	job.FinishedAt = &finished
	_ = m.jobRepo.Update(job)
}
