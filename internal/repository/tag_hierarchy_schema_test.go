package repository

import (
	"database/sql"
	"os"
	"path/filepath"
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
