# Lead Cat — agent guide

SaaS Telegram notify bot: **frontend** (React, lite FSD) + **backend** Go monolith (`cmd/server`: Fiber, bot, asynq, scenario engine).

## Stack

- **Auth:** Native JWT (email/phone OTP, passkey, GitHub/GitLab OAuth); TMA SDK for link Telegram only
- **Data:** Postgres (SoT), Redis (asynq queues)
- **Deploy:** Dokploy — see `docs/DEPLOY-DOKPLOY.md`

## Before refactoring

Read `docs/ARCHITECTURE.md`. Layers: `domain` ← `application` ← `infrastructure` / `delivery` / `platform` (under `backend/internal/`).

## Cursor rules

| Rule | Scope |
|------|--------|
| `lead-cat-core.mdc` | always |
| `go-backend.mdc` | `backend/**/*.go` |
| `frontend-fsd.mdc` | `frontend/**` |
| `scenarios.mdc` | scenario code |
| `redis-asynq.mdc` | queue |
| `lead-cat-auth.mdc` | auth |
| `cat-design.mdc` | UI |
| `migrations.mdc` | SQL migrations |

## Status

Implementation checklist: `PLAN.md`
