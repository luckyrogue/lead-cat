# WS4-e2e — End-to-End Test Harness + Two Flows (design)

**Date:** 2026-06-20
**Status:** approved — ready for implementation plan
**Part of:** Public-launch hardening, **WS4 (E2E)**.

## Goal

Stand up a Playwright E2E suite that runs the full stack with external services stubbed
(calendar stub + Mailpit), covering two critical journeys — (1) magic-link auth → create org →
create meeting, and (2) public booking → confirmation — runnable locally and in CI.

## Decisions (from brainstorming)

- **Tool:** `@playwright/test` (committed suite), not the interactive MCP.
- **Flows:** both — core admin journey AND public booking — in this slice.
- **Run target:** local + CI now.
- **Frontend:** the built admin SPA served by its nginx image (proxies `/api`→backend), the prod
  model — Playwright hits the admin origin.
- **No external deps:** `CALENDAR_STUB=true` (fake Meet links) + Mailpit (magic-link capture).

## Background — verified current state

- Per-service Dockerfiles exist: `apps/backend/Dockerfile`, `apps/admin/Dockerfile` (nginx serves
  the SPA + `proxy_pass $api_upstream` where `API_UPSTREAM=http://backend:8080`,
  `apps/admin/nginx.conf`).
- Infra compose `deploy/docker-compose.yml`: Postgres (5432), Redis (6379), **Mailpit** (SMTP
  1025, HTTP API 8025). NOTE the user's local "sadu" containers also grab 5432/6379 — the E2E
  compose must use its own network and either internal-only or distinct published ports.
- Backend env: `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `CALENDAR_STUB`, `APP_BASE_URL`,
  `SMTP_HOST`/`SMTP_PORT`/`SMTP_FROM`, `CORS_ALLOWED_ORIGINS`, `WEBAPP_URL`.
- Magic-link: `POST /api/auth/web/magic/request` emails a link to
  `{APP_BASE_URL}/api/auth/web/magic/verify?token=…`; the GET sets the session cookie + redirects
  post-login. So `APP_BASE_URL` must be the **admin origin** so the link routes back through the
  admin (which proxies `/api`→backend) and the cookie domain matches.
- WS5-rl just added rate-limits on `/api/book/:slug` (10/h) + magic-request (5/15min) — a single
  E2E booking + a few magic requests stay well under.
- No existing Playwright/e2e — this slice creates it.

## Design

### A. E2E stack — `deploy/docker-compose.e2e.yml`

Services (own network `e2e`, no host-port clashes with sadu — publish only the admin port):
- `postgres` (18-alpine), `redis` (8-alpine), `mailpit` (SMTP 1025 + API 8025) — internal.
- `backend` (build `apps/backend/Dockerfile`): env `CALENDAR_STUB=true`,
  `DATABASE_URL=postgres://…@postgres`, `REDIS_URL=redis://redis:6379/0`, `JWT_SECRET=e2e-secret`,
  `SMTP_HOST=mailpit SMTP_PORT=1025 SMTP_FROM=e2e@leadcat.test`,
  `APP_BASE_URL=http://admin`, `WEBAPP_URL=http://admin`, `CORS_ALLOWED_ORIGINS=http://admin`.
  (Migrations run on boot — the backend embeds + applies them.) Healthcheck on `/api/health`.
- `admin` (build `apps/admin/Dockerfile`): `API_UPSTREAM=http://backend:8080`; publishes
  `8090:80` (the origin Playwright targets). Depends-on backend healthy.

`pnpm e2e` (root script) = `docker compose -f deploy/docker-compose.e2e.yml up -d --build` → wait
for the admin/health to be ready → `playwright test` → `docker compose … down -v` (always, via a
wrapper script `scripts/e2e.sh` so teardown runs on failure too).

### B. Playwright project — `e2e/`

- `e2e/playwright.config.ts`: `baseURL = process.env.E2E_BASE_URL ?? "http://localhost:8090"`,
  one chromium project, trace `on-first-retry`, screenshot/video `retain-on-failure`, a sane
  `webServer`-less setup (the stack is booted by `scripts/e2e.sh`, not Playwright). Output
  `e2e/playwright-report` + `e2e/test-results` (gitignored).
- `e2e/helpers/mailpit.ts`: `getLatestMagicLink(email)` — poll `http://localhost:8025/api/v1/...`
  (the Mailpit API base from `E2E_MAILPIT_URL`) for the newest message to `email`, fetch its
  text/HTML, regex-extract the `…/api/auth/web/magic/verify?token=…` URL. (Mailpit API is exposed
  to the host on 8025 by the compose for the test runner to read.)
