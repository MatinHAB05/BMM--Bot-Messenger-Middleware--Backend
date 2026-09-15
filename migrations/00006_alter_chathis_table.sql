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
    users DROP COLUMN IF EXISTS is_verified_phone;

ALTER TABLE
    users DROP COLUMN IF EXISTS is_verified_email;