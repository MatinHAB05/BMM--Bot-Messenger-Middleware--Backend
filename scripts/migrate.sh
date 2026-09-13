#!/usr/bin/env bash
set -euo pipefail

# Applies pending Goose migrations against the database configured via env
# vars (or a sourced .env). The app also applies migrations automatically
# on boot (see bootstrap/init.go); this script is for CI or manual ops
# when you want to run them independently.

: "${DB_HOST:=localhost}"
: "${DB_PORT:=5432}"
: "${DB_USER:=postgres}"
: "${DB_PASSWORD:=postgres}"
: "${DB_NAME:=messenger_backend}"
: "${DB_SSLMODE:=disable}"

if ! command -v goose >/dev/null 2>&1; then
    echo "goose CLI not found. Install it with:" >&2
    echo "  go install github.com/pressly/goose/v3/cmd/goose@latest" >&2
    exit 1
fi

DSN="host=${DB_HOST} port=${DB_PORT} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} sslmode=${DB_SSLMODE}"

goose -dir migrations postgres "${DSN}" up
