package ai

import (
	"context"
	"net/http"
)

// DoubaoProvider 豆包视觉模型提供商实现
type DoubaoProvider struct {
	apiKey     string
	model      string
	endpoint   string
	httpClient *http.Client
}

type doubaoRequest = openAICompatibleRequest
type doubaoMessage = openAICompatibleMessage
type doubaoContent = openAICompatibleContent
type doubaoImageURL = openAICompatibleImageURL
type doubaoResponse = openAICompatibleResponse

// Name 返回提供商名称
func (p *DoubaoProvider) Name() string {
	return "doubao"
}

// GenerateTags 为图片生成标签
func (p *DoubaoProvider) GenerateTags(ctx context.Context, imageURL, prompt string) (*TagResult, error) {
	result, err := generateOpenAICompatibleTags(ctx, p.httpClient, p.endpoint, p.apiKey, p.model, imageURL, prompt, p.processImageURL)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// processImageURL 处理图片 URL
func (p *DoubaoProvider) processImageURL(imageURL string) (string, error) {
	return processImageURLForProvider(imageURL)
}

// parseTags 从响应内容解析标签
func (p *DoubaoProvider) parseTags(content string) []string {
	return parseCommaSeparatedTags(content)
}
