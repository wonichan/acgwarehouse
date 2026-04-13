package repository

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestImageTagSourceEnsureScanSchemaIsIdempotent(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-tag-source-schema.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(scanSchemaSQL); err != nil {
		t.Fatalf("apply base schema: %v", err)
	}

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() first run error = %v", err)
	}
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() second run error = %v", err)
	}

	if !hasColumn(t, db, "image_tags", "source") {
		t.Fatal("missing column source after EnsureScanSchema idempotent runs")
	}
}

func TestImageTagSourceMigrationAddsColumnAndBackfillsAI(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-tag-source-migration.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	baseSchema, err := os.ReadFile(filepath.Join("..", "..", "migrations", "001_initial_schema.up.sql"))
	if err != nil {
		t.Fatalf("read base migration: %v", err)
	}
	if _, err := db.Exec(string(baseSchema)); err != nil {
		t.Fatalf("apply base migration: %v", err)
	}

	mustSeedImageTagSourceMigrationData(t, db)

	upSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "003_add_image_tag_source.up.sql"))
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	if _, err := db.Exec(string(upSQL)); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}

	if !hasColumn(t, db, "image_tags", "source") {
		t.Fatal("missing source column after up migration")
	}

	var sourceWithObservation string
	if err := db.QueryRow(`SELECT source FROM image_tags WHERE image_id = 1 AND tag_id = 1`).Scan(&sourceWithObservation); err != nil {
		t.Fatalf("query source for AI tag: %v", err)
	}
	if sourceWithObservation != domain.ImageTagSourceAI {
		t.Fatalf("source for AI tag = %q, want %q", sourceWithObservation, domain.ImageTagSourceAI)
	}

	var sourceManual string
	if err := db.QueryRow(`SELECT source FROM image_tags WHERE image_id = 2 AND tag_id = 2`).Scan(&sourceManual); err != nil {
		t.Fatalf("query source for manual tag: %v", err)
	}
	if sourceManual != domain.ImageTagSourceManual {
		t.Fatalf("source for manual tag = %q, want %q", sourceManual, domain.ImageTagSourceManual)
	}
}

func TestImageTagSourceMigrationIsIdempotent(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-tag-source-migration-idempotent.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	baseSchema, err := os.ReadFile(filepath.Join("..", "..", "migrations", "001_initial_schema.up.sql"))
	if err != nil {
		t.Fatalf("read base migration: %v", err)
	}
	if _, err := db.Exec(string(baseSchema)); err != nil {
		t.Fatalf("apply base migration: %v", err)
	}

	upSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "003_add_image_tag_source.up.sql"))
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	if _, err := db.Exec(string(upSQL)); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}

	if err := ensureColumnExists(db, "image_tags", "source", "TEXT NOT NULL DEFAULT 'manual'"); err != nil {
		t.Fatalf("ensureColumnExists() first run error = %v", err)
	}
	if err := ensureColumnExists(db, "image_tags", "source", "TEXT NOT NULL DEFAULT 'manual'"); err != nil {
		t.Fatalf("ensureColumnExists() second run error = %v", err)
	}
}

func mustSeedImageTagSourceMigrationData(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Now()
	if _, err := db.Exec(`
		INSERT INTO images (id, path, filename, source_root, file_size, width, height, format, created_at, updated_at)
		VALUES
			(1, '/images/1.png', '1.png', '/images', 100, 100, 100, 'png', ?, ?),
			(2, '/images/2.png', '2.png', '/images', 100, 100, 100, 'png', ?, ?)
	`, now, now, now, now); err != nil {
		t.Fatalf("seed images: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO tags (id, preferred_label, slug, usage_count)
		VALUES
			(1, 'blue sky', 'blue-sky', 1),
			(2, 'rain night', 'rain-night', 1)
	`); err != nil {
		t.Fatalf("seed tags: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO tag_observations (id, image_id, raw_text, confidence, evidence_type, provider, model_name, prompt_version, created_at)
		VALUES (1, 1, 'blue sky', 0.9, 'ai_generated', 'qwen', 'qwen-vl-max', 'v1', ?)
	`, now); err != nil {
		t.Fatalf("seed observations: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO image_tags (image_id, tag_id, source_observation_id, confidence, review_state)
		VALUES
			(1, 1, 1, 0.9, 'pending'),
			(2, 2, NULL, 1.0, 'confirmed')
	`); err != nil {
		t.Fatalf("seed image_tags: %v", err)
	}
}
