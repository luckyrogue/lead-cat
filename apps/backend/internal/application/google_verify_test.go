package application_test

import (
	"errors"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/application"
	googleprobe "github.com/luckyrogue/lead-cat/internal/infrastructure/calendar/google"
)

func TestMapProbeError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want error
	}{
		{"ErrJSONParse", googleprobe.ErrJSONParse, application.ErrGoogleSAInvalid},
		{"wrapped JSON parse", &probeWrap{inner: googleprobe.ErrJSONParse}, application.ErrGoogleSAInvalid},
		{"ErrAPIDisabled", googleprobe.ErrAPIDisabled, application.ErrGoogleAPIDisabled},
		{"ErrSubject", googleprobe.ErrSubject, application.ErrGoogleSubjectInvalid},
		{"ErrCalendar", googleprobe.ErrCalendar, application.ErrGoogleCalendarNotAccessible},
		{"unknown", errors.New("network exploded"), application.ErrGoogleCalendarNotAccessible},
		{"nil", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := application.MapProbeError(tc.in)
			if tc.want == nil {
				if got != nil {
					t.Fatalf("want nil, got %v", got)
				}
				return
			}
			if !errors.Is(got, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

type probeWrap struct{ inner error }

func (p *probeWrap) Error() string { return "wrapped: " + p.inner.Error() }
func (p *probeWrap) Unwrap() error { return p.inner }
