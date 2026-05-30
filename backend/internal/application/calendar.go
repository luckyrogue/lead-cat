package application

import (
	"context"
	"time"
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
	DeleteEvent(ctx context.Context, eventID string) error
}
