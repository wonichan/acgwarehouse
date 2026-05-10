package sqliteutil

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
)

func TestOpenConfiguresSQLiteForConcurrentWorkers(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "open.db")
	db, err := Open(&config.Config{
		Database:   config.DatabaseConfig{Type: "sqlite", Path: dbPath},
		WorkerPool: config.WorkerPoolConfig{WorkerCount: 2},
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	if got := db.Stats().MaxOpenConnections; got < 2 {
		t.Fatalf("MaxOpenConnections = %d, want >= 2", got)
	}

	ctx := context.Background()
	conn1, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("Conn(1) error = %v", err)
	}
	defer conn1.Close()

	conn2Ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	conn2, err := db.Conn(conn2Ctx)
	if err != nil {
		t.Fatalf("Conn(2) error = %v", err)
	}
	defer conn2.Close()

	for index, conn := range []*sql.Conn{conn1, conn2} {
		busyTimeout := pragmaInt(t, conn, "PRAGMA busy_timeout")
		if busyTimeout < 5000 {
			t.Fatalf("conn[%d] busy_timeout = %d, want >= 5000", index, busyTimeout)
		}

		foreignKeys := pragmaInt(t, conn, "PRAGMA foreign_keys")
		if foreignKeys != 1 {
			t.Fatalf("conn[%d] foreign_keys = %d, want 1", index, foreignKeys)
		}

		journalMode := pragmaText(t, conn, "PRAGMA journal_mode")
		if journalMode != "wal" {
			t.Fatalf("conn[%d] journal_mode = %q, want %q", index, journalMode, "wal")
		}
	}
}

func TestOpenPrefersConnectionStringOverPath(t *testing.T) {
	t.Parallel()

	pathDB := filepath.Join(t.TempDir(), "path.db")
	connectionDB := filepath.Join(t.TempDir(), "connection.db")
	db, err := Open(&config.Config{
		Database: config.DatabaseConfig{
			Type:             "local",
			Path:             pathDB,
			ConnectionString: connectionDB,
		},
		WorkerPool: config.WorkerPoolConfig{WorkerCount: 1},
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	if _, err := db.Exec("CREATE TABLE marker(id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create marker table: %v", err)
	}
	if _, err := os.Stat(connectionDB); err != nil {
		t.Fatalf("expected connection string database at %s: %v", connectionDB, err)
	}
	if _, err := os.Stat(pathDB); !os.IsNotExist(err) {
		t.Fatalf("path database stat error = %v, want not exist", err)
	}
}

func pragmaInt(t *testing.T, conn *sql.Conn, query string) int {
	t.Helper()
	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		t.Fatalf("%s error = %v", query, err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("%s returned no rows", query)
	}
	var value int
	if err := rows.Scan(&value); err != nil {
		t.Fatalf("%s scan error = %v", query, err)
	}
	return value
}

func pragmaText(t *testing.T, conn *sql.Conn, query string) string {
	t.Helper()
	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		t.Fatalf("%s error = %v", query, err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("%s returned no rows", query)
	}
	var value string
	if err := rows.Scan(&value); err != nil {
		t.Fatalf("%s scan error = %v", query, err)
	}
	return value
}
