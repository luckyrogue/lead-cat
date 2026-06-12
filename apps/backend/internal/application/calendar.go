package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

// CalendarEvent is a calendar event to create (transport-agnostic).
// Type alias for domain/calendar.CalendarEvent.
type CalendarEvent = docalendar.CalendarEvent

// CalendarResult is what the calendar backend returns after creation.
type CalendarResult = docalendar.CalendarResult

// CalendarService is the port for the calendar backend (Google Calendar in
// production, a stub in tests/local). Implemented in infrastructure/calendar/*.
type CalendarService = docalendar.Service

// ErrGoogleNotConfigured is returned when an organization has no Google credentials.
var ErrGoogleNotConfigured = docalendar.ErrNotConfigured

// ErrInvalidInput marks client-side input errors (bad fields or times) so the
// HTTP layer can map them to 400 instead of 500.
var ErrInvalidInput = errors.New("invalid input")

// CalendarProvider resolves the CalendarService to use for a given organization.
type CalendarProvider interface {
	For(ctx context.Context, organizationID uuid.UUID) (CalendarService, error)
}

// GoogleProber verifies stored Google credentials. Implemented in
// infrastructure/calendar/google.
type GoogleProber = docalendar.Prober

// GoogleProbeResult is the calendar metadata from a successful probe.
type GoogleProbeResult = docalendar.ProbeResult

// Google credential-probe sentinels (aliased from domain/calendar) — the HTTP
// layer maps these to public error codes.
var (
	ErrGoogleSAInvalid             = docalendar.ErrProbeSAInvalid
	ErrGoogleAPIDisabled           = docalendar.ErrProbeAPIDisabled
	ErrGoogleSubjectInvalid        = docalendar.ErrProbeSubject
	ErrGoogleCalendarNotAccessible = docalendar.ErrProbeCalendar
)
