package service

import (
	"fmt"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

const (
	minSmallSize        = 200 * 1024
	minLargeSize        = 500 * 1024
	maxLargeSize        = 1024 * 1024
	maxAdjustIterations = 10

	smallFileThreshold  = 5 * 1024 * 1024
	mediumFileThreshold = 10 * 1024 * 1024
)

type thumbnailResizeProfile int

const (
	thumbnailResizeHighQuality thumbnailResizeProfile = iota
	thumbnailResizeBalanced
	thumbnailResizeFast
)

type thumbnailRenderFunc func(width, quality int) (*domain.Thumbnail, error)

func (s *ThumbnailService) paramsBySize(size string) (width, quality int, err error) {
	switch size {
	case "small":
		return s.SmallWidth, s.SmallQuality, nil
	case "large":
		return s.LargeWidth, s.LargeQuality, nil
	default:
		return 0, 0, fmt.Errorf("unsupported thumbnail size: %s", size)
	}
}

func selectThumbnailResizeProfile(fileSize int64) thumbnailResizeProfile {
	switch {
	case fileSize < smallFileThreshold:
		return thumbnailResizeHighQuality
	case fileSize < mediumFileThreshold:
		return thumbnailResizeBalanced
	default:
		return thumbnailResizeFast
	}
}

func shouldPreScaleThumbnail(fileSize int64) bool {
	return fileSize >= mediumFileThreshold
}

func maxDynamicPreScaleWidth(width, largeWidth int) int {
	maxPreScaleWidth := width * 4
	if largeWidth > width {
		maxPreScaleWidth = largeWidth * 4
	}
	return maxPreScaleWidth
}

func maxGenerateBothPreScaleWidth(largeWidth int) int {
	return largeWidth * 4
}

func runSmallThumbnailPolicy(srcWidth, width, quality int, render thumbnailRenderFunc) (*domain.Thumbnail, error) {
	currentWidth := width
	currentQuality := quality

	for i := 0; i < maxAdjustIterations; i++ {
		thumb, err := render(currentWidth, currentQuality)
		if err != nil {
			return nil, err
		}

		if len(thumb.Data) >= minSmallSize {
			return thumb, nil
		}

		if currentWidth < srcWidth {
			currentWidth = increaseThumbnailWidth(currentWidth, srcWidth)
			continue
		}

		if currentQuality < 100 {
			currentQuality = increaseThumbnailQuality(currentQuality)
			continue
		}

		return thumb, nil
	}

	return render(currentWidth, currentQuality)
}

func runLargeThumbnailPolicy(srcWidth, width, quality, minRequired int, render thumbnailRenderFunc) (*domain.Thumbnail, error) {
	currentWidth := width
	currentQuality := quality

	for i := 0; i < maxAdjustIterations; i++ {
		thumb, err := render(currentWidth, currentQuality)
		if err != nil {
			return nil, err
		}

		thumbSize := len(thumb.Data)
		if thumbSize >= minRequired && thumbSize <= maxLargeSize {
			return thumb, nil
		}

		if thumbSize < minRequired {
			if currentWidth < srcWidth {
				currentWidth = increaseThumbnailWidth(currentWidth, srcWidth)
				continue
			}

			if currentQuality < 100 {
				currentQuality = increaseThumbnailQuality(currentQuality)
				continue
			}

			return thumb, nil
		}

		if thumbSize > maxLargeSize {
			if currentQuality > 10 {
				currentQuality = decreaseThumbnailQuality(currentQuality)
				continue
			}
			return thumb, nil
		}
	}

	return render(currentWidth, currentQuality)
}

func minLargeThumbnailSize(smallSize int) int {
	if smallSize >= minLargeSize {
		return smallSize + 100*1024
	}
	return minLargeSize
}

func increaseThumbnailWidth(currentWidth, srcWidth int) int {
	newWidth := currentWidth + 100
	if newWidth > srcWidth {
		return srcWidth
	}
	return newWidth
}

func increaseThumbnailQuality(currentQuality int) int {
	newQuality := currentQuality + 5
	if newQuality > 100 {
		return 100
	}
	return newQuality
}

func decreaseThumbnailQuality(currentQuality int) int {
	newQuality := currentQuality - 5
	if newQuality < 10 {
		return 10
	}
	return newQuality
}
