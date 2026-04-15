# Local Service Commerce & Content Operations Portal

An offline-first web platform that lets field-service providers publish service
offerings, manage customer requests end-to-end, and govern user-generated
content — all without external network dependencies.

The system covers profile and address management, a service catalog with
region-based shipping fee calculation, a full ticket lifecycle with SLA timers
and file attachments, post-completion reviews and Q&A, an in-app notification
center, content moderation with escalating freezes, user-initiated data export
and deletion (GDPR-style), and a Bronze/Silver/Gold lakehouse with HMAC-protected
ingestion APIs.

---

## Stack

| Layer    | Technology                                            |
|----------|-------------------------------------------------------|
| Frontend | Vue 3 (JavaScript), Pinia, Vue Router, Vite, nginx    |
| Backend  | Go 1.22, Gin                                          |
| Database | MySQL 8.0                                             |
| Auth     | HttpOnly session cookies, CSRF tokens, bcrypt cost 12 |
| Crypto   | AES-256-GCM (field-level), HMAC-SHA256 (internal API) |
| Tests    | Go `testing` (unit + integration), Playwright (E2E)   |
| Runtime  | Docker + docker-compose                               |

---

## Features

- **Five-role RBAC** — Regular User, Service Agent, Moderator, Administrator, Data Operator
- **Encrypted PII** — phone numbers and address lines stored with AES-256-GCM
- **Account hardening** — 30-min inactivity timeout, 24-hr absolute timeout, 5-fail/10-min lockout, 60 req/min general rate limit
- **Service catalog** — categories, offerings, favorites, browsing history, region-aware shipping with cutoff-based ETA
- **Ticket lifecycle** — Accepted → Dispatched → In Service → Completed → Closed (or Cancelled), SLA tracking, multipart attachments (JPG/PNG/PDF, max 5 files × 5 MB)
- **Reviews & Q&A** — 1–5 star reviews with image uploads, aggregated metrics, moderator-only post deletion
- **Notifications** — templated in-app messages with outbox routing for users who disable in-app delivery
- **Content moderation** — sensitive-term dictionary, automatic prohibited-content blocking, borderline review queue, escalating posting freeze (24h → 7d)
- **Privacy center** — user-initiated ZIP data export and GDPR-style deletion with 30-day grace period and irreversible-action confirmation
- **Lakehouse** — Bronze/Silver/Gold layers on local disk, lineage tracking, schema evolution detection, lifecycle archival/purge, legal-hold protection

---

## Prerequisites

- **Docker 20.10+** and **docker-compose** (v1 or v2 plugin)
- That's it. No local Go, Node, or MySQL installation required.

---

## Start the application

```bash
./scripts/start.sh
```

This single command:

1. Copies `.env.example` to `.env` if you don't have one yet
2. Pulls `mysql:8.0` from Docker Hub on first run
3. Builds the backend and frontend images locally
4. Brings up MySQL and waits for the healthcheck
5. Runs database migrations (16 schema files)
6. Seeds reference data (users, categories, offerings, shipping templates)
7. Starts the backend API and the frontend

When startup finishes you can access:

| URL                            | What                       |
|--------------------------------|----------------------------|
| http://localhost:5173          | Frontend web app           |
| http://localhost:8080          | Backend REST API           |
| http://localhost:8080/health   | Health probe (DB + status) |

Stop everything with `Ctrl+C`, or fully tear down with:

```bash
cd deploy/docker && docker-compose down       # keep volumes (data persists)
cd deploy/docker && docker-compose down -v    # also delete volumes (fresh start)
```

### Seed accounts

The seed script creates one user per role. Password for all of them is `password`.

| Username        | Role           |
|-----------------|----------------|
| `regular_user`  | regular_user   |
| `service_agent` | service_agent  |
| `moderator`     | moderator      |
| `admin`         | administrator  |
| `data_operator` | data_operator  |

---

## Run the test suite

```bash
./scripts/run-tests.sh
```

This single command:

1. Builds the test images in parallel
2. Runs **backend tests** (`go test ./...`) against an isolated `service_portal_test` database — covers unit tests, package integration tests, and HTTP-level integration tests
3. Runs **Playwright E2E tests** against a freshly built frontend + backend stack
4. Exits non-zero on any failure (CI-friendly via `set -euo pipefail`)

No setup, no extra installs — Docker handles everything.

---

## Project layout

