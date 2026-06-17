# WS1 — CI Safety Net & Gates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every push/PR run `go test` + `golangci-lint` + frontend `typecheck` + `lint` + a PR-time Docker image-build validation, and fix the 6 current failures so the expanded gate starts green.

**Architecture:** Extend the existing reusable workflow `.github/workflows/_build.yml` (called by `ci.yml`) with the missing gate steps; add a PR-only Docker-validate job to `ci.yml` that reuses `_docker.yml` with `push: false`. Fix the small set of current lint failures (3 backend gofmt files, 1 unused Go func, 2 landing lint errors). Add `make test`/`make ci` for local parity.

**Tech Stack:** GitHub Actions (reusable workflows), Go 1.26 + golangci-lint v2 (`config/.golangci.yml`), pnpm workspace (eslint flat config + tsc), Docker buildx, GNU Make.

**Standing constraints (carry into every task):**
- Work directly on `main`; do **not** create feature branches. Commit locally per task; the human pushes on request.
- Before editing any `apps/landing/**` file (Task 2 only), run `git status` and `git log --oneline -3`; the human edits landing in parallel. Stage **only** the explicit paths listed; never `git add -A`.
- Run Go via `env -u GOROOT`. Commit trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- The `.github/workflows/_build.yml` working-tree change currently present (pnpm/node version bump) is the human's WIP — Task 4 edits `_build.yml`, so first reconcile: if that change is still uncommitted, incorporate around it and stage `_build.yml` deliberately (it's an explicit target in Task 4), but do not revert the human's version bump.

---

## File Structure

| File | Responsibility | Task |
|---|---|---|
| `apps/backend/internal/delivery/http/handlers/miniapp_read.go` | Remove dead `splitMeetingTime` func | 1 |
| `apps/backend/internal/application/command/meetings.go` | gofmt | 1 |
| `apps/backend/internal/application/command/ports.go` | gofmt | 1 |
| `apps/backend/internal/application/model/model.go` | gofmt | 1 |
| `apps/landing/app/features/landing/components/nav.tsx` | Remove unused `Badge` import | 2 |
| `apps/landing/app/features/landing/locale-landing-page.tsx` | New home for `LocaleLandingPage` (FSD-legal layer) | 2 |
| `apps/landing/app/shared/i18n/locale-landing-page.tsx` | Deleted (moved) | 2 |
| `apps/landing/app/routes/_index.tsx` | Update import path | 2 |
| `apps/landing/app/routes/$locale._index.tsx` | Update import path | 2 |
| `Makefile` | Add `test` + `ci` targets | 3 |
| `.github/workflows/_build.yml` | Add go test + golangci-lint + FE typecheck + FE lint | 4 |
| `.github/workflows/ci.yml` | Add PR-only `docker-validate` job (reuse `_docker.yml`, `push: false`) | 5 |

---

### Task 1: Fix backend lint failures

**Files:**
- Modify: `apps/backend/internal/delivery/http/handlers/miniapp_read.go` (remove `splitMeetingTime`, lines ~47-51)
- Modify (gofmt only): `apps/backend/internal/application/command/meetings.go`, `apps/backend/internal/application/command/ports.go`, `apps/backend/internal/application/model/model.go`

- [ ] **Step 1: Confirm the failures exist**

Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./...`
Expected: 4 issues — gofmt on the 3 files above, `unused` on `splitMeetingTime`.

- [ ] **Step 2: Remove the dead function**

In `apps/backend/internal/delivery/http/handlers/miniapp_read.go`, delete exactly this function (it is confirmed unused by the linter):

```go
func splitMeetingTime(startsAt, endsAt time.Time, loc *time.Location) (date, start, end string) {
	s := startsAt.In(loc)
	e := endsAt.In(loc)
	return s.Format("2006-01-02"), s.Format("15:04"), e.Format("15:04")
}
```

After removal, check that the `time` import is still used elsewhere in the file (it is — `miniappScopeWindow` uses `time.Time`). Do not remove the import.

- [ ] **Step 3: Apply gofmt to the three flagged files**

Run: `cd apps/backend && env -u GOROOT gofmt -w ./internal/application/command/meetings.go ./internal/application/command/ports.go ./internal/application/model/model.go`

This is pure formatting (import grouping / trailing whitespace); no behavior change.

- [ ] **Step 4: Verify lint is clean and the build/tests still pass**

Run: `cd apps/backend && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./...`
Expected: `0 issues.`

Run: `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./...`
Expected: build succeeds; tests pass (4 packages `ok`, rest `[no test files]`).

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/delivery/http/handlers/miniapp_read.go \
  apps/backend/internal/application/command/meetings.go \
  apps/backend/internal/application/command/ports.go \
  apps/backend/internal/application/model/model.go
git commit -m "$(cat <<'EOF'
style(backend): gofmt + drop dead splitMeetingTime for clean golangci-lint

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Fix landing lint failures (AUTHORIZED landing edit)

**Pre-flight (MANDATORY):** Run `git status` and `git log --oneline -3`. Confirm no uncommitted human WIP in the four landing files below. If there is, STOP and report — do not clobber it.

**Files:**
- Modify: `apps/landing/app/features/landing/components/nav.tsx` (line 1 import)
- Create: `apps/landing/app/features/landing/locale-landing-page.tsx`
- Delete: `apps/landing/app/shared/i18n/locale-landing-page.tsx`
- Modify: `apps/landing/app/routes/_index.tsx` (import path)
- Modify: `apps/landing/app/routes/$locale._index.tsx` (import path)

- [ ] **Step 1: Confirm the failures exist**

Run: `pnpm --filter landing lint`
Expected: 2 errors — `nav.tsx:1` unused `Badge`; `locale-landing-page.tsx:1` `import/no-restricted-paths`.

- [ ] **Step 2: Remove the unused `Badge` import**

In `apps/landing/app/features/landing/components/nav.tsx`, change line 1 from:

```tsx
import { Badge, Button } from "@leadcat/ui"
```

to:

```tsx
import { Button } from "@leadcat/ui"
```

(Verify `Badge` is not referenced elsewhere in the file before removing — the linter confirms it is unused.)

- [ ] **Step 3: Create the relocated component in the feature layer**

The violation is a `shared → features` import. `LocaleLandingPage` is a composition wrapper, so it belongs in the `landing` feature, not `shared`. Create `apps/landing/app/features/landing/locale-landing-page.tsx` with exactly:

```tsx
import { LocaleProvider } from "~/shared/i18n/context"
import { DocumentLang } from "~/shared/i18n/document-lang"
import type { Locale } from "~/shared/i18n/types"
import { LandingPage } from "~/features/landing/pages/landing-page"

