package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	googleprobe "github.com/luckyrogue/lead-cat/internal/infrastructure/calendar/google"
)

// Sentinel errors mapped to handler-level error codes.
var (
	ErrGoogleSAInvalid             = errors.New("google_sa_invalid")
	ErrGoogleSubjectInvalid        = errors.New("google_subject_invalid")
	ErrGoogleCalendarNotAccessible = errors.New("google_calendar_not_accessible")
	ErrGoogleAPIDisabled           = errors.New("google_api_disabled")
)

// GoogleVerifyResult is what the handler returns on success.
type GoogleVerifyResult struct {
	OK              bool   `json:"ok"`
	CalendarSummary string `json:"calendar_summary,omitempty"`
	TimeZone        string `json:"time_zone,omitempty"`
	AccessRole      string `json:"access_role,omitempty"`
}

// VerifyGoogleIntegration reads the organization's stored Google config,
// decrypts the SA JSON, runs Probe, and maps errors to public codes.
func (s *Services) VerifyGoogleIntegration(ctx context.Context, organizationID uuid.UUID) (*GoogleVerifyResult, error) {
	enc, subject, calendarID, err := s.Store.GetGoogleConfig(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 || subject == "" {
		return nil, ErrGoogleNotConfigured
	}
	saJSON, err := s.Cipher.Decrypt(enc)
	if err != nil {
		return nil, ErrGoogleSAInvalid
	}
	if calendarID == "" {
		calendarID = "primary"
	}
	cal, err := googleprobe.Probe(ctx, saJSON, subject, calendarID)
	if e := mapProbeError(err); e != nil {
		return nil, e
	}
	return &GoogleVerifyResult{
		OK:              true,
		CalendarSummary: cal.Summary,
		TimeZone:        cal.TimeZone,
	}, nil
}

// mapProbeError maps probe sentinels to handler-level errors. Unknown errors
// default to ErrGoogleCalendarNotAccessible (the most generic Google-side failure).
// nil → nil.
func mapProbeError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, googleprobe.ErrJSONParse):
		return ErrGoogleSAInvalid
	case errors.Is(err, googleprobe.ErrAPIDisabled):
		return ErrGoogleAPIDisabled
	case errors.Is(err, googleprobe.ErrSubject):
		return ErrGoogleSubjectInvalid
	case errors.Is(err, googleprobe.ErrCalendar):
		return ErrGoogleCalendarNotAccessible
	default:
		return ErrGoogleCalendarNotAccessible
	}
}
