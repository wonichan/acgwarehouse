package service

import (
	"bytes"
	"fmt"
	"image"
	"os"

	"github.com/disintegration/imaging"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	_ "golang.org/x/image/webp"
)

const (
	// minSmallSize is the minimum size for small thumbnails (200KB)
	minSmallSize = 200 * 1024
	// minLargeSize is the minimum size for large thumbnails (500KB)
	minLargeSize = 500 * 1024
	// maxLargeSize is the maximum size for large thumbnails (1MB)
	maxLargeSize = 1024 * 1024
	// maxAdjustIterations limits the number of adjustment iterations to prevent infinite loops
	maxAdjustIterations = 10

	// File size thresholds for tiered compression strategy
	smallFileThreshold  = 5 * 1024 * 1024  // 5MB - use existing Lanczos
	mediumFileThreshold = 10 * 1024 * 1024 // 10MB - use Linear interpolation
	// > 10MB: use pre-scale strategy with Box interpolation
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
		LargeWidth:   500,
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
// For large: use GenerateBoth which ensures large > small
func (s *ThumbnailService) GenerateThumbnailDynamic(imgPath string, size string) (*domain.Thumbnail, error) {
	// Get file size to determine strategy
	fileSize, err := getFileSize(imgPath)
	if err != nil {
		return nil, fmt.Errorf("get file size: %w", err)
	}

	src, err := imaging.Open(imgPath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}

	width, quality, err := s.paramsBySize(size)
	if err != nil {
		return nil, err
	}

	// Select filter based on file size tier
	filter := selectResizeFilter(fileSize)

	// For large files (>10MB), pre-scale to reduce memory
	if fileSize >= mediumFileThreshold {
		// Pre-scale to 4x the target width for better quality
		maxPreScaleWidth := width * 4
		if s.LargeWidth > width {
			maxPreScaleWidth = s.LargeWidth * 4
		}
		src = preScaleForLargeImage(src, maxPreScaleWidth)
	}

	// For small thumbnails, we may need to increase size
	if size == "small" {
		return s.generateSmallWithMinSize(src, width, quality, filter)
	}

	// For large thumbnails (standalone call), use standard generation with bounds
	if size == "large" {
		return s.generateLargeStandalone(src, width, quality, filter)
	}

	// Fallback to standard generation
	return s.generateThumbnail(src, width, quality, size, filter)
}

// generateSmallWithMinSize generates a small thumbnail ensuring it's at least minSmallSize
func (s *ThumbnailService) generateSmallWithMinSize(src image.Image, width, quality int, filter imaging.ResampleFilter) (*domain.Thumbnail, error) {
	currentWidth := width
	currentQuality := quality
	srcWidth := src.Bounds().Dx()

	for i := 0; i < maxAdjustIterations; i++ {
		thumb, err := s.generateThumbnail(src, currentWidth, currentQuality, "small", filter)
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
	return s.generateThumbnail(src, currentWidth, currentQuality, "small", filter)
}

// generateLargeWithMaxSize generates a large thumbnail ensuring it's within size bounds
// It must be: minLargeSize <= size <= maxLargeSize AND larger than small thumbnail
func (s *ThumbnailService) generateLargeWithMaxSize(src image.Image, width, quality int, smallSize int, filter imaging.ResampleFilter) (*domain.Thumbnail, error) {
	currentWidth := width
	currentQuality := quality
	srcWidth := src.Bounds().Dx()

	// First, ensure large is at least minLargeSize (500KB) and larger than small
	for i := 0; i < maxAdjustIterations; i++ {
		thumb, err := s.generateThumbnail(src, currentWidth, currentQuality, "large", filter)
		if err != nil {
			return nil, err
		}

		thumbSize := len(thumb.Data)

		// Check if size meets minimum requirement (must be larger than small AND >= minLargeSize)
		minRequired := minLargeSize
		if smallSize >= minLargeSize {
			// small is already at/past minLargeSize, so large must be even larger
			minRequired = smallSize + 100*1024 // at least 100KB larger than small
		}

		if thumbSize >= minRequired && thumbSize <= maxLargeSize {
			return thumb, nil
		}

		// If too small, increase width or quality
		if thumbSize < minRequired {
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

			// Can't increase further, return current result (best effort)
			return thumb, nil
		}

		// If too large, decrease quality
		if thumbSize > maxLargeSize {
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
	}

	// Return last generated thumbnail after max iterations
	return s.generateThumbnail(src, currentWidth, currentQuality, "large", filter)
}

// generateLargeStandalone generates a large thumbnail without comparison to small
// Used when GenerateThumbnailDynamic is called independently for "large"
func (s *ThumbnailService) generateLargeStandalone(src image.Image, width, quality int, filter imaging.ResampleFilter) (*domain.Thumbnail, error) {
	currentWidth := width
	currentQuality := quality
	srcWidth := src.Bounds().Dx()

	for i := 0; i < maxAdjustIterations; i++ {
		thumb, err := s.generateThumbnail(src, currentWidth, currentQuality, "large", filter)
		if err != nil {
			return nil, err
		}

		thumbSize := len(thumb.Data)

		// Check if size meets bounds: minLargeSize <= size <= maxLargeSize
		if thumbSize >= minLargeSize && thumbSize <= maxLargeSize {
			return thumb, nil
		}

		// If too small, increase width or quality
		if thumbSize < minLargeSize {
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

			// Can't increase further, return current result (best effort)
			return thumb, nil
		}

		// If too large, decrease quality
		if thumbSize > maxLargeSize {
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
	}

	// Return last generated thumbnail after max iterations
	return s.generateThumbnail(src, currentWidth, currentQuality, "large", filter)
}

// generateThumbnail is a helper that creates a thumbnail with given parameters
func (s *ThumbnailService) generateThumbnail(src image.Image, width, quality int, size string, filter imaging.ResampleFilter) (*domain.Thumbnail, error) {
	resized := imaging.Resize(src, width, 0, filter)

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
	// Get file size to determine strategy
	fileSize, err := getFileSize(imgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("get file size: %w", err)
	}

	src, err := imaging.Open(imgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open image: %w", err)
	}

	// Select filter based on file size tier
	filter := selectResizeFilter(fileSize)

	// For large files (>10MB), pre-scale to reduce memory
	workingImg := src
	if fileSize >= mediumFileThreshold {
		// Pre-scale to 4x the large width for better quality
		maxPreScaleWidth := s.LargeWidth * 4
		workingImg = preScaleForLargeImage(src, maxPreScaleWidth)
	}

	// Generate small first
	smallWidth, smallQuality, _ := s.paramsBySize("small")
	small, err = s.generateSmallWithMinSize(workingImg, smallWidth, smallQuality, filter)
	if err != nil {
		return nil, nil, err
	}

	// Generate large, ensuring it's larger than small
	largeWidth, largeQuality, _ := s.paramsBySize("large")
	large, err = s.generateLargeWithMaxSize(workingImg, largeWidth, largeQuality, len(small.Data), filter)
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

// getFileSize returns the file size in bytes
func getFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("stat file: %w", err)
	}
	return info.Size(), nil
}

// selectResizeFilter returns the appropriate resize filter based on file size
// < 5MB:  Lanczos (highest quality)
// 5-10MB: Linear (good quality, faster)
// > 10MB: Box (fastest, with pre-scale)
func selectResizeFilter(fileSize int64) imaging.ResampleFilter {
	switch {
	case fileSize < smallFileThreshold:
		return imaging.Lanczos
	case fileSize < mediumFileThreshold:
		return imaging.Linear
	default:
		return imaging.Box
	}
}

// preScaleForLargeImage pre-scales large images to reduce memory and processing time
// Returns a pre-scaled image that is at most maxPreScaleWidth pixels wide
func preScaleForLargeImage(img image.Image, maxPreScaleWidth int) image.Image {
	srcWidth := img.Bounds().Dx()
	if srcWidth <= maxPreScaleWidth {
		return img
	}
	// Use Box filter for fast downscaling
	return imaging.Resize(img, maxPreScaleWidth, 0, imaging.Box)
}
