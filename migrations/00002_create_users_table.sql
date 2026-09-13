-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL REFERENCES companies (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    username VARCHAR(64) NOT NULL,
    email VARCHAR(128) NULL,
    phone VARCHAR(32) NULL,
    password_hash VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_company_id ON users (company_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_company_username ON users (company_id, username)
WHERE
    deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email)
WHERE
    deleted_at IS NULL
    AND email IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone ON users (phone)
WHERE
    deleted_at IS NULL
    AND phone IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS users;