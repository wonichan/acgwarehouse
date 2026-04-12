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

type providerRequestError struct {
	message   string
	retryable bool
	code      string
}

func (e *providerRequestError) Error() string {
	return e.message
}

func newProviderRequestError(message string, retryable bool, code string) error {
	return &providerRequestError{message: message, retryable: retryable, code: code}
}

func classifyOpenAICompatibleAPIError(message, code string) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(code))
	switch normalizedCode {
	case "invalid_api_key", "invalid_request", "authentication_error", "unauthorized", "forbidden":
		return newProviderRequestError(fmt.Sprintf("api error: %s", message), false, normalizedCode)
	default:
		return newProviderRequestError(fmt.Sprintf("api error: %s", message), true, normalizedCode)
	}
}

type openAICompatibleRequest struct {
	Model           string                    `json:"model"`
	Messages        []openAICompatibleMessage `json:"messages"`
	Temperature     float64                   `json:"temperature,omitempty"`
	TopP            float64                   `json:"top_p,omitempty"`
	MaxTokens       int                       `json:"max_tokens,omitempty"`
	ReasoningEffort string                    `json:"reasoning_effort,omitempty"`
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
		return nil, newProviderRequestError(fmt.Sprintf("process image url: %v", err), false, "local_image_processing")
	}

	req := openAICompatibleRequest{
		Model:           model,
		Temperature:     0.25,
		MaxTokens:       10000,
		TopP:            0.7,
		ReasoningEffort: "high",
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
		return nil, newProviderRequestError(fmt.Sprintf("marshal request: %v", err), false, "marshal_request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, newProviderRequestError(fmt.Sprintf("create request: %v", err), false, "create_request")
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
		return nil, newProviderRequestError("rate limit exceeded", true, "rate_limit")
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, newProviderRequestError(fmt.Sprintf("service unavailable: %d", resp.StatusCode), true, fmt.Sprintf("http_%d", resp.StatusCode))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newProviderRequestError(fmt.Sprintf("unexpected status code: %d", resp.StatusCode), false, fmt.Sprintf("http_%d", resp.StatusCode))
	}

	var providerResp openAICompatibleResponse
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
	if err := validateLocalImagePath(imageURL); err != nil {
		return "", fmt.Errorf("validate local image path: %w", err)
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
