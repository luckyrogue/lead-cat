package application

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

// CalendarService is the port for the calendar backend (Google Calendar in
// production, a stub in tests/local). Implemented in infrastructure/calendar/*.
type CalendarService interface {
	CreateEvent(ctx context.Context, e CalendarEvent) (CalendarResult, error)
	UpdateEvent(ctx context.Context, eventID string, e CalendarEvent) error
	UpdateAttendees(ctx context.Context, eventID string, emails []string) error
	DeleteEvent(ctx context.Context, eventID string) error
}

// ErrGoogleNotConfigured is returned when a workspace has no Google credentials.
var ErrGoogleNotConfigured = errors.New("google not configured")

// ErrInvalidInput marks client-side input errors (bad fields or times) so the
// HTTP layer can map them to 400 instead of 500.
var ErrInvalidInput = errors.New("invalid input")

// CalendarProvider resolves the CalendarService to use for a given workspace.
type CalendarProvider interface {
	For(ctx context.Context, workspaceID uuid.UUID) (CalendarService, error)
}
