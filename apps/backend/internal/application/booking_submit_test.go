package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

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

func newSubmitServices(store *submitFakeStore) (*Services, *submitFakeCalService) {
	cal := &submitFakeCalService{meetLink: "https://meet.google.com/abc-defg-hij"}
	prov := &submitFakeCalProvider{svc: cal}
	cmd := &command.Meetings{Store: store, Calendar: prov, Queue: submitFakeQueue{}}
	s := &Services{Store: store, Commands: cmd}
	return s, cal
}

// freeMondaySlot returns 2026-06-22 (Monday) 10:00 Asia/Almaty as a future UTC instant.
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
	s, cal := newSubmitServices(store)

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
	// Event built in the event timezone: 10:00 Almaty.
	loc, _ := time.LoadLocation("Asia/Almaty")
	if got := ev.Start.In(loc); got.Hour() != 10 || got.Minute() != 0 {
		t.Errorf("expected event start 10:00 Almaty, got %v", got)
	}
}

func TestSubmitBooking_InvalidEmail(t *testing.T) {
	store := &submitFakeStore{et: submitEvent(), host: model.PlatformUser{Email: "host@example.com"}, hostOK: true}
	s, _ := newSubmitServices(store)
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "not-an-email", Start: freeMondaySlot(t),
	})
	if !errors.Is(err, model.ErrInvalidBooking) {
		t.Fatalf("expected ErrInvalidBooking, got %v", err)
	}
}

func TestSubmitBooking_PastStart(t *testing.T) {
	store := &submitFakeStore{et: submitEvent(), host: model.PlatformUser{Email: "host@example.com"}, hostOK: true}
	s, _ := newSubmitServices(store)
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
	s, _ := newSubmitServices(store)
	loc, _ := time.LoadLocation("Asia/Almaty")
	// 07:00 Almaty is before the 09:00 window start.
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
	s, _ := newSubmitServices(store)
	loc, _ := time.LoadLocation("Asia/Almaty")
	// 2099-06-20 is a Saturday — not in Mon-Fri.
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
	s, _ := newSubmitServices(store)
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "visitor@example.com", Start: start.UTC(),
	})
	if !errors.Is(err, model.ErrSlotTaken) {
		t.Fatalf("expected ErrSlotTaken, got %v", err)
	}
}

func TestSubmitBooking_UnknownSlug(t *testing.T) {
	store := &submitFakeStore{etErr: sql.ErrNoRows}
	s, _ := newSubmitServices(store)
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
	s, _ := newSubmitServices(store)
	_, err := s.SubmitBooking(context.Background(), "intro-call-abc123", BookingRequest{
		Name: "V", Email: "visitor@example.com", Start: freeMondaySlot(t),
	})
	if !errors.Is(err, model.ErrInvalidBooking) {
		t.Fatalf("expected ErrInvalidBooking for inactive, got %v", err)
	}
}

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
