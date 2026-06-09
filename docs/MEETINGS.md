# Meetings (Google Meet) — feature

A Telegram Mini App feature for scheduling and managing **Google Meet** meetings inside an organization, layered on top of the existing Lead Cat platform. The full specification (the source of truth for behavior) is **[NEW-FEATURES.md](NEW-FEATURES.md)** (ТЗ). This page is the engineering summary and status.

> **Status:** TMA auth, all read paths, all write paths (incl. recurring series), and admin setup are live. Create, edit, delete, and conflict warning go through `/api/tma/*` end-to-end. Recurring meetings support daily, weekly, custom-weekdays, and monthly kinds with a required end date; edit/cancel are scope-aware (`this` single occurrence vs `whole` series). Admin setup (Google integration, chat link, members sync, scenarios) is live under `/api/tma/admin/*`. Frontend still uses mock fixtures in a few places; see `frontend/README.md` for layout.

## Concept (per ТЗ)

- All meetings are created through **one corporate Google service account** — the organizer of every Google Calendar / Meet event.
- Users are bound by **Telegram ID + corporate email**; a user record is created automatically on `/start`.
- The employee directory (for adding participants) is loaded from a **CSV embedded at deploy time**.
- Transcription/post-processing is handled by an **external service** — out of scope for this bot.
- Base timezone: **UTC+5 (Almaty)**.
- Roles: **User** (auto on `/start`) and **Main Administrator** (full access to all meetings).

## Frontend (implemented)

Telegram Mini App under `frontend/src/routes/_tma/` + feature slices + `components/tma-shell/`. Five tabs (`TabKey`):

| Tab        | Route / page                          | Purpose                                |
| ---------- | ------------------------------------- | -------------------------------------- |
| `home`     | `/` → `features/home/pages/home-page` | Overview / quick actions               |
| `meetings` | `/meetings` → `meetings-list-page`    | List & view meetings (detail-as-sheet) |
| `checker`  | `/checker` → `checker-page`           | Common free-slot finder                |
| `auto`     | `/auto` → `auto-page` (not in TabBar) | Automation rules (mock fixtures)       |
| `profile`  | `/profile` → `profile-page`           | User profile & settings                |

Meeting creation: `/meetings/create` → `features/meeting-create/pages/create-page.tsx`. Cat design: [DESIGN-CATS.md](DESIGN-CATS.md). i18n: `shared/tma/i18n.ts` (`ru` / `kk` / `en`).

### Data model

- **`entities/employee/types`** — `Employee`
- **`entities/meeting/types`** — `Meeting`, `MeetingDraft`, `FreeSlot`
- OpenAPI DTOs: `shared/api/generated/schema.ts` (`TmaMeeting`, `TmaEmployee`, …)

## Functional areas (per ТЗ §4–5)

Create meeting (fields, meeting types, recurrence, naming standard), view meetings, manage participants (add/remove), edit (incl. recurring series), delete/cancel, employee schedule view, **time-conflict warning** on create/edit, **common free-time checker**, and **notifications** (reminders + Google Meet link). See [NEW-FEATURES.md](NEW-FEATURES.md) for exact rules.

## Backend (partial)

### Auth & read paths (done)

| Route                      | Status |
| -------------------------- | ------ |
| `POST /api/auth/tma`       | Done   |
| `GET /api/tma/me`          | Done   |
| `GET /api/tma/meetings`    | Done   |
| `GET /api/tma/schedule`    | Done   |
| `GET /api/tma/employees`   | Done   |
| `POST /api/tma/free-slots` | Done   |

### Write paths

| Route                                       | Status                                                  |
| ------------------------------------------- | ------------------------------------------------------- |
| `POST /api/tma/meetings`                    | Done (incl. recurring; `recurrence_until` / `_days`)    |
| `PATCH /api/tma/meetings/:id?scope=this`    | Done (organizer-only, 403) — single occurrence          |
| `PATCH /api/tma/meetings/:id?scope=whole`   | Done (organizer-only, 403) — entire series              |
| `DELETE /api/tma/meetings/:id?scope=this`   | Done (organizer-only, 403) — single occurrence          |
| `DELETE /api/tma/meetings/:id?scope=whole`  | Done (organizer-only, 403) — entire series              |
| `POST /api/tma/conflicts`                   | Done (occurrence-grouped response; series-aware)        |

Recurrence kinds: `once`, `daily`, `weekly`, `custom` (with `recurrence_days: [1..7]`, Mon=1..Sun=7), `monthly`. Non-once requires `recurrence_until` (YYYY-MM-DD).

### Setup cutover (done)

Admin setup (Google integration, chat link, members sync, scenarios toggle) is live in the Mini App under `/api/tma/admin/*`. Audit log at `GET /api/tma/admin/audit`. See [`docs/superpowers/specs/2026-06-09-slice-d-tma-admin-integrations-design.md`](superpowers/specs/2026-06-09-slice-d-tma-admin-integrations-design.md).

See [API.md](API.md) and [ARCHITECTURE.md](ARCHITECTURE.md).
