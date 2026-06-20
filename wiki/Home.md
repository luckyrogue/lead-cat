# Lead Cat Wiki

Google Meet meetings-management **Telegram Mini App**: schedule, edit, and cancel meetings; Google Calendar sync; notifications via Telegram bot.

> Detailed docs live in the [main repository](https://github.com/luckyrogue/lead-cat/tree/main/docs). This wiki is a navigation hub.

## Quick links

| Topic | Link |
| --- | --- |
| Local setup | [docs/LOCAL_DEV.md](https://github.com/luckyrogue/lead-cat/blob/main/docs/LOCAL_DEV.md) |
| Architecture | [docs/ARCHITECTURE.md](https://github.com/luckyrogue/lead-cat/blob/main/docs/ARCHITECTURE.md) |
| Auth (TMA + web) | [docs/AUTH.md](https://github.com/luckyrogue/lead-cat/blob/main/docs/AUTH.md) |
| Meetings product | [docs/MEETINGS.md](https://github.com/luckyrogue/lead-cat/blob/main/docs/MEETINGS.md) |
| Full spec (ТЗ) | [docs/NEW-FEATURES.md](https://github.com/luckyrogue/lead-cat/blob/main/docs/NEW-FEATURES.md) |
| Deploy (Dokploy) | [docs/DEPLOY-DOKPLOY.md](https://github.com/luckyrogue/lead-cat/blob/main/docs/DEPLOY-DOKPLOY.md) |
| API reference | [docs/API.md](https://github.com/luckyrogue/lead-cat/blob/main/docs/API.md) |
| Cat UI design | [docs/DESIGN-CATS.md](https://github.com/luckyrogue/lead-cat/blob/main/docs/DESIGN-CATS.md) |

## Repo layout

- `apps/backend/` — Go monolith (HTTP, Telegram bot, asynq workers)
- `apps/mini-app/` — Telegram Mini App (React)
- `apps/admin/` — web admin dashboard
- `apps/landing/` — marketing landing
- `packages/` — shared UI, brand, config
- `deploy/` — Docker Compose, Dokploy env examples

## Local quick start

```bash
make setup    # .env, docker, pnpm install
make migrate
make dev      # backend :8080, frontends on :3000 / :3001
```

See [CONTRIBUTING.md](https://github.com/luckyrogue/lead-cat/blob/main/CONTRIBUTING.md) for PR workflow and `make ci`.

## Stack

- **Backend:** Go, Fiber, Postgres, Redis (asynq)
- **Frontends:** React, Vite, Turbo monorepo
- **Integrations:** Telegram Bot API, Google Calendar / Meet

## Security

Report vulnerabilities privately — see [SECURITY.md](https://github.com/luckyrogue/lead-cat/blob/main/SECURITY.md).
