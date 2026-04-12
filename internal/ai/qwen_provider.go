package ai

import (
	"context"
	"net/http"
)

// QwenProvider 千问 VL 提供商实现
type QwenProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

type qwenRequest = openAICompatibleRequest
type qwenMessage = openAICompatibleMessage
type qwenContent = openAICompatibleContent
type qwenImageURL = openAICompatibleImageURL
type qwenResponse = openAICompatibleResponse

// Name 返回提供商名称
func (p *QwenProvider) Name() string {
	return "qwen"
}

// GenerateTags 为图片生成标签
func (p *QwenProvider) GenerateTags(ctx context.Context, imageURL, prompt string) (*TagResult, error) {
	result, err := generateOpenAICompatibleTags(ctx, p.httpClient, p.baseURL, p.apiKey, p.model, imageURL, prompt)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *QwenProvider) GenerateTagsBatch(ctx context.Context, requests []TagRequest) (*BatchTagResult, error) {
	return generateTagsBatchFallback(ctx, p, requests)
}
