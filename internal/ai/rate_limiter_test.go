package ai

import (
	"context"
	"testing"
	"time"
)

func TestRateLimitedClient_LimitsRequests(t *testing.T) {
	// Test 1: RateLimitedClient 限制请求频率
	mockProvider := &mockProvider{
		result: &TagResult{Tags: []string{"test"}},
		err:    nil,
	}

	client := NewRateLimitedClient(mockProvider, 600)

	start := time.Now()
	ctx := context.Background()

	// 发送 2 个请求，第二个应该等待
	_, err1 := client.GenerateTags(ctx, "url1", "prompt")
	if err1 != nil {
		t.Fatalf("first request failed: %v", err1)
	}

	_, err2 := client.GenerateTags(ctx, "url2", "prompt")
	if err2 != nil {
		t.Fatalf("second request failed: %v", err2)
	}

	elapsed := time.Since(start)

	if elapsed < 80*time.Millisecond {
		t.Errorf("expected at least 80ms between requests, got %v", elapsed)
	}

	// 验证底层 provider 被调用了两次
	if mockProvider.callCount != 2 {
		t.Errorf("expected 2 provider calls, got %d", mockProvider.callCount)
	}
}

func TestRateLimitedClient_ContinuousRequests(t *testing.T) {
	// Test 2: 连续请求等待正确时间
	mockProvider := &mockProvider{
		result: &TagResult{Tags: []string{"test"}},
		err:    nil,
	}

	client := NewRateLimitedClient(mockProvider, 1200)

	ctx := context.Background()
	start := time.Now()

	// 发送 3 个请求
	for i := 0; i < 3; i++ {
		_, err := client.GenerateTags(ctx, "url", "prompt")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
	}

	elapsed := time.Since(start)

	if elapsed < 90*time.Millisecond {
		t.Errorf("expected at least 90ms for 3 requests, got %v", elapsed)
	}
}

func TestRateLimitedClient_ContextCancel(t *testing.T) {
	// Test 3: context 取消时返回错误
	mockProvider := &mockProvider{
		result: &TagResult{Tags: []string{"test"}},
		err:    nil,
	}

	// 设置 1 请求/分钟，使等待时间较长
	client := NewRateLimitedClient(mockProvider, 1)

	ctx, cancel := context.WithCancel(context.Background())

	// 第一个请求立即通过
	_, err1 := client.GenerateTags(ctx, "url1", "prompt")
	if err1 != nil {
		t.Fatalf("first request failed: %v", err1)
	}

	// 在第二个请求等待期间取消 context
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err2 := client.GenerateTags(ctx, "url2", "prompt")
	if err2 == nil {
		t.Error("expected error due to context cancellation, got nil")
	}
	if err2 != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err2)
	}
}

func TestRateLimitedClient_DelegatesToProvider(t *testing.T) {
	// Test: RateLimitedClient 正确委托到底层 provider
	expectedResult := &TagResult{
		Tags:       []string{"girl", "outdoors"},
		Confidence: 0.9,
		ModelName:  "test-model",
	}

	mockProvider := &mockProvider{
		result: expectedResult,
		err:    nil,
	}

	// 设置高速率以避免等待
	client := NewRateLimitedClient(mockProvider, 1000)

	ctx := context.Background()
	result, err := client.GenerateTags(ctx, "test-url", "test-prompt")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if result != expectedResult {
		t.Error("expected result to be delegated from provider")
	}
	if mockProvider.lastURL != "test-url" {
		t.Errorf("expected URL 'test-url', got %s", mockProvider.lastURL)
	}
	if mockProvider.lastPrompt != "test-prompt" {
		t.Errorf("expected prompt 'test-prompt', got %s", mockProvider.lastPrompt)
	}
}

// mockProvider 用于测试的模拟提供商
type mockProvider struct {
	result     *TagResult
	err        error
	callCount  int
	lastURL    string
	lastPrompt string
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) GenerateTags(ctx context.Context, imageURL, prompt string) (*TagResult, error) {
	m.callCount++
	m.lastURL = imageURL
	m.lastPrompt = prompt
	return m.result, m.err
}

func (m *mockProvider) GenerateTagsBatch(ctx context.Context, requests []TagRequest) (*BatchTagResult, error) {
	m.callCount++
	return &BatchTagResult{Groups: make([][]string, len(requests))}, m.err
}
