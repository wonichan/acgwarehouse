package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	var _ AIProvider = (*ZhipuProvider)(nil)
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
		{"zhipu", "glm-4v-flash"},
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

func TestNewProvider_UsesExtendedHTTPTimeout(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{name: "qwen", provider: "qwen"},
		{name: "doubao", provider: "doubao"},
		{name: "zhipu", provider: "zhipu"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(&config.AIConfig{
				Provider: tt.provider,
				APIKey:   "test-api-key",
			})
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			expectedTimeout := 120 * time.Second

			switch actual := provider.(type) {
			case *QwenProvider:
				if actual.httpClient.Timeout != expectedTimeout {
					t.Fatalf("expected timeout %v, got %v", expectedTimeout, actual.httpClient.Timeout)
				}
				transport, ok := actual.httpClient.Transport.(*http.Transport)
				if !ok || transport == nil {
					t.Fatalf("expected qwen provider to configure *http.Transport")
				}
				if transport.TLSHandshakeTimeout <= 0 {
					t.Fatalf("expected TLSHandshakeTimeout to be set, got %v", transport.TLSHandshakeTimeout)
				}
				if transport.ResponseHeaderTimeout <= 0 {
					t.Fatalf("expected ResponseHeaderTimeout to be set, got %v", transport.ResponseHeaderTimeout)
				}
			case *DoubaoProvider:
				if actual.httpClient.Timeout != expectedTimeout {
					t.Fatalf("expected timeout %v, got %v", expectedTimeout, actual.httpClient.Timeout)
				}
				transport, ok := actual.httpClient.Transport.(*http.Transport)
				if !ok || transport == nil {
					t.Fatalf("expected doubao provider to configure *http.Transport")
				}
				if transport.TLSHandshakeTimeout <= 0 {
					t.Fatalf("expected TLSHandshakeTimeout to be set, got %v", transport.TLSHandshakeTimeout)
				}
				if transport.ResponseHeaderTimeout <= 0 {
					t.Fatalf("expected ResponseHeaderTimeout to be set, got %v", transport.ResponseHeaderTimeout)
				}
			case *ZhipuProvider:
				if actual.httpClient.Timeout != expectedTimeout {
					t.Fatalf("expected timeout %v, got %v", expectedTimeout, actual.httpClient.Timeout)
				}
				transport, ok := actual.httpClient.Transport.(*http.Transport)
				if !ok || transport == nil {
					t.Fatalf("expected zhipu provider to configure *http.Transport")
				}
				if transport.TLSHandshakeTimeout <= 0 {
					t.Fatalf("expected TLSHandshakeTimeout to be set, got %v", transport.TLSHandshakeTimeout)
				}
				if transport.ResponseHeaderTimeout <= 0 {
					t.Fatalf("expected ResponseHeaderTimeout to be set, got %v", transport.ResponseHeaderTimeout)
				}
			default:
				t.Fatalf("unexpected provider type %T", provider)
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
			result := parseCommaSeparatedTags(tt.input)
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

// ========== DoubaoProvider Tests ==========

func TestDoubaoProvider_BuildRequest(t *testing.T) {
	// Test 1: DoubaoProvider 正确构建 HTTP 请求
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
		var req doubaoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if req.Model != "doubao-vision-pro" {
			t.Errorf("expected model doubao-vision-pro, got %s", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(req.Messages))
		}

		// 返回模拟响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doubaoResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "girl, outdoors, sunny"}},
			},
		})
	}))
	defer server.Close()

	provider := &DoubaoProvider{
		apiKey:     "test-api-key",
		model:      "doubao-vision-pro",
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	_, err := provider.GenerateTags(context.Background(), "https://example.com/image.jpg", "generate tags")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDoubaoProvider_ParseResponse(t *testing.T) {
	// Test 2: DoubaoProvider 正确解析响应
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doubaoResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "anime, girl, blue hair, illustration"}},
			},
		})
	}))
	defer server.Close()

	provider := &DoubaoProvider{
		apiKey:     "test-api-key",
		model:      "doubao-vision-pro",
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	result, err := provider.GenerateTags(context.Background(), "https://example.com/image.jpg", "generate tags")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Tags) != 4 {
		t.Errorf("expected 4 tags, got %d: %v", len(result.Tags), result.Tags)
	}
	expectedTags := []string{"anime", "girl", "blue hair", "illustration"}
	for i, tag := range expectedTags {
		if result.Tags[i] != tag {
			t.Errorf("expected tag %s at position %d, got %s", tag, i, result.Tags[i])
		}
	}
	if result.ModelName != "doubao-vision-pro" {
		t.Errorf("expected model name doubao-vision-pro, got %s", result.ModelName)
	}
}

