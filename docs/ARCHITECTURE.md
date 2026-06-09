# Architecture

## Overview

Lead Cat is a **Google Meet meetings-management Telegram Mini App**. Employees schedule, edit, and cancel meetings directly inside Telegram; the system creates Google Calendar events and sends notifications via a Telegram bot.

- **Frontend** — React Telegram Mini App (`frontend/src/shared/miniapp`, `frontend/src/routes/_miniapp`).
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
│   ├── organizer_bridge.go  (EnsureMiniAppOrganizer)
│   ├── query/               (Mini App read-model assembly; CQRS read side)
│   └── services.go          (Services facade; workspace/member helpers)
├── delivery/
│   └── http/        ← Fiber handlers, middleware; no business logic
├── infrastructure/
│   ├── persistence/postgres/  ← pgx store (SoT)
│   ├── calendar/              ← Google Calendar API adapter
│   ├── queue/asynq/           ← job enqueue client
│   ├── telegram/              ← initData validator
│   └── crypto/                ← AES-GCM token cipher
└── platform/
    ├── auth/                  ← Mini App JWT issuer
    ├── employeedir/           ← CSV employee directory importer
    ├── meeting_notifier/      ← notification worker (created/cancelled/updated)
    ├── meetingedit/           ← bot FSM for editing meetings
    ├── meetingrecipients/     ← resolves notification recipients from participants
    ├── reminder_scheduler/    ← Redis-leader-locked scheduler for reminders
    ├── scheduleview/          ← colleague schedule query helper
    ├── botreg/                ← bot /start FSM (bot_users registration)
    ├── observability/         ← zap logger wiring
```

### Layer responsibilities

| Layer            | Responsibility                                                                                                                                                                                                          |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `domain/meeting` | Value types, recurrence maths, overlap/free-slot logic. No I/O.                                                                                                                                                         |
| `application`    | Orchestrates domain + infrastructure. Commands mutate state (CreateMeeting, UpdateMeeting, CancelMeeting). Queries read state (MeetingConflicts, FreeSlots, EmployeeSchedule). `EnsureMiniAppOrganizer` bridges identities. |
| `delivery/http`  | Maps HTTP ↔ application calls. Public health + `POST /api/auth/miniapp`; meetings product on `/api/miniapp/*` (+ `/api/miniapp/admin/*` for operators). Retired platform routes return 410.                               |
| `infrastructure` | Implements ports used by `application`: Postgres store, Google Calendar adapter, asynq queue client.                                                                                                                    |
| `platform`       | Cross-cutting runtime concerns: auth token issuers, bot FSMs, notification workers, reminder scheduler, observability.                                                                                                  |

---

## Identity

Two tables cooperate; only one is user-facing auth:

| World              | Table            | Key                                    | Created by                                                     |
| ------------------ | ---------------- | -------------------------------------- | -------------------------------------------------------------- |
| **Bot users**      | `bot_users`      | `telegram_id` + corporate email + role | Telegram bot `/start` FSM (`platform/botreg`)                  |
| **Platform users** | `platform_users` | UUID                                   | Lazily created by `EnsureMiniAppOrganizer` (organizer bridge) |

**Mini App JWT** (`tok_typ: "miniapp"`, 24 h) — issued by `POST /api/auth/miniapp` after validating Telegram `initData`. Claims carry the `bot_user` identity.

**`EnsureMiniAppOrganizer`** (`application/organizer_bridge.go`) — internal bridge at meeting-write time. Find-or-creates a `platform_users` row keyed by `auth_sub = "email:<email>"`, links the Telegram ID, returns the UUID used as `organizer_user_id`. Idempotent.

---

## Request flow

```
Mini App (Telegram WebApp)
  │
  │  POST /api/auth/miniapp   (initData in body)
  ▼
MiniAppAuth endpoint  →  validates initData (HMAC)  →  issues Mini App JWT (24 h)
  │
  │  GET/POST /api/miniapp/*  Authorization: Bearer <miniapp_jwt>
  ▼
MiniAppAuth middleware  →  resolves bot_user from JWT claims  →  sets c.Locals
  │
  ▼
Handler (delivery/http)
  │  maps request → input struct
  ▼
application.Services  (EnsureMiniAppOrganizer if write, then meeting command/query)
  │
  ├──▶ infrastructure/persistence/postgres  (meetings, participants, employees, workspaces)
  └──▶ infrastructure/calendar              (Google Calendar event create/update/delete)
```

**Mini App route group** (`/api/miniapp/*`):

| Method | Path                      | Application call                                        |
| ------ | ------------------------- | ------------------------------------------------------- |
| `GET`  | `/api/miniapp/me`         | — (reads `bot_user` from MiniAppAuth middleware locals) |
| `GET`  | `/api/miniapp/meetings`   | `MiniAppMyMeetings`                                     |
| `GET`  | `/api/miniapp/schedule`   | `MiniAppSchedule` (colleague schedule)                  |
| `GET`  | `/api/miniapp/employees`  | `ListEmployees`                                         |
| `POST` | `/api/miniapp/free-slots` | `FreeSlots`                                             |
| `POST` | `/api/miniapp/meetings`   | `CreateMeeting` (via `EnsureMiniAppOrganizer`)          |

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

## Appendix — Retired platform bootstrap

Platform JWT routes (`/api/auth/email/*`, passkey, OAuth) and `/api/workspaces/*` return **410 Gone**. Operator setup runs through Mini App admin (`/api/miniapp/admin/*`) inside Telegram — see `docs/SETUP.md` and `docs/API.md`.
