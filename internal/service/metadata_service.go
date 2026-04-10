package service

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/evanoberholster/imagemeta"
	_ "golang.org/x/image/webp"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

var SupportedFormats = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

type MetadataService struct{}

func NewMetadataService() *MetadataService {
	return &MetadataService{}
}

func (s *MetadataService) IsImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return SupportedFormats[ext]
}

func (s *MetadataService) ExtractMetadata(path string) (*domain.Image, error) {
	log.Printf("[service] ExtractMetadata started: path=%s", path)
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("[service] ExtractMetadata failed: path=%s error=%v", path, err)
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		log.Printf("[service] ExtractMetadata failed: path=%s error=%v", path, err)
		return nil, err
	}
	defer file.Close()

	_, _ = imagemeta.Decode(file)
	if _, err := file.Seek(0, 0); err != nil {
		log.Printf("[service] ExtractMetadata failed: path=%s error=%v", path, err)
		return nil, fmt.Errorf("reset image reader: %w", err)
	}

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		log.Printf("[service] ExtractMetadata failed: path=%s error=%v", path, err)
		return nil, fmt.Errorf("decode image config: %w", err)
	}

	modTime := info.ModTime()
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")

	log.Printf("[service] ExtractMetadata completed: path=%s width=%d height=%d format=%s size=%d", path, config.Width, config.Height, ext, info.Size())
	return &domain.Image{
		Path:      path,
		Filename:  filepath.Base(path),
		FileSize:  info.Size(),
		Width:     config.Width,
		Height:    config.Height,
		Format:    ext,
		CreatedAt: modTime,
		UpdatedAt: time.Now(),
	}, nil
}
