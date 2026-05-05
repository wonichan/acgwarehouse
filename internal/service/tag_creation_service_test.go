package service

import (
	"context"
	"errors"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestTagCreationServiceCreatesManualTagWithHierarchy(t *testing.T) {
	t.Parallel()

	tagRepo, aliasRepo := newTagCreationReposForTest(t)
	ctx := context.Background()

	root := &domain.Tag{PreferredLabel: "series", Slug: "series", Level: domain.TagLevelRoot, ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, root); err != nil {
		t.Fatalf("save root: %v", err)
	}

	svc := NewTagCreationService(tagRepo, aliasRepo)
	result, err := svc.ResolveOrCreateManualTag(ctx, TagCreationRequest{
		PreferredLabel: "main cast",
		Level:          domain.TagLevelParent,
		ParentID:       &root.ID,
	})
	if err != nil {
		t.Fatalf("ResolveOrCreateManualTag() error = %v", err)
	}
	if result.Reused {
		t.Fatal("ResolveOrCreateManualTag() reused existing tag, want new")
	}
	if result.Tag.ID == 0 || result.Tag.Slug != "main-cast" || result.Tag.ReviewState != "confirmed" {
		t.Fatalf("created tag = %+v", result.Tag)
	}
	if result.Tag.ParentID == nil || *result.Tag.ParentID != root.ID {
		t.Fatalf("created parent_id = %v, want %d", result.Tag.ParentID, root.ID)
	}
}

func TestTagCreationServiceReusesExistingLabel(t *testing.T) {
	t.Parallel()

	tagRepo, _ := newTagCreationReposForTest(t)
	ctx := context.Background()
	existing := &domain.Tag{PreferredLabel: "night rain", Slug: "night-rain", Level: domain.TagLevelChild, ReviewState: "pending"}
	if err := tagRepo.Save(ctx, existing); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	result, err := NewTagCreationService(tagRepo, nil).ResolveOrCreateManualTag(ctx, TagCreationRequest{
		PreferredLabel: "night rain",
		Level:          domain.TagLevelRoot,
	})
	if err != nil {
		t.Fatalf("ResolveOrCreateManualTag() error = %v", err)
	}
	if !result.Reused || result.Tag.ID != existing.ID || result.ActualLevel != domain.TagLevelChild {
		t.Fatalf("reuse result = %+v, existing ID %d", result, existing.ID)
	}
}

func TestTagCreationServiceReusesAliasTarget(t *testing.T) {
	t.Parallel()

	tagRepo, aliasRepo := newTagCreationReposForTest(t)
	ctx := context.Background()

	target := &domain.Tag{PreferredLabel: "sunset", Slug: "sunset", Level: domain.TagLevelChild, ReviewState: "confirmed"}
	if err := tagRepo.Save(ctx, target); err != nil {
		t.Fatalf("save target: %v", err)
	}
	if err := aliasRepo.Save(ctx, &domain.TagAlias{TagID: target.ID, Label: "golden hour"}); err != nil {
		t.Fatalf("save alias: %v", err)
	}

	result, err := NewTagCreationService(tagRepo, aliasRepo).ResolveOrCreateManualTag(ctx, TagCreationRequest{
		PreferredLabel: " golden hour ",
		Level:          domain.TagLevelRoot,
	})
	if err != nil {
		t.Fatalf("ResolveOrCreateManualTag() error = %v", err)
	}
	if !result.Reused || result.Tag.ID != target.ID || result.ActualLevel != target.Level {
		t.Fatalf("reuse alias result = %+v, target ID %d", result, target.ID)
	}
}

func TestTagCreationServiceRejectsInvalidHierarchy(t *testing.T) {
	t.Parallel()

	tagRepo, _ := newTagCreationReposForTest(t)
	ctx := context.Background()

	svc := NewTagCreationService(tagRepo, nil)
	_, err := svc.ResolveOrCreateManualTag(ctx, TagCreationRequest{
		PreferredLabel: "orphan parent",
		Level:          domain.TagLevelParent,
	})
	if !errors.Is(err, ErrInvalidHierarchy) {
		t.Fatalf("ResolveOrCreateManualTag() error = %v, want %v", err, ErrInvalidHierarchy)
	}
}

func newTagCreationReposForTest(t *testing.T) (repository.TagRepository, repository.TagAliasRepository) {
	t.Helper()

	_, tagRepo, aliasRepo, _ := newTagAdminServiceForTest(t)
	return tagRepo, aliasRepo
}
