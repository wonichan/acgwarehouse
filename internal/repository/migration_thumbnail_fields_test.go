package repository

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestThumbnailFieldMigrations(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "migrate-test.db"))
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

	upSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "002_add_thumbnail_fields.up.sql"))
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	if _, err := db.Exec(string(upSQL)); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}

	if !hasColumn(t, db, "images", "thumbnail_small_url") {
		t.Fatal("missing column thumbnail_small_url after up migration")
	}
	if !hasColumn(t, db, "images", "thumbnail_large_url") {
		t.Fatal("missing column thumbnail_large_url after up migration")
	}

	downSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "002_add_thumbnail_fields.down.sql"))
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if _, err := db.Exec(string(downSQL)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}

	if hasColumn(t, db, "images", "thumbnail_small_url") {
		t.Fatal("column thumbnail_small_url still exists after down migration")
	}
	if hasColumn(t, db, "images", "thumbnail_large_url") {
		t.Fatal("column thumbnail_large_url still exists after down migration")
	}
}

func hasColumn(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()

	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("query table info: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			defaultV  sql.NullString
			primaryKV int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &primaryKV); err != nil {
			t.Fatalf("scan table info: %v", err)
		}
		if name == column {
			return true
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info: %v", err)
	}

	return false
}
