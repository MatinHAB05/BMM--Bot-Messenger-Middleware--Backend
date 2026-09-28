# 🛡️ BMM Backend — Bot Messenger Middleware Engine

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Gin Framework](https://img.shields.io/badge/Gin-v1.12-008ECF?style=flat-square&logo=gin)](https://gin-gonic.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?style=flat-square&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ed?style=flat-square&logo=docker)](https://www.docker.com/)
[![CI / Docker](https://github.com/MatinHAB05/BMM--Bot-Messenger-Middleware--Backend/actions/workflows/docker-build.yml/badge.svg)](https://github.com/MatinHAB05/BMM--Bot-Messenger-Middleware--Backend/actions/workflows/docker-build.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=flat-square)](LICENSE)

The high-throughput, enterprise-grade core middleware engine powering the **Bot Messenger Middleware (BMM)** platform. Built with **Go** and designed according to **Clean Architecture** and **Domain-Driven Design (DDD)** principles, BMM unifies multi-tenant bot communication across heterogeneous messenger protocols (**Telegram**, **Bale**), orchestrates asynchronous media pipelines, manages granular role-based access control (RBAC), and serves the REST APIs consumed by the BMM Web Dashboard.

---

## 🏗️ Architectural Overview & Clean Architecture (DDD)

BMM Backend is structured into four decoupled concentric layers adhering strictly to Clean Architecture:

```
                      ┌───────────────────────────────────────┐
                      │    BMM Web Dashboard (Frontend)       │
                      └──────────────────┬────────────────────┘
                                         │  HTTP / REST APIs
                                         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ PRESENTATION LAYER (internal/presentation)                                  │
│  • REST API Controllers (/api/v1/*)                                         │
│  • Telegram & Bale Inbound Adapters (Long-Polling & Webhooks)               │
│  • Middlewares (PASETO Auth, Casbin RBAC, Redis Rate Limiting, OTP Parsers) │
├─────────────────────────────────────────────────────────────────────────────┤
│ APPLICATION LAYER (internal/application)                                    │
│  • Use Cases & Service Contracts (service_contract.Services)                │
│  • Business Logic Implementations (Auth, Chat, Broadcast, Attachment, etc.) │
│  • Concurrency & Worker Pools (ants.Pool for Emails & Broadcasts)           │
├─────────────────────────────────────────────────────────────────────────────┤
│ DOMAIN LAYER (internal/domain)                                              │
│  • Core Entities: Company, User, Chat, ChatHistory, Attachment, RBAC        │
│  • Value Objects: PASETO v4 Symmetric Maker, OTP Strategies                 │
│  • Centralized Domain Exceptions & Error Definitions                        │
│  • Repository Contracts (repository_contract.Repositories)                  │
├─────────────────────────────────────────────────────────────────────────────┤
│ INFRASTRUCTURE LAYER (internal/infrastructure)                              │
│  • PostgreSQL Persistence with GORM & Transaction Context Manager           │
│  • Redis 7 Cache, Token Revocation, Rate Limiting & Atomic Transactions     │
│  • S3 Object Storage (MinIO / RustFS) with Presigned URLs                   │
│  • Casbin Enforcer with Multi-Tenant Domain Matching (r = sub, dom, obj, act)│
│  • Database Auto-Migrations (Goose) & Idempotent Seeder                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## ✨ Core Features & Technical Highlights

### 1. Unified Multi-Messenger Engine (Telegram & Bale)
- **Bi-Directional Adapters:** Native integration with **Telegram Bot API** and **Bale Bot API** (Iranian messenger platform utilizing Telegram-compatible bot protocols).
- **Graceful Degradation:** The engine boots smoothly even if one or both bot tokens are omitted (allowing isolated frontend development or single-messenger deployments).
- **Inbound Message Normalization:** Text, photos, videos, voice notes, audio files, documents, and media groups from both platforms are converted into normalized `ChatMessage` and `Attachment` entities.
- **Three Channel Linking Strategies:**
  1. **Bot Token Connect:** Direct registration of custom bot credentials.
  2. **Deep-Link Authentication:** Automated linking handshake when opening bot start links with company tokens.
  3. **Invite Code & OTP Verification:** Bots added as channel administrators listen for authorization OTP commands to bind channels to companies.

### 2. Multi-Tenant Security & PASETO v4 Local Tokens
- **Complete Tenant Isolation:** Every request is authenticated and strictly scoped by `CompanyID`.
- **PASETO v4 Cryptographic Tokens:** Uses symmetric encryption (`v4.local`) over legacy JWTs, eliminating token tampering vulnerabilities and algorithm confusion attacks.
- **Revocation & Refresh Engine:** Refresh tokens and blacklisted access tokens are tracked atomically in Redis with automated TTL expiry.
- **Multi-Factor Verification (OTP Engine):** Modular OTP strategy pipeline (`strategy.go`) supporting Email verification, Phone SMS, company registration, and user invitation handshakes.

### 3. Granular RBAC Engine (Casbin with Domain Support)
- Model defined in [`config/rbac_model.conf`](config/rbac_model.conf):
  ```ini
  [request_definition]
  r = sub, dom, obj, act

  [policy_definition]
  p = sub, dom, obj, act

  [role_definition]
  g = _, _, _

  [matchers]
  m = g(r.sub, p.sub, r.dom) && r.obj == p.obj && r.act == p.act
  ```
- `dom` represents the multi-tenant `CompanyID`, guaranteeing that role assignments in one company never leak or grant access to another tenant's resources.
- Supported Roles: `super-admin` and `admin`.

### 4. High-Performance Concurrency & Broadcast Hub
- **Worker Pools (`ants.Pool`):**
  - `emailPool`: Background worker pool for asynchronous SMTP dispatch.
  - `broadcastPool`: Concurrent mass-message dispatcher across linked groups/channels.
  - `batchPool`: Bulk media and database write optimizer.
- **Media Ingestion & S3 Object Storage:**
  - Backed by **RustFS** / **MinIO** S3-compatible storage.
  - Secure presigned URL generator for client-side streaming and downloads without exposing raw storage credentials.
  - Dedicated Media Group reconciler (`MediaGroupService`) grouping multi-attachment broadcasts into single visual cards.

### 5. Production Observability & Resilience
- **Structured JSON Logging:** Zero-allocation logging powered by `zerolog` with separate logs for general runtime (`logs/app`) and error tracking (`logs/error`).
- **Graceful Shutdown:** Intercepts `SIGINT` / `SIGTERM` signals, closes active HTTP listeners, allows running worker pools to flush pending jobs, and shuts down database/redis connections gracefully within configured timeout windows.

---

## 📡 REST API Route Map (`/api/v1`)

All protected routes require the `Authorization: Bearer <PASETO-TOKEN>` header.

### 🔑 Authentication (`/api/v1/auth`)
| Method | Path | Auth / RBAC | Description |
| :--- | :--- | :---: | :--- |
| `POST` | `/auth/login` | Public (Rate-Limited) | Authenticate via username, email, or phone |
| `POST` | `/auth/register-with-company` | Public | Register new company + super-admin owner |
| `POST` | `/auth/refresh` | Public | Refresh expired access token using refresh token |
| `POST` | `/auth/logout` | Bearer Auth | Invalidate current token session in Redis |
| `POST` | `/auth/verify-register-otp` | Public | Verify registration OTP code |
| `POST` | `/auth/otp/send` | Public | Dispatch authentication or verification OTP |
| `POST` | `/auth/otp/verify` | Public | Verify OTP code |

### 👥 User Governance (`/api/v1/users`)
| Method | Path | Auth / RBAC | Description |
| :--- | :--- | :---: | :--- |
| `GET` | `/users/me` | Bearer + RBAC | Retrieve profile of currently authenticated user |
| `GET` | `/users` | Bearer + RBAC | Paginated and searchable company user list |
| `GET` | `/users/:id` | Bearer + RBAC | Fetch details for a specific user |
| `PUT` | `/users/:id` | Bearer + RBAC | Update personal user details (`firstname`, `lastname`) |
| `PUT` | `/users/:id/roles` | Bearer + RBAC (SuperAdmin) | Update assigned user roles (`super-admin`, `admin`) |
| `PUT` | `/users/:id/email` | Bearer + RBAC | Update email address (requires prior OTP verification) |
| `PUT` | `/users/:id/phone` | Bearer + RBAC | Update phone number (requires prior OTP verification) |
| `PUT` | `/users/:id/username`| Bearer + RBAC | Change user handle |
| `DELETE` | `/users/:id` | Bearer + RBAC (SuperAdmin) | Revoke user access / delete user |

### 🏢 Company Management (`/api/v1/companies`)
| Method | Path | Auth / RBAC | Description |
| :--- | :--- | :---: | :--- |
| `POST` | `/companies` | Public | Initialize new tenant company |
| `GET` | `/companies/me` | Bearer + RBAC | Fetch current company profile & settings |
| `PUT` | `/companies/:id` | Bearer + RBAC (SuperAdmin) | Update company metadata & configuration |
| `DELETE` | `/companies/:id` | Bearer + RBAC (SuperAdmin) | Deactivate company workspace |
| `POST` | `/companies/users/send-register-otp` | Bearer + RBAC | Generate and dispatch user invite code / OTP |

### 💬 Linked Chats & History (`/api/v1/chats`)
| Method | Path | Auth / RBAC | Description |
| :--- | :--- | :---: | :--- |
| `GET` | `/chats` | Bearer + RBAC | List all linked messenger channels/groups |
| `GET` | `/chats/:id` | Bearer + RBAC | Inspect single linked chat |
| `PUT` | `/chats/:id` | Bearer + RBAC (SuperAdmin) | Update chat configuration |
| `DELETE` | `/chats/:id` | Bearer + RBAC (SuperAdmin) | Unlink / disconnect chat from company |
| `POST` | `/chats/otp/send` | Bearer + RBAC | Generate linking handshake OTP for channel binding |
| `GET` | `/chats/:id/history` | Bearer + RBAC | Fetch paginated chat messages with attachments |
| `GET` | `/chats/:id/history/:message_id` | Bearer + RBAC | Get single message details |
| `DELETE` | `/chats/:id/history/:message_id` | Bearer + RBAC | Delete individual message |

### 📢 Broadcast Engine (`/api/v1/broadcast`)
| Method | Path | Auth / RBAC | Description |
| :--- | :--- | :---: | :--- |
| `POST` | `/broadcast` | Bearer + RBAC (Rate-Limited) | Dispatch mass message/media across multiple chats |
| `PUT` | `/broadcast/:id` | Bearer + RBAC | Edit caption of dispatched broadcast message |
| `DELETE` | `/broadcast/:id` | Bearer + RBAC | Delete broadcast message across all linked targets |
| `DELETE` | `/broadcast/batch` | Bearer + RBAC | Bulk delete multiple broadcast messages |

### 📁 Media & Attachments (`/api/v1/attachments`)
| Method | Path | Auth / RBAC | Description |
| :--- | :--- | :---: | :--- |
| `GET` | `/attachments` | Bearer + RBAC | Search and filter attachments (photos, videos, etc.) |
| `POST` | `/attachments/by-messages` | Bearer + RBAC | Batch fetch attachments for list of message IDs |
| `GET` | `/attachments/:id` | Bearer + RBAC | Get attachment metadata |
| `POST` | `/attachments/:id/links/download` | Bearer + RBAC | Generate secure presigned S3 download URL |
| `POST` | `/attachments/:id/links/download/batch` | Bearer + RBAC | Batch resolve presigned download URLs |
| `DELETE` | `/attachments/:id` | Bearer + RBAC | Soft delete attachment |
| `POST` | `/attachments/:id/restore` | Bearer + RBAC | Restore soft-deleted attachment |

---

## 📂 Project Directory Structure

```
BMM--Bot-Messenger-Middleware--Backend/
├── .github/
│   └── workflows/
│       └── docker-build.yml       # Automated Docker build & publish CI workflow
├── bootstrap/                     # Unified application lifecycle and dependency injection
│   ├── constant.go                # Application timeout, retry and buffer constants
│   ├── environment.go             # Viper configuration loader from .env
│   └── init.go                    # Sequential wiring: DB -> Redis -> Services -> Gin Engine
├── cmd/
│   └── app/                       # Application binary entrypoint (main.go)
├── config/
│   └── rbac_model.conf            # Casbin multi-tenant RBAC configuration
├── docker/
│   ├── Dockerfile                 # Multi-stage production Go build
│   ├── docker-compose.yaml        # Local development compose (Postgres, Redis, RustFS)
│   └── docker-compose.production.yaml # Production deployment stack
├── internal/
│   ├── application/               # Application Layer (Use Cases)
│   │   ├── contract/              # Service interfaces (auth, chat, broadcast, etc.)
│   │   └── service/               # Implementation of business logic & worker pools
│   ├── domain/                    # Domain Layer (Enterprise Rules)
│   │   ├── entity/                # Core entities (Company, User, Chat, Attachment, etc.)
│   │   ├── exception/             # Typed domain errors
│   │   ├── otp/                   # OTP generation, verification & strategy handlers
│   │   ├── paseto/                # PASETO v4 local token generator & claims
│   │   └── repository/            # Abstract repository interfaces
│   ├── infrastructure/            # Infrastructure Layer (External Tools & DBs)
│   │   ├── database/              # PostgreSQL GORM connection & transaction manager
│   │   ├── rbac/                  # Casbin Enforcer setup with GORM adapter
│   │   ├── redis/                 # Redis client, session manager & transactions
│   │   ├── repository/            # Concrete repository implementations
│   │   ├── seed/                  # Idempotent database seeder (SuperAdmin & chats)
│   │   └── storage/               # S3 / MinIO / RustFS repository implementation
│   └── presentation/              # Presentation Layer (HTTP & Bots)
│       ├── middleware/            # Auth, Casbin RBAC, rate-limiting & recovery
│       └── v1/
│           ├── api/               # REST API controllers & router definitions
│           ├── bale/              # Bale bot webhook & polling handlers
│           └── telegram/          # Telegram bot webhook & polling handlers
├── migrations/                    # Goose SQL migrations (00001 to 00010)
├── pkg/                           # Shared utility libraries
│   ├── hasher/                    # Bcrypt password hasher
│   ├── logger/                    # Zerolog structured logging
│   ├── messenger/                 # Bot interface contracts & Telegram adapter
│   ├── tellogger/                 # Messenger bot event streaming logger
│   └── wneessen-go-mail/          # High-performance SMTP email engine
├── scripts/                       # DevOps & migration helper scripts
│   ├── clear-logs.sh              # Log cleanup script
│   ├── format.sh                  # Code formatting
│   ├── generate-paseto-key.sh     # Key generator
│   ├── migrate.sh                 # Manual Goose migration script
│   └── run.sh                     # Local run helper
├── .env.example                   # Full environment variable documentation
├── adminchats.env.example.json    # Example configuration for pre-seeded admin chats
└── LICENSE                        # Apache License 2.0
```

---

## ⚙️ Getting Started & Local Development

### Prerequisites
- **Go**: `1.26+` installed
- **Docker & Docker Compose** (for running PostgreSQL, Redis, and RustFS)

### 1. Environment Configuration
Create a `.env` file from the provided template:
```bash
cp .env.example .env
```
Generate a cryptographically secure 32-byte hex key for PASETO:
```bash
# On Linux/macOS or Git Bash:
openssl rand -hex 32

# On Windows PowerShell:
-join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Max 256) })
```
Paste this value into `PASETO_SYMMETRIC_KEY` inside `.env`.

### 2. Start Infrastructure Dependencies
Spin up PostgreSQL 15, Redis 7, and RustFS S3 in Docker:
```bash
cd docker
docker compose up -d postgres redis rustfs
cd ..
```

### 3. Database Migrations (Automatic & Manual)

> [!NOTE]
> **✨ Automatic Migration Execution on Boot:**
> You do **not** need to manually run migrations during normal development or deployment! Whenever the application starts (`go run ./cmd/app` or inside Docker), all pending Goose SQL migrations in `./migrations` are **automatically applied (`up`)** to PostgreSQL during startup by `bootstrap.Init()`.

**🛠️ Manual Migration Ops (Optional):**
If you wish to check migration status, roll back migrations, or execute migrations independently in isolated CI/CD pipelines without starting the full server, you can use the helper script or Goose CLI:
```bash
# Using the provided migration script:
./scripts/migrate.sh

