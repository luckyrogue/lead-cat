# Deploy on Dokploy

Lead Cat runs as a single Go binary that serves the TMA frontend, REST API, and Telegram bot.

## 1. Postgres

Create a Postgres service → copy the connection string as `DATABASE_URL`.

## 2. Redis

Create a Redis 7 service → set `REDIS_URL=redis://:password@host:6379/0`.

## 3. Lead Cat app (pull-only)

CI builds and pushes the image; Dokploy only pulls from the registry.

| Item       | Value                                                                 |
| ---------- | --------------------------------------------------------------------- |
| Registry   | `ghcr.io/<github-owner>/lead-cat`                                     |
| Tag        | commit SHA on `main`, or `v*` on release tag (webhook sends this tag) |
| Port       | `8080`                                                                |
| Health     | `GET /api/health` (also in image `HEALTHCHECK`)                       |
| Entrypoint | `/app/lead-cat` (runs migrations on start when `AUTO_MIGRATE=true`)   |

**GHCR access:** make the package public, or add a registry credential in Dokploy with read-packages scope.

**Do not build on Dokploy** — set deploy type to _Docker image_ and point at the GHCR image above.

### GitHub Actions checklist

| Type   | Name              |
| ------ | ----------------- |
| Secret | `DOKPLOY_WEBHOOK` |

The Mini App frontend is compiled into the image with `VITE_AUTH_DEV_MODE=false`. No separate OIDC issuer is needed in CI.

### Environment variables

All variables are set by name in Dokploy's environment panel. Never paste real secrets into docs; use placeholders as shown.

#### Required

```env
# Telegram bot
BOT_TOKEN=<telegram-bot-token>
BOT_ADMIN_TELEGRAM_IDS=<comma-separated-telegram-user-ids>

# Data stores
DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=require
REDIS_URL=redis://:password@host:6379/0

# Encryption & auth
MASTER_ENCRYPTION_KEY=<min-32-char-random-secret>
JWT_SECRET=<min-16-char-random-secret>
JWT_ISSUER=lead-cat
JWT_TTL_HOURS=168

# App
WEBAPP_URL=https://your-domain.example.com
CORS_ALLOWED_ORIGINS=https://your-domain.example.com
HTTP_ADDR=:8080
AUTO_MIGRATE=true
```

#### Optional / production tuning

```env
# Logging
LOG_LEVEL=info          # debug | info | warn | error
LOG_FORMAT=json         # json (prod) | console (local)

# Calendar
CALENDAR_STUB=false     # true = skip real Google Calendar calls (staging/testing only)

# Static files (override embedded default)
STATIC_DIR=frontend/dist
```

#### Variable reference

| Variable                 | Purpose                                                                |
| ------------------------ | ---------------------------------------------------------------------- |
| `BOT_TOKEN`              | Telegram Bot API token; required in production.                        |
| `BOT_ADMIN_TELEGRAM_IDS` | Comma-separated Telegram user IDs granted the `admin` role in the app. |
| `DATABASE_URL`           | Postgres connection string (source of truth for all data).             |
| `REDIS_URL`              | Redis connection string used by the asynq job queue.                   |
| `JWT_SECRET`             | HMAC secret for signing JWT access tokens.                             |
| `JWT_ISSUER`             | `iss` claim embedded in issued JWTs.                                   |
| `JWT_TTL_HOURS`          | Token lifetime in hours (default 168 = 7 days).                        |
| `MASTER_ENCRYPTION_KEY`  | AES key for encrypting credentials at rest (Google SA JSON, etc.).     |
| `CALENDAR_STUB`          | When `true`, bypasses real Google Calendar/Meet API (for staging).     |
| `WEBAPP_URL`             | Public HTTPS base URL of the app; used in bot deep links and CORS.     |
| `STATIC_DIR`             | Directory for embedded frontend assets (default `frontend/dist`).      |
| `HTTP_ADDR`              | TCP address the server listens on (default `:8080`).                   |
| `AUTO_MIGRATE`           | When `true`, runs forward-only SQL migrations on startup.              |
| `CORS_ALLOWED_ORIGINS`   | Comma-separated allowed origins for CORS; defaults to `WEBAPP_URL`.    |
| `LOG_LEVEL`              | Structured log verbosity passed to zap.                                |
| `LOG_FORMAT`             | `json` for production; `console` for human-readable local output.      |

### Domains

- App: `https://your-domain.example.com` → port 8080 (Mini App + `/api`)

## 4. CI deploy webhook

Configure the Dokploy deploy hook to trigger on image push (`.github/workflows/_docker.yml` sets the `DOKPLOY_WEBHOOK` secret).

## 5. Google Calendar / Meet integration

The Google service account is configured at runtime — no env var is needed for the credentials themselves. An admin uses the TMA Admin overlay (Profile → Admin → Integrations) to paste the service account JSON and verify the connection. This calls `PATCH /api/tma/admin/integrations` (see `docs/superpowers/specs/2026-06-05-tma-setup-replacement-design.md`). Credentials are encrypted with `MASTER_ENCRYPTION_KEY` before storage.

Set `CALENDAR_STUB=false` (or omit it) in production to use the real Google API.

## 6. Employee directory

The employee directory is embedded in the binary as a CSV file at `backend/internal/platform/employeedir/employees.csv`. To update the directory: edit the CSV, rebuild the image, and redeploy. No env var or external service is required.

## Rollback

Redeploy the previous image tag in Dokploy. Migrations are forward-only — test on staging before promoting to production.

---

## Dev-only variables

| Variable          | Status                                              |
| ----------------- | --------------------------------------------------- |
| `AUTH_DEV_MODE`   | Mini App dev bypass — **must not be set in production**. |
| `VITE_AUTH_DEV_MODE` / `VITE_MINIAPP_DEV_TG_ID` | Frontend browser dev — local only. |
