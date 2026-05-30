# Lead Cat — agent guide

SaaS Telegram notify bot: **frontend** (React, lite FSD) + **backend** Go monolith (`cmd/server`: Fiber, bot, asynq, scenario engine).

## Stack

- **Auth:** Native JWT (email/phone OTP, passkey, GitHub/GitLab OAuth); TMA SDK for link Telegram only
- **Data:** Postgres (SoT), Redis (asynq queues)
- **Deploy:** Dokploy — see `docs/DEPLOY-DOKPLOY.md`
- **Meetings (in dev):** Google Meet management Mini App — frontend TMA built on mocks (`frontend/src/features/tma`), backend planned per `docs/NEW-FEATURES.md`. Summary: `docs/MEETINGS.md`.

## Before refactoring

Read `docs/ARCHITECTURE.md`. Layers: `domain` ← `application` ← `infrastructure` / `delivery` / `platform` (under `backend/internal/`).

## Engineering principles

Apply everywhere unless a rule below says otherwise.

| Principle              | In practice                                                                                                                                                         |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **KISS**               | Smallest change that solves the task. No speculative abstractions, flags, or “на будущее”.                                                                          |
| **DRY**                | One source of truth per concept. Extract only after the second real duplication — not preemptively.                                                                 |
| **SOLID**              | Depend on interfaces at layer boundaries; `domain` stays free of Fiber/pgx/Telegram. One reason to change per type; prefer composition over deep inheritance trees. |
| **Clean Architecture** | Dependencies point inward: `delivery` → `application` → `domain`; `infrastructure` implements ports defined by `application`/`domain`.                              |

### CQRS (backend)

- **Commands** change state (create/update/delete, enqueue run, link Telegram). Return error or IDs — not full read models.
- **Queries** read state only. No side effects, no publishing jobs from query handlers.
- Place handlers under `backend/internal/application/command/` and `.../query/` (or `services.go` until split). HTTP handlers in `delivery/http` only map request/response and call one application entry point.
- Shared validation/business rules live in `domain` or small `application` helpers — not duplicated in both command and query.
- Do not merge read and write paths “for convenience” (e.g. GET that inserts audit rows). Use an explicit command if a write is required.

### Logging & observability (backend)

- Logger: **zap** via `internal/platform/observability/log`. Env: `LOG_LEVEL`, `LOG_FORMAT` (`json` in prod, `console` locally).
- **Propagate context**: middleware sets `request_id`; add `workspace_id` / `scenario_id` / `run_id` with `log.With*` helpers. Log with `log.WithContext(ctx, logger)` so fields stay attached.
- **Levels**: `Debug` — dev detail; `Info` — lifecycle (start/stop, job enqueued, auth success); `Warn` — recoverable (retry, degraded); `Error` — failed request/job after handling; avoid `Fatal` except process bootstrap.
- **Structured fields** (`zap.String`, `zap.Error`, …). No secrets, OTP, JWT, `initData`, or raw tokens in logs.
- **Messages**: stable, searchable English snake_case or short phrase + fields for variables (prefer `scenario_run_failed` + `scenario_id`, not interpolated long sentences).
- **Errors**: wrap with `%w` up the stack; log once at the boundary (handler, worker, scheduler tick) with context — avoid logging the same error at every layer.
- **Frontend**: no PII in `console.log` in production paths; use existing toast/UI copy for user-facing failures.

## Cursor rules

| Rule                | Scope             |
| ------------------- | ----------------- |
| `lead-cat-core.mdc` | always            |
| `go-backend.mdc`    | `backend/**/*.go` |
| `frontend-fsd.mdc`  | `frontend/**`     |
| `scenarios.mdc`     | scenario code     |
| `redis-asynq.mdc`   | queue             |
| `lead-cat-auth.mdc` | auth              |
| `cat-design.mdc`    | UI                |
| `migrations.mdc`    | SQL migrations    |

## Status

Implementation checklist: `PLAN.md`
