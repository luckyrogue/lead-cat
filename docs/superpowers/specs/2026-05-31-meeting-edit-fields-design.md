# Increment A — bot-FSM meeting field editing (§4.4.1) — design

**Date:** 2026-05-31
**Scope:** ТЗ §4.4.1 (`docs/NEW-FEATURES.md`) — edit a meeting's fields via a Telegram bot FSM, sync the Google Calendar event, recompute the name, and notify participants. Participants management (§4.3) is a separate increment (B) on the same FSM scaffold.

## Goal

A registered user edits one of their upcoming meetings through a Telegram conversation: pick meeting → edit fields (date/time, dept, type, host, description, recurrence) → apply. On apply, the Google event is patched (which emails attendees), the meeting name is recomputed, the row is persisted, and participants + organizer get a Telegram DM about the change.

## Identity & scope

- **Who can edit, and which meetings:** the bot resolves `platform_users` by the Telegram ID on the incoming Update; it lists that user's **own upcoming** meetings (`status='scheduled' AND starts_at > now()`), joined via `meetings.organizer_user_id = platform_users.id`. Requires the organizer to have linked Telegram in the Mini App (`platform_users.telegram_id`).
- **Out of scope (this increment):**
  - Participant add/remove (§4.3) — Increment B.
  - Recurring-series edit (§4.4.2) — there is no real series yet (`buildEvent` sets no RRULE; recurrence is metadata only).
  - Admin editing **other** users' meetings via the bot (`BOT_ADMIN_TELEGRAM_IDS` / workspace owner) — the list query returns only the caller's own meetings.
  - Time-conflict warning (§4.7).

## Layering

- **Business command in `application`** (mirrors `CreateMeeting`):
  - `Services.UpdateMeeting(ctx, workspaceID, userID, meetingID uuid.UUID, in UpdateMeetingInput) (postgres.Meeting, error)`
  - `Services.ListEditableMeetings(ctx, telegramID int64) ([]postgres.Meeting, error)` (query)
- **FSM in a new package `internal/platform/meetingedit`** (mirrors `botreg`): Redis session, states, keyboards. Returns `(text, keyboard, handled)`; `MultiHandler` sends/edits the message. The FSM depends on a narrow backend interface (defined in `meetingedit`) implemented by `*application.Services`, so orchestration does not leak into the bot layer.
- **Wiring in `MultiHandler`** (`infrastructure/telegram`, which may import `application`): routes `/edit`, `medit:*` callbacks, and free text when an edit session is active (after `registrar.OnText` returns `handled=false`).
- **User resolution:** add `Store.GetUserByTelegramID(ctx, telegramID) (User, bool, error)`. `userID` is stored in the session at edit start; `workspace_id` comes from the picked meeting (already present in the list rows).

## Bot FSM

**Session** key `meetedit:<telegramID>`, TTL 15m, JSON:

```
{ userID, workspaceID, meetingID, awaitingField, overrides{dept?,type?,host?,description?,recurrence?,date?,start?,end?} }
```

**Flow (accumulate-then-apply — one GCal update, one notification):**

1. `/edit` (private chat) → resolve user; if missing/unlinked → prompt to link in the Mini App. Else list upcoming meetings with inline buttons `medit:pick:<meetingID>`.
2. **Field menu** (inline): Дата/время · Отдел · Тип · Ведущий · Описание · Частота · ✅ Применить · ✖ Отмена. Above it: the meeting's current state with accumulated overrides applied.
3. Tapping a field:
   - **Text fields** (dept/type/host/description): set `awaitingField`, prompt for a value; the next text message is written to `overrides`, then return to the field menu.
   - **Date/time**: prompt `ГГГГ-ММ-ДД ЧЧ:ММ–ЧЧ:ММ` (start–end, same day); parse in the workspace TZ; domain-validate; on error re-prompt.
   - **Recurrence**: inline options (Однократно/Ежедневно/Еженедельно/Раз в 2 недели/Ежемесячно) `medit:set:rec:<value>`.
4. **✅ Применить** → `UpdateMeeting(overrides)` → show the updated meeting (name/time/Meet), clear the session. **✖ Отмена** → clear the session.

**Rules:** empty overrides + Применить → "нет изменений". Name is always recomputed from the resulting fields (`meeting.GenerateName`). Session expires after 15m; free text with no active `meetingedit` session is ignored (`handled=false`) and falls through the handler chain.

**Pure (testable) parts:** the date/time input parser, the menu/summary renderers, and the state transitions are pure functions in `meetingedit`; the I/O (Redis, SendMessage) is a thin shell.

## Calendar `UpdateEvent` (Google Patch) + repo + command

**Port `CalendarService`** — add:

