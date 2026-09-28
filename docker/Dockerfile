# syntax=docker/dockerfile:1

# Arguments for flexible base image versions
ARG GO_VERSION=1.26-alpine
ARG ALPINE_VERSION=3.20

# ==============================================================================
# Stage 1: Build stage
# ==============================================================================
FROM golang:${GO_VERSION} AS builder

# Install build prerequisites
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Leverage caching for dependency downloads
COPY go.mod go.sum* ./
RUN go mod download

# Copy application source code
COPY . .

# Ensure adminchats.env.json exists; create fallback empty JSON array if missing
RUN if [ ! -f adminchats.env.json ]; then echo "[]" > adminchats.env.json; fi

# Build statically linked binary with stripped debug symbols
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /out/app ./cmd/app

# ==============================================================================
# Stage 2: Minimal & secure runtime stage
# ==============================================================================
FROM alpine:${ALPINE_VERSION}

# Install runtime certificates, timezones, and healthcheck utilities
RUN apk add --no-cache ca-certificates tzdata wget curl && \
    addgroup -g 10001 -S app && \
    adduser -u 10001 -S app -G app

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /out/app ./app

# Copy runtime assets and configurations required by the application
COPY --from=builder /src/migrations ./migrations
COPY --from=builder /src/config ./config
COPY --from=builder /src/static ./static
COPY --from=builder /src/adminchats.env.json ./adminchats.env.json

# Copy .env.example as baseline .env so Viper registers all key definitions
COPY --from=builder /src/.env.example ./.env

# Pre-create all logging directories with appropriate ownership
RUN mkdir -p logs/app logs/error logs/telegram logs/bale \
             clean-logs/app clean-logs/error clean-logs/telegram clean-logs/bale && \
    chown -R app:app /app

# Switch to non-root user
USER app

# Default HTTP application port
EXPOSE 8080

# Health check using HTTP GET /health
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["./app"]
