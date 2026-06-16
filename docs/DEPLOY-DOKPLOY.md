# Deploy on Dokploy

Lead Cat deploys as **four independent services** built from one monorepo:

| Service    | Source              | Runtime           | Port | Notes                                              |
| ---------- | ------------------- | ----------------- | ---- | -------------------------------------------------- |
| `backend`  | `apps/backend`      | Go binary         | 8080 | REST `/api`, Telegram bot, asynq workers, migrate. |
| `landing`  | `apps/landing`      | Node (SSR)        | 3000 | Marketing site, `ssr: true`, `react-router-serve`. |
| `admin`    | `apps/admin`        | nginx (static)    | 80   | SPA dashboard; nginx proxies `/api/*` → backend.   |
| `mini-app` | `apps/mini-app`     | nginx (static)    | 80   | Telegram Mini App SPA; same `/api` proxy.          |

Each app has its own `Dockerfile`. **Build context is the repository root** (the pnpm
workspace must resolve), so in Dokploy set the build context to `.` and the Dockerfile
path to `apps/<service>/Dockerfile`.

The SPA containers (`admin`, `mini-app`) run nginx that serves the static build **and**
reverse-proxies `/api/*` to the backend. This keeps the browser same-origin, so the
admin cookie session works with `SameSite=Lax` and no CORS is needed. The proxy target
is the `API_UPSTREAM` env var (default `http://backend:8080`). The SPAs are built with
`VITE_API_URL` empty so the client calls relative `/api` paths.

## 1. Postgres

Create a Postgres service → copy the connection string as `DATABASE_URL` on `backend`.

## 2. Redis

Create a Redis service → set `REDIS_URL=redis://:password@host:6379/0` on `backend`.

## 3. backend

| Item       | Value                                                              |
| ---------- | ------------------------------------------------------------------ |
| Dockerfile | `apps/backend/Dockerfile` (context `.`)                            |
| Port       | `8080`                                                             |
| Health     | `GET /api/health` (also in image `HEALTHCHECK`)                    |
| Entrypoint | `/app/lead-cat` (runs migrations on start when `AUTO_MIGRATE=true`)|

### Required env

```env
BOT_TOKEN=<telegram-bot-token>
BOT_ADMIN_TELEGRAM_IDS=<comma-separated-telegram-user-ids>
DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=require
REDIS_URL=redis://:password@host:6379/0
MASTER_ENCRYPTION_KEY=<min-32-char-random-secret>
JWT_SECRET=<min-16-char-random-secret>
JWT_ISSUER=lead-cat
JWT_TTL_HOURS=168
HTTP_ADDR=:8080
AUTO_MIGRATE=true
```

### Web auth + origins

```env
APP_BASE_URL=https://admin.your-domain.example.com   # admin origin (magic-link/SSO redirects)
WEBAPP_URL=https://app.your-domain.example.com        # mini-app / bot deep links
WEB_COOKIE_DOMAIN=                                    # set to a parent domain only if sharing cookies across subdomains
WEB_SESSION_TTL_HOURS=168
MAGIC_LINK_TTL_MINUTES=15
CORS_ALLOWED_ORIGINS=https://admin.your-domain.example.com
```

With the nginx `/api` proxy the admin browser is same-origin with the API, so
`CORS_ALLOWED_ORIGINS` only matters for any client that calls the API cross-origin.

### Optional

```env
GOOGLE_OAUTH_CLIENT_ID=        # SSO; empty disables the provider
GOOGLE_OAUTH_CLIENT_SECRET=
MICROSOFT_OAUTH_CLIENT_ID=
MICROSOFT_OAUTH_CLIENT_SECRET=
SMTP_HOST=                     # outbound email for magic links / invites
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM=Lead Cat <no-reply@your-domain.example.com>
LOG_LEVEL=info                 # debug | info | warn | error
LOG_FORMAT=json                # json (prod) | console (local)
CALENDAR_STUB=false            # true = skip real Google Calendar calls (staging)
```

The backend no longer serves the frontend; leave `STATIC_DIR` unset.

## 4. landing

