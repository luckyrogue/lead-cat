# Roadmap to Beta — Design

**Status:** approved (brainstorm), ready for slice-A brainstorm + writing-plans.
**Goal:** Reach **full ТЗ §4–§8 coverage beta** — every feature in `docs/NEW-FEATURES.md` §4 (meeting CRUD + recurring + colleague schedule + conflicts + free-slots), §5 (notifications + reminders), §6 (admin panel), §7 (user settings), §8 (commands & navigation) implemented and usable.
**Topic:** Decomposes the gap from current `main` (post-docs-rebuild) into 8 shippable increment slices, each its own brainstorm → spec → plan → subagent-driven execution cycle.

## Definition of "beta-ready"

The project is beta when all of the following hold:

1. All 8 slices below are merged to `main`.
2. The `docs/MEETINGS.md` engineering-status section shows every ТЗ §4–§8 row as **Done** (no "planned").
3. The `docs/OPERATIONS.md` meetings smoke checklist passes green end-to-end — every "(planned)" row from the docs-rebuild is now "(present)" and verified on the deployed environment.
4. Operators can onboard a new tenant **without curl** (Google SA configured via the TMA admin overlay; users self-register via `/start`).
5. A pre-beta hardening pass (slice H) has been performed: PII-leak audit on logs clean, backup/rollback drill rehearsed, ops runbook updated.

## Current state (verified, as of HEAD `94c0baa`)

Compact map informing the slice boundaries (full details in the brainstorming Explore report; see also `docs/MEETINGS.md`).

**Done:**

- TMA auth (`POST /api/auth/tma`), TMA reads (`GET /me`, `/meetings`, `/schedule`, `/employees`, `POST /free-slots`).
- TMA create (`POST /meetings`) — once-only; recurrence rejected with `meetings_recurring_unsupported`.
- Bot FSMs: `/start` registration (`botreg`), `/edit` (`meetingedit`), `/schedule` (`scheduleview`), `/checker` (`checker`); `botsettings` reminders toggle (RU only, no persistence).
- Notification dispatcher (asynq) for create/update/cancel/participant ±; reminder scheduler with hard-coded offsets (10m/15m/30m/1h/2h/1d) and Redis leader lock.
- Series engine server-side: `CreateMeetingSeries`, `UpdateSeries`, `CancelSeries` (used by bot `/edit`, not yet by TMA).
- Conflict core (`MeetingConflicts`) + free-slots core (`FreeSlots`).
- Employee directory: embedded CSV, seeded into Google-configured workspaces on startup.
- Deploy: Dockerfile, Dokploy config, healthcheck, embedded migrations, encrypted SA storage; OpenAPI served at `/openapi.json` and embedded into the binary.
- Docs: meetings-only doc set (`docs/README.md` index, ТЗ pointer, `SETUP.md`, `API.md` with planned/present split, deprecated alpha-setup appendix).

**Gaps to beta:**

- TMA write surface incomplete: `PATCH /meetings/:id`, `DELETE /meetings/:id`, `POST /conflicts` missing.
- Recurrence: wizard collects `rec`/`recDays` but no `until` input; backend requires it for non-once.
- User settings (§7): no `/api/tma/settings` route; UI is mock.
- Setup cutover (§6 admin + setup-replacement spec): no `/api/tma/admin/*` routes; Google SA still curl-only.
- Admin panel (§6): no UI/API for view-all-meetings, manage users, assign admin, change email.
- Reminders customization (§5.2): hardcoded; not driven by per-user settings.
- Bot commands (§8): missing `/menu`, `/new`, `/my_meetings`, `/help`, `/admin`.
- Ops hardening: no E2E smoke automation, no PII-log audit on file, no rollback drill record.

## Slices

Each slice is a single brainstorm → spec → plan → subagent-driven execution cycle, ending with ff-merge to main (per `meetings-increment-workflow` memory).

### Slice A — TMA writes (non-recurring) finish

