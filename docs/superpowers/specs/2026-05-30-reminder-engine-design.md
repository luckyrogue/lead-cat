# Design — Reminder engine (§5b-2)

**Date:** 2026-05-30
**Status:** Approved (brainstorm)
**Part of:** meetings notifications ([NEW-FEATURES.md](../../NEW-FEATURES.md) §5). Second half of §5b; consumes the per-user settings from §5b-1 (`bot_users.reminder_minutes`). Builds on bot registration + meetings.

## Goal

A scheduler that, every minute, DMs upcoming-meeting reminders over Telegram: to registered participants by their configured intervals, and to the organizer by a default interval — each reminder sent at most once (durable dedup).

## Decisions (from brainstorm)

- **Pattern:** a dedicated `reminder_scheduler` package mirroring `scenario_scheduler` (1-minute ticker + Redis leader lock `leadcat:reminders:leader`). Sends via `bot.SendMessage`.
- **Durable dedup:** a `meeting_reminders` table; claim-before-send via `INSERT … ON CONFLICT DO NOTHING` (atomic, survives restarts). Keyed `(meeting_id, telegram_id, offset_minutes)`.
- **Recipients:** registered **participants** (email → `bot_users`, by their `reminder_minutes`) **plus the organizer** (`meetings.organizer_user_id` → `platform_users.telegram_id`) with a **default of 15 minutes**. The organizer's path is skipped if their `telegram_id` is already among the participants (their own settings apply instead); the dedup key also prevents any overlap.
- **Testability:** pure helpers (`dueOffsets`/`offsetLabel`/`message`) unit-tested; the tick (DB + bot) is integration/manual, like `scenario_scheduler`.

## Data (new goose migration)

```sql
CREATE TABLE meeting_reminders (
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    telegram_id BIGINT NOT NULL,
    offset_minutes INT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (meeting_id, telegram_id, offset_minutes)
);
```

Store methods:

- `ListUpcomingMeetings(ctx, until time.Time) ([]Meeting, error)` — `status='scheduled' AND starts_at > now() AND starts_at <= until`, ordered by `starts_at`.
- `TryClaimReminder(ctx, meetingID uuid.UUID, telegramID int64, offset int) (bool, error)` — `INSERT INTO meeting_reminders ... ON CONFLICT DO NOTHING`; returns `true` if a row was inserted (claim → send), `false` on conflict (already sent).
- `GetUserTelegramID(ctx, userID uuid.UUID) (int64, bool, error)` — the organizer's linked Telegram id; `bool=false` when not linked (skip).
- (Existing: `ListParticipants(meetingID)`, `GetBotUserByEmail(email)`.)

## Pure helpers (`reminder_scheduler`, unit-tested)

- `dueOffsets(now, startsAt time.Time, offsets []int) []int` — offsets where `now >= startsAt.Add(-offset*min)` AND `now < startsAt`.
- `offsetLabel(min int) string` — `10→"10 минут", 15→"15 минут", 30→"30 минут", 60→"1 час", 120→"2 часа", 1440→"1 день"` (fallback `"<n> минут"`).
- `message(name, meetLink string, offset int) string` — `⏰ Напоминание: встреча через {label}!\n«{name}»\n🔗 {meetLink}` (the Meet link line is omitted if empty).
- Interval CSV parsing reuses an exported `botsettings.Parse(csv string) []int` (a thin wrapper over the existing internal `parse`).

`defaultOrganizerOffsets = []int{15}` — constant in the scheduler (configurable later).

## Scheduler (`internal/platform/reminder_scheduler`)

`New(store *postgres.Store, b *bot.Bot, log *zap.Logger)`; `Run(ctx)` — 1-minute ticker, Redis leader lock. `tick(ctx)`:

1. `now := time.Now()`, `until := now.Add(24h)` (max interval is 1 day).
2. `meetings := store.ListUpcomingMeetings(ctx, until)`.
3. For each meeting, build a `recipients map[int64][]int` (telegram_id → offsets):
   - `parts := store.ListParticipants(m.ID)`; for each with an email, `u := store.GetBotUserByEmail(email)` (skip on error / not found); `recipients[u.TelegramID] = botsettings.Parse(u.ReminderMinutes)`.
   - Organizer: if `m.OrganizerUserID != nil`, `tg, ok := store.GetUserTelegramID(*m.OrganizerUserID)`; if `ok` and `tg` is **not** already in `recipients`, set `recipients[tg] = defaultOrganizerOffsets`.
4. For each `(tg, offsets)`: for each `off in dueOffsets(now, m.StartsAt, offsets)`: if `store.TryClaimReminder(m.ID, tg, off)` returns `true`, `bot.SendMessage(ChatID: tg, Text: message(m.Name, m.MeetLink, off))`.

## Wiring

`cmd/server`: `remSched := reminder_scheduler.New(store, tg, logger); go remSched.Run(ctx)` (alongside the scenario scheduler). Best-effort sends: in dev/CI without a real bot token, `SendMessage` fails silently and the ticker is harmless.

## Testing

- **Unit (pure):** `dueOffsets` (below/at/after threshold, multiple offsets, past meeting), `offsetLabel` (each interval + fallback), `message` (with/without Meet link).
- **Integration/manual (out of CI):** real bot + a meeting starting soon with a registered participant and a linked organizer → each gets exactly one DM per due offset; a second tick sends nothing (dedup).

## Out of scope (later)

- §5a created-notification; change/cancel/participant notifications (edit increment).
- Configurable organizer default (fixed `{15}` now).
- Catch-up for reminders whose entire window elapsed during downtime (best-effort: skipped).
- Per-meeting reminder overrides; non-Telegram channels (Google email is already handled by the calendar adapter).
