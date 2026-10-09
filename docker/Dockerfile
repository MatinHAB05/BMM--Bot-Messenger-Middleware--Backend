# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26-alpine
ARG ALPINE_VERSION=3.20

# ==============================================================================
# Stage 1: Build stage
# ==============================================================================
FROM golang:${GO_VERSION} AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .


RUN if [ ! -f adminchats.env.json ]; then echo "[]" > adminchats.env.json; fi

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /out/app ./cmd/app

# ==============================================================================
# Stage 2: Minimal & secure runtime stage
# ==============================================================================
FROM alpine:${ALPINE_VERSION}

RUN apk add --no-cache ca-certificates tzdata wget curl && \
    addgroup -g 10001 -S app && \
    adduser -u 10001 -S app -G app

WORKDIR /app

COPY --from=builder /out/app ./app
COPY --from=builder /src/migrations ./migrations
COPY --from=builder /src/config ./config
COPY --from=builder /src/static ./static
COPY --from=builder /src/adminchats.env.json ./adminchats.env.json


COPY --from=builder /src/.env.example ./.env

RUN mkdir -p logs/app logs/error logs/telegram logs/bale \
             clean-logs/app clean-logs/error clean-logs/telegram clean-logs/bale && \
    chown -R app:app /app

USER app

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider "http://127.0.0.1:${PORT:-8080}/health" || exit 1

ENTRYPOINT ["./app"]