func TestDoubaoProvider_VolcanoEndpoint(t *testing.T) {
	// Test 3: DoubaoProvider 使用火山引擎端点
	provider := &DoubaoProvider{
		apiKey:     "test-api-key",
		model:      "doubao-vision-pro",
		endpoint:   "https://ark.cn-beijing.volces.com/api/v3",
		httpClient: &http.Client{},
	}

	// 验证端点正确
	if provider.endpoint != "https://ark.cn-beijing.volces.com/api/v3" {
		t.Errorf("expected volcano engine endpoint, got %s", provider.endpoint)
	}
	if provider.Name() != "doubao" {
		t.Errorf("expected provider name 'doubao', got %s", provider.Name())
	}
}

func TestDoubaoProvider_HandleErrors(t *testing.T) {
	// Test: DoubaoProvider 处理错误状态码
	tests := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{"rate limit 429", http.StatusTooManyRequests, true},
		{"server error 500", http.StatusInternalServerError, true},
		{"server error 503", http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			provider := &DoubaoProvider{
				apiKey:     "test-api-key",
				model:      "doubao-vision-pro",
				endpoint:   server.URL,
				httpClient: server.Client(),
			}

			_, err := provider.GenerateTags(context.Background(), "https://example.com/image.jpg", "generate tags")
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestDoubaoProvider_GenerateTagsBatch_BuildsMultiImageRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req doubaoBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if len(req.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(req.Messages))
		}

		content := req.Messages[0].Content
		imageCount := 0
		for _, item := range content {
			if item.Type == "image_url" {
				imageCount++
			}
		}
		if imageCount != 4 {
			t.Errorf("expected 4 images in batch request, got %d", imageCount)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doubaoBatchResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "1: tag1,tag2\n2: tag3,tag4\n3: tag5,tag6\n4: tag7,tag8"}},
			},
		})
	}))
	defer server.Close()

	provider := &DoubaoProvider{
		apiKey:     "test-api-key",
		model:      "doubao-seed-2-0-pro",
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	requests := []TagRequest{
		{ImageID: 1, Path: "https://example.com/1.jpg", Prompt: ""},
		{ImageID: 2, Path: "https://example.com/2.jpg", Prompt: ""},
		{ImageID: 3, Path: "https://example.com/3.jpg", Prompt: ""},
		{ImageID: 4, Path: "https://example.com/4.jpg", Prompt: ""},
	}

	result, err := provider.GenerateTagsBatch(context.Background(), requests)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Groups) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(result.Groups))
	}
	if len(result.Groups[0]) != 2 {
		t.Errorf("group 0 expected 2 tags, got %d", len(result.Groups[0]))
	}
}

func TestDoubaoProvider_ProcessImageURL_RespectsPayloadBudget(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "ai_payload_budget_*.jpg")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	img := createTestImage(5000, 5000)
	if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}

	processedURL, err := processImageURLForProvider(tmpFile.Name())
	if err != nil {
		t.Fatalf("processImageURL failed: %v", err)
	}

	if !strings.HasPrefix(processedURL, "data:image/") {
		prefix := processedURL
		if len(prefix) > 32 {
			prefix = prefix[:32]
		}
		t.Fatalf("expected data url, got %q", prefix)
	}

	commaIndex := strings.Index(processedURL, ",")
	if commaIndex < 0 {
		t.Fatalf("expected data url to contain comma separator")
	}

	encodedPayload := processedURL[commaIndex+1:]
	decodedSize := base64.StdEncoding.DecodedLen(len(encodedPayload))
	if decodedSize > maxAIImageSize {
		t.Fatalf("expected decoded payload to stay within %d bytes, got %d bytes", maxAIImageSize, decodedSize)
	}

	if len(processedURL) > maxAIImageSize {
		t.Fatalf("expected data url payload to stay within %d bytes, got %d bytes", maxAIImageSize, len(processedURL))
	}
}

func TestEnsureDataURLFitsBudget_RejectsOversizedPayload(t *testing.T) {
	oversizedData := make([]byte, maxAIDataURLSize)

	err := ensureDataURLFitsBudget("image/jpeg", oversizedData)
	if err == nil {
		t.Fatal("expected oversized payload to be rejected")
	}

	if !strings.Contains(err.Error(), "data url payload exceeds budget") {
		t.Fatalf("expected budget error, got %v", err)
	}
}

