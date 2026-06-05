# Architecture

## Overview

Lead Cat is a **Google Meet meetings-management Telegram Mini App**. Employees schedule, edit, and cancel meetings directly inside Telegram; the system creates Google Calendar events and sends notifications via a Telegram bot.

- **Frontend** — React Telegram Mini App (`frontend/src/features/tma`, `frontend/src/routes/_tma`).
- **Backend** — Go monolith (`cmd/server`): Fiber HTTP server, Telegram bot, asynq workers.
- **Infra** — Postgres (source of truth), Redis (asynq job queues + scheduler leader lock).

---

## Layers

Dependencies point inward: `delivery` / `infrastructure` → `application` → `domain`.

```
backend/internal/
├── domain/          ← pure business entities, no framework deps
│   └── meeting/     ← Meeting, Recurrence, Span, Occurrences, FreeSlots, Overlaps
├── application/     ← orchestration: commands, queries, identity bridge
│   ├── meeting_service.go   (CreateMeeting, UpdateMeeting, CancelMeeting, ListMeetings, GetMeeting)
│   ├── conflict.go          (MeetingConflicts, FreeSlots)
│   ├── series_edit.go       (series-level edits)
│   ├── tma_organizer.go     (EnsureTMAOrganizer)
│   └── services.go          (Services struct; workspace/member/scenario helpers — legacy)
├── delivery/
│   └── http/        ← Fiber handlers, middleware; no business logic
├── infrastructure/
│   ├── persistence/postgres/  ← pgx store (SoT)
│   ├── calendar/              ← Google Calendar API adapter
│   ├── queue/asynq/           ← job enqueue client
│   ├── auth/                  ← OAuth, WebAuthn adapters
│   ├── telegram/              ← initData validator
│   ├── crypto/                ← AES-GCM token cipher
│   └── vcs/                   ← GitHub/GitLab clients (legacy alpha)
└── platform/
    ├── auth/                  ← JWT + TMA token issuers, OTP, session store
    ├── employeedir/           ← CSV employee directory importer
    ├── meeting_notifier/      ← notification worker (created/cancelled/updated)
    ├── meetingedit/           ← bot FSM for editing meetings
    ├── meetingrecipients/     ← resolves notification recipients from participants
    ├── reminder_scheduler/    ← Redis-leader-locked scheduler for reminders
    ├── scheduleview/          ← colleague schedule query helper
    ├── botreg/                ← bot /start FSM (bot_users registration)
    ├── observability/         ← zap logger wiring
    └── scenario_*/            ← legacy alpha scheduler/executor (see Appendix)
```

### Layer responsibilities

| Layer | Responsibility |
|---|---|
| `domain/meeting` | Value types, recurrence maths, overlap/free-slot logic. No I/O. |
| `application` | Orchestrates domain + infrastructure. Commands mutate state (CreateMeeting, UpdateMeeting, CancelMeeting). Queries read state (MeetingConflicts, FreeSlots, EmployeeSchedule). `EnsureTMAOrganizer` bridges identities. |
| `delivery/http` | Maps HTTP ↔ application calls. Two handler groups: platform (`/api/*`) and TMA (`/api/tma/*`). |
| `infrastructure` | Implements ports used by `application`: Postgres store, Google Calendar adapter, asynq queue client. |
| `platform` | Cross-cutting runtime concerns: auth token issuers, bot FSMs, notification workers, reminder scheduler, observability. |

---

## Identity

Two identity worlds coexist:

| World | Table | Key | Created by |
|---|---|---|---|
| **Bot users** | `bot_users` | `telegram_id` + corporate email + role | Telegram bot `/start` FSM (`platform/botreg`) |
| **Platform users** | `platform_users` | UUID | OTP / passkey / OAuth login; or lazily by `EnsureTMAOrganizer` |

**TMA JWT** (`tok_typ: "tma"`, 24 h) — issued by `POST /api/auth/tma` after validating Telegram `initData`. Claims carry the `bot_user` identity.

**`EnsureTMAOrganizer`** (`application/tma_organizer.go`) — bridges the two worlds at meeting-write time. It find-or-creates a `platform_users` row keyed by `auth_sub = "email:<email>"` (same sub as native email-OTP login, so a meeting created via Mini App is editable by a web login and vice-versa), then links the Telegram ID. Returns the `platform_users` UUID used as `organizer_user_id` on the meeting row. Idempotent. Returns `ErrTelegramLinkedToOtherAccount` (→ HTTP 409) if the Telegram ID is already bound to a different account.

---

## Request flow

```
Mini App (Telegram WebApp)
  │
  │  POST /api/auth/tma   (initData in body)
  ▼
TMAAuth endpoint  →  validates initData (HMAC)  →  issues TMA JWT (24 h)
  │
  │  GET/POST /api/tma/*  Authorization: Bearer <tma_jwt>
  ▼
TMAAuth middleware  →  resolves bot_user from JWT claims  →  sets c.Locals
  │
  ▼
Handler (delivery/http)
  │  maps request → input struct
  ▼
application.Services  (EnsureTMAOrganizer if write, then meeting command/query)
  │
  ├──▶ infrastructure/persistence/postgres  (meetings, participants, employees, workspaces)
  └──▶ infrastructure/calendar              (Google Calendar event create/update/delete)
```

**TMA route group** (`/api/tma/*`):

| Method | Path | Application call |
|---|---|---|
| `GET` | `/api/tma/me` | — (reads `bot_user` from TMAAuth middleware locals) |
| `GET` | `/api/tma/meetings` | `TMAMyMeetings` |
| `GET` | `/api/tma/schedule` | `TMASchedule` (colleague schedule) |
| `GET` | `/api/tma/employees` | `ListEmployees` |
| `POST` | `/api/tma/free-slots` | `FreeSlots` |
| `POST` | `/api/tma/meetings` | `CreateMeeting` (via `EnsureTMAOrganizer`) |

---

## Async

```
application layer
  └─ enqueueCreated / enqueueCancelled / enqueueUpdated
       │   (asynq client → Redis)
       ▼
asynq worker process (same binary)
  └─ platform/meeting_notifier  →  Telegram bot send
```

`platform/reminder_scheduler` runs a Redis-leader-locked tick loop so only one replica fires reminders when the binary is scaled.

---

## Appendix — Deprecated: alpha setup (curl)

> These platform endpoints/flows exist only for alpha operator bootstrap and are
> being replaced by in-Mini-App admin (`/api/tma/admin/*`, see
> `docs/superpowers/specs/2026-06-05-tma-setup-replacement-design.md`). Not part
> of the product; slated for removal.

The platform JWT (`platform_users`, issued via OTP/passkey/OAuth) and the `/api/workspaces/*` group still exist to let an operator bootstrap the system: create a workspace, upload Google Service Account credentials (`PATCH .../integrations`), import employees via CSV, and manage members. These flows have no Mini App UI and are operated with curl or scripts. They are being superseded by a TMA admin surface.