- `e2e/helpers/auth.ts`: `loginViaMagicLink(page, email)` — go to `/login`, fill email, submit,
  `getLatestMagicLink`, `page.goto(link)`, assert landed authed.
- A `package.json` for `e2e/` (or root devDeps) with `@playwright/test`; `pnpm exec playwright
  install --with-deps chromium` in CI.

### C. Flow 1 — `e2e/tests/admin-core.spec.ts`

`loginViaMagicLink(page, "owner+{ts}@e2e.test")` (unique email per run) → expect redirect to
onboarding (no org) → fill org name → create → expect dashboard → nav to Meetings → "New
meeting" → fill the form (dept/type/host/date/time, a participant email) → submit → assert the
meeting row appears in the table (calendar stub returns a fake Meet link, so create succeeds).

### D. Flow 2 — `e2e/tests/public-booking.spec.ts`

Reuse the authed context (or a fresh login) → nav to Booking → "New event type" → fill
(title, 30 min, Mon–Fri, 09:00–17:00) → save → read the event's `/book/{slug}` link from the row
→ **open it in a fresh unauthenticated browser context** (`browser.newContext()`) → the page
loads the event + slots (stub host has no external busy → the 09:00–17:00 window yields slots) →
pick a day + slot → Continue → fill visitor name + email → submit → assert the confirmation panel
shows "You're booked!" + a Join link (stub Meet URL). (One booking — under the 10/h rate limit.)

### E. CI — `.github/workflows`

Add an `e2e` job (a reusable `_e2e.yml` called from `ci.yml` on `pull_request`): checkout →
setup Node/pnpm → `pnpm install` → `pnpm exec playwright install --with-deps chromium` →
`scripts/e2e.sh` (build+up the compose, run tests, down) → on failure, upload
`e2e/playwright-report` + traces as artifacts. Runs alongside the existing build/docker jobs.

### F. .gitignore

Add `e2e/playwright-report/`, `e2e/test-results/`, `e2e/node_modules/` to the repo `.gitignore`
**(this slice is explicitly authorized to touch `.gitignore` for these E2E artifact paths only).**

## Testing / verification

- `scripts/e2e.sh` builds the images, boots the stack, runs both specs green, tears down. Local
  run is the acceptance check (Docker + Playwright browsers required).
- A trivial smoke spec (`e2e/tests/smoke.spec.ts`: GET `/login` renders) proves the harness before
  the full flows.
- CI: the `e2e` job passes on a PR (or, if CI runner constraints bite, the job is present + green
  on a manual/`workflow_dispatch` first, then enabled on PR — the plan notes the fallback).
- **Verification is heavy** (image builds + browser install + multi-minute boots); each task is
  validated by actually running `scripts/e2e.sh` (or the relevant subset).

## Risks & mitigations

- **Port conflicts** with the user's sadu containers (5432/6379). *Mitigation:* the E2E compose
  uses an isolated network and publishes ONLY the admin (8090) + mailpit-api (8025) ports;
  postgres/redis stay internal (no host publish).
- **Flakiness / timing** (stack readiness, async magic-link email, slot rendering). *Mitigation:*
  health-gated `depends_on`, Playwright auto-waiting + explicit `expect.poll` for the Mailpit
  email, generous timeouts, trace-on-retry for debugging.
- **Magic-link cookie/origin correctness** (`APP_BASE_URL`=admin so the link + cookie domain
  align). *Mitigation:* set in the compose; the smoke/Flow-1 run validates it end to end.
- **CI runtime + image-build cost.** *Mitigation:* build with layer caching; chromium-only; the
  e2e job runs in parallel with others.
- **Booking slots depend on the stub host having availability.** With no calendar connection,
  external busy is empty → the configured window yields slots (correct). *Mitigation:* asserted
  by Flow 2.

## Done criteria

- `deploy/docker-compose.e2e.yml` (reuses backend+admin Dockerfiles, CALENDAR_STUB+Mailpit,
  isolated ports) + `scripts/e2e.sh` + root `pnpm e2e`.
- `e2e/` Playwright project (config + Mailpit/auth helpers + smoke + the two flow specs), both
  flows green locally via `scripts/e2e.sh`.
- CI `e2e` job (build → up → playwright → down, artifacts on failure).
- `.gitignore` updated for E2E artifact dirs only.
- More flows (mini-app, invites/join-requests, calendar-connect), real-credential integration,
  and visual/a11y testing explicitly deferred to follow-up slices.
```
