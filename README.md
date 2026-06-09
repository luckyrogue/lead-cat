# Lead Cat

Google Meet meeting-management Mini App — Telegram-delivered notifications, employee scheduling, and conflict detection (in development — see [docs/MEETINGS.md](docs/MEETINGS.md)).

**Repository:** [github.com/luckyrogue/lead-cat](https://github.com/luckyrogue/lead-cat)

## Quick start (local)

```bash
make setup          # .env, docker compose, pnpm install
# edit .env — BOT_TOKEN, MASTER_ENCRYPTION_KEY
make migrate
make dev            # backend :8080 + frontend :3000
```

Or step by step: `make help`

## Docs

| Doc                                              | Topic                               |
| ------------------------------------------------ | ----------------------------------- |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)     | System design                       |
| [docs/DEPLOY-DOKPLOY.md](docs/DEPLOY-DOKPLOY.md) | Dokploy                             |
| [docs/AUTH.md](docs/AUTH.md)                     | Mini App auth (Telegram `initData` → JWT) |
| [docs/SETUP.md](docs/SETUP.md)                   | Local dev & environment setup       |
| [docs/MEETINGS.md](docs/MEETINGS.md)             | Google Meet meetings (in dev)       |
| [docs/NEW-FEATURES.md](docs/NEW-FEATURES.md)     | Meetings spec (ТЗ)                  |
| [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md)     | Backend/frontend requirements       |
| [docs/DESIGN-CATS.md](docs/DESIGN-CATS.md)       | Cat UI                              |

## Repo layout

- `backend/` — Go monolith (`cmd/server`, `cmd/migrate`, `internal/`, `migrations/`, `test/smoke/` E2E)
- `frontend/` — React Mini App
- `deploy/` — `Dockerfile`, `docker-compose.yml` (local Postgres + Redis), `.env.example`
- `config/` — shared tooling configs (Prettier, tsconfig base, EditorConfig, golangci-lint)
- JS tooling (Prettier, shadcn CLI): `frontend/package.json` + `pnpm` only — `make fmt` runs `pnpm run format` there

Smoke (`make smoke`) and coverage gate (`make coverage`) are Go tests, not scripts.
