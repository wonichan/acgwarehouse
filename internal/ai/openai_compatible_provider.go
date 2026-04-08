package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type openAICompatibleRequest struct {
	Model       string                    `json:"model"`
	Messages    []openAICompatibleMessage `json:"messages"`
	Temperature float64                   `json:"temperature,omitempty"`
	TopP        float64                   `json:"top_p,omitempty"`
	MaxTokens   int                       `json:"max_tokens,omitempty"`
}

type openAICompatibleMessage struct {
	Role    string                    `json:"role"`
	Content []openAICompatibleContent `json:"content"`
}

type openAICompatibleContent struct {
	Type     string                    `json:"type"`
	Text     string                    `json:"text,omitempty"`
	ImageURL *openAICompatibleImageURL `json:"image_url,omitempty"`
}

type openAICompatibleImageURL struct {
	URL string `json:"url"`
}

type openAICompatibleResponse struct {
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

func generateOpenAICompatibleTags(ctx context.Context, httpClient *http.Client, endpoint, apiKey, model, imageURL, prompt string) (*TagResult, error) {
	processedURL, err := processImageURLForProvider(imageURL)
	if err != nil {
		return nil, fmt.Errorf("process image url: %w", err)
	}

	req := openAICompatibleRequest{
		Model:       model,
		Temperature: 0.2,
		MaxTokens:   1000,
		Messages: []openAICompatibleMessage{
			{
				Role: "user",
				Content: []openAICompatibleContent{
					{Type: "text", Text: prompt},
					{Type: "image_url", ImageURL: &openAICompatibleImageURL{URL: processedURL}},
				},
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("rate limit exceeded")
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, fmt.Errorf("service unavailable: %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var providerResp openAICompatibleResponse
	if err := json.Unmarshal(respBody, &providerResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if providerResp.Error != nil {
		return nil, fmt.Errorf("api error: %s", providerResp.Error.Message)
	}

	if len(providerResp.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	content := providerResp.Choices[0].Message.Content

	return &TagResult{
		Tags:        parseCommaSeparatedTags(content),
		Confidence:  0.85,
		ModelName:   model,
		RawResponse: string(respBody),
	}, nil
}

func processImageURLForProvider(imageURL string) (string, error) {
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		return imageURL, nil
	}

	data, contentType, err := PrepareImageForDataURL(imageURL)
	if err != nil {
		return "", fmt.Errorf("process image: %w", err)
	}

	return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data)), nil
}

func parseCommaSeparatedTags(content string) []string {
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
