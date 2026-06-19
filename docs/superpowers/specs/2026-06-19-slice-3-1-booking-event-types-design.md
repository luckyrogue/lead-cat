# Slice 3-1 — Booking Event-Type Config (design)

**Date:** 2026-06-19
**Status:** approved — ready for implementation plan
**Part of:** SaaS Product Completion epic, **Track 3 (booking links)**, sub-slice **1** of 3.

## Epic / track context

Track 1 (cross-calendar wedge) and Track 2a (onboarding) are shipped. Track 3 adds
**Calendly-style booking links** backed by the unified-availability engine. Decomposition:
- **3-1 (this)** — event-type config: a host defines/manages booking event types (table + admin CRUD).
- **3-2** — slot-engine refactor (timezone + configurable window) + public `/book/:slug` page (read).
- **3-3** — booking submission: public POST creates the meeting + confirmation.

**Decisions (from brainstorming):** full event-types (multiple per host); availability = a single
daily window across selected weekdays in the host's timezone; slug auto-generated; a dedicated
"Booking" admin nav item; the public page lives in the **admin** app (outside the auth layout).

## Goal

A logged-in host can create, edit, activate/deactivate, and delete **booking event types**
(title, duration, description, weekday+time availability, timezone) and see each one's shareable
`/book/:slug` link. (The link does not resolve to a working page until 3-2.)

## Background — verified current state

- `platform_users` has `id, email, timezone, language` (NO name/handle). Orgs have `slug` + `tz`.
- Slug generation pattern: `slugify(name) + "-" + 6-hex` (`application/org.go` / `services.go`).
  `gen_random_uuid()` is available for table id defaults.
- Web-session auth + the `Repository` interface pattern; handler identity
  `c.Locals("web_user").(model.PlatformUser)` (.ID). Self-service endpoints live under
  `/api/...` with `webAuth.Middleware` (e.g. `/api/auth/web/me/*`, `/api/booking/*` is new).
- Admin routing `apps/admin/app/routes.ts`: authed pages nested under `_app.tsx`; nav items in
  the sidebar (`nav.*` i18n keys). A new `/booking` page nests under `_app`.
- Admin entity/query/i18n conventions established in prior slices (axios `{ data }` client,
  TanStack Query, compile-enforced en/ru/kk parity, admin formal RU).

## Design

### A. Migration + repo

`booking_event_types`:
```sql
CREATE TABLE booking_event_types (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_user_id       UUID NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    organization_id    UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug               TEXT NOT NULL UNIQUE,
    title              TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    duration_mins      INT  NOT NULL,
    active             BOOLEAN NOT NULL DEFAULT true,
    timezone           TEXT NOT NULL DEFAULT '',
    avail_weekdays     INT[] NOT NULL DEFAULT '{1,2,3,4,5}',   -- 1=Mon..7=Sun
    avail_start_minute INT  NOT NULL DEFAULT 540,              -- 09:00
    avail_end_minute   INT  NOT NULL DEFAULT 1020,             -- 17:00
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX booking_event_types_host_idx ON booking_event_types (host_user_id);
```

`model.BookingEventType` (json tags snake_case): `ID, HostUserID, OrganizationID uuid.UUID;
Slug, Title, Description string; DurationMins int; Active bool; Timezone string; AvailWeekdays
[]int; AvailStartMinute, AvailEndMinute int; CreatedAt, UpdatedAt time.Time`.

Repo methods on `*Store`:
- `CreateBookingEventType(ctx, et model.BookingEventType) (model.BookingEventType, error)` — inserts
  (slug provided by the app layer); returns the row.
- `ListBookingEventTypesForUser(ctx, hostUserID uuid.UUID) ([]model.BookingEventType, error)`.
- `GetBookingEventType(ctx, id uuid.UUID) (model.BookingEventType, error)` — `IsNotFound` when absent.
- `GetBookingEventTypeBySlug(ctx, slug string) (model.BookingEventType, error)` — for 3-2; defined now.
- `UpdateBookingEventType(ctx, et model.BookingEventType) error` — updates the mutable fields +
  `updated_at`; scoped by `id` (ownership enforced in the app layer).
- `DeleteBookingEventType(ctx, id uuid.UUID) error`.

### B. Application

- `CreateEventType(ctx, hostUserID, orgID uuid.UUID, in EventTypeInput) (model.BookingEventType, error)`
  — validate (title non-empty, `duration_mins` in a sane range e.g. 5..480, weekdays subset of
  1..7, `0 <= start < end <= 1440`); default `timezone` to the host's `platform_users.timezone`
  (fallback org tz / "Asia/Almaty") when blank; generate a unique slug (`slugify(title)+"-"+6hex`,
  retry on the rare unique collision); persist.
