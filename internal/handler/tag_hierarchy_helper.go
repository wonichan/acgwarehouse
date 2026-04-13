package handler

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

type manualTagCreateInput struct {
	PreferredLabel  string
	PrimaryCategory string
	Level           string
	ParentID        *int64
	ReviewState     string
}

func resolveOrCreateManualTag(ctx context.Context, tagRepo repository.TagRepository, aliasRepo repository.TagAliasRepository, input manualTagCreateInput) (*domain.Tag, bool, string, error) {
	label := strings.TrimSpace(input.PreferredLabel)
	if label == "" {
		return nil, false, "", service.ErrInvalidHierarchy
	}

	existing, err := tagRepo.FindByLabel(ctx, label)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, "", err
	}
	if existing != nil {
		return existing, true, existing.Level, nil
	}

	if aliasRepo != nil {
		alias, aliasErr := aliasRepo.FindByNormalizedLabel(ctx, label)
		if aliasErr == nil {
			tag, findErr := tagRepo.FindByID(ctx, alias.TagID)
			if findErr != nil {
				return nil, false, "", findErr
			}
			return tag, true, tag.Level, nil
		}
		if !errors.Is(aliasErr, sql.ErrNoRows) {
			return nil, false, "", aliasErr
		}
	}

	level := strings.TrimSpace(input.Level)
	if level == "" {
		return nil, false, "", service.ErrInvalidHierarchy
	}
	if err := validateManualTagHierarchy(ctx, tagRepo, level, input.ParentID); err != nil {
		return nil, false, "", err
	}

	tag := &domain.Tag{
		PreferredLabel:  label,
		PrimaryCategory: strings.TrimSpace(input.PrimaryCategory),
		Slug:            makeSlug(label),
		Level:           level,
		ParentID:        input.ParentID,
		ReviewState:     input.ReviewState,
	}
	if tag.ReviewState == "" {
		tag.ReviewState = "confirmed"
	}
	if err := tagRepo.Save(ctx, tag); err != nil {
		return nil, false, "", err
	}
	return tag, false, tag.Level, nil
}

func validateManualTagHierarchy(ctx context.Context, tagRepo repository.TagRepository, level string, parentID *int64) error {
	switch level {
	case domain.TagLevelRoot:
		if parentID != nil {
			return service.ErrInvalidHierarchy
		}
		return nil
	case domain.TagLevelParent:
		if parentID == nil {
			return service.ErrInvalidHierarchy
		}
		parent, err := tagRepo.FindByID(ctx, *parentID)
		if err != nil {
			return service.ErrInvalidHierarchy
		}
		if parent.Level != domain.TagLevelRoot {
			return service.ErrInvalidHierarchy
		}
		return nil
	case domain.TagLevelChild:
		if parentID == nil {
			return nil
		}
		parent, err := tagRepo.FindByID(ctx, *parentID)
		if err != nil {
			return service.ErrInvalidHierarchy
		}
		if parent.Level != domain.TagLevelParent {
			return service.ErrInvalidHierarchy
		}
		return nil
	default:
		return service.ErrInvalidHierarchy
	}
}
