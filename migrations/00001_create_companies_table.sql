-- +goose Up
CREATE TABLE IF NOT EXISTS companies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    code VARCHAR(64) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now (),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_code ON companies (code)
WHERE
    deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_companies_deleted_at ON companies (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS companies;