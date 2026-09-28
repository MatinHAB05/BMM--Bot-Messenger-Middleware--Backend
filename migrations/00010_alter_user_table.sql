-- +goose Up
ALTER TABLE
    users
ADD
    COLUMN IF NOT EXISTS firstname VARCHAR(32);

ALTER TABLE
    users
ADD
    COLUMN IF NOT EXISTS lastname VARCHAR(32);

-- +goose Down
ALTER TABLE
    users DROP COLUMN IF EXISTS firstname;

ALTER TABLE
    users DROP COLUMN IF EXISTS lastname;