export function LocaleLandingPage({ locale }: { locale: Locale }) {
  return (
    <LocaleProvider locale={locale}>
      <DocumentLang />
      <LandingPage />
    </LocaleProvider>
  )
}
```

This is FSD-legal: a feature importing its own page (`features/landing/pages/...`) and `shared/i18n` are both downward/intra-feature imports allowed by `packages/config/eslint.config.ts` zones.

- [ ] **Step 4: Delete the old file**

Run: `git rm apps/landing/app/shared/i18n/locale-landing-page.tsx`

- [ ] **Step 5: Update the two route imports**

In `apps/landing/app/routes/_index.tsx`, change:

```tsx
import { LocaleLandingPage } from "~/shared/i18n/locale-landing-page"
```

to:

```tsx
import { LocaleLandingPage } from "~/features/landing/locale-landing-page"
```

Apply the identical change in `apps/landing/app/routes/$locale._index.tsx` (same old line → same new line).

- [ ] **Step 6: Verify lint, typecheck, and build are clean for landing**

Run: `pnpm --filter landing lint`
Expected: no errors.

Run: `pnpm --filter landing typecheck`
Expected: `Done` (no type errors — the import path resolves).

Run: `pnpm --filter landing build`
Expected: build succeeds.

- [ ] **Step 7: Commit (explicit paths only)**

```bash
git add apps/landing/app/features/landing/components/nav.tsx \
  apps/landing/app/features/landing/locale-landing-page.tsx \
  apps/landing/app/shared/i18n/locale-landing-page.tsx \
  apps/landing/app/routes/_index.tsx \
  'apps/landing/app/routes/$locale._index.tsx'
git commit -m "$(cat <<'EOF'
fix(landing): clean eslint — drop unused Badge, relocate LocaleLandingPage to feature layer

Resolves import/no-restricted-paths (shared must not import features) by moving
the LocaleLandingPage composition wrapper into features/landing.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Add Makefile `test` and `ci` targets

**Files:**
- Modify: `Makefile` (add two targets near the existing `lint`/`typecheck`/`build`)

- [ ] **Step 1: Inspect the existing variables and targets**

Run: `sed -n '1,116p' Makefile`
Note the variables used by existing targets: `$(BACKEND)`, `$(GO)`, `$(ROOT)`, `$(PNPM)`. Reuse them — do not invent new ones.

- [ ] **Step 2: Add a `test` target**

Add immediately after the `lint:` target block:

```makefile
test:
	@cd $(BACKEND) && $(GO) test ./...
```

- [ ] **Step 3: Add a `ci` aggregate target**

Add after the `build:` target block (so `build` is already defined):

```makefile
ci: fmt-check lint test typecheck build
	@echo "ci: all gates passed"
```

- [ ] **Step 4: Register both in `.PHONY` if one exists; otherwise skip**

Run: `grep -n "PHONY" Makefile`
If a `.PHONY:` line exists, append `test ci` to it. If not (the file uses no `.PHONY`), skip — the existing targets work without it.

- [ ] **Step 5: Verify both targets run green**

Run: `make test`
Expected: backend tests pass.

Run: `make ci`
Expected: runs fmt-check → lint → test → typecheck → build, ending with `ci: all gates passed`. (Requires Tasks 1 and 2 already committed so lint is clean.)

- [ ] **Step 6: Commit**

