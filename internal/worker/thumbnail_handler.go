package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type thumbnailGenerator interface {
	GenerateBoth(imgPath string) (small, large *domain.Thumbnail, err error)
}

type thumbnailUploader interface {
	Upload(ctx context.Context, filename, size string, data []byte) (string, error)
}

type thumbnailImageRepository interface {
	UpdateThumbnails(id int64, smallURL, largeURL string) error
}

type ThumbnailHandler struct {
	thumbnailSvc thumbnailGenerator
	cosSvc       thumbnailUploader
	imageRepo    thumbnailImageRepository
}

type thumbnailJobPayload struct {
	ImageID  int64  `json:"image_id"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

func NewThumbnailHandler(thumbnailSvc thumbnailGenerator, cosSvc thumbnailUploader, imageRepo thumbnailImageRepository) *ThumbnailHandler {
	return &ThumbnailHandler{thumbnailSvc: thumbnailSvc, cosSvc: cosSvc, imageRepo: imageRepo}
}

func (h *ThumbnailHandler) Handle(ctx context.Context, jobID int64, payload string) error {
	_ = jobID

	if h.thumbnailSvc == nil || h.cosSvc == nil || h.imageRepo == nil {
		return fmt.Errorf("thumbnail handler dependencies are not initialized")
	}

	var p thumbnailJobPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return fmt.Errorf("parse thumbnail job payload: %w", err)
	}
	if p.ImageID <= 0 || p.Path == "" {
		return fmt.Errorf("invalid thumbnail job payload")
	}
	uploadName := fmt.Sprintf("%d-%s", p.ImageID, p.Filename)

	small, large, err := h.thumbnailSvc.GenerateBoth(p.Path)
	if err != nil {
		return fmt.Errorf("generate thumbnails: %w", err)
	}

	smallURL, err := h.cosSvc.Upload(ctx, uploadName, "small", small.Data)
	if err != nil {
		return fmt.Errorf("upload small thumbnail: %w", err)
	}
	largeURL, err := h.cosSvc.Upload(ctx, uploadName, "large", large.Data)
	if err != nil {
		return fmt.Errorf("upload large thumbnail: %w", err)
	}

	if err := h.imageRepo.UpdateThumbnails(p.ImageID, smallURL, largeURL); err != nil {
		return fmt.Errorf("update thumbnail urls: %w", err)
	}

	return nil
}
