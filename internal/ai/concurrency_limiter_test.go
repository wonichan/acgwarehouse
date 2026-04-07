package ai

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrencyLimiter_Basic(t *testing.T) {
	limiter := NewConcurrencyLimiter(3)

	if limiter.Capacity() != 3 {
		t.Errorf("expected capacity 3, got %d", limiter.Capacity())
	}
	if limiter.Available() != 3 {
		t.Errorf("expected available 3, got %d", limiter.Available())
	}
}

func TestConcurrencyLimiter_AcquireAndRelease(t *testing.T) {
	limiter := NewConcurrencyLimiter(2)

	// 获取第一个槽位
	release1, err := limiter.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire 1 failed: %v", err)
	}
	if limiter.Available() != 1 {
		t.Errorf("expected available 1, got %d", limiter.Available())
	}

	// 获取第二个槽位
	release2, err := limiter.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire 2 failed: %v", err)
	}
	if limiter.Available() != 0 {
		t.Errorf("expected available 0, got %d", limiter.Available())
	}

	// 尝试非阻塞获取，应该失败
	_, acquired := limiter.TryAcquire()
	if acquired {
		t.Error("expected TryAcquire to fail when limiter is full")
	}

	// 释放一个槽位
	release1()
	if limiter.Available() != 1 {
		t.Errorf("expected available 1 after release, got %d", limiter.Available())
	}

	// 现在应该可以获取
	_, acquired = limiter.TryAcquire()
	if !acquired {
		t.Error("expected TryAcquire to succeed after release")
	}

	release2()
}

func TestConcurrencyLimiter_ConcurrentLimit(t *testing.T) {
	limiter := NewConcurrencyLimiter(3)

	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32
	var wg sync.WaitGroup

	// 启动 10 个 goroutine，验证最多同时只有 3 个在执行
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			release, err := limiter.Acquire(context.Background())
			if err != nil {
				t.Errorf("acquire failed: %v", err)
				return
			}
			defer release()

			// 增加当前计数
			curr := currentConcurrent.Add(1)
			defer currentConcurrent.Add(-1)

			// 更新最大并发数
			for {
				max := maxConcurrent.Load()
				if curr <= max || maxConcurrent.CompareAndSwap(max, curr) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
		}()
	}

	wg.Wait()

	if max := maxConcurrent.Load(); max != 3 {
		t.Errorf("expected max concurrent 3, got %d", max)
	}
}

func TestConcurrencyLimiter_ContextCancellation(t *testing.T) {
	limiter := NewConcurrencyLimiter(1)

	// 先占满槽位
	release, _ := limiter.Acquire(context.Background())
	defer release()

	// 创建一个会超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	// 尝试获取，应该超时
	start := time.Now()
	_, err := limiter.Acquire(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected error from Acquire with cancelled context")
	}
	if elapsed < 20*time.Millisecond {
		t.Errorf("expected to wait for timeout, but only waited %v", elapsed)
	}
}
