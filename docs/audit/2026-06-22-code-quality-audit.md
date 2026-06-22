# Code Quality & Structure Audit — lead-cat monorepo

**Date:** 2026-06-22
**Scope:** Full monorepo — `apps/backend`, `apps/admin`, `apps/mini-app`, `apps/landing`, `packages/*`
**Type:** Read-only assessment. No code changed. Method: objective tooling sweep (Layer C) + 6 parallel reviewer agents by area (Layer A); headline findings spot-verified.
**Design:** `docs/superpowers/specs/2026-06-22-code-quality-audit-design.md`

---

## Executive summary

The codebase is **healthy and disciplined on the surface, with a few real structural fault lines underneath**. Every objective gate is green: `golangci-lint` (0 issues, with depguard enforcing Clean-Architecture boundaries), `go vet`, `eslint` (FSD import boundaries enforced), and `tsc` all pass across the repo. Conventions are followed well — structured zap logging with no secrets, consistent TanStack Query + react-hook-form/zod on the frontend, strong auth hygiene (constant-time compares, AES-GCM, in-memory tokens), and near-total i18n coverage.

The problems tools cannot see cluster into four themes:

1. **The `platform` layer is mislayered** — ~half of it is Telegram delivery code that imports the Postgres driver directly, a Clean-Architecture inversion that passes lint only because depguard has no rule for `platform/**`. (BE-3)
2. **The backend CQRS split is abandoned mid-flight** — a 117-method `*Services` god-object and a 97-method `Repository` interface coexist with a ~10%-populated `command`/`query` package, producing forced helper/port duplication. (BE-1)
3. **Application-level write orchestration is not atomic** — repo-level multi-row writes correctly use transactions, but the orchestration that interleaves calendar I/O with several repo calls (series reshape, create) has partial-failure gaps with weak/no compensation. (BE-1, BE-2)
4. **The frontend design system is missing primitives**, so `Switch`, `Textarea`, `WeekdayPicker`, and a `Field` wrapper are hand-rolled and duplicated across admin and mini-app; and **`<html lang>` is wrong on SSR for every locale** in all three apps. (FE-1, FE-2, FE-3)

None of these block functionality today; they are maintainability, observability, and correctness-under-failure risks. All are mechanical-to-moderate fixes.

---

## Grades by area

Last column is **Risk** for backend areas, **Frontend-specific** (DS/a11y/SSR) for frontend areas.

| Area | Architecture | Code quality | Consistency | Maintainability | Risk / FE-specific |
|---|---|---|---|---|---|
| BE-1 domain + application | C | C+ | B | C+ | **D** (tx gaps) |
| BE-2 delivery + infrastructure | A− | C+ | C+ | B− | B+ |
| BE-3 platform | **D** | C+ | B | C | B− |
| FE-1 admin | B+ | B | B | B+ | B |
| FE-2 mini-app | A | B | A− | B+ | A− |
| FE-3 landing + packages | A− | B | B− | A− | **C+** (SSR lang) |

**Overall: B−.** Frontend is the stronger half (mini-app is the best-kept area in the repo). Backend architecture is dragged down by the platform mislayering and the half-done CQRS migration, not by code-level sloppiness.

---

## Objective baseline (Layer C)

| Check | Result |
|---|---|
| `golangci-lint run` (backend) | **0 issues** — depguard enforces domain/application/delivery layering |
| `go vet ./...` | clean |
| `eslint` (admin, mini-app, landing) | clean — FSD layer boundaries enforced |
| `tsc` (all apps) | clean — i18n parity compile-enforced (`typeof en`) |
| Backend files > 300-line cap | **10** (see P1-MAINT) |
| Frontend files > 300-line cap | 2 real (`meeting-form.tsx` 367, `meeting-create-page.tsx` 322) + data/generated |
| depguard rule for `platform/**` | **missing** (root cause of BE-3 P0) |
| gocyclo / dupl / knip / ts-prune / jscpd | not installed — **not run** (covered by agent judgment) |

---

## Findings

Severity: **P0** = correctness / data-integrity / serious architecture violation · **P1** = significant quality/maintainability/a11y · **P2** = polish.

### P0 — address first

