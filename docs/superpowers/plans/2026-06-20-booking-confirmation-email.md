# Booking Confirmation Email Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** After a successful public booking, send two best-effort emails — a confirmation to the booker (in the visitor's browser language) and a notification to the host (in the host's stored language).

**Architecture:** A new `emailtemplates/booking.go` with two render functions over one shared HTML/text template (mirrors `welcome.go`). `Services.SubmitBooking` renders and sends both after `CreateMeeting`, best-effort (nil-guard the mailer, log + continue — never fail the booking). The visitor language is threaded through a new `BookingRequest.Language` field, populated by the public handler from a new `language` body field, which the frontend fills from `navigator.language`.

**Tech Stack:** Go (Fiber handler, `emailtemplates` package, zap), React Router 8 (admin public route), Vitest not required (frontend change is build/typecheck-verified per repo convention).

## Global Constraints

- **Trilingual via `NormalizeLang`:** every render branches on `NormalizeLang(lang)` → `ru` | `en` | `kk`, default `ru`. Copy these exact values verbatim where shown.
- **Best-effort send:** sending must nil-guard `s.email` (it is `nil` when web-auth/mailer is unconfigured and in unit tests). A render or send error is logged via `s.Log.Warn(...)` and swallowed — `SubmitBooking` still returns success. Email never turns a created booking into an error.
- **No PII leakage / privacy:** the booker email must NOT include the host's email address. The host email includes the booker's name + email (that is the point of the notification).
- **No host name exists:** `model.PlatformUser` has no name field — never reference a host display name.
- **Timezones:** booker email times are formatted in the event-type timezone (`et.Timezone`); host email times in the host timezone (`host.Timezone`, fallback `et.Timezone`). Each email shows the IANA timezone string as a label.
- **HTML safety:** user-supplied strings (booker name/email, event title) render through `html/template` (auto-escaped) — never through string concatenation into HTML.
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: Booking email templates (`emailtemplates/booking.go`)

Two render functions over one shared template, modeled on `welcome.go`.

**Files:**
- Create: `apps/backend/internal/platform/emailtemplates/booking.go`
- Test: `apps/backend/internal/platform/emailtemplates/booking_test.go`

**Interfaces:**
- Produces (consumed by Task 2):
  ```go
  type BookingConfirmationData struct {
      Language, BookerName, EventTitle, Date, Time, Tz, MeetLink string
  }
  func RenderBookingConfirmation(d BookingConfirmationData) (subject, text, html string, err error)

  type BookingHostNotificationData struct {
      Language, EventTitle, BookerName, BookerEmail, Date, Time, Tz, MeetLink string
  }
  func RenderBookingHostNotification(d BookingHostNotificationData) (subject, text, html string, err error)
  ```

- [ ] **Step 1: Write the failing test `booking_test.go`**

