package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func TestTagGovernanceMergeTagsUsesExistingExactMatch(t *testing.T) {
	t.Parallel()

	service, tagRepo, imageTagRepo := newTagGovernanceServiceForTest(t)
	existing := mustSaveGovernanceTag(t, tagRepo, &domain.Tag{PreferredLabel: "blue sky", Slug: "blue-sky", ReviewState: "confirmed", UsageCount: 2})

	if err := service.MergeTags(context.Background(), 1, []string{"blue sky"}, 1, 0.9); err != nil {
		t.Fatalf("MergeTags() error = %v", err)
	}

	items, err := imageTagRepo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(items) != 1 || items[0].TagID != existing.ID {
		t.Fatalf("unexpected image tag association: %+v", items)
	}
}

func TestTagGovernanceMergeTagsCreatesMissingTag(t *testing.T) {
	t.Parallel()

	service, tagRepo, _ := newTagGovernanceServiceForTest(t)

	if err := service.MergeTags(context.Background(), 1, []string{"night rain"}, 1, 0.75); err != nil {
		t.Fatalf("MergeTags() error = %v", err)
	}

	tag, err := tagRepo.FindByLabel(context.Background(), "night rain")
	if err != nil {
		t.Fatalf("FindByLabel() error = %v", err)
	}
	if tag.ID == 0 {
		t.Fatal("expected created tag to have ID")
	}
}

func TestTagGovernanceMergeTagsCreatesPendingTagsByDefault(t *testing.T) {
	t.Parallel()

	service, tagRepo, imageTagRepo := newTagGovernanceServiceForTest(t)

	if err := service.MergeTags(context.Background(), 1, []string{"sun beam"}, 1, 0.82); err != nil {
		t.Fatalf("MergeTags() error = %v", err)
	}

	tag, err := tagRepo.FindByLabel(context.Background(), "sun beam")
	if err != nil {
		t.Fatalf("FindByLabel() error = %v", err)
	}
	if tag.ReviewState != "pending" {
		t.Fatalf("ReviewState = %q, want pending", tag.ReviewState)
	}

	items, err := imageTagRepo.FindByImageID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(items) != 1 || items[0].ReviewState != "pending" {
		t.Fatalf("unexpected image tag review state: %+v", items)
	}
}

func TestTagGovernanceMergeTagsIncrementsUsageCount(t *testing.T) {
	t.Parallel()

	service, tagRepo, _ := newTagGovernanceServiceForTest(t)
	existing := mustSaveGovernanceTag(t, tagRepo, &domain.Tag{PreferredLabel: "forest", Slug: "forest", ReviewState: "confirmed", UsageCount: 3})

	if err := service.MergeTags(context.Background(), 1, []string{"forest"}, 1, 0.9); err != nil {
		t.Fatalf("MergeTags() error = %v", err)
	}

	tag, err := tagRepo.FindByID(context.Background(), existing.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if tag.UsageCount != 4 {
		t.Fatalf("UsageCount = %d, want 4", tag.UsageCount)
	}
}

func newTagGovernanceServiceForTest(t *testing.T) (*TagGovernanceService, repository.TagRepository, repository.ImageTagRepository) {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-governance.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	mustSeedGovernanceData(t, db)

	tagRepo := repository.NewTagRepository(db)
	obsRepo := repository.NewTagObservationRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)

	return NewTagGovernanceService(tagRepo, obsRepo, imageTagRepo), tagRepo, imageTagRepo
}

func mustSeedGovernanceData(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES (1, '/images/1.png', '1.png', '/images', 100, 100, 100, 'png', ?, ?)
	`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("seed images: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO tag_observations (id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at)
		VALUES (1, 1, 'seed', 0.9, 'ai_generated', 'qwen', 'qwen-vl-max', 'v1', ?)
	`, time.Now())
	if err != nil {
		t.Fatalf("seed observations: %v", err)
	}
}

func mustSaveGovernanceTag(t *testing.T, repo repository.TagRepository, tag *domain.Tag) *domain.Tag {
	t.Helper()

	if err := repo.Save(context.Background(), tag); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	return tag
}
