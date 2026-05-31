# TMA Authentication & Identity — Design (frontend integration, sub-project 1)

**Status:** approved, ready for implementation plan.
**Part of:** the larger "wire the meetings Mini App to the backend" effort, decomposed into: **(1) TMA auth & identity [this spec]** → (2) API client + read paths → (3) write paths → (4) auto/profile tabs. Each sub-project gets its own spec → plan → implementation cycle.
**Spec source (ТЗ):** `docs/NEW-FEATURES.md` (users bound by Telegram ID + corporate email, auto-registered on `/start`). Feature status: `docs/MEETINGS.md`.

## Goal

Let a Telegram Mini App user authenticate to the backend with zero extra login and see their own identity. This is the foundational slice every other sub-project depends on. It delivers: a public `initData`→TMA-JWT exchange, a TMA auth middleware, and a `GET /api/tma/me` first consumer; on the frontend, an auth bootstrap + gate that replaces the mock `ME`. **No meetings/employees/availability wiring** (sub-projects 2–3).

## Decisions (locked during brainstorming)

1. **Telegram-native auth on `bot_users`.** A Telegram user is identified by `initData` → `bot_users` (global, `telegram_id ↔ email ↔ role`). This matches the meetings feature's actual identity model (every bot flow + the global-by-email read logic) and gives the correct Mini-App UX. The existing `platform_users`/workspace web-auth world is left untouched — the two identity worlds stay separate.
2. **Exchange `initData` for a short-lived TMA JWT** (not per-request `initData` validation). Centralizes validation, avoids `auth_date` TTL tension, and reuses the existing `Authorization: Bearer` middleware shape + the frontend's axios token plumbing.
3. **Registration stays owned by the bot.** If a Telegram user has no `bot_users` row, the exchange returns `401 not_registered`; the Mini App shows a "register in the bot first" screen with a deep link. No OTP onboarding is duplicated in the Mini App (DRY).
4. **Read scope is global-by-email** (deferred to sub-project 2, noted here for context): the Mini App will show the user's own meetings as organizer/participant, mirroring the bot's `/schedule` & `/checker`. Meeting *creation*'s workspace targeting is a sub-project-3 concern.

## Codebase facts (verified)

- **Module path:** `github.com/Jaryq-Lab/notify-bot`.
- **initData validator** (`internal/infrastructure/telegram/initdata.go`): `InitDataValidator{botToken}`, `NewInitDataValidator(botToken)`, `Validate(initData string) (InitDataUser, error)` verifies the Telegram HMAC (`WebAppData` keyed by bot token) and returns `InitDataUser{ID int64, Username string}`. **It does NOT check `auth_date` freshness** and does not extract it — this design adds an `AuthDate int64` field (non-breaking; the one existing caller, `LinkTelegram`, ignores it) and a freshness check in the TMA handler.
- **JWT** (`internal/platform/auth/jwt.go`): `JWT{secret, ttl, issuer}` via `NewJWT(secret, issuer, ttl)`; `Issue(userID uuid.UUID, authSub, email, phone string)` and `Parse(token) (*TokenClaims, error)`, HS256. `TokenClaims` is platform-user-shaped (`uid` UUID + `sub`) — **not reusable as-is for TMA** (a TMA user has no `platform_users` UUID). The TMA token is a parallel, minimal type reusing the same `JWT_SECRET` + HS256.
- **Platform auth middleware** (`internal/delivery/http/middleware/auth.go`): a global app-level middleware that early-returns (skips) for `/api/health`, `/metrics`, and `strings.HasPrefix(path, "/api/auth/")`. Everything else requires a platform `Bearer` JWT (or, in `AuthDevMode`, any token resolves a dev user). **This design adds `/api/tma/` to that skip list** so TMA routes use their own middleware instead.
- **bot_users** (`internal/infrastructure/persistence/postgres/`): `GetBotUserByTelegramID(ctx, telegramID int64) (BotUser, error)`; `BotUser{ID uuid.UUID, TelegramID int64, FullName, Email, Role, ReminderMinutes string}`. Absent row → a not-found error (handler maps to `401 not_registered`).
- **Config** (`internal/platform/config/config.go`): `BotToken`, `JWTSecret`, `AuthDevMode` are present. No new required env. (One optional dev var is added — see Dev mode.)
- **Frontend:** Telegram global `window.Telegram?.WebApp?.initData` is already used in `frontend/src/features/.../link-telegram-banner.tsx`. Axios client at `frontend/src/shared/api/client.ts` (baseURL `/api`); `Authorization` header is set in `providers.tsx`. TMA app entry `frontend/src/features/tma/tma-app.tsx`; context `frontend/src/shared/tma/context.tsx`; mock current user `ME` in `frontend/src/shared/tma/mock-data.ts`.
- Conventions: build/test/lint from repo root `make test && make lint && make build`; Go run as `env -u GOROOT go ...` from `backend/`; pure logic unit-tested, IO/middleware/wiring build-verified; zap structured logging, no secrets/initData/JWT in logs; `frontend/vite.config.ts` is a do-not-touch local edit.

