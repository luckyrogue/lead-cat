# Slice 3-3 — Booking Submission Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `POST /api/book/:slug` (public) books a slot — re-checks availability, creates a meeting on the host's calendar with the visitor invited (Meet/Teams link), and returns the link; the public page gets a name+email form + confirmation. Completes Track 3 + the epic.

**Architecture:** A `SubmitBooking` application method reuses `CreateMeeting` (organizer=host, attendee=visitor, once) after a re-availability `MeetingConflicts` check; a minimal optional `Title` override on `CreateInput` gives a clean booking meeting name; a public POST endpoint + the page form.

**Tech Stack:** Go 1.26 (Fiber), React Router v7 admin SPA.

## Global Constraints

- Backend at `apps/backend`; run Go as `env -u GOROOT go ...`. Spec: `docs/superpowers/specs/2026-06-19-slice-3-3-booking-submission-design.md`.
- depguard: `application` imports zero `internal/infrastructure`; sentinels in `internal/application/model`. No code comments in new Go/TS files.
- **Public endpoint = NO auth middleware** (register on `app`, next to the GET `/api/book/:slug`). Public page uses plain `fetch` (no shared client / no authed providers), English strings (consistent with 3-2).
- **No regression in `CreateMeeting`:** the new `Title` override is honored ONLY when non-empty; empty → existing `GenerateName` path unchanged. Existing create tests must still pass.
- Frontend: files ≤300 lines, no emoji (lucide only), no comments; never repo-wide prettier (additive); pnpm filter `admin`.
- gofmt all Go; `golangci-lint run --config ../../config/.golangci.yml ./...` = 0 issues; `go test -race ./...` green.
- Work on `main`; never `git add -A`; **verify HEAD before each commit** (user commits in parallel — commit on top, never rebase/reset). Trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

**Reference (verified):**
- `Services.CreateMeeting(ctx, orgID, organizerID uuid.UUID, in CreateMeetingInput) (model.Meeting, error)` (`meeting_service.go:44`) wraps `Commands.CreateMeeting`. `CreateMeetingInput` is the application alias of `command.CreateInput{ Dept, Type, Host, Date, Start, End, Recurrence, RecurrenceUntil string; RecurrenceDays []int; Description string; Participants []model.MeetingParticipant; Timezone string }`.
- `CreateMeeting` (`command/meetings.go`): once-path name at line ~58 `name := meeting.GenerateName(in.Dept, in.Type, in.Host, startsAt, rec)`; series-path at line ~87. Date format `2006-01-02`, time `15:04`, parsed in `in.Timezone`.
- `GenerateName(dept,type,host,date,r)` = `"dept | type | host | YYYY-MM-DD"` + optional freq label.
- `MeetingConflicts(ctx, requesterEmail, emails, start, end, excludeID uuid.UUID) ([]Conflict, error)` (1c) — DB ∪ external busy.
- 3-1/3-2: `GetBookingEventTypeBySlug`, `model.BookingEventType`, `GetPlatformUserByID(id)→(user,bool,err)`, `model.Meeting.MeetLink`. Public GET + page (`routes/book.$slug.tsx`) exist; the page holds a selected slot, "Continue" disabled.

---

### Task 1: `Title` override + `SubmitBooking` + public POST endpoint

**Files:**
- Modify: `apps/backend/internal/application/command/meetings.go` — add `Title` to `CreateInput`; honor it in the once-create name
- Modify: `apps/backend/internal/application/model/errors.go` — add `ErrInvalidBooking`, `ErrSlotTaken`
- Create: `apps/backend/internal/application/booking_submit.go` — `SubmitBooking` + types
- Create: `apps/backend/internal/delivery/http/handlers/public_booking_submit.go`
- Modify: `apps/backend/internal/delivery/http/app.go` — `POST /api/book/:slug`
- Test: `apps/backend/internal/application/booking_submit_test.go`, `apps/backend/internal/delivery/http/handlers/public_booking_submit_test.go`

**Interfaces:**
- Produces:
  - `CreateInput.Title string` (new optional field).
  - `model.ErrInvalidBooking`, `model.ErrSlotTaken`.
  - `application.BookingRequest{ Name, Email string; Start time.Time }`; `application.BookingConfirmation{ MeetLink string; Start, End time.Time }`.
  - `(s *Services) SubmitBooking(ctx, slug string, req BookingRequest) (BookingConfirmation, error)`.
  - `POST /api/book/:slug`.

- [ ] **Step 1: `Title` override (no-regression)** — in `command/meetings.go`:
  - Add `Title string` to `CreateInput` (after `Description`).
  - In the once-create path, replace `name := meeting.GenerateName(in.Dept, in.Type, in.Host, startsAt, rec)` with:
    ```go
    name := strings.TrimSpace(in.Title)
    if name == "" {
        name = meeting.GenerateName(in.Dept, in.Type, in.Host, startsAt, rec)
    }
    ```
    (Add `"strings"` if not imported.) Leave the series-path (line ~87) unchanged — booking is once-only; document that `Title` is honored only for once. Existing callers don't set `Title` → behavior unchanged.

