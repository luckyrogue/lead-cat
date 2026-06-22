# Contributing to Lead Cat

Thanks for helping improve Lead Cat — a Google Meet meetings-management Telegram Mini App.

## Before you start

- Read [AGENTS.md](../AGENTS.md) for stack, architecture, and conventions.
- Local setup: `make setup && make migrate && make dev` (see [README.md](../README.md)).

## Development workflow

1. Fork or branch from `main` (`feat/…`, `fix/…`, `docs/…`).
2. Keep changes focused — smallest diff that solves the task.
3. Push and open a PR — GitHub Actions runs lint, format, and build checks.
4. Update `AGENTS.md`, `deploy/README.md`, or `deploy/.env.example` when you change API, auth, env, or deploy.
5. Do not commit secrets (`.env`, tokens, credentials JSON).

Format locally: `make fmt`.

## Pull requests

- Link the issue or describe the user-visible change.
- Ensure GitHub Actions is green on your branch.

## Code style

- **Go:** Clean Architecture under `apps/backend/internal/` — see [AGENTS.md](../AGENTS.md).
- **Frontend:** lite FSD in `apps/mini-app`, `apps/admin`, `apps/landing` — see `.cursor/rules/frontend-fsd.mdc`.
- **UI:** cat design — see `.cursor/rules/cat-design.mdc`.

## Questions

Open a [GitHub Issue](https://github.com/luckyrogue/lead-cat/issues) for bugs or feature ideas.
