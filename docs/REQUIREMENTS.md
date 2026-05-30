# Requirements — Lead Cat

Multi-tenant SaaS Telegram notify bot: **Go monolith** backend (`cmd/server`: Fiber HTTP API, Telegram bot, asynq workers, scenario engine) + **React Mini App** frontend (lite FSD). Postgres is the source of truth; Redis backs asynq queues.

This document covers both **prerequisites/dependencies** (what you need to build & run) and **functional / non-functional requirements** (what the system must do) for backend and frontend.

---

## 1. Prerequisites (build & run)

| Tool / Service   | Version | Notes                                                               |
| ---------------- | ------- | ------------------------------------------------------------------- |
| Go               | 1.26.x  | `backend/go.mod` pins `go 1.26.3`; toolchain auto-fetches if newer. |
| Node.js          | 22.x    | Frontend build (Vite). Dockerfile uses `node:22-alpine`.            |
| pnpm             | 9.x     | Frontend package manager (`frontend/pnpm-lock.yaml`).               |
| Docker + Compose | recent  | Local Postgres + Redis via `deploy/docker-compose.yml`.             |
| PostgreSQL       | 18      | `postgres:18-alpine` in both local compose and CI smoke.            |
| Redis            | 8       | `redis:8-alpine` in both local compose and CI smoke.                |
| golangci-lint    | 2.x     | `make lint` / `make fmt` (config in `config/.golangci.yml`).        |
| air (optional)   | latest  | `make backend-watch` hot reload.                                    |

**One-shot local setup:** `make setup` → edit `.env` → `make migrate` → `make dev`.
Default ports: API `:8080`, frontend `:3000`, Postgres `5432`, Redis `6379`.

---

## 2. Backend

### 2.1 Dependencies

- **HTTP:** `gofiber/fiber/v2`
- **Telegram:** `go-telegram/bot`
- **Auth:** `go-webauthn/webauthn` (passkeys), JWT, GitHub/GitLab OAuth
- **Data:** PostgreSQL (`DATABASE_URL`), goose migrations (`pressly/goose` via `cmd/migrate`)
- **Queue:** Redis + asynq (`REDIS_URL`)
- **Layout:** clean architecture — `domain` ← `application` ← `infrastructure` / `delivery` / `platform` (under `backend/internal/`)

### 2.2 Configuration (environment)

| Variable                               | Required | Purpose                                                                  |
| -------------------------------------- | -------- | ------------------------------------------------------------------------ |
| `BOT_TOKEN`                            | prod     | Telegram bot token (@BotFather). May be empty when `AUTH_DEV_MODE=true`. |
| `DATABASE_URL`                         | yes      | Postgres DSN (source of truth).                                          |
| `REDIS_URL`                            | yes      | Redis DSN for asynq queues.                                              |
| `MASTER_ENCRYPTION_KEY`                | yes      | ≥32 chars; encrypts per-workspace secrets (e.g. VCS tokens).             |
| `JWT_SECRET`                           | yes      | ≥16 chars; signs session JWTs.                                           |
| `JWT_ISSUER`, `JWT_TTL_HOURS`          | no       | JWT issuer / lifetime (default issuer `lead-cat`, 168h).                 |
| `AUTH_DEV_MODE`                        | dev      | `true` ⇒ any bearer token maps to a dev user; Telegram polling disabled. |
| `AUTH_DEV_USER_SUB`, `AUTH_DEV_EMAIL`  | dev      | Identity used in dev mode.                                               |
| `AUTH_OTP_LOG`                         | dev      | Log OTP codes instead of sending.                                        |
| `WEBAUTHN_RP_ID`, `WEBAUTHN_RP_ORIGIN` | passkeys | WebAuthn relying-party config.                                           |
| `GITHUB_OAUTH_*`, `GITLAB_OAUTH_*`     | optional | OAuth login + VCS integration.                                           |
| `HTTP_ADDR`                            | no       | Listen address (default `:8080`).                                        |
| `WEBAPP_URL`, `CORS_ALLOWED_ORIGINS`   | yes      | Frontend origin(s) for links & CORS.                                     |
| `LOG_LEVEL`, `LOG_FORMAT`              | no       | Structured logging.                                                      |
| `AUTO_MIGRATE`                         | no       | `true` ⇒ run migrations on boot.                                         |

### 2.3 Functional requirements

- **F-B1 Multi-tenancy:** one platform `BOT_TOKEN`; each workspace's config lives in Postgres. Strict per-workspace access control (owner/member); cross-workspace access returns `403` (IDOR-protected).
- **F-B2 Native auth:** email/phone **OTP**, **passkey** (WebAuthn), **GitHub/GitLab OAuth**; issues session JWT. Telegram is used for _linking_ an account/chat only (TMA SDK), not as the primary login.
- **F-B3 Scenario engine:** n8n-like definitions (`nodes[]`, `edges[]`); manual and cron triggers; runs enqueued to asynq and executed by the worker in `cmd/server`; run status tracked (`pending`/`success`/`failed`).
- **F-B4 Notifications:** scenario actions deliver messages to a workspace's linked Telegram chat (`NotifyChatID`).
- **F-B5 VCS integration:** GitHub/GitLab commits report; per-workspace VCS token stored encrypted; `integrations/verify` endpoint.
- **F-B6 Migrations:** schema managed by goose; no silent auto-migrate in prod unless `AUTO_MIGRATE=true`.

