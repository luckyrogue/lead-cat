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

## TMA — planned

> **Not yet implemented.** These routes are planned but do not exist in the codebase.

| Method | Path               | Purpose                                            |
| ------ | ------------------ | -------------------------------------------------- |
| `*`    | `/api/tma/admin/*` | In-Mini-App admin setup (replace alpha curl flows) |

Admin spec: [`docs/superpowers/specs/2026-06-05-tma-setup-replacement-design.md`](superpowers/specs/2026-06-05-tma-setup-replacement-design.md).

---

## Appendix — Deprecated: alpha setup (curl)

> These platform endpoints/flows exist only for alpha operator bootstrap and are
> being replaced by in-Mini-App admin (`/api/tma/admin/*`, see
> `docs/superpowers/specs/2026-06-05-tma-setup-replacement-design.md`). Not part
> of the product; slated for removal.

All routes below require a platform JWT (`Authorization: Bearer <platform_jwt>`)
except the public auth endpoints. Operator-only; being retired.

**Auth (public)**

- `GET /api/auth/config`
- `POST /api/auth/email/send-code`, `POST /api/auth/email/verify`
- `POST /api/auth/phone/send-code`, `POST /api/auth/phone/verify`
- `POST /api/auth/passkey/login/begin`, `POST /api/auth/passkey/login/finish`
- `GET /api/auth/oauth/:provider`, `GET /api/auth/oauth/callback` → redirects to `{WEBAPP_URL}/?access_token=…`

**Auth (platform JWT required)**

- `POST /api/auth/passkey/register/begin`, `POST /api/auth/passkey/register/finish`

**Me**

- `GET /api/me`
- ~~`POST /api/me/link-telegram`~~ — **deprecated (410)**; use `POST /api/auth/tma` from the Mini App.

**Workspaces**

- `GET /api/workspaces`, `POST /api/workspaces`
- `GET /api/workspaces/:id`
- `POST /api/workspaces/:id/chat/link`, `GET /api/workspaces/:id/chat/status`

**Integrations** (Google service account — required before TMA can create meetings)

- `GET /api/workspaces/:id/integrations`
- `PATCH /api/workspaces/:id/integrations`
- `POST /api/workspaces/:id/integrations/verify`

**Members**

- `GET /api/workspaces/:id/members`
- `POST /api/workspaces/:id/members`
- `DELETE /api/workspaces/:id/members/:memberId`
- `POST /api/workspaces/:id/members/sync-chat`
- `PATCH /api/workspaces/:id/members/:username/vcs`

**Scenarios** (legacy — not used by the meetings Mini App)

- `GET /api/workspaces/:id/scenarios`
- `POST /api/workspaces/:id/scenarios`
- `GET /api/workspaces/:id/scenarios/:sid`
- `PATCH /api/workspaces/:id/scenarios/:sid`
- `DELETE /api/workspaces/:id/scenarios/:sid`
- `POST /api/workspaces/:id/scenarios/:sid/run`
- `GET /api/workspaces/:id/scenarios/:sid/runs`

**Workspace meetings** (superseded by TMA for all Mini App clients)

- `GET /api/workspaces/:id/employees`
- `GET /api/workspaces/:id/meetings`
- `POST /api/workspaces/:id/meetings`
- `GET /api/workspaces/:id/meetings/:mid`
- `DELETE /api/workspaces/:id/meetings/:mid`
- `POST /api/workspaces/:id/meetings/conflicts`
- `POST /api/workspaces/:id/meetings/free-slots`
