# Alpha smoke test

Automated E2E: `make smoke` (`go test -tags=smoke ./test/smoke/...`) against a running server — covers health, auth/me, workspace CRUD, ACL/IDOR, scenario create + run. See [REQUIREMENTS.md](REQUIREMENTS.md) §N-B4.

Manual alpha checklist (expected):

1. Health OK
2. Login at `/login` (or `AUTH_DEV_MODE` / `VITE_AUTH_DEV_MODE` for local)
3. Workspace CRUD + chat link
4. Scenario test run → `success` in runs (requires a linked notify chat)
5. `/test` in Telegram
6. Two workspaces → isolated chat notifications