## Architecture

Telegram-native auth as a self-contained vertical slice, separate from the platform-user web auth.

```
Mini App load → read window.Telegram.WebApp.initData
  → POST /api/auth/tma { init_data }            (public; skipped by platform mw)
      → InitDataValidator.Validate + freshness(auth_date ≤ 24h)
      → GetBotUserByTelegramID(tg_id)
          ├─ not found → 401 not_registered
          └─ found     → mint TMA JWT {tg_id,email,role,typ:tma}
      → { token, user:{telegram_id,name,email,role} }
  → store token, set axios Authorization: Bearer
  → subsequent calls under /api/tma/* → TMA middleware → c.Locals("bot_user")
GET /api/tma/me → returns the current bot_user identity
```

### Backend components

**1. TMA token (`internal/platform/auth/tma.go`, new file in the existing package)**
- `type TMAClaims struct { TelegramID int64 \`json:"tg_id"\`; Email string \`json:"email"\`; Role string \`json:"role"\`; Typ string \`json:"typ"\`; jwt.RegisteredClaims }`.
- `type TMAToken struct { secret []byte; ttl time.Duration; issuer string }` with `NewTMAToken(secret, issuer string, ttl time.Duration) (*TMAToken, error)` (reuses `JWT_SECRET`; default ttl 24h; same ≥16-char secret guard as `NewJWT`).
- `Issue(tgID int64, email, role string) (string, error)` — sets `Typ:"tma"`, HS256.
- `Parse(token string) (*TMAClaims, error)` — HS256 only; rejects when `Typ != "tma"` (so a platform JWT can't be used as a TMA token and vice-versa).

**2. initData freshness (`internal/infrastructure/telegram/initdata.go`, extend)**
- Add `AuthDate int64` to `InitDataUser`; parse `vals.Get("auth_date")` in `Validate` (best-effort; 0 if absent). Non-breaking.
- The TMA handler enforces freshness; the validator stays a pure HMAC+parse unit.

**3. Public exchange handler `POST /api/auth/tma` (`internal/delivery/http/handlers/tma_auth.go`, new)**
- Body `{ "init_data": string }`. Steps:
  1. **Dev mode** (`cfg.AuthDevMode`): skip HMAC + freshness; derive `telegram_id` from the `init_data` string parsed as int (frontend sends a configured dev id). Else:
  2. `validator.Validate(init_data)` → `401 invalid_init_data` on error.
  3. Freshness: reject if `auth_date` older than 24h → `401 invalid_init_data`.
  4. `GetBotUserByTelegramID(tg_id)`; not found → `401 not_registered` (distinct error code).
  5. `tmaToken.Issue(tg_id, email, role)`.
  6. Respond `{ "token": "...", "user": { "telegram_id": …, "name": full_name, "email": …, "role": … } }`.
- Registered at `/api/auth/tma` so the platform middleware's existing `/api/auth/` skip makes it public.

**4. TMA auth middleware (`internal/delivery/http/middleware/tma_auth.go`, new)**
- `type TMAAuth struct { cfg; token *auth.TMAToken; store *postgres.Store; log }`, `NewTMAAuth(...)`, `Middleware(c)`.
- Requires `Authorization: Bearer <tma jwt>`; `token.Parse` → on error `401`. Re-resolve `GetBotUserByTelegramID(claims.TelegramID)` (so role/email changes and de-registration take effect) → on error `401`. Set `c.Locals("bot_user", botUser)`. (Dev mode mirrors the platform middleware: accept a dev telegram id.)
- Applied to a `/api/tma` route group (excluding the public `/api/auth/tma`).

**5. `GET /api/tma/me` (`tma_handlers.go`, new)** — reads `c.Locals("bot_user")`, returns `{ telegram_id, name, email, role }`.

**6. Wiring (`internal/delivery/http/app.go`)**
- Add `strings.HasPrefix(c.Path(), "/api/tma/")` to the platform `Auth.Middleware` skip condition.
- Construct `TMAToken` (from `cfg.JWTSecret`) and `TMAAuth`; register `app.Post("/api/auth/tma", api.TMAAuth)` and a group `tma := app.Group("/api/tma", tmaAuth.Middleware)` with `tma.Get("/me", api.TMAMe)`.

### Frontend components

**1. TMA auth client (`frontend/src/shared/tma/auth.ts` or similar, new)**
- `tmaLogin(initData: string): Promise<{ token; user }>` → `POST /api/auth/tma`.
- Stores the token (in-memory module variable + `sessionStorage` for reloads) and sets the axios default `Authorization: Bearer` header (same mechanism `providers.tsx` uses).

**2. TMA auth provider/context (`frontend/src/shared/tma/auth-context.tsx`, new)**
- On mount: read `window.Telegram?.WebApp?.initData`; in `VITE_AUTH_DEV_MODE` fall back to a configured dev id string (`VITE_TMA_DEV_TG_ID`). Call `tmaLogin`.
- Exposes `{ status: "loading" | "authed" | "not_registered" | "error", user }`.
- On a 401 from a later API call, transparently re-run `tmaLogin` once before surfacing the error.

**3. Auth gate + identity (`frontend/src/features/tma/tma-app.tsx`)**
- Wrap the app: `loading` → spinner/skeleton; `not_registered` → a "register in the bot first" screen with a deep link (`https://t.me/<bot>?start`); `error` → retry; `authed` → render the app.
- Replace the mock `ME` with the provider's `user` (the rest of the screens still read mocks for their lists — that's sub-project 2).

