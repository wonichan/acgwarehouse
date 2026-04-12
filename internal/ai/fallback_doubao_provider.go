package ai

import (
	"context"
	"errors"
	"fmt"
)

type FallbackDoubaoProvider struct {
	clients []*DoubaoProvider
}

func NewFallbackDoubaoProvider(batchMode string, providers ...*DoubaoProvider) *FallbackDoubaoProvider {
	normalizedMode := normalizeDoubaoBatchMode(batchMode)
	clonedProviders := make([]*DoubaoProvider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		clone := *provider
		clone.batchMode = normalizedMode
		clonedProviders = append(clonedProviders, &clone)
	}
	if len(clonedProviders) == 0 {
		panic("doubao fallback provider requires at least one model client")
	}
	return &FallbackDoubaoProvider{clients: clonedProviders}
}

func (p *FallbackDoubaoProvider) Name() string {
	return "doubao"
}

func (p *FallbackDoubaoProvider) GenerateTags(ctx context.Context, imageURL, prompt string) (*TagResult, error) {
	if len(p.clients) == 0 {
		return nil, errors.New("no doubao providers configured")
	}

	var lastErr error
	for _, client := range p.clients {
		result, err := client.GenerateTags(ctx, imageURL, prompt)
		if err == nil {
			return result, nil
		}
		if !isRetryableDoubaoError(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all doubao providers failed: %w", lastErr)
}

func (p *FallbackDoubaoProvider) GenerateTagsBatch(ctx context.Context, requests []TagRequest) (*BatchTagResult, error) {
	if len(p.clients) == 0 {
		return nil, errors.New("no doubao providers configured")
	}

	var lastErr error
	for _, client := range p.clients {
		result, err := client.GenerateTagsBatch(ctx, requests)
		if err == nil {
			return result, nil
		}
		if !isRetryableDoubaoError(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all doubao providers failed for batch: %w", lastErr)
}

func isRetryableDoubaoError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var requestErr *providerRequestError
	if errors.As(err, &requestErr) {
		return requestErr.retryable
	}
	if err.Error() == "no doubao providers configured" {
		return false
	}

	return true
}