**P0-1 · `platform` bot packages bypass the application layer and import the Postgres driver directly** *(verified: 10 files)*
`internal/platform/{meetingedit,scheduleview,checker,reminder_scheduler,meeting_notifier,meetingrecipients,botreg,botsettings,employeedir}/…` import `internal/infrastructure/persistence/postgres` and type their interfaces in terms of `postgres.Meeting`/`postgres.Employee`/`postgres.BotUser`. These are constructed in `infrastructure/telegram/multitenant.go:41-45` and render Telegram keyboards — they are a second delivery layer. The `delivery-no-direct-db` depguard rule forbids exactly this for `delivery/**`, but there is no `platform/**` rule, so it passes lint. **Fix:** add a `platform-no-direct-db` depguard rule; have these packages depend on `application/model` types (the pattern `meeting_notifier/ports.go` already follows) rather than `postgres.*`.

**P0-2 · Application-level write orchestration is not atomic; series reshape has no rollback on late failure**
`application/series_edit.go:248-375` (`ChangeSeriesEnd`) and `application/command/meetings.go:147-188` (create) issue N calendar `CreateEvent` calls, then `CreateMeetingSeries`, `CancelSeriesOccurrences`, `SetSeriesRecurrenceUntil` as separate steps. The repo proves a transaction pattern exists (`repository.go:62` `UpdateMeetingsTx`, and `CreateMeetingSeries`/`CreateOrganization` use `tx.Begin/Rollback/Commit` correctly — BE-2 verified repo-level writes are transactional). But the *orchestration* interleaving calendar I/O with multiple repo calls is not wrapped; the create path has best-effort calendar-delete compensation, the cancel/until path has none. A mid-operation failure leaves a half-reshaped series (orphaned/extra occurrences, inconsistent `recurrence_until`). **Fix:** wrap the DB mutations of each operation in a single transaction; treat calendar as an outer best-effort layer with reconciliation, not interleaved.

