# Deploy on Dokploy

## 1. Postgres

Create Postgres service → copy `DATABASE_URL`.

## 2. Redis

Create Redis 7 → `REDIS_URL=redis://:password@host:6379/0`.

## 3. Lead Cat app (pull-only)

CI builds and pushes the image; Dokploy only pulls from registry.

| Item       | Value                                                                 |
| ---------- | --------------------------------------------------------------------- |
| Registry   | `ghcr.io/<github-owner>/lead-cat`                                     |
| Tag        | commit SHA on `main`, or `v*` on release tag (webhook sends this tag) |
| Port       | `8080`                                                                |
| Health     | `GET /api/health` (also in image `HEALTHCHECK`)                       |
| Entrypoint | `/app/lead-cat` (migrations on start when `AUTO_MIGRATE=true`)        |

**GHCR access:** make the package public, or add a registry credential in Dokploy (read packages).

**Do not build on Dokploy** — set deploy type to _Docker image_ and point at the GHCR image above.

### GitHub Actions checklist

| Type   | Name              |
| ------ | ----------------- |
| Secret | `DOKPLOY_WEBHOOK` |

Mini App is built into the image with `VITE_AUTH_DEV_MODE=false`. Auth is native JWT — no separate OIDC issuer in CI.

### Environment

```env
BOT_TOKEN=
DATABASE_URL=
REDIS_URL=
MASTER_ENCRYPTION_KEY=
JWT_SECRET=
JWT_ISSUER=lead-cat
WEBAPP_URL=https://your-domain
WEBAUTHN_RP_ID=your-domain.com
WEBAUTHN_RP_ORIGIN=https://your-domain
GITHUB_OAUTH_CLIENT_ID=
GITHUB_OAUTH_CLIENT_SECRET=
GITLAB_OAUTH_CLIENT_ID=
GITLAB_OAUTH_CLIENT_SECRET=
HTTP_ADDR=:8080
LOG_LEVEL=info
LOG_FORMAT=json
AUTO_MIGRATE=true
CORS_ALLOWED_ORIGINS=https://your-domain
```

See [AUTH.md](AUTH.md) for auth setup. OAuth callback URL: `{WEBAPP_URL}/api/auth/oauth/callback`.

### Domains

- App: `https://notify.example.com` → port 8080 (Mini App + `/api`)

## 4. CI webhook

Configure Dokploy deploy hook on image push (`.github/workflows/_docker.yml`).

## Rollback

Redeploy previous image tag; migrations are forward-only (test `migrate` on staging first).
