# Slice 3-2 — Public Booking Page + Slot Availability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `GET /api/book/:slug` (public) returns an event type + its available slots (next 14 days, host's real calendar); a public `/book/:slug` admin page renders the event + day + slots (selection only; submit is 3-3).

**Architecture:** A dedicated `BookingAvailability` application method (mirrors `FreeSlots` but uses the event type's tz/weekdays/window/duration; reuses `gatherExternalBusy` for the host's external calendar) + a public unauthenticated endpoint + a self-contained admin route using plain `fetch`.

**Tech Stack:** Go 1.26 (Fiber), React Router v7 admin SPA.

## Global Constraints

- Backend at `apps/backend`; run Go as `env -u GOROOT go ...`. Spec: `docs/superpowers/specs/2026-06-19-slice-3-2-public-booking-page-design.md`.
- depguard: `application` imports zero `internal/infrastructure`. No code comments in new Go/TS files.
- **Public endpoint = NO auth middleware** (register on `app` like the OAuth callback). Returns only FREE slots — never host email or busy details.
- **Public admin page uses plain `fetch`** (NOT the shared axios client, which injects session/CSRF/X-Org-Id) and NO authed providers (`useMe`/LocaleProvider). Public-page strings: plain English (or a tiny inline dict) — full public i18n deferred.
- Frontend: files ≤300 lines, no emoji (lucide only), no comments; never repo-wide prettier (additive edits); pnpm filter `admin`.
- gofmt all Go; `golangci-lint run --config ../../config/.golangci.yml ./...` = 0 issues; `go test -race ./...` green.
- Work on `main`; never `git add -A`; **verify HEAD before each commit** (user commits in parallel — commit on top, never rebase/reset). Trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

**Reference — `FreeSlots` to mirror (`internal/application/conflict.go`):** builds `busy []meeting.Span` from `Store.ListMeetingsOverlapping(emails, from, to)` (`m.StartsAt`/`m.EndsAt`) ∪ `s.gatherExternalBusy(ctx, requesterEmail, emails, from, to)`; then per-day: `winStart/winEnd := time.Date(y,mo,d, H,0,0,0, loc)`, `dayBusy` = busy overlapping the window (`meeting.Overlaps`), `meeting.FreeSlots(dayBusy, winStart, winEnd, minDur) []meeting.Span`. `meeting.Span{Start,End time.Time}`. Repository has `GetBookingEventTypeBySlug`, `GetPlatformUserByID(id)(model.PlatformUser,bool,error)`, `GetOrganization(id)(model.Organization,error)`, `ListMeetingsOverlapping`.

**Go weekday → ISO (Mon=1..Sun=7):** `iso := int(t.Weekday()); if iso == 0 { iso = 7 }` (Go Sunday=0).

---

### Task 1: `BookingAvailability` + `PublicBooking` + public endpoint

**Files:**
- Create: `apps/backend/internal/application/booking_availability.go`
- Modify: `apps/backend/internal/application/booking.go` (or new file) — `PublicBooking` + `BookingView`/`BookingSlot` types
- Create: `apps/backend/internal/delivery/http/handlers/public_booking.go`
- Modify: `apps/backend/internal/delivery/http/app.go` — register `GET /api/book/:slug` (no middleware)
- Test: `apps/backend/internal/application/booking_availability_test.go`, `apps/backend/internal/delivery/http/handlers/public_booking_test.go`

**Interfaces:**
- Produces:
  - `application.BookingSlot{ Start, End time.Time }` (json `start`/`end` RFC3339).
  - `application.BookingView{ Event BookingEventView; Slots []BookingSlot }`;
    `BookingEventView{ Title, Description string; DurationMins int; OrgName, Timezone string }`
    (json `title`/`description`/`duration_mins`/`org_name`/`timezone`; the view wraps as
    `{"event":{…},"slots":[…]}`).
  - `(s *Services) BookingAvailability(ctx, et model.BookingEventType, from, to time.Time) ([]BookingSlot, error)`
  - `(s *Services) PublicBooking(ctx, slug string, now time.Time) (BookingView, error)` — `IsNotFound` for unknown/inactive.

