# Slice B — Recurrence (series everywhere)

**Date:** 2026-06-07
**Branch (target):** `feat/meetings-recurrence-b` (from `main`).
**Roadmap:** see [2026-06-06-roadmap-to-beta-design.md](./2026-06-06-roadmap-to-beta-design.md) — second slice on the path to ТЗ beta.
**Predecessor:** Slice A — TMA writes (non-recurring) finish (merged on `main` 2026-06-07, `47ced23`).

## Goal

Enable recurring Google Meet meetings end-to-end through the TMA. After this slice:

- An authed Mini App user can create a daily / weekly / monthly / custom-weekdays meeting with a required end date.
- They can edit a single occurrence (`scope=this`) or the whole series (`scope=whole`).
- They can cancel a single occurrence or the whole series.
- The conflict-warning wizard step shows real conflicts for every occurrence in the series, grouped by date.
- The slice-A guardrail (`meetings_recurring_unsupported`, `recurringBlocked` in wizard) is removed.

## Decisions locked

| #   | Question                  | Decision                                                                                                                              |
| --- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Custom weekday recurrence | Add `Custom` to backend `Recurrence` enum; drop unused `Biweekly`. Domain `Input` gains `RecurrenceDays []int` (1=Mon..7=Sun).        |
| 2   | Edit/cancel scope set     | `scope=this \| whole` per ТЗ §4.4.2 literal. "whole" operates on the entire series (including past occurrences) keyed by `series_id`. |
| 3   | Recurrence end date UX    | Required date picker with smart defaults per rec kind. Validation: `until >= start_date`.                                             |
| 4   | Series conflict warning   | Full-series check — backend expands occurrences and returns occurrence-grouped conflicts.                                             |

## ТЗ alignment

- §4.1.3 «Частота повторения» — non-once asks for an end date.
- §4.4.2 «Редактирование повторяющихся встреч» — "this only" / "whole series".
- §4.5 (deletion) — same scope choice for cancel.
- §4 schema notes (line 820–822) — `recurrence VARCHAR (none/daily/weekly/monthly/custom)`, `recurrence_days JSON`, `recurrence_end DATE`.

## Backend changes

### Domain — `backend/internal/domain/meeting/`

**`meeting.go` (`Recurrence` enum):**

- Add `Custom Recurrence = "custom"`.
- Remove `Biweekly` constant and its `recurrenceLabels` entry.
- `Valid()` derives from the labels map — no extra code change.
- Add `RecurrenceDays []int` field to `Input` (zero/nil for non-custom; 1..7 ISO weekday for custom).

**`recurrence.go`:**

- `Occurrences(start, end time.Time, r Recurrence, days []int, until time.Time) ([]Span, error)` — signature gains `days`.
- `r == Custom`: step by one day; emit only when `weekday ∈ days`. Use `time.Weekday()` mapped to 1..7 (Sun=7).
- `r == Custom` with empty/invalid `days` → new `ErrRecurrenceDays = errors.New("custom recurrence needs at least one weekday")`.
- All existing kinds unchanged.
- `ErrTooManyOccurrences` cap unchanged.

**`validate.go`:**

- `Custom` requires non-empty `RecurrenceDays` (uses `ErrRecurrenceDays`).
- All non-once kinds require `until` (existing behavior via `ErrRecurrenceWindow`).

**`naming.go`:**

- `GenerateName(...)` — no change. `Recurrence.Label()` already covers Custom once the labels map has the entry.

**Tests (`recurrence_test.go`, `validate_test.go`):**

- TDD: write the failing tests for Custom (3-day-a-week pattern over a 4-week span) before extending `Occurrences`.
- Update any existing test referencing `Biweekly` to use Custom or remove if redundant.

### Persistence — `backend/internal/infrastructure/persistence/postgres/`

**Migration (next available number under `backend/migrations/`):**

```sql
ALTER TABLE meetings ADD COLUMN recurrence_days JSONB;
```