- [ ] **Step 2: Run existing create tests (no regression)** — `env -u GOROOT go test ./internal/application/command/ -run Create -v`. Expected: PASS (Title defaults empty → GenerateName as before).

- [ ] **Step 3: model sentinels** — append to `model/errors.go`: `var ErrInvalidBooking = errors.New("invalid booking")` and `var ErrSlotTaken = errors.New("slot taken")`.

- [ ] **Step 4: Failing `SubmitBooking` test** — `booking_submit_test.go` (`package application`): fake `Store` (override `GetBookingEventTypeBySlug`, `GetPlatformUserByID`) + a fake that captures the `CreateMeeting` call. Since `SubmitBooking` calls `s.CreateMeeting` (which calls `s.Commands.CreateMeeting`), inject a fake `Commands` OR test against a `Services` whose `Commands` uses fakes — SIMPLER: make `SubmitBooking` call a seam you can fake. Pragmatic approach: have `SubmitBooking` use `s.CreateMeeting` and `s.MeetingConflicts`; in the test, construct a `Services` with a fake `Store` + fake `Calendar` (stub provider) + fake `Queue` + a real `Commands` wired to those fakes (mirror `internal/application/command` fakes_test pattern), set `Services.Commands` accordingly. Assert:
  - happy path: valid req on a free slot → `CreateMeeting` invoked with `Recurrence:"once"`, `Participants:[{Email:visitor}]`, `Title` = the event title (non-empty), `Timezone` = event tz; returns a `BookingConfirmation` with the stub MeetLink.
  - invalid email → `ErrInvalidBooking`; past start → `ErrInvalidBooking`; start outside the event window/weekday → `ErrInvalidBooking`.
  - conflict: fake `MeetingConflicts` (or a busy DB meeting via the fake store) returns one → `ErrSlotTaken`.
  - unknown/inactive slug → `model.IsNotFound`.
  (If wiring a real `Commands` is heavy, an acceptable alternative: extract the create call behind a tiny unexported method `s.createBookingMeeting(...)` and assert the guard logic + a captured input via a fake store's `CreateMeeting`/`CreateMeetingSeries`. Choose the lighter path; keep the test focused on the booking guards + mapping.)

- [ ] **Step 5: Implement `SubmitBooking`** — `booking_submit.go`:
```go
package application

import (
	"context"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type BookingRequest struct {
	Name  string
	Email string
	Start time.Time
}

type BookingConfirmation struct {
	MeetLink string    `json:"meet_link"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
}

