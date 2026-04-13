package repository

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestTagHierarchyEnsureScanSchemaAddsColumnsIdempotently(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-hierarchy-schema.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(scanSchemaSQL); err != nil {
		t.Fatalf("apply scan schema: %v", err)
	}

	downSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "005_add_tag_hierarchy.down.sql"))
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if _, err := db.Exec(string(downSQL)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() first run error = %v", err)
	}
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() second run error = %v", err)
	}

	if !hasColumn(t, db, "tags", "level") {
		t.Fatal("missing level column after EnsureScanSchema")
	}
	if !hasColumn(t, db, "tags", "parent_id") {
		t.Fatal("missing parent_id column after EnsureScanSchema")
	}
	if !hasIndex(t, db, "tags", "idx_tags_parent_id") {
		t.Fatal("missing idx_tags_parent_id after EnsureScanSchema")
	}
	assertTagLevelConstraintRejectsInvalidInsert(t, db)
}

func TestTagHierarchyEnsureScanSchemaRebuildsLegacyTagsWithLevelConstraint(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-hierarchy-level-check.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	legacyScanSchemaSQL := strings.Replace(scanSchemaSQL, "level TEXT NOT NULL DEFAULT 'child' CHECK(level IN ('root', 'parent', 'child'))", "level TEXT NOT NULL DEFAULT 'child'", 1)
	if _, err := db.Exec(legacyScanSchemaSQL); err != nil {
		t.Fatalf("apply legacy scan schema: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO tags (preferred_label, slug, level, usage_count) VALUES ('legacy invalid', 'legacy-invalid', 'invalid', 0)`); err != nil {
		t.Fatalf("seed legacy tag: %v", err)
	}

	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() first run error = %v", err)
	}
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() second run error = %v", err)
	}

	assertTagsTableHasLevelCheckConstraint(t, db)

	var level string
	if err := db.QueryRow(`SELECT level FROM tags WHERE slug = 'legacy-invalid'`).Scan(&level); err != nil {
		t.Fatalf("query normalized level: %v", err)
	}
	if level != "child" {
		t.Fatalf("level = %q, want child", level)
	}

	if !hasIndex(t, db, "tags", "idx_tags_slug") {
		t.Fatal("missing idx_tags_slug after tags rebuild")
	}
	if !hasIndex(t, db, "tags", "idx_tags_category") {
		t.Fatal("missing idx_tags_category after tags rebuild")
	}
	if !hasIndex(t, db, "tags", "idx_tags_parent_id") {
		t.Fatal("missing idx_tags_parent_id after tags rebuild")
	}

	assertTagLevelConstraintRejectsInvalidInsert(t, db)
	assertTagUsageTriggersStillWorkAfterLevelConstraintMigration(t, db)
}

func TestTagHierarchyMigrationAddsColumnsAndBackfillsExistingRows(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-hierarchy-migration.db"))
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

	if _, err := db.Exec(`INSERT INTO tags (preferred_label, slug, usage_count) VALUES ('blue sky', 'blue-sky', 3)`); err != nil {
		t.Fatalf("seed tag: %v", err)
	}

	upSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "005_add_tag_hierarchy.up.sql"))
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	if _, err := db.Exec(string(upSQL)); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}

	if !hasColumn(t, db, "tags", "level") {
		t.Fatal("missing level column after up migration")
	}
	if !hasColumn(t, db, "tags", "parent_id") {
		t.Fatal("missing parent_id column after up migration")
	}

	var level string
	var parentID sql.NullInt64
	if err := db.QueryRow(`SELECT level, parent_id FROM tags WHERE preferred_label = 'blue sky'`).Scan(&level, &parentID); err != nil {
		t.Fatalf("query hierarchy fields: %v", err)
	}
	if level != "child" {
		t.Fatalf("level = %q, want child", level)
	}
	if parentID.Valid {
		t.Fatalf("parent_id = %v, want NULL", parentID)
	}
	if !hasIndex(t, db, "tags", "idx_tags_parent_id") {
		t.Fatal("missing idx_tags_parent_id after up migration")
	}
}

func TestTagHierarchyMigrationDownRemovesColumns(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tag-hierarchy-migration-down.db"))
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

	upSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "005_add_tag_hierarchy.up.sql"))
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	if _, err := db.Exec(string(upSQL)); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}

	downSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "005_add_tag_hierarchy.down.sql"))
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if _, err := db.Exec(string(downSQL)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}

	if hasColumn(t, db, "tags", "level") {
		t.Fatal("column level still exists after down migration")
	}
	if hasColumn(t, db, "tags", "parent_id") {
		t.Fatal("column parent_id still exists after down migration")
	}
	if hasIndex(t, db, "tags", "idx_tags_parent_id") {
		t.Fatal("idx_tags_parent_id still exists after down migration")
	}
}

func hasIndex(t *testing.T, db *sql.DB, table, index string) bool {
	t.Helper()

	rows, err := db.Query("PRAGMA index_list(" + table + ")")
	if err != nil {
		t.Fatalf("query index list: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan index list: %v", err)
		}
		if name == index {
			return true
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate index list: %v", err)
	}

	return false
}

func assertTagsTableHasLevelCheckConstraint(t *testing.T, db *sql.DB) {
	t.Helper()

	var createSQL string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'tags'`).Scan(&createSQL); err != nil {
		t.Fatalf("query tags create SQL: %v", err)
	}
	if createSQL == "" {
		t.Fatal("tags create SQL is empty")
	}
	if !containsTagLevelCheckConstraint(createSQL) {
		t.Fatalf("tags create SQL = %q, want level CHECK constraint", createSQL)
	}
}

func assertTagLevelConstraintRejectsInvalidInsert(t *testing.T, db *sql.DB) {
	t.Helper()

	assertTagsTableHasLevelCheckConstraint(t, db)

	if _, err := db.Exec(`INSERT INTO tags (preferred_label, slug, level) VALUES ('invalid level', 'invalid-level', 'nope')`); err == nil {
		t.Fatal("expected invalid tag level insert to fail")
	}
}

func assertTagUsageTriggersStillWorkAfterLevelConstraintMigration(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`INSERT INTO images (id, path, filename, source_root) VALUES (1, '/tmp/1.png', '1.png', '/tmp')`); err != nil {
		t.Fatalf("seed image: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO tags (id, preferred_label, slug, level, usage_count) VALUES (100, 'trigger tag', 'trigger-tag', 'child', 0)`); err != nil {
		t.Fatalf("seed trigger tag: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO image_tags (image_id, tag_id, source, review_state) VALUES (1, 100, 'manual', 'pending')`); err != nil {
		t.Fatalf("insert image tag: %v", err)
	}

	var usageCount int
	if err := db.QueryRow(`SELECT usage_count FROM tags WHERE id = 100`).Scan(&usageCount); err != nil {
		t.Fatalf("query usage count: %v", err)
	}
	if usageCount != 1 {
		t.Fatalf("usage_count = %d, want 1", usageCount)
	}
}

func containsTagLevelCheckConstraint(createSQL string) bool {
	return containsNormalizedSQL(createSQL, "CHECK(level IN ('root', 'parent', 'child'))")
}

func containsNormalizedSQL(sqlText, want string) bool {
	normalize := func(s string) string {
		replacer := strings.NewReplacer(" ", "", "\n", "", "\r", "", "\t", "")
		return strings.ToUpper(replacer.Replace(s))
	}

	return strings.Contains(normalize(sqlText), normalize(want))
}