```go
package emailtemplates

import (
	"html"
	"strings"
	"testing"
)

func TestRenderBookingConfirmation_Localized(t *testing.T) {
	for _, tc := range []struct{ lang, subjPart string }{
		{"ru", "подтвержден"},
		{"en", "confirmed"},
		{"kk", "расталды"},
	} {
		subject, text, out, err := RenderBookingConfirmation(BookingConfirmationData{
			Language: tc.lang, BookerName: "Visitor V", EventTitle: "Intro Call",
			Date: "Mon, 22 Jun 2026", Time: "10:00 – 10:30", Tz: "Asia/Almaty",
			MeetLink: "https://meet.google.com/abc-defg-hij",
		})
		if err != nil {
			t.Fatalf("[%s] render: %v", tc.lang, err)
		}
		if !strings.Contains(strings.ToLower(subject), tc.subjPart) {
			t.Fatalf("[%s] subject = %q, want contains %q", tc.lang, subject, tc.subjPart)
		}
		decoded := html.UnescapeString(out)
		for _, want := range []string{"Intro Call", "Mon, 22 Jun 2026", "10:00 – 10:30", "Asia/Almaty", "https://meet.google.com/abc-defg-hij"} {
			if !strings.Contains(decoded, want) {
				t.Fatalf("[%s] html missing %q", tc.lang, want)
			}
		}
		if !strings.Contains(text, "https://meet.google.com/abc-defg-hij") {
			t.Fatalf("[%s] text missing meet link", tc.lang)
		}
	}
}

func TestRenderBookingConfirmation_DefaultsToRu(t *testing.T) {
	subject, _, _, err := RenderBookingConfirmation(BookingConfirmationData{Language: "fr", EventTitle: "X"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(strings.ToLower(subject), "подтвержден") {
		t.Fatalf("garbage lang should fall back to ru, subject = %q", subject)
	}
}

func TestRenderBookingHostNotification_Localized(t *testing.T) {
	subject, text, out, err := RenderBookingHostNotification(BookingHostNotificationData{
		Language: "en", EventTitle: "Intro Call",
		BookerName: "Visitor V", BookerEmail: "visitor@example.com",
		Date: "Mon, 22 Jun 2026", Time: "10:00 – 10:30", Tz: "Asia/Almaty",
		MeetLink: "https://meet.google.com/abc-defg-hij",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(strings.ToLower(subject), "booking") {
		t.Fatalf("subject = %q", subject)
	}
	decoded := html.UnescapeString(out)
	for _, want := range []string{"Visitor V", "visitor@example.com", "Intro Call", "10:00 – 10:30", "https://meet.google.com/abc-defg-hij"} {
		if !strings.Contains(decoded, want) {
			t.Fatalf("html missing %q", want)
		}
	}
	if !strings.Contains(text, "visitor@example.com") {
		t.Fatalf("text missing booker email")
	}
}

func TestRenderBooking_EscapesUserContent(t *testing.T) {
	_, _, out, err := RenderBookingHostNotification(BookingHostNotificationData{
		Language: "ru", EventTitle: `<script>alert(1)</script>`, BookerName: "x", BookerEmail: "e@e.com",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Fatal("event title must be HTML-escaped")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/backend && go test ./internal/platform/emailtemplates/ -run TestRenderBooking -v`
Expected: FAIL — compile error `undefined: RenderBookingConfirmation` / `RenderBookingHostNotification`.

- [ ] **Step 3: Implement `booking.go`**

