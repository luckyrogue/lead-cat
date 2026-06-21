package application

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type submitFakeStore struct {
	Repository
	et          model.BookingEventType
	etErr       error
	host        model.PlatformUser
	hostOK      bool
	overlapping []model.Meeting

	createdMeeting model.Meeting
	createdInput   model.Meeting
}

func (f *submitFakeStore) GetBookingEventTypeBySlug(_ context.Context, _ string) (model.BookingEventType, error) {
	return f.et, f.etErr
}

func (f *submitFakeStore) GetPlatformUserByID(_ context.Context, _ uuid.UUID) (model.PlatformUser, bool, error) {
	return f.host, f.hostOK, nil
}

func (f *submitFakeStore) ListMeetingsOverlapping(_ context.Context, _ []string, _, _ time.Time) ([]model.Meeting, error) {
	return f.overlapping, nil
}

func (f *submitFakeStore) ListParticipants(_ context.Context, _ uuid.UUID) ([]model.MeetingParticipant, error) {
	return nil, nil
}

func (f *submitFakeStore) GetUserByID(_ context.Context, id uuid.UUID) (model.User, error) {
	if id == f.et.HostUserID {
		return model.User{ID: id, Email: f.host.Email}, nil
	}
	return model.User{ID: id}, nil
}

func (f *submitFakeStore) GetOrganization(_ context.Context, id uuid.UUID) (model.Organization, error) {
	return model.Organization{ID: id, TZ: "Asia/Almaty"}, nil
}

func (f *submitFakeStore) CreateMeeting(_ context.Context, m model.Meeting) (model.Meeting, error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	f.createdInput = m
	f.createdMeeting = m
	return m, nil
}

func (f *submitFakeStore) AddParticipants(_ context.Context, _ uuid.UUID, _ []model.MeetingParticipant) error {
	return nil
}

func (f *submitFakeStore) SearchEmployeesGlobal(_ context.Context, _ string) ([]model.Employee, error) {
	return nil, nil
}

type submitFakeCalProvider struct{ svc *submitFakeCalService }

func (p *submitFakeCalProvider) For(_ context.Context, _ uuid.UUID, _ string) (docalendar.Service, error) {
	return p.svc, nil
}

type submitFakeCalService struct {
	meetLink string
	created  []docalendar.CalendarEvent
}

func (s *submitFakeCalService) CreateEvent(_ context.Context, e docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	s.created = append(s.created, e)
	return docalendar.CalendarResult{EventID: "evt-1", MeetLink: s.meetLink}, nil
}

func (s *submitFakeCalService) UpdateEvent(_ context.Context, _ string, _ docalendar.CalendarEvent) error {
	return nil
}

func (s *submitFakeCalService) UpdateAttendees(_ context.Context, _ string, _ []string) error {
	return nil
}

func (s *submitFakeCalService) DeleteEvent(_ context.Context, _ string) error { return nil }

type fakeEmailSender struct {
	sent   []sentEmail
	failOn string // recipient substring that triggers an error; "" = never fail
}

type sentEmail struct{ to, subject, text, html string }

func (f *fakeEmailSender) SendMultipart(_ context.Context, to, subject, text, htmlBody, _ string) error {
	if f.failOn != "" && strings.Contains(to, f.failOn) {
		return errors.New("smtp boom")
	}
	f.sent = append(f.sent, sentEmail{to: to, subject: subject, text: text, html: htmlBody})
	return nil
}

type submitFakeQueue struct{}

func (submitFakeQueue) EnqueueMeetingCreated(_ context.Context, _, _ uuid.UUID) error   { return nil }
func (submitFakeQueue) EnqueueMeetingUpdated(_ context.Context, _, _ uuid.UUID) error   { return nil }
func (submitFakeQueue) EnqueueMeetingCancelled(_ context.Context, _, _ uuid.UUID) error { return nil }

func submitEvent() model.BookingEventType {
	return model.BookingEventType{
		ID:               uuid.MustParse("cccccccc-0000-0000-0000-000000000003"),
		HostUserID:       uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"),
		OrganizationID:   uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002"),
		Slug:             "intro-call-abc123",
		Title:            "Intro Call",
		Description:      "30 min intro",
		DurationMins:     30,
		Active:           true,
		Timezone:         "Asia/Almaty",
		AvailWeekdays:    []int{1, 2, 3, 4, 5},
		AvailStartMinute: 540,
		AvailEndMinute:   1020,
	}
}

func newSubmitServices(store *submitFakeStore) (*Services, *submitFakeCalService, *fakeEmailSender) {
	cal := &submitFakeCalService{meetLink: "https://meet.google.com/abc-defg-hij"}
	prov := &submitFakeCalProvider{svc: cal}
	cmd := &command.Meetings{Store: store, Calendar: prov, Queue: submitFakeQueue{}}
	mailer := &fakeEmailSender{}
	s := &Services{Store: store, Commands: cmd, email: mailer, Log: zap.NewNop()}
	return s, cal, mailer
}

func freeMondaySlot(t *testing.T) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Fatalf("load loc: %v", err)
	}
	return time.Date(2099, 6, 22, 10, 0, 0, 0, loc).UTC()
}