Nullable; safe online (no rewrite). No backfill needed (existing rows are non-custom → NULL).

**`meetings.go`:**

- `Meeting` struct gains `RecurrenceDays []int` (use `pgtype.JSONB`-style scan/marshal via existing helpers; if none, use `json.RawMessage` + helpers).
- `CreateMeetingSeries`: persist `recurrence_days` for series whose primary recurrence is Custom.
- `UpdateMeetingsTx`, `CancelSeriesOccurrences`: unchanged (no field changes).
- New: `ListSeriesAllOccurrences(ctx, workspaceID, seriesID) ([]Meeting, error)` — all occurrences for the series regardless of `StartsAt`.
- New: `CancelAllSeriesOccurrences(ctx, workspaceID, seriesID) (int, error)` — cancels entire series.

### Application — `backend/internal/application/`

**`meeting_service.go` (`CreateMeeting`):**

- Accept `Input.RecurrenceDays`; pass through to materialization.
- Pre-existing series materialization stays; just routes Custom days through `Occurrences`.

**`series_edit.go` (existing `UpdateSeries`/`CancelSeries`):**

- No semantic change. Add a one-line `// internal: not exposed via TMA HTTP; slice E admin scope may use this` comment.

**`series_edit.go` (new):**

- `UpdateWholeSeries(ctx, workspaceID, userID uuid.UUID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error)`:
  - Same auth/validation as `UpdateSeries`.
  - Calls `ListSeriesAllOccurrences` (all occurrences, not filtered by `picked.StartsAt`).
  - Otherwise same body: `applySeriesUpdate` per occurrence, `UpdateMeetingsTx`, best-effort Google patch, enqueue one `MeetingUpdated`.
- `CancelWholeSeries(ctx, workspaceID, userID uuid.UUID, meetingID uuid.UUID) (int, error)`:
  - Same auth/validation as `CancelSeries`.
  - Calls `CancelAllSeriesOccurrences`.
  - Best-effort Google delete for all events; enqueue one `MeetingCancelled`.

**`conflict.go` (new):**

- `MeetingSeriesConflicts(ctx context.Context, emails []string, firstStart, firstEnd time.Time, r meeting.Recurrence, days []int, until time.Time) (map[time.Time][]Conflict, error)`:
  - Calls `meeting.Occurrences(firstStart, firstEnd, r, days, until)` to expand.
  - For each `Span`, runs existing per-occurrence conflict query.
  - Aggregates by occurrence start (date+time UTC). Empty-conflict occurrences are omitted from the map.
- Existing `MeetingConflicts` (single-occurrence) stays.

### Delivery — `backend/internal/delivery/http/handlers/`

**`tma_write.go`:**

- `tmaCreateRequest`: add `RecurrenceUntil *string` (`YYYY-MM-DD`), `RecurrenceDays *[]int`. `toCreateMeetingInput` parses `RecurrenceUntil` as Almaty date and assigns to `Input.RecurrenceUntil`; assigns `Input.RecurrenceDays`. Validation uses existing domain `Validate()`.
- `TMACreateMeeting` handler: remove the slice-A early 400 `meetings_recurring_unsupported`. The domain `Validate()` now handles the recurrence requirements.
- `TMAUpdateMeeting`: read `scope` from `c.Query("scope")`; default to `"this"`; values `"this"` / `"whole"` accepted; anything else → 400 `validation_failed`. `"this"` → existing `UpdateMeeting` path. `"whole"` → `UpdateWholeSeries` with same `SeriesUpdateInput` build helper.
- `TMADeleteMeeting`: same `scope` parsing; `"this"` → existing `CancelMeeting`; `"whole"` → `CancelWholeSeries`. Status 204.

**`tma_write.go` — conflicts (`TMAConflicts`):**

