# WS2b — Application CQRS Handler Tests (design)

**Date:** 2026-06-17
**Status:** approved — ready for implementation plan
**Part of:** Public-launch hardening, workstream 2 (backend test coverage), sub-phase **b of d**.
Builds on WS2a (domain + persistence + `pgtest` harness, all green in CI). WS2b tests the application layer with in-memory fakes — no DB, no testcontainers.

## Goal

Cover the meeting CQRS handlers — the write-path orchestration (`command.Meetings`) and the read models (`query.Meetings`) — with fast, DB-free unit tests using in-memory fakes for the `Store`, `CalendarProvider`, and `JobQueue` ports. Verify the orchestration (validation, recurrence expansion, calendar calls, persistence, job enqueue, permission checks), not the SQL (covered in WS2a).

## Scope

### In scope
- `internal/application/command/meetings.go`: `Meetings.CreateMeeting`, `UpdateMeeting`, `CancelMeeting`, and pure helpers `ApplyMeetingUpdate`, `ownerOrOrganizer`.
- `internal/application/query/`: `Meetings.Schedule` (`meeting_list.go`) and `MeetingDTO` (`meeting_read.go`).

### Out of scope (deferred — not "command/query")
- Auth/org/google application services: `miniapp_org.go`, `admin_workspace.go`, `google_verify.go`, web-auth/session/magic-link services. These pair better with their adapters in WS2d (or a dedicated slice); folding them here would bloat the plan.
- Postgres SQL (WS2a, done). External adapters (WS2d). Telegram platform packages (WS2c).

## Background — verified current state

- `command.Meetings{Store, Calendar, Queue, Log}` depends on three interfaces in `command/ports.go`:
  - `Store` — 7 methods: `GetOrganization`, `GetMeeting`, `CreateMeeting`, `CreateMeetingSeries`, `UpdateMeeting`, `CancelMeeting`, `AddParticipants`.
  - `CalendarProvider` — `For(ctx, orgID) (docalendar.Service, error)`.
  - `JobQueue` — `EnqueueMeetingCreated/Updated/Cancelled`.
- `docalendar.Service` is the calendar port (`CreateEvent`/`UpdateEvent`/`DeleteEvent`) returned by `Calendar.For`.
- `CreateMeeting` flow: `GetOrganization` → resolve tz/loc → parse `Date+Start`/`Date+End` (`2006-01-02 15:04`) → build `meeting.Input` → `dom.Validate()` → parse `recurrence_until` → `meeting.Occurrences` → `Calendar.For` → for `once`: `CreateEvent` then `Store.CreateMeeting`; for recurring: series events then `Store.CreateMeetingSeries` → enqueue. Input errors wrap `ErrInvalidInput`; calendar errors wrap `calendar:`.
- `UpdateMeeting`/`CancelMeeting` gate on `ownerOrOrganizer(org, organizerUserID, userID)`.
- Existing fake pattern: `scheduler_agent`, `reminder_scheduler`, `meetingrecipients` tests use hand-written in-memory fakes in-package — WS2b follows it.

## Design

### 1. In-memory fakes (in the `command` test package)

Hand-written, map-backed, call-recording fakes:

- **`fakeStore`** implements `command.Store`. Backed by `map[uuid.UUID]model.Meeting` + a stored `model.Organization`. Records calls (e.g. `created []model.Meeting`, `seriesCreated int`, `updated`, `cancelled`). `GetOrganization` returns a configurable org (with `TZ`); `CreateMeeting` assigns an ID and stores; `CreateMeetingSeries` stores all; `GetMeeting` returns by id or a not-found error.
- **`fakeCalProvider`** implements `command.CalendarProvider`; `For` returns a `*fakeCalService` (or a configured error).
- **`fakeCalService`** implements `docalendar.Service`. `CreateEvent` returns a canned `{EventID, MeetLink}` and appends to `createdEvents`; `UpdateEvent`/`DeleteEvent` record calls. A `failCreate bool` (and similar) toggle forces errors to test the failure paths.
- **`fakeQueue`** implements `command.JobQueue`; records `createdEnq`, `updatedEnq`, `cancelledEnq` (meeting IDs).

Fakes live in a `fakes_test.go` in the `command` package (white-box, so tests can read recorded fields). No shared fakes package (YAGNI — query has different ports; it gets its own small fake).

### 2. Command tests (`command/meetings_test.go`)

- **CreateMeeting_Once_HappyPath:** valid single meeting → `fakeCalService.createdEvents` has 1; `fakeStore` has 1 meeting with `GoogleEventID`/`MeetLink` from the fake; `fakeQueue.createdEnq` has 1; returned meeting times are UTC.
- **CreateMeeting_Recurring_Series:** `recurrence=daily` with `recurrence_until` spanning 3 days → 3 calendar events + `CreateMeetingSeries` with 3 meetings; enqueue created.
- **CreateMeeting_InvalidInput:** bad start time, unknown recurrence, and bad `recurrence_until` each return an error wrapping `ErrInvalidInput`; nothing persisted, nothing enqueued.
- **CreateMeeting_CalendarFailure:** `fakeCalService.failCreate=true` → error wraps `calendar:`; `fakeStore` empty; `fakeQueue` empty.
- **UpdateMeeting_PermissionDenied:** a `userID` that is neither org owner nor the meeting organizer → permission error; no calendar/store/queue calls.
- **UpdateMeeting_HappyPath:** organizer updates a field → `ApplyMeetingUpdate` reflected, `fakeCalService` update recorded, `Store.UpdateMeeting` called, `updatedEnq` has 1.
- **CancelMeeting_HappyPath:** organizer cancels → calendar delete recorded (best-effort), `Store.CancelMeeting` called, `cancelledEnq` has 1.
- **CancelMeeting_PermissionDenied:** non-owner/non-organizer → error; no store/queue mutation.

### 3. Pure-helper tests (`command/meetings_helpers_test.go`)

- **ApplyMeetingUpdate:** partial updates (only `Dept` set, only times set), name recompute when relevant fields change, validation failure (end ≤ start) returns error. Driven directly (no fakes).
- **ownerOrOrganizer:** truth table — org owner (regardless of organizer), the organizer themselves, and an unrelated user.

### 4. Query tests (`query/meetings_test.go`)

A small fake implementing the query ports (`meetingListApp` / `meetingStore`):
- **Schedule:** returns the store's meetings for an email within `[from,to]` (delegation + shape).
- **MeetingDTO:** field/timezone formatting (date/time rendered in the provided `*time.Location`).

## Testing / verification

- `go test ./internal/application/...` green (fast, no Docker).
- `golangci-lint run` clean (WS1 gate) on the new test files.
- Failure-path tests assert **negative** outcomes too (nothing persisted/enqueued on error) — not just the happy path.

## Risks & mitigations

- **Fakes drift from real behavior.** Mitigation: fakes implement the exact port interfaces (compile-time enforced); the persistence behavior they stand in for is independently covered by WS2a's real-Postgres tests.
- **Over-asserting on internals.** Mitigation: assert on observable effects (calls recorded, errors returned, values persisted), not private intermediate state.
- **Calendar/Store signature mismatch in fakes.** Mitigation: add `var _ command.Store = (*fakeStore)(nil)` (and for each port) so a signature drift fails to compile.

## Done criteria

- `fakes_test.go` with the four fakes + compile-time interface assertions.
- Command, helper, and query tests covering the behaviors in §2–§4, including failure paths.
- `go test ./internal/application/...` and `golangci-lint run ./...` pass; no production code changed.
