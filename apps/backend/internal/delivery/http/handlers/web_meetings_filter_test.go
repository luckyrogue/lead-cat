package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseMeetingFilter(t *testing.T) {
	org := uuid.New()

	t.Run("empty is zero filter", func(t *testing.T) {
		f, err := parseMeetingFilter("", "", "", "", "")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if f.Status != "" || f.From != nil || f.To != nil || f.Dept != "" || f.Organizer != nil {
			t.Fatalf("filter not zero: %+v", f)
		}
	})

	t.Run("all maps to no status", func(t *testing.T) {
		f, err := parseMeetingFilter("all", "", "", "", "")
		if err != nil || f.Status != "" {
			t.Fatalf("f=%+v err=%v", f, err)
		}
	})

	t.Run("invalid status errors", func(t *testing.T) {
		if _, err := parseMeetingFilter("bogus", "", "", "", ""); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("dates parse; to is exclusive next day", func(t *testing.T) {
		f, err := parseMeetingFilter("scheduled", "2026-06-01", "2026-06-30", "Eng", org.String())
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if f.Status != "scheduled" || f.Dept != "Eng" {
			t.Fatalf("f=%+v", f)
		}
		wantFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		wantTo := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		if f.From == nil || !f.From.Equal(wantFrom) || f.To == nil || !f.To.Equal(wantTo) {
			t.Fatalf("from=%v to=%v", f.From, f.To)
		}
		if f.Organizer == nil || *f.Organizer != org {
			t.Fatalf("organizer=%v", f.Organizer)
		}
	})

	t.Run("bad date errors", func(t *testing.T) {
		if _, err := parseMeetingFilter("", "nope", "", "", ""); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("bad organizer errors", func(t *testing.T) {
		if _, err := parseMeetingFilter("", "", "", "", "not-a-uuid"); err == nil {
			t.Fatal("expected error")
		}
	})
}
