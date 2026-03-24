package repository

import (
	"database/sql"
	"testing"
)

func TestPlatformTaskSchemaStoresDedupeFields(t *testing.T) {
	t.Parallel()

	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	expectedColumns := map[string]string{
		"batch_id":          "INTEGER",
		"image_id":          "INTEGER",
		"task_type":         "TEXT",
		"image_version_key": "TEXT",
		"dedupe_key":        "TEXT",
	}

	for columnName, wantType := range expectedColumns {
		columnType, _, found := columnInfo(t, db, "platform_tasks", columnName)
		if !found {
			t.Fatalf("expected platform_tasks.%s column to exist", columnName)
		}
		if columnType != wantType {
			t.Fatalf("platform_tasks.%s type = %q, want %q", columnName, columnType, wantType)
		}
	}
}

func TestPlatformTaskSchemaCreatesDedupeLookupIndex(t *testing.T) {
	t.Parallel()

	db := newTaskPlatformSchemaTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'platform_tasks' AND name = ?`, "idx_platform_tasks_dedupe_key").Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Fatal("expected idx_platform_tasks_dedupe_key index to exist")
		}
		t.Fatalf("failed to lookup idx_platform_tasks_dedupe_key: %v", err)
	}
	if name != "idx_platform_tasks_dedupe_key" {
		t.Fatalf("index name = %q, want %q", name, "idx_platform_tasks_dedupe_key")
	}
}
