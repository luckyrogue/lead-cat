# Meetings (Google Meet) — feature

A Telegram Mini App feature for scheduling and managing **Google Meet** meetings inside an organization, layered on top of the existing Lead Cat platform. The full specification (the source of truth for behavior) is **[NEW-FEATURES.md](NEW-FEATURES.md)** (ТЗ). This page is the engineering summary and status.

> **Status:** frontend Mini App is built against **mock data** (`frontend/src/shared/tma/mock-data.ts`); the backend (Google Calendar integration, persistence, employee directory) is **not implemented yet** — to be built per the ТЗ. Do not treat backend endpoints as existing.

## Concept (per ТЗ)

- All meetings are created through **one corporate Google service account** — the organizer of every Google Calendar / Meet event.
- Users are bound by **Telegram ID + corporate email**; a user record is created automatically on `/start`.
- The employee directory (for adding participants) is loaded from a **CSV embedded at deploy time**.
- Transcription/post-processing is handled by an **external service** — out of scope for this bot.
- Base timezone: **UTC+5 (Almaty)**.
- Roles: **User** (auto on `/start`) and **Main Administrator** (full access to all meetings).

## Frontend (implemented, mock-backed)

Telegram Mini App under `frontend/src/features/tma/` + `frontend/src/shared/tma/`, shell in `frontend/src/widgets/tma-shell/`. Five tabs (`TabKey`):

| Tab        | Screen                | Purpose                                     |
| ---------- | --------------------- | ------------------------------------------- |
| `home`     | `home-screen.tsx`     | Overview / quick actions                    |
| `meetings` | `meetings-screen.tsx` | List & view meetings                        |
| `checker`  | `checker-screen.tsx`  | Common free-slot finder across participants |
| `auto`     | `auto-screen.tsx`     | Automation / recurring scenarios            |
| `profile`  | `profile-screen.tsx`  | User profile & settings                     |

Meeting creation flow: `create-wizard.tsx`. Uses the cat design system (see [DESIGN-CATS.md](DESIGN-CATS.md)) and i18n (`ru` / `kk` / `en`).

### Data model (`frontend/src/shared/tma/types.ts`)

- **`Employee`** — `id, name, email, dept, tg, role?`
- **`Meeting`** — `id, type, dept, host, date, start, end, rec, recDays?, organizer, participants[], desc?`
- **`MeetingDraft`** — create/edit form state (`dur`, `recDays`, `participants: Employee[]`, …)
- **`FreeSlot`** — `day, iso, start, end, mins` (free-slot checker results)

## Functional areas (per ТЗ §4–5)

Create meeting (fields, meeting types, recurrence, naming standard), view meetings, manage participants (add/remove), edit (incl. recurring series), delete/cancel, employee schedule view, **time-conflict warning** on create/edit, **common free-time checker**, and **notifications** (reminders + Google Meet link). See [NEW-FEATURES.md](NEW-FEATURES.md) for exact rules.

## Backend (planned)

