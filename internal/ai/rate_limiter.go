package ai

import (
	"context"
	"errors"

	"golang.org/x/time/rate"
)

// RateLimitedClient 限流客户端，包装底层 AIProvider
type RateLimitedClient struct {
	provider AIProvider
	limiter  *rate.Limiter
}

// NewRateLimitedClient 创建限流客户端
// requestsPerMinute 每分钟允许的请求数
func NewRateLimitedClient(provider AIProvider, requestsPerMinute int) *RateLimitedClient {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60 // 默认 60 请求/分钟 = 1 请求/秒
	}

	// 计算 rps (requests per second)
	rps := float64(requestsPerMinute) / 60.0

	// burst = 1 表示严格限制，不允许突发
	limiter := rate.NewLimiter(rate.Limit(rps), 1)

	return &RateLimitedClient{
		provider: provider,
		limiter:  limiter,
	}
}

// Name 返回底层提供商名称
func (c *RateLimitedClient) Name() string {
	return c.provider.Name()
}

// GenerateTags 生成标签，带限流控制
func (c *RateLimitedClient) GenerateTags(ctx interface{}, imageURL, prompt string) (*TagResult, error) {
	contextCtx, ok := ctx.(context.Context)
	if !ok {
		return nil, errors.New("invalid context type")
	}

	// 等待令牌
	if err := c.limiter.Wait(contextCtx); err != nil {
		return nil, err
	}

	// 调用底层提供商
	return c.provider.GenerateTags(ctx, imageURL, prompt)
}
