//go:build libvips

package ai

import (
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/wonichan/acgwarehouse-backend/internal/imageruntime"
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

	if err := imageruntime.EnsureStarted(); err != nil {
		return nil, "", fmt.Errorf("start vips runtime: %w", err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	img, err := vips.NewImageFromFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("open image: %w", err)
	}
	defer img.Close()

	if err := img.AutoRotate(); err != nil {
		return nil, "", fmt.Errorf("autorotate image: %w", err)
	}

	width := img.Width()
	height := img.Height()
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
		scale := float64(newWidth) / float64(width)
		if err := img.Resize(scale, vips.KernelLanczos3); err != nil {
			return nil, "", fmt.Errorf("resize image: %w", err)
		}
		width, height = newWidth, newHeight
	}

	data, err := exportJPEG(img, 85)
	if err != nil {
		return nil, "", fmt.Errorf("encode image: %w", err)
	}
	contentType := "image/jpeg"

	if len(data) <= maxAIImageSize {
		return data, contentType, nil
	}

	kernel := selectResizeKernelForAI(fileSize)
	compressedData, err := compressImageWithKernel(img, kernel)
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

	if err := imageruntime.EnsureStarted(); err != nil {
		return nil, "", fmt.Errorf("start vips runtime: %w", err)
	}

	img, err := vips.NewImageFromBuffer(data)
	if err != nil {
		return nil, "", fmt.Errorf("decode processed image: %w", err)
	}
	defer img.Close()

	kernel := selectResizeKernelForAI(int64(len(data)))
	adjustedData, err := compressImageWithKernelAndLimit(img, kernel, maxDecodedDataURLSize(contentType))
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

func selectResizeKernelForAI(fileSize int64) vips.Kernel {
	switch {
	case fileSize < smallFileThreshold:
		return vips.KernelLanczos3
	case fileSize < mediumFileThreshold:
		return vips.KernelLinear
	default:
		return vips.KernelNearest
	}
}

func compressImageWithKernel(img *vips.ImageRef, kernel vips.Kernel) ([]byte, error) {
	return compressImageWithKernelAndLimit(img, kernel, maxAIImageSize)
}

func compressImageWithKernelAndLimit(img *vips.ImageRef, kernel vips.Kernel, maxSize int) ([]byte, error) {
	quality := 85
	scale := 1.0
	maxIterations := 10
	minQuality := 50
	var lastEncoded []byte

	for i := 0; i < maxIterations; i++ {
		currentImg, err := img.Copy()
		if err != nil {
			return nil, fmt.Errorf("copy image: %w", err)
		}

		if scale < 1.0 {
			if err := currentImg.Resize(scale, kernel); err != nil {
				currentImg.Close()
				return nil, fmt.Errorf("resize image: %w", err)
			}
		}

		encoded, err := exportJPEG(currentImg, quality)
		currentImg.Close()
		if err != nil {
			return nil, fmt.Errorf("encode image: %w", err)
		}
		lastEncoded = encoded

		if len(encoded) <= maxSize {
			return encoded, nil
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
			return lastEncoded, nil
		}
	}

	if len(lastEncoded) > 0 {
		return lastEncoded, nil
	}

	fallbackImg, err := img.Copy()
	if err != nil {
		return nil, fmt.Errorf("copy image fallback: %w", err)
	}
	defer fallbackImg.Close()
	if scale < 1.0 {
		if err := fallbackImg.Resize(scale, kernel); err != nil {
			return nil, fmt.Errorf("resize image fallback: %w", err)
		}
	}

	return exportJPEG(fallbackImg, quality)
}

func exportJPEG(img *vips.ImageRef, quality int) ([]byte, error) {
	if img.HasAlpha() {
		if err := img.Flatten(&vips.Color{R: 255, G: 255, B: 255}); err != nil {
			return nil, fmt.Errorf("flatten alpha for jpeg: %w", err)
		}
	}

	params := vips.NewJpegExportParams()
	params.Quality = quality
	params.StripMetadata = true
	params.OptimizeCoding = true
	params.Interlace = true

	data, _, err := img.ExportJpeg(params)
	if err != nil {
		return nil, err
	}
	return data, nil
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
