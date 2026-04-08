package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func rewriteImageForRequest(r *http.Request, image domain.Image) domain.Image {
	image.ThumbnailSmallUrl = rewriteThumbnailURLForRequest(r, image.ThumbnailSmallUrl)
	image.ThumbnailLargeUrl = rewriteThumbnailURLForRequest(r, image.ThumbnailLargeUrl)
	return image
}

func rewriteImagesForRequest(r *http.Request, images []domain.Image) []domain.Image {
	if len(images) == 0 {
		return images
	}
	rewritten := make([]domain.Image, len(images))
	for i, image := range images {
		rewritten[i] = rewriteImageForRequest(r, image)
	}
	return rewritten
}

func rewriteViewerItemsForRequest(r *http.Request, items []any) []any {
	if len(items) == 0 {
		return items
	}
	rewritten := make([]any, len(items))
	for i, item := range items {
		image, ok := item.(domain.Image)
		if !ok {
			rewritten[i] = item
			continue
		}
		rewritten[i] = rewriteImageForRequest(r, image)
	}
	return rewritten
}

func rewriteThumbnailURLForRequest(r *http.Request, rawURL string) string {
	if r == nil || r.Host == "" || rawURL == "" {
		return rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || !parsed.IsAbs() {
		return rawURL
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "localhost" && host != "127.0.0.1" {
		return rawURL
	}

	parsed.Scheme = requestScheme(r)
	parsed.Host = r.Host
	return parsed.String()
}

func requestScheme(r *http.Request) string {
	if forwardedProto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); forwardedProto != "" {
		return strings.ToLower(forwardedProto)
	}
	if r.URL != nil && r.URL.Scheme != "" {
		return strings.ToLower(r.URL.Scheme)
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
