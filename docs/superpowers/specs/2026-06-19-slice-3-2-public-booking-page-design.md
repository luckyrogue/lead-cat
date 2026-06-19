# Slice 3-2 — Public Booking Page + Slot Availability (design)

**Date:** 2026-06-19
**Status:** approved — ready for implementation plan
**Part of:** SaaS Product Completion epic, **Track 3 (booking links)**, sub-slice **2** of 3. Follows 3-1.

## Epic / track context

3-1 shipped booking event-type config (table + admin CRUD). 3-2 makes the `/book/:slug` link
**resolve to a public page** showing the event + computed available slots (read/select only).
3-3 adds the actual booking submission (creates the meeting + confirmation).

**Decisions (from brainstorming):** a dedicated `BookingAvailability` method (do NOT change the
shared `FreeSlots`); a fixed 14-day forward window; the public page shows title + org name +
duration (host email not exposed); booking submit deferred to 3-3.

## Goal

`GET /api/book/:slug` (public, no auth) returns the event type + available slots over the next
14 days, computed against the host's real calendar (DB meetings ∪ the host's connected
external calendar). A public `/book/:slug` page in the admin app renders the event + a day
selector + the day's free slots (selecting a slot is local state; the confirm/submit is 3-3).

## Background — verified current state

- 3-1: `model.BookingEventType{… HostUserID, OrganizationID; Slug, Title, Description; DurationMins;
  Active; Timezone; AvailWeekdays []int; AvailStartMinute, AvailEndMinute …}`;
  `GetBookingEventTypeBySlug(ctx, slug)` on the `Repository`.
- 1c availability internals (`internal/application/conflict.go` / `availability.go`):
  `(s *Services) gatherExternalBusy(ctx, requesterEmail string, emails []string, from, to)
  map[string][]meeting.Span` (unexported, best-effort, reuses the `BusyResolver`); `FreeSlots`
  builds `busy []meeting.Span` from `Store.ListMeetingsOverlapping(emails, from, to)` then calls
  the domain `meeting.FreeSlots(dayBusy, winStart, winEnd, minDur) []meeting.Span`.
- `meeting.Span{Start, End time.Time}`; `application.FreeSlot{…}` (the existing slot DTO — reuse
  or define a booking `Slot{Start, End time.Time}`).
