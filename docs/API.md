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

## Web auth

Cookie-based sessions for the web app (see `docs/AUTH.md`).

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `GET` | `/api/auth/web/:provider/start` | Start OAuth (Google, Microsoft) |
| `GET` | `/api/auth/web/:provider/callback` | OAuth callback; sets session cookie |
| `POST` | `/api/auth/web/magic/request` | Request email magic link (`email`, optional `language`: `ru`\|`en`\|`kk`) |
| `GET` | `/api/auth/web/magic/verify` | Verify magic link |
| `POST` | `/api/auth/web/logout` | Revoke session |
| `GET` | `/api/auth/web/me` | Current user profile |

---

## Organizations (web)

Requires `lc_session` cookie. Org-scoped routes also require org membership (`RequireOrgMember`).

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `POST` | `/api/orgs` | Create organization |
| `GET` | `/api/orgs` | List my organizations |
| `GET` | `/api/orgs/:id/meetings` | List meetings (owner sees all; member sees own) |
| `POST` | `/api/orgs/:id/meetings` | Create meeting |
| `GET` | `/api/orgs/:id/meetings/:mid` | Get meeting |
| `PATCH` | `/api/orgs/:id/meetings/:mid?scope=this\|whole` | Update meeting / series |
| `DELETE` | `/api/orgs/:id/meetings/:mid?scope=this\|whole` | Cancel meeting / series |
| `POST` | `/api/orgs/:id/meetings/:mid/participants` | Add participant |
| `DELETE` | `/api/orgs/:id/meetings/:mid/participants?email=` | Remove participant |
| `GET` | `/api/orgs/:id/members` | List members |
| `PATCH` | `/api/orgs/:id/members/:uid/role` | Change role (admin) |
| `DELETE` | `/api/orgs/:id/members/:uid` | Remove member (admin) |
| `GET` | `/api/orgs/:id/invites` | List invites (admin) |
| `POST` | `/api/orgs/:id/invites` | Invite by email (admin) |
| `DELETE` | `/api/orgs/:id/invites/:iid` | Revoke invite (admin) |
| `GET` | `/api/orgs/:id/join-requests` | List pending join requests (admin) |
| `POST` | `/api/orgs/:id/join-requests/:rid/accept` | Accept join request (admin) |
| `POST` | `/api/orgs/:id/join-requests/:rid/decline` | Decline join request (admin) |

### Web — my invites & join requests

Requires `lc_session` cookie.

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `GET` | `/api/auth/web/me/invites` | List invites for current user |
| `POST` | `/api/auth/web/me/invites/:iid/accept` | Accept invite |
| `POST` | `/api/auth/web/me/invites/:iid/decline` | Decline invite |
| `GET` | `/api/auth/web/me/join-requests` | List my join requests |
| `POST` | `/api/auth/web/me/join-requests` | Request to join org by slug (`{ slug }`) |

---

## Booking (admin)

Requires `lc_session` cookie. Create also requires `X-Org-Id` header.

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `GET` | `/api/booking/event-types` | List my event types → `{ event_types: [...] }` |
| `POST` | `/api/booking/event-types` | Create event type |
| `PATCH` | `/api/booking/event-types/:id` | Update event type (empty 200) |
| `DELETE` | `/api/booking/event-types/:id` | Delete event type |

---

## Public booking

No auth. Rate-limited.

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `GET` | `/api/book/:slug` | Event + available slots |
| `POST` | `/api/book/:slug` | Submit booking → `{ meet_link, start, end }` |

---

## Calendar connections

Web routes require `lc_session`; mini-app routes require Bearer JWT.

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `POST` | `/api/calendar/connect/:provider/start` | Start OAuth → `{ auth_url }` |
| `GET` | `/api/calendar/connections` | List connections |
| `DELETE` | `/api/calendar/connections/:provider` | Disconnect |
| `POST` | `/api/miniapp/calendar/connect/:provider/start` | Mini-app OAuth start |
| `GET` | `/api/miniapp/calendar/connections` | Mini-app list |
| `DELETE` | `/api/miniapp/calendar/connections/:provider` | Mini-app disconnect |

---

## Mini App — meetings

All routes require `Authorization: Bearer <miniapp_jwt>`.