```
repo/
├── frontend/                    Vue 3 application
│   ├── src/
│   │   ├── views/               Page-level components (one per route)
│   │   ├── components/          Reusable UI (modals, drawers, widgets)
│   │   ├── stores/              Pinia stores (one per domain)
│   │   ├── router/              Vue Router config + auth guard
│   │   └── composables/         Shared composables (toast, etc.)
│   └── tests/e2e/               Playwright spec files
├── backend/
│   ├── cmd/server/              Single binary entry point (server | migrate | seed | test)
│   └── internal/
│       ├── apierr/              Standard JSON error envelope
│       ├── auth/                Session login, register, lockout
│       ├── audit/               Append-only audit log writer
│       ├── address/             Address book CRUD with AES-encrypted lines
│       ├── catalog/             Service categories + offerings
│       ├── config/              Env-loaded configuration
│       ├── crypto/              AES-256-GCM and HMAC-SHA256 helpers
│       ├── db/                  Migrations, seed runner, embedded SQL
│       ├── health/              Health check endpoint
│       ├── ingest/              Source registry + jobs + checkpoints + schema evolution
│       ├── lakehouse/           Bronze/Silver/Gold writers + lineage + lifecycle
│       ├── middleware/          Auth, CSRF, rate-limit, HMAC, RBAC
│       ├── models/              Shared struct definitions
│       ├── moderation/          Term dictionary, screening middleware, queue, freeze engine
│       ├── notification/        Template rendering, dispatch, list, outbox
│       ├── privacy/             Data export ZIP + deletion/anonymization workers
│       ├── profile/             Profile, preferences, favorites, history
│       ├── qa/                  Q&A threads + replies
│       ├── review/              Reviews, images, reports, summary
│       ├── router/              Route registration + cross-package wiring
│       ├── session/             Session store + cookie management
│       ├── shipping/            Region/template CRUD + ETA calculation
│       ├── testutil/            Integration test helpers (DBOrSkip, TruncateTables)
│       └── ticket/              Ticket CRUD + SLA engine + transition matrix
├── deploy/
│   ├── docker/                  Dockerfiles + docker-compose.yml + docker-compose.test.yml
│   └── seed/                    SQL files run by the seed step
├── storage/                     Persistent runtime data (uploads, exports, lakehouse, backups)
├── scripts/
│   ├── start.sh                 ← Start the application
│   └── run-tests.sh             ← Run all tests
└── README.md
```

---

## Configuration

All configuration is read from `.env` at the repo root (auto-created from
`.env.example` on first start). Key variables:

| Variable                         | Purpose                                             | Default                                              |
|----------------------------------|-----------------------------------------------------|------------------------------------------------------|
| `DB_NAME`                        | Application database name                           | `service_portal`                                     |
| `DB_TEST_NAME`                   | Isolated database used by the test runner           | `service_portal_test`                                |
| `DB_USER` / `DB_PASSWORD`        | Application DB credentials                          | `portal_user` / `portalpassword`                     |
| `FIELD_ENCRYPTION_KEY`           | 64-char hex (32-byte) AES-256 key for encrypted PII | dev placeholder; **must be rotated for production**  |
| `APP_ENV`                        | `development` / `production` / `test`               | `development`                                        |
| `FRONTEND_PORT`                  | Host port for the Vite-built frontend               | `5173`                                               |
| `BACKEND_PORT`                   | Host port for the Gin API                           | `8080`                                               |
| `TLS_CERT_FILE` / `TLS_KEY_FILE` | TLS certificate paths (HTTP-only when empty)        | empty                                                |

---

## API surface

All routes live under `/api/v1/`. State-changing requests require both a
session cookie and the `X-CSRF-Token` header obtained from `GET /auth/me`.
Internal data-pipeline routes under `/api/v1/internal/` require an HMAC-signed
request (`X-Key-ID` + `X-Signature: hmac-sha256 <hex>`).

Standard error envelope:

```json
{ "error": { "code": "validation_error", "message": "...", "details": {} } }
```

Cursor pagination shape on list endpoints:

```json
{ "items": [...], "next_cursor": 123 }
```

---

## Security highlights

- **Sessions** — opaque random IDs in HttpOnly cookies; CSRF tokens validated on every mutating request
- **Account lockout** — 5 failed logins in 10 minutes triggers a 15-minute lock; lockout event also fires a notification + audit log entry
- **AES-256-GCM** — phone number and address lines encrypted at rest; nonces generated per-record
- **HMAC-SHA256** — internal ingestion endpoints require signed `METHOD\nPATH\nsha256(body)` payloads with rotatable `hmac_keys`
- **Content moderation** — case-insensitive whole-word matching, prohibited terms blocked at the middleware layer (422 `content_blocked`), borderline content queued and demoted to `pending_moderation`
- **Posting freeze** — first violation = 24h freeze, second = 7-day freeze, attempts during a freeze return 403 `posting_frozen` with `freeze_until`
- **Audit log** — append-only `audit_logs` table; no UPDATE or DELETE endpoints; retention sweep removes rows older than 7 years

---

## Offline-first guarantees

- No outbound network calls anywhere in the codebase
- All MySQL, file storage, and lakehouse layers run on the local Docker network
- Embedded `time/tzdata` ensures timezone math works in minimal containers without system tz files
- Migrations and seed data are bundled into the backend binary via `embed`
