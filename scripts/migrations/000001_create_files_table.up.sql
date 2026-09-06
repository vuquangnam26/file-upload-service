-- +migrate Up
CREATE TABLE IF NOT EXISTS files (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255)  NOT NULL,
    original_name VARCHAR(255)  NOT NULL,
    mime_type     VARCHAR(100)  NOT NULL,
    size          BIGINT        NOT NULL,
    bucket        VARCHAR(100)  NOT NULL,
    object_key    VARCHAR(500)  NOT NULL,
    status        VARCHAR(50)   NOT NULL DEFAULT 'uploaded',
    checksum      VARCHAR(128),
    uploaded_by   VARCHAR(255),
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    CONSTRAINT uq_files_object_key UNIQUE (object_key)
);

CREATE INDEX IF NOT EXISTS idx_files_status       ON files (status);
CREATE INDEX IF NOT EXISTS idx_files_uploaded_by  ON files (uploaded_by);
CREATE INDEX IF NOT EXISTS idx_files_deleted_at   ON files (deleted_at);