```go
UpdateEvent(ctx context.Context, eventID string, e CalendarEvent) error
```

- **Google adapter:** `Events.Patch(calendarID, eventID, body).SendUpdates("all").Context(ctx).Do()`, where `body` carries `{Summary, Description, Start, End}` from `CalendarEvent`. It deliberately omits `Attendees` and `ConferenceData` — Patch leaves absent fields untouched, so the attendee list and Meet link are preserved (participants are Increment B). `SendUpdates("all")` emails attendees about the change (the "Google Calendar update" half of the ТЗ requirement).
- **Stub adapter:** no-op (like `DeleteEvent`).

**Repo** — `Store.UpdateMeeting(ctx, workspaceID, id uuid.UUID, m Meeting) error`:
`UPDATE meetings SET dept, type, host, starts_at, ends_at, recurrence, description, name = … WHERE workspace_id=$ AND id=$ AND status='scheduled'`. The `status='scheduled'` guard mirrors `CancelMeeting` (cannot edit a cancelled meeting).

**Command `Services.UpdateMeeting`** (mirrors `CreateMeeting`):

```go
type UpdateMeetingInput struct {
    Dept, Type, Host, Date, Start, End, Recurrence, Description *string // nil = leave unchanged
}
```

1. `GetMeeting(ws, id)` (workspace-scoped) → current meeting.
2. ACL: `userID == organizer` or workspace owner, else `ErrForbidden`.
3. Apply non-nil overrides onto the current fields; parse date/time in the workspace TZ (errors → `%w ErrInvalidInput`).
4. `meeting.Input.Validate()`; `name = GenerateName(...)`.
5. `Calendar.For(ws).UpdateEvent(eventID, CalendarEvent{Title:name, Description, Start, End})`; if `GoogleEventID == ""` (stub / no Google) skip the call; a GCal error → wrap with `%w` (the bot shows "не удалось обновить").
6. `Store.UpdateMeeting(...)`, then `Queue.EnqueueMeetingUpdated(ws, id)` (best-effort, `Warn` on failure — as in `CreateMeeting`).

**Query `Services.ListEditableMeetings(ctx, telegramID)`** → `Store.ListMeetingsByOrganizerTelegram(ctx, telegramID)` (JOIN `platform_users` ON `telegram_id`, `status='scheduled' AND starts_at > now() ORDER BY starts_at`).

## Notification `meeting:updated`

- **Queue:** new `TaskMeetingUpdated = "meeting:updated"`, payload `{workspace_id, meeting_id}`, `EnqueueMeetingUpdated`, `ParseMeetingUpdated` — alongside `meeting:created`. Register the handler in the asynq mux.
- **Worker:** `meeting_notifier.HandleUpdated(ctx, ws, id)` — loads meeting+workspace (error → `return err`, asynq retries), `meetingrecipients.Resolve` (error → `return err`), then DMs each recipient **best-effort** (log on failure).
- **No dedup, by design:** a meeting may be edited multiple times — each edit is its own notification. Double-send on retry is avoided the same way as §5a: all reads precede all sends, and sends are best-effort (handler returns `nil`), so a retry only happens before anything was sent. (The `meeting_reminders` sentinel is **not** reused — it is permanent.)
- **Message** (new pure function `buildUpdatedMessage`): `✏️ Встреча изменена\n«<name>»\n🗓 <date>, <HH:MM–HH:MM> (UTC+N)\n🔗 <link>` — reuses `tzLabel` and the time format from `message.go`.

## Testing

- **Unit — `meetingedit`:** date/time parser (ok / malformed / end-before-start), FSM transitions (pick → menu → awaiting → overrides → apply/cancel), summary rendering. Uses a fake store/sessions.
- **Unit — `buildUpdatedMessage`:** format, empty meet link omits the link line, UTC offset label.
- **Unit — `Services.UpdateMeeting`:** override application + name recompute + ACL (organizer ok / other → `ErrForbidden`) + `ErrInvalidInput` on a bad date — via a fake calendar/queue (domain logic tested; the storage path is build-verified).
- **Build-verified:** Google `UpdateEvent` Patch, repo `UpdateMeeting` / `ListMeetingsByOrganizerTelegram`, `HandleUpdated` I/O, wiring in `MultiHandler` / `main.go` (same convention as reminder/notifier — no DB harness in the postgres package).
- **Full suite:** `make test && make lint && make build`.

## Relationship to the platform

Additive. Reuses the asynq queue + `meeting_notifier` + `meetingrecipients` from §5a, the bot FSM pattern from `botreg`, the Google adapter, and the `CreateMeeting` orchestration shape. No changes to the notify-bot/scenario engine.
