package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (p *DoubaoProvider) GenerateTagsBatch(ctx context.Context, requests []TagRequest) (*BatchTagResult, error) {
	if len(requests) == 0 {
		return &BatchTagResult{Groups: [][]string{}}, nil
	}
	if len(requests) == 1 && p.effectiveBatchMode() != doubaoBatchModeMulti {
		result, err := p.GenerateTags(ctx, requests[0].Path, requests[0].Prompt)
		if err != nil {
			return nil, err
		}
		return &BatchTagResult{
			Groups:      [][]string{result.Tags},
			ModelName:   result.ModelName,
			RawResponse: result.RawResponse,
		}, nil
	}

	batchMessage, err := p.buildBatchMessage(requests)
	if err != nil {
		return nil, err
	}

	batchReq := doubaoBatchRequest{
		Model:       p.model,
		Temperature: 0.25,
		MaxTokens:   10000,
		TopP:        0.7,
		Messages:    []doubaoBatchMessage{batchMessage},
	}

	body, err := json.Marshal(batchReq)
	if err != nil {
		return nil, newProviderRequestError(fmt.Sprintf("marshal request: %v", err), false, "marshal_request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, newProviderRequestError(fmt.Sprintf("create request: %v", err), false, "create_request")
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

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, newProviderRequestError("rate limit exceeded", true, "rate_limit")
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, newProviderRequestError(fmt.Sprintf("service unavailable: %d", resp.StatusCode), true, fmt.Sprintf("http_%d", resp.StatusCode))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newProviderRequestError(fmt.Sprintf("unexpected status code: %d", resp.StatusCode), false, fmt.Sprintf("http_%d", resp.StatusCode))
	}

	var providerResp doubaoBatchResponse
	if err := json.Unmarshal(respBody, &providerResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if providerResp.Error != nil {
		return nil, classifyOpenAICompatibleAPIError(providerResp.Error.Message, providerResp.Error.Code)
	}

	if len(providerResp.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	content := providerResp.Choices[0].Message.Content
	groups := ParseBatchTagsResponse(content, len(requests))
	for i := range requests {
		if len(groups[i]) == 0 {
			return nil, fmt.Errorf("parse batch response: missing batch tags for request %d", i+1)
		}
	}

	return &BatchTagResult{
		Groups:      groups,
		ModelName:   p.model,
		RawResponse: string(respBody),
	}, nil
}

type doubaoBatchRequest struct {
	Model       string               `json:"model"`
	Messages    []doubaoBatchMessage `json:"messages"`
	Temperature float64              `json:"temperature,omitempty"`
	TopP        float64              `json:"top_p,omitempty"`
	MaxTokens   int                  `json:"max_tokens,omitempty"`
}

type doubaoBatchMessage struct {
	Role    string                   `json:"role"`
	Content []doubaoBatchContentItem `json:"content"`
}

type doubaoBatchContentItem struct {
	Type     string                    `json:"type"`
	Text     string                    `json:"text,omitempty"`
	ImageURL *openAICompatibleImageURL `json:"image_url,omitempty"`
}

type doubaoBatchResponse struct {
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

func (p *DoubaoProvider) buildBatchMessage(requests []TagRequest) (doubaoBatchMessage, error) {
	content := make([]doubaoBatchContentItem, 0, len(requests)*2)

	prompt := buildBatchPrompt(requests[0].Prompt)
	content = append(content, doubaoBatchContentItem{Type: "text", Text: prompt})

	for _, req := range requests {
		imgContent, err := imageToContentItem(req.Path)
		if err != nil {
			return doubaoBatchMessage{}, newProviderRequestError(fmt.Sprintf("process image url: %v", err), false, "local_image_processing")
		}
		content = append(content, imgContent)
	}

	return doubaoBatchMessage{Role: "user", Content: content}, nil
}

func buildBatchPrompt(basePrompt string) string {
	return fmt.Sprintf(`你将收到多张图片，请按编号 1-N 分别分析并输出标签。
【输出格式要求】
每张图片的标签占一行，格式为 "编号: 标签1,标签2,标签3,..."。
例如：
1: 泳装,黑丝,银发
2: 女仆装,白丝,短发
3: 泳装,比基尼,金发

【标签规则】
%s

【注意】输出格式以本处为准，每行必须以编号开头，不要合并所有图片的标签到一行。
直接输出结果，不要解释。`, basePrompt)
}

func imageToContentItem(path string) (doubaoBatchContentItem, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return doubaoBatchContentItem{
			Type:     "image_url",
			ImageURL: &openAICompatibleImageURL{URL: path},
		}, nil
	}

	processedURL, err := processImageURLForProvider(path)
	if err != nil {
		return doubaoBatchContentItem{}, err
	}

	return doubaoBatchContentItem{
		Type: "image_url",
		ImageURL: &openAICompatibleImageURL{
			URL: processedURL,
		},
	}, nil
}