```go
package emailtemplates

import (
	"bytes"
	"html/template"
	texttemplate "text/template"
)

// ----- Booker confirmation -----

type BookingConfirmationData struct {
	Language   string
	BookerName string
	EventTitle string
	Date       string
	Time       string
	Tz         string
	MeetLink   string
}

func bookingConfirmSubject(lang string) string {
	switch NormalizeLang(lang) {
	case "en":
		return "Your booking is confirmed 🐾"
	case "kk":
		return "Брондауыңыз расталды 🐾"
	default:
		return "Ваша бронь подтверждена 🐾"
	}
}

type bookingConfirmL struct{ Hi, Intro, EventLbl, WhenLbl, Cta, Foot, DefaultName string }

func bookingConfirmLabels(lang string) bookingConfirmL {
	switch NormalizeLang(lang) {
	case "en":
		return bookingConfirmL{Hi: "Hi", Intro: "Your meeting is booked. Here are the details:", EventLbl: "Meeting", WhenLbl: "When", Cta: "Join Google Meet", Foot: "You're receiving this because you booked a meeting via Lead Cat.", DefaultName: "there"}
	case "kk":
		return bookingConfirmL{Hi: "Сәлем", Intro: "Кездесуіңіз брондалды. Толығырақ:", EventLbl: "Кездесу", WhenLbl: "Қашан", Cta: "Google Meet-ке қосылу", Foot: "Бұл хатты Lead Cat арқылы кездесу брондағаныңыз үшін алдыңыз.", DefaultName: "дос"}
	default:
		return bookingConfirmL{Hi: "Привет", Intro: "Ваша встреча забронирована. Детали:", EventLbl: "Встреча", WhenLbl: "Когда", Cta: "Подключиться к Google Meet", Foot: "Вы получили это письмо, потому что забронировали встречу через Lead Cat.", DefaultName: "друг"}
	}
}

// ----- Host notification -----

type BookingHostNotificationData struct {
	Language    string
	EventTitle  string
	BookerName  string
	BookerEmail string
	Date        string
	Time        string
	Tz          string
	MeetLink    string
}

func bookingHostSubject(lang, eventTitle string) string {
	switch NormalizeLang(lang) {
	case "en":
		return "New booking: " + eventTitle
	case "kk":
		return "Жаңа брондау: " + eventTitle
	default:
		return "Новая бронь: " + eventTitle
	}
}

type bookingHostL struct{ Hi, Intro, EventLbl, WhoLbl, WhenLbl, Cta, Foot string }

func bookingHostLabels(lang string) bookingHostL {
	switch NormalizeLang(lang) {
	case "en":
		return bookingHostL{Hi: "Hi", Intro: "You have a new booking.", EventLbl: "Meeting", WhoLbl: "Booked by", WhenLbl: "When", Cta: "Join Google Meet", Foot: "You're receiving this because someone booked time with you via Lead Cat."}
	case "kk":
		return bookingHostL{Hi: "Сәлем", Intro: "Сізде жаңа брондау бар.", EventLbl: "Кездесу", WhoLbl: "Брондаған", WhenLbl: "Қашан", Cta: "Google Meet-ке қосылу", Foot: "Бұл хатты біреу Lead Cat арқылы сізбен уақыт брондағандықтан алдыңыз."}
	default:
		return bookingHostL{Hi: "Привет", Intro: "У вас новая бронь.", EventLbl: "Встреча", WhoLbl: "Забронировал", WhenLbl: "Когда", Cta: "Подключиться к Google Meet", Foot: "Вы получили это письмо, потому что кто-то забронировал время через Lead Cat."}
	}
}

// ----- Shared render scaffolding -----

type bookingRow struct{ Label, Value string }

type bookingRenderData struct {
	Lang    string
	Hi      string
	Name    string // greeting target; empty for host email (generic intro)
	Intro   string
	Rows    []bookingRow
	Foot    string
	Button  template.HTML
	HasMeet bool
}

var bookingHTMLTemplate = template.Must(template.New("booking").Parse(`<!DOCTYPE html>
<html lang="{{.Lang}}">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="color-scheme" content="light"><title>Lead Cat</title></head>
<body style="margin:0; padding:0; background-color:#F1EADF;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#F1EADF;"><tr><td align="center" style="padding:28px 12px;">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px; background-color:#FBF7F0; border-radius:18px; overflow:hidden; font-family:Georgia,'Times New Roman',serif;">
<tr><td style="background-color:#E8714C; padding:26px 32px;"><span style="font-size:22px; font-weight:bold; color:#FFFFFF;">🐾&nbsp;Lead&nbsp;Cat</span></td></tr>
<tr><td style="padding:34px 32px 6px 32px;"><h1 style="margin:0; font-size:26px; line-height:1.2; font-weight:bold; color:#3A332E;">{{.Hi}}{{if .Name}}, {{.Name}}{{end}}!</h1></td></tr>
<tr><td style="padding:10px 32px 8px 32px;"><p style="margin:0; font-size:16px; line-height:1.6; color:#5C5249; font-family:Arial,Helvetica,sans-serif;">{{.Intro}}</p></td></tr>
{{range .Rows}}<tr><td style="padding:6px 32px;"><p style="margin:0; font-size:15px; line-height:1.5; color:#5C5249; font-family:Arial,Helvetica,sans-serif;"><strong style="color:#3A332E;">{{.Label}}:</strong> {{.Value}}</p></td></tr>
{{end}}{{if .HasMeet}}<tr><td align="center" style="padding:24px 32px 8px 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td align="center" bgcolor="#E8714C" style="border-radius:12px;">{{.Button}}</td></tr></table></td></tr>{{end}}
<tr><td style="background-color:#3A332E; padding:20px 32px;"><p style="margin:0; font-size:12px; line-height:1.5; color:#B7ACA0; font-family:Arial,Helvetica,sans-serif;">{{.Foot}}</p></td></tr>
</table></td></tr></table></body></html>`))

var bookingTextTemplate = texttemplate.Must(texttemplate.New("booking.txt").Parse(
	`{{.Hi}}{{if .Name}}, {{.Name}}{{end}}!

{{.Intro}}
{{range .Rows}}
{{.Label}}: {{.Value}}{{end}}
{{if .HasMeet}}
{{.MeetLine}}{{end}}

