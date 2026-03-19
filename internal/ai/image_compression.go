package ai

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

const (
	maxAIImageSize      = 10 * 1024 * 1024 // 10MB - target max size for AI
	smallFileThreshold  = 5 * 1024 * 1024  // 5MB - use Lanczos
	mediumFileThreshold = 10 * 1024 * 1024 // 10MB - use Linear
	// > 10MB: use Box with pre-scale
)

// CompressImageIfNeeded reads an image file and returns bytes ready for base64 encoding.
// If the file exceeds 10MB, it compresses the image until under the limit.
// Uses tiered strategy based on original file size:
// - < 5MB: Lanczos (highest quality)
// - 5-10MB: Linear (good quality, faster)
// - > 10MB: Box with pre-scale (fastest)
// Returns: (imageData, contentType, error)
func CompressImageIfNeeded(filePath string) ([]byte, string, error) {
	// 1. Get file size to determine strategy
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	// 2. Read original file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}

	// Detect content type from file extension
	contentType := detectContentType(filePath)

	// 3. If size <= 10MB, return as-is
	if len(data) <= maxAIImageSize {
		return data, contentType, nil
	}

	// 4. Load image with imaging.Open
	img, err := imaging.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("open image: %w", err)
	}

	// 5. Select filter based on file size tier
	filter := selectResizeFilterForAI(fileSize)

	// 6. Pre-scale large images to reduce memory and processing time
	workingImg := img
	if fileSize >= mediumFileThreshold {
		// Estimate target width based on desired output size
		// For AI, we typically need max 2000px width for good quality
		maxWidth := 2000
		if img.Bounds().Dx() > maxWidth {
			workingImg = imaging.Resize(img, maxWidth, 0, imaging.Box)
		}
	}

	// 7. Compress the image
	compressedData, err := compressImageWithFilter(workingImg, filter)
	if err != nil {
		return nil, "", fmt.Errorf("compress image: %w", err)
	}

	// Compressed output is always JPEG
	return compressedData, "image/jpeg", nil
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
