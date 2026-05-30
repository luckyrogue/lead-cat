# Design — Telegram bot registration (§3)

**Date:** 2026-05-30
**Status:** Approved (brainstorm)
**Part of:** the meetings feature ([NEW-FEATURES.md](../../NEW-FEATURES.md) §3). Foundation for §5 (notifications — per-participant DMs need the TG-ID↔email map this builds).

## Goal

A Telegram bot `/start` registration flow (FSM): collect ФИО + corporate email, verify the email by OTP, and create a `bot_users` record (Telegram ID ↔ email ↔ name + role). One Telegram ID = one account; one email = one Telegram ID.

## Decisions (from brainstorm)

- **User model: dedicated `bot_users` table** — global, decoupled from `platform_users` (web native-auth) and `employees` (per-workspace directory). Matches ТЗ §3 literally; §5 will join `email → bot_users.telegram_id` for DMs.
- **Email: OTP-verified** — reuse the existing `platformauth.OTP` service (channel `"email"`), same as web auth. `AUTH_OTP_LOG=true` logs the code locally.
- **FSM state: Redis** — ephemeral, key `botreg:<telegram_id>`, TTL ~15m, JSON `{step, full_name, email}`.
- **Testability: injected interfaces** — the registration service depends on small interfaces (store, OTP, session store) so the FSM is unit-tested without Redis/DB/Telegram.
- **Admin bootstrap: env** — `BOT_ADMIN_TELEGRAM_IDS` (CSV); a registrant whose Telegram ID is listed gets `role='admin'`, else `'user'`.

## Data (new goose migration)

`bot_users`:
- `id UUID PK`, `telegram_id BIGINT NOT NULL UNIQUE`, `full_name TEXT NOT NULL`, `email TEXT NOT NULL UNIQUE`, `role TEXT NOT NULL DEFAULT 'user'`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`.

Store methods:
- `GetBotUserByTelegramID(ctx, telegramID int64) (BotUser, error)`
- `GetBotUserByEmail(ctx, email string) (BotUser, error)`
- `CreateBotUser(ctx, telegramID int64, fullName, email, role string) (BotUser, error)`

## Registration FSM (new package `internal/platform/botreg`)

States: `awaiting_name → awaiting_email → awaiting_otp → done`.

Service depends on injected interfaces:
```go
type userStore interface {
    GetBotUserByTelegramID(ctx, telegramID int64) (postgres.BotUser, error)
    GetBotUserByEmail(ctx, email string) (postgres.BotUser, error)
    CreateBotUser(ctx, telegramID int64, fullName, email, role string) (postgres.BotUser, error)
}
type otpSender interface {
    Send(ctx, channel, dest string) (string, error)
    Verify(ctx, channel, dest, code string) (bool, error)
}
type sessions interface { // Redis-backed; key botreg:<tgID>
    Get(ctx, telegramID int64) (*State, error)
    Set(ctx, telegramID int64, s State) error
    Del(ctx, telegramID int64) error
}
```

Flow:
- **Start(tgID):** if `GetBotUserByTelegramID` finds a user → reply "welcome back" (menu), no state. Else set `awaiting_name`, reply asking for ФИО.
- **OnText(tgID, text):** dispatch on current state:
  - `awaiting_name` → store `full_name = text`; set `awaiting_email`; reply ask email.
  - `awaiting_email` → validate format; if `GetBotUserByEmail` finds a row → reply "email taken", stay; else `OTP.Send("email", email)`, store `email`, set `awaiting_otp`, reply ask code.
  - `awaiting_otp` → `OTP.Verify("email", email, code)`; if ok → role = admin if tgID ∈ `BOT_ADMIN_TELEGRAM_IDS` else user; `CreateBotUser(...)`; `Del` state; reply "done, <name> 🐾" (menu). If bad → reply "wrong code", stay.
- Each method returns the bot reply text. No active session + non-command text → ignored.

## Telegram wiring

`MultiHandler.Handle`:
- On `/start` → `Registrar.Start(tgID)` → send reply.
- On non-command text → if `sessions.Get` returns an active state → `Registrar.OnText(tgID, text)` → send reply. Otherwise current behavior (ignore non-commands).

`NewMultiHandler` gains the OTP service + Redis client (built in `cmd/server` from `rdb`, `cfg.AuthOTPLog`); it constructs the `botreg` service + Redis session store.

## Config

New `BOT_ADMIN_TELEGRAM_IDS string` (CSV) in `platform/config`, parsed to a set of int64 used at registration to assign `role`.

## Testing

- **Unit (core):** `botreg` FSM with fake `userStore`/`otpSender`/`sessions` — cover: start (new vs already-registered), name→email, email-taken rejection, email→OTP-sent, OTP wrong (retry), OTP ok → user created with correct role (admin vs user). Pure, no Redis/DB/Telegram.
- **Store:** build-verified (no DB harness in the package).
- **Manual/integration (out of CI):** real bot polling with a real `BOT_TOKEN`; `/start` end-to-end. Not in CI (no Telegram in CI).

## Out of scope (later)

- Promoting other users to admin (a command/endpoint).
- Rich main menu / "Open App" web_app button after registration (minimal welcome only now).
- Changing email after registration (ТЗ: via admin — later).
- Per-participant notifications/reminders (§5) and the bot running in CI.
- Linking `bot_users` to `employees`/`platform_users` (future, when needed).
