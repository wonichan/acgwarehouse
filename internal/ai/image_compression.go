package ai

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

const (
	maxAIImageSize      = 10 * 1024 * 1024 // 10MB - target max size for AI
	maxAIPixelCount     = 36000000         // 36M pixels - API pixel limit
	targetPixelCount    = 30000000         // 30M pixels - target to stay safely under limit
	smallFileThreshold  = 5 * 1024 * 1024  // 5MB - use Lanczos
	mediumFileThreshold = 10 * 1024 * 1024 // 10MB - use Linear
	// > 10MB: use Box with pre-scale
)

// calculateResizeDimensions returns new dimensions that fit within maxPixels
// while preserving the aspect ratio.
func calculateResizeDimensions(width, height int, maxPixels int) (int, int) {
	pixels := width * height
	if pixels <= maxPixels {
		return width, height
	}

	// Calculate scale factor to fit within limit
	// scale = sqrt(maxPixels / pixels)
	scale := math.Sqrt(float64(maxPixels) / float64(pixels))
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	// Ensure at least 1 pixel
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	return newWidth, newHeight
}

// CompressImageIfNeeded reads an image file and returns bytes ready for base64 encoding.
// Checks pixel count FIRST (36M limit) before file size (10MB limit).
// If pixel count exceeds 36M, resizes proportionally to fit under limit.
// If file size exceeds 10MB after resize, compresses further.
// Small files (<=10MB AND <=36M pixels) are returned unchanged, preserving original format.
// Returns: (imageData, contentType, error)
func CompressImageIfNeeded(filePath string) ([]byte, string, error) {
	// 1. Get file size first for early check
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	// 2. Load image to check pixel dimensions
	img, err := imaging.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("open image: %w", err)
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	pixels := width * height

	// 3. Early return: if pixel count AND file size are under limits, return original unchanged
	if pixels <= maxAIPixelCount && fileSize <= int64(maxAIImageSize) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("read file: %w", err)
		}
		contentType := detectContentType(filePath)
		return data, contentType, nil
	}

	// 4. Resize if pixel count exceeds API limit
	if pixels > maxAIPixelCount {
		newWidth, newHeight := calculateResizeDimensions(width, height, targetPixelCount)
		img = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
		// Update dimensions for subsequent checks
		width, height = newWidth, newHeight
	}

	// 5. Encode the (possibly resized) image to check file size
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
		return nil, "", fmt.Errorf("encode image: %w", err)
	}
	data := buf.Bytes()
	contentType := "image/jpeg" // Processed output is always JPEG

	// 6. If size <= 10MB, return as-is
	if len(data) <= maxAIImageSize {
		return data, contentType, nil
	}

	// 7. Select filter based on file size tier
	filter := selectResizeFilterForAI(fileSize)

	// 8. Compress the image further
	compressedData, err := compressImageWithFilter(img, filter)
	if err != nil {
		return nil, "", fmt.Errorf("compress image: %w", err)
	}

	return compressedData, contentType, nil
}

// detectContentType returns the content type based on file extension
func detectContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// selectResizeFilterForAI returns the appropriate resize filter based on file size
// < 5MB:  Lanczos (highest quality)
// 5-10MB: Linear (good quality, faster)
// > 10MB: Box (fastest)
func selectResizeFilterForAI(fileSize int64) imaging.ResampleFilter {
	switch {
	case fileSize < smallFileThreshold:
		return imaging.Lanczos
	case fileSize < mediumFileThreshold:
		return imaging.Linear
	default:
		return imaging.Box
	}
}

// compressImageWithFilter compresses an image to be under 10MB using the specified filter
func compressImageWithFilter(img image.Image, filter imaging.ResampleFilter) ([]byte, error) {
	// Start with quality 85 (reduced from 90 for faster convergence)
	// Reduce by 10 each iteration for faster convergence
	// Minimum quality: 50
	// If quality at 50 still > 10MB, reduce dimensions and retry
	// Maximum 10 iterations (reduced from 20)

	quality := 85
	scale := 1.0
	maxIterations := 10
	minQuality := 50

	for i := 0; i < maxIterations; i++ {
		// Apply scale if needed
		currentImg := img
		if scale < 1.0 {
			newWidth := int(float64(img.Bounds().Dx()) * scale)
			currentImg = imaging.Resize(img, newWidth, 0, filter)
		}

		// Encode as JPEG with current quality
		var buf bytes.Buffer
		if err := imaging.Encode(&buf, currentImg, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
			return nil, fmt.Errorf("encode image: %w", err)
		}

		// Check size
		if buf.Len() <= maxAIImageSize {
			return buf.Bytes(), nil
		}

		// Reduce quality first (larger steps for faster convergence)
		if quality > minQuality {
			quality -= 10
			if quality < minQuality {
				quality = minQuality
			}
			continue
		}

		// If quality at minimum and still too large, reduce dimensions
		scale -= 0.15 // larger scale reduction for faster convergence
		if scale <= 0.1 {
			// Can't reduce further, return current result
			return buf.Bytes(), nil
		}
	}

	// Fallback: return last encoding attempt
	var buf bytes.Buffer
	scaledImg := imaging.Resize(img, int(float64(img.Bounds().Dx())*scale), 0, filter)
	if err := imaging.Encode(&buf, scaledImg, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}
	return buf.Bytes(), nil
}