- [ ] **Step 1: `BookingAvailability`** — `booking_availability.go`:
```go
package application

import (
	"context"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

type BookingSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func loadLoc(name string) *time.Location {
	if name == "" {
		return almatyLoc
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return almatyLoc
}

func weekdaySet(days []int) map[int]bool {
	m := make(map[int]bool, len(days))
	for _, d := range days {
		m[d] = true
	}
	return m
}

func (s *Services) BookingAvailability(ctx context.Context, et model.BookingEventType, from, to time.Time) ([]BookingSlot, error) {
	if et.DurationMins <= 0 {
		return nil, nil
	}
	host, ok, err := s.Store.GetPlatformUserByID(ctx, et.HostUserID)
	if err != nil || !ok || host.Email == "" {
		return nil, err
	}
	loc := loadLoc(et.Timezone)
	allowed := weekdaySet(et.AvailWeekdays)
	dur := time.Duration(et.DurationMins) * time.Minute

	ms, err := s.Store.ListMeetingsOverlapping(ctx, []string{host.Email}, from, to)
	if err != nil {
		return nil, err
	}
	busy := make([]meeting.Span, 0, len(ms))
	for _, m := range ms {
		busy = append(busy, meeting.Span{Start: m.StartsAt, End: m.EndsAt})
	}
	for _, sp := range s.gatherExternalBusy(ctx, host.Email, []string{host.Email}, from, to)[host.Email] {
		busy = append(busy, sp)
	}

	now := from
	var out []BookingSlot
	for day := from.In(loc); day.Before(to); day = day.AddDate(0, 0, 1) {
		iso := int(day.Weekday())
		if iso == 0 {
			iso = 7
		}
		if !allowed[iso] {
			continue
		}
		y, mo, d := day.Date()
		winStart := time.Date(y, mo, d, 0, 0, 0, 0, loc).Add(time.Duration(et.AvailStartMinute) * time.Minute)
		winEnd := time.Date(y, mo, d, 0, 0, 0, 0, loc).Add(time.Duration(et.AvailEndMinute) * time.Minute)
		var dayBusy []meeting.Span
		for _, b := range busy {
			if meeting.Overlaps(b.Start, b.End, winStart, winEnd) {
				dayBusy = append(dayBusy, b)
			}
		}
		for _, f := range meeting.FreeSlots(dayBusy, winStart, winEnd, dur) {
			if !f.Start.After(now) {
				continue
			}
			out = append(out, BookingSlot{Start: f.Start.UTC(), End: f.End.UTC()})
		}
	}
	return out, nil
}
```

- [ ] **Step 2: `PublicBooking`** — append to `booking.go` (or `booking_availability.go`):
```go
type BookingEventView struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	DurationMins int    `json:"duration_mins"`
	OrgName      string `json:"org_name"`
	Timezone     string `json:"timezone"`
}

type BookingView struct {
	Event BookingEventView `json:"event"`
	Slots []BookingSlot    `json:"slots"`
}

func (s *Services) PublicBooking(ctx context.Context, slug string, now time.Time) (BookingView, error) {
	et, err := s.Store.GetBookingEventTypeBySlug(ctx, slug)
	if err != nil {
		return BookingView{}, err
	}
	if !et.Active {
		return BookingView{}, sql.ErrNoRows // -> IsNotFound -> 404
	}
	slots, err := s.BookingAvailability(ctx, et, now, now.AddDate(0, 0, 14))
	if err != nil {
		return BookingView{}, err
	}
	orgName := ""
	if org, oerr := s.Store.GetOrganization(ctx, et.OrganizationID); oerr == nil {
		orgName = org.Name
	}
	return BookingView{
		Event: BookingEventView{Title: et.Title, Description: et.Description, DurationMins: et.DurationMins, OrgName: orgName, Timezone: et.Timezone},
		Slots: slots,
	}, nil
}
```
(Add `"database/sql"` import for `sql.ErrNoRows`; `model.IsNotFound(sql.ErrNoRows)` is true.)

- [ ] **Step 3: Handler** — `handlers/public_booking.go`:
```go
package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (a *API) PublicBooking(c *fiber.Ctx) error {
	slug := c.Params("slug")
	view, err := a.App.PublicBooking(c.UserContext(), slug, time.Now())
	if err != nil {
		if model.IsNotFound(err) {
			return fiber.NewError(fiber.StatusNotFound, "not_found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "booking_failed")
	}
	return c.JSON(view)
}
```

- [ ] **Step 4: Route** — in `app.go`, register on `app` (no middleware), near the calendar callback public route: `app.Get("/api/book/:slug", api.PublicBooking)`.

