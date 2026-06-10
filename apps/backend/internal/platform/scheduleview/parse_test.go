package scheduleview

import (
	"testing"
	"time"
)

func almatyTZ(t *testing.T) *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestParseDate(t *testing.T) {
	loc := almatyTZ(t)
	d, err := parseDate("2026-06-02", loc)
	if err != nil {
		t.Fatal(err)
	}
	if d.Year() != 2026 || d.Month() != 6 || d.Day() != 2 || d.Hour() != 0 {
		t.Fatalf("bad date: %v", d)
	}
	if _, err := parseDate("2026/06/02", loc); err == nil {
		t.Fatal("expected error for bad format")
	}
}

func TestParseRange(t *testing.T) {
	loc := almatyTZ(t)
	from, to, err := parseRange("2026-06-01..2026-06-03", loc)
	if err != nil {
		t.Fatal(err)
	}
	if from.Day() != 1 || to.Day() != 4 { // to is exclusive end = 2026-06-04 00:00
		t.Fatalf("range from=%v to=%v", from, to)
	}
	if _, _, err := parseRange("2026-06-03..2026-06-01", loc); err == nil {
		t.Fatal("expected error: end before start")
	}
	if _, _, err := parseRange("2026-06-01", loc); err == nil {
		t.Fatal("expected error: missing range separator")
	}
}

func TestDayWindow(t *testing.T) {
	loc := almatyTZ(t)
	now := time.Date(2026, 6, 2, 13, 0, 0, 0, loc)
	from, to, ok := dayWindow(now, "today", loc)
	if !ok || from.Day() != 2 || from.Hour() != 0 || to.Day() != 3 {
		t.Fatalf("today: from=%v to=%v ok=%v", from, to, ok)
	}
	from, to, ok = dayWindow(now, "tomorrow", loc)
	if !ok || from.Day() != 3 || to.Day() != 4 {
		t.Fatalf("tomorrow: from=%v to=%v", from, to)
	}
	from, _, ok = dayWindow(now, "upcoming", loc)
	if !ok || !from.Equal(now) {
		t.Fatalf("upcoming from should equal now, got %v", from)
	}
	if _, _, ok := dayWindow(now, "bogus", loc); ok {
		t.Fatal("unknown kind must return ok=false")
	}
}

func TestStatusEmoji(t *testing.T) {
	now := time.Date(2026, 6, 2, 13, 0, 0, 0, time.UTC)
	if statusEmoji(now.Add(time.Hour), now) != "🔜" {
		t.Fatal("future should be 🔜")
	}
	if statusEmoji(now.Add(-time.Hour), now) != "✅" {
		t.Fatal("past should be ✅")
	}
}
