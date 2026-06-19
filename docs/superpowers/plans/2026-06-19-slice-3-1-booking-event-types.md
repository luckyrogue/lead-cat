# Slice 3-1 — Booking Event-Type Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A host can create/edit/activate/delete booking event types (title, duration, weekday+time availability, timezone) and see each one's shareable `/book/:slug` link.

**Architecture:** A `booking_event_types` table + repo; web-session CRUD under `/api/booking/event-types` (host owns their own; `X-Org-Id` header gives the org); a new admin "Booking" page. The public page + slots (3-2) and submission (3-3) are deferred — the link is inert until 3-2.

**Tech Stack:** Go 1.26 (Fiber, pgx, goose, testcontainers), React Router v7 / shadcn / TanStack Query admin SPA.

## Global Constraints

- Backend at `apps/backend`; run Go as `env -u GOROOT go ...`. Spec: `docs/superpowers/specs/2026-06-19-slice-3-1-booking-event-types-design.md`.
- depguard: `application` imports zero `internal/infrastructure`; sentinels in `internal/application/model`. No code comments in new Go/TS files.
- Booking endpoints: **web-session auth** (`webAuth.Middleware`); host = `c.Locals("web_user").(model.PlatformUser).ID`; org id from the `X-Org-Id` header (`c.Get("X-Org-Id")`) — the admin client already sends it. Create verifies the host is a member of that org (`GetOrgMember`). List/update/delete are scoped by `host_user_id` (caller owns them).
- Frontend: files ≤300 lines, no emoji (lucide only), no comments; i18n en/ru/kk (admin **formal**), parity compile-enforced; never repo-wide prettier (additive edits); pnpm filter `admin`.
- gofmt all Go; `golangci-lint run --config ../../config/.golangci.yml ./...` = 0 issues; `go test -race ./...` green (testcontainers run when Docker present — Docker IS up in this environment now).
- Work on `main`; never `git add -A`; **verify HEAD before each commit** (user commits in parallel — commit on top, never rebase/reset). Trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

**Reference (verified):**
- `gen_random_uuid()` available. Slug pattern: `slugify(title) + "-" + 6-hex` (see `application/org.go` `slugify` + `services.go` org-slug generation — reuse `slugify`).
- `platform_users` has `timezone` (`GetPlatformUserByID` reads it). Org has `tz`.
- Handler identity `c.Locals("web_user").(model.PlatformUser)`. `X-Org-Id` header read via `c.Get("X-Org-Id")`. `GetOrgMember(ctx, orgID, userID) (model.Member, bool, error)` on the Repository (added in 2a-2).
- Admin: `routes.ts` nests authed pages under `_app.tsx`; sidebar nav uses `nav.*` i18n keys (check the sidebar component for how nav items are declared). Entity/query/i18n conventions per prior slices; api client axios `{ data }` + sends `X-Org-Id`. `appOrigin` for building the public link = `window.location.origin`.

---

### Task 1: Migration + repo + model

**Files:**
- Create: `apps/backend/migrations/20260619130000_booking_event_types.sql`
- Create: `apps/backend/internal/application/model/booking.go`
- Modify: `apps/backend/internal/application/model/errors.go` — add `ErrForbidden`
- Create: `apps/backend/internal/infrastructure/persistence/postgres/booking_repo.go`
- Test: `apps/backend/internal/infrastructure/persistence/postgres/booking_repo_test.go`

**Interfaces:**
- Produces:
  - `model.BookingEventType{ ID, HostUserID, OrganizationID uuid.UUID; Slug, Title, Description string; DurationMins int; Active bool; Timezone string; AvailWeekdays []int; AvailStartMinute, AvailEndMinute int; CreatedAt, UpdatedAt time.Time }` (json snake_case).
  - `model.ErrForbidden`.
  - `(*Store).CreateBookingEventType(ctx, et model.BookingEventType) (model.BookingEventType, error)`
  - `(*Store).ListBookingEventTypesForUser(ctx, hostUserID uuid.UUID) ([]model.BookingEventType, error)`
  - `(*Store).GetBookingEventType(ctx, id uuid.UUID) (model.BookingEventType, error)` — IsNotFound when absent
  - `(*Store).GetBookingEventTypeBySlug(ctx, slug string) (model.BookingEventType, error)`
  - `(*Store).UpdateBookingEventType(ctx, et model.BookingEventType) error`
  - `(*Store).DeleteBookingEventType(ctx, id uuid.UUID) error`