**P0-3 · `<html lang>` is wrong on SSR for every locale in all three apps** *(verified)*
`apps/landing/app/root.tsx:23` (`lang="ru" suppressHydrationWarning`), `apps/mini-app/app/root.tsx:42` (`"en"`), `apps/admin/app/root.tsx:33` (`"en"`). Locale is only applied client-side via a `useEffect` (`@leadcat/ui` `HtmlLangSync` / landing's `DocumentLang`). Server HTML for `/en` and `/kk` ships the wrong `lang`, hurting SEO and screen readers and causing a hydration attribute flip (which `suppressHydrationWarning` masks). The `{ html: { lang: locale } }` entry in `landing-meta.ts:83` is a **no-op** — React Router's `meta` export has no `html` descriptor — which is likely why the team believed it was handled. **Fix:** expose locale to the root (root loader / `useParams`) and render `<html lang={locale}>` in `Layout` server-side; then delete the client-side lang-sync components and the no-op meta entry.

### P1 — significant

**P1-1 · CQRS split abandoned mid-flight; `*Services` is a 117-method god-object**
`application/services.go:22-42` + `meeting_service.go:44-58` (pass-through shims). Only `CreateMeeting`/`UpdateMeeting`/`CancelMeeting` moved to `command.Meetings`; all series/participant/survey/booking commands and every query still hang off `*Services`. This is the largest backend maintainability liability and contradicts the stated CQRS convention. **Fix:** decide — finish the migration (writes → `command/`, reads → `query/`) or delete the half-built packages. The in-between state is the worst option and forces P1-2/P1-3.

**P1-2 · `Repository` is a 97-method fat interface (ISP violation)**
`application/repository.go:12-127` spans meetings, series, surveys, bookings, invites, sessions, magic links, OAuth state, audit, members; every consumer depends on the whole surface. Narrow interfaces at `query/meeting_read.go:30-33` and `command/ports.go:12-21` show the right pattern. **Fix:** define role-specific interfaces per consumer; stop threading the monolith.

**P1-3 · Forced helper & port duplication from the CQRS split**
`ownerOrOrganizer`/`organizerEmail`/`orStr`/`orDefault` defined twice (`command/meetings.go:268-273,54-63,389-401` and `application/participants.go:16-32`, `meeting_helpers.go:13-18`); meeting-update logic forked across `command/meetings.go:275-318` (`ApplyMeetingUpdate`) and `series_edit.go:24-60` (`applySeriesUpdate`); `UpdateSeries`/`UpdateWholeSeries` ~90% duplicated; parallel `CalendarProvider`/`JobQueue` port defs. **Fix:** resolves naturally once P1-1 is decided; until then, hoist shared helpers.

**P1-4 · `platform` conflates cross-cutting utilities with Telegram feature/presentation code**
`meetingedit`, `scheduleview`, `checker`, `scheduler_agent`, `botreg`, `botsettings`, `meeting_notifier`, `reminder_scheduler` are conversation features; `auth`, `config`, `observability`, `httpclient`, `boti18n`, `emailtemplates`, `fanio` are true support. **Fix:** split into `internal/delivery/telegram/<feature>` vs `internal/platform/<util>` so depguard can constrain each (pairs with P0-1).

**P1-5 · `meetingedit/service.go` is a 734-line god-service (5 responsibilities)**
`internal/platform/meetingedit/service.go` — callback routing (36-branch switch :63-100), state transitions, keyboard/text rendering (:501-652), participant sub-flow (:323-475), delete sub-flow (:533-587), DTO mapping (:671-715). 2.4× over cap. **Fix:** split along the existing function seams into `render.go` / `participants.go` / `apply.go` / `mapping.go`.

**P1-6 · Five byte-for-byte duplicated Redis session stores**
`internal/platform/{checker,botreg,scheduleview,meetingedit}/redis_sessions.go` (+ `scheduler_agent`) differ only by key prefix + state type. **Fix:** one generic `botsession.Store[T]` parameterized by prefix + TTL.

**P1-7 · Mini-app handlers drop request-scoped context (request_id + audit actor)** *(verified)*
All `miniapp_*` handlers pass Fiber's raw `c.Context()` instead of `c.UserContext()`, where `middleware.RequestContext` stores `request_id` and `withAuditActor` stores the audit actor. Mini-app logs carry no `request_id`; admin audit-actor is set but the subsequent app calls read the unmodified ctx. Web handlers do it correctly. **Fix:** replace `c.Context()` → `c.UserContext()` across mini-app handlers (28 sites in `miniapp_write.go` alone).

**P1-8 · Heavy duplicated boilerplate across write handlers**
`miniapp_write.go:91-516` and `web_meetings.go:43-309` repeat the same prologue (auth-extract → `uuid.Parse` → org-resolve → ensure-organizer → identical `errors.Is` error switch). `resolveParticipantOp` already proves the extraction works for two handlers. **Fix:** extract `resolveMeetingOp(c)` and a shared `mapMeetingWriteError` (the latter exists in `web_meetings.go:22` but mini-app re-implements it inline). Shrinks `miniapp_write.go` below cap.

**P1-9 · Agent booking + meeting-notifier are not idempotent under retry/multi-replica**
`scheduler_agent/service.go:135-155` clears `Pending` under a process-local `bookMu`, but `Book` runs after unlock and the mutex doesn't protect across replicas (multi-instance deploy implied by the reminder leader-lock). `meeting_notifier/notifier.go:147-175` — only `HandleCreated` claims via `TryClaimReminder`; `HandleUpdated`/`HandleCancelled`/participant paths send unconditionally, so an asynq retry after a partial send re-pings already-notified users. **Fix:** dedupe booking at the data layer (idempotency key) or hold the claim in Redis; add per-(meeting,recipient,event) claims to the notifier update/cancel paths.

**P1-10 · Design system is missing primitives → hand-rolled duplication across apps**
`@leadcat/ui` exports no `Switch`, `Textarea`, `WeekdayPicker`, or shared `Field`/`FormField`. As a result: toggle switch reimplemented 3× (`event-type-dialog.tsx:245`, `survey-dialog.tsx:158`, `question-editor.tsx:107` — two lack `aria-label`); weekday picker duplicated (`meeting-form.tsx:272`, `event-type-dialog.tsx:176`); `<textarea>` class-string inlined (`event-type-dialog.tsx:133`, public `survey-form.tsx:56`); local `Field` wrapper duplicated across **6** files spanning both admin and mini-app (`meeting-form.tsx:348`, `event-type-dialog-field.tsx`, mini-app `meeting-create-page.tsx:302`, `meeting-edit-dialog.tsx:173`, `checker-page.tsx:99`). **Fix:** add `Switch`/`Textarea`/`WeekdayPicker`/`Field` to `@leadcat/ui`; replace all call sites (also closes the a11y gap).

**P1-11 · Public booking/survey surfaces bypass the shared axios client + Query layer**
`routes/book.$slug.tsx:117`, `routes/book.$slug.form.tsx:81`, `features/public-survey/api.ts:27,38` use raw `fetch` + manual `useState`/`useEffect` state machines, losing centralized error normalization, CSRF, and cache/retry. `resolveBaseUrl` is reimplemented 4× (`shared/api/client.ts:21`, `responses-page.tsx:30`, `public-survey/api.ts:18`, `shared/auth/api.ts:23`). Public email validation hand-rolls a regex instead of zod (`book.$slug.form.tsx:44`). **Fix:** move data access into `entities/public-booking/api.ts` consumed via Query hooks; export one `resolveApiBaseUrl()`; convert the booking form to rhf+zod.

**P1-12 · Landing duplicates `HtmlLangSync` and mixes two i18n access patterns**
`apps/landing/app/shared/i18n/document-lang.tsx` re-implements `@leadcat/ui`'s `HtmlLangSync` (admin/mini-app import the package version). Landing components call both `useT()` (key-based, ru-fallback) and `useLandingDict()` (raw typed access, **no fallback**) — a latent correctness gap for `kk`/`en` array content. **Fix:** delete `document-lang.tsx` (or remove all client lang-sync once P0-3 lands); standardize i18n access — `useLandingDict` for typed arrays with fallback semantics, `useT` for scalars.

### P2 — polish

- **P2-1 · Five near-identical Telegram markup converters** — `infrastructure/telegram/multitenant.go:247,276,322,334,346` → one generic `toMarkup[T]` (Go 1.26 generics); file is 406 lines.
- **P2-2 · Hardcoded `Asia/Almaty` timezone in 4+ packages** — `meetingedit/service.go:294,725`, `scheduler_agent/tools.go:17`, `reminder_scheduler/scheduler.go:131`, `meeting_notifier/notifier.go:29` → one `platform/tz` helper (single-tenant assumption in a multi-tenant pivot).
- **P2-3 · Reminder leader-lock not renewed during a tick** — `reminder_scheduler/scheduler.go:45-93` (90s TTL, 60s interval, blocking work per tick); `TryClaimReminder` is the real safety net — renew the lock or document this.
- **P2-4 · Inconsistent error-code vocabulary across handlers** — `"invalid meeting id"` (spaces) vs `invalid_meeting_id` vs `internal` vs `internal_error`; forwarded verbatim to clients. Standardize on snake_case machine codes.
- **P2-5 · Inconsistent log-message naming** — spaces vs snake_case (`series_edit.go:99` vs `:192`). AGENTS.md asks for stable snake_case keys.
- **P2-6 · Ignored errors on bot session writes / UUID parses** — `meetingedit`, `scheduleview`, `checker` (`_ = sessions.Set(...)` ~20 sites) — at least `Warn`-log failed session writes (lost edit state, silent).
- **P2-7 · N+1 lookups** — `application/conflict.go:72-81` (participant/user per overlapping meeting + `personName` per email); `meeting_repo.go:96-105` participant INSERT-per-row in a loop (batch for large series).
- **P2-8 · Oversized frontend files** — `meeting-form.tsx` (367, extract schema/defaults like `event-type-dialog-helpers.ts`); `responses-page.tsx` (288, extract CSV-export + filter bar); mini-app `meeting-create-page.tsx` (322, extract a `useConflictCheck` hook — conflict-check is a raw fetch outside Query).
- **P2-9 · Mini-app polish** — emoji in JSX (`meetings-list-page.tsx:64` `🔁`) → lucide icon; `duration` accepts `NaN` (`checker-page.tsx:77`, guard `<= 0` misses NaN); `checker-page.tsx:24-48` `catch {}` swallows server error detail; dead theme-params plumbing in `telegram-env.ts:101-137` (`resolveAccentFromTelegram` always returns default).
- **P2-10 · `book.$slug.tsx:137` refetches on locale change** — `locale` in the fetch `useEffect` deps discards in-progress slot selection; resolves with the Query-hook migration (P1-11).
- **P2-11 · `api-client` violates repo `semi: false` prettier rule** — `packages/api-client/src/{index,client}.ts` use semicolons; run prettier (generated `schema.ts` correctly excluded).
- **P2-12 · Toast-helper inconsistency** — `survey-dialog.tsx:102,111` uses `toastApiError` while the rest uses `toastError(error, t, key)`.
- **P2-13 · OG/JSON-LD asset paths as string literals** — `landing-meta.ts:79` (`/og-image.png`, `/logo.svg`) should come from `@leadcat/brand` constants so a brand rename is type-checked, not a runtime 404.

---

## Corrected / non-issues (checked, not problems)

- **Cross-app `group-series`/`WEEKDAYS`/`toggleDay`/`groupBySeries`** — already extracted to `@leadcat/types` and imported by both apps; the old `lib/` dirs are empty. Not duplication.
- **`formatTimeRange` in admin vs mini-app** — *not* true duplication: admin operates on `starts_at`/`ends_at` ISO datetimes, mini-app on `date`/`start`/`end` strings (divergent meeting models). Not a good extraction candidate without unifying the API contracts.
- **`packages/ui/.../calendar.tsx` (198)** — vendored shadcn/react-day-picker; length is inherent, a11y delegated upstream. Acceptable.
- **`apps/admin/app/shared/lib/cn.ts`** — a thin re-export of `@leadcat/ui`'s `cn`, not a duplicate implementation.
- **Repo-level multi-row writes** — `CreateMeetingSeries`, `UpdateMeetingsTx`, `CreateOrganization` use explicit transactions correctly (the P0-2 gap is at the orchestration layer, not the repo).

---

## Prioritized remediation backlog

Ordered by value ÷ effort.

| # | Item | Findings | Effort | Why now |
|---|---|---|---|---|
| 1 | Add `platform-no-direct-db` depguard rule + flip the 10 bot packages to `application/model` types | P0-1 | M | Stops the boundary leak from spreading; makes the next refactor enforceable |
| 2 | Fix SSR `<html lang>` in all three roots; delete client lang-sync + no-op meta | P0-3, P1-12 | S | SEO/a11y correctness; tiny, high-confidence |
| 3 | `c.Context()` → `c.UserContext()` across mini-app handlers | P1-7 | S | Restores `request_id`/audit-actor in logs; mechanical |
| 4 | Add `Switch`/`Textarea`/`WeekdayPicker`/`Field` to `@leadcat/ui`; replace call sites | P1-10 | M | Removes 6-file duplication + a11y gaps across both frontends |
| 5 | Wrap series/create write orchestration in transactions; reconcile calendar outside | P0-2 | M–L | Data-integrity under partial failure |
| 6 | Decide CQRS: finish the split or delete the half-built packages | P1-1, P1-2, P1-3 | L | Unblocks the largest backend maintainability debt; removes forced duplication |
| 7 | Split `meetingedit/service.go` (734→4 files); reclassify platform features → `delivery/telegram` | P1-4, P1-5 | M–L | Pairs with #1; restores the platform boundary |
| 8 | Idempotency: Redis claim for agent booking; per-recipient claims in notifier update/cancel | P1-9 | M | Prevents double-book / duplicate pings on retry in multi-replica |
| 9 | Extract `resolveMeetingOp` + shared error-mapper; route public surfaces through Query layer | P1-8, P1-11 | M | Shrinks fat handlers; unifies public/authed data access |
| 10 | Generic `botsession.Store[T]`; generic Telegram `toMarkup[T]`; `platform/tz` helper | P1-6, P2-1, P2-2 | M | Collapses repeated infrastructure |
| 11 | P2 polish sweep (error-code/log naming, NaN guard, emoji→icon, prettier, oversized files) | P2-* | S–M | Low-risk consistency cleanup |

**Suggested first cut (low-risk, high-confidence):** items 1–4 — all small/medium, each independently shippable, no behavior change to product flows. Items 5–6 are the strategic backend decisions worth doing deliberately.
