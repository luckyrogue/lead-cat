# WS7a — Accessibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add axe-core accessibility scanning over the four front ends in the existing Playwright e2e harness, fix the critical/serious violations found, and have the scans run as a blocking CI gate.

**Architecture:** `@axe-core/playwright` runs axe inside loaded pages; a shared `expectNoA11yViolations` helper fails on critical/serious (reports moderate/minor). A11y specs live in `e2e/tests/*.a11y.spec.ts` and run in the existing `_e2e.yml` job (no new infra). Authenticated mini-app pages are reached via a test-only `stubTelegramWebApp` Playwright init-script + the e2e backend's existing `AUTH_DEV_MODE`. Fix-first: each task adds a surface's spec, runs discovery against the live stack, fixes the critical/serious violations it reports, and commits green.

**Tech Stack:** Playwright, `@axe-core/playwright`, docker-compose e2e stack (`deploy/docker-compose.e2e.yml` via `e2e/run.sh`), React Router front ends (admin, landing, mini-app).

## Global Constraints

- **Execution prerequisite:** these tasks require **Docker** (the compose e2e stack) running locally. Bring the stack up + run specs with `bash e2e/run.sh <spec-path...>` (it `docker compose up --build`s the stack, waits for readiness, runs `playwright test` with the passed args, and tears down on exit). The e2e Postgres/Redis are internal to the compose network (only mailpit `:8125` and apps `:8090/:8091/:8092` are host-exposed), so they do not collide with local dev DBs.
- **axe dependency** lives in the `e2e` workspace only: `cd e2e && pnpm add -D @axe-core/playwright` (e2e installs with `--ignore-workspace`, matching `_security.yml`).
- **Block threshold:** fail only on `impact ∈ {critical, serious}`. Always log ALL violations (every impact) before asserting, so CI output is actionable. `moderate`/`minor` are reported, never block, never fixed in 7a.
- **Fix-first, no baseline:** a task commits only when its a11y spec is GREEN (zero critical/serious) — so CI never goes red on merge. Fixes land in the responsible front-end app.
- **No production-code change for the TMA seam:** the mini-app auth bypass is purely the Playwright init-script (test code) + the pre-existing, production-guarded `AUTH_DEV_MODE` (already set in the e2e compose backend).
- **Fix scope is discovery-driven:** the exact violations are unknown until axe runs; fix steps apply the remediation axe reports (each violation carries its rule id, failing DOM node, and a help URL). Common fixes: associate `<label htmlFor>`/`aria-label`, give icon-only buttons an accessible name, set `<html lang>`, add `main`/`nav` landmarks/roles, raise text contrast to WCAG AA. Do NOT change behavior or copy — only accessibility attributes/markup/contrast.
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: axe tooling + a11y helper + landing & public-booking scans (+ fixes)

**Files:**
- Modify: `e2e/package.json` (+ `@axe-core/playwright`)
- Create: `e2e/helpers/a11y.ts`
- Create: `e2e/tests/landing.a11y.spec.ts`
- Create: `e2e/tests/booking.a11y.spec.ts`
- Modify (fixes, discovery-driven): `apps/landing/app/**` and `apps/admin/app/routes/book.$slug*` / related components

**Interfaces:**
- Produces: `expectNoA11yViolations(page, blockImpacts?)` (consumed by Tasks 2 & 3).

- [ ] **Step 1: Add the axe dependency**

```bash
cd e2e && pnpm add -D @axe-core/playwright && cd ..
```
Expected: `@axe-core/playwright` appears in `e2e/package.json` devDependencies; `e2e/pnpm-lock.yaml` updates.

- [ ] **Step 2: Create the a11y helper `e2e/helpers/a11y.ts`**

```ts
import AxeBuilder from "@axe-core/playwright"
import { expect, type Page } from "@playwright/test"

type Impact = "critical" | "serious" | "moderate" | "minor"

// expectNoA11yViolations scans the current page with axe-core, logs every
// violation grouped by impact, and fails the test if any violation's impact is
// in blockImpacts (default: critical + serious).
export async function expectNoA11yViolations(
  page: Page,
  label: string,
  blockImpacts: Impact[] = ["critical", "serious"],
): Promise<void> {
  const results = await new AxeBuilder({ page }).analyze()
  const violations = results.violations
  if (violations.length > 0) {
    const summary = violations
      .map(
        (v) =>
          `  [${v.impact}] ${v.id}: ${v.help} (${v.nodes.length} node(s))\n    ${v.helpUrl}`,
      )
      .join("\n")
    console.log(`a11y violations on ${label}:\n${summary}`)
  }
  const blocking = violations.filter(
    (v) => v.impact && (blockImpacts as string[]).includes(v.impact),
  )
  expect(
    blocking,
    `${label}: ${blocking.length} blocking a11y violation(s) — ${blocking
      .map((v) => `${v.impact}:${v.id}`)
      .join(", ")}`,
  ).toEqual([])
}
```

