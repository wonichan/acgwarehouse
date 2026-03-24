package repository

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestTaskBatchSchemaCreatesBatchTables(t *testing.T) {
	t.Parallel()

	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	mustHaveTable(t, db, "task_batches")
	mustHaveTable(t, db, "task_batch_sources")
	mustHaveTable(t, db, "platform_tasks")
}

func TestSchemaAddsNullablePlatformTaskIDToAsyncJobs(t *testing.T) {
	t.Parallel()

	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	columnType, notNull, found := columnInfo(t, db, "async_jobs", "platform_task_id")
	if !found {
		t.Fatal("expected async_jobs.platform_task_id column to exist")
	}
	if columnType != "INTEGER" {
		t.Fatalf("platform_task_id type = %q, want %q", columnType, "INTEGER")
	}
	if notNull {
		t.Fatal("expected async_jobs.platform_task_id to be nullable")
	}
}

func newTaskPlatformSchemaTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "task-platform-schema.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if err := EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}
	return db
}

func mustHaveTable(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()

	var found string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&found)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Fatalf("expected table %q to exist", tableName)
		}
		t.Fatalf("failed to lookup table %q: %v", tableName, err)
	}
	if found != tableName {
		t.Fatalf("found table = %q, want %q", found, tableName)
	}
}

func columnInfo(t *testing.T, db *sql.DB, tableName, columnName string) (columnType string, notNull bool, found bool) {
	t.Helper()

	rows, err := db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s) error = %v", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			declType   string
			notNullInt int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &declType, &notNullInt, &defaultVal, &pk); err != nil {
			t.Fatalf("scan table_info row error = %v", err)
		}
		if name == columnName {
			return declType, notNullInt == 1, true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info rows error = %v", err)
	}
	return "", false, false
}