— Lead Cat
{{.Foot}}
`))

// textRenderData wraps bookingRenderData for the text template (which needs the raw meet link).
type textRenderData struct {
	bookingRenderData
	MeetLine string
}

func renderBooking(rd bookingRenderData, meetLink, cta string) (text, htmlOut string, err error) {
	var hb bytes.Buffer
	if err = bookingHTMLTemplate.Execute(&hb, rd); err != nil {
		return "", "", err
	}
	trd := textRenderData{bookingRenderData: rd}
	if rd.HasMeet {
		trd.MeetLine = cta + ": " + meetLink
	}
	var tb bytes.Buffer
	if err = bookingTextTemplate.Execute(&tb, trd); err != nil {
		return "", "", err
	}
	return tb.String(), hb.String(), nil
}

// RenderBookingConfirmation renders the booker-facing confirmation email.
func RenderBookingConfirmation(d BookingConfirmationData) (subject, text, html string, err error) {
	lang := NormalizeLang(d.Language)
	l := bookingConfirmLabels(lang)
	name := d.BookerName
	if name == "" {
		name = l.DefaultName
	}
	rows := []bookingRow{
		{Label: l.EventLbl, Value: d.EventTitle},
		{Label: l.WhenLbl, Value: d.Date + ", " + d.Time + " (" + d.Tz + ")"},
	}
	rd := bookingRenderData{
		Lang: lang, Hi: l.Hi, Name: name, Intro: l.Intro, Rows: rows, Foot: l.Foot,
		HasMeet: d.MeetLink != "",
	}
	if rd.HasMeet {
		rd.Button = BulletproofButton(d.MeetLink, l.Cta)
	}
	text, html, err = renderBooking(rd, d.MeetLink, l.Cta)
	if err != nil {
		return "", "", "", err
	}
	return bookingConfirmSubject(lang), text, html, nil
}

// RenderBookingHostNotification renders the host-facing new-booking email.
func RenderBookingHostNotification(d BookingHostNotificationData) (subject, text, html string, err error) {
	lang := NormalizeLang(d.Language)
	l := bookingHostLabels(lang)
	rows := []bookingRow{
		{Label: l.EventLbl, Value: d.EventTitle},
		{Label: l.WhoLbl, Value: d.BookerName + " <" + d.BookerEmail + ">"},
		{Label: l.WhenLbl, Value: d.Date + ", " + d.Time + " (" + d.Tz + ")"},
	}
	rd := bookingRenderData{
		Lang: lang, Hi: l.Hi, Name: "", Intro: l.Intro, Rows: rows, Foot: l.Foot,
		HasMeet: d.MeetLink != "",
	}
	if rd.HasMeet {
		rd.Button = BulletproofButton(d.MeetLink, l.Cta)
	}
	text, html, err = renderBooking(rd, d.MeetLink, l.Cta)
	if err != nil {
		return "", "", "", err
	}
	return bookingHostSubject(lang, d.EventTitle), text, html, nil
}
```

> Note: `bookingRenderData` has no `MeetLine` field but the text template references `{{.MeetLine}}` via the `textRenderData` wrapper — the HTML template never references `MeetLine`. `BulletproofButton` is the existing helper in `button.go`.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd apps/backend && go test ./internal/platform/emailtemplates/ -run TestRenderBooking -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Run the full package test + vet to confirm nothing else broke**

Run: `cd apps/backend && go test ./internal/platform/emailtemplates/ && go vet ./internal/platform/emailtemplates/`
Expected: ok, no vet errors.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/platform/emailtemplates/booking.go apps/backend/internal/platform/emailtemplates/booking_test.go
git commit -m "$(cat <<'EOF'
feat(booking): trilingual booking confirmation + host notification email templates

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 2: Send both emails from `SubmitBooking`

**Files:**
- Modify: `apps/backend/internal/application/booking_submit.go`
- Test: `apps/backend/internal/application/booking_submit_test.go`

**Interfaces:**
- Consumes (from Task 1): `emailtemplates.RenderBookingConfirmation`, `emailtemplates.RenderBookingHostNotification`, and their `*Data` structs.
- Consumes (existing): `Services.email EmailSender` (unexported; `SendMultipart(ctx, to, subject, text, html, listUnsub) error`), `Services.Log *zap.Logger`, the package helper `loadLoc(tz string) *time.Location`, `et.Timezone`, `host model.PlatformUser` (`.Email`, `.Language`, `.Timezone`).
- Produces: a new field `Language string` on `BookingRequest`.

- [ ] **Step 1: Write the failing test (extend `booking_submit_test.go`)**

Add a fake email sender, update `newSubmitServices` to wire it plus a no-op logger, and add behavior tests. Add this fake near the other fakes:

```go
type fakeEmailSender struct {
	sent    []sentEmail
	failOn  string // recipient substring that triggers an error; "" = never fail
}

type sentEmail struct{ to, subject, text, html string }

func (f *fakeEmailSender) SendMultipart(_ context.Context, to, subject, text, htmlBody, _ string) error {
	if f.failOn != "" && strings.Contains(to, f.failOn) {
		return errors.New("smtp boom")
	}
	f.sent = append(f.sent, sentEmail{to: to, subject: subject, text: text, html: htmlBody})
	return nil
}
```

Replace the existing `newSubmitServices` with a version that wires the mailer + logger and returns the fake sender:

```go
func newSubmitServices(store *submitFakeStore) (*Services, *submitFakeCalService, *fakeEmailSender) {
	cal := &submitFakeCalService{meetLink: "https://meet.google.com/abc-defg-hij"}
	prov := &submitFakeCalProvider{svc: cal}
	cmd := &command.Meetings{Store: store, Calendar: prov, Queue: submitFakeQueue{}}
	mailer := &fakeEmailSender{}
	s := &Services{Store: store, Commands: cmd, email: mailer, Log: zap.NewNop()}
	return s, cal, mailer
}
```

Then update existing callers of `newSubmitServices` to the 3-return form (use `_` for the mailer where unused, e.g. `s, cal, _ := newSubmitServices(store)` and `s, _, _ := newSubmitServices(store)`).

Add new tests:

```go
func TestSubmitBooking_SendsBookerAndHostEmails(t *testing.T) {
	store := &submitFakeStore{
		et:     submitEvent(),
		host:   model.PlatformUser{Email: "host@example.com", Language: "en", Timezone: "Asia/Almaty"},
		hostOK: true,
	}
	s, _, mailer := newSubmitServices(store)

	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "Visitor V", Email: "visitor@example.com", Start: freeMondaySlot(t), Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mailer.sent) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(mailer.sent))
	}
	var toBooker, toHost bool
	for _, e := range mailer.sent {
		if e.to == "visitor@example.com" {
			toBooker = true
			if !strings.Contains(e.html, "https://meet.google.com/abc-defg-hij") {
				t.Errorf("booker email missing meet link")
			}
			if strings.Contains(e.html, "host@example.com") {
				t.Errorf("booker email must not expose host email")
			}
		}
		if e.to == "host@example.com" {
			toHost = true
			if !strings.Contains(e.html, "visitor@example.com") {
				t.Errorf("host email should include booker email")
			}
		}
	}
	if !toBooker || !toHost {
		t.Fatalf("expected emails to both booker and host; booker=%v host=%v", toBooker, toHost)
	}
}

func TestSubmitBooking_EmailFailureDoesNotFailBooking(t *testing.T) {
	store := &submitFakeStore{
		et:     submitEvent(),
		host:   model.PlatformUser{Email: "host@example.com"},
		hostOK: true,
	}
	s, cal, mailer := newSubmitServices(store)
	mailer.failOn = "@example.com" // both sends error

	conf, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "Visitor V", Email: "visitor@example.com", Start: freeMondaySlot(t),
	})
	if err != nil {
		t.Fatalf("booking must succeed despite email failure, got %v", err)
	}
	if conf.MeetLink != cal.meetLink {
		t.Errorf("expected meet link returned")
	}
}

func TestSubmitBooking_NilMailerIsNoop(t *testing.T) {
	store := &submitFakeStore{et: submitEvent(), host: model.PlatformUser{Email: "host@example.com"}, hostOK: true}
	s, _, _ := newSubmitServices(store)
	s.email = nil // unconfigured mailer
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "visitor@example.com", Start: freeMondaySlot(t),
	})
	if err != nil {
		t.Fatalf("nil mailer must be a no-op, got %v", err)
	}
}
```

Add the imports `"strings"` and `"go.uber.org/zap"` to the test file if not already present (`errors` and `context` already are).

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/backend && go test ./internal/application/ -run TestSubmitBooking -v`
Expected: FAIL — `BookingRequest` has no field `Language`, and the new email assertions fail (no emails sent yet).

- [ ] **Step 3: Implement — add `Language` field and the send logic in `booking_submit.go`**

Add the field to the request struct:

```go
type BookingRequest struct {
	Name     string
	Email    string
	Start    time.Time
	Language string
}
```

Add this import to `booking_submit.go` (it currently imports `context`, `net/mail`, `strings`, `time`, `uuid`, and the local `model` package):

```go
	"github.com/luckyrogue/lead-cat/internal/platform/emailtemplates"
