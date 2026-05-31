# §5a — Meeting-created notification (design)

**Date:** 2026-05-31
**Scope:** ТЗ §5a (`docs/NEW-FEATURES.md`). Closes the §5 notifications block (reminder engine §5b already done).

## Goal

When a meeting is created, send a Telegram DM to its recipients (registered
participants + the organizer) containing the meeting name, time, and Google Meet
link — immediately, durably, and without double-sending.

## Recipients

Same set as the reminder engine: registered participants resolved via
`email → bot_users → telegram_id`, plus the organizer (if their Telegram is
linked). The organizer is not duplicated when they are also a participant.

## Flow

```
POST /meetings → CreateMeeting (Google event + DB row + participants)
              → Queue.EnqueueMeetingCreated(workspace_id, meeting_id)  [best-effort]
                                    ↓ asynq (meeting:created)
              meeting_notifier.HandleCreated:
                load meeting (workspace-scoped)
                resolve recipients (participants + organizer)
                for each tg: TryClaim(meeting_id, tg, sentinel) → SendMessage(DM)
```

Enqueue happens **inside `CreateMeeting`** after the meeting is persisted (a
command may enqueue work, mirroring `RunScenario`). If the enqueue fails the
meeting is already created in Google + DB, so we **`log.Warn` and still return
the meeting successfully** — the create-notification is best-effort and must not
fail the create. This requires adding `Log *zap.Logger` to
`application.Services` (the logger is already available where `Services` is
constructed in `NewApp`).

## Queue (`internal/infrastructure/queue/asynq/queue.go`)

```go
const TaskMeetingCreated = "meeting:created"

type MeetingCreatedPayload struct {
    WorkspaceID string `json:"workspace_id"`
    MeetingID   string `json:"meeting_id"`
}

func (c *Client) EnqueueMeetingCreated(ctx context.Context, workspaceID, meetingID uuid.UUID) error
```

Enqueued with `asynq.MaxRetry(5)` (same as scenario runs) — hence the dedup
below.

`NewServer` currently takes a single `handler` bound to `scenario:run`. Change
its signature to accept `handlers map[string]asynq.HandlerFunc` and register all
types in a loop. One call site (`main.go`) is updated.

## Worker — package `internal/platform/meeting_notifier`

Mirrors `reminder_scheduler`.

```go
type Notifier struct {
    store *postgres.Store
    bot   *bot.Bot
    log   *zap.Logger
}

func New(store *postgres.Store, b *bot.Bot, log *zap.Logger) *Notifier
func (n *Notifier) HandleCreated(ctx context.Context, workspaceID, meetingID uuid.UUID) error
```

`HandleCreated`:

1. Load the meeting via `GetMeeting(ctx, workspaceID, meetingID)` (workspace
   scoped). Load the workspace for its timezone.
2. Resolve recipients via the shared resolver (below).
3. For each recipient telegram ID: `TryClaimReminder(meeting_id, tg, offsetCreated)`;
   if claimed, `SendMessage` the creation DM.
4. **Send failures**: `log.Warn("send meeting created", telegram_id, meeting_id, err)`
   and continue — do **not** return an error for a single failed send (that
   would make asynq retry and re-DM everyone else).
5. Return an `error` only when the meeting/participants/workspace cannot be read
   — in that case asynq retrying is correct.

### Message (pure function, unit-tested)

```
📅 Новая встреча
«<name>»
🗓 31.05.2026, 14:00–15:00 (Алматы)
🔗 <meet link>
```

- Date/time rendered in the workspace timezone (`Asia/Almaty` default), matching
  `CreateMeeting`'s parsing.
- The `🔗 <link>` line is omitted when the meet link is empty.

## Dedup (no migration)

asynq retries can re-run `HandleCreated`. Reuse the existing `meeting_reminders`
table (PK `meeting_id, telegram_id, offset_minutes`) with a **sentinel offset
`-1`** for the creation notice:

```go
const offsetCreated = -1
claimed, _ := store.TryClaimReminder(ctx, meetingID, tg, offsetCreated)
```

`-1` never collides with real reminder offsets (10/15/30/60/120/1440). One
ledger row per (meeting, user, "created"). No new table or migration.

## Shared recipient resolver (DRY)

`reminder_scheduler.recipients()` already does "participants by
`email → bot_users` + organizer by telegram". §5a is the second use, so extract a
shared helper into a new package `internal/platform/meetingrecipients`:

```go
type Recipient struct {
    TelegramID      int64
    ReminderMinutes string // from bot_users; empty for the organizer
    IsOrganizer     bool
}

func Resolve(ctx context.Context, store *postgres.Store, m postgres.Meeting) []Recipient
```

- `reminder_scheduler.recipients()` builds offsets on top: participants via
  `botsettings.Parse(r.ReminderMinutes)`, organizer via `defaultOrganizerOffsets`.
- `meeting_notifier` uses just the set of `TelegramID`s.

Organizer not duplicated when they are also a participant (current behavior
preserved). This refactor removes the duplication; both features read recipients
from one source of truth.

## Wiring (`cmd/server/main.go`)

- Set `Services.Log = logger`.
- Construct `meeting_notifier.New(store, tg, logger)`.
- Register both handlers in the asynq mux: `scenario:run` (existing) and
  `meeting:created` (new).

## Testing

- **Unit** — message builder: format, Almaty rendering, empty meet link omits the
  link line.
- **Unit** — `meetingrecipients.Resolve`: participant without a `bot_users`
  record is skipped; organizer-not-participant is added; organizer-also-participant
  is not duplicated.
- **Send path** — build-verified + manual run (same approach as
  `reminder_scheduler`; no DB harness in the postgres package).

## Out of scope

- Edit/update notifications (§4.3–4.4) and cancellation notices — separate
  increments.
- Localization of the message (ru only here, matching reminder engine).

## Relationship to the platform

Additive. Reuses the existing asynq queue, `bot_users`/`meeting_reminders`
tables, the bot wiring, and the recipient-resolution pattern from the reminder
engine. No changes to the notify-bot/scenario engine.
```