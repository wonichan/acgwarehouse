package service

import (
	"bytes"
	"fmt"

	"github.com/disintegration/imaging"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
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

func (s *ThumbnailService) GenerateBoth(imgPath string) (small, large *domain.Thumbnail, err error) {
	small, err = s.GenerateThumbnail(imgPath, "small")
	if err != nil {
		return nil, nil, err
	}

	large, err = s.GenerateThumbnail(imgPath, "large")
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
