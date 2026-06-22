# Lead Cat

Google Meet meeting-management Telegram Mini App — schedule, edit, and cancel meetings in Telegram with Google Calendar sync.

**Repository:** [github.com/luckyrogue/lead-cat](https://github.com/luckyrogue/lead-cat)

## Quick start (local)

```bash
make setup          # .env, docker compose, pnpm install
# edit .env — BOT_TOKEN, MASTER_ENCRYPTION_KEY
make migrate
make dev            # backend :8080 + mini-app dev server
```

Or step by step: `make help`

## Docs

| Doc | Topic |
| --- | ----- |
| [AGENTS.md](AGENTS.md) | Stack, architecture, conventions |
| [docs/README.md](docs/README.md) | Doc index |
| [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) | Workflow, PR checks via GitHub Actions |
| [deploy/README.md](deploy/README.md) | Dokploy deploy, env, CI |
| [deploy/.env.example](deploy/.env.example) | Local env template |

## Repo layout

- `apps/backend/` — Go monolith (`cmd/server`, `cmd/migrate`, `internal/`, `migrations/`)
- `apps/mini-app/`, `apps/admin/`, `apps/landing/` — React frontends
- `deploy/` — Dockerfiles (`deploy/docker/`), nginx, compose, env examples
- `packages/` — shared UI, brand, api-client, config
