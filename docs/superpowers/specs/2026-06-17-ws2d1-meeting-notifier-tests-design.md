# WS2d-1 — meeting_notifier Orchestration Tests (interface extraction) (design)

**Date:** 2026-06-17
**Status:** approved — ready for implementation plan
**Part of:** Public-launch hardening, workstream 2 (backend test coverage), sub-phase **2d-1** (first of WS2d's four adapter sub-phases). Closes the `meeting_notifier` send-orchestration deferred from WS2c.

## Goal

Make `meeting_notifier`'s send-orchestration (`HandleCreated`, `HandleUpdated`, `HandleCancelled`, `HandleParticipantAdded`, `HandleParticipantRemoved`) unit-testable by replacing its concrete `*postgres.Store` and `*bot.Bot` dependencies with narrow interfaces, then cover the orchestration with in-memory fakes. Behavior-preserving refactor; `main.go` unchanged.

## Background — verified current state

`Notifier{store *postgres.Store, bot *bot.Bot, log *zap.Logger}`; `New(store, b, log)`. The five `Handle*` methods all: load meeting + organization, resolve a `*time.Location` (org TZ, fallback `Asia/Almaty`, then `time.UTC` on error), build a message via the (already-tested, WS2c) pure builders, resolve recipients via `meetingrecipients.Resolve(ctx, n.store, m)`, and `n.bot.SendMessage(...)` per recipient. Specifics:
- `HandleCreated`: gates each send on `n.store.TryClaimReminder(ctx, m.ID, r.TelegramID, postgres.ReminderOffsetCreated)` (dedup) — a claim error aborts the whole call; an unclaimed recipient is skipped.
- `HandleParticipantAdded/Removed` (`notifyParticipant`): looks up the single participant by `GetBotUserByEmail(email)`; `postgres.IsNotFound(err)` → returns nil (no send); added → "➕ Вас добавили…", removed → removed message; sends to that user's `TelegramID`.
- `HandleCancelled`/`HandleUpdated`: send to **all** resolved recipients, **no** claim gating.
- Store methods used directly: `GetMeeting`, `GetOrganization`, `GetBotUserByEmail`, `TryClaimReminder`. Plus `meetingrecipients.Resolve(ctx, store, m)` needs the store to satisfy `meetingrecipients.Store`.
- `n.bot.SendMessage(ctx, *bot.SendMessageParams) (*models.Message, error)` is the only bot call; send errors are logged and swallowed (best-effort), not returned.

## Decisions (from brainstorming)

- **Full adapter unit tests via refactor** (user's choice for WS2d). 2d-1 is the cleanest such refactor — interface extraction over orchestration, no external SDK, no httptest.
- Keep the refactor behavior-preserving; `main.go` must compile unchanged (real `*postgres.Store`/`*bot.Bot` satisfy the new interfaces).

## Design

### 1. Interfaces (new, in package `meeting_notifier`)

```go
type sender interface {
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
}
```
(`models` = `github.com/go-telegram/bot/models`; `*bot.Bot` satisfies this.)

```go
type store interface {
	// direct calls
	GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (postgres.Meeting, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (postgres.Organization, error)
	GetBotUserByEmail(ctx context.Context, email string) (postgres.BotUser, error)
	TryClaimReminder(ctx context.Context, meetingID uuid.UUID, telegramID int64, offset int) (bool, error)
	// plus every method meetingrecipients.Resolve requires (enumerated from
	// meetingrecipients' Store interface in the plan), so this interface is
	// assignable to meetingrecipients.Store.
}
```
The plan will read `internal/platform/meetingrecipients` to enumerate the exact `Resolve` store methods and fold them in. `*postgres.Store` satisfies the union.

### 2. Notifier wiring

- `Notifier{store store; bot sender; log *zap.Logger}`; `New(store store, b sender, log *zap.Logger) *Notifier`.
- Method bodies unchanged except the receiver field types. `meetingrecipients.Resolve(ctx, n.store, m)` compiles because `store` is a superset of `meetingrecipients.Store` (Go interface-to-interface assignment).
- Compile-time guarantees the real types still satisfy: `var _ sender = (*bot.Bot)(nil)`; `var _ store = (*postgres.Store)(nil)` (in a non-test file so it's part of the build).
- `main.go` line `meeting_notifier.New(store, tg, logger)` is unchanged.

### 3. Tests (`notifier_test.go`, in-package)

`fakeStore` (configurable meeting/org/bot-users/participants + a `claim func` or map for `TryClaimReminder` + recorded calls) and `fakeSender` (records `[]sentMsg{ChatID, Text}`; optional error). Compile-time `var _ store = (*fakeStore)(nil)`, `var _ sender = (*fakeSender)(nil)`.

- **HandleCreated_SendsToClaimedOnly:** two recipients; `TryClaimReminder` returns true for one, false for the other → exactly one `SendMessage`, to the claimed recipient, text contains "📅 Новая встреча" and the meeting name.
- **HandleCreated_ClaimError_Aborts:** `TryClaimReminder` returns an error → `HandleCreated` returns that error; assert it surfaced.
- **HandleParticipantAdded_RoutesToUser:** `GetBotUserByEmail` returns a user → one send to that user's TelegramID with the "➕ …" text.
- **HandleParticipantRemoved_RoutesToUser:** removed text to the user.
- **HandleParticipant_NotFound_NoSend:** `GetBotUserByEmail` → `IsNotFound` → returns nil, zero sends.
- **HandleCancelled_AllRecipients / HandleUpdated_AllRecipients:** send to every resolved recipient (no claim gating); right header text.
- **TZ_Fallback:** org `TZ=""` (or invalid) → no error, message renders (location falls back) — assert a send happened with sane text.

To make `meetingrecipients.Resolve` yield known recipients, the `fakeStore` returns canned participants/bot-users exactly as `Resolve` reads them (the plan enumerates those reads). Tests assert observable effects (which `ChatID`s received which text, claim gating, error propagation), not internal state.

## Testing / verification

- `go test -race ./internal/platform/meeting_notifier/...` green (DB-free).
- `go build ./...` + `go vet ./...` confirm `main.go` and all callers still compile (the interface assertions catch real-type drift).
- `golangci-lint run ./...` clean.

## Risks & mitigations

- **Wide `store` interface mirroring `meetingrecipients.Store`.** Mitigation: enumerate the exact methods from source in the plan; the `var _ store = (*postgres.Store)(nil)` assertion guarantees completeness/correctness at build time.
- **`Resolve` has its own logic the fake must feed.** Mitigation: the plan reads `meetingrecipients.Resolve` and documents precisely which store methods/return values shape the recipient list, so the fake produces a deterministic recipient set.
- **Behavior drift during refactor.** Mitigation: only field/parameter types change; bodies are byte-identical otherwise; gated by build/vet/test/lint + CI.

## Done criteria

- `sender`/`store` interfaces added; `Notifier`/`New` use them; `main.go` unchanged and the whole module builds.
- Compile-time assertions that `*bot.Bot` and `*postgres.Store` satisfy the interfaces.
- `notifier_test.go` covers all five `Handle*` paths incl. claim-dedup, participant routing, not-found no-op, all-recipient broadcasts, and TZ fallback.
- `go test -race ./...` + `golangci-lint run ./...` pass; no production behavior changed.
