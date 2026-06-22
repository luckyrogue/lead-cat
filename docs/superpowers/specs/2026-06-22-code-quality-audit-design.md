# Code Quality & Structure Audit — Design

**Date:** 2026-06-22
**Type:** Assessment (read-only). No code changes. Remediation decided after the report.

## Goal

Produce an evidence-based evaluation of the architecture and code quality of the
entire monorepo (backend + admin + mini-app + landing + packages), delivered as a
single prioritized report with per-dimension grades and a remediation backlog.

## Scope

Full monorepo:

- `apps/backend` — Go monolith (Fiber HTTP, Telegram bot, asynq workers)
- `apps/admin` — React Router v7 SPA
- `apps/mini-app` — Telegram Mini App
- `apps/landing` — SSR landing
- `packages/*` — `ui`, `api-client`, `types`, `config`, `brand`

Out of scope: writing tests, fixing findings (a separate decision after the report).

## Dimensions (rubric)

Each area is graded A–F on the dimensions that apply:

1. **Architecture & boundaries** — Clean Architecture (dependencies point inward:
   delivery → application → domain; infrastructure implements ports), CQRS
   (command vs query separation, no writes in queries), dependency direction,
   module cohesion. Frontend: FSD layering (entities → features → routes).
2. **Code quality** — SOLID, DRY (real duplication, not preemptive abstraction),
   function/file complexity and size, KISS, dead code.
3. **Consistency & conventions** — error handling (`%w` wrapping, log once at
   boundary), structured logging (zap, no secrets), i18n parity (en/ru/kk),
   adherence to `.cursor/rules`.
4. **Maintainability** — file-size cap (≤300 lines per project convention),
   readability, testability seams (without writing tests).
5. **Frontend-specific** — component decomposition, design-system usage
   (`@leadcat/ui`, cat-design), accessibility, data-fetching (TanStack Query),
   form patterns (react-hook-form + zod).
6. **Risk flags** — bug-prone spots, partial-transaction risk, security-adjacent
   observations (noted in passing; not the focus).

## Method

### Layer C — tooling sweep (objective baseline, run first)

- Go: `golangci-lint run`, `go vet ./...`, file-size scan (>300 lines), and
  complexity/duplication (`gocyclo` / `dupl`) if available.
- Frontend: `eslint` (FSD boundaries already enforced), `tsc`, file-size scan,
  dead-code (`knip` / `ts-prune`) and duplication (`jscpd`) if available.
- Output: objective metrics table feeding the agent pass.

Tools that are not installed are noted as "not run" — no silent gaps.

### Layer A — multi-agent audit (fan-out over the baseline)

~6 reviewer agents in parallel, each returning structured findings with
`file:line`, a per-dimension grade, and a recommendation:

- **BE-1:** `domain` + `application` (CQRS, `services.go`, `conflict.go`,
  `series_edit.go`, `meetingedit/service.go`)
- **BE-2:** `delivery/http` (handlers, middleware, `app.go`) + `infrastructure`
  (persistence, calendar, oauth, telegram, queue)
- **BE-3:** `platform/*` (~25 packages — assess whether the platform layer has
  grown too broad / leaks concerns)
- **FE-1:** `apps/admin` (features, routes, components)
- **FE-2:** `apps/mini-app`
- **FE-3:** `apps/landing` + `packages/*`

The controller synthesizes all agent outputs into one report, deduping
cross-area findings.

## Deliverable

Report at `docs/audit/2026-06-22-code-quality-audit.md`:

1. Executive summary
2. Grades table — areas × dimensions (A–F)
3. Findings by severity (P0 / P1 / P2), each with `file:line` and recommendation
4. Prioritized remediation backlog (what to fix, in what order)

## Known starting signals (from initial sweep)

- Backend files over the 300-line cap: `platform/meetingedit/service.go` (734),
  `delivery/http/handlers/miniapp_write.go` (536), `application/command/meetings.go`
  (417), `infrastructure/telegram/multitenant.go` (406), `application/services.go`
  (400), `application/series_edit.go` (375),
  `infrastructure/persistence/postgres/meeting_repo.go` (370), `cmd/server/main.go`
  (313), `delivery/http/handlers/web_meetings.go` (309),
  `delivery/http/handlers/web_auth.go` (305).
- Frontend mostly within bounds; outliers: `admin/.../meeting-form.tsx` (367);
  i18n dictionaries and generated `schema.ts` are data/generated (excluded from
  the cap).

## Non-goals

- No tests written.
- No code modified during the audit.
- Security is not the focus; security-adjacent issues are noted but not deeply
  pursued.