```

Replace the final `return BookingConfirmation{...}, nil` with: send the two emails (best-effort) before returning. Insert immediately after the successful `CreateMeeting` (`m, err := s.CreateMeeting(...)` / `if err != nil { ... }`):

```go
	s.sendBookingEmails(ctx, et, host, req, name, start, end, m.MeetLink)

	return BookingConfirmation{MeetLink: m.MeetLink, Start: start.UTC(), End: end.UTC()}, nil
}

// sendBookingEmails sends the booker confirmation and the host notification.
// Best-effort: a nil mailer is a no-op; render/send errors are logged and swallowed
// so a created booking never becomes an error response.
func (s *Services) sendBookingEmails(
	ctx context.Context,
	et model.BookingEventType,
	host model.PlatformUser,
	req BookingRequest,
	bookerName string,
	start, end time.Time,
	meetLink string,
) {
	if s.email == nil {
		return
	}

	// Booker email — times in the event-type timezone, page/browser language.
	bookerDate := start.Format("Mon, 02 Jan 2006")
	bookerTime := start.Format("15:04") + " – " + end.Format("15:04")
	if subject, text, htmlBody, rerr := emailtemplates.RenderBookingConfirmation(emailtemplates.BookingConfirmationData{
		Language:   req.Language,
		BookerName: bookerName,
		EventTitle: et.Title,
		Date:       bookerDate,
		Time:       bookerTime,
		Tz:         et.Timezone,
		MeetLink:   meetLink,
	}); rerr != nil {
		s.Log.Warn("booking_confirmation_render_failed", zap.Error(rerr))
	} else if serr := s.email.SendMultipart(ctx, req.Email, subject, text, htmlBody, ""); serr != nil {
		s.Log.Warn("booking_confirmation_send_failed", zap.Error(serr))
	}

	// Host email — times in the host timezone (fallback event-type tz), host language.
	hostTz := host.Timezone
	if hostTz == "" {
		hostTz = et.Timezone
	}
	hostLoc := loadLoc(hostTz)
	hStart := start.In(hostLoc)
	hEnd := end.In(hostLoc)
	if subject, text, htmlBody, rerr := emailtemplates.RenderBookingHostNotification(emailtemplates.BookingHostNotificationData{
		Language:    host.Language,
		EventTitle:  et.Title,
		BookerName:  bookerName,
		BookerEmail: req.Email,
		Date:        hStart.Format("Mon, 02 Jan 2006"),
		Time:        hStart.Format("15:04") + " – " + hEnd.Format("15:04"),
		Tz:          hostTz,
		MeetLink:    meetLink,
	}); rerr != nil {
		s.Log.Warn("booking_host_notification_render_failed", zap.Error(rerr))
	} else if serr := s.email.SendMultipart(ctx, host.Email, subject, text, htmlBody, ""); serr != nil {
		s.Log.Warn("booking_host_notification_send_failed", zap.Error(serr))
	}
}
```

Add the `zap` import to `booking_submit.go`:

```go
	"go.uber.org/zap"
```

> Note: `start`/`end` are already computed earlier in `SubmitBooking` as `start := req.Start.In(loc)` (event-type loc) and `end := start.Add(dur)` — pass those exact variables. `loadLoc` is the existing package helper used at the top of `SubmitBooking`.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd apps/backend && go test ./internal/application/ -run TestSubmitBooking -v`
Expected: PASS (all existing `TestSubmitBooking_*` plus the 3 new ones).

- [ ] **Step 5: Run the full application package test + vet + race**

Run: `cd apps/backend && go test -race ./internal/application/ && go vet ./internal/application/`
Expected: ok, no vet errors.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/application/booking_submit.go apps/backend/internal/application/booking_submit_test.go
git commit -m "$(cat <<'EOF'
feat(booking): send booker confirmation + host notification on submit (best-effort)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 3: Plumb visitor language end-to-end (handler + frontend)

