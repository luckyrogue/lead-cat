# Recurring-series creation engine (materialized occurrences) — design

**Date:** 2026-05-31
**Scope:** Make recurrence real by materializing a recurring meeting into individual occurrence rows at creation. Prerequisite for §4.4.2 (edit this/whole series), which is a separate later increment. Editing/deleting series is out of scope here.

## Approach (chosen: materialize occurrences)

When a meeting is created with `recurrence != once`, expand the series into **N individual meeting rows** linked by a `series_id`; each occurrence is an ordinary single meeting (its own Google event, its own reminders). The series is bounded by a required `recurrence_until` date plus a defensive cap (≤100 occurrences). This reuses the entire existing engine (reminders, notifications, edit, delete operate per-row).

**Accepted tradeoffs of materialization (vs a true Google RRULE event):**
- Each occurrence has its **own Google event and its own Meet link** (not a single recurring series in attendees' calendars).
- N synchronous `CreateEvent` calls in one HTTP request — latency grows with N; the ≤100 cap bounds the worst case (realistic weekly series are ~12–52).
- No "infinite" series — only up to `until`; re-materialization of open-ended series is a deferred follow-up.

`recurrence == once` keeps the current single-meeting path unchanged (`series_id = NULL`).

## Domain

A pure occurrence-expansion function in `internal/domain/meeting`:

```go
type Span struct{ Start, End time.Time }

func Occurrences(start, end time.Time, r Recurrence, until time.Time) ([]Span, error)
```

- `once` → a single span `{start, end}`.
- Otherwise step from `start`: `daily +1d`, `weekly +7d`, `biweekly +14d`, `monthly` via `AddDate(0, 1, 0)`. Each occurrence keeps the same duration (`end - start`).
- Emit spans while the occurrence start's **date** is `<= until` (inclusive).
- Cap at `maxOccurrences = 100`; exceeding it returns `ErrTooManyOccurrences`.
- Validation (in the command, wrapping `ErrInvalidInput`): when `recurrence != once`, `until` must be set and `until.date >= start.date`.

`ErrTooManyOccurrences` is a domain sentinel; the command wraps it as `%w ErrInvalidInput` so the HTTP layer maps it to 400.

## Migration + model + repo

**Migration** `20260531130000_meeting_series.sql`:
```sql
-- +goose Up
ALTER TABLE meetings ADD COLUMN series_id UUID;
CREATE INDEX meetings_series_idx ON meetings (series_id);
-- +goose Down
DROP INDEX IF EXISTS meetings_series_idx;
ALTER TABLE meetings DROP COLUMN IF EXISTS series_id;
```
(`recurrence_until DATE` already exists in the meetings table.)

**Model:** `postgres.Meeting` gains `SeriesID *uuid.UUID` and `RecurrenceUntil *time.Time`. `meetingCols`, `meetingColsM`, `scanMeeting`, and the `CreateMeeting` INSERT all extend by `series_id, recurrence_until` (appended at the end of the column list and the scan, keeping every `scanMeeting`-based query consistent).

**Repo:** a transactional batch insert (DB all-or-nothing):
```go
func (s *Store) CreateMeetingSeries(ctx context.Context, ms []Meeting, ps []MeetingParticipant) ([]Meeting, error)
```
- `pool.Begin(ctx)` → for each meeting: `INSERT ... RETURNING <cols>` (scan into the result); for each participant: `INSERT meeting_participants (...) ON CONFLICT DO NOTHING` against the just-inserted meeting's ID — all within the tx → `Commit` (or `Rollback` on any error). Returns the inserted rows (with IDs) in order.

## CreateMeeting flow (`application`)

`CreateMeetingInput` gains `RecurrenceUntil string` (YYYY-MM-DD). The HTTP handler (`delivery/http/handlers/meetings.go`) maps the new field from the request body.

`Services.CreateMeeting`:
1. Parse `start`/`end` (as today) and `until` (if provided) in the workspace TZ; domain-validate, including "`until` required when `recurrence != once`" and `until.date >= start.date` (wrap `ErrInvalidInput`).
2. `spans, err := meeting.Occurrences(start, end, rec, until)` (one span for `once`).
3. `seriesID`: `nil` when `once` (single span); otherwise `uuid.New()`.
4. **`once` path (unchanged):** current single `CreateEvent` + `CreateMeeting` + `AddParticipants` + enqueue `meeting:created`.
5. **Series path:**
   - Resolve `calSvc := Calendar.For(ws)` once (a missing-Google workspace → `ErrGoogleNotConfigured` → 400, before any creation).
   - **Phase 1 — Google:** loop spans → `CreateEvent(span)` → collect `(eventID, meetLink)`; build the `Meeting` row (with `SeriesID`, `RecurrenceUntil`, span times, recomputed `name = GenerateName(dept,type,host, span.Start, rec)`). On any `CreateEvent` error → best-effort `DeleteEvent` the already-created events, return wrapped error (nothing in DB yet).
   - **Phase 2 — DB:** `Store.CreateMeetingSeries(rows, participants)` (one transaction). On error → best-effort `DeleteEvent` all created Google events, return error.
   - Enqueue `meeting:created` **once** for the anchor (first inserted row), best-effort (Warn on failure).
   - Return the anchor row with `Participants` set.

**Participants:** the same guest list is attached to **every** occurrence (inserted within the tx).

**Atomicity:** DB is all-or-nothing (transaction). Google orphans are possible only if a compensating `DeleteEvent` also fails (rare) — recorded as a known edge.

## Notifications & reminders

- `meeting:created` fires **once per series** (anchor only) — not N times. The created DM shows the anchor occurrence; the name already carries the recurrence label (e.g. "Еженедельно") via `GenerateName`.
- Reminders are unchanged: each occurrence is its own row, so `ListUpcomingMeetings` reminds before each occurrence independently.

## Testing

- **Unit (domain `Occurrences`):** once → 1 span; weekly until a date → correct count/step/duration; biweekly/monthly; `until == start` → 1; `until < start` → error; exceeding the cap → `ErrTooManyOccurrences`.
- **Unit (domain validation):** `recurrence != once` without `until` (or `until < start`) → error; `once` ignores `until`.
- **Build-verified:** the application series orchestration in `Services.CreateMeeting` (occurrence loop, Phase-1 Google + compensation, Phase-2 `CreateMeetingSeries` tx, single `EnqueueMeetingCreated`), the migration, the `CreateMeetingSeries` tx repo method, and the `scanMeeting`/cols extension.
- **Full suite:** `make test && make lint && make build`.

> **Testability note:** `Services.CreateMeeting` depends on the concrete `*postgres.Store` (no DB in unit tests), so — consistent with the existing `CreateMeeting`, which has no application-level unit test — the series orchestration (single-enqueue, Google compensation, tx insert) is **build-verified + exercised manually**, not unit-tested. The genuinely pure logic (`Occurrences` and validation) lives in `domain/meeting` and **is** unit-tested. Keep new pure decisions (e.g. mapping a `Span` → a `Meeting` row) as small pure helpers so they stay testable where it adds value.

## Out of scope (recorded)

- §4.4.2 editing "this occurrence" vs "whole/future series" (next increment).
- Re-materialization of open-ended series (no `until`).
- A single shared Meet link per series (a consequence of approach A).
- Deleting a whole series (§4.5 for series) — separate.

## Relationship to the platform

Additive. Adds a `series_id` column + a transactional batch insert; reuses the existing reminder/notification/edit/delete machinery per occurrence. No changes to the notify-bot/scenario engine.
