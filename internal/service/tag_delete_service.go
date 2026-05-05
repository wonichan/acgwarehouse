package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wonichan/acgwarehouse-backend/internal/logger"
)

func (s *TagAdminService) GetDeletePreview(ctx context.Context, tagID int64) (*TagDeletePreview, error) {
	if tagID <= 0 {
		return nil, ErrTagNotFound
	}

	tag, err := s.tagRepo.FindByID(ctx, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	children, err := s.tagRepo.FindChildrenByParent(ctx, tagID)
	if err != nil {
		return nil, err
	}
	directAssociationCount, err := s.countDirectAssociations(ctx, tagID)
	if err != nil {
		return nil, err
	}

	preview := &TagDeletePreview{
		TagID:              tag.ID,
		PreferredLabel:     tag.PreferredLabel,
		AffectedImageCount: directAssociationCount,
		ChildCount:         int64(len(children)),
		CanDelete:          true,
	}

	return preview, nil
}

func (s *TagAdminService) DeleteTag(ctx context.Context, tagID int64) (*TagDeleteResult, error) {
	if tagID <= 0 {
		return nil, ErrTagNotFound
	}
	result, err := s.adminStore.DeleteTag(ctx, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	return &TagDeleteResult{
		DeletedTagID:       result.DeletedTagID,
		AffectedImageCount: result.AffectedImageCount,
		DetachedChildCount: result.DetachedChildCount,
	}, nil
}

func (s *TagAdminService) CleanupUnusedTags(ctx context.Context, tagIDs []int64) (*TagCleanupResult, error) {
	logger.Infof("[service] TagAdmin CleanupUnusedTags started: tag_count=%d", len(tagIDs))
	result := &TagCleanupResult{
		Deleted: make([]TagCleanupEntry, 0),
		Blocked: make([]TagCleanupEntry, 0),
		Failed:  make([]TagCleanupEntry, 0),
	}

	for _, tagID := range tagIDs {
		preview, err := s.GetDeletePreview(ctx, tagID)
		if err != nil {
			logger.Errorf("[service] TagAdmin CleanupUnusedTags failed GetDeletePreview for tag_id=%d: %v", tagID, err)
			entry := TagCleanupEntry{TagID: tagID}
			if errors.Is(err, ErrTagNotFound) {
				entry.Error = "tag not found"
			} else {
				entry.Error = err.Error()
			}
			result.Failed = append(result.Failed, entry)
			continue
		}

		if preview.AffectedImageCount > 0 || preview.ChildCount > 0 {
			blockingReason := "merge_or_reclassify_required"
			if preview.ChildCount > 0 {
				blockingReason = "child_tags_exist"
			}
			result.Blocked = append(result.Blocked, TagCleanupEntry{
				TagID:              preview.TagID,
				PreferredLabel:     preview.PreferredLabel,
				AffectedImageCount: preview.AffectedImageCount,
				BlockingReason:     blockingReason,
			})
			continue
		}

		if err := s.deleteUnusedTag(ctx, preview.TagID); err != nil {
			logger.Errorf("[service] TagAdmin CleanupUnusedTags failed deleteUnusedTag for tag_id=%d: %v", preview.TagID, err)
			result.Failed = append(result.Failed, TagCleanupEntry{
				TagID:          preview.TagID,
				PreferredLabel: preview.PreferredLabel,
				Error:          err.Error(),
			})
			continue
		}

		result.Deleted = append(result.Deleted, TagCleanupEntry{
			TagID:          preview.TagID,
			PreferredLabel: preview.PreferredLabel,
		})
	}

	logger.Infof("[service] TagAdmin CleanupUnusedTags completed: deleted=%d blocked=%d failed=%d", len(result.Deleted), len(result.Blocked), len(result.Failed))
	return result, nil
}

func (s *TagAdminService) deleteUnusedTag(ctx context.Context, tagID int64) error {
	aliases, err := s.aliasRepo.FindByTagID(ctx, tagID)
	if err != nil {
		return err
	}
	for _, alias := range aliases {
		if err := s.aliasRepo.Delete(ctx, alias.ID); err != nil {
			return err
		}
	}
	if err := s.tagRepo.Delete(ctx, tagID); err != nil {
		return err
	}
	return nil
}

func (s *TagAdminService) countDirectAssociations(ctx context.Context, tagID int64) (int64, error) {
	return s.adminStore.CountDirectTagAssociations(ctx, tagID)
}