- Request gains optional `Recurrence *string`, `RecurrenceUntil *string`, `RecurrenceDays *[]int`.
- When `Recurrence` present and `!= "once"`: parse, call `MeetingSeriesConflicts`, return `{"occurrences": [{"date","start","end","conflicts":[…]}, …]}`.
- When `Recurrence` absent or `"once"`: keep existing single-shot path but **wrap response in the new shape** for uniformity — `{"occurrences": [{"date","start","end","conflicts":[…]}]}` with one entry.
- Each `conflicts` array uses the existing `tmaConflictDTO` from slice A (`toConflictDTO`).
- Date/start/end fields rendered in Almaty TZ.

**`app.go`:** no new routes (the three slice-A routes carry the new params).

### OpenAPI — `backend/openapi/openapi.json` (+ mirror to `backend/docs/openapi.json`)

- `POST /api/tma/meetings`: request schema adds optional `recurrence_until` (string, format=date), `recurrence_days` (array of integer 1..7).
- `PATCH /api/tma/meetings/{id}`: add `scope` query param (`enum: [this, whole]`, default `this`).
- `DELETE /api/tma/meetings/{id}`: same `scope` query param.
- `POST /api/tma/conflicts`: request adds the three optional fields; response schema replaced with the occurrence-grouped shape.
- New component schema `TmaOccurrenceConflicts { date, start, end, conflicts: TmaConflict[] }`.
- Regen `frontend/src/shared/api/generated/schema.ts` from the updated OpenAPI.

## Frontend changes

### `entities/meeting/` & `features/meetings/api.ts`

- `Meeting.seriesId?: string` — derived from new DTO field `series_id`. `MeetingDTO` and `toMeeting` flow it through.
- `MeetingInput` gains `recurrence_until?: string`, `recurrence_days?: number[]`.
- `MeetingPatch` unchanged (`scope` is a separate param to the mutation).
- `Conflict` unchanged.
- New: `OccurrenceConflicts { date: string; start: string; end: string; conflicts: Conflict[] }`. `fetchConflicts` return type becomes `OccurrenceConflicts[]`.
- `ConflictsParams` gains `recurrence?: string`, `recurrenceUntil?: string`, `recurrenceDays?: number[]`. Body mapping is camel→snake.
- `updateMeeting` / `deleteMeeting` signatures: `(id, patch?, opts?: { scope?: "this" | "whole" })` — scope defaults to `"this"`, sent as `?scope=…`.

### `features/meetings/queries.ts`

- `useUpdateMeeting` mutation args: `{ id: string; patch: MeetingPatch; scope?: "this" | "whole" }`.
- `useDeleteMeeting` mutation args: `{ id: string; scope?: "this" | "whole" }`.
- `useConflicts` return: `OccurrenceConflicts[]`. Invalidation unchanged.

### `features/meetings/lib/write-error.ts`

- No new mapped codes for slice B (backend reuses existing codes). Optionally add `recurrence_days_required` → `errGeneric` (or a new i18n key) if backend surfaces it; defer to TDD.

### `features/meeting-create/lib/use-create-wizard.ts`

- `MeetingDraft.until: string` added (init `""`); reset to smart default when `rec` changes:
  - `daily` → `+30d`
  - `weekly` → `+12w`
  - `monthly` → `+12m`
  - `custom` → `+12w`
  - `once` → `""` (cleared)
- Remove `recurringBlocked` flag (and the export from the hook).
- `useConflicts` call: pass `recurrence`, `recurrence_until`, `recurrence_days` when `rec !== "once"`.
- `canNext` for review step gains: if `rec !== "once"`, require `draft.until` non-empty AND `draft.until >= draft.date`.

### `features/meeting-create/components/wizard-step-when.tsx`

- When `draft.rec !== "once"`, render a date input labeled `t("untilLabel")` below the rec selector. Min = `draft.date`; default placeholder `t("untilPlaceholder")`.
- Invalid (empty or `< date`): show inline error `t("untilRequired")` / `t("untilBeforeStart")` and the wizard's `canNext` gating prevents advancing.

