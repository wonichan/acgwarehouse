package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
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
	log.Printf("[service] MergeTags started: image_id=%d tag_count=%d observation_id=%d", imageID, len(tags), observationID)
	if observationID > 0 {
		if _, err := s.obsRepo.FindByID(ctx, observationID); err != nil {
			log.Printf("[service] MergeTags obsRepo.FindByID error: %v", err)
			return err
		}
		if err := s.RemoveRejectedAITags(ctx, imageID); err != nil {
			log.Printf("[service] MergeTags RemoveRejectedAITags error: %v", err)
			return err
		}
	}

	existingImageTags, err := s.imageTagRepo.FindByImageID(ctx, imageID)
	if err != nil {
		log.Printf("[service] MergeTags imageTagRepo.FindByImageID error: %v", err)
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
				log.Printf("[service] MergeTags tagRepo.FindByLabel error: %v", err)
				return err
			}

			if s.aliasRepo != nil {
				alias, aliasErr := s.aliasRepo.FindByNormalizedLabel(ctx, normalized)
				if aliasErr == nil {
					tag, err = s.tagRepo.FindByID(ctx, alias.TagID)
					if err != nil {
						log.Printf("[service] MergeTags tagRepo.FindByID error: %v", err)
						return err
					}
				} else if !errors.Is(aliasErr, sql.ErrNoRows) {
					log.Printf("[service] MergeTags aliasRepo.FindByNormalizedLabel error: %v", aliasErr)
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
			}
			if err := s.tagRepo.Save(ctx, tag); err != nil {
				log.Printf("[service] MergeTags tagRepo.Save error: %v", err)
				return err
			}
		} else {
			if existing := existingByTagID[tag.ID]; existing != nil {
				// Already associated — skip; trigger maintains usage_count on first insert
				continue
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
			log.Printf("[service] MergeTags imageTagRepo.Save error: %v", err)
			return err
		}
		existingByTagID[tag.ID] = &domain.ImageTag{ImageID: imageID, TagID: tag.ID, Source: source, SourceObservationID: sourceObservationID, Confidence: confidence, ReviewState: "pending"}
	}

	log.Printf("[service] MergeTags completed: image_id=%d tags_merged=%d", imageID, len(tags))
	return nil
}

func (s *TagGovernanceService) RemovePendingAITags(ctx context.Context, imageID int64) error {
	log.Printf("[service] RemovePendingAITags started: image_id=%d", imageID)
	return s.removeAITagsByState(ctx, imageID, "pending")
}

func (s *TagGovernanceService) RemoveRejectedAITags(ctx context.Context, imageID int64) error {
	log.Printf("[service] RemoveRejectedAITags started: image_id=%d", imageID)
	return s.removeAITagsByState(ctx, imageID, "rejected")
}

func (s *TagGovernanceService) removeAITagsByState(ctx context.Context, imageID int64, state string) error {
	items, err := s.imageTagRepo.FindByImageID(ctx, imageID)
	if err != nil {
		log.Printf("[service] removeAITagsByState imageTagRepo.FindByImageID error: %v", err)
		return err
	}
	for _, item := range items {
		if item.Source != domain.ImageTagSourceAI || item.ReviewState != state {
			continue
		}
		_, err := s.imageTagRepo.Delete(ctx, imageID, item.TagID)
		if err != nil {
			log.Printf("[service] removeAITagsByState imageTagRepo.Delete error: %v", err)
			return err
		}
	}
	return nil
}

func slugify(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	text = strings.ReplaceAll(text, " ", "-")
	return text
}
