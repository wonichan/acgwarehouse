package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type monitoringEventBusAdminServiceStub struct {
	overview *TaskPlatformOverview
	err      error
	calls    int
}

func (s *monitoringEventBusAdminServiceStub) GetTaskPlatformOverview(ctx context.Context) (*TaskPlatformOverview, error) {
	s.calls++
	return s.overview, s.err
}

func TestNewMonitoringEventBusInitializesWithoutSubscribers(t *testing.T) {
	t.Parallel()

	bus := NewMonitoringEventBus(&monitoringEventBusAdminServiceStub{})
	if bus == nil {
		t.Fatal("NewMonitoringEventBus() returned nil")
	}
	if got := len(bus.subscribers); got != 0 {
		t.Fatalf("len(subscribers) = %d, want 0", got)
	}
}

func TestMonitoringEventBusSubscribeAndUnsubscribeClosesChannel(t *testing.T) {
	t.Parallel()

	bus := NewMonitoringEventBus(&monitoringEventBusAdminServiceStub{})
	ch, unsubscribe := bus.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe() returned nil channel")
	}
	if got := len(bus.subscribers); got != 1 {
		t.Fatalf("len(subscribers) after subscribe = %d, want 1", got)
	}

	unsubscribe()

	if got := len(bus.subscribers); got != 0 {
		t.Fatalf("len(subscribers) after unsubscribe = %d, want 0", got)
	}
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("subscriber channel should be closed after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscriber channel to close")
	}
}

func TestMonitoringEventBusBroadcastFansOutToAllSubscribers(t *testing.T) {
	t.Parallel()

	bus := NewMonitoringEventBus(&monitoringEventBusAdminServiceStub{})
	first, firstUnsubscribe := bus.Subscribe()
	defer firstUnsubscribe()
	second, secondUnsubscribe := bus.Subscribe()
	defer secondUnsubscribe()

	event := MonitoringEvent{
		Type:      "overview",
		Payload:   json.RawMessage(`{"queue":{"queue_size":2}}`),
		Timestamp: time.Now().UTC().Round(0),
	}

	bus.Broadcast(event)

	assertMonitoringEventReceived(t, first, event)
	assertMonitoringEventReceived(t, second, event)
}

func TestMonitoringEventBusDoesNotSendToUnsubscribedClients(t *testing.T) {
	t.Parallel()

	bus := NewMonitoringEventBus(&monitoringEventBusAdminServiceStub{})
	former, unsubscribe := bus.Subscribe()
	active, activeUnsubscribe := bus.Subscribe()
	defer activeUnsubscribe()

	unsubscribe()

	event := MonitoringEvent{
		Type:      "overview",
		Payload:   json.RawMessage(`{"tasks":{"running":1}}`),
		Timestamp: time.Now().UTC().Round(0),
	}

	bus.Broadcast(event)

	assertMonitoringEventReceived(t, active, event)
	select {
	case _, ok := <-former:
		if ok {
			t.Fatal("former subscriber should not receive events after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for former subscriber channel to remain closed")
	}
}

func TestMonitoringEventBusStartBroadcastsOverviewSnapshots(t *testing.T) {
	t.Parallel()

	adminSvc := &monitoringEventBusAdminServiceStub{
		overview: &TaskPlatformOverview{
			Queue: QueueOverview{IsRunning: true, QueueSize: 3, WorkerCount: 2},
			Tasks: map[string]int64{"running": 4},
		},
	}
	bus := NewMonitoringEventBus(adminSvc)
	ch, unsubscribe := bus.Subscribe()
	defer unsubscribe()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	bus.Start(ctx, 10*time.Millisecond)
	defer bus.Stop()

	select {
	case event := <-ch:
		if event.Type != "overview" {
			t.Fatalf("event.Type = %q, want %q", event.Type, "overview")
		}
		if event.Timestamp.IsZero() {
			t.Fatal("event.Timestamp should be set")
		}
		var payload TaskPlatformOverview
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("json.Unmarshal(event.Payload) error = %v", err)
		}
		if payload.Queue.QueueSize != 3 {
			t.Fatalf("payload.Queue.QueueSize = %d, want 3", payload.Queue.QueueSize)
		}
		if payload.Tasks["running"] != 4 {
			t.Fatalf("payload.Tasks[running] = %d, want 4", payload.Tasks["running"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for overview event")
	}

	if adminSvc.calls == 0 {
		t.Fatal("GetTaskPlatformOverview should be called at least once")
	}
}

func assertMonitoringEventReceived(t *testing.T, ch <-chan MonitoringEvent, want MonitoringEvent) {
	t.Helper()

	select {
	case got := <-ch:
		if got.Type != want.Type {
			t.Fatalf("event.Type = %q, want %q", got.Type, want.Type)
		}
		if string(got.Payload) != string(want.Payload) {
			t.Fatalf("event.Payload = %s, want %s", got.Payload, want.Payload)
		}
		if !got.Timestamp.Equal(want.Timestamp) {
			t.Fatalf("event.Timestamp = %s, want %s", got.Timestamp, want.Timestamp)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for monitoring event")
	}
}
