package ai

import (
	"context"
	"net/http"
)

// ZhipuProvider 智谱视觉模型提供商实现
type ZhipuProvider struct {
	apiKey     string
	model      string
	endpoint   string
	httpClient *http.Client
}

type zhipuRequest = openAICompatibleRequest
type zhipuMessage = openAICompatibleMessage
type zhipuContent = openAICompatibleContent
type zhipuImageURL = openAICompatibleImageURL
type zhipuResponse = openAICompatibleResponse

// Name 返回提供商名称
func (p *ZhipuProvider) Name() string {
	return "zhipu"
}

// GenerateTags 为图片生成标签
func (p *ZhipuProvider) GenerateTags(ctx context.Context, imageURL, prompt string) (*TagResult, error) {
	result, err := generateOpenAICompatibleTags(ctx, p.httpClient, p.endpoint, p.apiKey, p.model, imageURL, prompt)
	if err != nil {
		return nil, err
	}
	return result, nil
}
