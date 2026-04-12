package ai

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
)

const defaultAIRequestTimeout = 120 * time.Second

const (
	doubaoBatchModeSingle = "single"
	doubaoBatchModeAuto   = "auto"
	doubaoBatchModeMulti  = "multi"
)

// TagResult 表示 AI 生成的标签结果
type TagResult struct {
	Tags        []string `json:"tags"`         // 生成的标签列表
	Confidence  float64  `json:"confidence"`   // 整体置信度
	ModelName   string   `json:"model_name"`   // 使用的模型名称
	RawResponse string   `json:"raw_response"` // 原始响应（调试用）
}

// AIProvider 定义 AI 提供商接口
type AIProvider interface {
	// Name 返回提供商名称
	Name() string

	// GenerateTags 为图片生成标签
	// imageURL 可以是本地文件路径或远程 URL
	// prompt 是标签生成提示词
	GenerateTags(ctx context.Context, imageURL, prompt string) (*TagResult, error)

	// GenerateTagsBatch 批量为多张图片生成标签（一次对话处理多张图）
	// requests 中的每个请求包含 ImageID、Path 和 Prompt
	// 返回的 BatchTagResult 中 Groups[i] 对应 requests[i] 的标签
	// 如果提供商不支持批量，fallback 到逐个调用
	GenerateTagsBatch(ctx context.Context, requests []TagRequest) (*BatchTagResult, error)
}

// NewProvider 根据配置创建 AI 提供商实例
func NewProvider(cfg *config.AIConfig) (AIProvider, error) {
	if cfg == nil {
		return nil, errors.New("ai config is nil")
	}

	httpClient := &http.Client{
		Timeout: defaultAIRequestTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   120 * time.Second,
				KeepAlive: 120 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       120 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 120 * time.Second,
		},
	}

	switch cfg.Provider {
	case "qwen":
		model := cfg.Model
		if model == "" {
			model = "qwen-vl-max"
		}
		return &QwenProvider{
			apiKey:     cfg.APIKey,
			model:      model,
			baseURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1",
			httpClient: httpClient,
		}, nil
	case "doubao":
		model := cfg.Model
		if model == "" {
			model = "doubao-vision-pro"
		}
		batchMode := normalizeDoubaoBatchMode(cfg.DoubaoBatchMode)

		fallbackModels := config.NormalizeFallbackModelsForProvider(cfg.FallbackModels)
		clients := make([]*DoubaoProvider, 0, 1+len(fallbackModels))
		clients = append(clients, &DoubaoProvider{
			apiKey:     cfg.APIKey,
			model:      model,
			endpoint:   "https://ark.cn-beijing.volces.com/api/v3",
			httpClient: httpClient,
			batchMode:  batchMode,
		})
		for _, fbModel := range fallbackModels {
			if fbModel == "" {
				continue
			}
			clients = append(clients, &DoubaoProvider{
				apiKey:     cfg.APIKey,
				model:      fbModel,
				endpoint:   "https://ark.cn-beijing.volces.com/api/v3",
				httpClient: httpClient,
				batchMode:  batchMode,
			})
		}

		if len(clients) == 1 {
			return clients[0], nil
		}
		return NewFallbackDoubaoProvider(batchMode, clients...), nil
	case "zhipu":
		model := cfg.Model
		if model == "" {
			model = "glm-4v-flash"
		}
		return &ZhipuProvider{
			apiKey:     cfg.APIKey,
			model:      model,
			endpoint:   "https://open.bigmodel.cn/api/paas/v4",
			httpClient: httpClient,
		}, nil
	default:
		return nil, errors.New("unknown ai provider: " + cfg.Provider)
	}
}

func normalizeDoubaoBatchMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case doubaoBatchModeSingle, doubaoBatchModeAuto, doubaoBatchModeMulti:
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return doubaoBatchModeAuto
	}
}

func generateTagsBatchFallback(ctx context.Context, p AIProvider, requests []TagRequest) (*BatchTagResult, error) {
	groups := make([][]string, len(requests))
	var modelName string
	var rawResponse string

	for i, req := range requests {
		result, err := p.GenerateTags(ctx, req.Path, req.Prompt)
		if err != nil {
			return nil, err
		}
		groups[i] = result.Tags
		if i == 0 || result.ModelName != "" {
			modelName = result.ModelName
		}
		rawResponse = result.RawResponse
	}

	return &BatchTagResult{
		Groups:      groups,
		ModelName:   modelName,
		RawResponse: rawResponse,
	}, nil
}
