package google

import (
	"context"
	"errors"
	"fmt"
	"strings"

	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var (
	ErrJSONParse   = errors.New("sa_json_parse")
	ErrAPIDisabled = errors.New("calendar_api_disabled")
	ErrSubject     = errors.New("subject_impersonation")
	ErrCalendar    = errors.New("calendar_not_accessible")
)

func Probe(ctx context.Context, saJSON, subject, calendarID string) (*calendar.Calendar, error) {
	cfg, err := googleoauth.JWTConfigFromJSON([]byte(saJSON), calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJSONParse, err)
	}
	cfg.Subject = subject
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(cfg.Client(ctx)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPIDisabled, err)
	}
	cal, err := svc.Calendars.Get(calendarID).Context(ctx).Do()
	if err != nil {
		if isJSONParseErr(err) {
			return nil, fmt.Errorf("%w: %v", ErrJSONParse, err)
		}
		if isGoogleAPIDisabled(err) {
			return nil, fmt.Errorf("%w: %v", ErrAPIDisabled, err)
		}
		if isImpersonationFail(err) {
			return nil, fmt.Errorf("%w: %v", ErrSubject, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrCalendar, err)
	}
	return cal, nil
}

func isJSONParseErr(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "private key should be a PEM") ||
		strings.Contains(msg, "parse error:") && strings.Contains(msg, "asn1")
}

func isGoogleAPIDisabled(err error) bool {
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Code != 403 {
		return false
	}
	for _, d := range apiErr.Errors {
		if d.Reason == "accessNotConfigured" {
			return true
		}
	}
	return strings.Contains(apiErr.Message, "has not been used") || strings.Contains(apiErr.Message, "is disabled")
}

func isImpersonationFail(err error) bool {
	msg := err.Error()
	if strings.Contains(msg, "unauthorized_client") {
		return true
	}
	if strings.Contains(msg, "Not Authorized to access this resource") {
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 401 {
		return true
	}
	return false
}
