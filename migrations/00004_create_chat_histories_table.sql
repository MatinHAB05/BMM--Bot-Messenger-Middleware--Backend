-- +goose Up
CREATE TABLE IF NOT EXISTS chat_histories (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES chats (id) ON DELETE CASCADE ON UPDATE CASCADE,
    platform_message_id BIGINT,
    sender_id VARCHAR(128),
    sender_name VARCHAR(255),
    content TEXT,
    media_type VARCHAR(32) NOT NULL DEFAULT 'text',
    raw_payload JSONB,
    message_timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_chat_histories_chat_timestamp ON chat_histories (chat_id, message_timestamp);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_histories_chat_platform_msgid ON chat_histories (chat_id, platform_message_id)
WHERE
    platform_message_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chat_histories_deleted_at ON chat_histories (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS chat_histories;