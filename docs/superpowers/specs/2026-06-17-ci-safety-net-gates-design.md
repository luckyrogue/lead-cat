# WS1 — CI Safety Net & Gates (design)

**Date:** 2026-06-17
**Status:** approved — ready for implementation plan
**Part of:** Public-launch hardening (workstream 1 of 7). This is the foundational workstream; it must land before WS2–WS7 so that every test those workstreams add is actually enforced.

## Goal

Make every pull request run `go test`, `golangci-lint`, frontend `typecheck`, frontend `lint`, and a Docker image-build validation — and fail red on any of them. From this point forward, no change (including the later hardening workstreams) can silently break the build, regress a test, or rot lint/types.

## Background — verified current state (2026-06-17)

CI today (`.github/workflows/_build.yml`, called by `ci.yml`) runs only:
- `go` job: `go mod download` → `go vet ./...` → `go build -o /dev/null ./cmd/server ./cmd/migrate`
- `frontend` job: `pnpm install --frozen-lockfile` → `pnpm --filter admin --filter mini-app --filter landing build`

It does **not** run `go test`, `golangci-lint`, frontend `typecheck`, or frontend `lint`. Docker images are built only on push to `main`/tags (`_docker.yml`), never validated on PRs.

When the gates are turned on, the current failures are small and fully enumerated:

| Gate | Today | Failures to fix |
|---|---|---|
| backend `go test ./...` | passes (4 pkgs) | none |
| backend `golangci-lint` (`config/.golangci.yml`) | 4 issues | gofmt: `internal/application/command/meetings.go`, `internal/application/command/ports.go`, `internal/application/model/model.go`; unused func: `internal/delivery/http/handlers/miniapp_read.go:47 splitMeetingTime` |
| frontend `typecheck` (all 3) | passes | none |
| frontend `lint` | 2 errors (landing only) | `apps/landing/app/features/landing/components/nav.tsx` — unused import `Badge`; `apps/landing/app/shared/i18n/locale-landing-page.tsx` — `import/no-restricted-paths` FSD boundary violation |

mini-app and admin lint are clean. No frontend test suites exist yet (that is WS3) — so a frontend **test** gate is intentionally out of scope here.

## Decisions (resolved)

1. **Landing lint fixes are authorized.** The two `apps/landing/**` errors are normally in the user's parallel-WIP zone; the user explicitly approved fixing them so the gate can go green. At implementation time, verify HEAD and check for intermingled uncommitted landing WIP before editing, and stage only those two files.
2. **Docker-build validation runs on PRs**, build-only (no push, no Dokploy webhooks), to catch Dockerfile/`.dockerignore` breakage before merge.
3. **golangci-lint runs via the official `golangci/golangci-lint-action`** (pinned version, built-in caching), pointed at `config/.golangci.yml`.

## Scope

### In scope
- Extend `_build.yml` to add the test/lint/typecheck gates.
- Add PR-time Docker image-build validation (no push).
- Fix the 6 enumerated current failures so the expanded gate is green.
- Add `make test` and `make ci` targets for local parity with CI.

### Out of scope (other workstreams)
- Writing new tests (WS2 backend, WS3 frontend, WS4 E2E).
- Accessibility, security/rate-limiting, product gaps, load testing (WS3/WS5/WS6/WS7).
- Changing what deploys or how Dokploy is triggered.

## Design

### 1. `_build.yml` — `go` job
Append after the existing `go build` step, in `apps/backend`:
- `go test ./...`
- `golangci-lint` via `golangci/golangci-lint-action` (pinned), with `working-directory: apps/backend` and `args: --config ../../config/.golangci.yml ./...` (or `version:` pinned + config path), matching the `make lint` invocation.

Ordering: keep `go vet` and `go build` (fast fail), then `go test`, then lint.

### 2. `_build.yml` — `frontend` job
Add before/after the build step:
- `pnpm --filter admin --filter mini-app --filter landing typecheck`
- `pnpm --filter admin --filter mini-app --filter landing lint`

Build remains the last step so a type/lint error fails fast and cheap.

### 3. `_build.yml` — Docker validation (PR only)
Add a job (or reuse `_docker.yml` with a `push: false` input) that builds all four images (backend, admin, mini-app, landing) without pushing and without Dokploy webhooks, gated to `pull_request` events. Use buildx layer caching to keep it affordable. The existing push-on-main path in `_docker.yml` is unchanged.

### 4. Fix current failures (to make the gate green)
- **gofmt:** run `gofmt -w` on the three named backend files. Pure formatting; no behavior change.
- **unused func:** delete `splitMeetingTime` in `miniapp_read.go` (confirmed unused by the linter). If a follow-up workstream needs it, it can be reintroduced.
- **landing — `nav.tsx`:** remove the unused `Badge` import.
- **landing — `locale-landing-page.tsx`:** resolve the `import/no-restricted-paths` violation. The offender is `app/shared/i18n/locale-landing-page.tsx`, which imports `~/features/landing/pages/landing-page` — a `shared → features` import the FSD rule forbids. `LocaleLandingPage` is a composition wrapper (`LocaleProvider` + `DocumentLang` + `LandingPage`) used by two routes (`app/routes/$locale._index.tsx`, `app/routes/_index.tsx`). Fix by **relocating the file into the landing feature layer** (e.g. `app/features/landing/locale-landing-page.tsx`), where importing both the page and `shared/i18n` is allowed, then update the two route imports. Do not suppress the rule.

### 5. Makefile
- Add `test:` → `cd $(BACKEND) && $(GO) test ./...`.
- Add `ci:` → `fmt-check` + `lint` + `test` + `typecheck` + `build` (the local mirror of the CI gate), so a developer can reproduce CI with one command.

## Testing / verification

- Local: `make ci` is green from a clean checkout.
- Push a throwaway branch / open a draft PR and confirm: the `go` job runs vet+build+test+lint; the `frontend` job runs typecheck+lint+build; the docker job builds all four images without pushing; the whole run is green.
- Sanity-check the gate actually bites: temporarily introduce a failing test and a lint error locally and confirm `make ci` (and, if cheap, a draft PR) goes red, then revert.

## Risks & mitigations
- **CI wall-clock grows** (test + lint + 4 image builds). Mitigation: buildx caching, Go build/module caching (already via `setup-go`), pnpm cache (already present). Acceptable for a public-launch bar.
- **Landing WIP collision** — the user edits `apps/landing/**` in parallel. Mitigation: verify HEAD + working-tree state immediately before editing the two landing files; stage only those paths; never `git add -A`.
- **golangci-lint version drift** between local and CI. Mitigation: pin the action's version and keep it aligned with the documented local install.

## Done criteria
- `_build.yml` runs go test + golangci-lint + FE typecheck + FE lint + PR Docker validation.
- All gates green on a fresh PR.
- `make test` and `make ci` exist and pass locally.
- The 6 enumerated failures are fixed; no new failures introduced.
