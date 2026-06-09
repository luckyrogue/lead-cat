# API Reference

Base path: `/api`. Error shape: `{ "error": "code", "message": "…" }`.

The machine-readable source of truth is `docs/openapi.json`, served at `GET /openapi.json`
and mirrored to the generated TypeScript client at `frontend/src/shared/api/generated/`.

---

## Public

| Method | Path            | Notes                                                           |
| ------ | --------------- | --------------------------------------------------------------- |
| `GET`  | `/api/health`   | Postgres, Redis, bot liveness, version                          |
| `GET`  | `/metrics`      | Prometheus text (optional `METRICS_TOKEN`)                      |
| `GET`  | `/openapi.json` | OpenAPI 3.1 spec (embedded from `backend/openapi/openapi.json`) |

---

## TMA auth

Exchange a Telegram WebApp `initData` string for a TMA JWT.

```
POST /api/auth/tma
Body: { "init_data": "<Telegram WebApp initData>" }
→    { "token": "…", "user": { … } }
```

Returns `401 { "code": "not_registered" }` when no `bot_users` row exists for the
Telegram user. Registration is **not** HTTP — the user must start the bot with `/start`.

Subsequent calls to `/api/tma/*` require `Authorization: Bearer <tma_jwt>`.
Token type `typ:tma`, TTL 24 h.

---

## TMA — present

All routes require `Authorization: Bearer <tma_jwt>`.

| Method   | Path                                          | Purpose                                                               |
| -------- | --------------------------------------------- | --------------------------------------------------------------------- |
| `GET`    | `/api/tma/me`                                 | Current bot-user identity (name, email, role)                         |
| `GET`    | `/api/tma/meetings?scope=upcoming\|past\|all` | Authenticated user's own meetings                                     |
| `GET`    | `/api/tma/schedule?email=&scope=`             | Read a colleague's schedule (view-only)                               |
| `GET`    | `/api/tma/employees?q=`                       | Employee directory search / autocomplete                              |
| `POST`   | `/api/tma/free-slots`                         | Common free-time checker across participants                          |
| `POST`   | `/api/tma/meetings`                           | Create a meeting (recurring supported via `recurrence_until` + `recurrence_days`) |
| `PATCH`  | `/api/tma/meetings/:id?scope=this\|whole`     | Edit a meeting (organizer-only, 403); `scope` default `this`          |
| `DELETE` | `/api/tma/meetings/:id?scope=this\|whole`     | Cancel a meeting (organizer-only, 403); `scope` default `this`        |
| `POST`   | `/api/tma/conflicts`                          | Conflict check (single or expanded series); response: `occurrences[]` |

Recurrence kinds: `once`, `daily`, `weekly`, `custom` (with `recurrence_days: [1..7]`, Mon=1..Sun=7), `monthly`. Non-once requires `recurrence_until` (YYYY-MM-DD).

Write-path error codes: `meetings_not_configured` (Google integration missing), `forbidden` (not the organizer / not admin), `validation_failed` (bad input, incl. bad recurrence / scope).

---

### TMA — admin (slice D)

All routes require `Authorization: Bearer <tma_jwt>` AND `bot_users.role == "admin"`.

| Method   | Path                                          | Purpose                                        |
| -------- | --------------------------------------------- | ---------------------------------------------- |
| `GET`    | `/api/tma/admin/workspace`                    | Workspace status (auto-create on first call)   |
| `POST`   | `/api/tma/admin/workspace`                    | Idempotent ensure-workspace                    |
| `GET`    | `/api/tma/admin/integrations`                 | Google integration view (no secrets)           |
| `PATCH`  | `/api/tma/admin/integrations`                 | Set SA JSON / subject / calendar id / meet / tz |
| `POST`   | `/api/tma/admin/integrations/verify`          | Real Google verify (parse → impersonate → Calendars.Get) |
| `GET`    | `/api/tma/admin/chat/status`                  | Chat-link status                               |
| `POST`   | `/api/tma/admin/chat/link`                    | Link Telegram chat                             |
| `GET`    | `/api/tma/admin/members`                      | List workspace members                         |
| `POST`   | `/api/tma/admin/members/sync-chat`            | Sync members from linked chat                  |
| `GET`    | `/api/tma/admin/scenarios`                    | List scenarios                                 |
| `PATCH`  | `/api/tma/admin/scenarios/:id`                | Toggle `enabled` (only)                        |
| `POST`   | `/api/tma/admin/scenarios/:id/run`            | Manual run                                     |
| `GET`    | `/api/tma/admin/scenarios/:id/runs`           | Last 30 runs                                   |
| `GET`    | `/api/tma/admin/audit`                        | Audit log (filters: action, actor, limit≤200)  |

Error codes: `forbidden`, `unauthorized`, `validation_failed`, `workspace_not_found`, `google_sa_invalid`, `google_subject_invalid`, `google_calendar_not_accessible`, `google_api_disabled`, `google_not_configured`.

Mini App UI: **Profile → Admin → Настройка workspace** (`/profile/admin/setup`).

---

## Appendix — Retired platform API (410 Gone)

> Platform JWT bootstrap (`/api/auth/email/*`, passkey, OAuth, `/api/workspaces/*`) returns **410 Gone** with `Deprecation: true`. Use TMA admin routes above and `POST /api/auth/tma` for all operator setup.

> `PATCH /api/workspaces/:id/integrations` and `POST /api/workspaces/:id/chat/link` are superseded by `/api/tma/admin/*` (slice D). Kept for scripted operator use; will be removed after first beta release.
