# Increment B — bot-FSM participant management (§4.3) — design

**Date:** 2026-05-31
**Scope:** ТЗ §4.3 (`docs/NEW-FEATURES.md`) — add/remove meeting guests via the existing `/edit` bot FSM, syncing the Google Calendar attendee list and DMing the affected person. Builds on Increment A (`meetingedit`).

## Goal

From the `/edit` flow, after picking a meeting, an organizer opens a "Participants" sub-screen to add a guest (by email, or by searching the employee directory) or remove one (with confirmation). Each add/remove immediately updates the Google event's attendees (`SendUpdates=all` → Google emails the invite/cancellation) and enqueues a Telegram DM to the affected person.

## Entry point, flow, layering

- **Entry:** the same `/edit` FSM. The post-pick meeting menu gains a "👥 Участники" action. To avoid mixing with the field-edit accumulate-then-apply model, participants live on a **separate sub-screen** and operations are **immediate** (add/remove apply to Google + DB + notification at once). Field overrides remain pending until "Применить"; participant changes commit immediately. This asymmetry is intentional and documented in-app via the separate screen.
- **Flow:**
  - meeting menu → "👥 Участники" → sub-menu: current participants (each with ✖) + "➕ Добавить" + "⬅ Назад".
  - **Add:** prompt email-or-name → search `employees` (substring) → buttons for matches + (if input is a valid email) an "➕ Добавить <email>" button → tap adds.
  - **Remove:** tap ✖ on a participant → confirmation → removal.
- **Layering** (mirrors Increment A):
  - **Commands in `application`:** `AddParticipant(ctx, ws, userID, meetingID uuid.UUID, email string) error` and `RemoveParticipant(ctx, ws, userID, meetingID uuid.UUID, email string) error` — ACL (organizer or workspace owner), `status='scheduled'` is enforced by the meeting load + repo writes, Google attendee sync, enqueue notification. Query `SearchEmployees(ctx, ws uuid.UUID, query string) ([]postgres.Employee, error)`. (Email is the source of truth for attendees + notifications; `meeting_participants.employee_id` is left null in this increment — directory linkage is a later nicety. The employee search only helps the organizer find the right email.)
  - **FSM in `meetingedit`:** extend the `Backend` interface (add/remove/search/list-participants), add sub-states and `medit:parts|padd|prem|premc` callbacks.
  - **ACL:** `userID`/`workspaceID` come from the session (set in `pick` from the owned meeting) — the same airtight gate as Increment A.

## FSM sub-flow

