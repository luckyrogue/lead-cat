package google

import (
	"context"
	"errors"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type Prober struct{}

func NewProber() Prober { return Prober{} }

func (Prober) Probe(ctx context.Context, saJSON, subject, calendarID string) (docalendar.ProbeResult, error) {
	cal, err := Probe(ctx, saJSON, subject, calendarID)
	if e := mapProbeError(err); e != nil {
		return docalendar.ProbeResult{}, e
	}
	return docalendar.ProbeResult{Summary: cal.Summary, TimeZone: cal.TimeZone}, nil
}

func mapProbeError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrJSONParse):
		return docalendar.ErrProbeSAInvalid
	case errors.Is(err, ErrAPIDisabled):
		return docalendar.ErrProbeAPIDisabled
	case errors.Is(err, ErrSubject):
		return docalendar.ErrProbeSubject
	case errors.Is(err, ErrCalendar):
		return docalendar.ErrProbeCalendar
	default:
		return docalendar.ErrProbeCalendar
	}
}
