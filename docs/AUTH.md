# Authentication

Lead Cat has **two active user-facing auth surfaces**:

1. **Telegram Mini App** — `POST /api/auth/miniapp` → Mini App JWT → `/api/miniapp/*`
2. **Web app** — cookie sessions via `/api/auth/web/*` → `/api/orgs/*`

**Retired (410 Gone):** legacy platform bootstrap (`/api/auth/email/*`, passkey, OAuth, `/api/workspaces/*`). Operator setup for the single default org still runs through **Mini App admin** inside Telegram (`/api/miniapp/admin/*`).

> **Glossary:** **miniapp** — Telegram Mini App API layer (formerly `tma` in code and paths). **organization** — tenant/workspace row in `organizations` (TMA interim uses one default org; web supports multi-org).

---

## Overview

```
Telegram Mini App                    Web browser
       │                                   │
       ▼                                   ▼
POST /api/auth/miniapp            /api/auth/web/* (SSO, magic link)
       │                                   │
       ▼                                   ▼
 Mini App JWT (Bearer)            lc_session cookie + CSRF
       │                                   │
       ▼                                   ▼
 /api/miniapp/*                    /api/orgs/*
```

---

## Mini App flow

### Login — `POST /api/auth/miniapp`

The handler is in `backend/internal/delivery/http/handlers/miniapp_auth.go`.

**Request**

```json
{ "init_data": "<Telegram initData string>" }
```

**Validation steps**

1. **HMAC signature** — The `init_data` string is verified against `BOT_TOKEN` using the standard Telegram Web App verification algorithm: `HMAC-SHA256(HMAC-SHA256("WebAppData", botToken), dataCheckString)`. The `hash` field is compared; any mismatch returns `401 {"code":"invalid_init_data"}`.
2. **`auth_date` freshness** — `auth_date` must be within 24 hours of the server clock. A stale token returns `401 {"code":"invalid_init_data"}`.
3. **`bot_users` lookup** — The resolved `telegram_id` is looked up in `bot_users`. If no row exists the response is `401 {"code":"not_registered"}` — the user must open the bot and run `/start`.

**Dev-mode bypass** (`AUTH_DEV_MODE=true`)

When `AUTH_DEV_MODE` is set and `init_data` does not look like a real Telegram `initData` string (no `hash=` or `auth_date=`), the value is treated as a raw `telegram_id` integer. This allows browser-only development without a running Telegram client. The frontend counterpart is `VITE_MINIAPP_DEV_TG_ID`.

**Success response**

```json
{
  "token": "<miniapp_jwt>",
  "user": {
    "telegram_id": 123456789,
    "name": "Alice",
    "email": "alice@example.com",
    "role": "user"
  }
}
```

### Mini App JWT

The token is signed with `JWT_SECRET` using HS256. Key claims:

| Claim     | Value                                                                      |
| --------- | -------------------------------------------------------------------------- |
| `tok_typ` | `"miniapp"` — distinguishes Mini App tokens at parse time                  |
| `tg_id`   | Telegram user ID (integer)                                                   |
| `email`   | From `bot_users.email`                                                       |
| `role`    | From `bot_users.role` — `"user"` or `"admin"`                                |
| `exp`     | 24 hours from issuance (configurable via `JWT_TTL_HOURS`)                  |
| `iss`     | `JWT_ISSUER` (default `"lead-cat"`)                                        |

Source: `backend/internal/platform/auth/miniapp_token.go` (`MiniAppClaims`, `MiniAppToken`).

### `MiniAppAuth` middleware

`backend/internal/delivery/http/middleware/miniapp_auth.go` guards every route under `/api/miniapp/*`.

On each request it:

1. Extracts the `Authorization: Bearer <token>` header.
2. Parses and validates the Mini App JWT (signature + `tok_typ` check).
3. Calls `GetBotUserByTelegramID` against Postgres — live lookup so role or de-registration changes take effect immediately.
4. Sets `c.Locals("bot_user")` to the `postgres.BotUser` struct.

Any failure returns `401 unauthorized`.

### Roles and admin bootstrap

Roles live in `bot_users.role`. Supported values: `"user"` (default) and `"admin"`.

Admin accounts are bootstrapped by listing Telegram IDs in the `BOT_ADMIN_TELEGRAM_IDS` environment variable. The bot `/start` handler creates the `bot_users` row with `role = "admin"` for those IDs.

Admin-only routes are under `/api/miniapp/admin/*`.

---

## Web auth flow