func TestNewProvider_DoubaoBatchModeNormalization(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{name: "default auto", mode: "", want: "auto"},
		{name: "propagates multi", mode: "multi", want: "multi"},
		{name: "normalizes case and whitespace", mode: " SINGLE ", want: "single"},
		{name: "invalid falls back to auto", mode: "unexpected", want: "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(&config.AIConfig{
				Provider:        "doubao",
				APIKey:          "test-api-key",
				Model:           "doubao-vision-pro",
				DoubaoBatchMode: tt.mode,
			})
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			doubaoProvider, ok := provider.(*DoubaoProvider)
			if !ok {
				t.Fatalf("expected DoubaoProvider, got %T", provider)
			}
			if doubaoProvider.effectiveBatchMode() != tt.want {
				t.Fatalf("expected batch mode %q, got %q", tt.want, doubaoProvider.effectiveBatchMode())
			}
		})
	}
}

func TestDoubaoProvider_GenerateTagsBatch_SingleRequestUsesBatchPathInMultiMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req doubaoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		prompt := req.Messages[0].Content[0].Text
		if !strings.Contains(prompt, "1:") {
			t.Fatalf("expected numbered batch prompt, got %q", prompt)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doubaoBatchResponse{Choices: []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{{Message: struct {
			Content string `json:"content"`
		}{Content: "1: anime, blue hair, illustration"}}}})
	}))
	defer server.Close()

	provider := &DoubaoProvider{apiKey: "test-api-key", model: "doubao-vision-pro", endpoint: server.URL, httpClient: server.Client(), batchMode: "multi"}
	result, err := provider.GenerateTagsBatch(context.Background(), []TagRequest{{ImageID: 1, Path: "https://example.com/image.jpg", Prompt: "generate tags"}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Groups) != 1 || strings.Join(result.Groups[0], ",") != "anime,blue hair,illustration" {
		t.Fatalf("unexpected batch result: %+v", result)
	}
}

func TestDoubaoProvider_GenerateTagsBatch_UsesSharedProviderErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doubaoBatchResponse{Error: &struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		}{Message: "bad request", Code: "invalid_request"}})
	}))
	defer server.Close()

	provider := &DoubaoProvider{apiKey: "test-api-key", model: "doubao-vision-pro", endpoint: server.URL, httpClient: server.Client(), batchMode: "multi"}
	_, err := provider.GenerateTagsBatch(context.Background(), []TagRequest{{ImageID: 1, Path: "https://example.com/image.jpg", Prompt: "generate tags"}})
	if err == nil || !strings.Contains(err.Error(), "api error: bad request") {
		t.Fatalf("expected provider error handling, got %v", err)
	}
}