func TestSubmitBooking_HappyPath(t *testing.T) {
	store := &submitFakeStore{
		et:     submitEvent(),
		host:   model.PlatformUser{Email: "host@example.com"},
		hostOK: true,
	}
	s, cal, _ := newSubmitServices(store)

	conf, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name:  "Visitor V",
		Email: "visitor@example.com",
		Start: freeMondaySlot(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf.MeetLink != cal.meetLink {
		t.Errorf("expected meet link %q, got %q", cal.meetLink, conf.MeetLink)
	}
	if store.createdInput.Recurrence != "once" {
		t.Errorf("expected once recurrence, got %q", store.createdInput.Recurrence)
	}
	if store.createdInput.Name == "" {
		t.Errorf("expected non-empty meeting name (title)")
	}
	if len(cal.created) != 1 {
		t.Fatalf("expected 1 calendar event, got %d", len(cal.created))
	}
	ev := cal.created[0]
	foundVisitor := false
	for _, e := range ev.AttendeeEmails {
		if e == "visitor@example.com" {
			foundVisitor = true
		}
	}
	if !foundVisitor {
		t.Errorf("expected visitor email among attendees, got %v", ev.AttendeeEmails)
	}
	loc, _ := time.LoadLocation("Asia/Almaty")
	if got := ev.Start.In(loc); got.Hour() != 10 || got.Minute() != 0 {
		t.Errorf("expected event start 10:00 Almaty, got %v", got)
	}
}

func TestSubmitBooking_InvalidEmail(t *testing.T) {
	store := &submitFakeStore{et: submitEvent(), host: model.PlatformUser{Email: "host@example.com"}, hostOK: true}
	s, _, _ := newSubmitServices(store)
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "not-an-email", Start: freeMondaySlot(t),
	})
	if !errors.Is(err, model.ErrInvalidBooking) {
		t.Fatalf("expected ErrInvalidBooking, got %v", err)
	}
}

func TestSubmitBooking_PastStart(t *testing.T) {
	store := &submitFakeStore{et: submitEvent(), host: model.PlatformUser{Email: "host@example.com"}, hostOK: true}
	s, _, _ := newSubmitServices(store)
	loc, _ := time.LoadLocation("Asia/Almaty")
	past := time.Date(2000, 6, 22, 10, 0, 0, 0, loc).UTC()
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "visitor@example.com", Start: past,
	})
	if !errors.Is(err, model.ErrInvalidBooking) {
		t.Fatalf("expected ErrInvalidBooking, got %v", err)
	}
}

func TestSubmitBooking_OutsideWindow(t *testing.T) {
	store := &submitFakeStore{et: submitEvent(), host: model.PlatformUser{Email: "host@example.com"}, hostOK: true}
	s, _, _ := newSubmitServices(store)
	loc, _ := time.LoadLocation("Asia/Almaty")
	early := time.Date(2099, 6, 22, 7, 0, 0, 0, loc).UTC()
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "visitor@example.com", Start: early,
	})
	if !errors.Is(err, model.ErrInvalidBooking) {
		t.Fatalf("expected ErrInvalidBooking, got %v", err)
	}
}

func TestSubmitBooking_WrongWeekday(t *testing.T) {
	store := &submitFakeStore{et: submitEvent(), host: model.PlatformUser{Email: "host@example.com"}, hostOK: true}
	s, _, _ := newSubmitServices(store)
	loc, _ := time.LoadLocation("Asia/Almaty")
	sat := time.Date(2099, 6, 20, 10, 0, 0, 0, loc).UTC()
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "visitor@example.com", Start: sat,
	})
	if !errors.Is(err, model.ErrInvalidBooking) {
		t.Fatalf("expected ErrInvalidBooking, got %v", err)
	}
}

func TestSubmitBooking_Conflict(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2099, 6, 22, 10, 0, 0, 0, loc)
	store := &submitFakeStore{
		et:     submitEvent(),
		host:   model.PlatformUser{Email: "host@example.com"},
		hostOK: true,
		overlapping: []model.Meeting{
			{
				ID:              uuid.New(),
				StartsAt:        start.UTC(),
				EndsAt:          start.Add(30 * time.Minute).UTC(),
				OrganizerUserID: ptrUUID(submitEvent().HostUserID),
			},
		},
	}
	s, _, _ := newSubmitServices(store)
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "visitor@example.com", Start: start.UTC(),
	})
	if !errors.Is(err, model.ErrSlotTaken) {
		t.Fatalf("expected ErrSlotTaken, got %v", err)
	}
}

func TestSubmitBooking_UnknownSlug(t *testing.T) {
	store := &submitFakeStore{etErr: sql.ErrNoRows}
	s, _, _ := newSubmitServices(store)
	_, err := s.SubmitBooking(context.Background(), "no-such", BookingRequest{
		Name: "V", Email: "visitor@example.com", Start: freeMondaySlot(t),
	})
	if !model.IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}

func TestSubmitBooking_Inactive(t *testing.T) {
	et := submitEvent()
	et.Active = false
	store := &submitFakeStore{et: et, host: model.PlatformUser{Email: "host@example.com"}, hostOK: true}
	s, _, _ := newSubmitServices(store)
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "visitor@example.com", Start: freeMondaySlot(t),
	})
	if !errors.Is(err, model.ErrInvalidBooking) {
		t.Fatalf("expected ErrInvalidBooking for inactive, got %v", err)
	}
}

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

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
