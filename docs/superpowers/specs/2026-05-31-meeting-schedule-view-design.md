# Employee schedule view (§4.6) + participant UNIQUE constraint — design

**Date:** 2026-05-31
**Scope:** ТЗ §4.6 (read-only employee schedule view via the bot) + a defense-in-depth `UNIQUE (meeting_id, email)` constraint on `meeting_participants` (follow-up from the Increment B review). Recurring-series editing (§4.4.2) is a separate later increment.

## Part A — participant UNIQUE constraint

A migration adds `UNIQUE (meeting_id, email)` to `meeting_participants`, and `Store.AddParticipants` switches its INSERT to `ON CONFLICT (meeting_id, email) DO NOTHING`. This is a race backstop; the existing "already a participant" pre-check in `Services.AddParticipant` stays as the user-facing guard.

## Part B — §4.6 employee schedule view

A read-only Telegram flow: any registered user looks up an employee (by email, or by searching the directory) and views that person's meetings, filtered by a time window.

### Identity, access, data

- **Access:** all registered `bot_users`. The `/schedule` command is gated on `GetBotUserByTelegramID` (the correct gate here — §4.6 is "all registered users"; this is unlike `/edit`, which is keyed to `platform_users` meeting ownership).
- **Employee identity = email.** The bot is global (`bot_users` have no workspace); the `employees` directory is workspace-scoped and currently unseeded (CSV is a separate increment). So employee lookup is **global** (ILIKE across all `employees`), plus direct raw-email entry (mirrors Increment B's add flow). Once an email is chosen, its schedule is shown.
- **Schedule = meetings where the email participates OR organizes.** A single query:
  `SELECT DISTINCT <meeting cols> FROM meetings m LEFT JOIN meeting_participants mp ON mp.meeting_id=m.id LEFT JOIN platform_users pu ON pu.id=m.organizer_user_id WHERE (mp.email=$1 OR pu.email=$1) AND m.status='scheduled' AND m.starts_at>=$2 AND m.starts_at<$3 ORDER BY m.starts_at`. Organized meetings are included via the organizer's `platform_users.email`.
- **Date-window timezone:** day boundaries are computed in the platform base TZ **Asia/Almaty** (a schedule can span meetings from multiple workspaces; a single base TZ is KISS and matches the ТЗ base of UTC+5).
- **Row status:** `starts_at > now → предстоящая (🔜), else прошедшая (✅)` (pure function). Cancelled meetings are excluded (`status='scheduled'`).

### FSM (new package `internal/platform/scheduleview`)

Mirrors `meetingedit`: Redis session (key `sched:<telegramID>`, 15m TTL), `Reply{Text, Keyboard [][]Button, Edit bool}`, a `Backend` interface satisfied by `*application.Services`.

**State:** `{ Step, EmployeeEmail string, AwaitingKind string ("search"|"date"|"range"), Cands []string }`.

**Flow:**

1. `/schedule` → `AwaitingKind="search"`, prompt "Введи email сотрудника или часть имени:".
2. **OnText (search):** `SearchEmployeesGlobal(query)` → buttons per match (`sched:pick:<i>`, text "ФИО — email") + (if `query` is a valid email) a "Расписание <email>" button (`sched:pick:<i>`); candidates stored in `Cands`. Empty → "Ничего не найдено…". Stays awaiting (re-typing re-searches).
3. **`sched:pick:<i>`** → `EmployeeEmail = Cands[i]` (index is length-safe for callback_data, per Increment B) → show the period menu (Edit).
4. **Period menu** (inline): Сегодня (`sched:d:today`) · Завтра (`sched:d:tomorrow`) · Все предстоящие (`sched:d:upcoming`) · Конкретная дата (`sched:d:date`) · Диапазон (`sched:d:range`) · ⬅ Другой сотрудник (`sched:back`).
5. **today/tomorrow/upcoming** → compute the window and render the list immediately.
6. **date** → `AwaitingKind="date"`, prompt "Введи дату ГГГГ-ММ-ДД:" → parse → window `[D, D+1)` → list.
7. **range** → `AwaitingKind="range"`, prompt "Введи диапазон ГГГГ-ММ-ДД..ГГГГ-ММ-ДД:" → parse → window `[D1, D2+1)` → list.
8. **List:** header "Расписание <email>: <period>" + rows `<status emoji> «name» — DD.MM.YYYY HH:MM–HH:MM` (🔜 предстоящая / ✅ прошедшая). Empty → "Встреч нет". A ⬅ button returns to the period menu.

**Pure/testable parts:** `parseDate` (ГГГГ-ММ-ДД), `parseRange` (`..`, end≥start), `dayWindow(now, kind, loc) (from, to)` for the presets, list rendering + status emoji, index resolution. Backend calls (search, schedule query) are behind the interface (fakes in tests).

### Repo + application

**Repo:**

- `SearchEmployeesGlobal(ctx, query string) ([]Employee, error)` — `... WHERE full_name ILIKE '%'||$1||'%' OR email ILIKE '%'||$1||'%' ORDER BY full_name LIMIT 20` (parameterized).
- `ListScheduleForEmail(ctx, email string, from, to time.Time) ([]Meeting, error)` — the DISTINCT LEFT-JOIN query above, scanning `meetingColsM`.

**Application (thin delegates):**

- `SearchEmployeesGlobal(ctx, query) ([]postgres.Employee, error)` → Store.
- `EmployeeSchedule(ctx, email string, from, to time.Time) ([]postgres.Meeting, error)` → Store.ListScheduleForEmail.

The FSM computes `[from,to)` via the pure helpers with `time.Now()` in Almaty; the application stays free of time logic.

### Wiring

`MultiHandler` gains a `schedule *scheduleview.Service` field (built in `NewMultiHandler` from the same `services` backend + `rdb`). The `/schedule` command (gated on `GetBotUserByTelegramID`) calls `schedule.Start`; `sched:*` callbacks route to `schedule.OnCallback`; free text falls through the chain `registrar → editor → schedule.OnText`. The `NewMultiHandler` signature does not grow (the editor backend `services` already satisfies the new `Backend` too). No asynq changes (read-only).

## Testing

- **Unit (scheduleview):** `parseDate` (ok/malformed), `parseRange` (ok / `..` / end-before-start), `dayWindow` (today/tomorrow/upcoming boundaries in Almaty), status emoji (future/past by now), list render (incl. empty), index resolution; FSM flows via fakes — search → pick → today → `EmployeeSchedule` called with the right window; date/range input.
- **Build-verified:** repo `SearchEmployeesGlobal` / `ListScheduleForEmail` (ILIKE + JOIN), application delegates, the UNIQUE migration + `ON CONFLICT`, wiring (`MultiHandler`/`main.go`) — same convention as prior increments (no DB harness in the postgres package).
- **Full suite:** `make test && make lint && make build`.

## Out of scope (recorded)

- Recurring-series editing §4.4.2 (next increment; needs a real RRULE series engine).
- Seeding `employees` from the embedded CSV — search operates over whatever rows exist.
- Editing others' schedules — §4.6 is read-only by spec.

## Relationship to the platform

Additive. Part A hardens the participant table. Part B adds a read-only bot flow reusing the FSM pattern (`meetingedit`/`botreg`), the employee directory, and the meetings DB. No changes to the notify-bot/scenario engine.
