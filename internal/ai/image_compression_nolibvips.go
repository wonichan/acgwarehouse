//go:build !libvips

package ai

import (
	"bytes"
	"encoding/base64"
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
	maxAIImageSize      = 10 * 1024 * 1024
	maxAIPixelCount     = 36000000
	targetPixelCount    = 30000000
	maxAIDataURLSize    = 10 * 1024 * 1024
	smallFileThreshold  = 5 * 1024 * 1024
	mediumFileThreshold = 10 * 1024 * 1024
)

func calculateResizeDimensions(width, height int, maxPixels int) (int, int) {
	pixels := width * height
	if pixels <= maxPixels {
		return width, height
	}

	scale := math.Sqrt(float64(maxPixels) / float64(pixels))
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	return newWidth, newHeight
}

func CompressImageIfNeeded(filePath string) ([]byte, string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	img, err := imaging.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("open image: %w", err)
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	pixels := width * height

	if pixels <= maxAIPixelCount && fileSize <= int64(maxAIImageSize) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("read file: %w", err)
		}
		contentType := detectContentType(filePath)
		return data, contentType, nil
	}

	if pixels > maxAIPixelCount {
		newWidth, newHeight := calculateResizeDimensions(width, height, targetPixelCount)
		img = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
		width, height = newWidth, newHeight
		_ = width
		_ = height
	}

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
		return nil, "", fmt.Errorf("encode image: %w", err)
	}
	data := buf.Bytes()
	contentType := "image/jpeg"

	if len(data) <= maxAIImageSize {
		return data, contentType, nil
	}

	filter := selectResizeFilterForAI(fileSize)
	compressedData, err := compressImageWithFilter(img, filter)
	if err != nil {
		return nil, "", fmt.Errorf("compress image: %w", err)
	}

	return compressedData, contentType, nil
}

func PrepareImageForDataURL(filePath string) ([]byte, string, error) {
	data, contentType, err := CompressImageIfNeeded(filePath)
	if err != nil {
		return nil, "", err
	}

	if err := ensureDataURLFitsBudget(contentType, data); err == nil {
		return data, contentType, nil
	}

	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode processed image: %w", err)
	}

	filter := selectResizeFilterForAI(int64(len(data)))
	adjustedData, err := compressImageWithFilterAndLimit(img, filter, maxDecodedDataURLSize(contentType))
	if err != nil {
		return nil, "", fmt.Errorf("compress image for data url: %w", err)
	}
	if err := ensureDataURLFitsBudget("image/jpeg", adjustedData); err != nil {
		return nil, "", err
	}

	return adjustedData, "image/jpeg", nil
}

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

func compressImageWithFilter(img image.Image, filter imaging.ResampleFilter) ([]byte, error) {
	return compressImageWithFilterAndLimit(img, filter, maxAIImageSize)
}

func compressImageWithFilterAndLimit(img image.Image, filter imaging.ResampleFilter, maxSize int) ([]byte, error) {
	quality := 85
	scale := 1.0
	maxIterations := 10
	minQuality := 50

	for i := 0; i < maxIterations; i++ {
		currentImg := img
		if scale < 1.0 {
			newWidth := int(float64(img.Bounds().Dx()) * scale)
			currentImg = imaging.Resize(img, newWidth, 0, filter)
		}

		var buf bytes.Buffer
		if err := imaging.Encode(&buf, currentImg, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
			return nil, fmt.Errorf("encode image: %w", err)
		}

		if buf.Len() <= maxSize {
			return buf.Bytes(), nil
		}

		if quality > minQuality {
			quality -= 10
			if quality < minQuality {
				quality = minQuality
			}
			continue
		}

		scale -= 0.15
		if scale <= 0.1 {
			return buf.Bytes(), nil
		}
	}

	var buf bytes.Buffer
	scaledImg := imaging.Resize(img, int(float64(img.Bounds().Dx())*scale), 0, filter)
	if err := imaging.Encode(&buf, scaledImg, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}
	return buf.Bytes(), nil
}

func dataURLSize(contentType string, dataSize int) int {
	return len("data:") + len(contentType) + len(";base64,") + base64.StdEncoding.EncodedLen(dataSize)
}

func maxDecodedDataURLSize(contentType string) int {
	encodedBudget := maxAIDataURLSize - len("data:") - len(contentType) - len(";base64,")
	if encodedBudget <= 0 {
		return 0
	}
	return (encodedBudget / 4) * 3
}

func ensureDataURLFitsBudget(contentType string, data []byte) error {
	size := dataURLSize(contentType, len(data))
	if size > maxAIDataURLSize {
		return fmt.Errorf("data url payload exceeds budget: got %d bytes, limit %d", size, maxAIDataURLSize)
	}
	return nil
}
