package service

import (
	"bytes"
	"fmt"
	"image"

	"github.com/disintegration/imaging"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	_ "golang.org/x/image/webp"
)

const (
	// minSmallSize is the minimum size for small thumbnails (200KB)
	minSmallSize = 200 * 1024
	// maxLargeSize is the maximum size for large thumbnails (1MB)
	maxLargeSize = 1024 * 1024
	// maxAdjustIterations limits the number of adjustment iterations to prevent infinite loops
	maxAdjustIterations = 10
)

type ThumbnailService struct {
	SmallWidth   int
	LargeWidth   int
	SmallQuality int
	LargeQuality int
}

func NewThumbnailService() *ThumbnailService {
	return &ThumbnailService{
		SmallWidth:   200,
		LargeWidth:   600,
		SmallQuality: 85,
		LargeQuality: 90,
	}
}

func (s *ThumbnailService) GenerateThumbnail(imgPath string, size string) (*domain.Thumbnail, error) {
	src, err := imaging.Open(imgPath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}

	width, quality, err := s.paramsBySize(size)
	if err != nil {
		return nil, err
	}

	resized := imaging.Resize(src, width, 0, imaging.Lanczos)

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		return nil, fmt.Errorf("encode thumbnail: %w", err)
	}

	return &domain.Thumbnail{
		Data:   buf.Bytes(),
		Width:  resized.Bounds().Dx(),
		Height: resized.Bounds().Dy(),
		Size:   size,
	}, nil
}

// GenerateThumbnailDynamic generates a thumbnail with dynamic size adjustment
// For small: ensures the output is at least minSmallSize (200KB)
// For large: ensures the output is at most maxLargeSize (1MB)
func (s *ThumbnailService) GenerateThumbnailDynamic(imgPath string, size string) (*domain.Thumbnail, error) {
	src, err := imaging.Open(imgPath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}

	width, quality, err := s.paramsBySize(size)
	if err != nil {
		return nil, err
	}

	// For small thumbnails, we may need to increase size
	if size == "small" {
		return s.generateSmallWithMinSize(src, width, quality)
	}

	// For large thumbnails, we may need to decrease size
	if size == "large" {
		return s.generateLargeWithMaxSize(src, width, quality)
	}

	// Fallback to standard generation
	return s.generateThumbnail(src, width, quality, size)
}

// generateSmallWithMinSize generates a small thumbnail ensuring it's at least minSmallSize
func (s *ThumbnailService) generateSmallWithMinSize(src image.Image, width, quality int) (*domain.Thumbnail, error) {
	currentWidth := width
	currentQuality := quality
	srcWidth := src.Bounds().Dx()

	for i := 0; i < maxAdjustIterations; i++ {
		thumb, err := s.generateThumbnail(src, currentWidth, currentQuality, "small")
		if err != nil {
			return nil, err
		}

		// Check if size meets minimum requirement
		if len(thumb.Data) >= minSmallSize {
			return thumb, nil
		}

		// Try increasing width first (up to source image width)
		if currentWidth < srcWidth {
			newWidth := currentWidth + 100
			if newWidth > srcWidth {
				newWidth = srcWidth
			}
			currentWidth = newWidth
			continue
		}

		// If width is already max, try increasing quality
		if currentQuality < 100 {
			currentQuality += 5
			if currentQuality > 100 {
				currentQuality = 100
			}
			continue
		}

		// Can't increase further, return current result
		return thumb, nil
	}

	// Return last generated thumbnail after max iterations
	return s.generateThumbnail(src, currentWidth, currentQuality, "small")
}

// generateLargeWithMaxSize generates a large thumbnail ensuring it's at most maxLargeSize
func (s *ThumbnailService) generateLargeWithMaxSize(src image.Image, width, quality int) (*domain.Thumbnail, error) {
	currentQuality := quality

	for i := 0; i < maxAdjustIterations; i++ {
		thumb, err := s.generateThumbnail(src, width, currentQuality, "large")
		if err != nil {
			return nil, err
		}

		// Check if size meets maximum requirement
		if len(thumb.Data) <= maxLargeSize {
			return thumb, nil
		}

		// Try decreasing quality
		if currentQuality > 10 {
			currentQuality -= 5
			if currentQuality < 10 {
				currentQuality = 10
			}
			continue
		}

		// Can't decrease further, return current result
		return thumb, nil
	}

	// Return last generated thumbnail after max iterations
	return s.generateThumbnail(src, width, currentQuality, "large")
}

// generateThumbnail is a helper that creates a thumbnail with given parameters
func (s *ThumbnailService) generateThumbnail(src image.Image, width, quality int, size string) (*domain.Thumbnail, error) {
	resized := imaging.Resize(src, width, 0, imaging.Lanczos)

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		return nil, fmt.Errorf("encode thumbnail: %w", err)
	}

	return &domain.Thumbnail{
		Data:   buf.Bytes(),
		Width:  resized.Bounds().Dx(),
		Height: resized.Bounds().Dy(),
		Size:   size,
	}, nil
}

func (s *ThumbnailService) GenerateBoth(imgPath string) (small, large *domain.Thumbnail, err error) {
	small, err = s.GenerateThumbnailDynamic(imgPath, "small")
	if err != nil {
		return nil, nil, err
	}

	large, err = s.GenerateThumbnailDynamic(imgPath, "large")
	if err != nil {
		return nil, nil, err
	}

	return small, large, nil
}

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
