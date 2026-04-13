package repository

import (
	"database/sql"
	"fmt"
	"strings"
)

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
    phash_hex TEXT,
    sha256 TEXT,
    source_mtime_unix INTEGER,
    thumbnail_small_url TEXT,
    thumbnail_large_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_images_path ON images(path);
CREATE INDEX IF NOT EXISTS idx_images_source_root ON images(source_root);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    preferred_label TEXT UNIQUE NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    level TEXT NOT NULL DEFAULT 'child' CHECK(level IN ('root', 'parent', 'child')),
    parent_id INTEGER REFERENCES tags(id) ON DELETE SET NULL,
    primary_category TEXT,
    review_state TEXT DEFAULT 'pending',
    trust_score REAL DEFAULT 0.0,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tags_slug ON tags(slug);
CREATE INDEX IF NOT EXISTS idx_tags_category ON tags(primary_category);

CREATE TABLE IF NOT EXISTS tag_aliases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tag_id INTEGER NOT NULL,
    label TEXT NOT NULL,
    normalized_label TEXT NOT NULL,
    locale TEXT,
    alias_type TEXT DEFAULT 'synonym',
    is_preferred INTEGER DEFAULT 0,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_aliases_label ON tag_aliases(label);
CREATE INDEX IF NOT EXISTS idx_tag_aliases_normalized ON tag_aliases(normalized_label);

