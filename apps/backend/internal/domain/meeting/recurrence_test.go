package meeting

import (
	"errors"
	"testing"
	"time"
)

func ts(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, time.UTC)
}

func TestOccurrences_Once_SingleSpan(t *testing.T) {
	start, end := ts(2026, 6, 1, 10, 0), ts(2026, 6, 1, 11, 0)
	got, err := Occurrences(start, end, Once, nil, time.Time{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 || !got[0].Start.Equal(start) || !got[0].End.Equal(end) {
		t.Fatalf("got %+v", got)
	}
}

func TestOccurrences_Daily_InclusiveCount(t *testing.T) {
	start, end := ts(2026, 6, 1, 10, 0), ts(2026, 6, 1, 10, 30)
	until := ts(2026, 6, 5, 0, 0)
	got, err := Occurrences(start, end, Daily, nil, until)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("want 5 occurrences, got %d", len(got))
	}
	if got[0].End.Sub(got[0].Start) != 30*time.Minute {
		t.Fatalf("duration not preserved: %v", got[0].End.Sub(got[0].Start))
	}
}

func TestOccurrences_Weekly_StepsBySevenDays(t *testing.T) {
	start, end := ts(2026, 6, 1, 9, 0), ts(2026, 6, 1, 9, 30)
	until := ts(2026, 6, 22, 0, 0)
	got, err := Occurrences(start, end, Weekly, nil, until)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("want 4, got %d", len(got))
	}
	if got[1].Start.Sub(got[0].Start) != 7*24*time.Hour {
		t.Fatalf("not weekly: %v", got[1].Start.Sub(got[0].Start))
	}
}

func TestOccurrences_Custom_OnlyAllowedWeekdays(t *testing.T) {
	start, end := ts(2026, 6, 1, 8, 0), ts(2026, 6, 1, 8, 30)
	until := ts(2026, 6, 14, 0, 0)
	days := []int{1, 3}
	got, err := Occurrences(start, end, Custom, days, until)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected some occurrences")
	}
	allowed := map[int]bool{1: true, 3: true}
	for _, s := range got {
		if !allowed[isoWeekday(s.Start)] {
			t.Fatalf("occurrence on disallowed weekday: %v (iso %d)", s.Start, isoWeekday(s.Start))
		}
	}
}

func TestOccurrences_Custom_NoDays_Errors(t *testing.T) {
	_, err := Occurrences(ts(2026, 6, 1, 8, 0), ts(2026, 6, 1, 9, 0), Custom, nil, ts(2026, 6, 30, 0, 0))
	if !errors.Is(err, ErrRecurrenceDays) {
		t.Fatalf("want ErrRecurrenceDays, got %v", err)
	}
}

func TestOccurrences_MissingUntil_Errors(t *testing.T) {
	_, err := Occurrences(ts(2026, 6, 1, 8, 0), ts(2026, 6, 1, 9, 0), Daily, nil, time.Time{})
	if !errors.Is(err, ErrRecurrenceWindow) {
		t.Fatalf("want ErrRecurrenceWindow, got %v", err)
	}
}

func TestOccurrences_UntilBeforeStart_Errors(t *testing.T) {
	_, err := Occurrences(ts(2026, 6, 10, 8, 0), ts(2026, 6, 10, 9, 0), Daily, nil, ts(2026, 6, 1, 0, 0))
	if !errors.Is(err, ErrRecurrenceWindow) {
		t.Fatalf("want ErrRecurrenceWindow, got %v", err)
	}
}

func TestOccurrences_TooMany_Errors(t *testing.T) {
	start, end := ts(2026, 1, 1, 8, 0), ts(2026, 1, 1, 9, 0)
	until := start.AddDate(0, 0, 150)
	_, err := Occurrences(start, end, Daily, nil, until)
	if !errors.Is(err, ErrTooManyOccurrences) {
		t.Fatalf("want ErrTooManyOccurrences, got %v", err)
	}
}

func TestIsoWeekday_SundayIsSeven(t *testing.T) {
	sunday := ts(2026, 6, 7, 0, 0)
	if got := isoWeekday(sunday); got != 7 {
		t.Fatalf("want 7, got %d", got)
	}
}
