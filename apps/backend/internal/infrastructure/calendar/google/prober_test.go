package google

import (
	"errors"
	"testing"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

func TestMapProbeError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want error
	}{
		{"ErrJSONParse", ErrJSONParse, docalendar.ErrProbeSAInvalid},
		{"wrapped JSON parse", &probeWrap{inner: ErrJSONParse}, docalendar.ErrProbeSAInvalid},
		{"ErrAPIDisabled", ErrAPIDisabled, docalendar.ErrProbeAPIDisabled},
		{"ErrSubject", ErrSubject, docalendar.ErrProbeSubject},
		{"ErrCalendar", ErrCalendar, docalendar.ErrProbeCalendar},
		{"unknown", errors.New("network exploded"), docalendar.ErrProbeCalendar},
		{"nil", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapProbeError(tc.in)
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