func (s *Services) SubmitBooking(ctx context.Context, slug string, req BookingRequest) (BookingConfirmation, error) {
	et, err := s.Store.GetBookingEventTypeBySlug(ctx, slug)
	if err != nil {
		return BookingConfirmation{}, err
	}
	if !et.Active {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	if _, perr := mail.ParseAddress(req.Email); perr != nil {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	loc := loadLoc(et.Timezone)
	start := req.Start.In(loc)
	dur := time.Duration(et.DurationMins) * time.Minute
	end := start.Add(dur)
	if !start.After(time.Now()) {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	iso := int(start.Weekday())
	if iso == 0 {
		iso = 7
	}
	if !weekdaySet(et.AvailWeekdays)[iso] {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	minute := start.Hour()*60 + start.Minute()
	if minute < et.AvailStartMinute || minute+et.DurationMins > et.AvailEndMinute {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	host, ok, err := s.Store.GetPlatformUserByID(ctx, et.HostUserID)
	if err != nil || !ok || host.Email == "" {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	conflicts, err := s.MeetingConflicts(ctx, host.Email, []string{host.Email}, start.UTC(), end.UTC(), uuid.Nil)
	if err != nil {
		return BookingConfirmation{}, err
	}
	if len(conflicts) > 0 {
		return BookingConfirmation{}, model.ErrSlotTaken
	}
	m, err := s.CreateMeeting(ctx, et.OrganizationID, et.HostUserID, CreateMeetingInput{
		Title:        et.Title + " — " + name,
		Description:  "Booked via " + et.Title + " by " + name + " <" + req.Email + ">",
		Date:         start.Format("2006-01-02"),
		Start:        start.Format("15:04"),
		End:          end.Format("15:04"),
		Timezone:     et.Timezone,
		Recurrence:   "once",
		Participants: []model.MeetingParticipant{{Email: req.Email}},
	})
	if err != nil {
		return BookingConfirmation{}, err
	}
	return BookingConfirmation{MeetLink: m.MeetLink, Start: start.UTC(), End: end.UTC()}, nil
}
```
(`CreateMeetingInput` is the app alias; `loadLoc`/`weekdaySet` exist from 3-2. Confirm `model.IsNotFound` covers the unknown-slug repo error so the handler maps 404.)

- [ ] **Step 6: Handler** — `public_booking_submit.go`:
```go
package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (a *API) PublicBookingSubmit(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Start string `json:"start"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request")
	}
	start, err := time.Parse(time.RFC3339, body.Start)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad_start")
	}
	conf, err := a.App.SubmitBooking(c.UserContext(), slug, application.BookingRequest{Name: body.Name, Email: body.Email, Start: start})
	if err != nil {
		switch {
		case model.IsNotFound(err):
			return fiber.NewError(fiber.StatusNotFound, "not_found")
		case err == model.ErrSlotTaken:
			return fiber.NewError(fiber.StatusConflict, "slot_taken")
		case err == model.ErrInvalidBooking:
			return fiber.NewError(fiber.StatusBadRequest, "invalid_booking")
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "booking_failed")
		}
	}
	return c.JSON(conf)
}
```
(Use `errors.Is` if the sentinels may be wrapped; here they're returned directly, but `errors.Is(err, model.ErrSlotTaken)` is safer — prefer it.)

- [ ] **Step 7: Route** — in `app.go`, next to the GET: `app.Post("/api/book/:slug", api.PublicBookingSubmit)` (no middleware).

- [ ] **Step 8: Endpoint test** — `public_booking_submit_test.go`: fakes wired so POST returns 200 `{meet_link,start,end}` happy; 404 unknown slug; 409 when conflicts; 400 bad email / bad body / bad start. Pure fakes.

- [ ] **Step 9: Run + build/vet/lint** — `env -u GOROOT go test ./internal/application/... ./internal/delivery/http/handlers/ -run 'Booking|Create' && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. Green (existing create + booking tests pass).

- [ ] **Step 10: gofmt + commit**
```bash
gofmt -w internal/application/command/meetings.go internal/application/model/errors.go internal/application/booking_submit.go internal/delivery/http/handlers/public_booking_submit.go internal/delivery/http/app.go internal/application/booking_submit_test.go internal/delivery/http/handlers/public_booking_submit_test.go
git add apps/backend/internal/application/command/meetings.go apps/backend/internal/application/model/errors.go apps/backend/internal/application/booking_submit.go apps/backend/internal/delivery/http/handlers/public_booking_submit.go apps/backend/internal/delivery/http/app.go apps/backend/internal/application/booking_submit_test.go apps/backend/internal/delivery/http/handlers/public_booking_submit_test.go
git commit -m "feat(booking): public POST /api/book/:slug submission (re-availability + create as host)"
```

---

### Task 2: Public page — booking form + confirmation

**Files:**
- Modify: `apps/admin/app/routes/book.$slug.tsx` (+ optionally a `booking-form`/`confirmation` subcomponent if >300 lines)

**Interfaces:** `POST /api/book/:slug` via plain `fetch`.

- [ ] **Step 1:** When a slot is selected, enable "Continue" → reveal a form (name + email, required + email validation). Submit → `fetch("/api/book/"+slug, { method:"POST", headers:{"Content-Type":"application/json"}, body: JSON.stringify({name, email, start: selectedSlot.start}) })`. States:
  - 200 → parse `{meet_link, start, end}`; show a confirmation panel: "You're booked!", the date/time (event tz), and the join link (`<a href={meet_link}>`), plus "Added to your calendar — check your email."
  - 409 → "That time was just taken." + re-fetch slots (clear selection, reopen the picker).
  - 400 → inline "Please check your details."
  - network/500 → a generic error.
- [ ] **Step 2:** Keep plain `fetch` (no auth client), English strings, no comments/emoji, file(s) ≤300 lines (extract the form/confirmation into a sibling component if the route file would exceed it).
- [ ] **Step 3: Verify + commit**
```bash
pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build
git add apps/admin/app/routes/book.\$slug.tsx
# include any new sibling component file
git commit -m "feat(admin): booking form + confirmation on public /book/:slug page"
```

---

### Task 3: Whole-slice verification

**Files:** none

- [ ] **Step 1: Backend** — `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -race ./... && golangci-lint run --config ../../config/.golangci.yml ./...`. Green; existing meeting-create tests + new booking tests pass.
- [ ] **Step 2: Frontend** — `pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build`. Green.
- [ ] **Step 3: Public-path checks (documented)** — `POST /api/book/:slug` has no auth middleware; page uses plain `fetch`; 409 re-fetches slots; no host email in any response.
- [ ] **Step 4: Tree clean** — verify HEAD; `git status` no stray staged files.

---

## Notes for the executor

- **`Title` override is no-regression:** only honored when non-empty; existing create callers don't set it → `GenerateName` unchanged. Re-run the existing create tests (Step 2) to prove it.
- **Re-availability before create** (`MeetingConflicts` for the host) → 409 on race. A true TOCTOU remains (no slot lock) — accepted for MVP.
- **Reuse `CreateMeeting`** — it does calendar event + Meet/Teams link + visitor attendee invite + notifications + persistence. Booking = once, organizer = host, participant = visitor.
- **Public discipline:** endpoint unauthenticated; page plain `fetch`; no host email in responses.
- **Deferred (flagged for hardening/WS5):** rate-limiting on the public POST; explicit confirmation email; reschedule/cancel. After this slice, **Track 3 and the SaaS Product Completion epic are complete.**
```
