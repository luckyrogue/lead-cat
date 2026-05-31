# §4.4.2 — recurring-series editing ("this / whole series") — design

**Date:** 2026-05-31
**Scope:** ТЗ §4.4.2 — when editing a meeting that belongs to a series, let the organizer choose "this occurrence" or "the whole series (this and following)". Builds on the materialized-occurrence series engine + the `/edit` `meetingedit` FSM.

## Flow & scope

In `/edit`, after picking a meeting: if it has a `series_id`, ask **"Редактировать: эту встречу / всю серию (эту и далее)?"**. No `series_id` → the current single-occurrence edit (unchanged).

- **"This occurrence"** → the existing `UpdateMeeting` (full date+time, all fields) — unchanged.
- **"Whole series"** → new `UpdateSeries`: applies to the picked occurrence and all later ones (`series_id` + `starts_at >= picked.starts_at` + `status='scheduled'`). Past occurrences are not touched.

**Series-wide editable fields:** dept / type / host / description applied verbatim to every occurrence; **time-of-day** (HH:MM start–end) applied to each occurrence **keeping its own date**; the name is recomputed per occurrence. The recurrence pattern and the occurrence dates are NOT changed (re-materialization is out of scope).

**Notification:** one `meeting:updated` DM per series (for the picked occurrence) — reuses the §4.4.1 `meeting:updated` task + `HandleUpdated`. Participants are the same across occurrences, so one DM suffices.

**Ordering (DB-first for the series):** unlike single `UpdateMeeting` (Google-first), a series' Google patches are not reversible, so `UpdateSeries` writes the DB atomically (one transaction), then patches Google best-effort (logged on failure). Postgres stays the source of truth; a failed Google patch is reconciled by re-editing. Recorded as a known edge.

## FSM (`internal/platform/meetingedit`)

- **State** gains `SeriesID string` (from the picked `MeetingWithTZ.SeriesID`, empty if none) and `Scope string` (`"one"` | `"series"`).
- **pick:** loads the meeting; if `SeriesID != ""` → show a scope screen (inline): "✏️ Эту встречу" (`medit:scope:one`) / "🔁 Всю серию (эту и далее)" (`medit:scope:series`). No series → straight to the field menu (`Scope="one"`).
- **`medit:scope:one`** → `Scope="one"`, normal field menu (full date+time, "Частота" present).
- **`medit:scope:series`** → `Scope="series"`, **series menu**: the date/time field is labelled "🕒 Время" and its text prompt accepts only `ЧЧ:ММ–ЧЧ:ММ` (no date); there is no "Частота" button (pattern unchanged); dept/type/host/description as usual. The summary notes "(вся серия с DD.MM.YYYY)".
- **OnText (time field):** `Scope="series"` → `parseTimeRange(text)` (new pure parser `HH:MM–HH:MM`, end > start) → `overrides[start]`/`overrides[end]` (no `date`). `Scope="one"` → the existing `parseDateTime` (date+time).
- **apply:** `Scope=="series"` → `backend.UpdateSeries(ws,user,meetingID, seriesInput(overrides))`; error mapping as today (+ "not a series" → generic); success → "Готово ✏️ — обновлено N встреч серии". `Scope=="one"` → the existing `UpdateMeeting`.
- **Backend interface** gains `UpdateSeries(ctx, ws, userID, meetingID uuid.UUID, in application.SeriesUpdateInput) (int, error)` (satisfied by `*application.Services`).

**Pure/testable:** `parseTimeRange`; the series menu/summary renderers; the scope branching in OnCallback/OnText/apply — via fakes (as in the current `meetingedit` tests).

## Application + repo

**`SeriesUpdateInput`** (application): `Dept, Type, Host, Description *string` + `Start, End *string` (HH:MM). No `Date`, no `Recurrence`.

**Pure helper** `applySeriesUpdate(cur postgres.Meeting, in SeriesUpdateInput, loc *time.Location) (postgres.Meeting, error)` (mirrors `applyMeetingUpdate`): applies the non-nil field overrides; if `Start`+`End` are set, combines the occurrence's own date (in `loc`) with the new HH:MM → new `starts_at`/`ends_at` (UTC), else keeps the current times; domain-validates (wrap `ErrInvalidInput`); recomputes the name = `GenerateName(..., localStart, cur.Recurrence)` (the occurrence's recurrence label is preserved). Unit-tested.

**Repo:**
- `ListSeriesOccurrences(ctx, ws, seriesID uuid.UUID, fromStart time.Time) ([]Meeting, error)` — `series_id=$ AND workspace_id=$ AND starts_at>=$ AND status='scheduled' ORDER BY starts_at` (via `queryMeetings`/`scanMeeting`).
- Extract a shared `updateMeetingSQL` + `updateMeetingArgs` (refactor of the existing `UpdateMeeting`, like `insertMeetingSQL`), plus `UpdateMeetingsTx(ctx, ws uuid.UUID, ms []Meeting) error` — `pool.Begin` → per row `UPDATE … WHERE id+workspace_id+status='scheduled'`; `RowsAffected()==0 → ErrMeetingNotEditable` (rollback); `Commit`.

**`Services.UpdateSeries(ctx, ws, userID, meetingID uuid.UUID, in SeriesUpdateInput) (int, error)`:**
1. `GetMeeting` + `GetWorkspace`; ACL `ownerOrOrganizer`; `picked.SeriesID == nil → fmt.Errorf("%w: not a series", ErrInvalidInput)`.
2. `loc` from workspace TZ; `occs := ListSeriesOccurrences(ws, *picked.SeriesID, picked.StartsAt)`.
3. `rows := applySeriesUpdate(occ, in, loc)` for each (error → `%w ErrInvalidInput`).
4. `UpdateMeetingsTx(ws, rows)` (atomic).
5. Google: for each row with `GoogleEventID != ""` → `calSvc.UpdateEvent(eventID, {Title:name, Description, Start, End})` best-effort, log on failure.
6. `Queue.EnqueueMeetingUpdated(ws, meetingID)` once (reuses the §4.4.1 task + `HandleUpdated`).
7. Return `len(rows)`.

## Testing

- **Unit (domain/application):** `parseTimeRange` (ok / malformed / end ≤ start); `applySeriesUpdate` (field override + name recompute; time-of-day applied to the occurrence's date keeps the date and changes HH:MM; empty time overrides leave the time untouched; bad → `ErrInvalidInput`; occurrence's recurrence preserved).
- **Unit (`meetingedit`):** scope branching via fakes — a series meeting → scope screen; `scope:series` → series menu (no "Частота"); time input `ЧЧ:ММ–ЧЧ:ММ` → overrides without `date`; apply → `backend.UpdateSeries` called; a non-series meeting → straight to the field menu.
- **Build-verified:** `ListSeriesOccurrences`, `UpdateMeetingsTx` (tx), the `UpdateSeries` orchestration (Google best-effort + enqueue-once) — same convention as the `CreateMeeting` series path (concrete `Store`, no DB harness).
- **Full suite:** `make test && make lint && make build`.

## Out of scope (recorded)

- Changing the recurrence pattern/frequency of a series (re-materialization).
- §4.5 deleting a series ("this / whole").
- Editing past occurrences.

## Relationship to the platform

Additive. Reuses the `meetingedit` FSM, `applyMeetingUpdate`'s shape, `meeting:updated` notification, the Google `UpdateEvent` adapter, and the `series_id` model. No changes to the notify-bot/scenario engine.
