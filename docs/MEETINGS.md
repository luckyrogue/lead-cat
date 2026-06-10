# Meetings (Google Meet) — feature

A Telegram Mini App feature for scheduling and managing **Google Meet** meetings inside an organization. The full specification (the source of truth for behavior) is **[NEW-FEATURES.md](NEW-FEATURES.md)** (ТЗ). This page is the engineering summary and status.

> **Status:** Mini App auth, all read paths, all write paths (incl. recurring series), and admin setup are live. Create, edit, delete, and conflict warning go through `/api/miniapp/*` end-to-end. Recurring meetings support daily, weekly, custom-weekdays, and monthly kinds with a required end date; edit/cancel are scope-aware (`this` single occurrence vs `whole` series). Admin setup (Google integration, chat link, members sync, audit log) is live under `/api/miniapp/admin/*`.

## Concept (per ТЗ)

- All meetings are created through **one corporate Google service account** — the organizer of every Google Calendar / Meet event.
- Users are bound by **Telegram ID + corporate email**; a user record is created via bot `/start` (name → email, no OTP).
- The employee directory (for adding participants) is loaded from a **CSV embedded at deploy time**.
- Transcription/post-processing is handled by an **external service** — out of scope for this bot.
- Base timezone: **UTC+5 (Almaty)**.
- Roles: **User** (auto on `/start`) and **Main Administrator** (full access to all meetings).

## Frontend (implemented)

Telegram Mini App under `frontend/src/routes/_miniapp/` + feature slices + `components/miniapp-shell/`. Four tabs (`TabKey`):

| Tab        | Route / page                          | Purpose                                |
| ---------- | ------------------------------------- | -------------------------------------- |
| `home`     | `/` → `features/home/pages/home-page` | Overview / quick actions               |
| `meetings` | `/meetings` → `meetings-list-page`    | List & view meetings (detail-as-sheet) |
| `checker`  | `/checker` → `checker-page`           | Common free-slot finder                |
| `profile`  | `/profile` → `profile-page`           | User profile & settings                |

Meeting creation: `/meetings/create` → `features/meeting-create/pages/create-page.tsx`. Cat design: [DESIGN-CATS.md](DESIGN-CATS.md). i18n: `shared/miniapp/i18n.ts` (`ru` / `kk`).

### Data model

- **`entities/employee/types`** — `Employee`
- **`entities/meeting/types`** — `Meeting`, `MeetingDraft`, `FreeSlot`
- OpenAPI DTOs: `shared/api/generated/schema.ts` (`MiniAppMeeting`, `MiniAppEmployee`, …)

## Functional areas (per ТЗ §4–5)

Create meeting (fields, meeting types, recurrence, naming standard), view meetings, manage participants (add/remove), edit (incl. recurring series), delete/cancel, employee schedule view, **time-conflict warning** on create/edit, **common free-time checker**, and **notifications** (reminders + Google Meet link). See [NEW-FEATURES.md](NEW-FEATURES.md) for exact rules.

## Backend (done)

### Auth & read paths

| Route                           | Status |
| ------------------------------- | ------ |
| `POST /api/auth/miniapp`        | Done   |
| `GET /api/miniapp/me`           | Done   |
| `GET /api/miniapp/meetings`     | Done   |
| `GET /api/miniapp/schedule`     | Done   |
| `GET /api/miniapp/employees`    | Done   |
| `POST /api/miniapp/free-slots`  | Done   |

### Write paths

| Route                                            | Status                                                  |
| ------------------------------------------------ | ------------------------------------------------------- |
| `POST /api/miniapp/meetings`                     | Done (incl. recurring; `recurrence_until` / `_days`)    |
| `PATCH /api/miniapp/meetings/:id?scope=this`     | Done (organizer-only, 403) — single occurrence          |
| `PATCH /api/miniapp/meetings/:id?scope=whole`    | Done (organizer-only, 403) — entire series              |
| `DELETE /api/miniapp/meetings/:id?scope=this`    | Done (organizer-only, 403) — single occurrence          |
| `DELETE /api/miniapp/meetings/:id?scope=whole`   | Done (organizer-only, 403) — entire series              |
| `POST /api/miniapp/conflicts`                    | Done (occurrence-grouped response; series-aware)        |

Recurrence kinds: `once`, `daily`, `weekly`, `custom` (with `recurrence_days: [1..7]`, Mon=1..Sun=7), `monthly`. Non-once requires `recurrence_until` (YYYY-MM-DD).

### Admin setup

Admin setup (Google integration, chat link, members sync, audit log) is live in the Mini App under `/api/miniapp/admin/*`. Audit log at `GET /api/miniapp/admin/audit`.

### User settings (done)

Reminder intervals are user-configurable in the Profile screen, persisted in `bot_users.reminder_minutes`. See [`docs/superpowers/specs/2026-06-09-slice-c-user-settings-design.md`](superpowers/specs/2026-06-09-slice-c-user-settings-design.md). Timezone + language remain Slice H scope.

See [API.md](API.md) and [ARCHITECTURE.md](ARCHITECTURE.md).
