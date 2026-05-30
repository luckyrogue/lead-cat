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

To be implemented within the existing clean-architecture layout:

- **Google Calendar / Meet adapter** under `backend/internal/infrastructure/` (service-account auth, event CRUD, Meet link generation).
- **Meetings domain** (meeting, recurrence, conflict detection) under `backend/internal/domain/`.
- **Employee directory** seeded from the embedded CSV at deploy.
- New env/secrets (planned): Google service-account credentials, employees CSV path. Add to `deploy/.env.example` and [REQUIREMENTS.md](REQUIREMENTS.md) when implemented.

## Relationship to the platform

This feature is **additive**: the existing notify-bot / scenario engine, native auth, and VCS integration remain unchanged. Meetings reuse the same Go monolith, Postgres (SoT), and Telegram bot wiring.