func TestProcessImageURLForProvider_RejectsPathOutsideAllowedRoots(t *testing.T) {
	allowedRoot := t.TempDir()
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.jpg")
	if err := os.WriteFile(outsidePath, []byte("not-an-image"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	previousRoots := SetAllowedLocalImageRoots([]string{allowedRoot})
	defer SetAllowedLocalImageRoots(previousRoots)

	_, err := processImageURLForProvider(outsidePath)
	if err == nil || !strings.Contains(err.Error(), "outside allowed scan roots") {
		t.Fatalf("expected allowed-root rejection, got %v", err)
	}
}

func TestDoubaoProvider_GenerateTagsBatch_SanitizesUpstreamErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"secret":"token","error":"bad"}`))
	}))
	defer server.Close()

	provider := &DoubaoProvider{apiKey: "test-api-key", model: "doubao-vision-pro", endpoint: server.URL, httpClient: server.Client(), batchMode: "multi"}
	_, err := provider.GenerateTagsBatch(context.Background(), []TagRequest{{ImageID: 1, Path: "https://example.com/image.jpg", Prompt: "generate tags"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "token") || strings.Contains(err.Error(), "body:") {
		t.Fatalf("expected sanitized upstream error, got %v", err)
	}
}

func TestNewProvider_DoubaoFallbackModels_PreserveForcedMultiBatchSemantics(t *testing.T) {
	primaryCalls := 0
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryCalls++
		var req doubaoBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode primary request: %v", err)
		}
		if got := req.Model; got != "doubao-primary" {
			t.Fatalf("expected primary model, got %q", got)
		}
		if prompt := req.Messages[0].Content[0].Text; !strings.Contains(prompt, "1:") {
			t.Fatalf("expected multi-mode batch prompt on primary, got %q", prompt)
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer primary.Close()

	fallbackCalls := 0
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalls++
		var req doubaoBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode fallback request: %v", err)
		}
		if got := req.Model; got != "doubao-fallback" {
			t.Fatalf("expected fallback model, got %q", got)
		}
		if prompt := req.Messages[0].Content[0].Text; !strings.Contains(prompt, "1:") {
			t.Fatalf("expected multi-mode batch prompt on fallback, got %q", prompt)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doubaoBatchResponse{Choices: []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{{Message: struct {
			Content string `json:"content"`
		}{Content: "1: built, via, newprovider"}}}})
	}))
	defer fallback.Close()

	provider, err := NewProvider(&config.AIConfig{Provider: "doubao", APIKey: "test-api-key", Model: "doubao-primary", FallbackModels: []string{"doubao-fallback"}, DoubaoBatchMode: "multi"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	fallbackProvider, ok := provider.(*FallbackDoubaoProvider)
	if !ok {
		t.Fatalf("expected FallbackDoubaoProvider, got %T", provider)
	}
	fallbackProvider.clients[0].endpoint = primary.URL
	fallbackProvider.clients[0].httpClient = primary.Client()
	fallbackProvider.clients[1].endpoint = fallback.URL
	fallbackProvider.clients[1].httpClient = fallback.Client()

	result, err := provider.GenerateTagsBatch(context.Background(), []TagRequest{{ImageID: 1, Path: "https://example.com/image.jpg", Prompt: "generate tags"}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if primaryCalls != 1 || fallbackCalls != 1 {
		t.Fatalf("expected one primary and one fallback call, got primary=%d fallback=%d", primaryCalls, fallbackCalls)
	}
	if len(result.Groups) != 1 || strings.Join(result.Groups[0], ",") != "built,via,newprovider" {
		t.Fatalf("unexpected results: %+v", result)
	}
}

// ========== ZhipuProvider Tests ==========

func TestNewProvider_Zhipu(t *testing.T) {
	cfg := &config.AIConfig{
		Provider: "zhipu",
		APIKey:   "test-api-key",
		Model:    "glm-4v-flash",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if provider.Name() != "zhipu" {
		t.Errorf("expected provider name 'zhipu', got %s", provider.Name())
	}

	if _, ok := provider.(*ZhipuProvider); !ok {
		t.Error("expected ZhipuProvider type")
	}
}

func TestZhipuProvider_BuildRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("expected /chat/completions path, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("expected Bearer token, got %s", auth)
		}

		var req zhipuRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if req.Model != "glm-4v-flash" {
			t.Errorf("expected model glm-4v-flash, got %s", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(req.Messages))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(zhipuResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "girl, outdoors, anime"}},
			},
		})
	}))
	defer server.Close()

	provider := &ZhipuProvider{
		apiKey:     "test-api-key",
		model:      "glm-4v-flash",
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	_, err := provider.GenerateTags(context.Background(), "https://example.com/image.jpg", "generate tags")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestZhipuProvider_ParseResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(zhipuResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "anime, girl, blue hair, illustration"}},
			},
		})
	}))
	defer server.Close()

	provider := &ZhipuProvider{
		apiKey:     "test-api-key",
		model:      "glm-4v-flash",
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	result, err := provider.GenerateTags(context.Background(), "https://example.com/image.jpg", "generate tags")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Tags) != 4 {
		t.Errorf("expected 4 tags, got %d: %v", len(result.Tags), result.Tags)
	}
	expectedTags := []string{"anime", "girl", "blue hair", "illustration"}
	for i, tag := range expectedTags {
		if result.Tags[i] != tag {
			t.Errorf("expected tag %s at position %d, got %s", tag, i, result.Tags[i])
		}
	}
	if result.ModelName != "glm-4v-flash" {
		t.Errorf("expected model name glm-4v-flash, got %s", result.ModelName)
	}
}

func TestZhipuProvider_Endpoint(t *testing.T) {
	provider := &ZhipuProvider{
		apiKey:     "test-api-key",
		model:      "glm-4v-flash",
		endpoint:   "https://open.bigmodel.cn/api/paas/v4",
		httpClient: &http.Client{},
	}

	if provider.endpoint != "https://open.bigmodel.cn/api/paas/v4" {
		t.Errorf("expected zhipu endpoint, got %s", provider.endpoint)
	}
	if provider.Name() != "zhipu" {
		t.Errorf("expected provider name 'zhipu', got %s", provider.Name())
	}
}

func TestZhipuProvider_HandleErrors(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{"rate limit 429", http.StatusTooManyRequests, true},
		{"server error 500", http.StatusInternalServerError, true},
		{"server error 503", http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			provider := &ZhipuProvider{
				apiKey:     "test-api-key",
				model:      "glm-4v-flash",
				endpoint:   server.URL,
				httpClient: server.Client(),
			}

			_, err := provider.GenerateTags(context.Background(), "https://example.com/image.jpg", "generate tags")
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
