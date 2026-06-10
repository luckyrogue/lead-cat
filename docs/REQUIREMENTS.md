# Requirements — Lead Cat

Single-purpose Google Meet meetings-management **Telegram Mini App**: employees register via `/start`; inside the Mini App they create, edit, and delete meetings, receive conflict warnings, find common free time, view colleague schedules, and get Telegram reminders. Google Meet links are generated via a corporate service account.

This document covers prerequisites/dependencies (what you need to build & run) and functional requirements (what the system must do).

---

## 1. Purpose

Lead Cat is a Telegram-native meetings-management tool for a single organisation (single-tenant deployment). All Google Calendar events are created under one corporate service account. User identity is bound to a Telegram ID + corporate email pair, established automatically when an employee sends `/start` to the bot.

---

## 2. Actors

| Role                   | Description                                                                                                               | Provisioned                                                                   |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| **User**               | Any registered employee. Can create meetings, manage their own meetings, add/remove participants from their own meetings. | Automatically on `/start`                                                     |
| **Main Administrator** | Full access: view/edit/delete any meeting, manage users, assign/revoke admin rights.                                      | Manually by another admin, or first admin set via deploy-time config (TG ID). |

Access matrix summary (from ТЗ §2):

| Action                                | User | Admin |
| ------------------------------------- | :--: | :---: |
| Create meeting                        |  ✅  |  ✅   |
| View own meetings                     |  ✅  |  ✅   |
| View all meetings                     |  ❌  |  ✅   |
| View any colleague's schedule         |  ✅  |  ✅   |
| Edit own meeting                      |  ✅  |  ✅   |
| Edit any meeting                      |  ❌  |  ✅   |
| Delete own meeting                    |  ✅  |  ✅   |
| Delete any meeting                    |  ❌  |  ✅   |
| Add/remove participants (own meeting) |  ✅  |  ✅   |
| Add/remove participants (any meeting) |  ❌  |  ✅   |
| Conflict warning (automatic)          |  ✅  |  ✅   |
| Free-time checker                     |  ✅  |  ✅   |
| Assign admins / view user list        |  ❌  |  ✅   |

---

## 3. Feature set

### 3.1 Registration (§3)

- `/start` triggers registration if the Telegram ID is not yet known: bot collects full name → corporate email → validates uniqueness → creates record with role `user`.
- One Telegram ID maps to one account; one email maps to one Telegram ID. Email changes are admin-only.

### 3.2 Create meeting (§4.1)

**Required fields:** department (list + "Other"), meeting type (list + "Other"), meeting host, date (`DD.MM.YYYY`), start time, end time, recurrence frequency.
**Optional fields:** participants (CSV search or manual email), description.

**Meeting types (default list):** Планёрка, Еженедельная встреча, 1:1, Ретроспектива, Демо/Презентация, Интервью, Онбординг, Брейнштурм, Стратегическая сессия, Другое.

**Recurrence options:** once, daily, weekly, chosen weekdays, monthly. Recurring series requires an end date or "no limit".

**Auto-naming:** system generates `[Department] | [Type] | [Host] | [Date] | [Frequency]`. Users cannot set a custom title.

**On confirm:** event created in Google Calendar via service account → Meet link generated → participants added as attendees → organiser receives confirmation (title, datetime, Meet link, participant list).

### 3.3 View / list meetings (§4.2)

- Chronological list (soonest first); filters: all / upcoming / past.
- Each row: title, datetime, Meet link, participant count, status.
- Tap a meeting → detail card with full participant list and action buttons.
- Users see only their own meetings; admins see all meetings.

### 3.4 Manage participants (§4.3)

- **Add:** search CSV employee directory by name/email substring; fallback to manual email entry. Participant receives Telegram notification (if registered) + Google Calendar invite.
- **Remove:** organiser/admin selects from current participant list; removed participant notified via Telegram + Google Calendar.

### 3.5 Edit meeting (§4.4)

- Editable fields: date/time, department, meeting type, host, description, recurrence (for series).
- Title is regenerated automatically after any edit.
- For recurring series: bot asks "Edit this occurrence only" or "Edit entire series".
- All participants notified of changes (Telegram + Google Calendar update).