Thin mapping on both ends; no new behavior beyond passing a string. Verified by `go build`/`go vet` (Go) and `pnpm typecheck`/`build` (frontend) — consistent with the repo convention that thin delivery/frontend layers are build-verified rather than unit-tested (the behavior is covered by Task 2's application tests). The booker-language default (`ru`) already works without this task; this task makes the visitor's browser language take effect.

**Files:**
- Modify: `apps/backend/internal/delivery/http/handlers/public_booking_submit.go`
- Modify: `apps/admin/app/routes/book.$slug.form.tsx`

**Interfaces:**
- Consumes (from Task 2): `application.BookingRequest.Language`.

- [ ] **Step 1: Add `language` to the handler body struct and forward it**

In `public_booking_submit.go`, change the body struct and the `SubmitBooking` call:

```go
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Start    string `json:"start"`
		Language string `json:"language"`
	}
```

```go
	conf, err := a.App.SubmitBooking(c.UserContext(), slug, application.BookingRequest{
		Name: body.Name, Email: body.Email, Start: start, Language: body.Language,
	})
```

- [ ] **Step 2: Verify the backend builds + vets**

Run: `cd apps/backend && go build ./... && go vet ./internal/delivery/http/handlers/`
Expected: builds clean, no vet errors.

- [ ] **Step 3: Add `language` to the frontend submit body**

In `apps/admin/app/routes/book.$slug.form.tsx`, the `handleSubmit` `fetch` body currently is:

```ts
        body: JSON.stringify({
          name: name.trim(),
          email: email.trim(),
          start: selectedSlot.start,
        }),
```

Change it to include the visitor's browser language (primary subtag; the backend `NormalizeLang` maps anything outside ru/en/kk to ru):

```ts
        body: JSON.stringify({
          name: name.trim(),
          email: email.trim(),
          start: selectedSlot.start,
          language:
            typeof navigator !== "undefined"
              ? navigator.language.split("-")[0]
              : "",
        }),
```

- [ ] **Step 4: Verify the frontend typechecks + builds**

Run: `pnpm --filter ./apps/admin typecheck && pnpm --filter ./apps/admin build`
Expected: both succeed.

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/delivery/http/handlers/public_booking_submit.go "apps/admin/app/routes/book.\$slug.form.tsx"
git commit -m "$(cat <<'EOF'
feat(booking): forward visitor browser language to booking confirmation email

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- New `emailtemplates/booking.go`, two renderers + data structs → Task 1. ✓
- Send both emails from `SubmitBooking`, best-effort, nil-guard, log + continue → Task 2 (`sendBookingEmails`). ✓
- Booker times in `et.Timezone`, host times in `host.Timezone`?? `et.Timezone`, tz label shown → Task 2 (`bookerDate/Time` + `Tz: et.Timezone`; `hostTz` fallback + `Tz: hostTz`). ✓
- Booker email omits host email; host email shows booker name+email → Task 1 row construction + Task 2 test assertions. ✓
- No host name referenced (PlatformUser has none) → Task 1 uses `EventTitle` only for booker, booker name/email for host. ✓
- HTML escaping of user content → `html/template` + `TestRenderBooking_EscapesUserContent`. ✓
- Locale plumbing: `BookingRequest.Language` + handler `language` + frontend → Tasks 2 & 3. ✓
- Trilingual ru/en/kk via `NormalizeLang`, default ru → Task 1 labels/subjects + `TestRenderBookingConfirmation_DefaultsToRu`. ✓
- Unit tests for renderers + send behavior → Tasks 1 & 2. ✓
- Out of scope (SendUpdates, .ics, bookings table) → not implemented. ✓

**Resolved during planning (deviation from spec, noted):** the spec assumed a "booking page active locale"; the `/book/:slug` page is English-only with no i18n provider, so the visitor language is taken from `navigator.language` instead. Same intent (localize to the visitor), pragmatic source.

**Placeholder scan:** No TBD/TODO; every code step shows complete code. The two `> Note` blocks clarify real mechanics (the `MeetLine`/`textRenderData` split; reusing `start`/`end`/`loadLoc`), not deferred work.

**Type consistency:** `RenderBookingConfirmation`/`RenderBookingHostNotification` signatures and the `BookingConfirmationData`/`BookingHostNotificationData` field names are identical across Task 1 (definition), Task 1 tests, and Task 2 (call site). `BookingRequest.Language` is defined in Task 2 and consumed in Task 3. `newSubmitServices` 3-return form is updated consistently (Step 1 of Task 2 mandates updating all existing callers).