New callbacks (in addition to Increment A's `medit:*`):

- `medit:parts` → participants sub-menu: a `✖ <email>` row per current participant (`medit:prem:<i>`), plus `➕ Добавить` (`medit:padd`) and `⬅ Назад` (`medit:menu`, back to the field menu).
- `medit:padd` → `Step=awaiting`, `AwaitingField="participant"`; prompt "Введи email или часть имени:".
- **OnText when `participant`:** `SearchEmployees(query)` → buttons: one per matching employee (`medit:padd:<i>`, text "ФИО — email"); plus, if `query` is a valid email not already among matches, an "➕ Добавить <email>" button. The candidate emails are stored in the session (`PartCands`) so the callback resolves by index (length-safe, see below). Empty + not-an-email → "Ничего не найдено, введи корректный email или часть имени". Returns to `Step=menu`.
- `medit:padd:<i>` → resolve email from `PartCands[i]`; `AddParticipant(...)`; duplicate → "Уже участник"; success → refreshed participants sub-menu.
- `medit:prem:<i>` → resolve email from `PartList[i]`; confirmation: "Удалить <email>?" with `✅ Да` (`medit:premc:<i>`) / `⬅ Отмена` (`medit:parts`).
- `medit:premc:<i>` → `RemoveParticipant(...)` → refreshed participants sub-menu.

**callback_data length safety:** Telegram caps `callback_data` at 64 bytes; a long email may not fit. So remove/add-candidate callbacks carry an **index** into a session-stored list (`PartList` for current participants, `PartCands` for add candidates), set when the screen is rendered; the email is resolved from the session by index.

**Email normalization:** `mail.ParseAddress` + `strings.ToLower(addr.Address)` (same as `botreg`) before add — prevents case-variant duplicates and strips display-name forms.

**Pure/testable parts:** participants-sub-menu render, add-candidates render, email normalization — pure functions; backend calls behind the `Backend` interface (fakes in tests, as in Increment A).

## Google sync + repo + commands

**Port `CalendarService`** — add:

```go
UpdateAttendees(ctx context.Context, eventID string, emails []string) error
```

- **Google adapter:** `Events.Patch(calendarID, eventID, &calendar.Event{Attendees: <built from emails>, ForceSendFields: []string{"Attendees"}}).SendUpdates("all").Context(ctx).Do()`. Patch replaces the attendee array with the full desired list (post add/remove); `SendUpdates="all"` emails invites to added guests and cancellations to removed ones. `ForceSendFields:["Attendees"]` ensures an empty list actually clears guests.
- **Stub adapter:** no-op.

**Repo** — `RemoveParticipant(ctx, meetingID uuid.UUID, email string) error`:
`DELETE FROM meeting_participants WHERE meeting_id=$1 AND email=$2`. (`AddParticipants`, `ListParticipants` already exist.)

**Employee search** — `Services.SearchEmployees(ctx, ws, query)` = `ListEmployees(ws)` + an in-Go case-insensitive substring filter over `FullName`/`Email`. KISS — the directory is small; no new SQL. (Until the employee-CSV increment seeds the directory, this returns whatever is present, possibly empty.)

**Commands** (mirror `UpdateMeeting`'s ACL/pattern):

- `AddParticipant(ctx, ws, userID, meetingID, email)`: ACL via `GetMeeting`+`GetWorkspace` (organizer or owner, else `ErrForbidden`); normalize email; if already a participant (`ListParticipants`) → wrap `ErrInvalidInput` ("уже участник"); `AddParticipants([{Email: email}])` (employee_id null); recompute `emails := ListParticipants→Email`; if `GoogleEventID != ""` → `Calendar.For(ws).UpdateAttendees(eventID, emails)` (wrap errors `calendar: %w`); best-effort `Queue.EnqueueParticipantAdded(ws, meetingID, email)` (Warn on failure).
- `RemoveParticipant(ctx, ws, userID, meetingID, email)`: ACL; `Store.RemoveParticipant`; recompute `emails`; `UpdateAttendees`; best-effort `Queue.EnqueueParticipantRemoved(ws, meetingID, email)`.

**Ordering:** DB write first, then Google. Unlike `UpdateMeeting` (Google-first), the attendee list sent to Google is derived from the post-write `ListParticipants`, which forces DB-first. If Google then fails, the participant row is committed but the email did not go out — a manual retry reconciles, and Postgres (SoT) stays correct. Documented.

## Notifications `participant:added` / `participant:removed`

- **Queue:** `TaskParticipantAdded = "participant:added"`, `TaskParticipantRemoved = "participant:removed"`, payload `{workspace_id, meeting_id, email}`, `Enqueue*`/`Parse*` — alongside `meeting:updated`. Register both handlers in the asynq mux.
- **Worker (`meeting_notifier`):** `HandleParticipantAdded` / `HandleParticipantRemoved(ctx, ws, meetingID uuid.UUID, email string)` — load meeting+workspace (error → `return err`, asynq retries), resolve `GetBotUserByEmail(email)` → telegram_id (no record → return nil; the person still gets the Google email), DM **best-effort** (Warn on send failure). No dedup — same rationale as `meeting:updated` (all reads precede the single send; the handler returns nil after a best-effort send, so a retry only happens before anything was sent).
- **Messages** (pure functions): add reuses `buildEventMessage("➕ Вас добавили на встречу", name, meetLink, startsAt, endsAt, loc)` from Increment A; remove → new `buildRemovedMessage(name string, startsAt time.Time, loc *time.Location)` → `➖ Вас удалили из встречи\n«<name>»\n🗓 <date> (UTC+N)` (no meet link).

## Testing

- **Unit** — `SearchEmployees` filter (substring, case-insensitive, matches name and email); email normalization; participants sub-menu render and add-candidates render (index → callback data); FSM flows via fakes: `parts → padd → OnText(search) → pick → AddParticipant called` with the right email; duplicate → "Уже участник"; `prem → confirm → premc → RemoveParticipant called`; index→email resolution.
- **Unit** — `buildRemovedMessage` (format, UTC offset); add path reuses the already-covered `buildEventMessage`.
- **Build-verified** — Google `UpdateAttendees`, repo `RemoveParticipant`, `HandleParticipant*` I/O, wiring in `MultiHandler` / `main.go` (same convention as Increment A — no DB harness in the postgres package).
- **Full suite:** `make test && make lint && make build`.

## Out of scope (recorded)

- Seeding `employees` from the embedded CSV — separate increment; search operates over whatever rows exist.
- Participant management for recurring series (§4.4.2) — no series model yet.
- Roles beyond organizer/owner.
- Time-conflict warnings (§4.7).

## Relationship to the platform

Additive. Reuses the `meetingedit` FSM, the asynq queue + `meeting_notifier` + `meetingrecipients`, the Google adapter, and the `UpdateMeeting` command shape from Increment A. No changes to the notify-bot/scenario engine.
