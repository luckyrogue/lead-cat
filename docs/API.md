# API Reference

Base path: `/api`. Error shape: `{ "error": "code", "message": "…" }`.

The machine-readable source of truth is `backend/openapi/openapi.json`, served at `GET /openapi.json`
and mirrored to the generated TypeScript client at `frontend/src/shared/api/generated/schema.ts`.

---

## Public

| Method | Path            | Notes                                                           |
| ------ | --------------- | --------------------------------------------------------------- |
| `GET`  | `/api/health`   | Postgres, Redis, bot liveness, version                          |
| `GET`  | `/metrics`      | Prometheus text (optional `METRICS_TOKEN`)                      |
| `GET`  | `/openapi.json` | OpenAPI 3.1 spec (embedded from `backend/openapi/openapi.json`) |

---

## Mini App auth

Exchange a Telegram WebApp `initData` string for a Mini App JWT.

```
POST /api/auth/miniapp
Body: { "init_data": "<Telegram WebApp initData>" }
→    { "token": "…", "user": { … } }
```

Returns `401 { "code": "not_registered" }` when no `bot_users` row exists for the
Telegram user. Registration is **not** HTTP — the user must start the bot with `/start`.

Subsequent calls to `/api/miniapp/*` require `Authorization: Bearer <miniapp_jwt>`.
Token type `tok_typ:miniapp`, TTL 24 h.

---

## Mini App — meetings

All routes require `Authorization: Bearer <miniapp_jwt>`.

| Method   | Path                                              | Purpose                                                               |
| -------- | ------------------------------------------------- | --------------------------------------------------------------------- |
| `GET`    | `/api/miniapp/me`                                 | Current bot-user identity (name, email, role)                         |
| `GET`    | `/api/miniapp/meetings?scope=upcoming\|past\|all` | Authenticated user's own meetings                                     |
| `GET`    | `/api/miniapp/schedule?email=&scope=`             | Read a colleague's schedule (view-only)                               |
| `GET`    | `/api/miniapp/employees?q=`                       | Employee directory search / autocomplete                              |
| `POST`   | `/api/miniapp/free-slots`                         | Common free-time checker across participants                          |
| `POST`   | `/api/miniapp/meetings`                           | Create a meeting (recurring supported via `recurrence_until` + `recurrence_days`) |
| `PATCH`  | `/api/miniapp/meetings/:id?scope=this\|whole`     | Edit a meeting (organizer-only, 403); `scope` default `this`          |
| `DELETE` | `/api/miniapp/meetings/:id?scope=this\|whole`     | Cancel a meeting (organizer-only, 403); `scope` default `this`        |
| `POST`   | `/api/miniapp/conflicts`                          | Conflict check (single or expanded series); response: `occurrences[]` |

Recurrence kinds: `once`, `daily`, `weekly`, `custom` (with `recurrence_days: [1..7]`, Mon=1..Sun=7), `monthly`. Non-once requires `recurrence_until` (YYYY-MM-DD).

Write-path error codes: `meetings_not_configured` (Google integration missing), `forbidden` (not the organizer / not admin), `validation_failed` (bad input, incl. bad recurrence / scope).

---

### Mini App — admin

All routes require `Authorization: Bearer <miniapp_jwt>` AND `bot_users.role == "admin"`.

| Method   | Path                                          | Purpose                                        |
| -------- | --------------------------------------------- | ---------------------------------------------- |
| `GET`    | `/api/miniapp/admin/workspace`                | Workspace status (auto-create on first call)   |
| `POST`   | `/api/miniapp/admin/workspace`                | Idempotent ensure-workspace                    |
| `GET`    | `/api/miniapp/admin/integrations`             | Google integration view (no secrets)           |
| `PATCH`  | `/api/miniapp/admin/integrations`             | Set SA JSON / subject / calendar id / meet / tz |
| `POST`   | `/api/miniapp/admin/integrations/verify`      | Real Google verify (parse → impersonate → Calendars.Get) |
| `GET`    | `/api/miniapp/admin/chat/status`              | Chat-link status                               |
| `POST`   | `/api/miniapp/admin/chat/link`                | Link Telegram chat                             |
| `GET`    | `/api/miniapp/admin/members`                  | List workspace members                         |
| `POST`   | `/api/miniapp/admin/members/sync-chat`        | Sync members from linked chat                  |
| `GET`    | `/api/miniapp/admin/audit`                    | Audit log (filters: action, actor, limit≤200)  |

Error codes: `forbidden`, `unauthorized`, `validation_failed`, `workspace_not_found`, `google_sa_invalid`, `google_subject_invalid`, `google_calendar_not_accessible`, `google_api_disabled`, `google_not_configured`.

Mini App UI: **Profile → Admin → Настройка workspace** (`/profile/admin/setup`).

---

## Appendix — Retired platform API (410 Gone)

> Platform JWT bootstrap (`/api/auth/email/*`, passkey, OAuth, `/api/workspaces/*`) returns **410 Gone** with `Deprecation: true`. Use Mini App admin routes above and `POST /api/auth/miniapp` for all operator setup.