### 2.4 Non-functional requirements

- **N-B1 Source of truth:** Postgres authoritative; Redis is ephemeral (queues only).
- **N-B2 Security:** secrets encrypted at rest with `MASTER_ENCRYPTION_KEY`; JWT-signed sessions; workspace-scoped authorization on every route.
- **N-B3 Observability:** structured logs, Prometheus metrics, `GET /api/health` reporting `postgres`/`redis` status.
- **N-B4 Quality gates:** `go vet` + unit tests green; **coverage ≥ 50%** for `internal/delivery/http/middleware` and `internal/domain/scenario` (`make coverage`); E2E smoke (`make smoke`, `go test -tags=smoke`).
- **N-B5 Build:** static binary via multi-stage Docker (`CGO_ENABLED=0`); migrations bundled into the image.

---

## 3. Frontend

### 3.1 Dependencies

- **Framework:** React 19, Vite 6, TypeScript 5
- **Routing/data:** `@tanstack/react-router` (generated route tree), `@tanstack/react-query`
- **UI:** shadcn/ui + Radix, Tailwind CSS v4 (`@tailwindcss/vite`), `lucide-react`, cat design tokens
- **Scenario editor:** `@xyflow/react`
- **Auth:** `@simplewebauthn/browser` (passkeys), `axios` (Bearer JWT), `zod` (validation)
- **Layout:** lite Feature-Sliced Design — `app` → `routes` → `widgets` → `features` → `entities` → `shared`

### 3.2 Configuration (environment)

| Variable             | Purpose                                            |
| -------------------- | -------------------------------------------------- |
| `VITE_AUTH_DEV_MODE` | `true` ⇒ skip real auth locally (`/login` bypass). |

API base + Telegram link behavior are derived from backend `WEBAPP_URL` / build args (`VITE_AUTH_DEV_MODE` baked at build time in Docker).

### 3.3 Functional requirements

- **F-F1 Auth flows:** `/login` page supporting OTP, passkey, and GitHub/GitLab OAuth; JWT stored in `localStorage`, attached as `axios` Bearer; link-Telegram flow via TMA SDK.
- **F-F2 Workspace UI:** create/select workspaces; respect backend ACL (no cross-workspace data).
- **F-F3 Scenario builder:** visual node/edge editor (`@xyflow/react`); node-type presets must stay aligned with backend (`frontend/src/shared/presets.ts` ↔ `backend/.../scenario/presets.go`).
- **F-F4 Cat design:** mandatory cat theme tokens (`frontend/src/shared/theme/cat-tokens.css`); part of product identity.

### 3.4 Non-functional requirements

- **N-F1 Type safety:** `pnpm typecheck` (`tsc --noEmit`, strict) must pass; config extends `config/tsconfig.base.json`.
- **N-F2 Formatting:** Prettier via `config/prettier.config.mjs` (Tailwind class sorting); `pnpm run format:check` clean.
- **N-F3 Routing:** route tree is generated (`src/routeTree.gen.ts`) — never hand-edited.
- **N-F4 Build:** `pnpm build` (Vite) produces static assets served by the backend (`STATIC_DIR`).

---

## 4. Shared infrastructure

- **Services:** PostgreSQL + Redis (local: `deploy/docker-compose.yml`, project name `lead-cat`).
- **Container:** single multi-stage `deploy/Dockerfile` (frontend build → Go build → alpine runtime); build context is the repo root.
- **CI/CD:** GitHub Actions (`.github/workflows/`): build+test, docker build/push to GHCR, smoke (main only), deploy via Dokploy webhook.
- **Tooling config:** centralized in `config/` (Prettier, tsconfig base, EditorConfig, golangci-lint) — single source of truth.

---

## 5. Meetings (Google Meet) — in development

Additive feature; full spec in [NEW-FEATURES.md](NEW-FEATURES.md) (ТЗ), summary in [MEETINGS.md](MEETINGS.md).

- **Frontend (done, mock-backed):** Telegram Mini App (`frontend/src/features/tma`) — tabs home/meetings/checker/auto/profile; create-meeting wizard; free-slot checker; ru/kk/en i18n; cat design.
- **Backend (planned):** Google Calendar/Meet via one corporate **service account**; users bound by **Telegram ID + corporate email** (auto-register on `/start`); employee directory from an **embedded CSV** at deploy; meeting CRUD, recurrence, time-conflict detection, reminders. Base TZ **UTC+5 (Almaty)**. Roles: User / Main Administrator.
- **New prerequisites (planned, when backend lands):** Google service-account credentials + employees CSV — to be added to `deploy/.env.example` and §1–2 above.

---

See also: [ARCHITECTURE.md](ARCHITECTURE.md), [AUTH.md](AUTH.md), [SCENARIOS.md](SCENARIOS.md), [MEETINGS.md](MEETINGS.md), [LOCAL_DEV.md](LOCAL_DEV.md), [DEPLOY-DOKPLOY.md](DEPLOY-DOKPLOY.md).
