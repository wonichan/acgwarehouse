package worker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const defaultAITagBatchWaitWindow = 100 * time.Millisecond

type aiTagBatchItem struct {
	JobID   int64
	Payload AITagPayload
}

type aiTagBatchProcessor func(ctx context.Context, items []aiTagBatchItem) []error

type aiTagBatchCoordinator struct {
	mu           sync.Mutex
	maxBatchSize int
	waitWindow   time.Duration
	pending      []*aiTagBatchSubmission
	timer        *time.Timer
}

type aiTagBatchSubmission struct {
	item aiTagBatchItem
	err  error
	done chan struct{}
}

func newAITagBatchCoordinator(maxBatchSize int, waitWindow time.Duration) *aiTagBatchCoordinator {
	if maxBatchSize <= 0 {
		maxBatchSize = AiTagBatchSize
	}
	if waitWindow <= 0 {
		waitWindow = defaultAITagBatchWaitWindow
	}
	return &aiTagBatchCoordinator{
		maxBatchSize: maxBatchSize,
		waitWindow:   waitWindow,
	}
}

func NewAITagBatchCoordinator(maxBatchSize int, waitWindow time.Duration) *aiTagBatchCoordinator {
	return newAITagBatchCoordinator(maxBatchSize, waitWindow)
}

func (c *aiTagBatchCoordinator) Submit(ctx context.Context, item aiTagBatchItem, processor aiTagBatchProcessor) error {
	if processor == nil {
		return fmt.Errorf("ai tag batch processor is nil")
	}
	submission := &aiTagBatchSubmission{
		item: item,
		done: make(chan struct{}),
	}

	var batch []*aiTagBatchSubmission
	c.mu.Lock()
	c.pending = append(c.pending, submission)
	if len(c.pending) == 1 {
		c.startTimerLocked(processor)
	}
	if len(c.pending) >= c.maxBatchSize {
		batch = c.takePendingLocked()
	}
	c.mu.Unlock()

	if len(batch) > 0 {
		c.processBatch(batch, processor)
	}

	select {
	case <-submission.done:
		return submission.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *aiTagBatchCoordinator) startTimerLocked(processor aiTagBatchProcessor) {
	c.timer = time.AfterFunc(c.waitWindow, func() {
		c.flush(processor)
	})
}

func (c *aiTagBatchCoordinator) flush(processor aiTagBatchProcessor) {
	var batch []*aiTagBatchSubmission
	c.mu.Lock()
	batch = c.takePendingLocked()
	c.mu.Unlock()
	if len(batch) == 0 {
		return
	}
	c.processBatch(batch, processor)
}

func (c *aiTagBatchCoordinator) takePendingLocked() []*aiTagBatchSubmission {
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	if len(c.pending) == 0 {
		return nil
	}
	batch := append([]*aiTagBatchSubmission(nil), c.pending...)
	c.pending = nil
	return batch
}

func (c *aiTagBatchCoordinator) processBatch(batch []*aiTagBatchSubmission, processor aiTagBatchProcessor) {
	items := make([]aiTagBatchItem, len(batch))
	for i := range batch {
		items[i] = batch[i].item
	}

	var errs []error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err := fmt.Errorf("ai tag batch processor panic: %v", recovered)
				errs = make([]error, len(batch))
				for i := range errs {
					errs[i] = err
				}
			}
		}()
		errs = processor(context.Background(), items)
	}()

	if len(errs) != len(batch) {
		mismatchErr := fmt.Errorf("ai tag batch processor returned %d results for %d items", len(errs), len(batch))
		normalized := make([]error, len(batch))
		for i := range normalized {
			normalized[i] = mismatchErr
		}
		errs = normalized
	}

	for i := range batch {
		batch[i].err = errs[i]
		close(batch[i].done)
	}
}
