# Slice 1c — Unified Cross-Calendar Availability (design)

**Date:** 2026-06-18
**Status:** approved — ready for implementation plan
**Part of:** SaaS Product Completion epic, **Track 1 (cross-calendar wedge)**, slice **1c**. Follows 1a + 1b.

## Epic context

1a shipped per-user Google calendar OAuth (vault, organizer-aware `For`, SA fallback). 1b added
Microsoft (Teams events, raw-Graph adapter, composite resolver, and an MS `BusyReader` for
free/busy — built but unconsumed). **1c makes the wedge real:** the conflict checker and
free-slot finder now reflect each participant's **actual external calendar** busy times
(Google FreeBusy + MS getSchedule), not just meetings booked through this app.

**Sequence:** 1a [done] → 1b [done] → 1c [this] → 2a (onboarding) → 3 (booking links).

## Goal

Augment `MeetingConflicts` and `FreeSlots` so a participant's real Google/Microsoft calendar
busy intervals are merged into availability — best-effort, never hard-failing — using a hybrid
**union** model: the organizer's connection covers colleagues it can see, plus each connected
participant's own connection covers themselves (incl. cross-org). Backend-only; the existing
conflict-warning and free-slot UIs get richer data automatically.

## Decisions (from brainstorming)

- **Hybrid = union** (not gap-detection): external busy = (organizer's reader over all emails)
  ∪ (each connected participant's own reader over their own email), merged/deduped. Avoids the
  "unresolvable vs free" ambiguity (FreeBusy/getSchedule return empty for invisible emails).
- **External-busy conflicts carry no meeting name** — free/busy is opaque (privacy); the
  `Conflict.MeetingName` is left empty. An optional FE "Busy (external)" label is deferred.
- **Backend-only** — no UI task; the existing checker / free-slot screens consume the richer
  data unchanged.
- **Google reader uses the per-user connection token** for FreeBusy (organizer's token covers
  visible colleagues; each participant's own token covers themselves cross-org).
- **Best-effort throughout** — any reader error is logged and skipped; the conflict/free-slot
  query never fails because of external calendar access.

## Background — verified current state

- `internal/application/conflict.go`:
  - `MeetingConflicts(ctx, emails []string, start, end time.Time, excludeMeetingID uuid.UUID) ([]Conflict, error)`
    → `s.Store.ListMeetingsOverlapping(ctx, emails, start, end)` (app DB meetings only) →
    `[]Conflict`.
  - `FreeSlots(ctx, emails []string, from, to time.Time, durMins int) ([]FreeSlot, error)` →
    same DB query → `busy []meeting.Span` → `meeting.FreeSlots(dayBusy, winStart, winEnd, minDur)`.
  - `Conflict{ Email, PersonName, MeetingName string; Start, End time.Time }`.
  - `personName(ctx, email) string` resolves a display name from the directory.
  - `meetingEmails(ctx, organizationID, meetingID)` builds the participant+organizer email set;
    the organizer email is resolvable via `GetUserByID(*m.OrganizerUserID).Email` (the pattern
    already used here and in 1a).
- `internal/application/series_conflicts.go` calls `MeetingConflicts` per occurrence — it
  inherits external busy automatically once `MeetingConflicts` is augmented.
- `docalendar.Interval{Start, End time.Time}` + `docalendar.BusyReader{ BusyTimes(ctx, emails
  []string, from, to time.Time) (map[string][]Interval, error) }` exist (1b).
- MS adapter implements `BusyReader` (1b, getSchedule). The composite resolver
  (`internal/infrastructure/calendar/resolver`) currently resolves an **event** `Service` by
  organizer for create/update/cancel.
- `internal/platform/fanio.AllBestEffort(ctx, limit, n, fn func(ctx, int))` — bounded parallel,
  best-effort (no error propagation). Reuse for per-participant fetches.
- `model.CalendarConnection` has `Provider` + `UpdatedAt`; `Store.ListCalendarConnections(email)`
  returns a person's connections.

## Design

### A. Google `BusyReader`

Add to `internal/infrastructure/calendar/google` a `BusyTimes(ctx, emails []string, from, to
time.Time) (map[string][]docalendar.Interval, error)` implemented via the Calendar **FreeBusy**
API (`Freebusy.Query` with `Items: [{Id: email}]`, `TimeMin/TimeMax`), over the per-user OAuth
`*calendar.Service`. Parse each calendar's `Busy` ranges (RFC3339) into `docalendar.Interval`.
A small reader type (or a method on a per-user Google reader built from a `model.CalendarConnection`,
mirroring the MS factory's adapter construction with the self-persisting token source).
`var _ docalendar.BusyReader = (*googleReader)(nil)`.

### B. `BusyResolver` port + impl (by email)

Application port:
```go
type BusyResolver interface {
    ReaderFor(ctx context.Context, email string) (docalendar.BusyReader, bool)
}
```
(Aliased into `application`; `docalendar.BusyReader` is `application.BusyReader` via the existing
alias style.) Infrastructure impl (in `infrastructure/calendar/resolver` or a sibling): looks up
`email`'s most-recently-updated connection → microsoft → MS adapter (1b); google → Google reader
(§A); none → `(nil, false)`. Wired in `main.go` from the connection store + the Google/MS
connectors. `Services` gains a `Busy BusyResolver` field (nil-safe: when nil, the engine simply
skips external busy — behavior identical to today).

### C. `gatherExternalBusy` orchestrator

In `internal/application` (e.g. `availability.go`):
```go
func (s *Services) gatherExternalBusy(ctx context.Context, organizerEmail string, emails []string, from, to time.Time) map[string][]meeting.Span
```
1. If `s.Busy == nil` → return empty (no external data; today's behavior).
2. If `organizerEmail != ""` and `s.Busy.ReaderFor(organizerEmail)` ok →
   `reader.BusyTimes(ctx, emails, from, to)` (best-effort; on error log `external_busy_fetch_failed`
   + email-hash, skip).
3. For each `email` in `emails` with its own reader → `reader.BusyTimes(ctx, []string{email}, from, to)`,
   run via `fanio.AllBestEffort` (bounded parallel), results merged under a mutex.
4. Merge + dedupe intervals per email into `[]meeting.Span`.
- Never returns an error. No tokens/raw emails logged.

### D. Engine integration

- `MeetingConflicts`: after building DB conflicts, call `gatherExternalBusy(ctx, organizerEmail,
  emails, start, end)`; for each external span overlapping `[start,end)` (excluding the meeting
  being edited is N/A — external isn't ours), append `Conflict{Email: email, PersonName:
  personName(ctx, email), MeetingName: "", Start: span.Start, End: span.End}`. Keep the existing
  sort. The `organizerEmail` is threaded in from the caller (see below).
- `FreeSlots`: after building `busy` from DB meetings, append the external spans for all emails,
  then compute free slots as today.
- **Threading the organizer email:** `MeetingConflicts`/`FreeSlots` currently take `emails`.
  Add an `organizerEmail string` parameter (or derive it as the first/explicit organizer in the
  set). Update callers: `MeetingUpdateConflicts`/`meetingEmails` already resolve the organizer;
  the free-slot/checker handlers pass the caller's email as organizer. (Choose the minimal
  signature change; document exact call sites in the plan.)
- `series_conflicts.go` needs no change beyond passing the organizer through `MeetingConflicts`.

### E. Wiring (`main.go`)

Build the `BusyResolver` from the connection store + Google reader factory + MS factory (reuse
1b's MS factory; add a Google reader factory in the google package), and set
`services.Busy = busyResolver`. When neither Google nor MS is configured, leave `services.Busy`
nil (engine skips external busy).

## Testing / verification

- **Google `BusyReader`** (httptest pointed at a fake FreeBusy endpoint via the existing
  `option.WithEndpoint`/`WithHTTPClient` test harness from WS2d): request items + `timeMin/Max`,
  busy-range parsing.
- **`BusyResolver.ReaderFor`** (fake connection store): microsoft→MS reader, google→Google
  reader, none→`false`, most-recent tiebreak.
- **`gatherExternalBusy`** (fake `BusyResolver`): organizer∪own union + dedupe; one reader errors
  → others still merged (best-effort); `s.Busy == nil` → empty.
- **`MeetingConflicts`/`FreeSlots`** with a fake `BusyResolver`: external busy produces an
  external conflict (empty `MeetingName`) / shrinks free slots; with no resolver/connections the
  output is byte-identical to today (regression guard).
- `env -u GOROOT go build/vet`, `go test -race ./...`, `golangci-lint run ./...` clean; existing
  conflict/free-slot/series tests still pass.

## Risks & mitigations

- **Latency / API fan-out.** N participants ⇒ up to N+1 calendar calls per conflict/free-slot
  check; series checks multiply by occurrences. *Mitigation:* `fanio.AllBestEffort` bounded
  parallelism; organizer's single multi-email call covers the common same-team case; best-effort
  so a slow/failed reader never blocks. Log slow gathers. (A response cache is out of scope.)
- **Privacy.** Free/busy must never leak event titles. *Mitigation:* external conflicts carry no
  `MeetingName`; only busy spans are used.
- **"Unresolvable vs free" ambiguity.** *Mitigation:* union model (don't infer gaps); a
  participant with no connection simply contributes only their app-DB meetings (unchanged).
- **Signature change to `MeetingConflicts`/`FreeSlots`.** *Mitigation:* mechanical; all call
  sites enumerated in the plan; gated by build/vet/test.
- **Best-effort hiding real failures.** *Mitigation:* structured `external_busy_fetch_failed`
  logs (email-hashed) so failures are observable without breaking the request.

## Done criteria

- Google `BusyReader` (FreeBusy) implemented + httptest-tested; MS `BusyReader` (1b) now consumed.
- `BusyResolver` port + impl resolve a per-email reader; wired in `main.go`; `Services.Busy`
  nil-safe.
- `MeetingConflicts` + `FreeSlots` (and thus `series_conflicts`) merge external busy via the
  hybrid union; external conflicts carry no meeting name; DB-only path unchanged when no
  connections exist.
- Best-effort + parallel; no tokens/raw-email logged.
- `go test -race ./...` + `golangci-lint` green; existing availability tests still pass.
- Onboarding, booking links, FE "external" label, and connection-lookup batching explicitly
  deferred.
