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

const maxAIImageSize = 10 * 1024 * 1024 // 10MB

// CompressImageIfNeeded reads an image file and returns bytes ready for base64 encoding.
// If the file exceeds 10MB, it compresses the image until under the limit.
// Returns: (imageData, contentType, error)
func CompressImageIfNeeded(filePath string) ([]byte, string, error) {
	// 1. Read original file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}

	// Detect content type from file extension
	contentType := detectContentType(filePath)

	// 2. If size <= 10MB, return as-is
	if len(data) <= maxAIImageSize {
		return data, contentType, nil
	}

	// 3. Load image with imaging.Open
	img, err := imaging.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("open image: %w", err)
	}

	// 4. Compress the image
	compressedData, err := compressImage(img)
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

// compressImage compresses an image to be under 10MB
func compressImage(img image.Image) ([]byte, error) {
	// Start with quality 90, reduce by 5 each iteration
	// Minimum quality: 50
	// If quality at 50 still > 10MB, reduce dimensions by 10% and retry
	// Maximum 20 iterations to prevent infinite loops

	quality := 90
	scale := 1.0
	maxIterations := 20
	minQuality := 50

	for i := 0; i < maxIterations; i++ {
		// Apply scale if needed
		currentImg := img
		if scale < 1.0 {
			newWidth := int(float64(img.Bounds().Dx()) * scale)
			currentImg = imaging.Resize(img, newWidth, 0, imaging.Lanczos)
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

		// Reduce quality first
		if quality > minQuality {
			quality -= 5
			continue
		}

		// If quality at minimum and still too large, reduce dimensions
		scale -= 0.1
		if scale <= 0.1 {
			// Can't reduce further, return current result
			return buf.Bytes(), nil
		}
	}

	// Fallback: return last encoding attempt
	var buf bytes.Buffer
	scaledImg := imaging.Resize(img, int(float64(img.Bounds().Dx())*scale), 0, imaging.Lanczos)
	if err := imaging.Encode(&buf, scaledImg, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}
	return buf.Bytes(), nil
}
