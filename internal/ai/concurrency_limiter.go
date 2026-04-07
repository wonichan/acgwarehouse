package ai

import (
	"context"
	"sync"
)

// ConcurrencyLimiter 限制同时执行的 AI 请求数量
type ConcurrencyLimiter struct {
	active int
	limit  int
	mu     sync.Mutex
	notify chan struct{}
}

// NewConcurrencyLimiter 创建并发限制器
// maxConcurrent 同时执行的最大请求数
func NewConcurrencyLimiter(maxConcurrent int) *ConcurrencyLimiter {
	if maxConcurrent <= 0 {
		maxConcurrent = 3 // 默认 3 个并发
	}
	return &ConcurrencyLimiter{
		limit:  maxConcurrent,
		notify: make(chan struct{}),
	}
}

// SetLimit 动态调整并发限制
func (l *ConcurrencyLimiter) SetLimit(newLimit int) {
	if newLimit <= 0 {
		newLimit = 1
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.limit = newLimit
	l.signalLocked()
}

// Acquire 获取一个执行槽位，如果达到并发上限则阻塞
// 返回 release 函数，调用后释放槽位
func (l *ConcurrencyLimiter) Acquire(ctx context.Context) (release func(), err error) {
	for {
		l.mu.Lock()
		if l.active < l.limit {
			l.active++
			l.mu.Unlock()

			var once sync.Once
			return func() {
				once.Do(func() {
					l.mu.Lock()
					if l.active > 0 {
						l.active--
					}
					l.signalLocked()
					l.mu.Unlock()
				})
			}, nil
		}
		notify := l.notify
		l.mu.Unlock()

		select {
		case <-notify:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// TryAcquire 尝试获取槽位，不阻塞
// 如果成功返回 release 函数，如果已满返回 nil
func (l *ConcurrencyLimiter) TryAcquire() (release func(), acquired bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.active >= l.limit {
		return nil, false
	}

	l.active++
	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Lock()
			if l.active > 0 {
				l.active--
			}
			l.signalLocked()
			l.mu.Unlock()
		})
	}, true
}

// Available 返回当前可用的槽位数
func (l *ConcurrencyLimiter) Available() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	available := l.limit - l.active
	if available < 0 {
		return 0
	}
	return available
}

// Waiting 返回当前等待/使用中的槽位数
func (l *ConcurrencyLimiter) Waiting() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.active
}

// Capacity 返回总槽位容量
func (l *ConcurrencyLimiter) Capacity() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit
}

func (l *ConcurrencyLimiter) signalLocked() {
	close(l.notify)
	l.notify = make(chan struct{})
}