# Or directly using Goose CLI:
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=messenger_backend sslmode=disable" status
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=messenger_backend sslmode=disable" up
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=messenger_backend sslmode=disable" reset
```

### 4. Run Application Locally
```bash
go run ./cmd/app
```
The server will initialize dependencies, automatically run pending database migrations, seed the default `super-admin` account, connect active bot adapters, and start listening on `:8080`.

---

## 🐳 Docker Deployment & CI/CD

### Local Docker Build
Build the container image locally:
```bash
docker build -f docker/Dockerfile -t messenger-backend:latest .
```

### Complete Stack via Docker Compose
To deploy the entire production stack (App + DB + Redis + S3):
```bash
cd docker
docker compose -f docker-compose.production.yaml up -d
```

### 🔄 GitHub Actions CI/CD Pipeline
An automated container build and publication pipeline is configured at [`.github/workflows/docker-build.yml`](.github/workflows/docker-build.yml):
- **Triggers:**
  - **Exclusively Tag-Driven:** Triggers **ONLY** when pushing a release tag matching `v*` (e.g., `git tag v1.0.0 && git push origin v1.0.0`). Regular branch pushes do **not** trigger image builds, preventing registry clutter and conserving CI resources.
  - **Manual Execution:** Can also be triggered manually on-demand from the GitHub Actions web interface (`workflow_dispatch`) with an optional registry push toggle.
- **Container Registry & Automated Lowercasing:**
  - Automatically sanitizes repository names to lowercase OCI compliance and publishes directly to **GitHub Container Registry (GHCR)**:
    `ghcr.io/<owner>/bmm--bot-messenger-middleware--backend`
- **Semantic Version Tagging:**
  - Automatically generates semver release tags (`v1`, `v1.0`, `v1.0.0`), short commit SHA, and `latest` tags via `docker/metadata-action`.
- **Fast Layer Caching:**
  - Implements GitHub Actions Cache backend (`type=gha,mode=max`) for fast incremental layer caching.

---

## 📄 License

This project is open-source software licensed under the **Apache License 2.0**.
See the [LICENSE](LICENSE) file for the full license text.