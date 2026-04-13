package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/logger"
)

const monitoringEventBusSubscriberBuffer = 16

type monitoringEventBusOverviewService interface {
	GetTaskPlatformOverview(ctx context.Context) (*TaskPlatformOverview, error)
}

type MonitoringEvent struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

type MonitoringEventBus struct {
	mu          sync.Mutex
	subscribers map[int]chan MonitoringEvent
	nextID      int
	adminSvc    monitoringEventBusOverviewService
	stopCh      chan struct{}
	running     bool
}

func NewMonitoringEventBus(adminSvc monitoringEventBusOverviewService) *MonitoringEventBus {
	return &MonitoringEventBus{
		subscribers: make(map[int]chan MonitoringEvent),
		adminSvc:    adminSvc,
	}
}

func (b *MonitoringEventBus) Subscribe() (<-chan MonitoringEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++
	ch := make(chan MonitoringEvent, monitoringEventBusSubscriberBuffer)
	b.subscribers[id] = ch

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		subscriber, ok := b.subscribers[id]
		if !ok {
			return
		}
		delete(b.subscribers, id)
		close(subscriber)
	}

	return ch, unsubscribe
}

func (b *MonitoringEventBus) Broadcast(event MonitoringEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (b *MonitoringEventBus) Start(ctx context.Context, interval time.Duration) {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	logger.Infof("[service] MonitoringEventBus started: interval=%v", interval)
	b.stopCh = make(chan struct{})
	stopCh := b.stopCh
	b.mu.Unlock()

	go func() {
		defer func() {
			b.mu.Lock()
			b.running = false
			if b.stopCh == stopCh {
				b.stopCh = nil
			}
			b.mu.Unlock()
		}()

		if interval <= 0 {
			interval = time.Second
		}

		b.publishOverview(ctx)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				b.publishOverview(ctx)
			}
		}
	}()
}

func (b *MonitoringEventBus) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	logger.Infof("[service] MonitoringEventBus stopped")

	if b.stopCh == nil {
		return
	}
	close(b.stopCh)
	b.stopCh = nil
	b.running = false
}

func (b *MonitoringEventBus) publishOverview(ctx context.Context) {
	if b.adminSvc == nil {
		return
	}

	overview, err := b.adminSvc.GetTaskPlatformOverview(ctx)
	if err != nil || overview == nil {
		return
	}

	payload, err := json.Marshal(overview)
	if err != nil {
		return
	}

	b.Broadcast(MonitoringEvent{
		Type:      "overview",
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	})
}
