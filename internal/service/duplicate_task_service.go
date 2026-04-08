package service

import (
	"fmt"
	"sync"
	"time"
)

const duplicateTaskSubscriberBuffer = 16

const (
	DuplicateTaskStatusQueued     = "queued"
	DuplicateTaskStatusPreparing  = "preparing"
	DuplicateTaskStatusHashing    = "hashing"
	DuplicateTaskStatusGrouping   = "grouping"
	DuplicateTaskStatusPersisting = "persisting"
	DuplicateTaskStatusCompleted  = "completed"
	DuplicateTaskStatusFailed     = "failed"
)

type DuplicateTaskSnapshot struct {
	TaskID      string     `json:"task_id"`
	Status      string     `json:"status"`
	Progress    float64    `json:"progress"`
	Processed   int        `json:"processed"`
	Total       int        `json:"total"`
	Message     string     `json:"message"`
	Error       string     `json:"error,omitempty"`
	GroupsFound int        `json:"groups_found,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type DuplicateTaskService struct {
	mu         sync.RWMutex
	tasks      map[string]DuplicateTaskSnapshot
	subs       map[string]map[int]chan DuplicateTaskSnapshot
	nextSubID  int
	nextTaskID int64
}

func NewDuplicateTaskService() *DuplicateTaskService {
	return &DuplicateTaskService{
		tasks: make(map[string]DuplicateTaskSnapshot),
		subs:  make(map[string]map[int]chan DuplicateTaskSnapshot),
	}
}

func (s *DuplicateTaskService) CreateTask() DuplicateTaskSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextTaskID++
	taskID := fmt.Sprintf("dup-%d-%d", time.Now().UnixMilli(), s.nextTaskID)
	now := time.Now().UTC()
	snapshot := DuplicateTaskSnapshot{
		TaskID:    taskID,
		Status:    DuplicateTaskStatusQueued,
		Progress:  0,
		Processed: 0,
		Total:     0,
		Message:   "queued",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.tasks[taskID] = snapshot
	return snapshot
}

func (s *DuplicateTaskService) GetTask(taskID string) (DuplicateTaskSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, ok := s.tasks[taskID]
	return snapshot, ok
}

func (s *DuplicateTaskService) Subscribe(taskID string) (<-chan DuplicateTaskSnapshot, func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, ok := s.tasks[taskID]
	if !ok {
		return nil, func() {}, false
	}

	if _, ok := s.subs[taskID]; !ok {
		s.subs[taskID] = make(map[int]chan DuplicateTaskSnapshot)
	}

	id := s.nextSubID
	s.nextSubID++
	ch := make(chan DuplicateTaskSnapshot, duplicateTaskSubscriberBuffer)
	s.subs[taskID][id] = ch

	select {
	case ch <- snapshot:
	default:
	}

	unsubscribe := func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		taskSubs, exists := s.subs[taskID]
		if !exists {
			return
		}
		sub, exists := taskSubs[id]
		if !exists {
			return
		}
		delete(taskSubs, id)
		close(sub)
		if len(taskSubs) == 0 {
			delete(s.subs, taskID)
		}
	}

	return ch, unsubscribe, true
}

func (s *DuplicateTaskService) UpdateTask(taskID string, apply func(*DuplicateTaskSnapshot)) (DuplicateTaskSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.tasks[taskID]
	if !ok {
		return DuplicateTaskSnapshot{}, false
	}

	before := current
	apply(&current)

	if current.Progress < before.Progress {
		current.Progress = before.Progress
	}
	if current.Progress > 100 {
		current.Progress = 100
	}
	if current.Processed < before.Processed {
		current.Processed = before.Processed
	}
	if current.Total < before.Total {
		current.Total = before.Total
	}
	if current.Status == DuplicateTaskStatusCompleted || current.Status == DuplicateTaskStatusFailed {
		now := time.Now().UTC()
		if current.CompletedAt == nil {
			current.CompletedAt = &now
		}
	}
	current.UpdatedAt = time.Now().UTC()

	s.tasks[taskID] = current
	s.broadcastLocked(taskID, current)
	return current, true
}

func (s *DuplicateTaskService) broadcastLocked(taskID string, snapshot DuplicateTaskSnapshot) {
	taskSubs := s.subs[taskID]
	for _, ch := range taskSubs {
		select {
		case ch <- snapshot:
		default:
		}
	}
}
