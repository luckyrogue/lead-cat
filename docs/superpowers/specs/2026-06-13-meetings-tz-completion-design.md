# Finish the Meetings ТЗ — design

Date: 2026-06-13
Status: approved (pending spec review)
Topic: closing the remaining ТЗ polish gaps for the meetings product

## Context

The meetings product (Telegram Mini App + web admin) implements essentially all
of the meetings ТЗ: full CRUD, recurring series (daily/weekly/monthly/custom
weekdays), participant add/remove, conflict checking, free-slot finder,
reminders, and org/member/invite management. A feature audit against
`docs/NEW-FEATURES.md` / `docs/MEETINGS.md` surfaced four remaining polish gaps.
This epic closes them in a single spec, implemented in sequence.

A key audit correction: a recurring series is stored as **N concrete `meeting`
rows sharing a `series_id`**, and per-occurrence edit (`scope=this`) patches just
that one row **without clearing `series_id`**. So skip (cancel `scope=this`) and
move (edit `scope=this`) of a single occurrence already work today and stay in
the series. "Recurring exceptions" is therefore not a data-model change — it is
UX clarity plus one genuinely-missing capability (editing a series' end date
after creation).

## Goals

Close these four gaps:

1. Web admin meeting filters (date range / department / organizer / status).
2. Telegram bot commands `/menu`, `/new`, `/help`, `/admin`.
3. Per-user timezone and language (ru/en/kk), affecting both display and input.
4. Recurring-series UX: explicit "skip this occurrence", series grouping in
   lists, and an editable series end date.

## Non-goals

- Microsoft/Outlook calendar integration and the cross-calendar availability
  wedge (separate, larger effort — deferred).
- Billing/plans, analytics/reporting, granular permissions beyond
  owner/admin/member (out of ТЗ scope).
- Per-meeting reminder overrides (ТЗ specifies reminders are global per user).

## Ground rules (cross-cutting)

- **Contract-first.** `apps/backend/openapi/openapi.json` is the source of truth.
  Every backend surface change updates it and regenerates `@leadcat/api-client`;
  both frontends derive their meeting/settings types from the generated client.
- **CQRS preserved.** Reads go through `query.Meetings`; writes through
  `command.Meetings`. Filters are queries. Series-reshape and settings writes are
  commands. Query handlers stay side-effect free.
- **Two identities.** Per-user settings live on both `bot_users` (Telegram) and
  `platform_users` (web), each with a fallback to the org timezone / default
  language. No new unified settings table — mirror the existing
  `bot_users.reminder_minutes` precedent.
- **Layering.** New ports stay in `application`/`domain`; infrastructure
  implements them. depguard already forbids `internal/infrastructure` imports
  from `application`.

## Sequencing caveat

There is a large in-flight refactor in the working tree (CQRS command bus,
`@leadcat/types` package, admin/mini-app page splits, frontend types moved to
`@leadcat/api-client`, comment removal). These four features touch the same
files (`meetings-page`, `miniapp_write`, settings API, meeting types/forms).
**Implementation must start after that refactor is committed (or rebase onto
it)** to avoid conflicts. Each feature is implemented and merged in order before
the next begins (the established ff-merge-to-main cadence).

---

## Feature 1 — Web admin meeting filters (small)

**Backend.** Extend `GET /api/orgs/:id/meetings` with optional query params:

