# Alpha smoke test

See [PLAN.md](../PLAN.md) Smoke E2E section.

Expected:

1. Health OK
2. Login at `/login` (or `AUTH_DEV_MODE` / `VITE_AUTH_DEV_MODE` for local)
3. Workspace CRUD + chat link
4. Scenario test run → `success` in runs
5. `/test` in Telegram
6. Two workspaces → isolated chat notifications

Record failures in PLAN.md Blockers.