- The host's email is derivable from `HostUserID` via `GetPlatformUserByID(...).Email`.
- Org name via `GetOrganization(orgID).Name` (or the org repo's get-by-id).
- Public routes are registered directly on `app` with no middleware (e.g. the calendar OAuth
  callback `GET /api/calendar/connect/:provider/callback`).
- Admin routing: top-level routes outside `_app`/`_auth` are public (`/onboarding`, `/logout`).
  A `/book/:slug` route added at the top level is public; it must NOT use the authed api client
  paths (no `X-Org-Id`, no session) — use a plain fetch to `/api/book/:slug`.

## Design

### A. `BookingAvailability` (application)

```go
type BookingSlot struct { Start, End time.Time }   // json start/end (RFC3339)

func (s *Services) BookingAvailability(ctx context.Context, et model.BookingEventType, from, to time.Time) ([]BookingSlot, error)
```
Steps:
1. Resolve the host email (`GetPlatformUserByID(et.HostUserID).Email`); load the event tz
   (`time.LoadLocation(et.Timezone)`, fallback Almaty/UTC on error).
2. Gather busy once over `[from,to]`: DB meetings (`ListMeetingsOverlapping([hostEmail], from,
   to)` → spans) **∪** `s.gatherExternalBusy(ctx, hostEmail, []string{hostEmail}, from, to)[hostEmail]`.
3. For each calendar day `d` in `[from,to)` **in the event tz**: if `weekday(d) ∈ et.AvailWeekdays`
   (Mon=1..Sun=7), compute `winStart = d@AvailStartMinute`, `winEnd = d@AvailEndMinute` (in tz),
   filter busy to that day, and append `meeting.FreeSlots(dayBusy, winStart, winEnd,
   et.DurationMins*time.Minute)`.
4. Drop slots in the past (`Start <= now`). Return `[]BookingSlot` (UTC instants).
- Read-only, best-effort on external busy (a failed host-calendar read degrades to DB-only — the
  slot just may be offered when the host is actually busy; acceptable, surfaced again at 3-3
  conflict-check). Never errors on calendar access.

### B. Public endpoint

`GET /api/book/:slug` (registered on `app`, no auth):
- `GetBookingEventTypeBySlug(slug)` → not found OR `!active` → 404.
- Compute `BookingAvailability(et, now, now+14d)`.
- Resolve org name (`GetOrganization(et.OrganizationID).Name`).
- Response: `{ "event": {"title","description","duration_mins","org_name","timezone"},
  "slots": [{"start":"<RFC3339>","end":"<RFC3339>"}, …] }`. (No host email, no busy details.)
- A small app method `(s *Services) PublicBooking(ctx, slug string) (BookingView, error)` wraps
  the lookup + availability + org name; handler maps `IsNotFound`→404.

### C. Public page (admin)

- Top-level route `route("book/:slug", "routes/book.$slug.tsx")` in `apps/admin/app/routes.ts`
  (outside `_app`/`_auth`).
- A self-contained page (no authed providers / no `useMe`): fetch `/api/book/:slug` with a plain
  `fetch` (the page is public; the admin axios client adds session/CSRF/X-Org-Id headers which
  must NOT be sent — use `fetch`, not the shared client). Loading/404 states.
- Renders: event title, duration, org name; a **day selector** (the days that have slots, or a
  simple next-14-days list); the selected day's slot buttons (times shown in the event tz). A
  selected slot is held in component state; a "Continue" button is present but **disabled /
  inert until 3-3** (or shows "booking opens soon"). i18n: a minimal public `book.*` block
  (en/ru/kk) — the page has no locale gate, so default to a single language or detect via
  `navigator.language`; the plan picks the simplest (English default + the existing dict if
  reachable without the authed LocaleProvider).

### D. Slot display / timezone

Slots are computed in the event tz and returned as UTC instants + the event `timezone`; the page
formats them in the event tz (so a visitor sees the host's offered times in the host's tz, with
the tz label shown). Visitor-tz conversion is a nicety deferred.

## Testing / verification

- **`BookingAvailability`** unit test (fake `Store` + fake `BusyResolver`): given a host with a
  busy block, the returned slots exclude it; weekdays outside `AvailWeekdays` yield no slots;
  duration slicing correct; past slots dropped. (No Docker — pure with fakes.)
- **Public endpoint** (httptest + fakes): `GET /api/book/:slug` returns event + slots; unknown
  slug → 404; inactive event → 404; response shape (no host email).
- **Frontend:** admin typecheck/lint/build green; the public route renders without the authed
  providers (no `useMe`/session dependency).
- `go test -race ./...` + `golangci-lint` clean.

## Risks & mitigations

- **Public endpoint reads the host's calendar** (acts as the host). *Mitigation:* only FREE slots
  are returned — no event titles or busy details leak; best-effort so a calendar error degrades
  to DB-only.
- **Authed api client on a public page.** The admin client injects session/CSRF/X-Org-Id.
  *Mitigation:* the public page uses a plain `fetch`, not the shared client; no auth headers.
- **Locale on a public page** (no LocaleProvider). *Mitigation:* the plan picks the simplest —
  English default (or `navigator.language` best-effort); full public i18n is deferred.
- **Timezone correctness** (per-day windows across DST). *Mitigation:* compute day windows via
  `time.Date(..., loc)` in the event tz; unit-test a busy-exclusion case.
- **Slot fan-out cost** (14 days × external calendar). *Mitigation:* gather busy ONCE for the
  whole window, then slice per day in memory (one external read, not per-day).

## Done criteria

- `BookingAvailability` + `PublicBooking` app methods; `BookingSlot`/`BookingView` types.
- `GET /api/book/:slug` public endpoint (event + 14-day slots; 404 on unknown/inactive; no host
  email exposed).
- Public `/book/:slug` admin page (plain fetch, no authed providers) rendering event + day +
  slots; slot selection is local state; submit deferred to 3-3.
- Unit + httptest coverage; `-race` + lint green; admin typecheck/lint/build green.
- Booking submission + meeting creation + confirmation (3-3) explicitly deferred.
