package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	docalendar "github.com/Jaryq-Lab/notify-bot/internal/domain/calendar"
)

// CalendarEvent is a calendar event to create (transport-agnostic).
// Type alias for domain/calendar.CalendarEvent.
type CalendarEvent = docalendar.CalendarEvent

// CalendarResult is what the calendar backend returns after creation.
type CalendarResult = docalendar.CalendarResult

// CalendarService is the port for the calendar backend (Google Calendar in
// production, a stub in tests/local). Implemented in infrastructure/calendar/*.
type CalendarService = docalendar.Service

// ErrGoogleNotConfigured is returned when a workspace has no Google credentials.
var ErrGoogleNotConfigured = docalendar.ErrNotConfigured

// ErrInvalidInput marks client-side input errors (bad fields or times) so the
// HTTP layer can map them to 400 instead of 500.
var ErrInvalidInput = errors.New("invalid input")

// CalendarProvider resolves the CalendarService to use for a given workspace.
type CalendarProvider interface {
	For(ctx context.Context, workspaceID uuid.UUID) (CalendarService, error)
}