- [ ] **Step 3: Create `e2e/tests/landing.a11y.spec.ts`**

```ts
import { test } from "@playwright/test"
import { expectNoA11yViolations } from "../helpers/a11y"

test("landing home (ru) has no critical/serious a11y violations", async ({ page }) => {
  await page.goto("http://localhost:8091/")
  await expectNoA11yViolations(page, "landing /")
})

test("landing home (en) has no critical/serious a11y violations", async ({ page }) => {
  await page.goto("http://localhost:8091/en")
  await expectNoA11yViolations(page, "landing /en")
})
```

- [ ] **Step 4: Create `e2e/tests/booking.a11y.spec.ts`**

Reuse the org+event-type setup from `public-booking.spec.ts` (admin login → create org → `POST /api/booking/event-types` → slug), then scan the public page:

```ts
import { test } from "@playwright/test"
import { loginViaMagicLink } from "../helpers/auth"
import { expectNoA11yViolations } from "../helpers/a11y"

test("public booking page has no critical/serious a11y violations", async ({ page }) => {
  const email = `a11y-${Date.now()}@e2e.test`
  await loginViaMagicLink(page, email)
  await page.getByLabel("Organization name").fill(`A11y Org ${Date.now()}`)
  await page.getByRole("button", { name: "Create organization" }).click()
  await page.waitForURL((url) => url.pathname === "/")

  const orgsRes = await page.request.get("/api/orgs")
  const orgId: string = (await orgsRes.json()).organizations[0].id
  const csrf =
    (await page.context().cookies()).find((c) => c.name === "lc_csrf")?.value ?? ""
  const etRes = await page.request.post("/api/booking/event-types", {
    headers: { "X-Org-Id": orgId, "X-CSRF-Token": csrf },
    data: {
      title: "A11y Intro",
      description: "",
      duration_mins: 30,
      timezone: "Asia/Almaty",
      avail_weekdays: [1, 2, 3, 4, 5],
      avail_start_minute: 540,
      avail_end_minute: 1020,
      active: true,
    },
  })
  const slug: string = (await etRes.json()).slug

  await page.goto(`/book/${slug}`)
  await expectNoA11yViolations(page, `/book/${slug}`)
})
```

> The public booking page is served by the admin app at `:8090` (Playwright `baseURL`), so the relative `/book/${slug}` navigation works after the API setup.

- [ ] **Step 5: Discovery run — enumerate violations on the public surfaces**

Run: `bash e2e/run.sh landing.a11y.spec.ts booking.a11y.spec.ts`
Expected: the specs run; for any failing surface, the console logs each violation as `[impact] rule-id: help (N nodes) helpUrl`. Record the critical/serious list (rule id + surface) — this is the fix inventory for this task.

- [ ] **Step 6: Fix the critical/serious violations**

For each critical/serious violation reported in Step 5, apply the remediation in the responsible app (`apps/landing/app/**` for landing; `apps/admin/app/routes/book.$slug*` and the components it renders for booking). The violation's `id`/`helpUrl` names the fix; apply the minimal accessibility change (label/`aria-*`/role/`lang`/landmark/contrast) without altering behavior or copy. Re-run `bash e2e/run.sh landing.a11y.spec.ts booking.a11y.spec.ts` after each batch.

> If Step 5 reports zero critical/serious on a surface, that surface needs no fix — its spec already passes. The deliverable is still the committed spec (the gate).

- [ ] **Step 7: Verify green + frontend builds**

Run: `bash e2e/run.sh landing.a11y.spec.ts booking.a11y.spec.ts` (PASS), then `pnpm --filter ./apps/landing build && pnpm --filter ./apps/admin build` (the fixed apps still build).
Expected: both a11y specs PASS; both builds succeed.

- [ ] **Step 8: Commit**