## Data flow & error handling

| Case | Backend | Frontend |
| --- | --- | --- |
| Valid initData, registered | 200 `{token,user}` | store token, render app |
| Bad/forged initData | 401 `invalid_init_data` | error state + retry |
| Stale initData (`auth_date` > 24h) | 401 `invalid_init_data` | re-read initData & retry once |
| Valid initData, no `bot_users` row | 401 `not_registered` | "register in the bot first" screen |
| Expired TMA JWT on a `/api/tma/*` call | 401 | re-run `tmaLogin` once, replay |
| Not in Telegram & not dev | no initData | error state ("open in Telegram") |

No `initData`, JWT, email beyond the response payload, or secrets are logged. The exchange logs `tma_auth_ok{telegram_id}` (Info) and `tma_auth_unregistered{telegram_id}` (Info) / `tma_auth_invalid` (Warn) — counts/ids only.

## Dev mode

Local dev runs outside Telegram, so there is no real `initData`. With `cfg.AuthDevMode`, `POST /api/auth/tma` skips HMAC + freshness and treats `init_data` as a raw `telegram_id` (integer string); the frontend, under `VITE_AUTH_DEV_MODE`, sends `VITE_TMA_DEV_TG_ID`. The `bot_users` row for that id must already exist (registered via the bot once, or seeded). This mirrors how the platform middleware already trusts a dev token. No HMAC bypass is possible when `AuthDevMode` is false.

## Testing

- **Backend unit (pure-ish):** `TMAToken` Issue/Parse round-trip; `Parse` rejects a token with `typ != "tma"`; `Parse` rejects HS-mismatch/expired. initData `auth_date` parsing + the freshness predicate.
- **Backend build-verified:** the exchange handler, TMA middleware, and route wiring (repo convention — no HTTP harness).
- **Frontend:** the auth provider state machine (loading → authed / not_registered / error; 401 re-login-once). The `not_registered` screen renders the bot deep link.
- Gate before merge: `make test && make lint && make build` from repo root.

## Out of scope (YAGNI — later sub-projects or never)

- Any meetings/employees/availability endpoints or screen wiring (sub-projects 2–3).
- DTO↔UI field mapping (sub-project 2).
- Refresh tokens / token rotation (24h TTL + re-exchange on load is sufficient for a Mini App).
- Merging `bot_users` with `platform_users`, or workspace selection in the Mini App.
- In-Mini-App registration/OTP onboarding (the bot owns it).
- Admin-only data views (sub-project 4); `role` is carried in the token now but not yet acted on beyond being returned by `/me`.
