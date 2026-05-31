# §4.5 — meeting/series deletion ("this / whole series") + cancellation notification — design

**Date:** 2026-05-31
**Scope:** ТЗ §4.5 — delete a meeting from the `/edit` flow; for a series, delete "this occurrence" or "the whole series (this and following)". Adds the cancellation notification (`meeting:cancelled`) that the ТЗ requires (currently missing). Builds on the materialized-series model + the `meetingedit` scope FSM (§4.4.2).

## Architecture

Deletion is reached from the **/edit menu** (reuses meeting selection + the scope screen from §4.4.2). The scope screen is reworded neutrally ("Эта встреча / Вся серия (эта и далее)?") since the chosen scope now governs both Apply (edit) and Delete.

- **"🗑 Удалить"** button in the menu (both scopes) → confirmation → cancel per the chosen scope.
- **Scope=one** → the existing `CancelMeeting`, retrofitted: an early return when the meeting is no longer `scheduled` (idempotent, no spurious notice), and an `enqueue meeting:cancelled` after a successful cancel — this closes the current gap (so the REST delete also notifies).
- **Scope=series** → new `CancelSeries`: cancels the picked occurrence and all later ones (`series_id` + `starts_at >= picked.starts_at` + `status='scheduled'`).

**Cancellation notification** `meeting:cancelled` — to participants + organizer, **one DM** (for the picked occurrence). The worker reads the (now-cancelled) meeting + participant rows from the DB (`GetMeeting` has no status filter; participant rows are not deleted on cancel).

**CancelSeries ordering (DB-first):** list occurrences (for their Google event IDs) → cancel all future occurrences in **one atomic UPDATE** → Google `DeleteEvent` per occurrence **best-effort** (a Google delete is irreversible, so DB-first keeps Postgres the source of truth; a lingering Google event is the lesser evil, logged on failure) → one `meeting:cancelled`. The single `CancelMeeting` keeps its current order (Google-delete → DB); only the enqueue is added.

**Future-from-selected** (consistent with §4.4.2). A confirmation step is required (ТЗ).

## FSM (`internal/platform/meetingedit`)

- **`menuKeyboard`** (both scopes) gains a `{🗑 Удалить}` row (`medit:delete`) before "Применить/Отмена".
- **`medit:delete`** → confirmation screen (Edit): text by scope — series: "Удалить всю серию (эту и далее)? Это отменит все будущие встречи серии." / one: "Удалить эту встречу?"; buttons `✅ Да, удалить` (`medit:delconf`) / `⬅ Отмена` (`medit:menu`, already exists).
- **`medit:delconf`** → `Scope=="series"` → `backend.CancelSeries(ws,user,mid)` → "Удалено встреч серии: N"; else → `backend.CancelMeeting(ws,user,mid)` → "Встреча удалена ❌". Error mapping: `ErrForbidden` → "нет доступа" + session del; `postgres.ErrMeetingNotEditable` → "встреча больше недоступна" + del; else generic. Success → session del.
- **Backend interface** gains `CancelMeeting(ctx, ws, userID, meetingID uuid.UUID) error` and `CancelSeries(ctx, ws, userID, meetingID uuid.UUID) (int, error)` (both on `*application.Services`; `CancelMeeting` already exists there).
- **Scope screen** reworded: "Эта встреча или вся серия (эта и далее)?" with `📍 Эта встреча` (`medit:scope:one`) / `🔁 Вся серия (эта и далее)` (`medit:scope:series`) — text only; callbacks unchanged.
- The existing `Scope=="" && SeriesID!=""` guard already blocks any apply/delete before a scope is chosen (delete buttons live in the menu, rendered only after scope).

**Pure/testable:** the scope-aware confirm render; the `delconf` branching via fakes (CancelMeeting vs CancelSeries per scope; non-series → CancelMeeting).

## Application + repo

**Repo:**
- `CancelSeriesOccurrences(ctx, ws, seriesID uuid.UUID, fromStart time.Time) (int, error)` — one atomic `UPDATE meetings SET status='cancelled', updated_at=now() WHERE series_id=$1 AND workspace_id=$2 AND starts_at>=$3 AND status='scheduled'`; returns `RowsAffected` (no transaction needed — single statement).

**Application:**
- `CancelMeeting` (retrofit): after the ACL check, `if m.Status != "scheduled" { return nil }`; then the current Google-delete best-effort + `Store.CancelMeeting`; finally `s.enqueueCancelled(ctx, ws, id)`.
- `CancelSeries(ctx, ws, userID, meetingID uuid.UUID) (int, error)`: `GetMeeting` + `GetWorkspace`; ACL `ownerOrOrganizer`; `picked.SeriesID == nil → %w ErrInvalidInput`; `occs := ListSeriesOccurrences(ws, *picked.SeriesID, picked.StartsAt)` (for the event IDs); `n := CancelSeriesOccurrences(...)`; Google `DeleteEvent` per `occs[i].GoogleEventID` (best-effort, logged via the existing `deleteEventsBestEffort`); `s.enqueueCancelled(ws, meetingID)` once; return `n`.
- `enqueueCancelled` — a shared best-effort helper (mirrors `enqueueCreated`).

## Notification `meeting:cancelled`

- **Queue:** `TaskMeetingCancelled = "meeting:cancelled"`, payload `{workspace_id, meeting_id}`, `EnqueueMeetingCancelled`, `ParseMeetingCancelled` — alongside created/updated. Register the handler in the asynq mux (`main.go`).
- **Worker (`meeting_notifier`):** `HandleCancelled(ctx, ws, meetingID)` (mirrors `HandleUpdated`): loads the meeting (already `cancelled`; `GetMeeting` has no status filter) + workspace → loc → `meetingrecipients.Resolve` → best-effort DM each. No dedup (reads precede sends; best-effort).
- **Message** (pure `buildCancelledMessage`): `❌ Встреча отменена\n«<name>»\n🗓 <date> (UTC+N)` (reuses `tzLabel`, no meet link).

## Testing

- **Unit:** `buildCancelledMessage` (format, UTC offset, no 🔗); FSM deletion via fakes — `medit:delete` → confirm (text by scope), `medit:delconf` with scope=one → `CancelMeeting` called, scope=series → `CancelSeries` called; a non-series meeting → `CancelMeeting`.
- **Build-verified:** `CancelSeriesOccurrences`, `CancelSeries` orchestration (Google best-effort + enqueue-once), the `CancelMeeting` retrofit (early-return + enqueue), `HandleCancelled` I/O, the asynq handler registration — per the repo convention (concrete `Store`, no DB harness).
- **Full suite:** `make test && make lint && make build`.

## Out of scope (recorded)

- Deleting "all including past" occurrences (only future-from-selected).
- Restoring cancelled meetings.
- Changing the recurrence pattern.

## Relationship to the platform

Additive. Reuses the `meetingedit` scope FSM, `ListSeriesOccurrences`, `ownerOrOrganizer`, the Google `DeleteEvent` adapter, and the notification pattern. The cancellation notification also benefits the existing single-cancel + REST delete paths. No changes to the notify-bot/scenario engine.
