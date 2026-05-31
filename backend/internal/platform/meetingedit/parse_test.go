package meetingedit

import "testing"

func TestParseDateTime_OK(t *testing.T) {
	for _, in := range []string{"2026-06-01 14:00–15:00", "2026-06-01 14:00-15:00"} {
		d, s, e, err := parseDateTime(in)
		if err != nil {
			t.Fatalf("%q: unexpected error %v", in, err)
		}
		if d != "2026-06-01" || s != "14:00" || e != "15:00" {
			t.Fatalf("%q -> %q %q %q", in, d, s, e)
		}
	}
}

func TestParseDateTime_Errors(t *testing.T) {
	bad := []string{
		"",                       // empty
		"2026-06-01",             // no time range
		"2026/06/01 14:00-15:00", // bad date
		"2026-06-01 14:00",       // single time
		"2026-06-01 9:00-10:00",  // not HH:MM
		"2026-06-01 15:00-14:00", // end before start
		"2026-06-01 14:00-14:00", // equal
	}
	for _, in := range bad {
		if _, _, _, err := parseDateTime(in); err == nil {
			t.Fatalf("%q: expected error", in)
		}
	}
}
