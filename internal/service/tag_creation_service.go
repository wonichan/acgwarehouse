package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type TagCreationRequest struct {
	PreferredLabel  string
	PrimaryCategory string
	Level           string
	ParentID        *int64
	ReviewState     string
}

type TagCreationResult struct {
	Tag         *domain.Tag
	Reused      bool
	ActualLevel string
}

type TagCreationService struct {
	tagRepo   repository.TagRepository
	aliasRepo repository.TagAliasRepository
}

func NewTagCreationService(tagRepo repository.TagRepository, aliasRepo repository.TagAliasRepository) *TagCreationService {
	return &TagCreationService{tagRepo: tagRepo, aliasRepo: aliasRepo}
}

func (s *TagCreationService) ResolveOrCreateManualTag(ctx context.Context, req TagCreationRequest) (*TagCreationResult, error) {
	label := strings.TrimSpace(req.PreferredLabel)
	if label == "" {
		return nil, ErrInvalidHierarchy
	}

	existing, err := s.tagRepo.FindByLabel(ctx, label)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existing != nil {
		return &TagCreationResult{Tag: existing, Reused: true, ActualLevel: existing.Level}, nil
	}

	if s.aliasRepo != nil {
		alias, aliasErr := s.aliasRepo.FindByNormalizedLabel(ctx, label)
		if aliasErr == nil {
			tag, findErr := s.tagRepo.FindByID(ctx, alias.TagID)
			if findErr != nil {
				return nil, findErr
			}
			return &TagCreationResult{Tag: tag, Reused: true, ActualLevel: tag.Level}, nil
		}
		if !errors.Is(aliasErr, sql.ErrNoRows) {
			return nil, aliasErr
		}
	}

	level := strings.TrimSpace(req.Level)
	if level == "" {
		return nil, ErrInvalidHierarchy
	}
	if err := s.validateManualTagHierarchy(ctx, level, req.ParentID); err != nil {
		return nil, err
	}

	tag := &domain.Tag{
		PreferredLabel:  label,
		PrimaryCategory: strings.TrimSpace(req.PrimaryCategory),
		Slug:            slugify(label),
		Level:           level,
		ParentID:        req.ParentID,
		ReviewState:     req.ReviewState,
	}
	if tag.ReviewState == "" {
		tag.ReviewState = domain.ReviewStateConfirmed
	}
	if err := s.tagRepo.Save(ctx, tag); err != nil {
		return nil, err
	}

	return &TagCreationResult{Tag: tag, Reused: false, ActualLevel: tag.Level}, nil
}

func (s *TagCreationService) validateManualTagHierarchy(ctx context.Context, level string, parentID *int64) error {
	switch level {
	case domain.TagLevelRoot:
		if parentID != nil {
			return ErrInvalidHierarchy
		}
		return nil
	case domain.TagLevelParent:
		if parentID == nil {
			return ErrInvalidHierarchy
		}
		parent, err := s.tagRepo.FindByID(ctx, *parentID)
		if err != nil {
			return ErrInvalidHierarchy
		}
		if parent.Level != domain.TagLevelRoot {
			return ErrInvalidHierarchy
		}
		return nil
	case domain.TagLevelChild:
		if parentID == nil {
			return nil
		}
		parent, err := s.tagRepo.FindByID(ctx, *parentID)
		if err != nil {
			return ErrInvalidHierarchy
		}
		if parent.Level != domain.TagLevelParent {
			return ErrInvalidHierarchy
		}
		return nil
	default:
		return ErrInvalidHierarchy
	}
}
