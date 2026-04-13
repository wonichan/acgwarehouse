package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
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
	logger.Infof("[service] BatchAddTags started: image_count=%d tag_count=%d", len(imageIDs), len(tagIDs))
	if len(imageIDs) == 0 || len(tagIDs) == 0 {
		return 0, errors.New("image_ids and tag_ids must not be empty")
	}

	validTags := make(map[int64]struct{}, len(tagIDs))
	for _, tagID := range tagIDs {
		if _, checked := validTags[tagID]; checked {
			continue
		}
		_, err := s.tagRepo.FindByID(ctx, tagID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			logger.Errorf("[service] BatchAddTags failed: tagRepo.FindByID err=%v", err)
			return 0, err
		}
		validTags[tagID] = struct{}{}
	}
	if len(validTags) == 0 {
		logger.Infof("[service] BatchAddTags completed: success_count=%d", 0)
		return 0, nil
	}

	successCount := 0
	for _, imageID := range imageIDs {
		// Verify image exists
		_, err := s.imageRepo.FindByID(imageID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue // Skip non-existent images
			}
			logger.Errorf("[service] BatchAddTags failed: imageRepo.FindByID err=%v", err)
			return successCount, err
		}

		for _, tagID := range tagIDs {
			if _, ok := validTags[tagID]; !ok {
				continue
			}

			// Add tag to image (using pending state)
			imageTag := &domain.ImageTag{
				ImageID:     imageID,
				TagID:       tagID,
				ReviewState: "pending",
			}
			if err := s.imageTagRepo.Save(ctx, imageTag); err != nil {
				logger.Errorf("[service] BatchAddTags failed: imageTagRepo.Save err=%v", err)
				return successCount, err
			}
		}
		successCount++
	}

	logger.Infof("[service] BatchAddTags completed: success_count=%d", successCount)
	return successCount, nil
}

// BatchRemoveTags removes tags from multiple images
func (s *BatchService) BatchRemoveTags(ctx context.Context, imageIDs, tagIDs []int64) (int, error) {
	logger.Infof("[service] BatchRemoveTags started: image_count=%d tag_count=%d", len(imageIDs), len(tagIDs))
	if len(imageIDs) == 0 || len(tagIDs) == 0 {
		return 0, errors.New("image_ids and tag_ids must not be empty")
	}

	successCount := 0
	for _, imageID := range imageIDs {
		anyRemoved := false
		for _, tagID := range tagIDs {
			rowsAffected, err := s.imageTagRepo.Delete(ctx, imageID, tagID)
			if err != nil {
				logger.Errorf("[service] BatchRemoveTags failed: imageTagRepo.Delete err=%v", err)
				continue
			}
			if rowsAffected > 0 {
				anyRemoved = true
			}
		}
		if anyRemoved {
			successCount++
		}
	}

	logger.Infof("[service] BatchRemoveTags completed: removed_count=%d", successCount)
	return successCount, nil
}

// BatchMoveToCollection moves images to a collection
func (s *BatchService) BatchMoveToCollection(ctx context.Context, imageIDs []int64, collectionID int64) (int, error) {
	logger.Infof("[service] BatchMoveToCollection started: image_count=%d collection_id=%d", len(imageIDs), collectionID)
	if len(imageIDs) == 0 {
		return 0, errors.New("image_ids must not be empty")
	}

	// Verify collection exists
	_, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		logger.Errorf("[service] BatchMoveToCollection failed: collectionRepo.FindByID err=%v", err)
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
			logger.Errorf("[service] BatchMoveToCollection failed: imageRepo.FindByID err=%v", err)
			return successCount, err
		}

		if err := s.collectionRepo.AddImage(ctx, collectionID, imageID); err != nil {
			logger.Errorf("[service] BatchMoveToCollection failed: collectionRepo.AddImage err=%v", err)
			return successCount, err
		}
		successCount++
	}

	// Auto-update cover after batch move
	if successCount > 0 {
		err = s.collectionRepo.UpdateCover(ctx, collectionID, imageIDs[len(imageIDs)-1])
		if err != nil {
			logger.Errorf("[service] BatchMoveToCollection failed: collectionRepo.UpdateCover err=%v", err)
		}
	}

	logger.Infof("[service] BatchMoveToCollection completed: moved_count=%d", successCount)
	return successCount, nil
}

// BatchRemoveFromCollection removes images from a collection
func (s *BatchService) BatchRemoveFromCollection(ctx context.Context, imageIDs []int64, collectionID int64) (int, error) {
	logger.Infof("[service] BatchRemoveFromCollection started: image_count=%d collection_id=%d", len(imageIDs), collectionID)
	if len(imageIDs) == 0 {
		return 0, errors.New("image_ids must not be empty")
	}

	// Verify collection exists
	collection, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		logger.Errorf("[service] BatchRemoveFromCollection failed: collectionRepo.FindByID err=%v", err)
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
			logger.Errorf("[service] BatchRemoveFromCollection failed: collectionRepo.RemoveImage err=%v", err)
			continue // Continue with other removals
		}
		successCount++
	}

	// Auto-update cover if we removed the cover image
	if coverNeedsUpdate && successCount > 0 {
		latestImageID, err := s.collectionRepo.GetLatestImageID(ctx, collectionID)
		if err != nil {
			logger.Errorf("[service] BatchRemoveFromCollection failed: collectionRepo.GetLatestImageID err=%v", err)
		}
		if latestImageID != nil {
			err = s.collectionRepo.UpdateCover(ctx, collectionID, *latestImageID)
			if err != nil {
				logger.Errorf("[service] BatchRemoveFromCollection failed: collectionRepo.UpdateCover err=%v", err)
			}
		}
	}

	logger.Infof("[service] BatchRemoveFromCollection completed: removed_count=%d", successCount)
	return successCount, nil
}

// BatchDeleteImages deletes multiple images
func (s *BatchService) BatchDeleteImages(ctx context.Context, imageIDs []int64) (int, error) {
	logger.Infof("[service] BatchDeleteImages started: image_count=%d", len(imageIDs))
	if len(imageIDs) == 0 {
		return 0, errors.New("image_ids must not be empty")
	}

	successCount := 0
	for _, imageID := range imageIDs {
		// Delete image (cascade will remove related image_tags)
		if err := s.imageRepo.Delete(imageID); err != nil {
			logger.Errorf("[service] BatchDeleteImages failed: imageRepo.Delete err=%v", err)
			continue
		}
		successCount++
	}

	logger.Infof("[service] BatchDeleteImages completed: deleted_count=%d", successCount)
	return successCount, nil
}
