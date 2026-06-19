# Slice 3-3 — Booking Submission (design)

**Date:** 2026-06-19
**Status:** approved — ready for implementation plan
**Part of:** SaaS Product Completion epic, **Track 3 (booking links)**, sub-slice **3 of 3 — the final slice of the epic.**

## Epic / track context

3-1 (event-type config) and 3-2 (public page + slot availability) are shipped. 3-3 makes the
public page **actually book**: a visitor picks a slot, enters name + email, and a meeting is
created on the host's calendar with the visitor invited — completing the Calendly-style flow and
the whole SaaS Product Completion epic.

**Decisions (from brainstorming):** reuse `CreateMeeting` (organizer=host, attendee=visitor,
once); a re-availability conflict check before create → **409** on race; confirmation = the
calendar invite + API response (explicit email deferred); **no rate-limiting** in 3-3 (flagged
for WS5/hardening).

## Goal

`POST /api/book/:slug` (public, no auth) `{name, email, start}` creates a meeting on the host's
connected calendar (Meet/Teams link, visitor as attendee, lifecycle notifications) and returns
the join link; the public `/book/:slug` page gains a name+email form and a confirmation panel.

## Background — verified current state

- `command.CreateMeeting(ctx, organizationID, organizerID uuid.UUID, in CreateInput) (model.Meeting, error)`:
  builds `name := meeting.GenerateName(in.Dept, in.Type, in.Host, startsAt, rec)`; parses
  `Date`("2006-01-02") + `Start`/`End`("15:04") in `in.Timezone` (fallback org tz); for a `once`
  recurrence creates ONE calendar event via the Track-1 resolver (`Calendar.For(orgID,
  organizerEmail)` → host's connected calendar, MeetLink populated), persists the meeting +
  `Participants`, and enqueues notifications. Participants are plain emails (external OK).
  Exposed to the app as `Services.Commands.CreateMeeting` (or via a `Services` wrapper — confirm
  the call path the booking layer uses).
- `meeting.GenerateName(dept, mtype, host, date, r)` (`domain/meeting/naming.go`) — the ТЗ name
  standard; the plan confirms the output for the booking mapping (event title via one field).
- 1c conflict engine: `MeetingConflicts(ctx, requesterEmail, emails, start, end, excludeID)` —
  checks DB meetings ∪ external calendar busy. A re-availability check = `MeetingConflicts(ctx,
  hostEmail, []string{hostEmail}, start, end, uuid.Nil)`; non-empty → slot taken.
- 3-1/3-2: `GetBookingEventTypeBySlug(slug)`, `model.BookingEventType` (HostUserID, OrganizationID,
  Title, Description, DurationMins, Active, Timezone, AvailWeekdays, AvailStartMinute,
  AvailEndMinute); `GetPlatformUserByID(id)→(user,bool,err)`; `BookingAvailability`.
- Public GET endpoint + page exist (3-2); the page holds a selected slot, "Continue" disabled.
- `model.Meeting` has `MeetLink`.

## Design

### A. Application: `SubmitBooking`

```go
type BookingRequest struct { Name, Email string; Start time.Time }
type BookingConfirmation struct { MeetLink string; Start, End time.Time }

func (s *Services) SubmitBooking(ctx context.Context, slug string, req BookingRequest) (BookingConfirmation, error)
```
Steps:
1. `GetBookingEventTypeBySlug(slug)`; not found or `!Active` → `model.IsNotFound` (→404).
2. Validate: `Name` non-empty; `Email` parses (`mail.ParseAddress`); `Start` is in the future,
   weekday ∈ `AvailWeekdays`, and `[Start, Start+dur]` lies within the event window for that day
   in the event tz → else `ErrInvalidBooking` (→400). `end := Start.Add(DurationMins*min)`.
3. Resolve host email (`GetPlatformUserByID(HostUserID).Email`).
4. **Re-availability:** `MeetingConflicts(ctx, hostEmail, []string{hostEmail}, Start, end, uuid.Nil)`
   → if non-empty, `ErrSlotTaken` (→409).
5. Create the meeting via `CreateMeeting(ctx, et.OrganizationID, et.HostUserID, CreateInput{...})`:
   - `Date` = Start in event tz formatted `2006-01-02`; `Start`/`End` = `15:04` in event tz;
     `Timezone` = et.Timezone; `Recurrence` = "once".
   - Name mapping: the meeting name = the event title — pass the title via the `Type` field
     (`GenerateName("", et.Title, "", …)`); the plan verifies the rendered name and trims any
     separator artifacts (or, if cleaner, the plan adds an optional `Title` override to
     `CreateInput` used only here — minimal command change).
   - `Description` = `"Booked via {et.Title} by {req.Name} <{req.Email}>"`.
   - `Participants` = `[]model.MeetingParticipant{{Email: req.Email}}`.
6. Return `BookingConfirmation{MeetLink: m.MeetLink, Start, End: end}`.
- The visitor is a calendar attendee → receives the calendar invite (the confirmation). The
  existing lifecycle notifications fire (host side).

### B. Public POST endpoint

`POST /api/book/:slug` (registered on `app`, no auth, next to the GET):
- Body `{ "name", "email", "start" }` (start RFC3339). Bad body → 400.
- `SubmitBooking` → map errors: `IsNotFound`→404, `ErrInvalidBooking`→400, `ErrSlotTaken`→409,
  else 500. Success → `200 { "meet_link", "start", "end" }`.

### C. Frontend (public page)

In `routes/book.$slug.tsx`: enable "Continue" when a slot is selected → reveal a small form
(name + email, plain inputs, client-side required/email validation) → `fetch` POST to
`/api/book/${slug}` with `{name, email, start: selectedSlot.start}`. States:
- success → a confirmation panel: "You're booked!", the date/time (event tz), and the meet link
  ("Join link" — also "added to your calendar; check your email").
- 409 → "That time was just taken." + refetch slots (clear selection).
- 400 → inline error; 404 → the existing not-available state.
Plain `fetch` (no auth client), English strings (consistent with 3-2). ≤300 lines (split a
`booking-form`/`confirmation` subcomponent if the route file would exceed it).

### D. Errors

- `model` sentinels: `ErrInvalidBooking`, `ErrSlotTaken` (or reuse `ErrForbidden`? no — distinct).
- Re-availability uses the same conflict engine as the in-app create, so a booking can't
  double-book the host beyond the existing app's guarantees.

## Testing / verification

- **`SubmitBooking`** (fakes): happy path → calls `CreateMeeting` with once + visitor participant
  + tz, returns the meet link; invalid email/past-start/out-of-window → `ErrInvalidBooking`;
  conflict (fake `MeetingConflicts` returns one) → `ErrSlotTaken`; unknown/inactive slug →
  IsNotFound. (Use a fake that captures the `CreateMeeting` input to assert the mapping;
  `CreateMeeting` itself is already tested — here assert the booking→input mapping + the
  guards.)
- **POST endpoint** (httptest + fakes): 200 {meet_link} happy; 404 unknown; 409 conflict; 400 bad
  email / bad body.
- **Frontend:** admin typecheck/lint/build green; the page posts and renders confirmation/409/400.
- `go test -race ./...` + `golangci-lint` clean.

## Risks & mitigations

- **No rate-limiting on a public mutating endpoint** (abuse: spam bookings → spam calendar
  events). *Mitigation:* validate email + slot realness + re-availability; flag rate-limiting as
  a follow-up (WS5 hardening). Explicitly out of 3-3 scope.
- **Race / double-book:** two visitors grab the same slot concurrently. *Mitigation:* the
  re-availability `MeetingConflicts` check narrows the window; a true TOCTOU remains (no slot
  lock). Accept for MVP (the host sees both and can adjust); a hard lock is a later concern.
- **GenerateName mapping** may render an odd title for empty dept/host. *Mitigation:* the plan
  verifies the output and trims, or adds a minimal `Title` override to `CreateInput`.
- **CreateMeeting side effects** (notifications to the host) are desirable; the visitor invite is
  the calendar attendee invite. Confirm `CreateMeeting`'s notification path doesn't choke on an
  external participant (it shouldn't — participants are emails).

## Done criteria

- `SubmitBooking` app method (validate → re-availability 409 → `CreateMeeting` as host with the
  visitor attendee → return meet link) + `model` sentinels.
- Public `POST /api/book/:slug` (200 {meet_link,start,end}; 404/400/409 mapped).
- Public page: name+email form + confirmation panel + 409/400 handling.
- Unit + httptest coverage; `-race` + lint green; admin typecheck/lint/build green.
- Rate-limiting + explicit confirmation email + reschedule/cancel explicitly deferred. **Track 3
  and the SaaS Product Completion epic complete.**
