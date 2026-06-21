# WS7a — Accessibility (design)

## Context: WS7 hardening, sub-project a of 3

WS7 (the last public-launch hardening workstream) covers accessibility, load/perf, and
pentest — three independent efforts decomposed into **7a (a11y, this spec)**, 7b
(load), 7c (pentest/DAST). 7a is first: highest public-launch value, and it rides the
existing Playwright e2e harness.

The repo already has an e2e harness (`e2e/`, Playwright + chromium) driven by a
docker-compose stack (`deploy/docker-compose.e2e.yml`) that builds and serves all four
front ends — admin (`:8090`), landing (`:8091`), mini-app (`:8092`) — plus backend,
Postgres, Redis, Mailpit. Auth helpers exist (`loginViaMagicLink` via Mailpit). No
accessibility tooling exists yet.

## Problem

The public-facing web surfaces have never been checked for accessibility (WCAG)
violations — missing form labels, insufficient contrast, missing ARIA/roles, keyboard
traps, etc. For a public launch this is a real gap across landing, the public booking
page, the admin web app, and the Telegram mini-app.

## Goal

Add automated accessibility scanning (axe-core) over the four front ends in the
existing e2e harness, **fix the critical/serious violations found**, and turn the scan
into a blocking CI gate so new critical/serious violations cannot merge. Moderate/minor
findings are reported but do not block.

## Design

### Tooling

`@axe-core/playwright` runs the axe-core engine inside a loaded Playwright page and
returns violations grouped by `impact` (`critical` | `serious` | `moderate` | `minor`).
A shared helper drives every scan:

```
expectNoA11yViolations(page, blockImpacts = ["critical", "serious"])
  → const results = await new AxeBuilder({ page }).analyze()
  → log all violations (grouped by impact) for visibility
  → assert zero violations whose impact ∈ blockImpacts
```

A11y specs live in `e2e/tests/*.a11y.spec.ts` and run in the existing `_e2e.yml` CI job
(same compose stack, no new infra). They are additive to the functional e2e specs.

### Coverage — four surfaces

- **landing** (unauth): `/` and `/en` (`:8091`).
- **public booking** (unauth): `/book/:slug` — seed an event type via the API setup
  already used in `public-booking.spec.ts` (admin login → `POST /api/booking/event-types`
  → use the returned slug), then scan the public page.
- **admin** (authed via `loginViaMagicLink`): `/onboarding`, `/` (dashboard),
  `/meetings`, and the booking-config page — the key authenticated routes.
- **mini-app** (authed via the new TMA stub below): the key authenticated pages
  reachable after a stubbed Telegram session (home/meetings/profile-settings; the
  discovery task confirms which render).

### TMA-auth e2e seam (no production-code change)

Reaching authenticated mini-app pages needs a Telegram session, which the prod-built
mini-app cannot get in e2e. The seam is **test-only**:

- The e2e backend already runs `AUTH_DEV_MODE=true` + `APP_ENV=development`; in that
  mode `POST /api/auth/miniapp` accepts a plain int64 telegram id as `init_data` and
  maps it to a dev user (`miniapp_auth.go`). `AUTH_DEV_MODE` is hard-blocked when
  `APP_ENV=production` (`config.go`), so this path cannot exist in real prod.
- A Playwright helper `stubTelegramWebApp(page, telegramID)` uses `page.addInitScript`
  to set `window.Telegram = { WebApp: { initData: "<id>", initDataUnsafe: {...}, ready(){}, expand(){}, colorScheme: "light" } }` **before** the mini-app loads. The
  prod-built `getInitData()` reads `window.Telegram.WebApp.initData`, posts it, and the
  dev-mode backend authenticates it.
- If the dev telegram id must correspond to a registered `bot_user` (with org) for
  authenticated pages to render, the discovery task seeds one (via the admin/miniapp
  API or a DB insert in the compose stack) — the minimal seeding needed is determined
  from what the mini-app requires after auth.

No mini-app/frontend production source changes: the only "bypass" is the pre-existing,
production-guarded `AUTH_DEV_MODE` plus the Playwright init-script (test code).

### Fix-first (no baseline allowlist)

Because the violation set is unknown until axe runs against the live stack, the plan
front-loads a **discovery** step (bring up the compose stack, run axe across all four
surfaces, enumerate violations by impact/surface). Then:
- **Fix** every `critical`/`serious` violation in the responsible front-end app
  (label/`aria-*`/role/contrast/landmark/name fixes — concrete edits sized from
  discovery).
- Re-run until the a11y specs are green with zero critical/serious.
- The green specs then become the blocking gate (no baseline/allowlist — fix-first).

`moderate`/`minor` violations are logged in the spec output and recorded as known a11y
debt, not fixed in 7a.

### Gate

A11y specs run in the existing `_e2e.yml` job; a critical/serious violation fails the
job and blocks the PR, exactly like a functional e2e failure.

## Out of scope

- **7b** (load/perf) and **7c** (pentest/DAST) — separate sub-projects.
- `moderate`/`minor` a11y violations — reported, not fixed in 7a.
- Non-web surfaces; visual-design/contrast-theme overhauls beyond the specific
  contrast violations axe flags.
- Screen-reader manual testing (axe is automated static analysis; it catches a large
  but not exhaustive subset of WCAG).

## Error handling / fallbacks

- The a11y helper always logs the full violation list (all impacts) before asserting,
  so failures are actionable in CI output.
- If a surface is unreachable in the stack (e.g. mini-app auth seeding incomplete), the
  spec fails loudly rather than silently skipping — no false green.

## Testing

The deliverable *is* tests, plus the fixes that make them pass:
- Per surface, an `*.a11y.spec.ts` that navigates to the page(s) (using existing
  auth/seed helpers + the new `stubTelegramWebApp`) and calls `expectNoA11yViolations`.
- The discovery step's enumeration is captured (committed as a short findings note or
  the spec output) so the fix scope is traceable.
- Final state: `pnpm`-built front ends + the e2e a11y specs pass with zero
  critical/serious violations across landing, booking, admin, and mini-app; the
  existing functional e2e + backend/frontend suites remain green.
