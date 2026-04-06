package sidecar

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRuntimeLifecycleTransitionsToReadyOnProbeSuccess(t *testing.T) {
	t.Parallel()

	process := newFakeProcess()
	runtime := NewRuntime(RuntimeConfig{
		StartupTimeout: 200 * time.Millisecond,
		ProbeInterval:  10 * time.Millisecond,
		CommandFactory: func(context.Context) (Process, error) {
			return process, nil
		},
		Probe: func(context.Context) error {
			return nil
		},
		ShutdownProbe: func(context.Context) error {
			return nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got := runtime.State(); got != StateReady {
		t.Fatalf("State() = %q, want %q", got, StateReady)
	}
}

func TestRuntimeStartupTimeoutTransitionsToDegradedWithError(t *testing.T) {
	t.Parallel()

	process := newFakeProcess()
	runtime := NewRuntime(RuntimeConfig{
		StartupTimeout: 60 * time.Millisecond,
		ProbeInterval:  10 * time.Millisecond,
		CommandFactory: func(context.Context) (Process, error) {
			return process, nil
		},
		Probe: func(context.Context) error {
			return errors.New("sidecar not ready")
		},
		ShutdownProbe: func(context.Context) error {
			return nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runtime.Start(ctx)
	if err == nil {
		t.Fatal("Start() error = nil, want timeout error")
	}
	if got := runtime.State(); got != StateDegraded {
		t.Fatalf("State() = %q, want %q", got, StateDegraded)
	}
	if summary := runtime.Status().LastError; !strings.Contains(summary, "startup timeout") {
		t.Fatalf("Status().LastError = %q, want startup timeout summary", summary)
	}
}

func TestRuntimeShutdownFallsBackToKillAndWaitsForProcess(t *testing.T) {
	t.Parallel()

	process := newFakeProcess()
	runtime := NewRuntime(RuntimeConfig{
		StartupTimeout: 200 * time.Millisecond,
		ProbeInterval:  10 * time.Millisecond,
		CommandFactory: func(context.Context) (Process, error) {
			return process, nil
		},
		Probe: func(context.Context) error {
			return nil
		},
		ShutdownProbe: func(context.Context) error {
			return errors.New("graceful shutdown endpoint timeout")
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer stopCancel()

	if err := runtime.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if got := runtime.State(); got != StateStopped {
		t.Fatalf("State() = %q, want %q", got, StateStopped)
	}
	if process.KillCalls() != 1 {
		t.Fatalf("Kill() calls = %d, want 1", process.KillCalls())
	}
	if process.WaitCalls() != 1 {
		t.Fatalf("Wait() calls = %d, want 1", process.WaitCalls())
	}
}

func TestRuntimeShutdownKillsProcessWhenWaitTimesOutAfterSuccessfulProbe(t *testing.T) {
	t.Parallel()

	process := newFakeProcess()
	runtime := NewRuntime(RuntimeConfig{
		StartupTimeout: 200 * time.Millisecond,
		ProbeInterval:  10 * time.Millisecond,
		CommandFactory: func(context.Context) (Process, error) {
			return process, nil
		},
		Probe: func(context.Context) error {
			return nil
		},
		ShutdownProbe: func(context.Context) error {
			return nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stopCancel()

	if err := runtime.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v, want nil after forced kill", err)
	}

	if process.KillCalls() != 1 {
		t.Fatalf("Kill() calls = %d, want 1", process.KillCalls())
	}
	if process.WaitCalls() != 1 {
		t.Fatalf("Wait() calls = %d, want 1", process.WaitCalls())
	}
}

type fakeProcess struct {
	mu        sync.Mutex
	killCalls int
	waitCalls int
	waitCh    chan struct{}
	waitErr   error
}

func newFakeProcess() *fakeProcess {
	return &fakeProcess{waitCh: make(chan struct{})}
}

func (f *fakeProcess) Kill() error {
	f.mu.Lock()
	f.killCalls++
	select {
	case <-f.waitCh:
		// already closed
	default:
		close(f.waitCh)
	}
	f.mu.Unlock()
	return nil
}

func (f *fakeProcess) Wait() error {
	f.mu.Lock()
	f.waitCalls++
	waitCh := f.waitCh
	waitErr := f.waitErr
	f.mu.Unlock()
	<-waitCh
	return waitErr
}

func (f *fakeProcess) KillCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.killCalls
}

func (f *fakeProcess) WaitCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.waitCalls
}
