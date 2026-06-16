# Local development

```bash
make setup    # .env, Postgres/Redis, pnpm install
make migrate
make dev      # backend :8080 (hot reload) + frontend :3000 (HMR)
```

## Backend hot reload

`make dev` uses [Air](https://github.com/air-verse/air): при сохранении `.go` сервер пересобирается и перезапускается.

```bash
env -u GOROOT go install github.com/air-verse/air@latest
export PATH="$(env -u GOROOT go env GOPATH)/bin:$PATH"   # если air not found
make backend-watch   # только API
```

Без Air: `make backend` — один запуск через `go run`.

### Ошибка `go1.26.0 does not match go tool version go1.26.3`

Часто из‑за `GOROOT` от goenv на старой версии при бинарнике Go 1.26.3. Варианты:

```bash
env -u GOROOT go install github.com/air-verse/air@latest   # разово
goenv install 1.26.3 && goenv local 1.26.3                  # в каталоге проекта
```

`make backend-watch` уже запускает `air` с `env -u GOROOT`.

All targets: `make help`

## Web auth env keys (SaaS Phase 0)

These keys are required for the web dashboard auth flow added in SaaS Phase 0.
Set them in your `.env` file.

### Core

| Key | Purpose | Default |
| --- | ------- | ------- |
| `APP_BASE_URL` | Public base URL of the app. Used to build OAuth redirect URLs, magic-link verification URLs, and to derive whether cookies should be `Secure`. | `http://localhost:8080` |
| `WEB_COOKIE_DOMAIN` | Domain scope for web session cookies. Leave empty for `localhost`. | _(empty)_ |
| `WEB_SESSION_TTL_HOURS` | Lifetime of a web session cookie in hours. | `720` (30 days) |
| `MAGIC_LINK_TTL_MINUTES` | How long a magic-link token remains valid. | `15` |

### Magic-link (SMTP)

Magic-link sign-in sends a one-time login email. All `SMTP_*` vars must be set.

| Key | Purpose | Local dev value |
| --- | ------- | --------------- |
| `SMTP_HOST` | SMTP server hostname. | `localhost` |
| `SMTP_PORT` | SMTP server port. | `1025` |
| `SMTP_USERNAME` | SMTP auth username. | _(empty — Mailpit needs none)_ |
| `SMTP_PASSWORD` | SMTP auth password. | _(empty — Mailpit needs none)_ |
| `SMTP_FROM` | Sender address on outgoing emails. | `dev@lead-cat.local` |

**Local dev:** `make up` starts a [Mailpit](https://github.com/axllent/mailpit) container
(SMTP at `localhost:1025`, inbox UI at <http://localhost:8025>). Set the vars above to the
local values shown — no credentials required.

### Google OAuth SSO

Optional. The Google SSO provider is skipped silently if these are not set.

| Key | Purpose |
| --- | ------- |
| `GOOGLE_OAUTH_CLIENT_ID` | Google OAuth 2.0 client ID from Google Cloud Console. |
| `GOOGLE_OAUTH_CLIENT_SECRET` | Corresponding client secret. |

Redirect URI to register: `{APP_BASE_URL}/api/auth/web/google/callback`

### Microsoft OAuth SSO

Optional. The Microsoft SSO provider is skipped silently if these are not set.

| Key | Purpose |
| --- | ------- |
| `MICROSOFT_OAUTH_CLIENT_ID` | Azure AD app (client) ID. |
| `MICROSOFT_OAUTH_CLIENT_SECRET` | Corresponding client secret. |

Redirect URI to register: `{APP_BASE_URL}/api/auth/web/microsoft/callback`

### Minimal local `.env` additions

```dotenv
# Web auth — core
APP_BASE_URL=http://localhost:8080
WEB_COOKIE_DOMAIN=
WEB_SESSION_TTL_HOURS=720
MAGIC_LINK_TTL_MINUTES=15

# Magic-link SMTP (Mailpit, started by make up)
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM=dev@lead-cat.local

# SSO — leave blank to disable providers locally
GOOGLE_OAUTH_CLIENT_ID=
GOOGLE_OAUTH_CLIENT_SECRET=
MICROSOFT_OAUTH_CLIENT_ID=
MICROSOFT_OAUTH_CLIENT_SECRET=
```

## Checks

```bash
make lint
make typecheck
make build
```
