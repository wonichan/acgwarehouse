//go:build !libvips

package ai

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"math"
	"net/http"
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

var resizeImage = imaging.Resize

func calculateResizeDimensions(width, height int, maxPixels int) (int, int) {
	if width <= 0 || height <= 0 || maxPixels <= 0 {
		return 1, 1
	}

	if width <= maxPixels/height {
		return width, height
	}

	scale := math.Sqrt(float64(maxPixels) / (float64(width) * float64(height)))
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	maxWidth := maxPixels / newHeight
	if maxWidth < 1 {
		maxWidth = 1
	}
	if newWidth > maxWidth {
		newWidth = maxWidth
	}

	maxHeight := maxPixels / newWidth
	if maxHeight < 1 {
		maxHeight = 1
	}
	if newHeight > maxHeight {
		newHeight = maxHeight
	}

	return newWidth, newHeight
}

func CompressImageIfNeeded(filePath string) ([]byte, string, error) {
	if strings.EqualFold(filepath.Ext(filePath), ".gif") {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("read gif file: %w", err)
		}
		if err := validateImageBytes(data, "image/gif"); err != nil {
			return nil, "", err
		}
		return data, "image/gif", nil
	}

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
		contentType := http.DetectContentType(data)
		if err := validateImageBytes(data, ""); err != nil {
			return nil, "", err
		}
		return data, contentType, nil
	}

	if pixels > maxAIPixelCount {
		newWidth, newHeight := calculateResizeDimensions(width, height, targetPixelCount)
		img = resizeImage(img, newWidth, newHeight, imaging.Lanczos)
		width, height = newWidth, newHeight
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

func validateImageBytes(data []byte, expected string) error {
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("invalid image content type: %s", contentType)
	}
	if expected != "" && contentType != expected {
		return fmt.Errorf("unexpected image content type: %s", contentType)
	}
	return nil
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
	case ".jpg":
		return "image/jpeg"
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
	maxIterations := 10
	minQuality := 50
	currentImg := img
	currentScale := 1.0
	var lastEncoded []byte

	for i := 0; i < maxIterations; i++ {
		var buf bytes.Buffer
		if err := imaging.Encode(&buf, currentImg, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
			return nil, fmt.Errorf("encode image: %w", err)
		}
		lastEncoded = buf.Bytes()

		if len(lastEncoded) <= maxSize {
			return lastEncoded, nil
		}

		if quality > minQuality {
			quality -= 10
			if quality < minQuality {
				quality = minQuality
			}
			continue
		}

		nextScale := currentScale - 0.15
		if nextScale <= 0.1 {
			return lastEncoded, nil
		}

		relativeScale := nextScale / currentScale
		newWidth := int(float64(currentImg.Bounds().Dx()) * relativeScale)
		if newWidth < 1 {
			newWidth = 1
		}

		currentImg = resizeImage(currentImg, newWidth, 0, filter)
		currentScale = nextScale
	}

	if len(lastEncoded) > 0 {
		return lastEncoded, nil
	}

	return nil, fmt.Errorf("failed to encode image")
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
