package checker

import (
	"testing"
	"time"
)

func TestParseRange(t *testing.T) {
	loc := almaty()
	from, to, err := parseRange("2026-06-01..2026-06-03", loc)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if from.Day() != 1 || to.Day() != 3 {
		t.Fatalf("from=%v to=%v", from, to)
	}
	if _, _, err := parseRange("2026-06-03..2026-06-01", loc); err == nil {
		t.Fatal("expected error for reversed range")
	}
	if _, _, err := parseRange("bad", loc); err == nil {
		t.Fatal("expected format error")
	}
}

func TestDayLabel(t *testing.T) {
	loc := almaty()
	d := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	if got := dayLabel(d, loc); got != "Пн, 01.06" {
		t.Fatalf("got %q", got)
	}
}