**Ships:** `PATCH /api/tma/meetings/:id`, `DELETE /api/tma/meetings/:id`, real `POST /api/tma/conflicts`; CreateWizard threads `editId`, uses real authed organizer, calls live conflicts on review step, guards recurring with a localized note + disabled confirm.
**Acceptance:** an authed TMA user can create / edit / delete a single meeting they organized; conflicts surface real overlaps from other employees (not just the caller's view); the wizard's "edit" path PATCHes the source meeting instead of duplicating it; organizer-only UI for edit/delete (backend still enforces 403).
**Reuses:** existing `EnsureTMAOrganizer`, `CreateMeeting`/`UpdateMeeting`/`CancelMeeting`, `MeetingConflicts`. Note: an earlier sub-project-3 brainstorm exists (`docs/superpowers/specs/2026-06-05-tma-write-paths-design.md`) and its core decisions still hold — Slice A's brainstorm should reality-check it against `main` (some backend bits may already have landed) and trim/extend accordingly.
**Effort:** ~1 week.

### Slice B — Recurrence (series everywhere)

**Ships:** CreateWizard learns `recurrence_until` (date picker, defaults reasonable); `recDays` "custom" reconciled with the backend recurrence enum (or "custom" deferred with a localized note if the engine doesn't model it); TMA create accepts `rec != once` and routes through `CreateMeetingSeries`; PATCH/DELETE in TMA accept a `scope=this|whole` parameter and route through `UpdateSeries` / `CancelSeries`.
**Acceptance:** an authed TMA user can create a weekly/daily/monthly/specific-weekdays meeting with an end date; can edit either this-occurrence or whole-series; can cancel either; series notifications fire once per series action (matching the bot `/edit` behavior).
**Open question for brainstorm:** does `recDays` ("custom" weekday set) map to a backend recurrence kind, or do we restrict the wizard to the four kinds the engine supports?
**Effort:** ~1.5 weeks.

### Slice C — User settings (ТЗ §7)

**Ships:** `GET /api/tma/settings`, `PATCH /api/tma/settings`; persistence in `bot_users` (extend `reminder_minutes` to JSON or add columns for `timezone`, `language`). Reminder intervals user-configurable from {10m, 15m, 30m, 1h, 2h, 1d} (multi-select); timezone (Almaty default, picker for per-user override); language ru/kk/en (persists; the Mini App reads it on auth-bootstrap).
**Acceptance:** Profile screen reflects/edits real settings; the reminder scheduler reads per-user offsets when dispatching; the Mini App boots into the user's saved language.
**Effort:** 0.5–1 week.

### Slice D — Setup cutover (TMA admin integrations)

**Ships:** `/api/tma/admin/*` route group (TMA-authed, `bot_users.role == admin` guard); first endpoints per the existing setup-replacement spec — `GET /api/tma/admin/integrations`, `PATCH /api/tma/admin/integrations` (Google SA JSON + subject + calendar id), `POST /api/tma/admin/integrations/verify`. Admin overlay UI in Profile → Admin → Integrations.
**Acceptance:** an admin can paste the SA JSON inside the Mini App and verify the connection; after that, `POST /api/tma/meetings` succeeds against the configured workspace. The curl-based path stays as deprecated appendix until D ships.
**Reuses:** `docs/superpowers/specs/2026-06-05-tma-setup-replacement-design.md`.
**Effort:** ~1 week.

### Slice E — Admin panel (ТЗ §6)

**Ships:** `GET /api/tma/admin/meetings` (list-all with filters: by date range, by organizer email, by status); admin write paths `PATCH`/`DELETE` for any meeting (admin scope of existing TMA write commands). `GET /api/tma/admin/users` (list with role); `PATCH /api/tma/admin/users/:id/email` (§6.2 — admin corrects a user's email); `PATCH /api/tma/admin/users/:id/role` (assign/revoke admin). Admin overlay UI for each.
**Acceptance:** the admin can perform every §6 capability inside the Mini App; non-admins get 403.
**Effort:** 1.5–2 weeks.

### Slice F — Notifications customization (ТЗ §5)

**Ships:** Reminder dispatcher reads per-user offsets from Slice C settings (no more hard-coded list); audit + verify every lifecycle DM (created / updated / cancelled / participant added / removed) — content matches ТЗ §5 wording, includes the right deep-link / Meet URL; per-event toggles if ТЗ allows them.
**Acceptance:** disabling reminders in Profile actually stops reminder DMs for that user; changing the offset list takes effect on the next scheduled tick; every lifecycle event has a verified DM matching its ТЗ template.
**Depends on:** Slice C.
**Effort:** 0.5–1 week.

### Slice G — Bot commands (ТЗ §8) + polish

**Ships:** Missing bot commands wired: `/menu` (opens Mini App / main menu), `/new` (probably routes to Mini App create), `/my_meetings` (list deep-links into Mini App or a short summary), `/help` (short usage card), `/admin` (admin-only gate). BotFather command list synced (per `docs/BOTFATHER.md`). Mini App ru/kk/en finishing: any missing keys, any RTL-edge fixed.
**Acceptance:** every command listed in `docs/NEW-FEATURES.md` §8 responds in the bot; the BotFather command list in production matches `BOTFATHER.md`.
**Effort:** ~0.5 week.

### Slice H — Pre-beta hardening

**Ships:** E2E smoke automation (a small script that runs the `docs/OPERATIONS.md` checklist end-to-end against a staging deployment); explicit PII-leak audit on log output (grep all log calls for `email`, `init_data`, `Authorization`, etc.); backup-and-rollback drill rehearsed and recorded; basic load smoke (N participants × N meetings within the conflict checker); ops runbook updated; a health/metrics dashboard sketch (even a Grafana JSON in `deploy/`).
**Acceptance:** the smoke script passes green; the PII audit has zero unintended leaks; the runbook describes "what to do at 3am if reminders stop firing"; backup/rollback was actually performed on staging once.
**Effort:** ~1 week.

## Dependency graph

```
A → B
A → C → F
A → D → E
G       (independent)
A,B,C,D,E,F,G → H
```

Serialized order for solo execution: **A → B → C → D → E → F → G → H** (≈ 8 weeks).

If a parallel hand were ever brought in, C and D can go in parallel after A; G can go alongside any other slice.

## Known risks / unknowns (to revisit at each slice's brainstorm)

- **Recurrence model mismatch.** The wizard's `custom` weekday set vs the backend recurrence engine — decide at slice B whether to support, defer, or restrict.
- **Multi-workspace.** TMA create currently uses the first Google-configured workspace. If beta tenants are multi-workspace this surfaces at slice D's brainstorm.
- **Test bar.** Backend convention is "pure logic unit-tested, I/O build-verified." For beta, consider at least one HTTP-level integration test per admin route — flag at slice E.
- **Frontend tests.** No test runner today (typecheck + build). Decide at slice H whether to add Vitest + Playwright smoke (vs the deployment-side OPERATIONS smoke script).
- **OpenAPI drift.** With every new TMA route the schema must be regenerated; consider a CI gate at slice H if it isn't already there.

## Process notes

- Per the meetings-increment-workflow memory: each slice is its own branch `feat/meetings-<slice>`, full superpowers chain, ff-merge to main when finished (push when requested).
- Per the concurrent-git-on-shared-branch memory: stage explicit paths, verify HEAD between tasks, use `git push . feat:main` for ff merges when the working tree is dirty.
- The ТЗ `docs/NEW-FEATURES.md` is the source of truth for every slice's acceptance criteria; do not edit it.
- After each slice merges, refresh `docs/MEETINGS.md` and the relevant `docs/*.md` (e.g. `API.md` "planned" → "present" rows) — this keeps the docs aligned with reality.

## Out of scope (this roadmap)

- Multi-tenant SaaS features (deleted from the product).
- Native web app for non-Telegram users.
- ТЗ §11 items (transcription, exports, CSV via UI, per-user OAuth, recording management).
- A full Vitest+Playwright frontend test runner unless slice H decides to add it.
- Anything past §8 (the ТЗ stops at §8 functionally; §9–§12 are stack/UC/scope notes).

## Next step

**Start Slice A** — TMA writes (non-recurring) finish. There's an existing sub-project-3 design (`docs/superpowers/specs/2026-06-05-tma-write-paths-design.md`) that covers most of slice A; its core decisions still hold but the spec should be reality-checked against current `main` (some bits may already have landed in your recent code work) before going to writing-plans.
