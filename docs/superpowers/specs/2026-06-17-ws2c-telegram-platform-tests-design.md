# WS2c — Telegram Platform Package Tests (design)

**Date:** 2026-06-17
**Status:** approved — ready for implementation plan
**Part of:** Public-launch hardening, workstream 2 (backend test coverage), sub-phase **c of d**.
Builds on WS2a (harness/domain/persistence) and WS2b (application CQRS), both green in CI. WS2c is DB-free — in-memory fakes only.

## Goal

Cover the stateful Telegram platform packages — the conversational command handlers (`checker`, `meetingedit`, `scheduleview`, `botreg`), the reminder-settings helpers (`botsettings`), and the pure notification-message builders (`meeting_notifier/message.go`) — with fast unit tests using in-memory fakes for their small port interfaces. Verify the state-machine transitions, parsing, permission/admin logic, and message formatting.

## Scope

### In scope
- `internal/platform/checker` — `Service{backend Backend, sess sessions}` + `parse.go`.
- `internal/platform/meetingedit` — `Service{backend Backend, sess sessions}` + `parse.go`.
- `internal/platform/scheduleview` — `Service{backend Backend, sess sessions}` + `parse.go`.
- `internal/platform/botreg` — `Service{users userStore, sess sessions, adminIDs []int64}`.
- `internal/platform/botsettings` — pure `Parse`/`Format` (+ internal `parse`/`format`) and `Service{store}`.
- `internal/platform/meeting_notifier/message.go` — pure builders (`buildMessage`, `buildEventMessage`, `buildUpdatedMessage`, `buildRemovedMessage`, `buildCancelledMessage`, `tzLabel`).

### Out of scope (deferred)
- `meeting_notifier` **send-orchestration** (`HandleCreated`/`HandleUpdated`/`HandleParticipantAdded`/`HandleParticipantRemoved`/`HandleCancelled`). These depend on concrete `*postgres.Store` + `*bot.Bot`; testing them requires either a production refactor to interfaces or a real-DB + bot harness. The Telegram-send path is WS2d's subject (Telegram dispatch/adapter), and the data pieces are already covered (`GetMeeting`/`GetOrganization`/`TryClaimReminder` in WS2a; `meetingrecipients.Resolve` has its own test). No production refactor in this test-only slice.
- External adapters (WS2d). Application CQRS (WS2b, done). Persistence (WS2a, done).

## Background — verified current state

- `checker`, `meetingedit`, `scheduleview` each follow an identical shape: `New(backend Backend, sess sessions) *Service`, with `Backend`/`sessions` as **small unexported-or-local interfaces**, plus `State`/`Button`/`Reply` value types and a pure `parse.go`. The Service is a conversational state machine: an input (text or callback) advances state (loaded from `sessions`) and returns a `Reply` (text + buttons).
- `botreg`: `New(users userStore, sess sessions, adminIDs []int64) *Service` — the /start registration flow (collect name → email → create user; admin IDs flagged).
- `botsettings`: pure `Parse(csv) []int` / `Format(mins) string` and `New(store) *Service` over a one-method `store` interface.
- `meeting_notifier/message.go`: pure text builders + `tzLabel`; `notifier.go` holds the concrete-dependency orchestration (out of scope).
- Existing precedent: `scheduler_agent` tests already fake a bot backend + sessions in-package — WS2c follows the same idiom.
- Tests run under the WS1 gate, which now includes `go test -race`.

## Design

### Fakes (per package, in-package `_test.go`)

Each Service package gets small hand-written fakes implementing its own port interfaces, with compile-time assertions (`var _ Backend = (*fakeBackend)(nil)`, etc.):
- **`fakeSessions`** — map-backed `Get`/`Set`/`Del` (or whatever the package's `sessions` interface declares), so state persists across simulated turns within a test.
- **`fakeBackend`** (checker/meetingedit/scheduleview) — returns canned data for whatever the package's `Backend` interface exposes (e.g. free-slot lookups, meeting lists/edits, schedules); records calls where a write is involved.
- **`fakeUserStore`** (botreg) — records created users; returns configurable existing-user/lookups.
- **`fakeStore`** (botsettings) — captures the persisted reminder-minutes CSV.

Fakes are per-package (not shared) — each package's interfaces differ, and per-package fakes keep tests readable (YAGNI on a shared fakes module).

### Tests

- **checker:** the find-common-slot flow — start → collect participants/window → produce slot buttons; `parse.go` (parsing user input into the checker's domain) edge cases (valid, empty, malformed). Assert `Reply` text/buttons and recorded backend calls.
- **meetingedit:** the edit flow — select meeting → choose field → submit new value → confirm; `parse.go` field/value parsing; the "this vs whole series" scope branch if present. Assert state transitions and the edit call the backend receives.
- **scheduleview:** the schedule-viewing flow — select target/colleague → render schedule; `parse.go` parsing. Assert rendered `Reply`.
- **botreg:** /start → ask name → ask email → create user; admin-ID detection sets the right role/flag; re-entry when already registered. Assert `fakeUserStore` got the right create and `Reply` copy advances.
- **botsettings:** `Parse`/`Format` round-trips (CSV → []int → CSV), dedup, sort, drop invalid/out-of-range, empty; `Service` persists via `fakeStore`.
- **meeting_notifier message builders:** each builder renders the expected text for a meeting (name, Meet link present/absent, time range in the given `*time.Location`); `tzLabel` for whole-hour and half-hour offsets (`UTC+5`, `UTC+5:30`, negative).

Tests assert observable effects (returned `Reply`, recorded fake calls, formatted strings), not private intermediate state.

## Testing / verification

- `go test -race ./internal/platform/...` green (fast, no Docker).
- `golangci-lint run ./...` clean on the new files.
- Where a flow spans multiple turns, the test threads state through `fakeSessions` exactly as the bot dispatcher would (set after one turn, read at the next).

## Risks & mitigations

- **Interface shapes differ per package** — the exact `Backend`/`sessions`/`userStore`/`store` method sets must be read from each package before writing fakes. Mitigation: the plan enumerates each interface's methods from source; compile-time `var _ = ...` assertions catch any drift.
- **State-machine entry points vary** — each Service may expose different method names (e.g. `OnText`/`OnCallback`/`Handle`). Mitigation: the plan reads each Service's exported methods and drives them exactly; if a flow is more tangled than a clean turn-by-turn machine, the test covers the reachable transitions and notes any gap rather than forcing artificial coverage.
- **Over-coupling to copy/wording** — assert on structural outcomes (which button, which backend call, state advanced) and only key phrases, not full localized strings, to avoid brittle tests.

## Done criteria

- Per-package fakes with compile-time interface assertions.
- Tests covering the flows/parsers in §Tests for all six in-scope packages.
- `go test -race ./internal/platform/...` and `golangci-lint run ./...` pass; no production code changed.