- [ ] **Step 1: Migration** — `20260619130000_booking_event_types.sql`:
```sql
-- +goose Up
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
    avail_weekdays     INT[] NOT NULL DEFAULT '{1,2,3,4,5}',
    avail_start_minute INT  NOT NULL DEFAULT 540,
    avail_end_minute   INT  NOT NULL DEFAULT 1020,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX booking_event_types_host_idx ON booking_event_types (host_user_id);

-- +goose Down
DROP TABLE booking_event_types;
```

- [ ] **Step 2: Model + error** — `model/booking.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type BookingEventType struct {
	ID               uuid.UUID `json:"id"`
	HostUserID       uuid.UUID `json:"host_user_id"`
	OrganizationID   uuid.UUID `json:"organization_id"`
	Slug             string    `json:"slug"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	DurationMins     int       `json:"duration_mins"`
	Active           bool      `json:"active"`
	Timezone         string    `json:"timezone"`
	AvailWeekdays    []int     `json:"avail_weekdays"`
	AvailStartMinute int       `json:"avail_start_minute"`
	AvailEndMinute   int       `json:"avail_end_minute"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
```
Append to `model/errors.go`: `var ErrForbidden = errors.New("forbidden")`. Add `postgres.BookingEventType` alias if the package aliases model types.

- [ ] **Step 3: Failing repo test** — `booking_repo_test.go` (`package postgres_test`, reuse `newStore()`/`testDB.Truncate(t)` + `seedOrg`/`seedUser`):
```go
func TestBookingEventType_CRUD(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, _ := seedOrg(t, store)
	hostID := seedUser(t, store, "host@x.com")

	et := pg.BookingEventType{
		HostUserID: hostID, OrganizationID: orgID, Slug: "intro-abc123",
		Title: "Intro call", DurationMins: 30, Active: true, Timezone: "Asia/Almaty",
		AvailWeekdays: []int{1, 2, 3, 4, 5}, AvailStartMinute: 540, AvailEndMinute: 1020,
	}
	created, err := store.CreateBookingEventType(ctx, et)
	if err != nil || created.ID == uuid.Nil {
		t.Fatalf("create: %v %+v", err, created)
	}
	got, err := store.GetBookingEventTypeBySlug(ctx, "intro-abc123")
	if err != nil || got.Title != "Intro call" || len(got.AvailWeekdays) != 5 {
		t.Fatalf("by slug: %v %+v", err, got)
	}
	list, err := store.ListBookingEventTypesForUser(ctx, hostID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	created.Title = "Intro (30m)"
	created.AvailEndMinute = 1080
	if err := store.UpdateBookingEventType(ctx, created); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = store.GetBookingEventType(ctx, created.ID)
	if got.Title != "Intro (30m)" || got.AvailEndMinute != 1080 {
		t.Fatalf("update not applied: %+v", got)
	}
	if err := store.DeleteBookingEventType(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.GetBookingEventType(ctx, created.ID); !model.IsNotFound(err) {
		t.Fatalf("expected IsNotFound after delete, got %v", err)
	}
}
```

- [ ] **Step 4: Run; expect FAIL** — `env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestBookingEventType -v`