Active routes under `/api/auth/web/*` (registered **before** the legacy 410 catch-all):

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `GET` | `/api/auth/web/:provider/start` | OAuth/SSO redirect (Google, Microsoft) |
| `GET` | `/api/auth/web/:provider/callback` | OAuth callback; sets `lc_session` cookie |
| `POST` | `/api/auth/web/magic/request` | Email magic-link request |
| `GET` | `/api/auth/web/magic/verify` | Legacy: consume magic link from query; sets session cookie |
| `POST` | `/api/auth/web/magic/verify` | Preferred: `{ "token": "..." }` in body; sets session cookie; returns `{ "redirect": "/..." }` |

Magic-link emails point to the admin SPA at `/auth/magic?token=…`, which POSTs the token to the verify endpoint (token is not sent to the API in the email URL path).
| `POST` | `/api/auth/web/logout` | Revoke session (requires session) |
| `GET` | `/api/auth/web/me` | Current `platform_users` profile (requires session) |

Sessions are stored in `web_sessions` (hashed token, TTL, user agent, IP). The browser sends the `lc_session` httpOnly cookie; mutating `/api/orgs/*` requests also require the `X-CSRF-Token` header matching the cookie value.

Web users are rows in `platform_users` with `auth_sub = email:<addr>` (same convention as the TMA organizer bridge), so TMA and web identities **merge by email**.

Org-scoped API: `/api/orgs/*` — see `docs/API.md`.

---

## Registration

**Mini App:** Users register via the Telegram bot, not through HTTP forms.

1. User opens the bot and sends `/start`.
2. Bot FSM: full name → corporate email → `bot_users` row created (no email OTP).
3. On the next Mini App open, `POST /api/auth/miniapp` succeeds and issues a Mini App JWT.

**Web:** Sign-in via SSO or magic link; org membership via invites (`organization_invites`) or creating a new org.

---

## Identity model

| World | Table | Key | Created by |
| ----- | ----- | --- | ---------- |
| **Bot users** | `bot_users` | `telegram_id` + email + role | Telegram bot `/start` FSM |
| **Platform users** | `platform_users` | UUID + `auth_sub` | Web sign-in (`UpsertWebIdentity`) or TMA bridge (`EnsureMiniAppOrganizer`) |
| **Web sessions** | `web_sessions` | hashed cookie token | `POST /api/auth/web/*` success |

**`EnsureMiniAppOrganizer`** (`application/organizer_bridge.go`) — at TMA meeting-write time, find-or-creates `platform_users` by `auth_sub = email:<email>`, links Telegram ID, returns organizer UUID. Idempotent; merges with web accounts sharing the same email.

**TMA tenancy (interim):** meeting writes resolve the **default organization** (`EnsureDefaultOrganization`) with Google configured — not org-picker in JWT yet. Web uses explicit org membership. Convergence is a later phase (see SaaS Phase 0 spec decision #8).

---

## Environment variables

| Variable                   | Purpose                                                                                       |
| -------------------------- | --------------------------------------------------------------------------------------------- |
| `JWT_SECRET`               | HS256 signing key — minimum 16 characters                                                     |
| `JWT_ISSUER`               | Claim `iss` (default `"lead-cat"`)                                                            |
| `JWT_TTL_HOURS`            | Mini App token lifetime in hours (default `24`)                                               |
| `BOT_TOKEN`                | Telegram bot token — used for `initData` HMAC validation and bot polling                      |
| `BOT_ADMIN_TELEGRAM_IDS`   | Comma-separated Telegram IDs to bootstrap as `role="admin"`                                   |
| `AUTH_DEV_MODE`            | Enable dev bypass: raw telegram_id accepted as `init_data` when it lacks `hash=`/`auth_date=` |
| `VITE_MINIAPP_DEV_TG_ID`   | Frontend: telegram_id sent as `init_data` in browser-only dev mode                            |
| `WEB_SESSION_TTL`          | Web session lifetime (see `docs/DEPLOY-DOKPLOY.md`)                                           |
| `WEB_COOKIE_DOMAIN`        | Optional cookie domain for cross-subdomain sessions                                           |
| OAuth client IDs/secrets     | Google/Microsoft web SSO (see deploy docs)                                                    |

---

## Appendix — Retired platform auth (410 Gone)

Explicit legacy prefixes (`/api/auth/email/*`, passkey, OAuth, etc.) and the catch-all `/api/auth/*` (except routes registered earlier: `miniapp` + `web`) return **410 Gone** with `Deprecation: true`. Use `/api/auth/web/*`, `/api/orgs/*`, or `/api/miniapp/admin/*` for operator setup — see `docs/SETUP.md`.
