package meeting_notifier

import (
	"strings"
	"testing"
	"time"
)

func almaty(t *testing.T) *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Fatalf("load loc: %v", err)
	}
	return loc
}

func TestBuildMessage_WithAndWithoutLink(t *testing.T) {
	loc := almaty(t)
	start := time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC) // 10:00 Almaty
	end := time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC)  // 10:30
	with := buildMessage("ru", "Sync", "https://meet.google.com/abc", start, end, loc)
	for _, want := range []string{"📅 Новая встреча", "«Sync»", "01.06.2026", "10:00–10:30", "UTC+5", "🔗 https://meet.google.com/abc"} {
		if !strings.Contains(with, want) {
			t.Fatalf("missing %q in:\n%s", want, with)
		}
	}
	without := buildMessage("ru", "Sync", "", start, end, loc)
	if strings.Contains(without, "🔗") {
		t.Fatalf("no link icon expected without meet link:\n%s", without)
	}
}

func TestBuildUpdatedRemovedCancelled(t *testing.T) {
	loc := almaty(t)
	start := time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC)
	if !strings.Contains(buildUpdatedMessage("ru", "S", "", start, end, loc), "✏️ Встреча изменена") {
		t.Fatal("updated header")
	}
	if !strings.Contains(buildRemovedMessage("ru", "S", start, loc), "➖") {
		t.Fatal("removed header")
	}
	if !strings.Contains(buildCancelledMessage("ru", "S", start, loc), "❌ Встреча отменена") {
		t.Fatal("cancelled header")
	}
}

func TestBuildMessage_Localized(t *testing.T) {
	loc := almaty(t)
	start := time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC)
	en := buildMessage("en", "Sync", "", start, end, loc)
	if !strings.Contains(en, "📅 New meeting") || strings.Contains(en, "Новая встреча") {
		t.Fatalf("expected English header, got:\n%s", en)
	}
	if !strings.Contains(en, "«Sync»") || !strings.Contains(en, "01.06.2026") {
		t.Fatalf("neutral content missing:\n%s", en)
	}
}

func TestTzLabel(t *testing.T) {
	whole := time.Date(2026, 6, 1, 0, 0, 0, 0, time.FixedZone("x", 5*3600))
	if got := tzLabel(whole); got != "UTC+5" {
		t.Fatalf("tzLabel whole = %q", got)
	}
	half := time.Date(2026, 6, 1, 0, 0, 0, 0, time.FixedZone("x", 5*3600+1800))
	if got := tzLabel(half); got != "UTC+5:30" {
		t.Fatalf("tzLabel half = %q", got)
	}
	neg := time.Date(2026, 6, 1, 0, 0, 0, 0, time.FixedZone("x", -4*3600))
	if got := tzLabel(neg); got != "UTC-4" {
		t.Fatalf("tzLabel neg = %q", got)
	}
}
