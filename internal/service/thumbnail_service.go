//go:build libvips

package service

import (
	"fmt"
	"os"
	"time"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/imageruntime"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
	_ "golang.org/x/image/webp"
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
	if err := imageruntime.EnsureStarted(); err != nil {
		return nil, fmt.Errorf("start vips runtime: %w", err)
	}

	src, err := vips.NewImageFromFile(imgPath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer src.Close()

	if err := src.AutoRotate(); err != nil {
		return nil, fmt.Errorf("autorotate image: %w", err)
	}

	width, quality, err := s.paramsBySize(size)
	if err != nil {
		return nil, err
	}

	return s.generateThumbnail(src, width, quality, size, vips.KernelLanczos3)
}

func (s *ThumbnailService) GenerateThumbnailDynamic(imgPath string, size string) (*domain.Thumbnail, error) {
	if err := imageruntime.EnsureStarted(); err != nil {
		return nil, fmt.Errorf("start vips runtime: %w", err)
	}

	fileSize, err := getFileSize(imgPath)
	if err != nil {
		return nil, fmt.Errorf("get file size: %w", err)
	}

	src, err := vips.NewImageFromFile(imgPath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer src.Close()

	if err := src.AutoRotate(); err != nil {
		return nil, fmt.Errorf("autorotate image: %w", err)
	}

	width, quality, err := s.paramsBySize(size)
	if err != nil {
		return nil, err
	}

	kernel := resizeKernelForProfile(selectThumbnailResizeProfile(fileSize))

	workingImg := src
	if shouldPreScaleThumbnail(fileSize) {
		maxPreScaleWidth := maxDynamicPreScaleWidth(width, s.LargeWidth)
		workingImg, err = preScaleForLargeImage(src, maxPreScaleWidth)
		if err != nil {
			return nil, err
		}
		defer workingImg.Close()
	}

	if size == "small" {
		return s.generateSmallWithMinSize(workingImg, width, quality, kernel)
	}

	if size == "large" {
		return s.generateLargeStandalone(workingImg, width, quality, kernel)
	}

	return s.generateThumbnail(workingImg, width, quality, size, kernel)
}

func (s *ThumbnailService) generateSmallWithMinSize(src *vips.ImageRef, width, quality int, kernel vips.Kernel) (*domain.Thumbnail, error) {
	return runSmallThumbnailPolicy(src.Width(), width, quality, func(currentWidth, currentQuality int) (*domain.Thumbnail, error) {
		return s.generateThumbnail(src, currentWidth, currentQuality, "small", kernel)
	})
}

func (s *ThumbnailService) generateLargeWithMaxSize(src *vips.ImageRef, width, quality int, smallSize int, kernel vips.Kernel) (*domain.Thumbnail, error) {
	return runLargeThumbnailPolicy(src.Width(), width, quality, minLargeThumbnailSize(smallSize), func(currentWidth, currentQuality int) (*domain.Thumbnail, error) {
		return s.generateThumbnail(src, currentWidth, currentQuality, "large", kernel)
	})
}

func (s *ThumbnailService) generateLargeStandalone(src *vips.ImageRef, width, quality int, kernel vips.Kernel) (*domain.Thumbnail, error) {
	return runLargeThumbnailPolicy(src.Width(), width, quality, minLargeSize, func(currentWidth, currentQuality int) (*domain.Thumbnail, error) {
		return s.generateThumbnail(src, currentWidth, currentQuality, "large", kernel)
	})
}

func (s *ThumbnailService) generateThumbnail(src *vips.ImageRef, width, quality int, size string, kernel vips.Kernel) (*domain.Thumbnail, error) {
	working, err := src.Copy()
	if err != nil {
		return nil, fmt.Errorf("copy image: %w", err)
	}
	defer working.Close()

	srcWidth := working.Width()
	if srcWidth <= 0 {
		return nil, fmt.Errorf("invalid source width")
	}

	scale := float64(width) / float64(srcWidth)
	if scale <= 0 {
		return nil, fmt.Errorf("invalid resize scale")
	}

	if err := working.Resize(scale, kernel); err != nil {
		return nil, fmt.Errorf("resize thumbnail: %w", err)
	}

	encoded, err := exportThumbnailJPEG(working, quality)
	if err != nil {
		return nil, fmt.Errorf("encode thumbnail: %w", err)
	}

	return &domain.Thumbnail{
		Data:   encoded,
		Width:  working.Width(),
		Height: working.Height(),
		Size:   size,
	}, nil
}

func (s *ThumbnailService) GenerateBoth(imgPath string) (small, large *domain.Thumbnail, err error) {
	startedAt := time.Now()
	logger.Infof("thumbnail generate-both started: path=%s", imgPath)

	if err := imageruntime.EnsureStarted(); err != nil {
		logger.Errorf("thumbnail runtime start failed: path=%s error=%v", imgPath, err)
		return nil, nil, fmt.Errorf("start vips runtime: %w", err)
	}

	fileSize, err := getFileSize(imgPath)
	if err != nil {
		logger.Errorf("thumbnail stat failed: path=%s error=%v", imgPath, err)
		return nil, nil, fmt.Errorf("get file size: %w", err)
	}

	src, err := vips.NewImageFromFile(imgPath)
	if err != nil {
		logger.Errorf("thumbnail open failed: path=%s error=%v", imgPath, err)
		return nil, nil, fmt.Errorf("open image: %w", err)
	}
	defer src.Close()

	if err := src.AutoRotate(); err != nil {
		logger.Errorf("thumbnail autorotate failed: path=%s error=%v", imgPath, err)
		return nil, nil, fmt.Errorf("autorotate image: %w", err)
	}
	logger.Infof("thumbnail source loaded: path=%s file_size=%d width=%d height=%d", imgPath, fileSize, src.Width(), src.Height())

	kernel := resizeKernelForProfile(selectThumbnailResizeProfile(fileSize))

	workingImg := src
	if shouldPreScaleThumbnail(fileSize) {
		maxPreScaleWidth := maxGenerateBothPreScaleWidth(s.LargeWidth)
		workingImg, err = preScaleForLargeImage(src, maxPreScaleWidth)
		if err != nil {
			logger.Errorf("thumbnail pre-scale failed: path=%s max_pre_scale_width=%d error=%v", imgPath, maxPreScaleWidth, err)
			return nil, nil, err
		}
		defer workingImg.Close()
		logger.Infof("thumbnail pre-scale applied: path=%s max_pre_scale_width=%d working_width=%d working_height=%d", imgPath, maxPreScaleWidth, workingImg.Width(), workingImg.Height())
	}

	smallWidth, smallQuality, _ := s.paramsBySize("small")
	small, err = s.generateSmallWithMinSize(workingImg, smallWidth, smallQuality, kernel)
	if err != nil {
		logger.Errorf("thumbnail small generation failed: path=%s width=%d quality=%d error=%v", imgPath, smallWidth, smallQuality, err)
		return nil, nil, err
	}
	logger.Infof("thumbnail small generated: path=%s bytes=%d width=%d height=%d", imgPath, len(small.Data), small.Width, small.Height)

	largeWidth, largeQuality, _ := s.paramsBySize("large")
	large, err = s.generateLargeWithMaxSize(workingImg, largeWidth, largeQuality, len(small.Data), kernel)
	if err != nil {
		logger.Errorf("thumbnail large generation failed: path=%s width=%d quality=%d small_bytes=%d error=%v", imgPath, largeWidth, largeQuality, len(small.Data), err)
		return nil, nil, err
	}
	logger.Infof(
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

func getFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("stat file: %w", err)
	}
	return info.Size(), nil
}

func resizeKernelForProfile(profile thumbnailResizeProfile) vips.Kernel {
	switch profile {
	case thumbnailResizeHighQuality:
		return vips.KernelLanczos3
	case thumbnailResizeBalanced:
		return vips.KernelLinear
	default:
		return vips.KernelNearest
	}
}

func preScaleForLargeImage(img *vips.ImageRef, maxPreScaleWidth int) (*vips.ImageRef, error) {
	copyImg, err := img.Copy()
	if err != nil {
		return nil, fmt.Errorf("copy image: %w", err)
	}

	srcWidth := copyImg.Width()
	if srcWidth <= maxPreScaleWidth {
		return copyImg, nil
	}

	scale := float64(maxPreScaleWidth) / float64(srcWidth)
	if err := copyImg.Resize(scale, vips.KernelNearest); err != nil {
		copyImg.Close()
		return nil, fmt.Errorf("pre-scale image: %w", err)
	}

	return copyImg, nil
}

func exportThumbnailJPEG(img *vips.ImageRef, quality int) ([]byte, error) {
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