```bash
git add e2e/package.json e2e/pnpm-lock.yaml e2e/helpers/a11y.ts e2e/tests/landing.a11y.spec.ts e2e/tests/booking.a11y.spec.ts apps/landing apps/admin
git commit -m "$(cat <<'EOF'
test(ws7a): axe a11y scans for landing + public booking; fix critical/serious

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

> Stage `apps/landing`/`apps/admin` only if Step 6 changed files there; if a surface had zero violations, stage only the e2e files. Never `git add -A`; run `git status --porcelain` first to stage exactly the touched paths.

---

### Task 2: admin authenticated scans (+ fixes)

**Files:**
- Create: `e2e/tests/admin.a11y.spec.ts`
- Modify (fixes, discovery-driven): `apps/admin/app/**`

**Interfaces:**
- Consumes: `expectNoA11yViolations` (Task 1), `loginViaMagicLink` (existing).

- [ ] **Step 1: Create `e2e/tests/admin.a11y.spec.ts`**

```ts
import { test } from "@playwright/test"
import { loginViaMagicLink } from "../helpers/auth"
import { expectNoA11yViolations } from "../helpers/a11y"

test("admin authed pages have no critical/serious a11y violations", async ({ page }) => {
  const email = `a11y-admin-${Date.now()}@e2e.test`

  await loginViaMagicLink(page, email)
  // Onboarding (pre-org) page.
  await expectNoA11yViolations(page, "admin /onboarding")

  await page.getByLabel("Organization name").fill(`A11y Admin ${Date.now()}`)
  await page.getByRole("button", { name: "Create organization" }).click()
  await page.waitForURL((url) => url.pathname === "/")
  await expectNoA11yViolations(page, "admin / (dashboard)")

  await page.goto("/meetings")
  await page.waitForURL(/\/meetings/)
  await expectNoA11yViolations(page, "admin /meetings")

  // Booking config page (route that lists/creates event types).
  await page.goto("/booking")
  await expectNoA11yViolations(page, "admin /booking")
})
```

> Confirm the booking-config route path during the run (the repo has `apps/admin/app/routes/_app.booking._index.tsx` → `/booking`). If the path differs, adjust the `goto` to the actual route; the run will 404 visibly if wrong.

- [ ] **Step 2: Discovery run**

Run: `bash e2e/run.sh admin.a11y.spec.ts`
Expected: logs critical/serious violations per admin page. Record the inventory.

- [ ] **Step 3: Fix the critical/serious violations in `apps/admin`**

Apply the axe-reported remediation per violation (label/`aria-*`/role/landmark/contrast), minimal and behavior-preserving. Re-run after each batch.

- [ ] **Step 4: Verify green + build**

Run: `bash e2e/run.sh admin.a11y.spec.ts` (PASS), then `pnpm --filter ./apps/admin build`.
Expected: spec PASS; admin builds.

- [ ] **Step 5: Commit**

```bash
git status --porcelain   # stage exactly the touched paths
git add e2e/tests/admin.a11y.spec.ts apps/admin
git commit -m "$(cat <<'EOF'
test(ws7a): axe a11y scans for admin authed pages; fix critical/serious

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 3: mini-app TMA-auth seam + authenticated scans (+ fixes); final gate verification

**Files:**
- Create: `e2e/helpers/tma.ts`
- Create: `e2e/tests/mini-app.a11y.spec.ts`
- Modify (fixes, discovery-driven): `apps/mini-app/app/**`

**Interfaces:**
- Consumes: `expectNoA11yViolations` (Task 1).
- Produces: `stubTelegramWebApp(page, telegramID)`.

- [ ] **Step 1: Create the TMA stub helper `e2e/helpers/tma.ts`**

```ts
import type { Page } from "@playwright/test"

// stubTelegramWebApp injects a fake Telegram WebApp object before the mini-app
// loads, so getInitData() returns `telegramID`. The e2e backend runs with
// AUTH_DEV_MODE=true (APP_ENV=development), where POST /api/auth/miniapp accepts a
// plain int64 telegram id as init_data and maps it to a dev user. Test-only; no
// production code path enables this.
export async function stubTelegramWebApp(page: Page, telegramID: number): Promise<void> {
  await page.addInitScript((id: number) => {
    ;(window as unknown as { Telegram: unknown }).Telegram = {
      WebApp: {
        initData: String(id),
        initDataUnsafe: { user: { id, first_name: "A11y" } },
        ready() {},
        expand() {},
        colorScheme: "light",
        openLink() {},
      },
    }
  }, telegramID)
}
```

- [ ] **Step 2: Create `e2e/tests/mini-app.a11y.spec.ts`**

```ts
import { test } from "@playwright/test"
import { expectNoA11yViolations } from "../helpers/a11y"
import { stubTelegramWebApp } from "../helpers/tma"

const MINI_APP = "http://localhost:8092"

test("mini-app authed pages have no critical/serious a11y violations", async ({ page }) => {
  // Unique dev telegram id per run (AUTH_DEV_MODE maps it to a dev user).
  const tgId = Math.floor(Date.now() / 1000)
  await stubTelegramWebApp(page, tgId)

  await page.goto(`${MINI_APP}/`)
  // Wait for the app shell to settle after the dev-auth round-trip.
  await page.waitForLoadState("networkidle")
  await expectNoA11yViolations(page, "mini-app /")

  // Key authed routes (adjust to actual paths confirmed during the run).
  await page.goto(`${MINI_APP}/meetings`)
  await page.waitForLoadState("networkidle")
  await expectNoA11yViolations(page, "mini-app /meetings")
})
```

> Discovery determines what renders after dev-auth. If the dev telegram id must be a
> registered `bot_user` for authed pages to load (rather than a "register in the bot"
> state), seed it: the simplest seam is an authenticated state the mini-app accepts —
> confirm during Step 3. If only the post-login shell/registration state renders,
> scan that state (still real coverage) and note the limitation. Adjust the route
> list to the mini-app's actual authed routes (e.g. `/`, `/meetings`, `/profile`).

- [ ] **Step 3: Discovery run — confirm reachable authed pages + enumerate violations**

Run: `bash e2e/run.sh mini-app.a11y.spec.ts`
Expected: confirms which mini-app pages render post-stub-auth; logs critical/serious violations. If a `goto` lands on an unexpected state, adjust the spec's routes to the real authed paths, then re-run. Record the inventory.

- [ ] **Step 4: Fix the critical/serious violations in `apps/mini-app`**

Apply axe-reported remediations (label/`aria-*`/role/landmark/contrast), minimal and behavior-preserving. Re-run after each batch.

- [ ] **Step 5: Full-suite verification (functional + all a11y) + builds**

Run: `bash e2e/run.sh` (no args → runs the ENTIRE e2e suite: existing functional specs + all four a11y specs), then `pnpm --filter ./apps/mini-app build`.
Expected: every spec PASS (the a11y specs are auto-included by the `_e2e.yml` job since it runs `playwright test` with no filter — so this also confirms the blocking gate is wired with no extra config). mini-app builds.

- [ ] **Step 6: Commit**

```bash
git status --porcelain   # stage exactly the touched paths
git add e2e/helpers/tma.ts e2e/tests/mini-app.a11y.spec.ts apps/mini-app
git commit -m "$(cat <<'EOF'
test(ws7a): TMA-auth e2e seam + mini-app axe a11y scans; fix critical/serious

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- `@axe-core/playwright` + `expectNoA11yViolations` helper (block critical/serious, log all) → Task 1 Steps 1-2. ✓
- Coverage of all four surfaces — landing, public booking, admin authed, mini-app authed → Tasks 1/2/3 specs. ✓
- TMA-auth e2e seam, test-only (init-script + existing AUTH_DEV_MODE), no prod change → Task 3 Step 1 + helper doc. ✓
- Fix-first, no baseline (commit only when green) → each task's discovery→fix→verify→commit order. ✓
- moderate/minor reported not fixed/blocked → helper logs all, blocks only critical/serious. ✓
- Blocking gate via existing `_e2e.yml` (auto-includes `*.a11y.spec.ts`) → Task 3 Step 5 confirms full-suite green + auto-inclusion. ✓
- Out of scope (7b/7c, moderate/minor, manual SR testing) → not in plan. ✓

**Placeholder scan:** The a11y specs and helpers are complete, runnable code. The FIX steps are intentionally discovery-driven — axe output is self-describing (rule id + node + help URL), so "apply the reported remediation" is a concrete, actionable instruction, not a TODO; common-fix examples are listed in Global Constraints. The only run-time adjustments flagged are the exact mini-app authed routes and the admin booking route path, which the discovery run confirms (the specs fail loudly if a path is wrong, never silently skip).

**Type consistency:** `expectNoA11yViolations(page, label, blockImpacts?)` and `stubTelegramWebApp(page, telegramID)` signatures are consistent between their definitions (Tasks 1 & 3) and all call sites (Tasks 1/2/3 specs). The booking-setup mirrors the proven `public-booking.spec.ts` API sequence (`/api/orgs`, `X-Org-Id`, `X-CSRF-Token`, `/api/booking/event-types`).

**Execution note:** every task requires Docker (the compose e2e stack). If the executing environment cannot run Docker, this plan cannot be implemented or verified there — flag and run where Docker is available.
