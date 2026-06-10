package meetingedit

import "testing"

func TestParseTimeRange(t *testing.T) {
	for _, in := range []string{"14:00–15:00", "14:00-15:00"} {
		s, e, err := parseTimeRange(in)
		if err != nil || s != "14:00" || e != "15:00" {
			t.Fatalf("%q -> %q %q err %v", in, s, e, err)
		}
	}
	for _, bad := range []string{"", "14:00", "9:00-10:00", "15:00-14:00", "14:00-14:00"} {
		if _, _, err := parseTimeRange(bad); err == nil {
			t.Fatalf("%q: expected error", bad)
		}
	}
}

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
		"",
		"2026-06-01",
		"2026/06/01 14:00-15:00",
		"2026-06-01 14:00",
		"2026-06-01 9:00-10:00",
		"2026-06-01 15:00-14:00",
		"2026-06-01 14:00-14:00",
	}
	for _, in := range bad {
		if _, _, _, err := parseDateTime(in); err == nil {
			t.Fatalf("%q: expected error", in)
		}
	}
}
