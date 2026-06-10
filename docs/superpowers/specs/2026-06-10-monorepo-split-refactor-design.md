# Monorepo split + full refactor — architecture & migration design

**Status:** proposal (awaiting user approval before any execution).
**Date:** 2026-06-10
**Goal:** Restructure lead-cat into a monorepo with separately-deployed services — preserve the Go backend business logic (refactored to enforced clean architecture), delete the frontend to zero and rebuild it as a unified workspace of cute, 3D-accented apps.

## Locked decisions (from brainstorm)

| # | Decision |
|---|----------|
| 1 | **Monorepo**, single frontend workspace. |
| 2 | **Preserve** Go backend business logic (meetings, Google Calendar, reminders, series, Phase 0 web SSO + magic-link + multi-tenant orgs); refactor — do NOT rewrite. |
| 3 | **Delete the entire current frontend to zero**; rebuild as separate apps. |
| 4 | **Merge `feat/saas-phase0-tenancy-web-auth` → main first**, carry auth/orgs into the new backend. |
| 5 | Services (Dokploy): `apps/backend` (Go), `apps/landing` (marketing, 3D), `apps/admin` (dashboard, 3D), `apps/mini-app` (Telegram, lightweight). |
| 6 | Shared packages: `packages/ui`, `packages/api-client` (from OpenAPI), `packages/types`, `packages/config`. |
| 7 | Frontend sharing = **build-time workspace packages, NOT runtime Vite module federation** (revisit federation only if runtime composition is ever needed). |
| 8 | Backend fan-in/fan-out + clean-arch dependency rules via **`depguard`** (already in golangci-lint) — no new tool. |
| 9 | **Remove ALL code comments**; relax linter doc-comment rules (revive/godot) so lint stays green. |
| 10 | Cute 3D stack on **landing + admin**: three.js + @react-three/fiber + drei, pixi.js, motion. **mini-app stays lightweight** (no three.js/pixi — TG WebView perf). |

## Target monorepo layout

```
lead-cat/
├── apps/
│   ├── backend/            # Go service (moved from ./backend, refactored)
│   ├── admin/              # React+Vite+TS+Tailwind — web dashboard, tasteful 3D
│   ├── landing/            # React+Vite+TS+Tailwind — marketing, heavy 3D
│   └── mini-app/           # React+Vite+TS+Tailwind — Telegram Mini App, lite
├── packages/
│   ├── ui/                 # cute design system: tailwind preset + components + motion;
│   │                       #   3D primitives behind a SEPARATE entry (@leadcat/ui/3d)
│   ├── api-client/         # typed client generated from backend OpenAPI + RQ hooks
│   ├── types/              # shared TS types/enums (re-export generated + hand types)
│   └── config/             # base tsconfig, eslint, tailwind preset, prettier
├── deploy/                 # per-service Dockerfiles + Dokploy/compose
├── docs/
├── package.json            # npm workspaces root: ["apps/*","packages/*"]
└── go.work?                # optional Go workspace if backend ever splits modules
```

**Frontend tooling:** npm workspaces (already on npm — no new package manager). Root scripts fan out per app. Turborepo/pnpm deliberately NOT added now (only-existing-libs); can add Turbo later purely for task caching if builds get slow.

