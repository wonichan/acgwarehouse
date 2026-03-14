-- Images table (metadata only, no binary blobs)
CREATE TABLE images (
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

CREATE INDEX idx_images_path ON images(path);
CREATE INDEX idx_images_source_root ON images(source_root);
CREATE INDEX idx_images_phash ON images(phash);

-- Standard tags
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    preferred_label TEXT UNIQUE NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    primary_category TEXT,
    review_state TEXT DEFAULT 'pending',
    trust_score REAL DEFAULT 0.0,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tags_slug ON tags(slug);
CREATE INDEX idx_tags_category ON tags(primary_category);

-- Tag aliases
CREATE TABLE tag_aliases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tag_id INTEGER NOT NULL,
    label TEXT NOT NULL,
    normalized_label TEXT NOT NULL,
    locale TEXT,
    alias_type TEXT DEFAULT 'synonym',
    is_preferred INTEGER DEFAULT 0,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_tag_aliases_label ON tag_aliases(label);
CREATE INDEX idx_tag_aliases_normalized ON tag_aliases(normalized_label);

-- AI raw observations
CREATE TABLE tag_observations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    image_id INTEGER NOT NULL,
    raw_text TEXT NOT NULL,
    confidence REAL,
    evidence_type TEXT DEFAULT 'ai_generated',
    provider TEXT,
    model_name TEXT,
    prompt_version TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE
);

CREATE INDEX idx_tag_observations_image ON tag_observations(image_id);
CREATE INDEX idx_tag_observations_provider ON tag_observations(provider);

-- Image-tag associations
CREATE TABLE image_tags (
    image_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    source_observation_id INTEGER,
    confidence REAL,
    review_state TEXT DEFAULT 'pending',
    PRIMARY KEY (image_id, tag_id),
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    FOREIGN KEY (source_observation_id) REFERENCES tag_observations(id) ON DELETE SET NULL
);

CREATE INDEX idx_image_tags_tag ON image_tags(tag_id);

-- Collections
CREATE TABLE collections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    cover_image_id INTEGER,
    image_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cover_image_id) REFERENCES images(id) ON DELETE SET NULL
);

CREATE TABLE collection_images (
    collection_id INTEGER NOT NULL,
    image_id INTEGER NOT NULL,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (collection_id, image_id),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE
);

-- Async jobs queue
CREATE TABLE async_jobs (
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

CREATE INDEX idx_async_jobs_status ON async_jobs(status);
CREATE INDEX idx_async_jobs_type ON async_jobs(type);
