package ai

import (
	"errors"
	"net/http"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
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
	GenerateTags(ctx interface{}, imageURL, prompt string) (*TagResult, error)
}

// NewProvider 根据配置创建 AI 提供商实例
func NewProvider(cfg *config.AIConfig) (AIProvider, error) {
	if cfg == nil {
		return nil, errors.New("ai config is nil")
	}

	httpClient := &http.Client{
		Timeout: 60 * time.Second,
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
		return &DoubaoProvider{
			apiKey:     cfg.APIKey,
			model:      model,
			endpoint:   "https://ark.cn-beijing.volces.com/api/v3",
			httpClient: httpClient,
		}, nil
	default:
		return nil, errors.New("unknown ai provider: " + cfg.Provider)
	}
}