```bash
git add Makefile
git commit -m "$(cat <<'EOF'
build(make): add test and ci targets mirroring the CI gate

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Add test + lint + typecheck gates to `_build.yml`

**Files:**
- Modify: `.github/workflows/_build.yml` (extend the `go` and `frontend` jobs)

**Note:** This file may carry the human's uncommitted pnpm/node version bump. Preserve it — only add the new steps described below.

- [ ] **Step 1: Read the current file**

Run: `cat .github/workflows/_build.yml`
Confirm the `go` job ends with `go build -o /dev/null ./cmd/server ./cmd/migrate` and the `frontend` job ends with the `pnpm --filter ... build` step.

- [ ] **Step 2: Extend the `go` job — add test + golangci-lint**

In the `go` job, after the `- run: go build -o /dev/null ./cmd/server ./cmd/migrate` step, add:

```yaml
      - run: go test ./...

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v2.12.2
          working-directory: apps/backend
          args: --config ../../config/.golangci.yml
```

(`version: v2.12.2` matches the documented local install and the `version: "2"` schema in `config/.golangci.yml`. The `go` job already sets `working-directory: apps/backend` as a job default, but the action needs its own `working-directory` input since it does not inherit the job default.)

- [ ] **Step 3: Extend the `frontend` job — add typecheck + lint**

In the `frontend` job, after `- run: pnpm install --frozen-lockfile` and **before** the `pnpm --filter ... build` step, add:

```yaml
      - run: pnpm --filter admin --filter mini-app --filter landing typecheck
      - run: pnpm --filter admin --filter mini-app --filter landing lint
```

(Ordering: typecheck and lint fail fast and cheap before the heavier build.)

- [ ] **Step 4: Validate the workflow YAML locally**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/_build.yml')); print('yaml ok')"`
Expected: `yaml ok`.

If `actionlint` is installed (`command -v actionlint`), run `actionlint .github/workflows/_build.yml` and expect no errors. If not installed, skip — the YAML parse + the later live run cover it.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/_build.yml
git commit -m "$(cat <<'EOF'
ci: gate on go test, golangci-lint, and frontend typecheck/lint

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Add PR-time Docker-build validation to `ci.yml`

**Files:**
- Modify: `.github/workflows/ci.yml` (add a `docker-validate` job)

**Context:** `_docker.yml` already accepts `push: boolean`; with `push: false` it skips GHCR login, uses `load: true` (build-only), and skips the Dokploy trigger. We reuse it on PRs.

- [ ] **Step 1: Read the current `ci.yml`**

Run: `cat .github/workflows/ci.yml`
Confirm the existing `docker` job is gated to `push` on `main`/tags (it is).

- [ ] **Step 2: Add the `docker-validate` job**

Add this job to `ci.yml` (sibling of `build` and `docker`), so PRs build all four images without pushing:

```yaml
  docker-validate:
    name: docker (validate)
    needs: build
    if: github.event_name == 'pull_request'
    uses: ./.github/workflows/_docker.yml
    with:
      push: false
    permissions:
      contents: read
      packages: write
```

(`needs: build` so images are only built once the cheap gates pass. `packages: write` mirrors the reusable workflow's declared permissions even though `push: false` skips the push step. No secrets block is needed — the Dokploy step is `if: inputs.push` and won't run.)

- [ ] **Step 3: Validate the workflow YAML locally**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml')); print('yaml ok')"`
Expected: `yaml ok`.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "$(cat <<'EOF'
ci: validate all Docker images on PRs (build-only, no push)

Catches Dockerfile/.dockerignore breakage before merge by reusing _docker.yml
with push:false.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Verify the gate end-to-end

**Files:** none (verification only)

- [ ] **Step 1: Full local gate**

Run: `make ci`
Expected: green, ending `ci: all gates passed`.

- [ ] **Step 2: Prove the gate bites (local only — do NOT push these)**

Inject a failing test temporarily:

```bash
cat > apps/backend/internal/platform/scheduler_agent/zz_gatecheck_test.go <<'EOF'
package scheduler_agent

import "testing"

func TestGateBites(t *testing.T) { t.Fatal("intentional") }
EOF
make test || echo "EXPECTED: test gate is red"
rm apps/backend/internal/platform/scheduler_agent/zz_gatecheck_test.go
```

Expected: `make test` fails (non-zero), confirming the test gate would block CI; the file is then removed.

- [ ] **Step 3: Confirm clean tree**

Run: `git status --short`
Expected: only the human's pre-existing untracked/modified items (if any) remain; no `zz_gatecheck_test.go`, no stray changes from this task.

- [ ] **Step 4: Post-push observation (after the human pushes — informational, not a code step)**

Once the human pushes `main`, the `build` job runs vet+build+test+lint and the `frontend` job runs typecheck+lint+build on the push. On the next PR, `docker-validate` additionally builds all four images. Confirm the Actions run is green (e.g. `gh run watch` or the GitHub Actions UI). If anything is red, treat the failure output as the next task's input.

---

## Notes on execution order
Tasks 1 and 2 must precede Tasks 3–5 (the gate steps assume lint is already clean). Task 4 before/after Task 5 is interchangeable, but keep them as separate commits. Task 6 is the final verification.
