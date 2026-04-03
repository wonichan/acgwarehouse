package sqliteutil

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
)

const busyTimeoutMilliseconds = 5000
const minSQLiteOpenConns = 4
const sqliteConnectionHeadroom = 2
const maxSQLiteOpenConns = 16

func Open(cfg *config.Config) (*sql.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if strings.EqualFold(cfg.Database.Type, "postgres") {
		return nil, fmt.Errorf("postgres scanning bootstrap is not implemented yet")
	}

	dbPath := strings.TrimSpace(cfg.Database.Path)
	if dbPath == "" {
		return nil, fmt.Errorf("sqlite database path is required")
	}

	dsn, err := buildSQLiteDSN(dbPath)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	maxOpenConns := 1
	if !isInMemorySQLite(dbPath) {
		maxOpenConns = desiredSQLiteOpenConns(cfg)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns)

	if isInMemorySQLite(dbPath) {
		pragmas := []string{
			fmt.Sprintf("PRAGMA busy_timeout = %d", busyTimeoutMilliseconds),
			"PRAGMA foreign_keys = ON",
		}
		for _, pragma := range pragmas {
			if _, err := db.Exec(pragma); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("configure sqlite with %q: %w", pragma, err)
			}
		}
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func buildSQLiteDSN(dbPath string) (string, error) {
	if isInMemorySQLite(dbPath) {
		return dbPath, nil
	}

	base := strings.TrimSpace(dbPath)
	if !strings.HasPrefix(strings.ToLower(base), "file:") {
		base = "file:" + filepath.ToSlash(base)
	}

	query := make(url.Values)
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busyTimeoutMilliseconds))
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "foreign_keys(ON)")
	separator := "?"
	if strings.Contains(base, "?") {
		separator = "&"
	}
	return base + separator + query.Encode(), nil
}

func desiredSQLiteOpenConns(cfg *config.Config) int {
	workerCount := 0
	if cfg != nil {
		workerCount = cfg.WorkerPool.WorkerCount
	}
	poolSize := workerCount + sqliteConnectionHeadroom
	if poolSize < minSQLiteOpenConns {
		poolSize = minSQLiteOpenConns
	}
	if poolSize > maxSQLiteOpenConns {
		poolSize = maxSQLiteOpenConns
	}
	return poolSize
}

func isInMemorySQLite(dbPath string) bool {
	path := strings.ToLower(strings.TrimSpace(dbPath))
	return path == ":memory:" || strings.Contains(path, "mode=memory")
}
