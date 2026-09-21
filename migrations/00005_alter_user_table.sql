-- +goose Up
ALTER TABLE
    users
ADD
    COLUMN IF NOT EXISTS is_verified_phone bool DEFAULT false;

ALTER TABLE
    users
ADD
    COLUMN IF NOT EXISTS is_verified_email bool DEFAULT false;

-- +goose Down
ALTER TABLE
    users DROP COLUMN IF EXISTS is_verified_phone;

ALTER TABLE
    users DROP COLUMN IF EXISTS is_verified_email;