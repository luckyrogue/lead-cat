# Authentication

Lead Cat is a single-purpose Google Meet meetings-management Telegram Mini App. **Telegram-native auth is the only user-facing flow.** The platform auth stack (email/phone OTP, passkey, OAuth) exists only for alpha operator bootstrap and is being retired — see the appendix.

---

## Overview

All Mini App users authenticate through Telegram. There is no login page, no password, and no OAuth consent screen for end users. The backend issues a short-lived **TMA JWT** that the frontend stores locally and attaches to every `/api/tma/*` request.

```
Telegram client  →  POST /api/auth/tma  →  TMA JWT
                                               ↓
                        Authorization: Bearer <tma_jwt>
                                               ↓
                              /api/tma/*  (TMAAuth middleware)
                                               ↓
                           c.Locals("bot_user")  →  handler
```

---

## TMA flow

### Login — `POST /api/auth/tma`

The handler is in `backend/internal/delivery/http/handlers/tma_auth.go`.

**Request**

```json
{ "init_data": "<Telegram initData string>" }
```

**Validation steps**

1. **HMAC signature** — The `init_data` string is verified against `BOT_TOKEN` using the standard Telegram Web App verification algorithm: `HMAC-SHA256(HMAC-SHA256("WebAppData", botToken), dataCheckString)`. The `hash` field is compared; any mismatch returns `401 {"code":"invalid_init_data"}`.
2. **`auth_date` freshness** — `auth_date` must be within 24 hours of the server clock. A stale token returns `401 {"code":"invalid_init_data"}`.
3. **`bot_users` lookup** — The resolved `telegram_id` is looked up in `bot_users`. If no row exists the response is `401 {"code":"not_registered"}` — the user must open the bot and run `/start`.

**Dev-mode bypass** (`AUTH_DEV_MODE=true`)

When `AUTH_DEV_MODE` is set and `init_data` does not look like a real Telegram `initData` string (no `hash=` or `auth_date=`), the value is treated as a raw `telegram_id` integer. This allows browser-only development without a running Telegram client. The frontend counterpart is `VITE_TMA_DEV_TG_ID`.

**Success response**

```json
{
  "token": "<tma_jwt>",
  "user": {
    "telegram_id": 123456789,
    "name": "Alice",
    "email": "alice@example.com",
    "role": "user"
  }
}
```

### TMA JWT

The token is signed with `JWT_SECRET` using HS256. Key claims:

| Claim     | Value                                                                 |
| --------- | --------------------------------------------------------------------- |
| `tok_typ` | `"tma"` — distinguishes TMA tokens from platform tokens at parse time |
| `tg_id`   | Telegram user ID (integer)                                            |
| `email`   | From `bot_users.email`                                                |
| `role`    | From `bot_users.role` — `"user"` or `"admin"`                         |
| `exp`     | 24 hours from issuance (configurable via `JWT_TTL_HOURS`)             |
| `iss`     | `JWT_ISSUER` (default `"lead-cat"`)                                   |

Source: `backend/internal/platform/auth/tma.go` (`TMAClaims`, `TMAToken`).

### `TMAAuth` middleware

`backend/internal/delivery/http/middleware/tma_auth.go` guards every route under `/api/tma/*`.

On each request it:

1. Extracts the `Authorization: Bearer <token>` header.
2. Parses and validates the TMA JWT (signature + `tok_typ` check).
3. Calls `GetBotUserByTelegramID` against Postgres — live lookup so role or de-registration changes take effect immediately.
4. Sets `c.Locals("bot_user")` to the `postgres.BotUser` struct.

Any failure returns `401 unauthorized`.

### Roles and admin bootstrap

Roles live in `bot_users.role`. Supported values: `"user"` (default) and `"admin"`.

Admin accounts are bootstrapped by listing Telegram IDs in the `BOT_ADMIN_TELEGRAM_IDS` environment variable. The bot `/start` handler upserts the `bot_users` row with `role = "admin"` for those IDs.

Admin-only routes are under `/api/tma/admin/*`.

---

## Registration

Users are registered by the Telegram bot, not through the Mini App directly.

1. User opens the bot and sends `/start`.
2. The bot FSM upserts a `bot_users` row keyed on `telegram_id`, storing `email`, display name, and role.
3. On the next Mini App open, `POST /api/auth/tma` succeeds and issues a TMA JWT.

There is no self-service registration form in the Mini App. Unregistered users see a "register in the bot" screen (triggered by the `not_registered` error code).

---

## Identity model

**`bot_users`** is the source of truth for Mini App identity: `telegram_id`, `email`, `full_name`, `role`.

**`platform_users`** is the legacy table for platform auth subjects. It is no longer relevant to normal product flows; the `EnsureTMAOrganizer` helper find-or-creates a linked `platform_users` row by email on first meeting write (internal plumbing only).

---

## Environment variables

| Variable                 | Purpose                                                                                       |
| ------------------------ | --------------------------------------------------------------------------------------------- |
| `JWT_SECRET`             | HS256 signing key (shared by TMA and platform tokens) — minimum 16 characters                 |
| `JWT_ISSUER`             | Claim `iss` (default `"lead-cat"`)                                                            |
| `JWT_TTL_HOURS`          | TMA token lifetime in hours (default `24`)                                                    |
| `BOT_TOKEN`              | Telegram bot token — used for `initData` HMAC validation and bot polling                      |
| `BOT_ADMIN_TELEGRAM_IDS` | Comma-separated Telegram IDs to bootstrap as `role="admin"`                                   |
| `AUTH_DEV_MODE`          | Enable dev bypass: raw telegram_id accepted as `init_data` when it lacks `hash=`/`auth_date=` |
| `VITE_TMA_DEV_TG_ID`     | Frontend: telegram_id sent as `init_data` in browser-only dev mode                            |

---

## Appendix — Deprecated: alpha setup (curl)

> These platform endpoints/flows exist only for alpha operator bootstrap and are
> being replaced by in-Mini-App admin (`/api/tma/admin/*`, see
> `docs/superpowers/specs/2026-06-05-tma-setup-replacement-design.md`). Not part
> of the product; slated for removal.

The platform auth stack authenticates `platform_users` (workspace operators) using email/phone OTP, WebAuthn passkey, or GitHub/GitLab OAuth. On success it issues a **platform JWT** (distinct from the TMA JWT — no `tok_typ:"tma"` claim) accepted on `/api/workspaces/*` and related operator endpoints.

| Method                | Endpoints                                             |
| --------------------- | ----------------------------------------------------- |
| Email OTP             | `POST /api/auth/email/send-code`, `.../verify`        |
| Phone OTP             | `POST /api/auth/phone/send-code`, `.../verify`        |
| Passkey login         | `POST /api/auth/passkey/login/begin`, `.../finish`    |
| Passkey register      | `POST /api/auth/passkey/register/begin`, `.../finish` |
| GitHub / GitLab OAuth | `GET /api/auth/oauth/{github\|gitlab}` → redirect     |

Public config endpoint: `GET /api/auth/config`.

Example operator bootstrap (curl):

```bash
curl -H "Authorization: Bearer dev" -H "Content-Type: application/json" \
  -d '{"name":"Acme","slug":"acme"}' http://localhost:8080/api/workspaces
```

(`AUTH_DEV_MODE=true` accepts `Bearer dev` as a valid platform token in local development.)

This entire stack is being retired. Do not build new features against it.
