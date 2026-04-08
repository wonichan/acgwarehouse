package service

import (
	"testing"
)

func TestDuplicateTaskService_CreateGetSubscribe(t *testing.T) {
	t.Parallel()

	svc := NewDuplicateTaskService()
	task := svc.CreateTask()

	if task.TaskID == "" {
		t.Fatal("TaskID is empty")
	}
	if task.Status != DuplicateTaskStatusQueued {
		t.Fatalf("status = %q, want %q", task.Status, DuplicateTaskStatusQueued)
	}

	got, ok := svc.GetTask(task.TaskID)
	if !ok {
		t.Fatal("GetTask() ok = false, want true")
	}
	if got.TaskID != task.TaskID {
		t.Fatalf("GetTask().TaskID = %q, want %q", got.TaskID, task.TaskID)
	}

	ch, unsubscribe, ok := svc.Subscribe(task.TaskID)
	if !ok {
		t.Fatal("Subscribe() ok = false, want true")
	}
	defer unsubscribe()

	initial := <-ch
	if initial.Status != DuplicateTaskStatusQueued {
		t.Fatalf("initial status = %q, want queued", initial.Status)
	}
}

func TestDuplicateTaskService_MonotonicProgressAndTerminalState(t *testing.T) {
	t.Parallel()

	svc := NewDuplicateTaskService()
	task := svc.CreateTask()

	if _, ok := svc.UpdateTask(task.TaskID, func(s *DuplicateTaskSnapshot) {
		s.Status = DuplicateTaskStatusPreparing
		s.Progress = 15
		s.Processed = 2
		s.Total = 10
		s.Message = "preparing"
	}); !ok {
		t.Fatal("UpdateTask #1 failed")
	}

	updated, ok := svc.UpdateTask(task.TaskID, func(s *DuplicateTaskSnapshot) {
		s.Status = DuplicateTaskStatusHashing
		s.Progress = 10 // should be clamped to 15
		s.Processed = 1 // should be clamped to 2
		s.Total = 9     // should be clamped to 10
		s.Message = "hashing"
	})
	if !ok {
		t.Fatal("UpdateTask #2 failed")
	}

	if updated.Progress != 15 {
		t.Fatalf("progress = %.1f, want 15.0", updated.Progress)
	}
	if updated.Processed != 2 {
		t.Fatalf("processed = %d, want 2", updated.Processed)
	}
	if updated.Total != 10 {
		t.Fatalf("total = %d, want 10", updated.Total)
	}

	terminal, ok := svc.UpdateTask(task.TaskID, func(s *DuplicateTaskSnapshot) {
		s.Status = DuplicateTaskStatusCompleted
		s.Progress = 100
		s.Processed = 10
		s.Total = 10
		s.GroupsFound = 3
		s.Message = "completed"
	})
	if !ok {
		t.Fatal("UpdateTask terminal failed")
	}

	if terminal.CompletedAt == nil {
		t.Fatal("CompletedAt is nil for terminal state")
	}
}
