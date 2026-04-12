package worker

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestAITagBatchCoordinator_FlushesAtFour(t *testing.T) {
	coordinator := newAITagBatchCoordinator(4, 50*time.Millisecond)

	var (
		mu      sync.Mutex
		batches [][]aiTagBatchItem
	)
	processor := func(ctx context.Context, items []aiTagBatchItem) []error {
		mu.Lock()
		defer mu.Unlock()
		copied := append([]aiTagBatchItem(nil), items...)
		batches = append(batches, copied)
		return make([]error, len(items))
	}

	var wg sync.WaitGroup
	for i := int64(1); i <= 4; i++ {
		wg.Add(1)
		go func(jobID int64) {
			defer wg.Done()
			err := coordinator.Submit(context.Background(), aiTagBatchItem{
				JobID: jobID,
				Payload: AITagPayload{
					ImageID: jobID,
					Path:    "/images/test.png",
				},
			}, processor)
			if err != nil {
				t.Errorf("Submit() error = %v", err)
			}
		}(i)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(batches) != 1 {
		t.Fatalf("expected 1 flushed batch, got %d", len(batches))
	}
	if len(batches[0]) != 4 {
		t.Fatalf("expected batch size 4, got %d", len(batches[0]))
	}
}

func TestAITagBatchCoordinator_FlushesPartialBatchAfterWaitWindow(t *testing.T) {
	coordinator := newAITagBatchCoordinator(4, 10*time.Millisecond)

	var (
		mu      sync.Mutex
		batches [][]aiTagBatchItem
	)
	processor := func(ctx context.Context, items []aiTagBatchItem) []error {
		mu.Lock()
		defer mu.Unlock()
		copied := append([]aiTagBatchItem(nil), items...)
		batches = append(batches, copied)
		return make([]error, len(items))
	}

	var wg sync.WaitGroup
	for i := int64(1); i <= 2; i++ {
		wg.Add(1)
		go func(jobID int64) {
			defer wg.Done()
			err := coordinator.Submit(context.Background(), aiTagBatchItem{
				JobID: jobID,
				Payload: AITagPayload{
					ImageID: jobID,
					Path:    "/images/test.png",
				},
			}, processor)
			if err != nil {
				t.Errorf("Submit() error = %v", err)
			}
		}(i)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(batches) != 1 {
		t.Fatalf("expected 1 flushed batch, got %d", len(batches))
	}
	if len(batches[0]) != 2 {
		t.Fatalf("expected partial batch size 2, got %d", len(batches[0]))
	}
}
