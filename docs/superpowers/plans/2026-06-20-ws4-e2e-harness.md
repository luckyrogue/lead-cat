# WS4-e2e — End-to-End Test Harness + Two Flows Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Playwright E2E suite running the full stack (calendar stub + Mailpit) that covers magic-link→org→meeting and public-booking→confirmation, locally and in CI.

**Architecture:** A `docker-compose.e2e.yml` reuses the existing backend + admin Dockerfiles (admin nginx proxies `/api`→backend); a `scripts/e2e.sh` builds+boots the stack, runs `@playwright/test`, tears down; a CI job does the same.

**Tech Stack:** Docker Compose, `@playwright/test`, Mailpit HTTP API, the existing Go backend (`CALENDAR_STUB`) + admin SPA.

## Global Constraints

- Repo root for the suite (`e2e/`), backend at `apps/backend`, admin at `apps/admin`. Spec: `docs/superpowers/specs/2026-06-20-ws4-e2e-harness-design.md`.
- **No external dependencies:** `CALENDAR_STUB=true` + Mailpit. No real Google/MS/Telegram creds.
- **Port isolation:** the E2E compose uses its own network; publish ONLY admin (`8090`) + mailpit-API (`8025`) to the host (the user's sadu containers hold 5432/6379 — postgres/redis stay internal, no host publish).
- No code comments in new TS files; Playwright specs may use minimal describe/step labels (not `//` noise).
- `.gitignore`: this slice is **explicitly authorized** to add E2E artifact dirs (`e2e/playwright-report/`, `e2e/test-results/`, `e2e/node_modules/`) ONLY — no other `.gitignore` changes.
- Work on `main`; never `git add -A`; **verify HEAD before each commit** (user commits in parallel — commit on top, never rebase/reset). Trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- **Verification is heavy:** each task is validated by actually running `scripts/e2e.sh` (or its relevant subset) — Docker image builds + Playwright chromium install + multi-minute boots. Budget for it.

**Reference (verified):**
- `apps/backend/Dockerfile` (Go API), `apps/admin/Dockerfile` (`ENV API_UPSTREAM=http://backend:8080`, nginx serves SPA + `proxy_pass $api_upstream`, `apps/admin/nginx.conf`).
- Infra `deploy/docker-compose.yml`: postgres:18-alpine, redis:8-alpine, mailpit (SMTP 1025 / API 8025).
- Backend env: `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `CALENDAR_STUB`, `APP_BASE_URL`, `SMTP_HOST/PORT/FROM`, `CORS_ALLOWED_ORIGINS`, `WEBAPP_URL`. Health: `GET /api/health`. Backend applies embedded migrations on boot.
- Magic-link email links to `{APP_BASE_URL}/api/auth/web/magic/verify?token=…` → set `APP_BASE_URL=http://admin` so it routes back through the admin (which proxies `/api`).
- Admin routes: `/login`, `/onboarding`, `/` (dashboard), `/meetings`, `/booking`, public `/book/:slug`.

---

### Task 1: E2E stack + Playwright harness + smoke

**Files:**
- Create: `deploy/docker-compose.e2e.yml`
- Create: `scripts/e2e.sh`
- Create: `e2e/package.json`, `e2e/playwright.config.ts`, `e2e/tsconfig.json`
- Create: `e2e/helpers/mailpit.ts`, `e2e/helpers/auth.ts`
- Create: `e2e/tests/smoke.spec.ts`
- Modify: root `package.json` — add an `e2e` script; `.gitignore` — E2E artifact dirs

- [ ] **Step 1: Compose** — `deploy/docker-compose.e2e.yml`:
  - `postgres` (POSTGRES_PASSWORD/DB), `redis`, `mailpit` (publish `8025:8025` for the runner; SMTP internal) — all with healthchecks.
  - `backend`: `build: { context: ., dockerfile: apps/backend/Dockerfile }`; env per the spec (`CALENDAR_STUB=true`, DB/Redis to the services, `JWT_SECRET=e2e-secret`, `SMTP_HOST=mailpit SMTP_PORT=1025 SMTP_FROM=e2e@leadcat.test`, `APP_BASE_URL=http://admin`, `WEBAPP_URL=http://admin`, `CORS_ALLOWED_ORIGINS=http://admin`); `depends_on` pg+redis healthy; healthcheck `wget -qO- http://localhost:8080/api/health` (or the binary's health path).
  - `admin`: `build: { context: ., dockerfile: apps/admin/Dockerfile }`; `environment: API_UPSTREAM=http://backend:8080`; `ports: ["8090:80"]`; `depends_on` backend healthy.
  - one network `e2e`.
  (Confirm the backend Dockerfile's healthcheck path + the exact admin nginx listen port `80`.)

- [ ] **Step 2: Runner script** — `scripts/e2e.sh` (bash, `set -euo pipefail`):
```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
compose="docker compose -f deploy/docker-compose.e2e.yml -p leadcat-e2e"
cleanup() { $compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT
$compose up -d --build
# wait for admin to serve + backend health (poll http://localhost:8090 and the proxied /api/health)
for i in $(seq 1 60); do
  if curl -fsS http://localhost:8090/ >/dev/null 2>&1 && curl -fsS http://localhost:8090/api/health >/dev/null 2>&1; then ready=1; break; fi
  sleep 3
done
[ "${ready:-}" = 1 ] || { echo "stack did not become ready"; $compose logs --tail=50; exit 1; }
( cd e2e && pnpm exec playwright test "$@" )
```

- [ ] **Step 3: Playwright project** — `e2e/package.json` (`@playwright/test` devDep, a `test` script), `e2e/tsconfig.json`, `e2e/playwright.config.ts`:
```ts
import { defineConfig } from "@playwright/test"
export default defineConfig({
  testDir: "./tests",
  timeout: 60_000,
  expect: { timeout: 15_000 },
  retries: process.env.CI ? 1 : 0,
  reporter: [["list"], ["html", { outputFolder: "playwright-report", open: "never" }]],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:8090",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
})
```

- [ ] **Step 4: Mailpit helper** — `e2e/helpers/mailpit.ts`:
```ts
const MAILPIT = process.env.E2E_MAILPIT_URL ?? "http://localhost:8025"

export async function getLatestMagicLink(email: string): Promise<string> {
  for (let i = 0; i < 30; i++) {
    const res = await fetch(`${MAILPIT}/api/v1/search?query=to:${encodeURIComponent(email)}`)
    if (res.ok) {
      const data = (await res.json()) as { messages?: { ID: string }[] }
      const id = data.messages?.[0]?.ID
      if (id) {
        const msg = await fetch(`${MAILPIT}/api/v1/message/${id}`)
        const body = (await msg.json()) as { Text?: string; HTML?: string }
        const m = (body.Text ?? "" + (body.HTML ?? "")).match(/https?:\/\/[^\s"'<>]*magic\/verify[^\s"'<>]*/)
        if (m) return m[0]
      }
    }
    await new Promise((r) => setTimeout(r, 1000))
  }
  throw new Error(`no magic link for ${email}`)
}
```
(Confirm Mailpit's API shape — `/api/v1/search` + `/api/v1/message/{id}` with `Text`/`HTML` fields; adjust to the running Mailpit version.)

- [ ] **Step 5: Auth helper** — `e2e/helpers/auth.ts`: `loginViaMagicLink(page, email)` — `page.goto("/login")`, fill the email field, submit, `const link = await getLatestMagicLink(email)`, `await page.goto(link)`, await navigation away from `/login`.

- [ ] **Step 6: Smoke spec** — `e2e/tests/smoke.spec.ts`: `test("login page renders", async ({ page }) => { await page.goto("/login"); await expect(page.getByRole(...)).toBeVisible() })` (match an actual element on the login page).

- [ ] **Step 7: Root wiring** — root `package.json`: `"e2e": "bash scripts/e2e.sh"`. `chmod +x scripts/e2e.sh`. Add to `.gitignore`: `e2e/playwright-report/`, `e2e/test-results/`, `e2e/node_modules/`.

- [ ] **Step 8: RUN IT** — install browsers (`cd e2e && pnpm install && pnpm exec playwright install chromium`), then `pnpm e2e tests/smoke.spec.ts`. Expected: stack builds + boots, the smoke test passes, teardown runs. If the stack won't boot, debug via `docker compose -f deploy/docker-compose.e2e.yml -p leadcat-e2e logs`. Do NOT proceed until the smoke test is green.

- [ ] **Step 9: Commit**
```bash
git add deploy/docker-compose.e2e.yml scripts/e2e.sh e2e/package.json e2e/playwright.config.ts e2e/tsconfig.json e2e/helpers e2e/tests/smoke.spec.ts package.json .gitignore
# include e2e/pnpm-lock.yaml if generated and intended to be committed
git commit -m "test(e2e): Playwright harness + compose stack (calendar stub + mailpit) + smoke"
```

---

### Task 2: Flow 1 — auth → org → meeting

**Files:**
- Create: `e2e/tests/admin-core.spec.ts`

- [ ] **Step 1:** `loginViaMagicLink(page, "owner-"+Date.now()+"@e2e.test")` → expect `/onboarding` (no org). Inspect the real onboarding/dashboard/meetings DOM (run the app, or read the components: `routes/onboarding.tsx`, `features/dashboard`, `features/meetings`) for stable selectors (prefer `getByRole`/`getByLabel`/`getByText` over CSS).
- [ ] **Step 2:** Fill the org name, submit → expect the dashboard (`/`). Nav to `/meetings`.
- [ ] **Step 3:** Open the "New meeting" dialog → fill the required fields (dept, type, host, date, start/end, a participant email) → submit → assert the created meeting appears in the table (text match on the meeting name / a row).
- [ ] **Step 4: RUN** `pnpm e2e tests/admin-core.spec.ts` until green (calendar stub makes create succeed). Commit:
```bash
git add e2e/tests/admin-core.spec.ts
git commit -m "test(e2e): flow 1 — magic-link auth -> create org -> create meeting"
```

---

### Task 3: Flow 2 — public booking

**Files:**
- Create: `e2e/tests/public-booking.spec.ts`

- [ ] **Step 1:** Log in (reuse the helper) → nav to `/booking` → "New event type" → fill (title, 30 min, weekdays Mon–Fri, 09:00–17:00, active) → save → read the `/book/{slug}` link from the row (the page renders `${origin}/book/${slug}`).
- [ ] **Step 2:** `const ctx = await browser.newContext()` (fresh, unauthenticated) → `const visitor = await ctx.newPage()` → `visitor.goto(bookSlugUrl)` → expect the event title + a day with slots → click a day → click a slot → "Continue" → fill visitor name + email → submit.
- [ ] **Step 3:** Assert the confirmation panel ("You're booked!" + a Join link). Close the context.
- [ ] **Step 4: RUN** `pnpm e2e tests/public-booking.spec.ts` until green. Commit:
```bash
git add e2e/tests/public-booking.spec.ts
git commit -m "test(e2e): flow 2 — public /book/:slug booking -> confirmation"
```

---

### Task 4: CI job

**Files:**
- Create: `.github/workflows/_e2e.yml`
- Modify: `.github/workflows/ci.yml` — call the e2e job on `pull_request`

- [ ] **Step 1:** `_e2e.yml` (reusable `workflow_call`): runs-on ubuntu; checkout; setup pnpm + Node; `cd e2e && pnpm install && pnpm exec playwright install --with-deps chromium`; `bash scripts/e2e.sh`; on failure `upload-artifact` `e2e/playwright-report` + `e2e/test-results`. (Docker is available on ubuntu runners.)
- [ ] **Step 2:** In `ci.yml`, add an `e2e` job `uses: ./.github/workflows/_e2e.yml` gated `if: github.event_name == 'pull_request'` (mirror the existing `docker-validate` gating).
- [ ] **Step 3:** Validate the workflow YAML (`actionlint` if available, else careful review). The real green-on-CI proof comes from a PR run; if the runner can't build the images in time, note the fallback: gate the job behind `workflow_dispatch` first, then flip to PR. Commit:
```bash
git add .github/workflows/_e2e.yml .github/workflows/ci.yml
git commit -m "ci(e2e): run Playwright E2E (compose stack) on PRs + upload traces"
```

---

### Task 5: Whole-slice verification

**Files:** none

- [ ] **Step 1: Full run** — `pnpm e2e` (both specs + smoke) green end-to-end via `scripts/e2e.sh`; teardown leaves no leftover containers (`docker ps` clean; `docker compose -p leadcat-e2e ps` empty).
- [ ] **Step 2: No port clash** — confirm the E2E compose published only 8090 + 8025 and did not collide with the user's sadu 5432/6379 (the run succeeded → no clash).
- [ ] **Step 3: Artifacts gitignored** — `git status` shows no `playwright-report`/`test-results` staged; the only `.gitignore` change is the three E2E dirs.
- [ ] **Step 4: Tree clean** — verify HEAD; no stray files.

---

## Notes for the executor

- **Run the stack for real at every task** — E2E is verified by execution, not by reasoning. Budget for image builds (Go + pnpm) + chromium install + multi-minute boots.
- **Selectors:** prefer role/label/text; read the actual admin components for stable handles; avoid brittle CSS.
- **Mailpit API**: confirm the running version's endpoints (`/api/v1/search`, `/api/v1/message/{id}`) + field names; adjust the helper to what the container actually returns.
- **`APP_BASE_URL=http://admin`** is load-bearing for the magic-link round-trip (link + cookie domain align through the admin proxy).
- **Isolated ports** (only 8090 + 8025 published) avoid the sadu 5432/6379 clash.
- **`.gitignore`** change is scoped to the three E2E artifact dirs only (explicitly authorized).
- **Deferred:** mini-app flows, invites/join-request E2E, real-credential integration, visual/a11y.
```
