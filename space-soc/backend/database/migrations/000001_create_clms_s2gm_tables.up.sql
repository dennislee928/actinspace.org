-- CLMS/S2GM 最小表結構（PostgreSQL）
-- SQLite 開發環境請依 GORM AutoMigrate 建立表

CREATE TABLE IF NOT EXISTS dataset_ref (
    id SERIAL PRIMARY KEY,
    dataset_uid TEXT NOT NULL,
    download_info_id TEXT NOT NULL,
    title TEXT,
    product_family TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS source_job (
    id SERIAL PRIMARY KEY,
    source TEXT NOT NULL CHECK (source IN ('clms', 's2gm')),
    aoi_geom TEXT,
    time_start TIMESTAMPTZ,
    time_end TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending',
    external_task_id TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_source_job_source ON source_job(source);
CREATE INDEX IF NOT EXISTS idx_source_job_status ON source_job(status);
CREATE INDEX IF NOT EXISTS idx_source_job_external_task_id ON source_job(external_task_id);

CREATE TABLE IF NOT EXISTS asset (
    id SERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES source_job(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    mime TEXT,
    crs TEXT,
    resolution TEXT,
    bands TEXT,
    storage_url TEXT NOT NULL,
    checksum TEXT,
    size_bytes BIGINT DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_asset_job_id ON asset(job_id);
