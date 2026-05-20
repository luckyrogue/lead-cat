package postgres

import "testing"

func TestNormalizeUsername(t *testing.T) {
	if got := normalizeUsername("@User"); got != "user" {
		t.Fatalf("got %q", got)
	}
}
