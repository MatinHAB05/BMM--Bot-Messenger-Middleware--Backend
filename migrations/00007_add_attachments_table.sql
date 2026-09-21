-- +goose Up
CREATE TABLE IF NOT EXISTS attachments (
    id BIGSERIAL PRIMARY KEY,
    chat_history_id BIGINT NOT NULL REFERENCES chat_histories (id) ON DELETE CASCADE,
    platform_file_id VARCHAR(255) NOT NULL,
    file_type VARCHAR(32) NOT NULL,
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    mime_type VARCHAR(128) NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    thumbnail_platform_file_id VARCHAR(255) NOT NULL DEFAULT '',
    storage_path VARCHAR(1024),
    thumbnail_storage_path VARCHAR(1024),
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    duration INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_attachments_chat_history_id ON attachments (chat_history_id);

CREATE INDEX IF NOT EXISTS idx_attachments_platform_file_id ON attachments (platform_file_id);

CREATE INDEX IF NOT EXISTS idx_attachments_thumb_file_id ON attachments (thumbnail_platform_file_id);

CREATE INDEX IF NOT EXISTS idx_attachments_file_type ON attachments (file_type);

CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments (deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_attachments_deleted_at;

DROP INDEX IF EXISTS idx_attachments_file_type;

DROP INDEX IF EXISTS idx_attachments_thumb_file_id;

DROP INDEX IF EXISTS idx_attachments_platform_file_id;

DROP INDEX IF EXISTS idx_attachments_chat_history_id;

DROP TABLE IF EXISTS attachments;