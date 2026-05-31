# §4.7–4.8 — Time-conflict warning + common-free-time checker

**Date:** 2026-05-31
**ТЗ:** `docs/NEW-FEATURES.md` §4.7 (Предупреждение о накладке времени), §4.8 (Чекер общего свободного времени)
**Status:** design approved, ready for plan

## Summary

Two related meeting features, built on **one shared detection core**:

- **§4.7** — when a meeting's time/participants are chosen on create/edit, automatically warn if any participant (incl. organizer) already has an overlapping meeting. Non-blocking: the user may continue or change the time.
- **§4.8** — a checker that, given a participant set, a date range, and a desired duration, finds time windows when *all* participants are free.

## Key decisions (from brainstorming)

1. **Both surfaces.** Shared logic in `domain`/`application`, exposed via **bot FSMs** (`/edit` warning + new `/checker`) **and REST** (for the mocked Mini App create wizard + checker tab).
2. **Internal DB as the busyness source** (not Google freebusy). Consistent with §4.6 schedule view; lets the §4.7 warning show real meeting **names**; fully unit-testable; works in stub/local. **Trade-off:** external/personal Google Calendar events are not seen — only bot-created meetings.
3. **Working hours fixed at 09:00–18:00 Almaty** (ТЗ default). Configurable hours deferred.
4. **Weekends skipped** — only Mon–Fri are searched by the §4.8 checker ("рабочий день").
5. **Conflict is never blocking** (§4.7.3) — create/update logic is unchanged; the warning is advisory only.

## Scope boundary (explicit)

The ТЗ "create from slot" (§4.8.5) and "change time / continue" (§4.7.3) hand off into the **standard create wizard**, which today exists **only as the Mini App (on mocks)** — there is no bot create FSM.

- **REST** returns everything the Mini App needs to pre-fill its existing create wizard (slot + participants). The frontend prefill/navigation is **out of scope** (Mini App stays mocked).
- **Bot `/checker`** shows free slots as a **read-only result list**. The "Создать встречу на этот слот" button is **OUT OF SCOPE** — a bot create FSM is a separate, larger increment.

## Architecture

Dependencies point inward (`delivery`/`platform` → `application` → `domain`); the postgres repo implements an `application`-defined need.

### 1. `domain/meeting` (pure, unit-tested)

Two additions — pure interval math, no DB/Google awareness:

```go
// Overlaps reports whether two spans intersect (partial or full). §4.7.1
func Overlaps(aStart, aEnd, bStart, bEnd time.Time) bool

// FreeSlots subtracts merged busy spans from one working window [winStart,winEnd),
// returning the gaps whose duration ≥ minDur, in chronological order. §4.8.3
func FreeSlots(busy []Span, winStart, winEnd time.Time, minDur time.Duration) []Span
```

- `Overlaps` = `aStart < bEnd && bStart < aEnd` (half-open; touching edges do NOT overlap).
- `FreeSlots`: sort busy by start, merge overlapping/adjacent, clip to the window, walk gaps, keep gaps ≥ `minDur`. The caller invokes it **once per working day** and skips Sat/Sun.
- Reuses the existing `Span{Start, End time.Time}` type from `recurrence.go`.

### 2. `application` (orchestration, build-verified)

Two methods on `Services`:

```go
type Conflict struct {
    Email       string
    PersonName  string    // employee/organizer display name (best-effort; falls back to email)
    MeetingName string
    Start, End  time.Time // conflicting meeting span (UTC)
}

// MeetingConflicts returns overlaps with the given span across all emails
// (participants + organizer), excluding excludeMeetingID (zero = none). §4.7
func (s *Services) MeetingConflicts(ctx, workspaceID uuid.UUID, emails []string,
    start, end time.Time, excludeMeetingID uuid.UUID) ([]Conflict, error)

type FreeSlot struct {
    Day        time.Time // start-of-day in workspace TZ
    Start, End time.Time // UTC
    Mins       int
}

// FreeSlots finds windows where ALL emails are free, within [from,to) (day-exclusive
// upper bound), Mon–Fri, 09:00–18:00 Almaty, gaps ≥ durMins. §4.8
func (s *Services) FreeSlots(ctx, workspaceID uuid.UUID, emails []string,
    from, to time.Time, durMins int) ([]FreeSlot, error)
```