- `ListMyEventTypes(ctx, hostUserID)`.
- `UpdateEventType(ctx, hostUserID, id uuid.UUID, in EventTypeInput) error` — load, verify
  `host_user_id == caller` (else `model.ErrForbidden` → 403), apply, save. Slug is immutable.
- `DeleteEventType(ctx, hostUserID, id uuid.UUID) error` — ownership-checked.
- `EventTypeInput{ Title, Description string; DurationMins int; Timezone string; AvailWeekdays []int;
  AvailStartMinute, AvailEndMinute int; Active bool }`.
- `model.ErrForbidden` sentinel (new, in `model/errors.go`) for ownership violations.
- New `Repository` interface methods for the repo calls above + `GetPlatformUserByID` (if not
  already on the interface — for the host-tz default).

### C. HTTP endpoints (web-session auth)

Under a new `booking` group `app.Group("/api/booking", webAuth.Middleware)`:
- `GET /api/booking/event-types` → the caller's event types.
- `POST /api/booking/event-types` `{title, description, duration_mins, timezone, avail_weekdays,
  avail_start_minute, avail_end_minute, active}` → 201 the created event type. Validation error → 400.
- `PATCH /api/booking/event-types/:id` → 200; not owner → 403; not found → 404.
- `DELETE /api/booking/event-types/:id` → 204; not owner → 403.
- All read/write the caller's own (`web_user.ID` as `host_user_id`). `organization_id` = the
  caller's active org (from a header `X-Org-Id` like the existing org routes, or the caller's first
  org — the plan picks the existing convention; reuse how the app already resolves the active org
  for a web user).

### D. Frontend (admin)

- New nav item **Booking** (`nav.booking` i18n) + route `/booking` under `_app` →
  `features/booking/pages/booking-page.tsx`.
- `entities/booking-event-type/{types,api,queries}`: `useMyEventTypes`, `useCreateEventType`,
  `useUpdateEventType`, `useDeleteEventType`.
- The page lists event types (title, duration, active badge/toggle, the public link
  `${appOrigin}/book/${slug}` with a copy-to-clipboard button) + a create/edit dialog: title,
  description, duration (select: 15/30/45/60), weekday multi-select (Mon..Sun), start/end time
  (time inputs → minutes), timezone (reuse the settings tz picker pattern), active toggle.
- i18n `booking.*` in en/ru/kk (admin formal).

### E. Error handling / validation

- Title required; `duration_mins` 5..480; `avail_weekdays` ⊆ 1..7 and non-empty; `0 ≤
  avail_start_minute < avail_end_minute ≤ 1440`. Invalid → 400.
- Ownership: mutate of another user's event type → 403 (`ErrForbidden`).
- Slug collision on create → regenerate (bounded retries); never surfaced.

## Testing / verification

- **Repo** (testcontainers): create→get/list (host-scoped); update mutable fields; bySlug;
  delete; unique slug. (Run with Docker; skip-clean otherwise.)
- **Application:** create defaults timezone to host tz + generates a slug; update enforces
  ownership (→ ErrForbidden for a non-owner); validation rejects bad duration/weekdays/window.
- **Handlers** (fakes): list/create(201)/patch(200)/delete(204); non-owner→403; bad input→400;
  not-found→404.
- **Frontend:** admin typecheck/lint/build green; i18n parity en/ru/kk.
- `go test -race ./...` + `golangci-lint` clean.

## Risks & mitigations

- **Active-org resolution for a web user.** *Mitigation:* reuse the existing convention the app
  already uses to scope web requests to an org (header / membership); the plan pins the exact
  mechanism.
- **Timezone validity.** Host tz may be `''`. *Mitigation:* default to org tz / Almaty; 3-2's slot
  engine will `LoadLocation` defensively.
- **Slug uniqueness race.** *Mitigation:* unique constraint + regenerate-on-conflict.
- **Scope creep into slots/booking.** *Mitigation:* 3-1 is config-only; the link is inert until
  3-2 — stated in the UI ("preview available after you publish"/just show the link).

## Done criteria

- `booking_event_types` table + repo (incl. `GetEventTypeBySlug` for 3-2) + `model.BookingEventType`.
- App CRUD with validation + ownership (`ErrForbidden`) + host-tz default + slug generation.
- Web-session endpoints `GET/POST /api/booking/event-types` + `PATCH/DELETE /:id`.
- Admin "Booking" nav + page: list/create/edit/delete/activate + copy link; i18n en/ru/kk.
- `-race` + lint green; admin typecheck/lint/build green.
- Public page + slot engine (3-2) and booking submission (3-3) explicitly deferred.
