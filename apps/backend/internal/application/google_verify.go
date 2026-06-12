package application

import (
	"context"

	"github.com/google/uuid"
)

// GoogleVerifyResult is what the handler returns on success.
type GoogleVerifyResult struct {
	OK              bool   `json:"ok"`
	CalendarSummary string `json:"calendar_summary,omitempty"`
	TimeZone        string `json:"time_zone,omitempty"`
	AccessRole      string `json:"access_role,omitempty"`
}

// VerifyGoogleIntegration reads the organization's stored Google config,
// decrypts the SA JSON, and probes it via the GoogleProber port. Probe failures
// surface as the calendar package's classified sentinels.
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
	res, err := s.GoogleProber.Probe(ctx, saJSON, subject, calendarID)
	if err != nil {
		return nil, err
	}
	return &GoogleVerifyResult{
		OK:              true,
		CalendarSummary: res.Summary,
		TimeZone:        res.TimeZone,
	}, nil
}
