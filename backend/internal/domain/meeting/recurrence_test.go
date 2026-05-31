package meeting

import (
	"errors"
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 10, 0, 0, 0, time.UTC)
}

func TestOccurrences_Once(t *testing.T) {
	spans, err := Occurrences(d(2026, 6, 1), d(2026, 6, 1).Add(time.Hour), Once, time.Time{})
	if err != nil || len(spans) != 1 {
		t.Fatalf("once: spans=%d err=%v", len(spans), err)
	}
}

func TestOccurrences_Weekly(t *testing.T) {
	start := d(2026, 6, 1)
	spans, err := Occurrences(start, start.Add(time.Hour), Weekly, d(2026, 6, 22))
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 4 {
		t.Fatalf("weekly want 4, got %d", len(spans))
	}
	if !spans[1].Start.Equal(d(2026, 6, 8)) {
		t.Fatalf("2nd occurrence = %v", spans[1].Start)
	}
	if spans[0].End.Sub(spans[0].Start) != time.Hour {
		t.Fatalf("duration not preserved")
	}
}

func TestOccurrences_Daily(t *testing.T) {
	start := d(2026, 6, 1)
	spans, _ := Occurrences(start, start.Add(time.Hour), Daily, d(2026, 6, 3))
	if len(spans) != 3 {
		t.Fatalf("daily want 3, got %d", len(spans))
	}
}

func TestOccurrences_Monthly(t *testing.T) {
	start := d(2026, 1, 15)
	spans, _ := Occurrences(start, start.Add(time.Hour), Monthly, d(2026, 4, 15))
	if len(spans) != 4 {
		t.Fatalf("monthly want 4, got %d", len(spans))
	}
}

func TestOccurrences_UntilEqualsStart(t *testing.T) {
	start := d(2026, 6, 1)
	spans, err := Occurrences(start, start.Add(time.Hour), Weekly, d(2026, 6, 1))
	if err != nil || len(spans) != 1 {
		t.Fatalf("until==start: spans=%d err=%v", len(spans), err)
	}
}

func TestOccurrences_Errors(t *testing.T) {
	start := d(2026, 6, 1)
	if _, err := Occurrences(start, start.Add(time.Hour), Weekly, time.Time{}); err == nil {
		t.Fatal("recurring without until must error")
	}
	if _, err := Occurrences(start, start.Add(time.Hour), Weekly, d(2026, 5, 1)); err == nil {
		t.Fatal("until before start must error")
	}
	if _, err := Occurrences(start, start.Add(time.Hour), Daily, d(2030, 1, 1)); !errors.Is(err, ErrTooManyOccurrences) {
		t.Fatalf("want ErrTooManyOccurrences, got %v", err)
	}
}
