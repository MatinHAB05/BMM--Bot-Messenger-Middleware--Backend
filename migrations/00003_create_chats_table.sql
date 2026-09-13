-- +goose Up
CREATE TABLE IF NOT EXISTS chats (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NULL REFERENCES companies (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    platform VARCHAR(16) NOT NULL CHECK (platform IN ('telegram', 'bale')),
    platform_chat_id VARCHAR(128) NOT NULL,
    title VARCHAR(255),
    username VARCHAR(128),
    chat_type VARCHAR(32) CHECK (
        chat_type IS NULL
        OR chat_type IN ('channel', 'group', 'supergroup')
    ),
    is_private BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_chats_company_id ON chats (company_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chats_company_platform_chatid ON chats (company_id, platform, platform_chat_id)
WHERE
    deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_chats_deleted_at ON chats (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS chats;