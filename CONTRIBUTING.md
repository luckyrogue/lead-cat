# Contributing to Lead Cat

Thanks for helping improve Lead Cat — a Google Meet meetings-management Telegram Mini App.

## Before you start

- Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for backend layering and [AGENTS.md](AGENTS.md) for engineering conventions.
- Product behavior: [docs/NEW-FEATURES.md](docs/NEW-FEATURES.md) (spec) and [docs/MEETINGS.md](docs/MEETINGS.md).
- Local setup: [docs/LOCAL_DEV.md](docs/LOCAL_DEV.md) or `make setup && make migrate && make dev`.

## Development workflow

1. Fork or branch from `main` (`feat/…`, `fix/…`, `docs/…`).
2. Keep changes focused — smallest diff that solves the task.
3. Run checks before opening a PR:

```bash
make ci    # fmt-check, lint, test, typecheck, build
```

Backend tests only:

```bash
cd apps/backend && env -u GOROOT go test ./...
```

4. Update `docs/*` when you change API, auth, env vars, or deploy behavior.
5. Do not commit secrets (`.env`, tokens, credentials JSON).

## Pull requests

- Link the issue or describe the user-visible change.
- Ensure CI is green (`make ci` locally when possible).

## Code style

- **Go:** Clean Architecture under `apps/backend/internal/` — see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
- **Frontend:** lite FSD in `apps/mini-app`, `apps/admin`, `apps/landing` — see `frontend/README.md` and `.cursor/rules/frontend-fsd.mdc`.
- **UI:** cat design tokens — see [docs/DESIGN-CATS.md](docs/DESIGN-CATS.md).

## Questions

Open a [GitHub Issue](https://github.com/luckyrogue/lead-cat/issues) for bugs or feature ideas.
