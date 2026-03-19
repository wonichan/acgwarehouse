package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// QwenProvider 千问 VL 提供商实现
type QwenProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// Name 返回提供商名称
func (p *QwenProvider) Name() string {
	return "qwen"
}

// qwenRequest 千问 API 请求结构
type qwenRequest struct {
	Model    string        `json:"model"`
	Messages []qwenMessage `json:"messages"`
}

type qwenMessage struct {
	Role    string        `json:"role"`
	Content []qwenContent `json:"content"`
}

type qwenContent struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *qwenImageURL `json:"image_url,omitempty"`
}

type qwenImageURL struct {
	URL string `json:"url"`
}

// qwenResponse 千问 API 响应结构
type qwenResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// GenerateTags 为图片生成标签
func (p *QwenProvider) GenerateTags(ctx interface{}, imageURL, prompt string) (*TagResult, error) {
	contextCtx, ok := ctx.(context.Context)
	if !ok {
		return nil, errors.New("invalid context type")
	}

	// 处理图片 URL（支持本地文件和远程 URL）
	processedURL, err := p.processImageURL(imageURL)
	if err != nil {
		return nil, fmt.Errorf("process image url: %w", err)
	}

	// 构建请求
	req := qwenRequest{
		Model: p.model,
		Messages: []qwenMessage{
			{
				Role: "user",
				Content: []qwenContent{
					{Type: "text", Text: prompt},
					{Type: "image_url", ImageURL: &qwenImageURL{URL: processedURL}},
				},
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 发送请求
	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(contextCtx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 检查错误状态码
	if resp.StatusCode == 429 {
		return nil, errors.New("rate limit exceeded")
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("service unavailable: %d", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var qwenResp qwenResponse
	if err := json.Unmarshal(respBody, &qwenResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if qwenResp.Error != nil {
		return nil, fmt.Errorf("api error: %s", qwenResp.Error.Message)
	}

	if len(qwenResp.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	content := qwenResp.Choices[0].Message.Content

	// 解析标签
	tags := p.parseTags(content)

	return &TagResult{
		Tags:        tags,
		Confidence:  0.85, // 默认置信度
		ModelName:   p.model,
		RawResponse: string(respBody),
	}, nil
}

// processImageURL 处理图片 URL
func (p *QwenProvider) processImageURL(imageURL string) (string, error) {
	// 检查是否是远程 URL
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		return imageURL, nil
	}

	// 本地文件：使用压缩工具处理
	data, contentType, err := CompressImageIfNeeded(imageURL)
	if err != nil {
		return "", fmt.Errorf("process image: %w", err)
	}

	return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data)), nil
}

// parseTags 从响应内容解析标签
func (p *QwenProvider) parseTags(content string) []string {
	// 按逗号分割，清理空白
	parts := strings.Split(content, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
