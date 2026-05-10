package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wonichan/acgwarehouse-backend/internal/logger"
)

func (s *TagAdminService) MergeTags(ctx context.Context, sourceTagID, targetTagID int64) (*TagMergeResult, error) {
	logger.Infof("[service] TagAdmin MergeTags started: source_id=%d target_id=%d", sourceTagID, targetTagID)
	if sourceTagID <= 0 || targetTagID <= 0 {
		return nil, ErrTagNotFound
	}
	if sourceTagID == targetTagID {
		return nil, ErrMergeSameSourceTarget
	}
	if s.adminStore == nil {
		return nil, errors.New("tag admin store is required")
	}

	sourceTag, err := s.tagRepo.FindByID(ctx, sourceTagID)
	if err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	targetTag, err := s.tagRepo.FindByID(ctx, targetTagID)
	if err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	if sourceTag.Level != targetTag.Level {
		return nil, ErrCrossLevelMerge
	}
	children, err := s.tagRepo.FindChildrenByParent(ctx, sourceTagID)
	if err != nil {
		return nil, err
	}
	if len(children) > 0 {
		return nil, ErrMergeSourceHasChildren
	}

	mergeResult, err := s.adminStore.MergeTags(ctx, sourceTagID, targetTagID)
	if err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	if err := s.imageTagRepo.SyncFTSForTag(ctx, targetTagID); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	logger.Infof("[service] TagAdmin MergeTags completed: source_id=%d target_id=%d migrated_images=%d migrated_aliases=%d", sourceTagID, targetTagID, mergeResult.MigratedImageAssociations, mergeResult.MigratedAliases)
	return &TagMergeResult{
		SourceTagID:               mergeResult.SourceTagID,
		TargetTagID:               mergeResult.TargetTagID,
		MigratedImageAssociations: mergeResult.MigratedImageAssociations,
		MigratedAliases:           mergeResult.MigratedAliases,
	}, nil
}
