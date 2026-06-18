package application

import (
	"context"

	"github.com/google/uuid"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type CalendarEvent = docalendar.CalendarEvent

type CalendarResult = docalendar.CalendarResult

type CalendarService = docalendar.Service

var ErrGoogleNotConfigured = docalendar.ErrNotConfigured

type CalendarProvider interface {
	For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (CalendarService, error)
}

type GoogleProber = docalendar.Prober

type GoogleProbeResult = docalendar.ProbeResult

var (
	ErrGoogleSAInvalid             = docalendar.ErrProbeSAInvalid
	ErrGoogleAPIDisabled           = docalendar.ErrProbeAPIDisabled
	ErrGoogleSubjectInvalid        = docalendar.ErrProbeSubject
	ErrGoogleCalendarNotAccessible = docalendar.ErrProbeCalendar
)
