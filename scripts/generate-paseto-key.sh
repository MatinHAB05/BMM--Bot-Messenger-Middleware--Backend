#!/usr/bin/env bash
set -euo pipefail

# Generates a 32-byte, hex-encoded symmetric key suitable for
# PASETO_SYMMETRIC_KEY in .env.
openssl rand -hex 32