### 3.6 Delete / cancel meeting (§4.5)

- For recurring meetings: delete single occurrence or entire series.
- Confirmation required. On confirm: event removed from Google Calendar; all participants notified (Telegram + Google Calendar cancellation).

### 3.7 Colleague schedule view (§4.6)

- Search any employee from the CSV directory.
- Shows their meetings (title, date/time, status) — read-only.
- Day navigation: today, tomorrow, specific date, date range.
- Data source: internal database (meetings created through this bot).

### 3.8 Time-conflict detection (§4.7)

- Triggered automatically during meeting creation and editing, after date/time/participants are set.
- Checks Google Calendar freebusy for all participants including the organiser.
- Any partial or full overlap is a conflict.
- If conflicts found, bot shows warning listing each conflicted participant and the clashing meeting. User may proceed ("Create anyway") or go back to change the time. Non-blocking.

### 3.9 Free-time checker (§4.8)

- User selects participants (CSV search), date range, desired duration (15 min / 30 min / 45 min / 1 h / 1.5 h / 2 h / custom).
- Working hours for search: 09:00–18:00 Almaty (UTC+5) by default.
- Bot queries `freebusy` API for each participant, computes intersection of free windows, filters by minimum duration, returns chronological list of available slots.
- Selecting a slot pre-fills date, start/end time, and participants in the standard create-meeting flow.
- If no slot found: informs user; suggests wider date range, shorter duration, or fewer participants.

### 3.10 Notifications (§5)

| Event                     | Recipients                   | Channel                                            |
| ------------------------- | ---------------------------- | -------------------------------------------------- |
| Meeting created           | Organiser + all participants | Telegram + Google Calendar (email)                 |
| Participant added         | Added participant            | Telegram (if registered) + Google Calendar (email) |
| Participant removed       | Removed participant          | Telegram (if registered) + Google Calendar (email) |
| Meeting edited            | All participants             | Telegram + Google Calendar (email)                 |
| Meeting cancelled/deleted | All participants             | Telegram + Google Calendar (email)                 |
| Reminder before meeting   | All participants             | Telegram (user-configured)                         |

**Reminder settings (§5.2):** user configures in Settings — intervals: 10 min / 15 min / 30 min / 1 h / 2 h / 1 day; multiple intervals allowed; can be fully disabled. Applied globally to all meetings.

### 3.11 Admin panel (§6)

- View all meetings with filters (date, department, organiser).
- Edit and delete any meeting.
- View registered user list (TG ID, full name, email, role, registration date).
- Assign / revoke admin rights (cannot self-demote).
- Correct a user's email.

### 3.12 User settings (§7)

| Setting            | Default                    |
| ------------------ | -------------------------- |
| Timezone           | UTC+5 (Almaty)             |
| Reminder intervals | Off                        |
| Interface language | Russian and Kazakh |

### 3.13 Commands and navigation (§8)

**Bot commands:**

| Command        | Description                                           |
| -------------- | ----------------------------------------------------- |
| `/start`       | Launch bot; register if unknown, else open main menu. |
| `/menu`        | Open main menu.                                       |
| `/new`         | Create a new meeting.                                 |
| `/my_meetings` | View own meetings.                                    |
| `/schedule`    | View a colleague's schedule.                          |
| `/checker`     | Free-time checker.                                    |
| `/settings`    | User settings.                                        |
| `/help`        | Command reference.                                    |
| `/admin`       | Admin panel (admins only).                            |

**Main menu buttons:** Create meeting, My meetings, Colleague schedule, Free-time checker, Settings, Help, Admin panel (admins only).

---

## 4. Prerequisites / stack

| Tool / Service   | Version | Notes                                                               |
| ---------------- | ------- | ------------------------------------------------------------------- |
| Go               | 1.26.x  | `backend/go.mod` pins `go 1.26.3`; toolchain auto-fetches if newer. |
| Node.js          | 22.x    | Frontend build (Vite). Dockerfile uses `node:22-alpine`.            |
| pnpm             | 9.x     | Frontend package manager (`frontend/pnpm-lock.yaml`).               |
| Docker + Compose | recent  | Local Postgres + Redis via `deploy/docker-compose.yml`.             |
| PostgreSQL       | 18      | `postgres:18-alpine` in local compose and CI smoke.                 |
| Redis            | 8       | `redis:8-alpine`; asynq queues and scheduler leader-lock.           |
| golangci-lint    | 2.x     | `make lint` / `make fmt` (config in `config/.golangci.yml`).        |
| air (optional)   | latest  | `make backend-watch` hot reload.                                    |

