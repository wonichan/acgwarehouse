package service

import (
	"net/url"
	"path"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
)

func BuildThumbnailBaseURL(endpoint string, useSSL bool) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return strings.TrimRight(trimmed, "/")
	}
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	return scheme + "://" + strings.TrimRight(trimmed, "/")
}

func ResolveThumbnailBaseURL(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	return BuildThumbnailBaseURL(cfg.Minio.Endpoint, cfg.Minio.UseSSL)
}

func ResolveThumbnailURL(baseURL, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.IsAbs() {
		return trimmed
	}
	resolvedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if resolvedBase == "" {
		return trimmed
	}
	relative := strings.TrimLeft(strings.ReplaceAll(trimmed, "\\", "/"), "/")
	if relative == "" {
		return trimmed
	}
	return resolvedBase + "/" + relative
}

func NormalizeThumbnailStoragePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.IsAbs() {
		trimmed = parsed.Path
	}
	normalized := path.Clean(strings.ReplaceAll(trimmed, "\\", "/"))
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "." {
		return ""
	}
	return normalized
}
