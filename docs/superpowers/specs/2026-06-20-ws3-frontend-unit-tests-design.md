# WS3 — Frontend Unit Tests (design)

**Part of:** Public-launch hardening (workstream 3 of 7). WS1 wired the CI gates and
explicitly deferred a frontend **test** gate to WS3 because no frontend test suites
existed. WS2 (a–d) gave the Go backend fast, dependency-free unit coverage. WS3 is the
frontend equivalent — and the lone remaining gap in the test matrix.

## Problem

The frontend workspace (`apps/mini-app`, `apps/admin`, `apps/landing`, `packages/ui`)
has **no test runner**. CI runs `typecheck`, `lint`, and `build` only
(`_build.yml` → `pnpm turbo run typecheck lint build --filter=./apps/*`). Pure logic —
date/time math, name formatting, series grouping, the activation checklist, i18n
resolution, SEO meta generation — is unverified and can silently regress.

## Goal

Wire **Vitest** into the monorepo as a cached turbo `test` task and a red-on-fail PR
gate, then cover the pure-logic helpers across all four packages with fast, DOM-free
unit tests. This mirrors WS2's approach: test pure functions and orchestration, not
framework internals.

## Scope

**In scope**

- Vitest runner wired per package (approach A below), a turbo `test` task, and a CI gate.
- Unit tests for **pure functions only** — deterministic input → output, no DOM, no
  network, no real timers.

**Out of scope** (later workstreams / future slices)

- Component / DOM tests (React Testing Library, jsdom). Deliberately excluded to keep
  this slice fast and low-maintenance; a future slice can add jsdom if a component
  warrants it.
- `apps/admin/.../checklist-dismissed.ts` — `window.localStorage`-bound; in a node
  environment only its `typeof window === "undefined"` fallback branch is reachable, so
  it carries no testable logic here. Deferred to a future jsdom-based slice.
- `apps/landing/.../motion.ts` — GSAP/`useEffect`/`matchMedia` hooks; DOM-bound, not
  node-testable.
- E2E (WS4), accessibility/security/load (WS5–WS7).

## Approach: per-app standalone Vitest config (approach A)

Each app and `packages/ui` gets a **standalone** `vitest.config.ts` (~6 lines) using the
`vite-tsconfig-paths` plugin to resolve the `~/*` alias and `@leadcat/*` workspace
imports from the package's existing `tsconfig.json` — no manual alias upkeep.

- `test.environment: 'node'` — every target is a pure function; no jsdom needed.
- **Standalone**, i.e. not reusing each app's react-router Vite config, so the
  react-router plugin does not interfere with test runs.
- No shared base config in `@leadcat/config` yet: each package still needs its own
  tsconfig-paths resolution, so the shareable part is thin. Per the repo's
  "extract after the second real duplication" rule, keep configs local until a third
  package proves the duplication. (Rejected alternatives: a shared `vitest.base.ts` —
  premature; a single root `projects` config — couples all packages and fights turbo's
  per-package caching.)

Reference shape (per package):

```ts
// vitest.config.ts
import { defineConfig } from "vitest/config"
import tsconfigPaths from "vite-tsconfig-paths"

export default defineConfig({
  plugins: [tsconfigPaths()],
  test: { environment: "node", include: ["**/*.test.ts"] },
})
```

## Wiring

1. **`turbo.json`** — add a `test` task. No `dependsOn: ["^build"]`: the targets import
   only types (erased at compile) and static data, so tests need no prior build. Cache
   enabled; `outputs: []`.
2. **Per-package `package.json`** — add `"test": "vitest run"` and devDependencies
   `vitest` + `vite-tsconfig-paths` to `apps/mini-app`, `apps/admin`, `apps/landing`,
   and `packages/ui`.
3. **`_build.yml`** — append `test` to the frontend job's turbo line and widen the
   filter so `packages/ui` runs too (the current `--filter=./apps/*` excludes it, and
   `test` has no `^test` dependency that would pull it in transitively):
   `pnpm turbo run typecheck lint build test --filter=./apps/* --filter=./packages/ui`.
   This makes a failing frontend test fail the PR — closing the gate WS1 deferred.
   (`typecheck`/`lint`/`build` for `packages/ui` already run transitively via the apps'
   `^build`; adding it to the filter is harmless and makes its `test` explicit.)

## Test targets

One `*.test.ts` colocated next to each source file (FSD colocation, matching the
backend's `_test.go` convention). Pure logic only; behavior, not snapshots.

### apps/mini-app

- `app/shared/lib/format.ts` — `formatDate` / `formatDateLong` (locale output +
  invalid-iso passthrough), `formatTimeRange` (empty start, with/without end),
  `addDaysIso` (month rollover, invalid-iso → today branch is non-deterministic, so
  assert only the deterministic valid-iso cases).
- `app/shared/lib/display-name.ts` — `getGreetingName` (telegram-first-name wins,
  FIO given-name = second token, single token, empty → "").
- `app/features/meetings/lib/group-series.ts` — `groupBySeries` (first-seen series order
  preserved, singles appended after series, mixed input, empty input).

### apps/admin

- `app/features/dashboard/lib/checklist-steps.ts` — `computeSteps` (each step's done
  condition: calendar = any connected; invite = `invitesCount > 0` OR `membersCount > 1`;
  meeting = `meetingsCount > 0`), `allDone`, `doneCount`.
- `app/features/meetings/lib/format.ts` and `app/features/meetings/lib/group-series.ts` —
  same shapes as mini-app; assert behavior against the actual exports.

### packages/ui

- `src/lib/date.ts` — `parseIsoDate` / `formatIsoDate` round-trip + invalid input →
  `undefined`; `parseTimeValue` (valid, empty, garbage → `{0,0}`); `formatTimeValue`
  padding; `addMinutesToTime` (wrap past midnight, negative wrap); `timeToMinutes`;
  `diffMinutes` (positive delta, non-positive → `DEFAULT_MEETING_DURATION_MIN`);
  `buildTimeOptions` (count for default + custom step); `dateFnsLocaleFromCode` /
  `dayPickerLocaleFromCode` (ru/kk → ru, else en).

### apps/landing

- `app/shared/i18n/locale-path.ts` — `localePath` (default locale → `/` + hash,
  non-default → `/{locale}` + hash).
- `app/shared/i18n/locale-request.ts` — `parseUrlLocale` (valid url-locale, invalid,
  undefined → null), `localeCookieHeader` (cookie string shape).
- `app/shared/i18n/translate.ts` — `translate` (nested-key lookup, fallback dict when
  missing in active, key returned when absent in both, `{param}` interpolation incl.
  missing param left as `{param}`).
- `app/shared/seo/landing-meta.ts` — pass an explicit `siteUrl` for determinism:
  `landingCanonicalPath`, `sitemapXml` / `robotsTxt` (absolute-URL composition, expected
  string structure), and a focused `landingMeta` / `landingJsonLd` assertion (canonical
  href, `html.lang`, hreflang alternates, JSON-LD `@graph` URLs).

## Testing (of this change)

`pnpm turbo run test` passes locally and in CI across all four packages; intentionally
breaking one helper turns the suite — and the PR — red. `typecheck`, `lint`, `build`
remain green.

## Out-of-scope follow-ups (noted, not built)

- jsdom + React Testing Library slice for component and `localStorage`-bound logic.
- Coverage thresholds / reporting (add once suites stabilize).