**Backend:** Go, Fiber (`gofiber/fiber/v2`), asynq, pgx, goose migrations. Clean architecture — `domain` ← `application` ← `infrastructure` / `delivery` / `platform`.

**Frontend:** React 19, Vite 6, TypeScript 5, TanStack Router + Query, shadcn/ui + Tailwind CSS v4, lite Feature-Sliced Design. Telegram Mini App SDK for auth (no separate login screen — identity comes from Telegram `initData`).

**Integrations:** Google Calendar API v3 via a single corporate service account (domain-wide delegation); Google Meet links generated by setting `conferenceData` on Calendar events.

> **Note:** The ТЗ §9 mentions a tentative Python/Node stack; the actual implementation is Go + React.

**One-shot local setup:** `make setup` → edit `.env` → `make migrate` → `make dev`.  
Default ports: API `:8080`, frontend `:3000`, Postgres `5432`, Redis `6379`.

### 4.1 Key environment variables

| Variable                             | Required | Purpose                                                         |
| ------------------------------------ | -------- | --------------------------------------------------------------- |
| `BOT_TOKEN`                          | prod     | Telegram bot token.                                             |
| `DATABASE_URL`                       | yes      | Postgres DSN (source of truth).                                 |
| `REDIS_URL`                          | yes      | Redis DSN for asynq queues.                                     |
| `MASTER_ENCRYPTION_KEY`              | yes      | ≥32 chars; encrypts service-account JSON at rest.               |
| `JWT_SECRET`                         | yes      | ≥16 chars; signs session JWTs.                                  |
| `JWT_ISSUER`, `JWT_TTL_HOURS`        | no       | JWT issuer / lifetime (defaults: `lead-cat`, 168 h).            |
| `AUTH_DEV_MODE`                      | dev      | `true` → any bearer token maps to a dev user.                   |
| `WEBAPP_URL`, `CORS_ALLOWED_ORIGINS` | yes      | Frontend origin(s) for links & CORS.                            |
| `LOG_LEVEL`, `LOG_FORMAT`            | no       | Structured logging (zap).                                       |
| `AUTO_MIGRATE`                       | no       | `true` → run migrations on boot.                                |
| `CALENDAR_STUB`                      | dev/CI   | `true` → use Google Calendar stub (no real credentials needed). |

**Google service-account credentials** are configured per workspace via `PATCH /api/workspaces/:id/integrations` (no env var). Without credentials, meeting creation returns 400.

**Employee directory** is an embedded CSV (`backend/internal/platform/employeedir/employees.csv`), full-synced into Google-configured workspaces on boot. To update the directory, edit the CSV and redeploy — there is no in-app management UI for this.

---

## 5. Out of scope

The following are explicitly excluded (ТЗ §11) or removed from the product:

- **Meeting transcription** — handled by an external service connected to the service-account mailbox.
- **Transcript export to third-party systems** — responsibility of a separate script/bot.
- **CSV employee directory management via the bot UI** — list is updated manually (edit CSV + redeploy).
- **Per-user Google OAuth** — a single corporate service account is used for all Calendar operations.
- **Video recording management** — not controlled by this bot.
- **Multi-tenant SaaS model** — removed from the product; single-organisation deployment only.
- **Scenario / automation engine** — n8n-like scenario builder removed from the product.
- **VCS integration** — GitHub/GitLab commit reporting removed from the product.

---

See also: [ARCHITECTURE.md](ARCHITECTURE.md), [AUTH.md](AUTH.md), [MEETINGS.md](MEETINGS.md), [LOCAL_DEV.md](LOCAL_DEV.md), [DEPLOY-DOKPLOY.md](DEPLOY-DOKPLOY.md), [SETUP.md](SETUP.md).
