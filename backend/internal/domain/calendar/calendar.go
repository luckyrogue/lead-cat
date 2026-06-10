package calendar

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// CalendarEvent is a calendar event to create (transport-agnostic).
type CalendarEvent struct {
	Title          string
	Description    string
	Start          time.Time
	End            time.Time
	AttendeeEmails []string
}

// CalendarResult is what the calendar backend returns after creation.
type CalendarResult struct {
	EventID  string
	MeetLink string
}

// Service is the port for the calendar backend (Google Calendar in
// production, a stub in tests/local). Implemented in infrastructure/calendar/*.
type Service interface {
	CreateEvent(ctx context.Context, e CalendarEvent) (CalendarResult, error)
	UpdateEvent(ctx context.Context, eventID string, e CalendarEvent) error
	UpdateAttendees(ctx context.Context, eventID string, emails []string) error
	DeleteEvent(ctx context.Context, eventID string) error
}

// Provider resolves the Service to use for a given organization.
type Provider interface {
	For(ctx context.Context, organizationID uuid.UUID) (Service, error)
}

// ErrNotConfigured is returned when an organization has no Google credentials.
var ErrNotConfigured = errors.New("google not configured")