- `MeetingConflicts`: one repo query returning plain overlapping `[]Meeting` (participants loaded). **Attribution is done in Go**: for each meeting, intersect its participant emails (and the organizer's email) with the query email set to determine which person(s) it conflicts; build one `Conflict` per (person, meeting). `PersonName` resolved via the employee directory (best-effort, falls back to email); the organizer's email is resolved via the existing user lookup.
- `FreeSlots`: load the participants' scheduled meetings across `[from,to)` once (same repo query). For free-slot math only the **union of busy spans** matters, so no per-person attribution is needed — every returned meeting's span is busy for the group. Then per weekday build the 09:00–18:00 window (in workspace TZ), collect the busy spans for that day, and call `meeting.FreeSlots(busy, winStart, winEnd, dur)`. Concatenate chronologically. Empty result is valid (§4.8.6).
- Working-hours constants (`workStart = 9h`, `workEnd = 18h`) live in `application` (single source).

### 3. `infrastructure/persistence/postgres`

One new query, generalizing `ListScheduleForEmail` to a **set of emails** + a **time-overlap** predicate:

```go
// ListMeetingsOverlapping returns scheduled meetings overlapping [from,to) where any
// of emails is a participant or the organizer. Participants are loaded on each row.
func (s *Store) ListMeetingsOverlapping(ctx, workspaceID uuid.UUID, emails []string,
    from, to time.Time) ([]Meeting, error)
```

- Reuses the participant/organizer join shape from `ListScheduleForEmail` (lines ~241–249), swapping `mp.email = $1` for `= ANY($emails)` and the `starts_at` BETWEEN filter for the half-open overlap predicate `starts_at < to AND ends_at > from`.
- Scoped by `workspace_id`, `status = 'scheduled'`.
- Returns **plain `[]Meeting`** (with participants populated, as elsewhere) — no per-row email wrapper. The application layer derives which queried person each meeting concerns, in Go (see §2). This keeps the repo returning the same shape as the other meeting queries.

### 4. Bot — §4.7 in `/edit` (`internal/platform/meetingedit`)

After a datetime override is set, **before `medit:apply` commits**:

- The service calls `MeetingConflicts(workspaceID, participants+organizer, newSpan, excludeMeetingID=thisMeeting)`.
- **Conflicts found** → render the ⚠ warning in the §4.7.2 layout and show **[Да, применить] / [Изменить время]**:
  - `medit:applyforce` (or reuse the existing apply callback past a confirm flag) → proceed with the normal apply path.
  - `medit:field:datetime` → return to the datetime step, keeping all other overrides (existing behavior).
- **No conflicts** → apply silently (current behavior, no extra screen).
- State gets no new persisted field beyond what's needed to remember "warning shown" if required to distinguish the two apply taps; prefer a distinct callback (`medit:applyforce`) to avoid state growth.

Warning text (per §4.7.2):

```
⚠ Внимание! У следующих участников уже есть встречи в это время:
- {PersonName} — «{MeetingName}» ({HH:MM}–{HH:MM})
...

Продолжить создание встречи?  [Да, применить] [Изменить время]
```

Times rendered in the workspace TZ.

### 5. Bot — §4.8 new `/checker` FSM (`internal/platform/checker`)

Mirrors the `scheduleview`/`meetingedit` shape: `State` + `RedisSessions` + `Service` returning `Reply{Text, Keyboard, Edit}`; session key `checker:{telegramID}`, 15-min TTL; wired in `infrastructure/telegram/multitenant.go` (the `*application.Services` backend satisfies a small `checker.Backend` interface).

Steps:

1. **participants** — text search of the directory (`SearchEmployeesGlobal`), tap to add; accumulate emails; "Готово" to proceed. (Min 1 participant.)
2. **range** — text `YYYY-MM-DD..YYYY-MM-DD`, reusing the scheduleview parse convention (`parseRange` → `[from, to)` day-exclusive).
3. **duration** — inline keyboard: 15 / 30 / 45 / 60 / 90 / 120 мин.
4. **results** — call `FreeSlots`; render §4.8.4 list or §4.8.6 "no slots" message.

Results layout (§4.8.4):

```
✅ Общее свободное время для {N} участников:

📅 {Пн, 02.06} — {11:00–12:30} (90 мин свободно)
...
```

No-slots layout (§4.8.6):

```
Общих свободных слотов в выбранном диапазоне не найдено.
Попробуйте: расширить диапазон дат / уменьшить длительность / изменить состав участников.
```

Day labels (`Пн, 02.06`) rendered in workspace TZ. **No "create on this slot" button (out of scope).**

`Backend` interface for the FSM:

```go
type Backend interface {
    SearchEmployeesGlobal(ctx, query string) ([]postgres.Employee, error)
    FreeSlotsForTelegram(ctx, telegramID int64, emails []string, from, to time.Time, durMins int) ([]application.FreeSlot, error)
}
```

`FreeSlotsForTelegram` resolves the caller's workspace (same way `/edit` and `/schedule` resolve the acting user → workspace) then delegates to `Services.FreeSlots`.

### 6. REST (`delivery/http`)

Two new routes under the existing `ws := ap.Group("/workspaces/:id", RequireWorkspaceAccess)` group in `app.go` (after the meetings routes):

```
POST /workspaces/:id/meetings/conflicts   → api.MeetingConflicts
POST /workspaces/:id/meetings/free-slots  → api.FreeSlots
```

- **`/meetings/conflicts`** request `{ date, start, end, participants: []string, exclude_meeting_id?: string }` → `{ conflicts: [ { email, person_name, meeting_name, start, end } ] }`. Advisory; the Mini App calls it before submitting create/edit. Create/update endpoints are unchanged.
- **`/meetings/free-slots`** request `{ from, to, participants: []string, duration_mins: int }` → `{ slots: [ { day, start, end, mins } ] }`.
- Handlers live in `delivery/http/handlers` alongside the existing meetings handlers; they map request/response and call the single application entry point (CQRS query path — read-only, no side effects).

## Data flow

**§4.7 (edit):** `/edit` datetime set → `meetingedit.Service` → `Services.MeetingConflicts` → `Store.ListMeetingsOverlapping` → in-Go per-person attribution + name resolve → warning or silent apply.

**§4.8 (checker):** `/checker` collects {emails, range, dur} → `Services.FreeSlots` → `Store.ListMeetingsOverlapping` (per range) → per-weekday `meeting.FreeSlots` → list.

**REST:** Mini App → `POST .../conflicts` or `.../free-slots` → same application methods.

## Error handling

- Invalid date range / duration → user-facing validation error (bot: Russian message reusing scheduleview parse errors; REST: 400 with message).
- Empty participant set → reject (min 1).
- Workspace not resolvable for a bot user → existing "not registered" path.
- DB errors → logged once at the FSM/handler boundary (zap, structured fields `workspace_id`), generic failure message to the user.
- No conflicts / no free slots are **normal results**, not errors (§4.7.3 silent, §4.8.6 informative message).

## Testing

Per repo convention (pure logic unit-tested, I/O build-verified):

- **Unit (`domain/meeting`):** `Overlaps` (touching edges, partial, full, disjoint); `FreeSlots` (empty busy = full window, busy spanning whole window = none, merge adjacent, min-duration filter, busy outside window clipped). Table-driven.
- **Unit (`application`):** day-iteration + weekend skip + TZ window construction can be tested with a fake repo returning fixed busy spans (no DB). If wiring a fake is heavy, at minimum unit-test the pure per-day assembly helper.
- **Build-verified:** repo query, REST handlers, bot FSM wiring (no DB harness in the postgres package).
- `make test && make lint && make build` from repo root; gofmt-check via `make lint`.

## Out of scope

- Google freebusy / external-calendar visibility (internal DB only).
- Configurable working hours; weekend inclusion.
- Bot "create meeting from slot" / any bot create FSM.
- Mini App frontend wiring (stays on mocks); only the REST endpoints it will call.
- §4.7 conflict check inside the REST *create* path as a returned warning — create stays unchanged; the Mini App calls `/conflicts` separately first.
