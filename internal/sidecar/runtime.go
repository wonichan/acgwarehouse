package sidecar

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type State string

const (
	StateNotStarted State = "not_started"
	StateStarting   State = "starting"
	StateReady      State = "ready"
	StateDegraded   State = "degraded"
	StateStopping   State = "stopping"
	StateStopped    State = "stopped"
)

type Process interface {
	Kill() error
	Wait() error
}

type RuntimeConfig struct {
	StartupTimeout time.Duration
	ProbeInterval  time.Duration
	CommandFactory func(ctx context.Context) (Process, error)
	Probe          func(ctx context.Context) error
	ShutdownProbe  func(ctx context.Context) error
}

type Status struct {
	State     State
	LastError string
}

type Runtime struct {
	mu        sync.RWMutex
	state     State
	lastError string

	cfg     RuntimeConfig
	process Process
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	if cfg.StartupTimeout <= 0 {
		cfg.StartupTimeout = 5 * time.Second
	}
	if cfg.ProbeInterval <= 0 {
		cfg.ProbeInterval = 100 * time.Millisecond
	}

	return &Runtime{cfg: cfg, state: StateNotStarted}
}

func (r *Runtime) State() State {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

func (r *Runtime) Status() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return Status{State: r.state, LastError: r.lastError}
}

func (r *Runtime) Start(ctx context.Context) error {
	r.mu.Lock()
	r.state = StateStarting
	r.lastError = ""
	r.mu.Unlock()

	if r.cfg.CommandFactory == nil {
		return r.transitionToDegraded(errors.New("sidecar command factory is not configured"))
	}
	if r.cfg.Probe == nil {
		return r.transitionToDegraded(errors.New("sidecar probe is not configured"))
	}

	proc, err := r.cfg.CommandFactory(ctx)
	if err != nil {
		return r.transitionToDegraded(fmt.Errorf("start sidecar process: %w", err))
	}

	r.mu.Lock()
	r.process = proc
	r.mu.Unlock()

	startupCtx, cancel := context.WithTimeout(ctx, r.cfg.StartupTimeout)
	defer cancel()

	ticker := time.NewTicker(r.cfg.ProbeInterval)
	defer ticker.Stop()

	for {
		if err := r.cfg.Probe(startupCtx); err == nil {
			r.mu.Lock()
			r.state = StateReady
			r.mu.Unlock()
			return nil
		}

		select {
		case <-startupCtx.Done():
			_ = r.forceKillAndReap(context.Background())
			return r.transitionToDegraded(fmt.Errorf("startup timeout: %w", startupCtx.Err()))
		case <-ticker.C:
		}
	}
}

func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	if r.state == StateStopped || r.state == StateNotStarted {
		r.mu.Unlock()
		return nil
	}
	r.state = StateStopping
	r.mu.Unlock()

	err := r.terminateProcess(ctx)

	r.mu.Lock()
	r.state = StateStopped
	r.mu.Unlock()

	return err
}

func (r *Runtime) transitionToDegraded(err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = StateDegraded
	r.lastError = err.Error()
	return err
}

func (r *Runtime) terminateProcess(ctx context.Context) error {
	r.mu.RLock()
	proc := r.process
	shutdownProbe := r.cfg.ShutdownProbe
	r.mu.RUnlock()

	if proc == nil {
		return nil
	}

	shouldKill := shutdownProbe == nil
	if shutdownProbe != nil {
		if err := shutdownProbe(ctx); err != nil {
			shouldKill = true
		}
	}

	if shouldKill {
		if err := proc.Kill(); err != nil {
			return err
		}
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- proc.Wait()
	}()

	select {
	case err := <-waitDone:
		return err
	case <-ctx.Done():
		if killErr := proc.Kill(); killErr != nil {
			return killErr
		}
		select {
		case err := <-waitDone:
			return err
		case <-time.After(200 * time.Millisecond):
			return ctx.Err()
		}
	}
}

func (r *Runtime) forceKillAndReap(ctx context.Context) error {
	r.mu.RLock()
	proc := r.process
	r.mu.RUnlock()

	if proc == nil {
		return nil
	}

	if err := proc.Kill(); err != nil {
		return err
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- proc.Wait()
	}()

	select {
	case err := <-waitDone:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