- `status` — `scheduled` | `cancelled` | `all` (default `all`, i.e. no status
  filter, preserving today's behavior where the list returns every meeting).
- `from`, `to` — ISO dates; inclusive range on `starts_at`.
- `dept` — exact match on the meeting `dept` field.
- `organizer` — member user id (`organizer_user_id`).

Add `Repository.ListMeetingsFiltered(ctx, orgID, MeetingFilter)` and a query
handler; the existing `ListMeetings` becomes the no-filter path (or delegates
with an empty filter). Filtering is server-side SQL (date/dept/organizer/status
predicates), keeping pagination-friendliness for orgs with many meetings.
Update openapi + regenerate api-client.

**Admin UI.** A filter bar above `MeetingsTable`:

- date-range inputs, status select, department select, organizer select
  (options sourced from the org's members and/or the distinct depts present).
- TanStack Query key includes the filter object; changing a filter refetches.
- Empty/default filter renders today's list unchanged.

**Errors.** Bad date params → 400. Unknown organizer/dept → empty result (not an
error).

---

## Feature 2 — Bot commands (small-medium, backend-only)

Register four commands in the bot and publish them via `SetMyCommands` so they
appear in Telegram's command menu:

- `/menu` — reply with a WebApp button opening the Mini App home.
- `/new` — open the Mini App deep-linked to the create screen.
- `/help` — static text: what the bot does + the command list.
- `/admin` — bot admins get a WebApp button to the admin section; non-admins get
  a short "you're not an administrator" reply.

**Mini App.** Small addition: read the Telegram start-param
(`tgWebAppStartParam`) on launch and route `/new` to the create screen. No other
frontend change (the screens already exist).

**Errors.** Unknown/unauthorized `/admin` use returns the polite non-admin reply,
not an error.

---

## Feature 3 — Per-user timezone + language (largest; ru/en/kk; display + input)

**Migration.** Add nullable `timezone TEXT` and `language TEXT` to `bot_users`
and `platform_users`. Null → fall back to org `tz` / default language (`ru`).

**Backend.**

- Mini App: extend `GET/PATCH /api/miniapp/settings` to include `timezone` and
  `language` alongside reminders.
- Web: add `GET/PATCH /api/me/settings` for platform users (timezone, language).
- Replace the hardcoded `almatyLoc()` in mini-app meeting DTO rendering with the
  caller's effective timezone (user setting → org tz → default).
- Create/update parse `date`+`time` in the **user's** effective timezone (not the
  org tz) to produce the correct UTC `starts_at`/`ends_at`.
- Validate `timezone` against the IANA database (`time.LoadLocation`) and
  `language` against the supported set `{ru, en, kk}`; invalid → 400.

**Frontend.**

- i18n extended to **ru / en / kk** in the Mini App and admin. Every user-facing
  string gets all three translations.
- Settings/profile gains a language selector and an IANA-timezone picker
  (curated common-zones list with search), persisted to the backend and applied
  immediately.
- All meeting times render in the user's selected timezone; create/edit time
  inputs default to it and convert on save.

**Errors.** Invalid timezone or unsupported language → 400 with a clear code.

---

## Feature 4 — Recurring series: skip + grouping + editable end date (medium)

**Skip this occurrence.** Surface the existing `cancel scope=this` as an explicit
"Skip this occurrence" action on a series occurrence in both apps. Labeling /
affordance only — no new backend.

**Series grouping.** In both meeting lists, group occurrences sharing a
`series_id` under one collapsible series header (show the next occurrence + a
count); expanding reveals individual occurrences (each still skippable/editable).
Frontend-only — the data already carries `series_id` and `rec`.

**Editable series end date (the real backend work).** A `command` to reshape a
series' tail by changing its `recurrence_until`:

- **Extend** (new date later): generate the additional future occurrences from
  the series' cadence, create their Google events, and persist them under the
  same `series_id` (reuse the create-series rollback pattern on partial failure).
- **Trim** (new date earlier): cancel future occurrences beyond the new end date
  and delete their Google events (best-effort, logged).
- Only future occurrences are affected; past/cancelled rows are untouched. Auth:
  organizer or org owner (reuse `ownerOrOrganizer`).
- New endpoint on web + mini-app (e.g. `PATCH .../meetings/:id/series-end`);
  openapi + api-client updated.

**Errors.** Non-series meeting → 400; new end date before the series start →
400; forbidden → 403; reuse the existing not-editable mapping.

---

## Error handling (summary)

- Input validation (dates, timezone, language, series end) → `ErrInvalidInput` →
  400 with stable codes.
- Authorization reuses `ownerOrOrganizer` → `ErrForbidden` → 403.
- Google event create/delete in series reshape is best-effort with the same
  rollback/log pattern as series creation; a calendar failure on extend rolls
  back the new rows.

## Testing

- **Backend**: table tests for filter-query predicate building, user-timezone
  parsing (create/update), and series-reshape occurrence delta (extend/trim
  counts and boundaries); handler tests extended where they already exist.
- **Frontend**: the established `typecheck` + `lint` + `build` gates per app;
  i18n completeness checked by the existing key structure (no missing-key
  fallback in production paths).

## Build order (one phased plan)

1. **Web meeting filters** — isolated quick win, validates the contract-regen
   loop on a small surface.
2. **Bot commands** — isolated, backend-only.
3. **Recurring series** — skip + grouping (frontend) + editable end date
   (backend command).
4. **Per-user timezone + language** — heaviest and most cross-cutting (migration,
   both identities, both settings endpoints, DTO tz rendering, both frontends'
   i18n + tz input/display); done last so it builds on a settled base.

Each phase ships independently (build/test/lint green, openapi+api-client synced)
and merges to main before the next begins.