- [ ] **Step 5: Implement** — `booking_repo.go`. Use `github.com/jackc/pgx/v5/pgtype` or plain `[]int` scanning for the `INT[]` column — pgx v5 scans `int[]` into `[]int32`/`[]int` via the array codec; use `[]int32` in scan then convert, OR rely on pgx's native `[]int` support. Safest: scan into `[]int32` and convert, write with `$n` binding a `[]int`. Concretely:
```go
package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) CreateBookingEventType(ctx context.Context, et model.BookingEventType) (model.BookingEventType, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO booking_event_types
			(host_user_id, organization_id, slug, title, description, duration_mins, active,
			 timezone, avail_weekdays, avail_start_minute, avail_end_minute)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at, updated_at`,
		et.HostUserID, et.OrganizationID, et.Slug, et.Title, et.Description, et.DurationMins,
		et.Active, et.Timezone, et.AvailWeekdays, et.AvailStartMinute, et.AvailEndMinute).
		Scan(&et.ID, &et.CreatedAt, &et.UpdatedAt)
	return et, err
}

func scanBookingRow(row rowScanner) (model.BookingEventType, error) {
	var et model.BookingEventType
	var weekdays []int32
	if err := row.Scan(&et.ID, &et.HostUserID, &et.OrganizationID, &et.Slug, &et.Title,
		&et.Description, &et.DurationMins, &et.Active, &et.Timezone, &weekdays,
		&et.AvailStartMinute, &et.AvailEndMinute, &et.CreatedAt, &et.UpdatedAt); err != nil {
		return model.BookingEventType{}, err
	}
	et.AvailWeekdays = make([]int, len(weekdays))
	for i, w := range weekdays {
		et.AvailWeekdays[i] = int(w)
	}
	return et, nil
}

const bookingCols = `id, host_user_id, organization_id, slug, title, description, duration_mins,
	active, timezone, avail_weekdays, avail_start_minute, avail_end_minute, created_at, updated_at`

func (s *Store) GetBookingEventType(ctx context.Context, id uuid.UUID) (model.BookingEventType, error) {
	return scanBookingRow(s.pool.QueryRow(ctx, `SELECT `+bookingCols+` FROM booking_event_types WHERE id = $1`, id))
}

func (s *Store) GetBookingEventTypeBySlug(ctx context.Context, slug string) (model.BookingEventType, error) {
	return scanBookingRow(s.pool.QueryRow(ctx, `SELECT `+bookingCols+` FROM booking_event_types WHERE slug = $1`, slug))
}

func (s *Store) ListBookingEventTypesForUser(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+bookingCols+` FROM booking_event_types WHERE host_user_id = $1 ORDER BY created_at DESC`, hostUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.BookingEventType{}
	for rows.Next() {
		et, err := scanBookingRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, et)
	}
	return out, rows.Err()
}

