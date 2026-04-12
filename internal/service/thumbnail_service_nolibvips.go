//go:build !libvips

package service

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"os"
	"time"

	"github.com/disintegration/imaging"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	_ "golang.org/x/image/webp"
)

const (
	minSmallSize        = 200 * 1024
	minLargeSize        = 500 * 1024
	maxLargeSize        = 1024 * 1024
	maxAdjustIterations = 10

	smallFileThreshold  = 5 * 1024 * 1024
	mediumFileThreshold = 10 * 1024 * 1024
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
		LargeWidth:   800,
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

func (s *ThumbnailService) GenerateThumbnailDynamic(imgPath string, size string) (*domain.Thumbnail, error) {
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

	filter := selectResizeFilter(fileSize)

	if fileSize >= mediumFileThreshold {
		maxPreScaleWidth := width * 4
		if s.LargeWidth > width {
			maxPreScaleWidth = s.LargeWidth * 4
		}
		src = preScaleForLargeImage(src, maxPreScaleWidth)
	}

	if size == "small" {
		return s.generateSmallWithMinSize(src, width, quality, filter)
	}

	if size == "large" {
		return s.generateLargeStandalone(src, width, quality, filter)
	}

	return s.generateThumbnail(src, width, quality, size, filter)
}

func (s *ThumbnailService) generateSmallWithMinSize(src image.Image, width, quality int, filter imaging.ResampleFilter) (*domain.Thumbnail, error) {
	currentWidth := width
	currentQuality := quality
	srcWidth := src.Bounds().Dx()

	for i := 0; i < maxAdjustIterations; i++ {
		thumb, err := s.generateThumbnail(src, currentWidth, currentQuality, "small", filter)
		if err != nil {
			return nil, err
		}

		if len(thumb.Data) >= minSmallSize {
			return thumb, nil
		}

		if currentWidth < srcWidth {
			newWidth := currentWidth + 100
			if newWidth > srcWidth {
				newWidth = srcWidth
			}
			currentWidth = newWidth
			continue
		}

		if currentQuality < 100 {
			currentQuality += 5
			if currentQuality > 100 {
				currentQuality = 100
			}
			continue
		}

		return thumb, nil
	}

	return s.generateThumbnail(src, currentWidth, currentQuality, "small", filter)
}

func (s *ThumbnailService) generateLargeWithMaxSize(src image.Image, width, quality int, smallSize int, filter imaging.ResampleFilter) (*domain.Thumbnail, error) {
	currentWidth := width
	currentQuality := quality
	srcWidth := src.Bounds().Dx()

	for i := 0; i < maxAdjustIterations; i++ {
		thumb, err := s.generateThumbnail(src, currentWidth, currentQuality, "large", filter)
		if err != nil {
			return nil, err
		}

		thumbSize := len(thumb.Data)
		minRequired := minLargeSize
		if smallSize >= minLargeSize {
			minRequired = smallSize + 100*1024
		}

		if thumbSize >= minRequired && thumbSize <= maxLargeSize {
			return thumb, nil
		}

		if thumbSize < minRequired {
			if currentWidth < srcWidth {
				newWidth := currentWidth + 100
				if newWidth > srcWidth {
					newWidth = srcWidth
				}
				currentWidth = newWidth
				continue
			}

			if currentQuality < 100 {
				currentQuality += 5
				if currentQuality > 100 {
					currentQuality = 100
				}
				continue
			}

			return thumb, nil
		}

		if thumbSize > maxLargeSize {
			if currentQuality > 10 {
				currentQuality -= 5
				if currentQuality < 10 {
					currentQuality = 10
				}
				continue
			}
			return thumb, nil
		}
	}

	return s.generateThumbnail(src, currentWidth, currentQuality, "large", filter)
}

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
		if thumbSize >= minLargeSize && thumbSize <= maxLargeSize {
			return thumb, nil
		}

		if thumbSize < minLargeSize {
			if currentWidth < srcWidth {
				newWidth := currentWidth + 100
				if newWidth > srcWidth {
					newWidth = srcWidth
				}
				currentWidth = newWidth
				continue
			}

			if currentQuality < 100 {
				currentQuality += 5
				if currentQuality > 100 {
					currentQuality = 100
				}
				continue
			}

			return thumb, nil
		}

		if thumbSize > maxLargeSize {
			if currentQuality > 10 {
				currentQuality -= 5
				if currentQuality < 10 {
					currentQuality = 10
				}
				continue
			}
			return thumb, nil
		}
	}

	return s.generateThumbnail(src, currentWidth, currentQuality, "large", filter)
}

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
	startedAt := time.Now()
	log.Printf("thumbnail generate-both started: path=%s", imgPath)

	fileSize, err := getFileSize(imgPath)
	if err != nil {
		log.Printf("thumbnail stat failed: path=%s error=%v", imgPath, err)
		return nil, nil, fmt.Errorf("get file size: %w", err)
	}

	src, err := imaging.Open(imgPath)
	if err != nil {
		log.Printf("thumbnail open failed: path=%s error=%v", imgPath, err)
		return nil, nil, fmt.Errorf("open image: %w", err)
	}
	log.Printf("thumbnail source loaded: path=%s file_size=%d width=%d height=%d", imgPath, fileSize, src.Bounds().Dx(), src.Bounds().Dy())

	filter := selectResizeFilter(fileSize)

	workingImg := src
	if fileSize >= mediumFileThreshold {
		maxPreScaleWidth := s.LargeWidth * 4
		workingImg = preScaleForLargeImage(src, maxPreScaleWidth)
		log.Printf("thumbnail pre-scale applied: path=%s max_pre_scale_width=%d working_width=%d working_height=%d", imgPath, maxPreScaleWidth, workingImg.Bounds().Dx(), workingImg.Bounds().Dy())
	}

	smallWidth, smallQuality, _ := s.paramsBySize("small")
	small, err = s.generateSmallWithMinSize(workingImg, smallWidth, smallQuality, filter)
	if err != nil {
		log.Printf("thumbnail small generation failed: path=%s width=%d quality=%d error=%v", imgPath, smallWidth, smallQuality, err)
		return nil, nil, err
	}
	log.Printf("thumbnail small generated: path=%s bytes=%d width=%d height=%d", imgPath, len(small.Data), small.Width, small.Height)

	largeWidth, largeQuality, _ := s.paramsBySize("large")
	large, err = s.generateLargeWithMaxSize(workingImg, largeWidth, largeQuality, len(small.Data), filter)
	if err != nil {
		log.Printf("thumbnail large generation failed: path=%s width=%d quality=%d small_bytes=%d error=%v", imgPath, largeWidth, largeQuality, len(small.Data), err)
		return nil, nil, err
	}
	log.Printf(
		"thumbnail generate-both completed: path=%s duration=%s small_bytes=%d small_width=%d small_height=%d large_bytes=%d large_width=%d large_height=%d",
		imgPath,
		time.Since(startedAt),
		len(small.Data),
		small.Width,
		small.Height,
		len(large.Data),
		large.Width,
		large.Height,
	)

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

func getFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("stat file: %w", err)
	}
	return info.Size(), nil
}

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

func preScaleForLargeImage(img image.Image, maxPreScaleWidth int) image.Image {
	srcWidth := img.Bounds().Dx()
	if srcWidth <= maxPreScaleWidth {
		return img
	}
	return imaging.Resize(img, maxPreScaleWidth, 0, imaging.Box)
}
