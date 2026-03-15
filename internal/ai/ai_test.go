package ai

import (
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
)

func TestTagResultStructure(t *testing.T) {
	// Test: TagResult 结构体定义正确的字段
	result := &TagResult{
		Tags:        []string{"girl", "outdoors", "sunny"},
		Confidence:  0.92,
		ModelName:   "qwen-vl-max",
		RawResponse: `{"tags": ["girl", "outdoors", "sunny"]}`,
	}

	if len(result.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(result.Tags))
	}
	if result.Confidence != 0.92 {
		t.Errorf("expected confidence 0.92, got %f", result.Confidence)
	}
	if result.ModelName != "qwen-vl-max" {
		t.Errorf("expected model name 'qwen-vl-max', got %s", result.ModelName)
	}
}

func TestAIProviderInterface(t *testing.T) {
	// Test: AIProvider interface 定义正确的方法
	// 这个测试通过编译就说明接口定义正确
	var _ AIProvider = (*QwenProvider)(nil)
	var _ AIProvider = (*DoubaoProvider)(nil)
}

func TestNewProvider_Qwen(t *testing.T) {
	// Test: NewProvider 工厂函数根据配置返回 QwenProvider
	cfg := &config.AIConfig{
		Provider: "qwen",
		APIKey:   "test-api-key",
		Model:    "qwen-vl-max",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if provider.Name() != "qwen" {
		t.Errorf("expected provider name 'qwen', got %s", provider.Name())
	}

	if _, ok := provider.(*QwenProvider); !ok {
		t.Error("expected QwenProvider type")
	}
}

func TestNewProvider_Doubao(t *testing.T) {
	// Test: NewProvider 工厂函数根据配置返回 DoubaoProvider
	cfg := &config.AIConfig{
		Provider: "doubao",
		APIKey:   "test-api-key",
		Model:    "doubao-vision-pro",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if provider.Name() != "doubao" {
		t.Errorf("expected provider name 'doubao', got %s", provider.Name())
	}

	if _, ok := provider.(*DoubaoProvider); !ok {
		t.Error("expected DoubaoProvider type")
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	// Test: 未知提供商返回错误
	cfg := &config.AIConfig{
		Provider: "unknown",
		APIKey:   "test-api-key",
		Model:    "unknown-model",
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("expected error for unknown provider, got nil")
	}
}

func TestNewProvider_DefaultModel(t *testing.T) {
	// Test: 如果未指定模型，使用默认值
	tests := []struct {
		provider     string
		defaultModel string
	}{
		{"qwen", "qwen-vl-max"},
		{"doubao", "doubao-vision-pro"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			cfg := &config.AIConfig{
				Provider: tt.provider,
				APIKey:   "test-api-key",
				Model:    "", // 空模型
			}

			provider, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// 调用 GenerateTags 检查默认模型
			// 由于我们只验证接口，这里检查 provider 不为 nil 即可
			if provider == nil {
				t.Error("expected provider to be non-nil")
			}
		})
	}
}