CREATE TABLE IF NOT EXISTS async_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform_task_id INTEGER,
    type TEXT NOT NULL,
    status TEXT DEFAULT 'ready',
    payload TEXT,
    progress REAL DEFAULT 0.0,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    FOREIGN KEY (platform_task_id) REFERENCES platform_tasks(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_async_jobs_status ON async_jobs(status);
CREATE INDEX IF NOT EXISTS idx_async_jobs_type ON async_jobs(type);
CREATE INDEX IF NOT EXISTS idx_async_jobs_platform_task_id ON async_jobs(platform_task_id);

CREATE TABLE IF NOT EXISTS task_batches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_type TEXT NOT NULL,
    trigger_key TEXT DEFAULT '',
    summary_label TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    total_images INTEGER NOT NULL DEFAULT 0,
    new_images INTEGER NOT NULL DEFAULT 0,
    skipped_images INTEGER NOT NULL DEFAULT 0,
    skipped_unchanged INTEGER NOT NULL DEFAULT 0,
    skipped_duplicate_tasks INTEGER NOT NULL DEFAULT 0,
    latest_error_summary TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_task_batches_source_type ON task_batches(source_type);
CREATE INDEX IF NOT EXISTS idx_task_batches_status ON task_batches(status);
CREATE INDEX IF NOT EXISTS idx_task_batches_created_at ON task_batches(created_at DESC);

CREATE TABLE IF NOT EXISTS task_batch_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    batch_id INTEGER NOT NULL,
    source_root TEXT NOT NULL,
    source_label TEXT DEFAULT '',
    FOREIGN KEY (batch_id) REFERENCES task_batches(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_task_batch_sources_batch_id ON task_batch_sources(batch_id);

CREATE TABLE IF NOT EXISTS platform_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    batch_id INTEGER NOT NULL,
    image_id INTEGER NOT NULL,
    task_type TEXT NOT NULL,
    source_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    dedupe_key TEXT NOT NULL,
    image_version_key TEXT NOT NULL,
    latest_async_job_id INTEGER,
    skip_reason TEXT,
    error_summary TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    queued_at TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    FOREIGN KEY (batch_id) REFERENCES task_batches(id) ON DELETE CASCADE,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    FOREIGN KEY (latest_async_job_id) REFERENCES async_jobs(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_platform_tasks_batch_id ON platform_tasks(batch_id);
CREATE INDEX IF NOT EXISTS idx_platform_tasks_image_id ON platform_tasks(image_id);
CREATE INDEX IF NOT EXISTS idx_platform_tasks_status ON platform_tasks(status);
CREATE INDEX IF NOT EXISTS idx_platform_tasks_task_type ON platform_tasks(task_type);
CREATE INDEX IF NOT EXISTS idx_platform_tasks_dedupe_key ON platform_tasks(dedupe_key);

CREATE TABLE IF NOT EXISTS tag_observations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    image_id INTEGER NOT NULL,
    raw_text TEXT NOT NULL,
    confidence REAL DEFAULT 0.0,
    evidence_type TEXT DEFAULT 'ai_generated',
    provider TEXT,
    model_name TEXT,
    prompt_version TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tag_observations_image_id ON tag_observations(image_id);
CREATE INDEX IF NOT EXISTS idx_tag_observations_provider ON tag_observations(provider);

CREATE TABLE IF NOT EXISTS image_tags (
    image_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    source TEXT NOT NULL DEFAULT 'manual',
    source_observation_id INTEGER,
    confidence REAL,
    review_state TEXT DEFAULT 'pending',
    PRIMARY KEY (image_id, tag_id),
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    FOREIGN KEY (source_observation_id) REFERENCES tag_observations(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_image_tags_tag ON image_tags(tag_id);

-- FTS5 全文索引虚拟表
CREATE VIRTUAL TABLE IF NOT EXISTS images_fts USING fts5(
    image_id UNINDEXED,
    filename,
    tags,
    tokenize = 'unicode61'
);

-- 同步触发器：插入图片时自动添加 FTS 记录
CREATE TRIGGER IF NOT EXISTS images_ai AFTER INSERT ON images BEGIN
    INSERT INTO images_fts(image_id, filename, tags)
    VALUES (new.id, new.filename, '');
END;

-- 同步触发器：删除图片时自动清理 FTS 记录
CREATE TRIGGER IF NOT EXISTS images_ad AFTER DELETE ON images BEGIN
    DELETE FROM images_fts WHERE image_id = old.id;
END;

-- 同步触发器：更新图片时自动更新 FTS 记录
CREATE TRIGGER IF NOT EXISTS images_au AFTER UPDATE ON images BEGIN
    UPDATE images_fts SET filename = new.filename WHERE image_id = old.id;
END;

-- 收藏夹表
CREATE TABLE IF NOT EXISTS collections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    cover_image_id INTEGER,
    image_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cover_image_id) REFERENCES images(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_collections_name ON collections(name);
CREATE INDEX IF NOT EXISTS idx_collections_created_at ON collections(created_at);

-- 收藏夹-图片关联表
CREATE TABLE IF NOT EXISTS collection_images (
    collection_id INTEGER NOT NULL,
    image_id INTEGER NOT NULL,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (collection_id, image_id),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_collection_images_image_id ON collection_images(image_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_collection_images_image_id_unique ON collection_images(image_id);
CREATE INDEX IF NOT EXISTS idx_collection_images_added_at ON collection_images(added_at);
`

const tagUsageMigrationSQL = `
DROP TRIGGER IF EXISTS trg_image_tags_after_insert;
DROP TRIGGER IF EXISTS trg_image_tags_after_delete;
DROP TRIGGER IF EXISTS trg_image_tags_after_update;
DROP TRIGGER IF EXISTS trg_image_tags_after_update_tagid;
DROP TRIGGER IF EXISTS trg_image_tags_after_update_review;

CREATE TRIGGER trg_image_tags_after_insert
AFTER INSERT ON image_tags
FOR EACH ROW
WHEN NEW.review_state != 'rejected'
BEGIN
	UPDATE tags SET usage_count = usage_count + 1 WHERE id = NEW.tag_id;
END;

CREATE TRIGGER trg_image_tags_after_delete
AFTER DELETE ON image_tags
FOR EACH ROW
WHEN OLD.review_state != 'rejected'
BEGIN
	UPDATE tags SET usage_count = MAX(usage_count - 1, 0) WHERE id = OLD.tag_id;
END;

CREATE TRIGGER trg_image_tags_after_update_tagid
AFTER UPDATE ON image_tags
FOR EACH ROW
WHEN OLD.tag_id != NEW.tag_id
BEGIN
	UPDATE tags SET usage_count = MAX(usage_count - 1, 0) WHERE id = OLD.tag_id AND OLD.review_state != 'rejected';
	UPDATE tags SET usage_count = usage_count + 1 WHERE id = NEW.tag_id AND NEW.review_state != 'rejected';
END;

CREATE TRIGGER trg_image_tags_after_update_review
AFTER UPDATE ON image_tags
FOR EACH ROW
WHEN OLD.review_state != NEW.review_state
BEGIN
	UPDATE tags SET usage_count = MAX(usage_count - 1, 0) WHERE id = NEW.tag_id AND OLD.review_state != 'rejected' AND NEW.review_state = 'rejected';
	UPDATE tags SET usage_count = usage_count + 1 WHERE id = NEW.tag_id AND OLD.review_state = 'rejected' AND NEW.review_state != 'rejected';
END;
`

func EnsureScanSchema(db *sql.DB) error {
	if _, err := db.Exec(scanSchemaSQL); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "async_jobs", "platform_task_id", "INTEGER REFERENCES platform_tasks(id) ON DELETE SET NULL"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "tags", "level", "TEXT NOT NULL DEFAULT 'child'"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "tags", "parent_id", "INTEGER REFERENCES tags(id) ON DELETE SET NULL"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "image_tags", "source", "TEXT NOT NULL DEFAULT 'manual'"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "images", "phash_hex", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "images", "sha256", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "images", "source_mtime_unix", "INTEGER"); err != nil {
		return err
	}
	if _, err := db.Exec(`
		DELETE FROM collection_images
		WHERE rowid NOT IN (
			SELECT MIN(rowid)
			FROM collection_images
			GROUP BY image_id
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_collection_images_image_id_unique ON collection_images(image_id)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_tags_parent_id ON tags(parent_id)`); err != nil {
		return err
	}
	if err := ensureTagLevelCheckConstraint(db); err != nil {
		return err
	}
	_, err := db.Exec(`
		UPDATE tags
		SET level = 'child'
		WHERE level IS NULL OR TRIM(level) = ''
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE image_tags
		SET source = 'ai'
		WHERE source_observation_id IS NOT NULL
		  AND (source IS NULL OR source = '' OR source != 'ai')
	`)
	if err != nil {
		return err
	}

	// Always recreate usage-count triggers to recover from any prior partial migration
	if _, err := db.Exec(tagUsageMigrationSQL); err != nil {
		return fmt.Errorf("tag usage triggers: %w", err)
	}

	if _, err := db.Exec(`
		UPDATE tags
		SET usage_count = (
			SELECT COUNT(*) FROM image_tags WHERE tag_id = tags.id AND review_state != 'rejected'
		)
	`); err != nil {
		return err
	}

	return nil
}

func ensureTagLevelCheckConstraint(db *sql.DB) (err error) {
	var tableSQL sql.NullString
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='tags'`).Scan(&tableSQL); err != nil {
		return err
	}
	if strings.Contains(strings.ToUpper(tableSQL.String), "CHECK(LEVEL IN") {
		return nil
	}

	if _, err := db.Exec(`
		UPDATE tags
		SET level = 'child'
		WHERE level IS NULL OR TRIM(level) = '' OR level NOT IN ('root', 'parent', 'child')
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	foreignKeysDisabled := true
	defer func() {
		if !foreignKeysDisabled {
			return
		}
		if pragmaErr := setForeignKeysPragma(db, true); err == nil {
			err = pragmaErr
		}
	}()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx == nil {
			return
		}
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec(`
		CREATE TABLE tags_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			preferred_label TEXT UNIQUE NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			level TEXT NOT NULL DEFAULT 'child' CHECK(level IN ('root', 'parent', 'child')),
			parent_id INTEGER REFERENCES tags(id) ON DELETE SET NULL,
			primary_category TEXT,
			review_state TEXT DEFAULT 'pending',
			trust_score REAL DEFAULT 0.0,
			usage_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO tags_new (id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at)
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE tags`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE tags_new RENAME TO tags`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX idx_tags_slug ON tags(slug)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX idx_tags_category ON tags(primary_category)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX idx_tags_parent_id ON tags(parent_id)`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	tx = nil

	if err := setForeignKeysPragma(db, true); err != nil {
		return err
	}
	foreignKeysDisabled = false

	return nil
}

func ensureColumnExists(db *sql.DB, tableName, columnName, definition string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			declType   string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &declType, &notNull, &defaultVal, &pk); err != nil {
			return err
		}
		if name == columnName {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, definition))
	return err
}

func setForeignKeysPragma(db *sql.DB, enabled bool) error {
	value := "OFF"
	if enabled {
		value = "ON"
	}
	_, err := db.Exec("PRAGMA foreign_keys = " + value)
	return err
}
