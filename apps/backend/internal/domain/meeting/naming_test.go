package meeting

import (
	"strings"
	"testing"
)

func TestGenerateName_WithFrequencyLabel(t *testing.T) {
	got := GenerateName("Eng", "Standup", "Mia", ts(2026, 6, 1, 9, 0), Daily)
	if !strings.HasPrefix(got, "Eng | Standup | Mia | 2026-06-01") {
		t.Fatalf("bad prefix: %q", got)
	}
	if !strings.HasSuffix(got, "| "+Daily.Label()) {
		t.Fatalf("missing frequency label: %q", got)
	}
}

func TestGenerateName_OnceHasNoFrequencySuffix(t *testing.T) {
	got := GenerateName("Eng", "Sync", "Mia", ts(2026, 6, 1, 9, 0), Once)
	if got != "Eng | Sync | Mia | 2026-06-01" {
		t.Fatalf("unexpected: %q", got)
	}
}
