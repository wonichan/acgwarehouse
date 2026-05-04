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

	provider := strings.ToLower(strings.TrimSpace(cfg.ThumbnailStorageProvider))
	switch provider {
	case "minio":
		base := BuildThumbnailBaseURL(cfg.Minio.Endpoint, cfg.Minio.UseSSL)
		if !isValidHTTPBaseURL(base) {
			return ""
		}
		return base
	case "cos":
		base := strings.TrimRight(strings.TrimSpace(cfg.COS.BucketURL), "/")
		if !isValidHTTPBaseURL(base) {
			return ""
		}
		return base
	default:
		return ""
	}
}

// ResolveRelativeThumbnailBaseURL resolves the base URL used for relative thumbnail paths.
// Compatibility rule:
// 1) If MinIO base URL is configured and valid, always use it for relative paths.
// 2) Otherwise fallback to provider-resolved base URL (e.g. COS when provider=cos).
func ResolveRelativeThumbnailBaseURL(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}

	minioBase := BuildThumbnailBaseURL(cfg.Minio.Endpoint, cfg.Minio.UseSSL)
	if isValidHTTPBaseURL(minioBase) {
		return minioBase
	}

	return ResolveThumbnailBaseURL(cfg)
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
	if !isValidHTTPBaseURL(resolvedBase) {
		return trimmed
	}
	relative := strings.TrimLeft(strings.ReplaceAll(trimmed, "\\", "/"), "/")
	if relative == "" {
		return trimmed
	}
	return resolvedBase + "/" + relative
}

func isValidHTTPBaseURL(raw string) bool {
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil {
		return false
	}
	if !parsed.IsAbs() || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
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
