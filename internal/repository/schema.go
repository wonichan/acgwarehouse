package repository

import "database/sql"

const scanSchemaSQL = `
CREATE TABLE IF NOT EXISTS images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE NOT NULL,
    filename TEXT NOT NULL,
    source_root TEXT NOT NULL,
    file_size INTEGER,
    width INTEGER,
    height INTEGER,
    format TEXT,
    phash INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_images_path ON images(path);
CREATE INDEX IF NOT EXISTS idx_images_source_root ON images(source_root);

CREATE TABLE IF NOT EXISTS async_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    status TEXT DEFAULT 'ready',
    payload TEXT,
    progress REAL DEFAULT 0.0,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_async_jobs_status ON async_jobs(status);
CREATE INDEX IF NOT EXISTS idx_async_jobs_type ON async_jobs(type);
`

func EnsureScanSchema(db *sql.DB) error {
	_, err := db.Exec(scanSchemaSQL)
	return err
}
