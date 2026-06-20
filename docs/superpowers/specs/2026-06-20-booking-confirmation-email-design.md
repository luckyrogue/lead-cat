# Booking Confirmation Email (design)

## Problem

The public booking flow (`POST /api/book/:slug`) creates a Google Calendar + Meet
event for the host and stores the visitor as a participant, but **sends no
application email**. The confirmation screen tells the visitor *"Added to your
calendar — check your email"* (`apps/admin/app/routes/book.$slug.form.tsx`), yet
the backend never sends one — the only possible email is Google's own
attendee invite, which is suppressed entirely in calendar-stub mode. The host
likewise gets no email (only a Telegram message, and only if they linked
Telegram). The booker email is a visible broken promise; the host email is a
real gap for hosts who have not connected Telegram.

## Goal

After a successful booking, send two best-effort transactional emails:

1. **Booker confirmation** — to the visitor, in the booking page's locale.
2. **Host notification** — to the host, in the host's stored language.

Both follow the existing `emailtemplates` pattern (trilingual ru/en/kk via
`NormalizeLang`, default ru) and the existing best-effort send pattern
(nil-guard the mailer, log on error, never fail the booking — it already
succeeded).

## Scope

**In scope**
- New `emailtemplates/booking.go` with two render functions + data structs.
- Sending both emails from `Services.SubmitBooking` after `CreateMeeting`.
- Threading the booking page locale into `SubmitBooking` (new `Language` field
  on `BookingRequest`, new `language` field on the submit request body, and the
  frontend passing its active locale).
- Unit tests for the renderers and for the send behavior.

**Out of scope** (separate follow-ups, decided explicitly)
- The Google `Insert(...).SendUpdates("all")` change — `Insert` is shared by all
  meeting creation, so flipping it sends Google attendee mail for *every*
  meeting, not just bookings. Broader blast radius → its own decision.
- `.ics` attachment on the confirmation email.
- A separate `bookings` ledger table (visitor is still stored as a meeting
  participant, unchanged).

## Design

### Templates — `apps/backend/internal/platform/emailtemplates/booking.go`

Two emails, one file (closely related — same booking, two perspectives), mirroring
the structure of `welcome.go` / `reminder.go` (a `*Data` struct with a `Language`
field, a `switch NormalizeLang(...)` for copy, a shared render helper, and a
`Render*` function returning `(subject, text, html string, err error)`).

```go
// Sent to the external visitor who booked.
type BookingConfirmationData struct {
    Language   string // booking page locale; NormalizeLang → ru/en/kk
    BookerName string // for the greeting
    EventTitle string // booking event type title (the meeting identity)
    Date       string // e.g. "Sat, 20 Jun 2026", formatted in the event-type timezone
    TimeRange  string // e.g. "14:00 – 14:30 (GMT+5)" — includes a timezone label
    MeetLink   string
}
func RenderBookingConfirmation(d BookingConfirmationData) (subject, text, html string, err error)

// Sent to the host (org member who owns the event type).
type BookingHostNotificationData struct {
    Language    string // host's stored platform_users.language
    EventTitle  string
    BookerName  string
    BookerEmail string
    Date        string // formatted in the host timezone (fallback: event-type tz)
    TimeRange   string // includes a timezone label
    MeetLink    string
}
func RenderBookingHostNotification(d BookingHostNotificationData) (subject, text, html string, err error)
```

Copy intent: booker email = "Your booking is confirmed" + event title, date,
time, Join Google Meet button. Host email = "New booking: {EventTitle}" + booker
name/email, date, time, Join Google Meet button. Reuse the existing `button.go`
CTA helper for the Meet link.

Note: `model.PlatformUser` has **no** name field — do not reference a host name;
the booker email identifies the meeting by `EventTitle` only (and does not expose
the host's email, for privacy). The host email shows the booker's name and email.

### Timezone in the rendered times

- Booker email: format `Date`/`TimeRange` in the **event-type timezone**
  (`et.Timezone`) — the same zone the visitor saw when picking the slot.
- Host email: format in the **host timezone** (`host.Timezone`), falling back to
  `et.Timezone` when the host has none stored.
- Both `TimeRange` strings carry an explicit timezone label so the recipient is
  never guessing. The values are computed in `SubmitBooking` (which already holds
  `start`/`end` and the zones) and passed into the data structs as preformatted
  strings — the templates do no timezone math.

### Locale plumbing

- Add `Language string` to `application.BookingRequest`.
- `handlers.PublicBookingSubmit`: add `Language string \`json:"language"\`` to the
  request body struct and pass `body.Language` into `BookingRequest`.
- Frontend `apps/admin/app/routes/book.$slug.form.tsx`: include
  `language: <active page locale>` in the `fetch('/api/book/${slug}')` JSON body.
  Use the page's existing active-locale source (the plan pins the exact hook while
  implementing). Empty/unknown → `NormalizeLang` falls back to ru.

### Send path — `Services.SubmitBooking`

After `CreateMeeting` succeeds (just before returning `BookingConfirmation`):

1. Build the booker `BookingConfirmationData` (lang = `req.Language`,
   times in `et.Timezone`) and the host `BookingHostNotificationData`
   (lang = `host.Language`, times in `host.Timezone` ?? `et.Timezone`).
2. For each: render, then `s.email.SendMultipart(ctx, to, subject, text, html, "")`.
3. **Best-effort, never blocking:** if `s.email == nil` (mailer not configured —
   e.g. CALENDAR/web-auth not wired in some envs), skip silently. If render or
   send returns an error, log it (`Warn`, with the booking/meeting id, no PII
   beyond what we already store) and continue. The booking is already persisted;
   email failure must not turn a successful booking into an error response.

This mirrors `sendWelcomeEmail` (`services.go:373`) and the invite send
(`services.go:304`), which are both best-effort.

### Error handling summary

- Booking creation errors propagate as today (unchanged).
- Email errors are swallowed after logging — they never change the HTTP result.
- A nil mailer is a no-op, not a panic.

## Testing

- **`emailtemplates/booking_test.go`** (pure, like `email_test.go`): for each
  renderer, assert ru/en/kk produce a non-empty localized subject, and the text
  body contains the Meet link and the time range. Assert default/garbage language
  falls back to ru.
- **`SubmitBooking` test** (`booking_submit_test.go` or extend existing) with an
  in-memory fake `EmailSender` that records calls and one that returns an error:
  - On success: two emails are sent — one to the booker email, one to the host
    email — and the returned `BookingConfirmation` is correct.
  - When the fake mailer returns an error (or is nil): `SubmitBooking` still
    returns success and the meeting is still created.
  - Reuse the existing booking-submit test fakes for `Store`/`CalendarProvider`.

## Out-of-scope follow-ups (noted)

- `Insert(...).SendUpdates("all")` on the Google adapter — decide separately
  (affects all meeting creation).
- `.ics` attachment; dedicated `bookings` table; host email when host has no
  stored language already defaults to ru via `NormalizeLang`.
