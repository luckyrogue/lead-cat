# Architecture

## Infra (Dokploy)

- **Postgres** — source of truth
- **Redis** — asynq scenario job queue + scheduler leader lock
- **lead-cat** — single Go binary (API + Mini App static + bot + asynq worker)

## Go layers

```
delivery/http     → Fiber, JWT middleware, static miniapp
application/      → command, query, scenario RunScenario
domain/           → workspace, scenario, user entities
infrastructure/   → postgres, asynq, telegram, vcs, crypto
platform/         → scenario_scheduler, scenario_executor, log
```

## Auth flow

1. User logs in on `/login` (email/phone OTP, passkey, or GitHub/GitLab OAuth).
2. API issues HS256 JWT; Mini App sends `Authorization: Bearer <token>`.
3. Optional: link Telegram via `POST /api/me/link-telegram` + `X-Telegram-Init-Data`.

See [AUTH.md](AUTH.md).

## Scenario execution

```
cron scheduler → enqueue (Redis/asynq) → worker → executor → nodes → Telegram/VCS
```

Run state persisted in `scenario_runs` / `scenario_run_steps`.

## VCS

- **Login OAuth** (GitHub/GitLab for sign-in) ≠ **workspace VCS token** (encrypted in Postgres for commits API).

## Meetings (Google Meet) — in development

Additive Telegram Mini App for scheduling Google Meet meetings. Frontend (`frontend/src/features/tma`, `shared/tma`) is implemented on **mock data**; backend is **planned** per [NEW-FEATURES.md](NEW-FEATURES.md):

```
frontend TMA (tabs: home/meetings/checker/auto/profile)  ──▶  [planned] meetings API
                                                                │
[planned] domain/meeting + infrastructure/google (Calendar/Meet, service account)
[planned] employee directory seeded from embedded CSV at deploy
```

Reuses the same monolith, Postgres, and Telegram wiring. See [MEETINGS.md](MEETINGS.md).