func (s *Store) UpdateBookingEventType(ctx context.Context, et model.BookingEventType) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE booking_event_types SET
			title=$2, description=$3, duration_mins=$4, active=$5, timezone=$6,
			avail_weekdays=$7, avail_start_minute=$8, avail_end_minute=$9, updated_at=now()
		WHERE id=$1`,
		et.ID, et.Title, et.Description, et.DurationMins, et.Active, et.Timezone,
		et.AvailWeekdays, et.AvailStartMinute, et.AvailEndMinute)
	return err
}

func (s *Store) DeleteBookingEventType(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM booking_event_types WHERE id = $1`, id)
	return err
}
```
(`rowScanner` already exists in this package from the 2a-1 invite repo. If pgx rejects writing `[]int` for the `INT[]` param, write `int32` slice — convert before the Exec/QueryRow. Confirm against the running DB in Step 6.)

- [ ] **Step 6: Run; expect PASS** — `env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestBookingEventType -v`

- [ ] **Step 7: gofmt + commit**
```bash
gofmt -w internal/application/model/booking.go internal/application/model/errors.go internal/infrastructure/persistence/postgres/booking_repo.go internal/infrastructure/persistence/postgres/booking_repo_test.go
git add apps/backend/migrations/20260619130000_booking_event_types.sql apps/backend/internal/application/model/booking.go apps/backend/internal/application/model/errors.go apps/backend/internal/infrastructure/persistence/postgres/booking_repo.go apps/backend/internal/infrastructure/persistence/postgres/booking_repo_test.go
# include models.go if aliased
git commit -m "feat(booking): booking_event_types table + repo + model"
```

---

### Task 2: Application CRUD + HTTP endpoints

**Files:**
- Create: `apps/backend/internal/application/booking.go`
- Modify: `apps/backend/internal/application/repository.go` — add the 6 repo methods (+ `GetPlatformUserByID` if absent)
- Create: `apps/backend/internal/delivery/http/handlers/booking.go`
- Modify: `apps/backend/internal/delivery/http/app.go` — register the `/api/booking` group
- Test: `apps/backend/internal/delivery/http/handlers/booking_test.go`

**Interfaces:**
- Produces:
  - `application.EventTypeInput{ Title, Description string; DurationMins int; Timezone string; AvailWeekdays []int; AvailStartMinute, AvailEndMinute int; Active bool }`
  - `(s *Services) CreateEventType(ctx, hostUserID, orgID uuid.UUID, in EventTypeInput) (model.BookingEventType, error)`
  - `(s *Services) ListMyEventTypes(ctx, hostUserID uuid.UUID) ([]model.BookingEventType, error)`
  - `(s *Services) UpdateEventType(ctx, hostUserID, id uuid.UUID, in EventTypeInput) error`
  - `(s *Services) DeleteEventType(ctx, hostUserID, id uuid.UUID) error`
  - Endpoints under `/api/booking/event-types`.

- [ ] **Step 1: Repository port** — add to `application/repository.go`'s `Repository`: the 6 booking methods (Task 1 signatures). Add `GetPlatformUserByID(ctx, id uuid.UUID) (model.PlatformUser, error)` if not already present (check first).

- [ ] **Step 2: Application** — `application/booking.go`:
```go
package application

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type EventTypeInput struct {
	Title            string
	Description      string
	DurationMins     int
	Timezone         string
	AvailWeekdays    []int
	AvailStartMinute int
	AvailEndMinute   int
	Active           bool
}

var ErrInvalidEventType = errors.New("invalid event type")

func validateEventType(in EventTypeInput) error {
	if strings.TrimSpace(in.Title) == "" {
		return ErrInvalidEventType
	}
	if in.DurationMins < 5 || in.DurationMins > 480 {
		return ErrInvalidEventType
	}
	if len(in.AvailWeekdays) == 0 {
		return ErrInvalidEventType
	}
	for _, d := range in.AvailWeekdays {
		if d < 1 || d > 7 {
			return ErrInvalidEventType
		}
	}
	if in.AvailStartMinute < 0 || in.AvailEndMinute > 1440 || in.AvailStartMinute >= in.AvailEndMinute {
		return ErrInvalidEventType
	}
	return nil
}

func (s *Services) CreateEventType(ctx context.Context, hostUserID, orgID uuid.UUID, in EventTypeInput) (model.BookingEventType, error) {
	if err := validateEventType(in); err != nil {
		return model.BookingEventType{}, err
	}
	if _, ok, err := s.Store.GetOrgMember(ctx, orgID, hostUserID); err != nil {
		return model.BookingEventType{}, err
	} else if !ok {
		return model.BookingEventType{}, ErrForbidden
	}
	tz := strings.TrimSpace(in.Timezone)
	if tz == "" {
		if u, err := s.Store.GetPlatformUserByID(ctx, hostUserID); err == nil && u.Timezone != "" {
			tz = u.Timezone
		} else {
			tz = "Asia/Almaty"
		}
	}
	et := model.BookingEventType{
		HostUserID: hostUserID, OrganizationID: orgID,
		Slug:  slugify(in.Title) + "-" + randomSuffix(6),
		Title: in.Title, Description: in.Description, DurationMins: in.DurationMins,
		Active: in.Active, Timezone: tz, AvailWeekdays: in.AvailWeekdays,
		AvailStartMinute: in.AvailStartMinute, AvailEndMinute: in.AvailEndMinute,
	}
	return s.Store.CreateBookingEventType(ctx, et)
}