- [ ] **Step 5: Failing tests**
- `booking_availability_test.go` (`package application`): a fake `Store` (override `GetPlatformUserByID`, `ListMeetingsOverlapping`) + a fake `BusyResolver` set on `Services.Busy` (or nil) — build an event type (Mon–Fri, 540–1020, 30min, tz "Asia/Almaty"), a window over a known week, inject a busy meeting blocking one slot, assert: a weekday returns slots; the busy block is excluded; a non-allowed weekday (e.g. weekend-only event) yields none; slots in the past are dropped. Use fixed dates (no `time.Now()` in the pure assertions — pass an explicit `from/to`).
- `public_booking_test.go`: fake `Repository` (override `GetBookingEventTypeBySlug`, `GetPlatformUserByID`, `ListMeetingsOverlapping`, `GetOrganization`) → `GET /api/book/:slug` returns `{event, slots}`; unknown slug (ErrNoRows) → 404; inactive event → 404; assert the JSON has no host email field.

- [ ] **Step 6: Run + build/vet/lint** — `env -u GOROOT go test ./internal/application/ ./internal/delivery/http/handlers/ -run 'Booking|PublicBooking' -v && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. Green.

- [ ] **Step 7: gofmt + commit**
```bash
gofmt -w internal/application/booking_availability.go internal/application/booking.go internal/delivery/http/handlers/public_booking.go internal/delivery/http/app.go internal/application/booking_availability_test.go internal/delivery/http/handlers/public_booking_test.go
git add apps/backend/internal/application/booking_availability.go apps/backend/internal/application/booking.go apps/backend/internal/delivery/http/handlers/public_booking.go apps/backend/internal/delivery/http/app.go apps/backend/internal/application/booking_availability_test.go apps/backend/internal/delivery/http/handlers/public_booking_test.go
git commit -m "feat(booking): public GET /api/book/:slug + BookingAvailability slot engine"
```

---

### Task 2: Public booking page (admin)

**Files:**
- Create: `apps/admin/app/routes/book.$slug.tsx` (self-contained public page)
- Modify: `apps/admin/app/routes.ts` — add the top-level `book/:slug` route (outside `_app`/`_auth`)

**Interfaces:** Consumes `GET /api/book/:slug` via plain `fetch` (no auth client).

- [ ] **Step 1: Route** — add `route("book/:slug", "routes/book.$slug.tsx")` to `routes.ts` at the TOP LEVEL (a sibling of `onboarding`/`logout`, NOT under `_app`/`_auth`).
- [ ] **Step 2: Page** — `book.$slug.tsx`: a default-exported component using `useParams()` for `slug` and a `useEffect` + plain `fetch(`/api/book/${slug}`)` (NOT the shared axios client) into local state (`loading` / `notFound` / `data`). Render: event title, duration ("{n} min"), org name; a day selector built from the distinct dates present in `slots` (format in the event `timezone` via `Intl.DateTimeFormat(undefined, { timeZone })`); the selected day's slot buttons (start time formatted in the event tz, tz label shown); selecting a slot sets local state and shows a **disabled** "Continue" button with helper text like "Booking confirmation coming soon" (the submit is 3-3). 404 → a simple "This booking link isn't available." Loading → a spinner. Plain English strings (no `useT`/LocaleProvider — this page is public, outside the authed tree). Reuse `@leadcat/ui` primitives (Card/Button) — they don't require auth context. ≤300 lines, no comments, no emoji.
- [ ] **Step 3: Verify + commit**
```bash
pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build
git add apps/admin/app/routes/book.\$slug.tsx apps/admin/app/routes.ts
git commit -m "feat(admin): public /book/:slug page (event + slots, select only)"
```
(Confirm the route file name React Router expects for a `:slug` param — match the existing flat-route convention, e.g. `book.$slug.tsx`.)

---

### Task 3: Whole-slice verification

**Files:** none

- [ ] **Step 1: Backend** — `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. Green.
- [ ] **Step 2: Frontend** — `pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build`. Green.
- [ ] **Step 3: Public-path checks (documented)** — confirm: `GET /api/book/:slug` has no auth middleware in `app.go`; the page uses plain `fetch` (no session/CSRF/X-Org-Id headers); the page does not import `useMe`/LocaleProvider/authed providers; the response carries no host email.
- [ ] **Step 4: Tree clean** — verify HEAD; `git status` no stray staged files.

---

## Notes for the executor

- **Reuse, don't refactor:** `BookingAvailability` mirrors `FreeSlots` but with the event tz/window/weekdays/duration; the shared `FreeSlots` is UNCHANGED.
- **Gather busy ONCE** over the whole 14-day window, then slice per day in memory (one external-calendar read).
- **Best-effort external busy** (`gatherExternalBusy` never errors) — a host calendar read failure degrades to DB-only; 3-3 re-checks conflicts at submit.
- **Public surface discipline:** endpoint has no auth; page uses plain `fetch`; no host email in the response; English-only strings.
- **Deferred:** booking submission → meeting creation + confirmation (3-3); visitor-tz conversion; public i18n.
```
