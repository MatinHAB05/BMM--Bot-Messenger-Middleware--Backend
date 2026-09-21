-- +goose Up
ALTER TABLE
    chat_histories
ADD
    COLUMN IF NOT EXISTS is_broadcast bool DEFAULT false;

ALTER TABLE
    chat_histories
ADD
    COLUMN IF NOT EXISTS broadcast_uuid UUID NULL DEFAULT NULL;

-- +goose Down
ALTER TABLE
    chat_histories DROP COLUMN IF EXISTS broadcast_uuid;

ALTER TABLE
    chat_histories DROP COLUMN IF EXISTS is_broadcast;