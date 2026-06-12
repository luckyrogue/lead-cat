package calendar

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type CalendarEvent struct {
	Title          string
	Description    string
	Start          time.Time
	End            time.Time
	AttendeeEmails []string
}

type CalendarResult struct {
	EventID  string
	MeetLink string
}

type Service interface {
	CreateEvent(ctx context.Context, e CalendarEvent) (CalendarResult, error)
	UpdateEvent(ctx context.Context, eventID string, e CalendarEvent) error
	UpdateAttendees(ctx context.Context, eventID string, emails []string) error
	DeleteEvent(ctx context.Context, eventID string) error
}

type Provider interface {
	For(ctx context.Context, organizationID uuid.UUID) (Service, error)
}

type ProbeResult struct {
	Summary  string
	TimeZone string
}

type Prober interface {
	Probe(ctx context.Context, saJSON, subject, calendarID string) (ProbeResult, error)
}

var ErrNotConfigured = errors.New("google not configured")

var (
	ErrProbeSAInvalid   = errors.New("google_sa_invalid")
	ErrProbeAPIDisabled = errors.New("google_api_disabled")
	ErrProbeSubject     = errors.New("google_subject_invalid")
	ErrProbeCalendar    = errors.New("google_calendar_not_accessible")
)