func (s *Services) ListMyEventTypes(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error) {
	return s.Store.ListBookingEventTypesForUser(ctx, hostUserID)
}

func (s *Services) UpdateEventType(ctx context.Context, hostUserID, id uuid.UUID, in EventTypeInput) error {
	if err := validateEventType(in); err != nil {
		return err
	}
	et, err := s.Store.GetBookingEventType(ctx, id)
	if err != nil {
		return err
	}
	if et.HostUserID != hostUserID {
		return ErrForbidden
	}
	tz := strings.TrimSpace(in.Timezone)
	if tz == "" {
		tz = et.Timezone
	}
	et.Title, et.Description, et.DurationMins = in.Title, in.Description, in.DurationMins
	et.Active, et.Timezone = in.Active, tz
	et.AvailWeekdays, et.AvailStartMinute, et.AvailEndMinute = in.AvailWeekdays, in.AvailStartMinute, in.AvailEndMinute
	return s.Store.UpdateBookingEventType(ctx, et)
}

func (s *Services) DeleteEventType(ctx context.Context, hostUserID, id uuid.UUID) error {
	et, err := s.Store.GetBookingEventType(ctx, id)
	if err != nil {
		return err
	}
	if et.HostUserID != hostUserID {
		return ErrForbidden
	}
	return s.Store.DeleteBookingEventType(ctx, id)
}
```
`ErrForbidden` is `model.ErrForbidden` — reference it as `model.ErrForbidden` (or add an `application` alias `var ErrForbidden = model.ErrForbidden`). Reuse the existing `slugify` (application pkg) + the existing random-suffix helper used for org slugs (find its name in `services.go`/`org.go`; if it's inline, factor a tiny `randomSuffix(n int) string` or reuse the org-slug code path).

- [ ] **Step 3: Handlers** — `handlers/booking.go`: `BookingListEventTypes`, `BookingCreateEventType`, `BookingUpdateEventType`, `BookingDeleteEventType`. Host = `c.Locals("web_user").(model.PlatformUser).ID`. For create, org id from `c.Get("X-Org-Id")` (parse uuid; missing/bad → 400). Map errors: `application.ErrInvalidEventType` → 400, `model.ErrForbidden` → 403, `model.IsNotFound` → 404. Parse `:id` via uuid.Parse. Request body matches `EventTypeInput` json (snake_case): `{title, description, duration_mins, timezone, avail_weekdays, avail_start_minute, avail_end_minute, active}`.

- [ ] **Step 4: Routes** — in `app.go`, add a group: `booking := app.Group("/api/booking", webAuth.Middleware)`; `booking.Get("/event-types", api.BookingListEventTypes)`; `booking.Post("/event-types", api.BookingCreateEventType)`; `booking.Patch("/event-types/:id", api.BookingUpdateEventType)`; `booking.Delete("/event-types/:id", api.BookingDeleteEventType)`.

- [ ] **Step 5: Handler test** — `booking_test.go`: fake `Repository` (embed `application.Repository`, override booking methods + `GetOrgMember` + `GetPlatformUserByID`) + real `Services` + middleware injecting `web_user`; set the `X-Org-Id` header on requests. Assert: list; create with member → 201 + slug present + tz defaulted; create when GetOrgMember false → 403; create bad input (duration 0) → 400; patch by owner → 200; patch of another user's (fake returns a different host) → 403; delete → 204; patch unknown id → 404. (Pure fakes; no Docker. Extend any shared `stubRepo` in calendar/invite/join-request tests with the new interface methods so they still compile.)

- [ ] **Step 6: Run + build/vet/lint** — `env -u GOROOT go test ./internal/delivery/http/... -run Booking -v && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. Green.

- [ ] **Step 7: gofmt + commit**
```bash
gofmt -w internal/application/booking.go internal/application/repository.go internal/delivery/http/handlers/booking.go internal/delivery/http/app.go internal/delivery/http/handlers/booking_test.go
git add apps/backend/internal/application/booking.go apps/backend/internal/application/repository.go apps/backend/internal/delivery/http/handlers/booking.go apps/backend/internal/delivery/http/app.go apps/backend/internal/delivery/http/handlers/booking_test.go
# include any stubRepo test files you extended
git commit -m "feat(booking): event-type CRUD endpoints (validation + ownership + slug)"
```

