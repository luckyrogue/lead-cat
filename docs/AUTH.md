# Authentication

Lead Cat is a single-purpose Google Meet meetings-management Telegram Mini App. **Telegram-native auth is the only user-facing flow.** Platform auth (email/phone OTP, passkey, OAuth) and `/api/workspaces/*` return **410 Gone** — operator setup uses Mini App admin inside Telegram.

> **Glossary:** **miniapp** — Telegram Mini App API layer (formerly `tma` in code and paths).

---

## Overview

All Mini App users authenticate through Telegram. There is no login page, no password, and no OAuth consent screen for end users. The backend issues a short-lived **Mini App JWT** that the frontend stores locally and attaches to every `/api/miniapp/*` request.

```
Telegram client  →  POST /api/auth/miniapp  →  Mini App JWT
                                                    ↓
                        Authorization: Bearer <miniapp_jwt>
                                                    ↓
                              /api/miniapp/*  (MiniAppAuth middleware)
                                                    ↓
                           c.Locals("bot_user")  →  handler
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

## Registration

Users are registered by the Telegram bot, not through the Mini App directly.

1. User opens the bot and sends `/start`.
2. Bot FSM: full name → corporate email → `bot_users` row created (no email OTP).
3. On the next Mini App open, `POST /api/auth/miniapp` succeeds and issues a Mini App JWT.

There is no self-service registration form in the Mini App. Unregistered users see a "register in the bot" screen (triggered by the `not_registered` error code).

---

## Identity model

**`bot_users`** is the source of truth for Mini App identity: `telegram_id`, `email`, `full_name`, `role`.

**`platform_users`** is an internal organizer bridge: `EnsureMiniAppOrganizer` find-or-creates a linked `platform_users` row by email on first meeting write. It is not a login surface.

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

---

## Appendix — Retired platform auth (410 Gone)

All routes except `POST /api/auth/miniapp` under `/api/auth/*` return **410 Gone** with `Deprecation: true`. Operator bootstrap uses `/api/miniapp/admin/*` with a Mini App JWT from an admin `bot_user` — see `docs/SETUP.md`.