> **Increment 1 (done):** meeting CRUD over REST (`/api/workspaces/:id/meetings`, `/employees`) backed by a stubbed `CalendarService`. Real Google Calendar adapter, recurrence series, conflict detection, free-slot checker, notifications, and bot registration remain planned (below).
> **Increment 2 (done):** real Google Calendar adapter — per-workspace encrypted service-account creds + domain-wide delegation, Meet link via `conferenceData`. Provider is selected per workspace; `CALENDAR_STUB=true` forces the stub (local/CI). Configure via `PATCH /api/workspaces/:id/integrations` (`google_sa_json`, `google_subject`, `google_calendar_id`); a workspace without creds returns **400 "google not configured"** on meeting create.
> **Bot registration (done):** `/start` FSM (ФИО → corporate email → OTP) creates a global `bot_users` record (Telegram ID ↔ email ↔ name + role). Admins bootstrapped via `BOT_ADMIN_TELEGRAM_IDS`. FSM state in Redis; OTP reuses the email auth service. Requires the bot polling (real `BOT_TOKEN`, non-dev). Per-participant notifications (§5) will join `email → bot_users.telegram_id`.
> **Reminder settings (done):** `/settings` shows an inline keyboard (10м/15м/30м/1ч/2ч/1день); tapping toggles the interval and saves it to `bot_users.reminder_minutes` (CSV, default `15`, empty = off). The reminder **engine** that sends them (§5b-2) is the next increment.
> **Reminder engine (done):** a 1-minute scheduler (Redis leader lock) DMs upcoming-meeting reminders — registered participants by their `/settings` intervals, plus the organizer (linked Telegram) at a 15-minute default. Durable dedup via `meeting_reminders` (one DM per meeting/user/offset). Best-effort send; needs the bot polling.
> **Meeting-created notification (§5a, done):** on create, a `meeting:created` asynq job DMs the recipients (registered participants + organizer) the meeting name, time (workspace TZ), and Meet link. Recipient resolution is shared with the reminder engine (`meetingrecipients.Resolve`); dedup reuses `meeting_reminders` with a sentinel offset (`postgres.ReminderOffsetCreated`). Best-effort send; needs the bot polling.
> **Meeting field editing (§4.4.1, done):** a `/edit` bot FSM (Redis session) lets an organizer edit their upcoming meeting's fields (date/time, dept, type, host, description, recurrence); on apply the name is recomputed, the Google event is patched (`SendUpdates=all` emails attendees), the row is persisted, and a `meeting:updated` asynq job DMs participants + organizer. Meetings are resolved via `platform_users.telegram_id`. Participant management (§4.3), recurring-series edit (§4.4.2), bot admin-edit of others' meetings, and conflict warnings (§4.7) remain planned.
> **Participant management (§4.3, done):** the `/edit` FSM gains a "👥 Участники" sub-screen — add a guest by email or by searching the employee directory, remove one with confirmation. Each op syncs the Google attendee list (`UpdateAttendees`, `SendUpdates=all` → Google emails the invite/cancellation) and enqueues a `participant:added`/`participant:removed` asynq job that DMs the affected person (if registered). Participant ops apply immediately (separate from field accumulate-then-apply). Employee-directory CSV seeding, recurring-series participants (§4.4.2), and conflict checks (§4.7) remain planned.
> **Employee schedule view (§4.6, done):** a read-only `/schedule` bot flow — look up an employee (global directory search or raw email), then view their scheduled meetings (where they participate or organize) filtered by Сегодня/Завтра/Все предстоящие/конкретная дата/диапазон. Day windows are computed in Asia/Almaty; rows show 🔜 upcoming / ✅ past. Also: `meeting_participants` now has a `UNIQUE (meeting_id, email)` constraint (AddParticipants uses `ON CONFLICT DO NOTHING`). Recurring-series editing (§4.4.2) and employee-CSV seeding remain planned.
> **Recurring-series creation (done):** creating a meeting with `recurrence != once` + a required `recurrence_until` materializes the series into individual occurrence rows (linked by `series_id`), each its own Google event and reminders. Occurrences are expanded by the pure `meeting.Occurrences` (cap 100). Google events are created first (best-effort compensation, logged, on failure), then all rows + participants are inserted in one DB transaction (`CreateMeetingSeries`); the `meeting:created` DM fires once per series. Tradeoffs: each occurrence has its own Meet link, and series are bounded by `until` (no open-ended/re-materialization yet). Editing a series "this/whole" (§4.4.2) is the next increment.
> **Recurring-series editing (§4.4.2, done):** editing a meeting that belongs to a series, `/edit` asks "эту встречу / всю серию (эту и далее)". "Whole series" (`UpdateSeries`) applies dept/type/host/description and a time-of-day change (HH:MM kept on each occurrence's own date) to the picked occurrence and all later scheduled ones — one DB transaction, Google events patched best-effort, one `meeting:updated` DM. The recurrence pattern and occurrence dates are unchanged (re-materialization out of scope). "This occurrence" is the existing single-meeting flow.
> **Deletion + cancellation notice (§4.5, done):** `/edit` gains a "🗑 Удалить" action with a confirmation step; for a series the chosen scope ("эта встреча / вся серия") governs it — single → `CancelMeeting`, series → `CancelSeries` (cancels the picked occurrence and all later scheduled ones in one atomic UPDATE, deletes Google events best-effort). All cancellations (incl. the single path and the REST delete) now enqueue a `meeting:cancelled` DM to participants + organizer (one per delete). `CancelMeeting` is idempotent (no-op on an already-cancelled meeting). Restoring cancelled meetings and deleting past occurrences are out of scope.
> **Conflict warning + free-time checker (§4.7–4.8, done):** `/edit` warns before applying a **single-meeting time change** if any participant or the organizer has an overlapping meeting (⚠ list with names + meeting titles; **[Да, применить] / [Изменить время]**, non-blocking per §4.7.3). A new `/checker` bot flow finds common free time: pick participants (directory search) → date range → duration preset → slots when everyone is free (Mon–Fri, 09:00–18:00 Almaty, §4.8.4) or a "no slots" message (§4.8.6). Busyness is read from the internal DB (global-by-email, like §4.6) — external/personal Google events are not seen; bot "create from slot" and series-time-edit conflict checks are out of scope. Also over REST: `POST /workspaces/:id/meetings/conflicts` and `.../free-slots` for the (mocked) Mini App. Core interval math is pure (`domain/meeting.Overlaps`/`FreeSlots`); conflict attribution is in Go (`application.MeetingConflicts`).
> **Employee directory CSV seeding (§1.2/§9.4, done):** on startup the server full-syncs an **embedded** `internal/platform/employeedir/employees.csv` (columns `full_name,email,department`) into every Google-configured workspace (`google_sa_json_enc IS NOT NULL`): rows missing from the CSV are deleted, present rows upserted (`has_telegram` untouched). Pure `employeedir.Parse` is unit-tested; the per-workspace sync is one transaction (`SyncEmployees`). Best-effort (logs `employees_synced` / `employee_seed_failed`, never fatal); an empty CSV is skipped (guard). To change the directory: edit the CSV, rebuild, redeploy. Hot-reload and a bot/admin management UI remain out of scope.
> **Mini App auth (frontend integration #1, done):** the Mini App authenticates Telegram-natively — `POST /api/auth/tma` validates `initData` (HMAC + `auth_date` freshness ≤ 24h; dev mode bypasses HMAC and treats `init_data` as a telegram id) and exchanges it for a short-lived **TMA JWT** (`tg_id`/`email`/`role`, `tok_typ:tma`, 24h) resolved against `bot_users`; unregistered → `401 {code:not_registered}`, infra error → 500. A dedicated `TMAAuth` middleware guards `/api/tma/*` (re-resolves the `bot_users` row per request) and `GET /api/tma/me` returns the identity. Registration stays owned by the bot `/start` flow. Frontend: an auth provider/gate (loading / authed / not_registered / error) replaces the mock current user on the home/profile/meetings screens. Meetings/employees/availability wiring is the next sub-project (still mock-backed).

To be implemented within the existing clean-architecture layout:

- **Google Calendar / Meet adapter** under `backend/internal/infrastructure/` (service-account auth, event CRUD, Meet link generation).
- **Meetings domain** (meeting, recurrence, conflict detection) under `backend/internal/domain/`.
- **Employee directory** seeded from the embedded CSV at deploy.
- New env/secrets (planned): Google service-account credentials, employees CSV path. Add to `deploy/.env.example` and [REQUIREMENTS.md](REQUIREMENTS.md) when implemented.

## Relationship to the platform

This feature is **additive**: the existing notify-bot / scenario engine, native auth, and VCS integration remain unchanged. Meetings reuse the same Go monolith, Postgres (SoT), and Telegram bot wiring.
