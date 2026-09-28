-- +goose Up
ALTER TABLE
    companies
ADD
    COLUMN IF NOT EXISTS description TEXT DEFAULT '';

-- +goose Down
ALTER TABLE
    companies DROP COLUMN IF EXISTS description;