**3D isolation:** three.js/pixi/@react-three/* live ONLY in `packages/ui`'s `/3d` subpath export and in landing/admin. `mini-app` imports `@leadcat/ui` (non-3D) only, so its bundle never pulls three.js. 3D scenes are dynamically imported (`React.lazy`) so even landing/admin defer the heavy chunk.

## Backend clean architecture + fan-in/fan-out (depguard)

Layers (dependencies point inward): `delivery`/`infrastructure` → `application` → `domain`.

**depguard ruleset (in `.golangci.yml`)** — encodes fan-in/fan-out by denying outward/cross imports:

| Layer (files) | Denied imports (deny-list) |
|---|---|
| `internal/domain/**` | Fiber, pgx, asynq, telegram, AND `internal/{application,infrastructure,delivery,platform}` — domain depends on nothing (max fan-in, zero fan-out). |
| `internal/application/**` | Fiber, pgx, telegram concretes, `internal/delivery`, `internal/infrastructure/**` — application depends on `domain` + its OWN port interfaces only. |
| `internal/delivery/**` | pgx, `internal/infrastructure/persistence` directly (go through application) — delivery maps HTTP↔application. |

**Required backend refactor to satisfy these rules** (this is the substantive backend work): today `application` imports `infrastructure/persistence/postgres` directly (e.g. `Services.Store *postgres.Store`, methods returning `postgres.Meeting`). To enforce fan-in/fan-out, introduce **port interfaces** (`application` defines `MeetingRepo`, `OrgRepo`, `SessionRepo`, … ; `domain` owns the entities) and have `postgres.Store` implement them; `application` holds interfaces, not the concrete Store; DTO mapping moves to a boundary. Tests guard behavior throughout. This is the biggest single piece of the refactor and is sequenced as its own phase.

**Comments:** strip all Go + TS comments; relax `revive` (exported, package-comments), `godot`, and any comment-requiring linters in `.golangci.yml` and the frontend eslint config so the gate stays green commentless.

## Frontend architecture (cute, 3D)

- `packages/ui`: a Tailwind **preset** (soft palette, rounded-2xl, playful shadows, springy motion tokens) + headless components (Button/Input/Card/Dialog/Toast…) + `motion` wrappers. A separate `@leadcat/ui/3d` entry exports an `<R3FScene>` wrapper (Canvas + drei helpers) and pixi overlay components — imported only by landing/admin.
- `apps/landing`: marketing. Full-screen three.js hero, scroll-driven `motion` storytelling, pixi particle accents. Static-ish, no auth.
- `apps/admin`: the SaaS dashboard. Consumes Phase 0 web auth (Google/MS SSO + magic-link) + org/member/invite + meetings APIs via `api-client`. Tasteful 3D accents (lazy), motion page transitions, cute empty-states.
- `apps/mini-app`: Telegram Mini App. Lightweight; the meetings UX (create/edit/list/checker/profile) on `api-client`; no three.js/pixi; existing `initData`→miniapp JWT auth.
- `packages/api-client`: `openapi-typescript` generates types from `apps/backend`'s OpenAPI (script already exists); a thin layer adds the axios instance (cookie creds for admin web auth; bearer for mini-app) + TanStack Query hooks. **Single source of truth for the API contract.**

## Phased migration strategy (nothing breaks the preserved backend)

Each phase is its own spec→plan→execute cycle (superpowers chain), executed via `/senior-backend` (backend) and `/senior-frontend` (frontend), orchestrated with per-task review.

**Phase A — Merge + monorepo skeleton (safe, no deletion yet)**
1. ff-merge `feat/saas-phase0-tenancy-web-auth` → main.
2. `git mv backend/ apps/backend/`; fix module path refs, Makefile, Dockerfile, CI, embedded paths. Backend builds + all tests green from new path.
3. Add npm workspaces root `package.json`, scaffold `packages/config`.

**Phase B — Frontend wipe + scaffold**
4. Delete `frontend/` entirely (the destructive step — only after Phase A is green and approved).
5. Scaffold `apps/{landing,admin,mini-app}` (Vite+React+TS+Tailwind) + `packages/{ui,api-client,types}`.
6. Generate `api-client` from backend OpenAPI; wire shared config + tailwind preset.

**Phase C — Backend clean-arch + depguard + de-comment**
7. Introduce ports, invert `application`→`postgres` deps, add depguard rules, strip comments, relax doc-lint. Tests green throughout.

**Phase D — Rebuild the apps**
8. `mini-app` (meetings UX, lightweight) → 9. `admin` (orgs+meetings dashboard, web auth, 3D) → 10. `landing` (marketing, heavy 3D).

**Phase E — Deploy wiring**
11. Per-service Dockerfiles + Dokploy configs + CORS/origins/env for backend/admin/landing/mini-app.

## Risks & mitigations

- **Backend dependency-inversion (Phase C)** is the largest refactor — mitigated by the existing test suite + TDD per task; keep it a dedicated phase, do not mix with feature work.
- **3D bundle weight** — isolate three.js/pixi to `@leadcat/ui/3d` + lazy import; mini-app never imports it.
- **No-comments vs Go lint** — relax revive/godot in golangci config; depguard still enforces structure.
- **Frontend deletion is irreversible** — Phase B runs only after explicit approval + Phase A merged (work recoverable from git history regardless).
- **Telegram parity** — mini-app must reach functional parity with today's TMA before the old frontend is considered fully replaced.

## Open choices to confirm

- npm workspaces (recommended) vs adding pnpm/Turborepo. Default: npm workspaces.
- Depth of Phase C dependency-inversion: full (all repos behind ports) vs pragmatic (only enforce domain+delivery rules now, application↔infra inversion later). Default: full, since the request is an explicit "полный рефактор".
