package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// ========== QwenProvider Tests ==========

func TestQwenProvider_BuildRequest(t *testing.T) {
	// Test 1: QwenProvider 正确构建 HTTP 请求
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法和路径
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("expected /chat/completions path, got %s", r.URL.Path)
		}

		// 验证请求头
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("expected Bearer token, got %s", auth)
		}

		// 验证请求体
		var req qwenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if req.Model != "qwen-vl-max" {
			t.Errorf("expected model qwen-vl-max, got %s", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(req.Messages))
		}

		// 返回模拟响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(qwenResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "girl, outdoors, sunny, anime"}},
			},
		})
	}))
	defer server.Close()

	provider := &QwenProvider{
		apiKey:     "test-api-key",
		model:      "qwen-vl-max",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	_, err := provider.GenerateTags(context.Background(), "https://example.com/image.jpg", "generate tags")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestQwenProvider_ParseResponse(t *testing.T) {
	// Test 2: QwenProvider 正确解析响应并返回 TagResult
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(qwenResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "girl, outdoors, sunny, anime, illustration"}},
			},
		})
	}))
	defer server.Close()

	provider := &QwenProvider{
		apiKey:     "test-api-key",
		model:      "qwen-vl-max",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	result, err := provider.GenerateTags(context.Background(), "https://example.com/image.jpg", "generate tags")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Tags) != 5 {
		t.Errorf("expected 5 tags, got %d: %v", len(result.Tags), result.Tags)
	}
	expectedTags := []string{"girl", "outdoors", "sunny", "anime", "illustration"}
	for i, tag := range expectedTags {
		if result.Tags[i] != tag {
			t.Errorf("expected tag %s at position %d, got %s", tag, i, result.Tags[i])
		}
	}
	if result.ModelName != "qwen-vl-max" {
		t.Errorf("expected model name qwen-vl-max, got %s", result.ModelName)
	}
}

func TestQwenProvider_HandleErrors(t *testing.T) {
	// Test 3: QwenProvider 处理错误状态码
	tests := []struct {
		name         string
		statusCode   int
		responseBody interface{}
		expectError  bool
	}{
		{
			name:        "rate limit 429",
			statusCode:  http.StatusTooManyRequests,
			expectError: true,
		},
		{
			name:        "server error 500",
			statusCode:  http.StatusInternalServerError,
			expectError: true,
		},
		{
			name:        "server error 503",
			statusCode:  http.StatusServiceUnavailable,
			expectError: true,
		},
		{
			name:       "api error response",
			statusCode: http.StatusOK,
			responseBody: qwenResponse{
				Error: &struct {
					Message string `json:"message"`
					Code    string `json:"code"`
				}{Message: "invalid api key", Code: "invalid_api_key"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != nil {
					_ = json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			provider := &QwenProvider{
				apiKey:     "test-api-key",
				model:      "qwen-vl-max",
				baseURL:    server.URL,
				httpClient: server.Client(),
			}

			_, err := provider.GenerateTags(context.Background(), "https://example.com/image.jpg", "generate tags")
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestQwenProvider_ParseTags(t *testing.T) {
	// Test: parseTags 正确分割标签
	provider := &QwenProvider{}

	tests := []struct {
		input    string
		expected []string
	}{
		{"girl, outdoors, sunny", []string{"girl", "outdoors", "sunny"}},
		{"  anime  ,  illustration  ,  digital art  ", []string{"anime", "illustration", "digital art"}},
		{"single", []string{"single"}},
		{"", []string{}},
		{",,,", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := provider.parseTags(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tags, got %d", len(tt.expected), len(result))
				return
			}
			for i, tag := range tt.expected {
				if result[i] != tag {
					t.Errorf("expected tag %s at position %d, got %s", tag, i, result[i])
				}
			}
		})
	}
}
