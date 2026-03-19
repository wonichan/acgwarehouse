package ai

import (
	"context"
)

// ConcurrencyLimiter 限制同时执行的 AI 请求数量
type ConcurrencyLimiter struct {
	sem chan struct{}
}

// NewConcurrencyLimiter 创建并发限制器
// maxConcurrent 同时执行的最大请求数
func NewConcurrencyLimiter(maxConcurrent int) *ConcurrencyLimiter {
	if maxConcurrent <= 0 {
		maxConcurrent = 3 // 默认 3 个并发
	}
	return &ConcurrencyLimiter{
		sem: make(chan struct{}, maxConcurrent),
	}
}

// Acquire 获取一个执行槽位，如果达到并发上限则阻塞
// 返回 release 函数，调用后释放槽位
func (l *ConcurrencyLimiter) Acquire(ctx context.Context) (release func(), err error) {
	select {
	case l.sem <- struct{}{}:
		return func() { <-l.sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TryAcquire 尝试获取槽位，不阻塞
// 如果成功返回 release 函数，如果已满返回 nil
func (l *ConcurrencyLimiter) TryAcquire() (release func(), acquired bool) {
	select {
	case l.sem <- struct{}{}:
		return func() { <-l.sem }, true
	default:
		return nil, false
	}
}

// Available 返回当前可用的槽位数
func (l *ConcurrencyLimiter) Available() int {
	return cap(l.sem) - len(l.sem)
}

// Waiting 返回当前等待/使用中的槽位数
func (l *ConcurrencyLimiter) Waiting() int {
	return len(l.sem)
}

// Capacity 返回总槽位容量
func (l *ConcurrencyLimiter) Capacity() int {
	return cap(l.sem)
}
