# Authentication

Lead Cat uses **native auth** (no Authentik): email/phone OTP, WebAuthn passkey, and GitHub/GitLab OAuth. The API issues **JWT** access tokens (`Authorization: Bearer`).

## Methods

| Method           | Endpoints                                                       | Notes                                                         |
| ---------------- | --------------------------------------------------------------- | ------------------------------------------------------------- |
| Email OTP        | `POST /api/auth/email/send-code`, `POST /api/auth/email/verify` | Code in Redis 10 min; logged to stdout if `AUTH_OTP_LOG=true` |
| Phone OTP        | `POST /api/auth/phone/send-code`, `POST /api/auth/phone/verify` | Same as email; wire SMS provider for production               |
| Passkey          | `POST /api/auth/passkey/login/begin`, `.../finish`              | WebAuthn discoverable login                                   |
| Passkey register | `POST /api/auth/passkey/register/begin`, `.../finish`           | Requires logged-in user                                       |
| GitHub / GitLab  | `GET /api/auth/oauth/{github\|gitlab}`                          | Redirect flow → `/login?access_token=...`                     |

Public config: `GET /api/auth/config` → which providers are enabled.

## Environment

| Variable                            | Purpose                                                           |
| ----------------------------------- | ----------------------------------------------------------------- |
| `JWT_SECRET`                        | HS256 signing key (min 16 chars; required unless `AUTH_DEV_MODE`) |
| `JWT_ISSUER`                        | Claim `iss` (default `lead-cat`)                                  |
| `JWT_TTL_HOURS`                     | Token lifetime (default 168)                                      |
| `AUTH_DEV_MODE`                     | Accept `Bearer dev` or any Bearer as `auth_sub` (local/CI)        |
| `AUTH_OTP_LOG`                      | Print OTP codes to server logs                                    |
| `WEBAUTHN_RP_ID`                    | Relying party ID (hostname, e.g. `localhost`)                     |
| `WEBAUTHN_RP_ORIGIN`                | Full origin (e.g. `http://localhost:3000`)                        |
| `GITHUB_OAUTH_*` / `GITLAB_OAUTH_*` | OAuth app credentials                                             |
| `WEBAPP_URL`                        | OAuth redirect target base (`/login` callback)                    |

OAuth callback URL to register in GitHub/GitLab:

`{WEBAPP_URL}/api/auth/oauth/callback` — when the API and UI share one host use `WEBAPP_URL=https://your-domain`. For Vite dev (`:3000`), proxy `/api` to Go and set `WEBAPP_URL=http://localhost:3000`.

## Mini App

- Route `/login` — email/phone, passkey, OAuth links
- JWT stored in `localStorage` (`leadcat_access_token`)
- `VITE_AUTH_DEV_MODE=true` — skip login (Bearer `dev`)

## Dev

```bash
cd backend && AUTH_DEV_MODE=true AUTH_OTP_LOG=true WEBAPP_URL=http://localhost:3000 go run ./cmd/server
cd frontend && pnpm dev
```

## Telegram link

After login, link Telegram via `POST /api/me/link-telegram` + `X-Telegram-Init-Data` (unchanged).

## User identity (`platform_users.auth_sub`)

- `email:user@example.com`
- `phone:+77001234567`
- `github:{id}` / `gitlab:{id}`
- Passkeys attach to existing users