---

### Task 3: Admin frontend — Booking page

**Files:**
- Create: `apps/admin/app/entities/booking-event-type/{types.ts,api.ts,queries.ts}`
- Create: `apps/admin/app/features/booking/pages/booking-page.tsx` (+ a create/edit dialog component)
- Create: `apps/admin/app/routes/_app.booking._index.tsx` (route shell → `<BookingPage/>`)
- Modify: `apps/admin/app/routes.ts` — add the `booking` route under the `_app` layout
- Modify: the sidebar nav component — add a "Booking" item (find where `members`/`invites`/`meetings` nav items are declared)
- Modify: `apps/admin/app/shared/i18n/dictionaries/{en,ru,kk}.ts` — `nav.booking` + `booking.*`

**Interfaces:** Consumes `/api/booking/event-types` (the admin client auto-sends `X-Org-Id`).

- [ ] **Step 1: entity** — `types.ts` (`BookingEventType` mirroring the backend json), `api.ts` (`listEventTypes`/`createEventType`/`updateEventType`/`deleteEventType` via the axios client), `queries.ts` (`useMyEventTypes` query; create/update/delete mutations invalidating the list key). Mirror `entities/org`.
- [ ] **Step 2: page + dialog** — `booking-page.tsx`: `useMyEventTypes()` list (title, duration, active badge, the public link `${window.location.origin}/book/${slug}` + a copy button, edit/delete). A create/edit dialog (react-hook-form + zod): title, description, duration (select 15/30/45/60), weekday multi-select (Mon..Sun → ints 1..7), start/end time inputs (HH:MM → minutes), timezone (native `<select>` or the settings tz pattern), active toggle. ≤300 lines per file (split the dialog into its own component if needed), no comments, no emoji.
- [ ] **Step 3: route + nav** — add `route("booking", "routes/_app.booking._index.tsx")` under the `_app` layout in `routes.ts`; create the route shell; add a "Booking" sidebar nav item (lucide icon, `t("nav.booking")`) next to meetings.
- [ ] **Step 4: i18n** — `nav.booking` + `booking.{title,description,empty,create,edit,delete,linkCopied,copyLink,active,fields...}` in en/ru/kk (formal RU). Real translations for all keys in all three dicts.
- [ ] **Step 5: Verify + commit**
```bash
pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build
git add apps/admin/app/entities/booking-event-type apps/admin/app/features/booking apps/admin/app/routes/_app.booking._index.tsx apps/admin/app/routes.ts apps/admin/app/shared/i18n/dictionaries
# include the sidebar nav file you modified
git commit -m "feat(admin): booking event-types page + nav + i18n"
```

---

### Task 4: Whole-slice verification

**Files:** none

- [ ] **Step 1: Backend** — `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. Green (booking repo + handler tests run; Docker is up).
- [ ] **Step 2: Frontend** — `pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build`. Green; i18n parity.
- [ ] **Step 3: Tree clean** — verify HEAD; `git status` no stray staged files; user parallel WIP untouched.

---

## Notes for the executor

- **Org id from `X-Org-Id` header** (the admin client sends it); only `Create` needs it (+ a `GetOrgMember` membership check). List/update/delete are scoped by `host_user_id` (caller-owned).
- **Slug immutable** after create; auto-generated `slugify(title)+"-"+6hex`.
- **`INT[]` scanning**: confirm pgx writes/reads the `avail_weekdays` array (use `[]int32` scan + convert if `[]int` doesn't bind); the Step-6 DB run is the real check (Docker is up).
- **`ErrForbidden`** lives in `model`; handlers map it to 403.
- **Deferred:** public `/book/:slug` page + slot computation (3-2); booking submission + meeting creation (3-3). The shareable link is shown but inert until 3-2.
```
