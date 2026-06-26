package service

import (
	"context"
	"sync"
	"time"

	pkgerrors "github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

const defaultViewBufferSize = 100

// ImageEventRepository 定义图片行为事件持久化能力。
type ImageEventRepository interface {
	CreateImageEvents(ctx context.Context, events []do.ImageEvent) error
}

// ViewBuffer 批量缓冲图片浏览事件。
type ViewBuffer struct {
	repo          ImageEventRepository
	flushSize     int
	flushInterval time.Duration
	mu            sync.Mutex
	pending       []do.ImageEvent
	stop          chan struct{}
	done          chan struct{}
}

// NewViewBuffer 创建图片浏览事件缓冲器。
func NewViewBuffer(repo ImageEventRepository, flushInterval time.Duration) *ViewBuffer {
	return &ViewBuffer{
		repo:          repo,
		flushSize:     defaultViewBufferSize,
		flushInterval: flushInterval,
		pending:       make([]do.ImageEvent, 0, defaultViewBufferSize),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
}

// Start 启动定时刷新循环。
func (b *ViewBuffer) Start(ctx context.Context) {
	go b.run(ctx)
}

// RecordView 追加浏览事件并在达到阈值时刷新。
func (b *ViewBuffer) RecordView(ctx context.Context, event do.ImageEvent) error {
	if b == nil {
		return nil
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.Value == 0 {
		event.Value = 1
	}
	b.mu.Lock()
	b.pending = append(b.pending, event)
	shouldFlush := len(b.pending) >= b.flushSize
	b.mu.Unlock()
	if shouldFlush {
		return b.Flush(ctx)
	}
	return nil
}

// Flush 将当前缓冲事件批量落库。
func (b *ViewBuffer) Flush(ctx context.Context) error {
	if b == nil {
		return nil
	}
	events := b.drain()
	if len(events) == 0 {
		return nil
	}
	if err := b.repo.CreateImageEvents(ctx, events); err != nil {
		return pkgerrors.WithMessage(err, "flush view events")
	}
	return nil
}

// Stop 停止定时循环并刷新剩余事件。
func (b *ViewBuffer) Stop(ctx context.Context) error {
	if b == nil {
		return nil
	}
	close(b.stop)
	<-b.done
	return b.Flush(ctx)
}

// run 按固定间隔刷新浏览事件。
func (b *ViewBuffer) run(ctx context.Context) {
	defer close(b.done)
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := b.Flush(ctx); err != nil {
				logger.Warn(ctx, "flush view events failed", zap.Error(err))
			}
		case <-b.stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

// drain 取出待刷新事件并清空缓冲。
func (b *ViewBuffer) drain() []do.ImageEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.pending) == 0 {
		return []do.ImageEvent{}
	}
	events := make([]do.ImageEvent, len(b.pending))
	copy(events, b.pending)
	b.pending = b.pending[:0]
	return events
}
