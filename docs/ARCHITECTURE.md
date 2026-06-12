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
│   ├── command/             (meeting write commands: create, update, cancel)
│   ├── meeting_service.go   (thin facade → command; list/get meetings)
│   ├── conflict.go          (MeetingConflicts, FreeSlots)
│   ├── series_edit.go       (series-level edits)
│   ├── organizer_bridge.go  (EnsureMiniAppOrganizer)
│   ├── miniapp_org.go       (ResolveMiniAppOrganization — interim default org)
│   ├── query/               (Mini App read-model assembly; CQRS read side)
│   └── services.go          (Services facade; org/member/web-auth helpers)
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
| `application`    | Orchestrates domain + infrastructure. **Commands** (`command/`) mutate meetings. **Queries** (`query/`) assemble read models. `EnsureMiniAppOrganizer` / `UpsertWebIdentity` bridge `platform_users`. Web sessions + org CRUD live on `Services`. |
| `delivery/http`  | Maps HTTP ↔ application. Active: `POST /api/auth/miniapp`, `/api/auth/web/*`, `/api/orgs/*`, `/api/miniapp/*`, `/api/miniapp/admin/*`. Legacy `/api/workspaces/*` and old platform `/api/auth/*` return 410.                               |
| `infrastructure` | Implements ports used by `application`: Postgres store, Google Calendar adapter, asynq queue client.                                                                                                                    |
| `platform`       | Cross-cutting runtime concerns: auth token issuers, bot FSMs, notification workers, reminder scheduler, observability.                                                                                                  |

---

## Identity

| World              | Table            | Key                                    | Created by                                                     |
| ------------------ | ---------------- | -------------------------------------- | -------------------------------------------------------------- |
| **Bot users**      | `bot_users`      | `telegram_id` + corporate email + role | Telegram bot `/start` FSM (`platform/botreg`)                  |
| **Platform users** | `platform_users` | UUID + `auth_sub`                      | Web sign-in or `EnsureMiniAppOrganizer` (same `email:<addr>` sub) |
| **Web sessions**   | `web_sessions`   | hashed cookie token                    | `/api/auth/web/*` success                                      |

**Mini App JWT** (`tok_typ: "miniapp"`, 24 h) — `POST /api/auth/miniapp` → `/api/miniapp/*`.

**Web cookie session** — `/api/auth/web/*` → `web_sessions` → `/api/orgs/*` (org membership via `organization_members`).

**`EnsureMiniAppOrganizer`** — TMA meeting-write bridge; merges with web identity by email (`auth_sub`).

**TMA org (interim):** `ResolveMiniAppOrganization` pins writes to the default org with Google configured; web uses per-user org list + `RequireOrgMember`.

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
  ├──▶ infrastructure/persistence/postgres  (meetings, participants, employees, organizations)
  └──▶ infrastructure/calendar              (Google Calendar event create/update/delete)
```

**Web route group** (`/api/orgs/:id/*`):

```
Browser  →  WebAuth middleware (lc_session cookie)
        →  RequireOrgMember
        →  application.Services.CreateMeeting / …  (same command layer as TMA)
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

Legacy platform JWT routes (`/api/auth/email/*`, passkey, old OAuth) and `/api/workspaces/*` return **410 Gone**. Active operator paths: **web** (`/api/auth/web/*` + `/api/orgs/*`) and **TMA admin** (`/api/miniapp/admin/organization`, deprecated alias `/admin/workspace`) — see `docs/SETUP.md` and `docs/API.md`.
