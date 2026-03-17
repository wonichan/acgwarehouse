package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// BatchService provides batch operations for images
type BatchService struct {
	imageRepo      repository.ImageRepository
	tagRepo        repository.TagRepository
	imageTagRepo   repository.ImageTagRepository
	collectionRepo repository.CollectionRepository
}

// NewBatchService creates a new BatchService instance
func NewBatchService(
	imageRepo repository.ImageRepository,
	tagRepo repository.TagRepository,
	imageTagRepo repository.ImageTagRepository,
	collectionRepo repository.CollectionRepository,
) *BatchService {
	return &BatchService{
		imageRepo:      imageRepo,
		tagRepo:        tagRepo,
		imageTagRepo:   imageTagRepo,
		collectionRepo: collectionRepo,
	}
}

// BatchAddTags adds tags to multiple images
func (s *BatchService) BatchAddTags(ctx context.Context, imageIDs, tagIDs []int64) (int, error) {
	if len(imageIDs) == 0 || len(tagIDs) == 0 {
		return 0, errors.New("image_ids and tag_ids must not be empty")
	}

	successCount := 0
	for _, imageID := range imageIDs {
		// Verify image exists
		_, err := s.imageRepo.FindByID(imageID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue // Skip non-existent images
			}
			return successCount, err
		}

		for _, tagID := range tagIDs {
			// Verify tag exists
			_, err := s.tagRepo.FindByID(ctx, tagID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue // Skip non-existent tags
				}
				return successCount, err
			}

			// Add tag to image (using pending state)
			imageTag := &domain.ImageTag{
				ImageID:     imageID,
				TagID:       tagID,
				ReviewState: "pending",
			}
			if err := s.imageTagRepo.Save(ctx, imageTag); err != nil {
				return successCount, err
			}
		}
		successCount++
	}

	return successCount, nil
}

// BatchRemoveTags removes tags from multiple images
func (s *BatchService) BatchRemoveTags(ctx context.Context, imageIDs, tagIDs []int64) (int, error) {
	if len(imageIDs) == 0 || len(tagIDs) == 0 {
		return 0, errors.New("image_ids and tag_ids must not be empty")
	}

	successCount := 0
	for _, imageID := range imageIDs {
		for _, tagID := range tagIDs {
			if err := s.imageTagRepo.Delete(ctx, imageID, tagID); err != nil {
				// Continue with other deletions even if one fails
				continue
			}
		}
		successCount++
	}

	return successCount, nil
}

// BatchMoveToCollection moves images to a collection
func (s *BatchService) BatchMoveToCollection(ctx context.Context, imageIDs []int64, collectionID int64) (int, error) {
	if len(imageIDs) == 0 {
		return 0, errors.New("image_ids must not be empty")
	}

	// Verify collection exists
	_, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		return 0, errors.New("collection not found")
	}

	successCount := 0
	for _, imageID := range imageIDs {
		// Verify image exists
		_, err := s.imageRepo.FindByID(imageID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue // Skip non-existent images
			}
			return successCount, err
		}

		if err := s.collectionRepo.AddImage(ctx, collectionID, imageID); err != nil {
			return successCount, err
		}
		successCount++
	}

	// Auto-update cover after batch move
	if successCount > 0 {
		_ = s.collectionRepo.UpdateCover(ctx, collectionID, imageIDs[len(imageIDs)-1])
	}

	return successCount, nil
}

// BatchRemoveFromCollection removes images from a collection
func (s *BatchService) BatchRemoveFromCollection(ctx context.Context, imageIDs []int64, collectionID int64) (int, error) {
	if len(imageIDs) == 0 {
		return 0, errors.New("image_ids must not be empty")
	}

	// Verify collection exists
	collection, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		return 0, errors.New("collection not found")
	}

	successCount := 0
	coverNeedsUpdate := false

	for _, imageID := range imageIDs {
		// Check if this image is the cover
		if collection.CoverImageID != nil && *collection.CoverImageID == imageID {
			coverNeedsUpdate = true
		}

		if err := s.collectionRepo.RemoveImage(ctx, collectionID, imageID); err != nil {
			continue // Continue with other removals
		}
		successCount++
	}

	// Auto-update cover if we removed the cover image
	if coverNeedsUpdate && successCount > 0 {
		latestImageID, _ := s.collectionRepo.GetLatestImageID(ctx, collectionID)
		if latestImageID != nil {
			_ = s.collectionRepo.UpdateCover(ctx, collectionID, *latestImageID)
		}
	}

	return successCount, nil
}

// BatchDeleteImages deletes multiple images
func (s *BatchService) BatchDeleteImages(ctx context.Context, imageIDs []int64) (int, error) {
	if len(imageIDs) == 0 {
		return 0, errors.New("image_ids must not be empty")
	}

	successCount := 0
	for _, imageID := range imageIDs {
		// Delete image (cascade will handle related records)
		if err := s.imageRepo.Delete(imageID); err != nil {
			continue // Continue with other deletions
		}
		successCount++
	}

	return successCount, nil
}