### `features/meeting-create/components/wizard-step-review.tsx`

- Replace single `conflictPeople` block with the grouped-by-date list. Render up to 5 dates; "+ еще N" overflow with `t("seriesConflictsMore")`.
- Drop the slice-A `recurringSoon` warning entirely.
- For non-once: header reads `t("seriesConflicts")`; for once: existing single-block render against `[0]` of the occurrences array.

### `features/meeting-create/components/create-wizard.tsx`

- Confirm button disabled rule no longer references `recurringBlocked`. Standard `!canNext` is sufficient.

### `features/meeting-create/pages/create-page.tsx`

- `completeCreate`: when building `MeetingInput`, pass `recurrence_until: m.until || undefined` and `recurrence_days: m.recDays?.length ? m.recDays : undefined`.
- Edit path: build `MeetingPatch` as today; pass `scope` from URL (router param to be added — see Routes below).

### `features/meetings/components/meeting-detail-actions.tsx` + `meeting-detail.tsx`

- `meeting-detail.tsx` accepts `m: Meeting` with `seriesId?: string`.
- When `seriesId` present (and `canManage && !past`): render `<MeetingDetailActions>` in a "series-aware" variant — two grouped buttons each:
  - `t("edit")` + dropdown `t("editThis")` / `t("editSeries")`
  - `t("del")` + dropdown `t("delThis")` / `t("delSeries")`
- Implementation can be a bottom sheet with three buttons (cancel, this, series) instead of dropdown — cat-design pattern fits sheets better. Decision deferred to T12 implementation; both flow into the same `onEdit(scope)` / `onDelete(scope)` callbacks.
- `onEdit(scope)` navigates `/meetings/create/$editId?scope=…`; `onDelete(scope)` calls `useDeleteMeeting().mutateAsync({ id, scope })`.

### Routes / search params

- `meetings/create/$editId` route: add `scope` to its search params (`scope: "this" | "whole"` optional; default `"this"`).
- `create-page.tsx` reads `scope` from search; if `scope === "whole"`, the wizard locks date/recurrence/until via a flag.

### Whole-series edit field locking

