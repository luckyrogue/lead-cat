# Lead Cat — agent guide

Single-purpose **Google Meet meetings-management Telegram Mini App**: **frontend** (React TMA) + **backend** Go monolith (`cmd/server`: Fiber HTTP, Telegram bot, asynq workers).

## Stack

- **Auth:** TMA Telegram (primary — Mini App JWT via `POST /api/auth/tma`); platform OTP/passkey/OAuth for alpha operator bootstrap only (see `docs/AUTH.md`).
- **Data:** Postgres (SoT), Redis (asynq job queues + scheduler leader lock)
- **Deploy:** Dokploy — see `docs/DEPLOY-DOKPLOY.md`
- **Product overview:** `docs/MEETINGS.md`; full spec (ТЗ): `docs/NEW-FEATURES.md`; frontend structure: `frontend/README.md`.

## Before refactoring

Read `docs/ARCHITECTURE.md`. Layers: `domain` ← `application` ← `infrastructure` / `delivery` / `platform` (under `backend/internal/`).

## Engineering principles

Apply everywhere unless a rule below says otherwise.

| Principle              | In practice                                                                                                                                                         |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **KISS**               | Smallest change that solves the task. No speculative abstractions, flags, or "на будущее".                                                                          |
| **DRY**                | One source of truth per concept. Extract only after the second real duplication — not preemptively.                                                                 |
| **SOLID**              | Depend on interfaces at layer boundaries; `domain` stays free of Fiber/pgx/Telegram. One reason to change per type; prefer composition over deep inheritance trees. |
| **Clean Architecture** | Dependencies point inward: `delivery` → `application` → `domain`; `infrastructure` implements ports defined by `application`/`domain`.                              |

### CQRS (backend)

- **Commands** change state (create/update/delete, enqueue notifications, link Telegram). Return error or IDs — not full read models.
- **Queries** read state only. No side effects, no publishing jobs from query handlers.
- Place handlers under `backend/internal/application/command/` and `.../query/` (or `services.go` until split). HTTP handlers in `delivery/http` only map request/response and call one application entry point.
- Shared validation/business rules live in `domain` or small `application` helpers — not duplicated in both command and query.
- Do not merge read and write paths "for convenience" (e.g. GET that inserts audit rows). Use an explicit command if a write is required.

### Logging & observability (backend)

- Logger: **zap** via `internal/platform/observability/log`. Env: `LOG_LEVEL`, `LOG_FORMAT` (`json` in prod, `console` locally).
- **Propagate context**: middleware sets `request_id`; add `workspace_id` / `meeting_id` with `log.With*` helpers. Log with `log.WithContext(ctx, logger)` so fields stay attached.
- **Levels**: `Debug` — dev detail; `Info` — lifecycle (start/stop, job enqueued, auth success); `Warn` — recoverable (retry, degraded); `Error` — failed request/job after handling; avoid `Fatal` except process bootstrap.
- **Structured fields** (`zap.String`, `zap.Error`, …). No secrets, OTP, JWT, `initData`, or raw tokens in logs.
- **Messages**: stable, searchable English snake_case or short phrase + fields for variables (prefer `meeting_create_failed` + `meeting_id`, not interpolated long sentences).
- **Errors**: wrap with `%w` up the stack; log once at the boundary (handler, worker, scheduler tick) with context — avoid logging the same error at every layer.
- **Frontend**: no PII in `console.log` in production paths; use existing toast/UI copy for user-facing failures.

## Cursor rules

| Rule                | Scope             |
| ------------------- | ----------------- |
| `lead-cat-core.mdc` | always            |
| `go-backend.mdc`    | `backend/**/*.go` |
| `frontend-fsd.mdc`  | `frontend/**`     |
| `redis-asynq.mdc`   | queue             |
| `lead-cat-auth.mdc` | auth              |
| `cat-design.mdc`    | UI                |
| `migrations.mdc`    | SQL migrations    |
