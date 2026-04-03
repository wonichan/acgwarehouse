package service

import "github.com/wonichan/acgwarehouse-backend/internal/domain"

// ResolveAITagImagePath returns the best image source for AI tagging.
// Prefer the large thumbnail URL when available to avoid uploading originals.
func ResolveAITagImagePath(image *domain.Image) string {
	if image == nil {
		return ""
	}
	if image.ThumbnailLargeUrl != "" {
		return image.ThumbnailLargeUrl
	}
	return image.Path
}