| Method   | Path                                              | Purpose                                                               |
| -------- | ------------------------------------------------- | --------------------------------------------------------------------- |
| `GET`    | `/api/miniapp/me`                                 | Current bot-user identity (name, email, role)                         |
| `GET`    | `/api/miniapp/settings`                           | Current user reminder settings                                        |
| `PATCH`  | `/api/miniapp/settings`                           | Update reminder minutes (whitelist enforced)                          |
| `GET`    | `/api/miniapp/meetings?scope=upcoming\|past\|all` | Authenticated user's own meetings                                     |
| `GET`    | `/api/miniapp/schedule?email=&scope=`             | Read a colleague's schedule (view-only)                               |
| `GET`    | `/api/miniapp/employees?q=`                       | Employee directory search / autocomplete                              |
| `POST`   | `/api/miniapp/free-slots`                         | Common free-time checker across participants                          |
| `POST`   | `/api/miniapp/meetings`                           | Create a meeting (recurring supported via `recurrence_until` + `recurrence_days`) |
| `PATCH`  | `/api/miniapp/meetings/:id?scope=this\|whole`     | Edit a meeting (organizer-only, 403); `scope` default `this`          |
| `DELETE` | `/api/miniapp/meetings/:id?scope=this\|whole`     | Cancel a meeting (organizer-only, 403); `scope` default `this`        |
| `POST`   | `/api/miniapp/conflicts`                          | Conflict check (single or expanded series); `participants` must be org employee emails — unknown emails → `400 unknown_participant`; response: `occurrences[]` |
| `GET`    | `/api/miniapp/meetings/:id`                       | Single meeting detail (JWT)                                                                   |

Recurrence kinds: `once`, `daily`, `weekly`, `custom` (with `recurrence_days: [1..7]`, Mon=1..Sun=7), `monthly`. Non-once requires `recurrence_until` (YYYY-MM-DD).

Write-path error codes: `meetings_not_configured` (Google integration missing), `forbidden` (not organizer / org owner), `unknown_participant` (conflicts/free-slots with non-employee email), `validation_failed` (bad input, incl. bad recurrence / scope).

---

### Mini App — admin

All routes require `Authorization: Bearer <miniapp_jwt>` AND `bot_users.role == "admin"`.

| Method   | Path                                          | Purpose                                        |
| -------- | --------------------------------------------- | ---------------------------------------------- |
| `GET`    | `/api/miniapp/admin/organization`             | Organization status (auto-create on first call) |
| `POST`   | `/api/miniapp/admin/organization`             | Idempotent ensure default organization         |
| `GET`    | `/api/miniapp/admin/workspace`                | **Deprecated** alias of `/organization`        |
| `POST`   | `/api/miniapp/admin/workspace`                | **Deprecated** alias of `/organization`        |
| `GET`    | `/api/miniapp/admin/integrations`             | Google integration view (no secrets)           |
| `PATCH`  | `/api/miniapp/admin/integrations`             | Set SA JSON / subject / calendar id / meet / tz |
| `POST`   | `/api/miniapp/admin/integrations/verify`      | Real Google verify (parse → impersonate → Calendars.Get) |
| `GET`    | `/api/miniapp/admin/chat/status`              | Chat-link status                               |
| `POST`   | `/api/miniapp/admin/chat/link`                | Link Telegram chat                             |
| `GET`    | `/api/miniapp/admin/members`                  | List organization members                      |
| `POST`   | `/api/miniapp/admin/members/sync-chat`        | Sync members from linked chat                  |
| `GET`    | `/api/miniapp/admin/audit`                    | Audit log (filters: action, actor, limit≤200)  |

Error codes: `forbidden`, `unauthorized`, `validation_failed`, `organization_not_found`, `google_sa_invalid`, `google_subject_invalid`, `google_calendar_not_accessible`, `google_api_disabled`, `google_not_configured`.

Mini App UI: **Profile → Admin → setup** (`/profile/admin/setup`).

---

## Appendix — Retired platform API (410 Gone)

> Legacy platform JWT bootstrap (`/api/auth/email/*`, passkey, old OAuth) and `/api/workspaces/*` return **410 Gone** with `Deprecation: true`. Active paths: `/api/auth/web/*`, `/api/orgs/*`, `/api/miniapp/admin/organization`, and `POST /api/auth/miniapp`.
