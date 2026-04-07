package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type TagGovernanceService struct {
	tagRepo      repository.TagRepository
	aliasRepo    repository.TagAliasRepository
	obsRepo      repository.TagObservationRepository
	imageTagRepo repository.ImageTagRepository
}

func NewTagGovernanceService(
	tagRepo repository.TagRepository,
	aliasRepo repository.TagAliasRepository,
	obsRepo repository.TagObservationRepository,
	imageTagRepo repository.ImageTagRepository,
) *TagGovernanceService {
	return &TagGovernanceService{
		tagRepo:      tagRepo,
		aliasRepo:    aliasRepo,
		obsRepo:      obsRepo,
		imageTagRepo: imageTagRepo,
	}
}

func (s *TagGovernanceService) MergeTags(ctx context.Context, imageID int64, tags []string, observationID int64, confidence float64) error {
	if observationID > 0 {
		if _, err := s.obsRepo.FindByID(ctx, observationID); err != nil {
			return err
		}
		if err := s.RemoveRejectedAITags(ctx, imageID); err != nil {
			return err
		}
	}

	existingImageTags, err := s.imageTagRepo.FindByImageID(ctx, imageID)
	if err != nil {
		return err
	}
	existingByTagID := make(map[int64]*domain.ImageTag, len(existingImageTags))
	for _, item := range existingImageTags {
		existingByTagID[item.TagID] = item
	}

	for _, rawTag := range tags {
		normalized := strings.TrimSpace(rawTag)
		if normalized == "" {
			continue
		}

		tag, err := s.tagRepo.FindByLabel(ctx, normalized)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			if s.aliasRepo != nil {
				alias, aliasErr := s.aliasRepo.FindByNormalizedLabel(ctx, normalized)
				if aliasErr == nil {
					tag, err = s.tagRepo.FindByID(ctx, alias.TagID)
					if err != nil {
						return err
					}
				} else if !errors.Is(aliasErr, sql.ErrNoRows) {
					return aliasErr
				}
			}
		}

		if tag == nil {
			tag = &domain.Tag{
				PreferredLabel: normalized,
				Slug:           slugify(normalized),
				ReviewState:    "pending",
				TrustScore:     confidence,
				UsageCount:     1,
			}
			if err := s.tagRepo.Save(ctx, tag); err != nil {
				return err
			}
		} else {
			if existing := existingByTagID[tag.ID]; existing != nil && existing.ReviewState == "confirmed" {
				continue
			}
			// Check if image-tag association already exists before incrementing count
			// This prevents double-counting when retrying failed AI tag generation tasks
			_, exists := existingByTagID[tag.ID]
			if !exists {
				if err := s.tagRepo.IncrementUsageCount(ctx, tag.ID); err != nil {
					return err
				}
			}
		}

		var sourceObservationID *int64
		source := domain.ImageTagSourceManual
		if observationID > 0 {
			sourceObservationID = &observationID
			source = domain.ImageTagSourceAI
		}

		if err := s.imageTagRepo.Save(ctx, &domain.ImageTag{
			ImageID:             imageID,
			TagID:               tag.ID,
			Source:              source,
			SourceObservationID: sourceObservationID,
			Confidence:          confidence,
			ReviewState:         "pending",
		}); err != nil {
			return err
		}
		existingByTagID[tag.ID] = &domain.ImageTag{ImageID: imageID, TagID: tag.ID, Source: source, SourceObservationID: sourceObservationID, Confidence: confidence, ReviewState: "pending"}
	}

	return nil
}

func (s *TagGovernanceService) RemovePendingAITags(ctx context.Context, imageID int64) error {
	return s.removeAITagsByState(ctx, imageID, "pending")
}

func (s *TagGovernanceService) RemoveRejectedAITags(ctx context.Context, imageID int64) error {
	return s.removeAITagsByState(ctx, imageID, "rejected")
}

func (s *TagGovernanceService) removeAITagsByState(ctx context.Context, imageID int64, state string) error {
	items, err := s.imageTagRepo.FindByImageID(ctx, imageID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Source != domain.ImageTagSourceAI || item.ReviewState != state {
			continue
		}
		rowsAffected, err := s.imageTagRepo.Delete(ctx, imageID, item.TagID)
		if err != nil {
			return err
		}
		if rowsAffected > 0 {
			if err := s.tagRepo.DecrementUsageCount(ctx, item.TagID); err != nil {
				return err
			}
		}
	}
	return nil
}

func slugify(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	text = strings.ReplaceAll(text, " ", "-")
	return text
}
