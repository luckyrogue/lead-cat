package scheduler_agent

import (
	"strings"
	"testing"
)

func TestDescribeBooking(t *testing.T) {
	b := PendingBooking{
		Type:   "Sync",
		Date:   "2026-06-22",
		Start:  "10:00",
		End:    "10:30",
		Emails: []string{"mia@co.com", "alex@co.com"},
	}
	out := describeBooking(b, "ru")
	for _, want := range []string{"Sync", "2026-06-22", "10:00", "10:30", "mia@co.com", "alex@co.com"} {
		if !strings.Contains(out, want) {
			t.Fatalf("describeBooking missing %q; got:\n%s", want, out)
		}
	}
}
