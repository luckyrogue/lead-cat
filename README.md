# Lead Cat

Multi-tenant SaaS: Telegram reminders with evil cats, n8n-like scenarios, Mini App admin.

## Quick start (local)

```bash
make setup          # .env, docker compose, pnpm install
# edit .env — BOT_TOKEN, MASTER_ENCRYPTION_KEY
make migrate
make dev            # backend :8080 + frontend :3000
```

Or step by step: `make help`

## Docs

| Doc | Topic |
|-----|--------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design |
| [docs/DEPLOY-DOKPLOY.md](docs/DEPLOY-DOKPLOY.md) | Dokploy |
| [docs/AUTH.md](docs/AUTH.md) | Login (OTP, passkey, GitHub/GitLab) |
| [docs/SCENARIOS.md](docs/SCENARIOS.md) | Workflow builder |
| [docs/DESIGN-CATS.md](docs/DESIGN-CATS.md) | Cat UI |

## Repo layout

- `backend/` — Go monolith (`cmd/server`, `cmd/migrate`, `internal/`, `migrations/`)
- `frontend/` — React Mini App
- `docker-compose.yml` — local Postgres + Redis (optional)
- `scripts/` — CI smoke & coverage gates
