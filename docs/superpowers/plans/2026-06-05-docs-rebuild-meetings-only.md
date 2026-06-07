# Docs Rebuild — Meetings-Only Bot — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild every project doc + agent-guidance file to describe a single-purpose Google Meet meetings-management Telegram Mini App, with the surviving SaaS/platform layer demoted to a short "deprecated alpha-setup" appendix.

**Architecture:** Curated in-place rewrite of the flat `docs/` directory (delete obsolete, rewrite the rest meetings-only, add an index), plus `AGENTS.md` and `.cursor/rules/*` updates. No code changes. Each task commits only its own docs/agent-guide files.

**Tech Stack:** Markdown docs describing a Go (Fiber, asynq, pgx/Postgres) + React (Vite, TanStack Router/Query) Telegram Mini App.

**Spec:** `docs/superpowers/specs/2026-06-05-docs-rebuild-meetings-only-design.md`

---

## Grounded facts (verified — rely on these; re-verify before asserting line numbers)

- **Module:** `github.com/Jaryq-Lab/notify-bot`. Product name in docs: **Lead Cat**.
- **Current routes** (`backend/internal/delivery/http/app.go`):
  - Public: `GET /api/health`, `GET /openapi.json`, `GET /metrics`.
  - TMA auth: `POST /api/auth/tma`.
  - **TMA group `/api/tma` (TMAAuth)** — PRESENT: `GET /me`, `GET /meetings`, `GET /schedule`, `GET /employees`, `POST /free-slots`, `POST /meetings`. NOT YET present (mark **planned**): `PATCH /meetings/:id`, `DELETE /meetings/:id`, `POST /conflicts`, and the whole `/api/tma/admin/*` group.
  - **Platform (DEPRECATED appendix only):** `authPub` `/api/auth/*` (email/phone OTP, passkey, oauth); `ap` `/api/me`, `/api/workspaces`; `ws` `/api/workspaces/:id/*` (chat link/status, integrations [+verify], members [+vcs, +sync-chat], scenarios [+run, +runs], employees, meetings [+conflicts, +free-slots]).
