-- +goose Up
ALTER TABLE
    chat_histories
ADD
    COLUMN IF NOT EXISTS has_attachments bool DEFAULT false;

-- +goose Down
ALTER TABLE
    users DROP COLUMN IF EXISTS has_attachments;