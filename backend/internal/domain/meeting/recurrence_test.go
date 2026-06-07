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
	spans, err := Occurrences(d(2026, 6, 1), d(2026, 6, 1).Add(time.Hour), Once, nil, time.Time{})
	if err != nil || len(spans) != 1 {
		t.Fatalf("once: spans=%d err=%v", len(spans), err)
	}
}

func TestOccurrences_Weekly(t *testing.T) {
	start := d(2026, 6, 1)
	spans, err := Occurrences(start, start.Add(time.Hour), Weekly, nil, d(2026, 6, 22))
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
	spans, _ := Occurrences(start, start.Add(time.Hour), Daily, nil, d(2026, 6, 3))
	if len(spans) != 3 {
		t.Fatalf("daily want 3, got %d", len(spans))
	}
}

func TestOccurrences_Monthly(t *testing.T) {
	start := d(2026, 1, 15)
	spans, _ := Occurrences(start, start.Add(time.Hour), Monthly, nil, d(2026, 4, 15))
	if len(spans) != 4 {
		t.Fatalf("monthly want 4, got %d", len(spans))
	}
}

func TestOccurrences_UntilEqualsStart(t *testing.T) {
	start := d(2026, 6, 1)
	spans, err := Occurrences(start, start.Add(time.Hour), Weekly, nil, d(2026, 6, 1))
	if err != nil || len(spans) != 1 {
		t.Fatalf("until==start: spans=%d err=%v", len(spans), err)
	}
}

func TestOccurrences_Errors(t *testing.T) {
	start := d(2026, 6, 1)
	if _, err := Occurrences(start, start.Add(time.Hour), Weekly, nil, time.Time{}); err == nil {
		t.Fatal("recurring without until must error")
	}
	if _, err := Occurrences(start, start.Add(time.Hour), Weekly, nil, d(2026, 5, 1)); err == nil {
		t.Fatal("until before start must error")
	}
	if _, err := Occurrences(start, start.Add(time.Hour), Daily, nil, d(2030, 1, 1)); !errors.Is(err, ErrTooManyOccurrences) {
		t.Fatalf("want ErrTooManyOccurrences, got %v", err)
	}
}

func TestOccurrences_Custom(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, loc) // Mon
	end := time.Date(2026, 6, 1, 10, 0, 0, 0, loc)
	until := time.Date(2026, 6, 21, 0, 0, 0, 0, loc)
	got, err := Occurrences(start, end, Custom, []int{1, 3, 5}, until)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 9 {
		t.Fatalf("expected 9 occurrences (3 wks × 3 days), got %d", len(got))
	}
	wantDates := []string{
		"2026-06-01", "2026-06-03", "2026-06-05",
		"2026-06-08", "2026-06-10", "2026-06-12",
		"2026-06-15", "2026-06-17", "2026-06-19",
	}
	for i, sp := range got {
		if sp.Start.Format("2006-01-02") != wantDates[i] {
			t.Errorf("occurrence %d: want %s, got %s", i, wantDates[i], sp.Start.Format("2006-01-02"))
		}
	}
}

func TestOccurrences_CustomEmptyDays(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	end := time.Date(2026, 6, 1, 10, 0, 0, 0, loc)
	until := time.Date(2026, 6, 7, 0, 0, 0, 0, loc)
	_, err := Occurrences(start, end, Custom, nil, until)
	if !errors.Is(err, ErrRecurrenceDays) {
		t.Fatalf("want ErrRecurrenceDays, got %v", err)
	}
}
