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

To be implemented within the existing clean-architecture layout:

- **Google Calendar / Meet adapter** under `backend/internal/infrastructure/` (service-account auth, event CRUD, Meet link generation).
- **Meetings domain** (meeting, recurrence, conflict detection) under `backend/internal/domain/`.
- **Employee directory** seeded from the embedded CSV at deploy.
- New env/secrets (planned): Google service-account credentials, employees CSV path. Add to `deploy/.env.example` and [REQUIREMENTS.md](REQUIREMENTS.md) when implemented.

## Relationship to the platform

This feature is **additive**: the existing notify-bot / scenario engine, native auth, and VCS integration remain unchanged. Meetings reuse the same Go monolith, Postgres (SoT), and Telegram bot wiring.