- **Env vars** (`backend/internal/platform/config/config.go`): meetings-relevant = `BOT_TOKEN`, `BOT_ADMIN_TELEGRAM_IDS`, `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `JWT_ISSUER`, `JWT_TTL_HOURS`, `MASTER_ENCRYPTION_KEY`, `CALENDAR_STUB`, `WEBAPP_URL`, `STATIC_DIR`, `HTTP_ADDR`, `AUTO_MIGRATE`, `CORS_ALLOWED_ORIGINS`, `LOG_LEVEL`, `LOG_FORMAT`. Deprecated/SaaS = `GITHUB_OAUTH_*`, `GITLAB_OAUTH_*`, `WEBAUTHN_RP_*`, `AUTH_DEV_*`, `AUTH_OTP_LOG`. (Confirm against `deploy/.env.example`.)
- **`.cursor/rules/`** files: `cat-design.mdc`, `docs.mdc`, `frontend-fsd.mdc`, `go-backend.mdc`, `lead-cat-auth.mdc`, `lead-cat-core.mdc`, `migrations.mdc`, `redis-asynq.mdc`, `scenarios.mdc`.
- **Inbound links to retire** (`SCENARIOS.md` / `ALPHA-SMOKE.md` / `ONBOARDING-WORKSPACE.md` / `scenarios.mdc`): found in `AGENTS.md`, `docs/REQUIREMENTS.md`, repo-root `README.md` (and `.cursor/rules/scenarios.mdc` itself, which is deleted).
- **OpenAPI:** `docs/openapi.json` (served at `GET /openapi.json`), mirrored to `frontend/src/shared/api/generated/`. API.md references it as the machine-readable source of truth.
- **ТЗ:** `docs/NEW-FEATURES.md` — DO NOT EDIT. §4 = features, §5 = notifications, §6 = admin, §7 = settings, §8 = commands/nav, §9 = stack (note: superseded by Go/React).
- **Identity model** (for ARCHITECTURE/AUTH): `bot_users` (telegram_id ↔ email ↔ role; bot `/start` registration; TMA JWT `tok_typ:"tma"`) vs `platform_users` (UUID, organizer of meetings). The `EnsureTMAOrganizer` bridge links them by email.

## Conventions

- **Each task commits ONLY the doc/agent-guide files it names** — `git add <explicit paths>` (NEVER `git add -A`/`-A`), then commit. The working tree has unrelated in-progress code changes; do not stage or touch any code file, and never stage `frontend/vite.config.ts`.
- Commit messages end with `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- **No secrets** in any doc (no real tokens, SA JSON, JWTs).
- **Meetings-only framing:** outside the explicit "Deprecated — alpha setup" appendices, NO surviving doc may present scenarios/n8n, notify chat-binding, VCS, or multi-tenant workspaces as current product.
- **Deprecated appendix block** (reuse this exact shape wherever a doc must mention the platform layer):

  ```markdown
  ## Appendix — Deprecated: alpha setup (curl)

  > These platform endpoints/flows exist only for alpha operator bootstrap and are
  > being replaced by in-Mini-App admin (`/api/tma/admin/*`, see
  > `docs/superpowers/specs/2026-06-05-tma-setup-replacement-design.md`). Not part
  > of the product; slated for removal.
  ```

- Markdown style: match the existing docs' tone (concise, tables where they help). Use `elements-of-style` clarity if available.

## File structure (created / renamed / deleted / modified)

- **Delete:** `docs/SCENARIOS.md`, `docs/ALPHA-SMOKE.md`, `.cursor/rules/scenarios.mdc`.
- **Create:** `docs/README.md`.
- **Rename + rewrite:** `docs/ONBOARDING-WORKSPACE.md` → `docs/SETUP.md`.
- **Rewrite:** `docs/REQUIREMENTS.md`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/AUTH.md`, `docs/DEPLOY-DOKPLOY.md`, `docs/BOTFATHER.md`, `docs/OPERATIONS.md`, `docs/REDIS.md`.
- **Light refresh:** `docs/MEETINGS.md`, `docs/LOCAL_DEV.md`. **Confirm-no-op:** `docs/DESIGN-CATS.md`, `docs/MIGRATIONS.md`.
- **Agent guides:** rewrite `AGENTS.md`; rewrite `.cursor/rules/{lead-cat-core,redis-asynq,lead-cat-auth}.mdc`; audit/light-touch `.cursor/rules/{go-backend,frontend-fsd,docs}.mdc`; keep `.cursor/rules/{cat-design,migrations}.mdc`.
- **Link fixes:** `AGENTS.md`, repo-root `README.md`, plus any link discovered by the final grep.

---

## Task 1: Deletes + index page

**Files:**

- Delete: `docs/SCENARIOS.md`, `docs/ALPHA-SMOKE.md`, `.cursor/rules/scenarios.mdc`
- Create: `docs/README.md`

- [ ] **Step 1: Delete the three obsolete files**

```bash
git rm docs/SCENARIOS.md docs/ALPHA-SMOKE.md .cursor/rules/scenarios.mdc
```

- [ ] **Step 2: Create `docs/README.md`**

Write an index with: (a) the product one-liner (see below), (b) a table mapping each doc → one-line purpose → audience (User / Admin / Operator / Dev), listing the ТЗ first. Only list docs that will exist after this rebuild (NEW-FEATURES, README, REQUIREMENTS, ARCHITECTURE, API, AUTH, DEPLOY-DOKPLOY, SETUP, BOTFATHER, OPERATIONS, REDIS, MEETINGS, DESIGN-CATS, LOCAL_DEV, MIGRATIONS). Do NOT list SCENARIOS/ALPHA-SMOKE/ONBOARDING-WORKSPACE.

Product one-liner to use verbatim at the top:

```markdown
# Lead Cat — docs

**Lead Cat** is a single-purpose **Google Meet meetings-management Telegram Mini App**.
Employees register via the bot's `/start`; inside the Mini App they create, edit, and
delete meetings, get conflict warnings, find common free time, view colleague schedules,
and receive Telegram reminders. Google Meet links are created through a corporate Google
service account. Admins configure the integration inside the Mini App. Stack: Go (Fiber,
asynq, Postgres) + React Telegram Mini App.

Start with the **ТЗ**: [NEW-FEATURES.md](NEW-FEATURES.md).
```

Then the doc-index table.

- [ ] **Step 3: Verify the deletions and index**

Run: `ls docs/SCENARIOS.md docs/ALPHA-SMOKE.md 2>&1 | grep -c "No such file"` → expect `2`. Run: `test -f docs/README.md && echo OK`.

- [ ] **Step 4: Commit**

```bash
git add docs/README.md docs/SCENARIOS.md docs/ALPHA-SMOKE.md .cursor/rules/scenarios.mdc
git commit -m "docs: delete scenarios/alpha-smoke, add docs index (meetings-only)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Rewrite `docs/REQUIREMENTS.md`

**Files:** Modify: `docs/REQUIREMENTS.md`

- [ ] **Step 1: Read the current file** to preserve any still-accurate prerequisites/wording: `docs/REQUIREMENTS.md`.

- [ ] **Step 2: Rewrite** with these sections, in order:
  1. **Purpose** — the meetings-only product one-liner (from Task 1).
  2. **Actors** — User (any registered employee) and Main Administrator (ТЗ §2). No "workspace owner/member" SaaS roles.
  3. **Feature set** — bullet summary of ТЗ §4–§8: create (fields, types, recurrence, auto-naming), view/list + detail, participant add/remove, edit (incl. recurring scope), delete/cancel (single/series), colleague schedule, conflict detection, free-time checker, notifications, admin panel, user settings (tz/reminders/lang), commands/nav.
  4. **Prerequisites / stack** — Go (Fiber, asynq, pgx), React (Vite, TanStack), PostgreSQL, Redis (asynq + scheduler lock), Google Calendar API v3 via service account, Telegram bot. Explicitly note: ТЗ §9's "Python/Node" is superseded by Go/React.
  5. **Out of scope** — copy ТЗ §11 (transcription, exports, CSV-via-UI, per-user OAuth, recording mgmt) + add: multi-tenant SaaS, scenario automation, VCS integration (removed from product).
- **Exclude:** scenario engine, coverage gate, VCS, notify chat-binding, multi-tenant workspace model — except a single line in "Out of scope".
- **Fix the inbound link:** this file referenced a now-deleted doc (`SCENARIOS.md` and/or `ONBOARDING-WORKSPACE.md`) — remove/redirect it (→ `SETUP.md` where appropriate).

- [ ] **Step 3: Verify** no stray references: `grep -nE "scenario|n8n|VCS|workspace|multi-tenant" docs/REQUIREMENTS.md` — every hit must be inside "Out of scope" or absent. `grep -nE "SCENARIOS\.md|ONBOARDING-WORKSPACE\.md" docs/REQUIREMENTS.md` → no output.

- [ ] **Step 4: Commit**

```bash
git add docs/REQUIREMENTS.md
git commit -m "docs: rewrite REQUIREMENTS for meetings-only product

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Rewrite `docs/ARCHITECTURE.md`

**Files:** Modify: `docs/ARCHITECTURE.md`

- [ ] **Step 1: Read** the current file + skim `backend/internal/` layer dirs to keep layer descriptions accurate.

- [ ] **Step 2: Rewrite** with sections:
  1. **Overview** — meetings-only product one-liner; "frontend = React Telegram Mini App; backend = Go monolith (`cmd/server`: Fiber HTTP, bot, asynq workers)".
  2. **Layers** — `domain ← application ← infrastructure / delivery / platform` (under `backend/internal/`); dependency rule (inward). The meeting domain (`domain/meeting`), application commands/queries (CreateMeeting/UpdateMeeting/CancelMeeting/MeetingConflicts/FreeSlots/EmployeeSchedule), delivery (`delivery/http` handlers + middleware), infrastructure (postgres, Google calendar, asynq), platform (auth, bot FSMs, employeedir).
  3. **Identity** — `bot_users` (telegram-native, `/start` registration, TMA JWT) vs `platform_users` (meeting organizer UUID); the `EnsureTMAOrganizer` email bridge.
  4. **Request flow** — Mini App → `/api/tma/*` (TMAAuth middleware → `bot_user` locals) → application → Postgres/Google.
  5. **Async** — asynq for reminder/notification jobs; scheduler leader lock (Redis).
  6. **Appendix — Deprecated: alpha setup (curl)** — one paragraph: platform JWT + `/api/workspaces/*` still exist for operator bootstrap; being replaced by TMA admin.
- **Exclude** from the main body: scenario scheduler/engine, VCS, multi-tenant workspace model.

- [ ] **Step 3: Verify** `grep -nE "scenario|VCS|gitlab|github" docs/ARCHITECTURE.md` → only inside the deprecated appendix (or none). The phrase "Mini App" and "meeting" appear in the overview.

- [ ] **Step 4: Commit**

```bash
git add docs/ARCHITECTURE.md
git commit -m "docs: rewrite ARCHITECTURE for meetings-only Mini App

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Rewrite `docs/API.md`

**Files:** Modify: `docs/API.md`

- [ ] **Step 1: Read** the current file and `backend/internal/delivery/http/app.go` (the route registrations) to keep the route list exact.

- [ ] **Step 2: Rewrite** with sections:
  1. **Overview** — base path `/api`; auth = TMA JWT (bearer) for `/api/tma/*`; link `docs/openapi.json` (served at `GET /openapi.json`) as the machine-readable source of truth; generated client at `frontend/src/shared/api/generated/`.
  2. **Public** — `GET /api/health`, `GET /metrics`, `GET /openapi.json`.
  3. **TMA auth** — `POST /api/auth/tma` (initData → TMA JWT).
  4. **TMA (present)** — table: `GET /api/tma/me`, `GET /api/tma/meetings?scope=upcoming|past|all`, `GET /api/tma/schedule?email=&scope=`, `GET /api/tma/employees?q=`, `POST /api/tma/free-slots`, `POST /api/tma/meetings`. One-line purpose each.
  5. **TMA (planned)** — clearly labelled: `PATCH /api/tma/meetings/:id`, `DELETE /api/tma/meetings/:id`, `POST /api/tma/conflicts`, and `/api/tma/admin/*` (per the setup-replacement design). Mark "planned — not yet implemented."
  6. **Appendix — Deprecated: alpha setup (curl)** — list the platform groups compactly: `/api/auth/*` (OTP/passkey/oauth), `/api/me`, `/api/workspaces` + `/api/workspaces/:id/*` (chat, integrations[+verify], members[+vcs,+sync-chat], scenarios[+run,+runs], employees, meetings[+conflicts,+free-slots]). One line that these are operator-only and being retired.
- **Accuracy:** do not list TMA PATCH/DELETE/conflicts/admin as present (they aren't yet). Re-verify against `app.go` before writing.

- [ ] **Step 3: Verify** `grep -nE "/api/tma/meetings|/api/auth/tma|openapi.json" docs/API.md` → present. The platform routes appear only under the deprecated appendix heading.

- [ ] **Step 4: Commit**

```bash
git add docs/API.md
git commit -m "docs: rewrite API for TMA stack; platform routes as deprecated appendix

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Rewrite `docs/AUTH.md`

**Files:** Modify: `docs/AUTH.md`

- [ ] **Step 1: Read** the current file + `backend/internal/platform/auth/tma.go` and `delivery/http/middleware` (TMAAuth) for accuracy.

- [ ] **Step 2: Rewrite** with sections:
  1. **Overview** — Telegram-native auth is the only user-facing flow.
  2. **TMA flow** — `POST /api/auth/tma` validates Telegram `initData` (HMAC + `auth_date` freshness; dev-mode bypass), mints a short-lived **TMA JWT** (`tok_typ:"tma"`, reuses `JWT_SECRET`) resolved against `bot_users`; `middleware.TMAAuth` guards `/api/tma/*` and sets `c.Locals("bot_user")`. Roles: `bot_users.role` (`user`/`admin`); admin bootstrap via `BOT_ADMIN_TELEGRAM_IDS`.
  3. **Registration** — bot `/start` upserts a `bot_user` (telegram_id ↔ email).
  4. **Appendix — Deprecated: alpha setup (curl)** — platform OTP (email/phone), passkey, GitHub/GitLab OAuth issue a platform JWT for `/api/workspaces/*`; operator-only, being retired.
- **Exclude** from main body: passkey/OAuth/OTP as user features.
- **No secrets.**

- [ ] **Step 3: Verify** `grep -nE "passkey|oauth|OTP" docs/AUTH.md` → only inside the deprecated appendix. `grep -n "tok_typ" docs/AUTH.md` → present.

- [ ] **Step 4: Commit**

```bash
git add docs/AUTH.md
git commit -m "docs: rewrite AUTH — TMA primary, platform auth deprecated appendix

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Rewrite `docs/DEPLOY-DOKPLOY.md`

**Files:** Modify: `docs/DEPLOY-DOKPLOY.md`

- [ ] **Step 1: Read** the current file + `deploy/.env.example` to keep env accurate.

- [ ] **Step 2: Rewrite** keeping the Dokploy deploy mechanics, but the env section lists only meetings-relevant vars (see Grounded facts): `BOT_TOKEN`, `BOT_ADMIN_TELEGRAM_IDS`, `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `JWT_ISSUER`, `JWT_TTL_HOURS`, `MASTER_ENCRYPTION_KEY`, `CALENDAR_STUB`, `WEBAPP_URL`, `STATIC_DIR`, `HTTP_ADDR`, `AUTO_MIGRATE`, `CORS_ALLOWED_ORIGINS`, `LOG_LEVEL`, `LOG_FORMAT`. Add a one-line note: Google service account is configured via the integration (today curl/alpha; future TMA admin), and the employee directory ships as an embedded CSV. Move OAuth/WebAuthn/AUTH_DEV vars into a short "Deprecated alpha-setup env" note (or omit with a pointer).
- **Exclude:** scenario/chat-webhook env and steps.

- [ ] **Step 3: Verify** `grep -nE "WEBHOOK|scenario|chat" docs/DEPLOY-DOKPLOY.md` → none in main body. `grep -n "BOT_TOKEN" docs/DEPLOY-DOKPLOY.md` → present.

- [ ] **Step 4: Commit**

```bash
git add docs/DEPLOY-DOKPLOY.md
git commit -m "docs: rewrite DEPLOY-DOKPLOY env for meetings-only

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Rename + rewrite `docs/ONBOARDING-WORKSPACE.md` → `docs/SETUP.md`

**Files:** Rename+modify: `docs/ONBOARDING-WORKSPACE.md` → `docs/SETUP.md`

- [ ] **Step 1: Rename via git** (preserves history):

```bash
git mv docs/ONBOARDING-WORKSPACE.md docs/SETUP.md
```

- [ ] **Step 2: Rewrite `docs/SETUP.md`** as the operator/admin setup path:
  1. **Create the Telegram bot** (BotFather) → `BOT_TOKEN`; pointer to `BOTFATHER.md`.
  2. **Configure Google** — service account JSON, calendar subject, calendar id (today: platform `PATCH /api/workspaces/:id/integrations` via curl with a platform JWT — mark this as the interim alpha path; target: TMA admin `/api/tma/admin/integrations`, see setup-replacement spec).
  3. **Employee directory** — embedded CSV (`backend/internal/platform/employeedir/employees.csv`); edit → rebuild → redeploy.
  4. **Users self-register** — via the bot `/start`.
  5. **Admins** — `BOT_ADMIN_TELEGRAM_IDS` bootstrap; admin surfaces in the Mini App (planned).
- Frame the curl/platform-JWT steps explicitly as **deprecated alpha setup**, being replaced by TMA admin.

- [ ] **Step 3: Verify** `test -f docs/SETUP.md && test ! -f docs/ONBOARDING-WORKSPACE.md && echo OK`. `grep -nE "scenario" docs/SETUP.md` → none.

- [ ] **Step 4: Commit**

```bash
git add docs/SETUP.md docs/ONBOARDING-WORKSPACE.md
git commit -m "docs: rename ONBOARDING-WORKSPACE -> SETUP, rewrite for meetings

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Rewrite `docs/BOTFATHER.md`

**Files:** Modify: `docs/BOTFATHER.md`

- [ ] **Step 1: Read** the current file; keep BotFather token-creation mechanics.

- [ ] **Step 2: Rewrite** to cover: creating the bot + token, setting the Mini App (Web App) URL, `/start` registration, and the meetings menu commands per ТЗ §8 (the user-facing command set). Remove notify commands `/test`, `/chatid`, `/report`, `/leave` and any Telegram-group chat-binding instructions.

- [ ] **Step 3: Verify** `grep -nE "/test|/chatid|/report|/leave|chatid" docs/BOTFATHER.md` → none. `grep -n "/start" docs/BOTFATHER.md` → present.

- [ ] **Step 4: Commit**

```bash
git add docs/BOTFATHER.md
git commit -m "docs: rewrite BOTFATHER for meetings bot (drop notify commands)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 9: Rewrite `docs/OPERATIONS.md` (absorb smoke checklist)

**Files:** Modify: `docs/OPERATIONS.md`

- [ ] **Step 1: Read** the current `docs/OPERATIONS.md` (and recall the deleted `ALPHA-SMOKE.md` intent from git: `git show HEAD~N:docs/ALPHA-SMOKE.md` is optional — its meetings-relevant checks are re-authored here).

- [ ] **Step 2: Rewrite** with sections:
  1. **Logging** — zap, `LOG_LEVEL`/`LOG_FORMAT`, structured fields (request_id, workspace_id/meeting_id), no secrets.
  2. **Health & metrics** — `GET /api/health`, `GET /metrics`; list the HTTP counters; remove any `*_scenario_runs_*` metric.
  3. **Postgres** — backup / rollback notes (keep existing, generic).
  4. **Redis** — scheduler leader lock for reminders (pointer to REDIS.md).
  5. **Meetings smoke checklist** — a short ordered E2E: authed `POST /api/auth/tma` (dev initData) → `GET /api/tma/meetings` → `POST /api/tma/meetings` (once-only) → `POST /api/tma/free-slots` → cancel (when DELETE lands). Mark steps that depend on planned routes as "(planned)".
- **Exclude** scenario run metrics/ops.

- [ ] **Step 3: Verify** `grep -nE "scenario" docs/OPERATIONS.md` → none. `grep -nE "health|metrics|smoke" docs/OPERATIONS.md` → present.

- [ ] **Step 4: Commit**

```bash
git add docs/OPERATIONS.md
git commit -m "docs: rewrite OPERATIONS for meetings (+ smoke checklist, drop scenario metrics)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 10: Rewrite `docs/REDIS.md`

**Files:** Modify: `docs/REDIS.md`

- [ ] **Step 1: Read** the current file.

- [ ] **Step 2: Rewrite**: Redis is used for (1) **asynq queues** carrying reminder/notification jobs and (2) the **scheduler leader lock** (single active scheduler across replicas). Note the footprint shrank now that scenario tasks are gone. Keep any accurate connection/`REDIS_URL` notes.
- **Exclude** scenario-task queue descriptions.

- [ ] **Step 3: Verify** `grep -nE "scenario" docs/REDIS.md` → none. `grep -nE "asynq|REDIS_URL|lock" docs/REDIS.md` → present.

- [ ] **Step 4: Commit**

```bash
git add docs/REDIS.md
git commit -m "docs: rewrite REDIS around reminders + scheduler lock

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 11: Light refresh — `MEETINGS.md`, `LOCAL_DEV.md`; confirm `DESIGN-CATS.md`, `MIGRATIONS.md`

**Files:** Modify (as needed): `docs/MEETINGS.md`, `docs/LOCAL_DEV.md`

- [ ] **Step 1: `MEETINGS.md`** — read it; update the status summary to current reality: TMA auth + read paths done; write paths (create present, edit/delete/conflicts planned); setup cutover planned (link the setup-replacement spec). Keep its structure. Remove any stale "SaaS"/scenario phrasing.

- [ ] **Step 2: `LOCAL_DEV.md`** — read it; remove any scenario/notify references; confirm the `make` workflow + ports are accurate. If nothing references the old product, leave content as-is.

- [ ] **Step 3: Confirm no-op** for `DESIGN-CATS.md` and `MIGRATIONS.md`: `grep -nE "scenario|notify|multi-tenant|VCS" docs/DESIGN-CATS.md docs/MIGRATIONS.md` → expect no output. If any hit appears, make the minimal edit and include the file in the commit.

- [ ] **Step 4: Verify** `grep -nE "scenario" docs/MEETINGS.md docs/LOCAL_DEV.md` → none.

- [ ] **Step 5: Commit** (stage only the files actually changed)

```bash
git add docs/MEETINGS.md docs/LOCAL_DEV.md
git commit -m "docs: refresh MEETINGS + LOCAL_DEV for meetings-only

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 12: Rewrite `AGENTS.md` + `.cursor/rules/*`

**Files:** Modify: `AGENTS.md`, `.cursor/rules/lead-cat-core.mdc`, `.cursor/rules/redis-asynq.mdc`, `.cursor/rules/lead-cat-auth.mdc`, `.cursor/rules/go-backend.mdc`, `.cursor/rules/frontend-fsd.mdc`, `.cursor/rules/docs.mdc`

- [ ] **Step 1: `AGENTS.md`** — read it; rewrite the top description to the meetings-only product + Go/React stack. KEEP the engineering-principles table (KISS/DRY/SOLID/Clean Arch), CQRS, and logging/observability sections (generic — still apply). Remove scenario-engine and multi-tenant SaaS framing. Update the **Cursor rules table**: drop the `scenarios.mdc` row. Fix any inbound link to deleted/renamed docs (it referenced `SCENARIOS.md` / `ONBOARDING-WORKSPACE.md`). Update the "Status / Implementation checklist" pointer if it names removed docs.

- [ ] **Step 2: `.cursor/rules/lead-cat-core.mdc`** — read; rewrite to "single-purpose meetings Mini App bot" (remove "SaaS multi-tenant", "one platform BOT_TOKEN" scenario framing). Keep generic always-on guidance.

- [ ] **Step 3: `.cursor/rules/redis-asynq.mdc`** — read; reframe around reminder/notification jobs + scheduler lock (not scenario runs).

- [ ] **Step 4: `.cursor/rules/lead-cat-auth.mdc`** — read; make TMA Telegram auth primary; platform OTP/passkey/OAuth secondary/deprecated.

- [ ] **Step 5: Audit `.cursor/rules/{go-backend,frontend-fsd,docs}.mdc`** — read each; remove scenario/notify/multi-tenant mentions. Specifically `docs.mdc` says "…or scenario nodes — update the matching file under docs/" → drop "scenario nodes"; ensure its glob list doesn't name deleted docs. `frontend-fsd.mdc` → align with the TMA feature-slice structure (`frontend/src/features/*`), drop old SaaS web references. `go-backend.mdc` → drop scenario engine mentions if any. Make minimal edits; if a file has no offending content, leave it.

- [ ] **Step 6: Verify** `grep -rniE "scenario|n8n|multi-tenant" AGENTS.md .cursor/rules/` → only acceptable hits (e.g., a deliberate "no longer supported" note) or none. `grep -rnE "SCENARIOS\.md|ALPHA-SMOKE\.md|ONBOARDING-WORKSPACE\.md|scenarios\.mdc" AGENTS.md .cursor/rules/` → none.

- [ ] **Step 7: Commit** (stage only files actually changed)

```bash
git add AGENTS.md .cursor/rules/lead-cat-core.mdc .cursor/rules/redis-asynq.mdc .cursor/rules/lead-cat-auth.mdc .cursor/rules/go-backend.mdc .cursor/rules/frontend-fsd.mdc .cursor/rules/docs.mdc
git commit -m "docs: rewrite AGENTS + cursor rules for meetings-only

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 13: Link-fix sweep + final verification

**Files:** Modify (as needed): repo-root `README.md`, any file flagged by the grep

- [ ] **Step 1: Find remaining inbound links** to deleted/renamed docs:

```bash
grep -rnE "SCENARIOS\.md|ALPHA-SMOKE\.md|ONBOARDING-WORKSPACE\.md|scenarios\.mdc" . \
  --include="*.md" --include="*.mdc" --include="*.go" 2>/dev/null \
  | grep -v "docs/superpowers/"
```

- [ ] **Step 2: Fix each hit** — repoint `ONBOARDING-WORKSPACE.md` → `SETUP.md`; remove `SCENARIOS.md`/`ALPHA-SMOKE.md`/`scenarios.mdc` references (the repo-root `README.md` is a known hit — update its docs links + product blurb to meetings-only). Re-run the grep until it returns nothing (ignoring `docs/superpowers/`).

- [ ] **Step 3: Framing consistency sweep** — outside deprecated appendices, confirm no surviving doc presents the old product as current:

```bash
grep -rniE "n8n|scenario engine|notify-bot|multi-tenant" docs/*.md AGENTS.md .cursor/rules/ \
  | grep -viE "deprecated|no longer|removed|out of scope"
```

Expect no output (investigate/fix any hit).

- [ ] **Step 4: Sanity — code untouched** — confirm this project staged no code: `git diff --name-only HEAD~12..HEAD | grep -vE "^(docs/|AGENTS.md|README.md|.cursor/)" || echo "docs-only OK"` (adjust the range to this project's commits). Optionally `make build` to confirm nothing code-side broke.

- [ ] **Step 5: Commit** (only if Steps 2 changed files)

```bash
git add README.md   # + any other flagged files
git commit -m "docs: fix inbound links after meetings-only docs rebuild

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes (for the executor)

- **Spec coverage:** deletes + index → Task 1; REQUIREMENTS → 2; ARCHITECTURE → 3; API (+ openapi pointer, planned vs present, platform appendix) → 4; AUTH → 5; DEPLOY → 6; SETUP rename → 7; BOTFATHER → 8; OPERATIONS (+ smoke absorbed) → 9; REDIS → 10; MEETINGS/LOCAL_DEV refresh + DESIGN-CATS/MIGRATIONS confirm → 11; AGENTS + cursor rules → 12; link-fix + framing sweep + code-untouched check → 13. NEW-FEATURES.md untouched throughout.
- **Deprecated-appendix consistency:** the exact appendix block (Conventions) is reused in API/AUTH/ARCHITECTURE/SETUP/DEPLOY where the platform layer must be mentioned.
- **Accuracy guardrails:** route lists from `app.go` + `docs/openapi.json` (TMA PATCH/DELETE/conflicts/admin marked planned); env from config + `deploy/.env.example`; no secrets.
- **Commit hygiene:** every task stages only its named docs/agent-guide files — never `git add -A`, never code, never `frontend/vite.config.ts`. If the working tree's concurrent code changes cause a staging surprise, STOP and report rather than broadening the add.
- **Known approximation:** where backend code is mid-refactor, docs describe the intended contract (ТЗ + setup-replacement design) and label not-yet-built routes "planned" rather than asserting them as present.

```

```
