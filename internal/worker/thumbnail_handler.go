package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type thumbnailGenerator interface {
	GenerateBoth(imgPath string) (small, large *domain.Thumbnail, err error)
}

type ThumbnailUploader interface {
	Upload(ctx context.Context, filename, size string, data []byte) (string, error)
	DeleteByURL(ctx context.Context, objectURL string) error
}

type thumbnailImageRepository interface {
	UpdateThumbnails(id int64, smallURL, largeURL string) error
}

type ThumbnailHandler struct {
	thumbnailSvc thumbnailGenerator
	uploader     ThumbnailUploader
	imageRepo    thumbnailImageRepository
}

type thumbnailJobPayload struct {
	ImageID  int64  `json:"image_id"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

func NewThumbnailHandler(thumbnailSvc thumbnailGenerator, uploader ThumbnailUploader, imageRepo thumbnailImageRepository) *ThumbnailHandler {
	return &ThumbnailHandler{thumbnailSvc: thumbnailSvc, uploader: uploader, imageRepo: imageRepo}
}

func (h *ThumbnailHandler) Handle(ctx context.Context, jobID int64, payload string) error {
	if h.thumbnailSvc == nil || h.uploader == nil || h.imageRepo == nil {
		return fmt.Errorf("thumbnail handler dependencies are not initialized")
	}

	var p thumbnailJobPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return fmt.Errorf("parse thumbnail job payload: %w", err)
	}
	if p.ImageID <= 0 || p.Path == "" {
		return fmt.Errorf("invalid thumbnail job payload")
	}

	startedAt := time.Now()
	log.Printf("thumbnail task started: job_id=%d image_id=%d filename=%s path=%s", jobID, p.ImageID, p.Filename, p.Path)

	uploadName := fmt.Sprintf("%d-%s", p.ImageID, p.Filename)

	small, large, err := h.thumbnailSvc.GenerateBoth(p.Path)
	if err != nil {
		log.Printf("thumbnail generation failed: job_id=%d image_id=%d error=%v", jobID, p.ImageID, err)
		return fmt.Errorf("generate thumbnails: %w", err)
	}
	log.Printf(
		"thumbnail generation completed: job_id=%d image_id=%d small_bytes=%d small_width=%d small_height=%d large_bytes=%d large_width=%d large_height=%d",
		jobID,
		p.ImageID,
		len(small.Data),
		small.Width,
		small.Height,
		len(large.Data),
		large.Width,
		large.Height,
	)

	smallURL, err := h.uploader.Upload(ctx, uploadName, "small", small.Data)
	if err != nil {
		log.Printf("thumbnail upload failed: job_id=%d image_id=%d size=small error=%v", jobID, p.ImageID, err)
		return fmt.Errorf("upload small thumbnail: %w", err)
	}
	log.Printf("thumbnail upload completed: job_id=%d image_id=%d size=small url=%s", jobID, p.ImageID, smallURL)
	largeURL, err := h.uploader.Upload(ctx, uploadName, "large", large.Data)
	if err != nil {
		log.Printf("thumbnail upload failed: job_id=%d image_id=%d size=large error=%v", jobID, p.ImageID, err)
		if rollbackErr := h.uploader.DeleteByURL(ctx, smallURL); rollbackErr != nil {
			log.Printf("thumbnail rollback failed: job_id=%d image_id=%d size=small url=%s error=%v", jobID, p.ImageID, smallURL, rollbackErr)
			return fmt.Errorf("upload large thumbnail: %w", errors.Join(err, fmt.Errorf("rollback small thumbnail failed: %w", rollbackErr)))
		}
		log.Printf("thumbnail rollback completed: job_id=%d image_id=%d size=small url=%s", jobID, p.ImageID, smallURL)
		return fmt.Errorf("upload large thumbnail: %w", err)
	}
	log.Printf("thumbnail upload completed: job_id=%d image_id=%d size=large url=%s", jobID, p.ImageID, largeURL)

	if err := h.imageRepo.UpdateThumbnails(p.ImageID, smallURL, largeURL); err != nil {
		log.Printf("thumbnail db update failed: job_id=%d image_id=%d error=%v", jobID, p.ImageID, err)
		return fmt.Errorf("update thumbnail urls: %w", err)
	}
	log.Printf(
		"thumbnail task completed: job_id=%d image_id=%d duration=%s small_url=%s large_url=%s",
		jobID,
		p.ImageID,
		time.Since(startedAt),
		smallURL,
		largeURL,
	)

	return nil
}