| Item       | Value                                   |
| ---------- | --------------------------------------- |
| Dockerfile | `apps/landing/Dockerfile` (context `.`) |
| Port       | `3000` (`PORT` env, default 3000)       |
| Runtime    | Node SSR via `react-router-serve`       |
| Build arg  | `VITE_SITE_URL` — public site URL for canonical, Open Graph, sitemap |

```env
VITE_SITE_URL=https://your-domain.example.com
```

Set at **build time** (canonical URLs, `og:image`, `robots.txt`, `sitemap.xml`). Routes:
`/` (ru), `/en`, `/kk`. Shared brand assets live in `packages/brand/public` and are
synced into `apps/landing/public` before build (`make brand-sync`).

Point the marketing domain at port 3000.

## 5. admin

| Item       | Value                                                      |
| ---------- | ---------------------------------------------------------- |
| Dockerfile | `apps/admin/Dockerfile` (context `.`)                      |
| Port       | `80`                                                       |
| Env        | `API_UPSTREAM=http://backend:8080` (internal backend URL)  |
| Build arg  | `VITE_API_URL` — leave empty (relative `/api` via proxy)   |

Domain example: `https://admin.your-domain.example.com` → port 80. Set `API_UPSTREAM`
to the backend's internal service URL on the Dokploy network.

## 6. mini-app

| Item       | Value                                                                  |
| ---------- | ---------------------------------------------------------------------- |
| Dockerfile | `apps/mini-app/Dockerfile` (context `.`)                               |
| Port       | `80`                                                                   |
| Env        | `API_UPSTREAM=http://backend:8080`                                     |
| Build args | `VITE_API_URL` (empty), `VITE_BOT_USERNAME`, `VITE_TMA_DEV_TG_ID` (empty in prod) |

Domain example: `https://app.your-domain.example.com` → port 80. Configure this URL as
the Telegram Mini App URL in BotFather.

## Local full-stack run

```bash
cp deploy/.env.example deploy/.env   # fill secrets
docker compose -f deploy/docker-compose.full.yml up --build
```

Brings up Postgres, Redis, mailpit, and all four services wired together
(backend :8080, landing :3000, admin :3001, mini-app :3002). `deploy/docker-compose.yml`
remains the lightweight infra-only stack used by `make up` for local development.

## Google Calendar / Meet integration

The Google service account is configured at runtime — no env var holds the credentials.
An admin pastes the service-account JSON via the Mini App admin overlay
(Profile → Admin → Integrations), which calls `PATCH /api/miniapp/admin/integrations`.
Credentials are encrypted with `MASTER_ENCRYPTION_KEY` before storage. Set
`CALENDAR_STUB=false` (or omit) in production.

## Employee directory

Embedded in the backend binary as `apps/backend/internal/platform/employeedir/employees.csv`.
To update: edit the CSV, rebuild the `backend` image, redeploy.

## Rollback

Redeploy the previous image tag per service in Dokploy. Migrations are forward-only —
test on staging before promoting.

## GitHub Actions (build → GHCR → Dokploy)

On push to `main` or tag `v*.*.*`:

1. **build** — Go vet + build + frontend `pnpm build` (also on PRs).
2. **docker** — per service: build image, push to `ghcr.io/<owner>/lead-cat-<service>`, trigger Dokploy webhook.

Images are tagged with commit SHA on `main`, or the git tag on releases (`:latest` is updated too).

### Repository secrets

Create one Dokploy **Deploy Webhook** per application and add GitHub secrets:

| Secret | Service |
| ------ | ------- |
| `DOKPLOY_WEBHOOK_BACKEND` | `backend` |
| `DOKPLOY_WEBHOOK_LANDING` | `landing` |
| `DOKPLOY_WEBHOOK_ADMIN` | `admin` |
| `DOKPLOY_WEBHOOK_MINI_APP` | `mini-app` |

If a secret is missing, that service is skipped (image is still pushed). In Dokploy, point each app at the matching GHCR image, e.g. `ghcr.io/<owner>/lead-cat-backend`.

## Dev-only variables

| Variable                              | Status                                            |
| ------------------------------------- | ------------------------------------------------- |
| `AUTH_DEV_MODE`                       | Mini App dev bypass — **must not be set in prod**. |
| `VITE_TMA_DEV_TG_ID`                  | Frontend browser dev (mini-app) — local only.     |