- `useCreateWizard` accepts `lockedFields?: { date?: boolean; rec?: boolean; until?: boolean; participants?: boolean }`. When `scope=whole`:
  - Date is read-only.
  - Recurrence selector + until picker read-only.
  - Participants list is read-only (matches `SeriesUpdateInput` which doesn't change participants).
- A localized banner in step-when: `t("seriesEditLockedNote")`.

### i18n keys (ru/kk/en)

Add to `frontend/src/shared/tma/i18n.ts`:

- `untilLabel`, `untilPlaceholder`, `untilRequired`, `untilBeforeStart`
- `seriesConflicts`, `seriesConflictsMore`
- `editThis`, `editSeries`, `delThis`, `delSeries`
- `seriesEditLockedNote`

Remove (no longer used): `recurringSoon` (slice A guardrail). Keep `recurringSoon` if `write-error.ts` still maps `meetings_recurring_unsupported` defensively — backend stops emitting it but other servers in transition might. Decision: keep the key + mapping for one slice, remove in slice H.

## Notifications

- `CreateMeeting` with recurrence already routes through `enqueueCreated` once per series (slice A behavior).
- `UpdateWholeSeries` enqueues one `MeetingUpdated` (matching existing `UpdateSeries`).
- `CancelWholeSeries` enqueues one `MeetingCancelled`.
- `UpdateMeeting` / `CancelMeeting` (scope=this) — unchanged: one notification per occurrence.

No worker changes; the worker already handles created/updated/cancelled.

## Conflict expansion cost note

`MeetingSeriesConflicts` runs N participant-queries per occurrence (existing per-occurrence path). With weekly × 12 weeks × 8 participants ≈ 12 occurrence-checks, fast. Daily × 30 days × 8 participants ≈ 30 checks, still acceptable. If oncall observes latency on this route in slice H, batch into one SQL with `tstzrange &&`. Not premature optimization yet.

## Task list (~14 sequential)

| #     | Task                                                                                                            | Layer          |
| ----- | --------------------------------------------------------------------------------------------------------------- | -------------- |
| B-T0  | Branch `feat/meetings-recurrence-b` from `main`                                                                 | git            |
| B-T1  | Domain: `Custom`, drop `Biweekly`, `Input.RecurrenceDays`, `Occurrences` extension, validate + TDD              | Go domain      |
| B-T2  | Migration `recurrence_days JSONB` + `postgres.Meeting` field + scan/write                                       | Go infra       |
| B-T3  | Application: `ListSeriesAllOccurrences`, `CancelAllSeriesOccurrences`, `UpdateWholeSeries`, `CancelWholeSeries` | Go application |
| B-T4  | Application: `MeetingSeriesConflicts` + test                                                                    | Go application |
| B-T5  | HTTP: create accepts `recurrence_until`/`recurrence_days`; remove `meetings_recurring_unsupported` block        | Go delivery    |
| B-T6  | HTTP: `scope=this\|whole` on PATCH + DELETE                                                                     | Go delivery    |
| B-T7  | HTTP: conflicts accepts recurrence params; returns occurrence-grouped shape (uniform for once + series)         | Go delivery    |
| B-T8  | OpenAPI 3 changes (parallel mirror) + frontend schema regen                                                     | docs + ts      |
| B-T9  | Frontend types/mutations: `scope`, `recurrence_until`, `recurrence_days`, `OccurrenceConflicts`                 | frontend       |
| B-T10 | Wizard: until-picker, smart defaults, validation; drop `recurringBlocked`                                       | frontend       |
| B-T11 | Wizard review: grouped-by-date conflicts + i18n keys                                                            | frontend       |
| B-T12 | Meeting detail: dual-scope edit/cancel; series_id DTO field; whole-edit lockedFields + banner                   | frontend       |
| B-T13 | Docs refresh (`MEETINGS.md`, `API.md`) + `make test/lint/build` + `pnpm typecheck/format/build`                 | docs + verify  |

## Acceptance

- Create a weekly meeting with a custom 3-weekday pattern, end-date `+12w`. Backend materializes occurrences; Google events created; one `MeetingCreated` enqueued.
- On review step, real conflicts list grouped by date (when any participant has overlaps).
- Edit one occurrence: only that row changes; Google patches one event; one `MeetingUpdated`.
- Edit whole series: all occurrences change; Google patches each; one `MeetingUpdated`.
- Cancel one / cancel whole: same scope-aware behavior; one `MeetingCancelled`.
- All flows organizer-only (403 otherwise), admin overlay carries over from slice A.
- `make test`, `make lint`, `make build` green; `pnpm -C frontend typecheck/format/build` green.

## Risks / open at implementation

- Biweekly removal: ensure no fixtures/migrations/tests reference `"biweekly"`. Grep before T1.
- Wizard "edit whole" UX (dropdown vs bottom sheet) — decide at T12; design-wise the bottom sheet pattern matches cat-design.
- Drop `recurringSoon` from i18n now or in slice H — leaning H to keep blast radius minimal.
- `ListSeriesAllOccurrences` and `CancelAllSeriesOccurrences` need careful index review; series_id should already be indexed from earlier work, verify in T2.

## Out of scope (deferred to later slices)

- Per-occurrence exception edits as RRULE EXDATE (true Google-Calendar-style "edit this only" creates an exception event linked to the series). For Slice B, "edit this" mutates the materialized row independently; Google patches only that event. Acceptable since DB is SoT.
- Admin scope for series writes (slice E).
- "This and following" scope (`scope=forward`) — backend has `UpdateSeries`/`CancelSeries` internally; expose later if needed.
- Per-occurrence reminder overrides (slice F).
