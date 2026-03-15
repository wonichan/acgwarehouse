package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestObservationRepositorySavePersistsObservation(t *testing.T) {
	t.Parallel()

	repo := newObservationRepositoryForTest(t)
	obs := &domain.TagObservation{ImageID: 10, RawText: "blue sky", Confidence: 0.91, Provider: "qwen", ModelName: "qwen-vl-max", PromptVersion: "v1"}

	if err := repo.Save(context.Background(), obs); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if obs.ID == 0 {
		t.Fatal("expected Save() to assign observation ID")
	}
}

func TestObservationRepositoryFindByImageIDReturnsNewestFirst(t *testing.T) {
	t.Parallel()

	repo := newObservationRepositoryForTest(t)
	mustSaveObservation(t, repo, &domain.TagObservation{ImageID: 77, RawText: "first", Provider: "qwen", CreatedAt: time.Now().Add(-time.Minute)})
	mustSaveObservation(t, repo, &domain.TagObservation{ImageID: 77, RawText: "second", Provider: "qwen", CreatedAt: time.Now()})

	observations, err := repo.FindByImageID(context.Background(), 77)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(observations) != 2 {
		t.Fatalf("len(observations) = %d, want 2", len(observations))
	}
	if observations[0].RawText != "second" || observations[1].RawText != "first" {
		t.Fatalf("unexpected order: got [%q, %q]", observations[0].RawText, observations[1].RawText)
	}
}

func TestObservationRepositoryFindByProviderFiltersResults(t *testing.T) {
	t.Parallel()

	repo := newObservationRepositoryForTest(t)
	mustSaveObservation(t, repo, &domain.TagObservation{ImageID: 1, RawText: "one", Provider: "qwen"})
	mustSaveObservation(t, repo, &domain.TagObservation{ImageID: 2, RawText: "two", Provider: "doubao"})
	mustSaveObservation(t, repo, &domain.TagObservation{ImageID: 3, RawText: "three", Provider: "qwen"})

	observations, err := repo.FindByProvider(context.Background(), "qwen", 10)
	if err != nil {
		t.Fatalf("FindByProvider() error = %v", err)
	}
	if len(observations) != 2 {
		t.Fatalf("len(observations) = %d, want 2", len(observations))
	}
	for _, obs := range observations {
		if obs.Provider != "qwen" {
			t.Fatalf("Provider = %q, want qwen", obs.Provider)
		}
	}
}

func newObservationRepositoryForTest(t *testing.T) TagObservationRepository {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-observation-repo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	return NewTagObservationRepository(db)
}

func mustSaveObservation(t *testing.T, repo TagObservationRepository, obs *domain.TagObservation) *domain.TagObservation {
	t.Helper()

	if err := repo.Save(context.Background(), obs); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	return obs
}
