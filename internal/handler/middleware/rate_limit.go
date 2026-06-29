package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/yachiyo/acgwarehouse/internal/handler"
	"github.com/yachiyo/acgwarehouse/pkg/errors"
)

// RateLimiter 保存内存令牌桶状态。
type RateLimiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[string]*rateBucket
	now     func() time.Time
}

type rateBucket struct {
	tokens float64
	seenAt time.Time
}

// NewRateLimiter 创建单实例内存限流器。
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &RateLimiter{
		rate:    rate,
		burst:   float64(burst),
		buckets: make(map[string]*rateBucket),
		now:     time.Now,
	}
}

// RateLimit 返回按客户端 IP 与路由分桶的限流中间件。
func RateLimit(limiter *RateLimiter) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		if limiter != nil && !limiter.Allow(rateLimitKey(ctx)) {
			handler.Fail(c, ctx, consts.StatusTooManyRequests, errors.CodeInvalidParam, "请求过于频繁", nil)
			ctx.Abort()
			return
		}
		ctx.Next(c)
	}
}

// Allow 判断指定 key 是否还有可用令牌。
func (l *RateLimiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	bucket := l.buckets[key]
	if bucket == nil {
		l.buckets[key] = &rateBucket{tokens: l.burst - 1, seenAt: now}
		return true
	}

	elapsed := now.Sub(bucket.seenAt).Seconds()
	bucket.seenAt = now
	bucket.tokens = minFloat(l.burst, bucket.tokens+elapsed*l.rate)
	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

// rateLimitKey 生成限流分桶键。
func rateLimitKey(ctx *app.RequestContext) string {
	path := ctx.FullPath()
	if path == "" {
		path = string(ctx.Path())
	}
	return ctx.ClientIP() + ":" + path
}

// minFloat 返回较小浮点数。
func minFloat(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
