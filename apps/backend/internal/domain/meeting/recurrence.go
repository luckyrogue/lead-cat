package meeting

import (
	"errors"
	"time"
)

// Span is one occurrence's start/end.
type Span struct {
	Start time.Time
	End   time.Time
}

const maxOccurrences = 100

// ErrTooManyOccurrences means the series would exceed the materialization cap.
var ErrTooManyOccurrences = errors.New("too many occurrences (max 100)")

// ErrRecurrenceWindow means a recurring series has a missing or invalid end date.
var ErrRecurrenceWindow = errors.New("recurring meeting needs a valid end date (>= start)")

// ErrRecurrenceDays means a Custom recurrence was requested without any selected weekdays.
var ErrRecurrenceDays = errors.New("custom recurrence needs at least one weekday")

// Occurrences expands a recurring meeting into spans from start to until
// (inclusive by date). Once returns a single span and ignores until/days.
// Custom steps daily and emits only when weekday ∈ days (1=Mon..7=Sun).
// Non-once requires a valid until (date >= start's date); capped at maxOccurrences.
func Occurrences(start, end time.Time, r Recurrence, days []int, until time.Time) ([]Span, error) {
	if r == Once {
		return []Span{{Start: start, End: end}}, nil
	}
	if r == Custom && len(days) == 0 {
		return nil, ErrRecurrenceDays
	}
	if until.IsZero() {
		return nil, ErrRecurrenceWindow
	}
	startDay := dateOnly(start)
	untilDay := dateOnly(until)
	if untilDay.Before(startDay) {
		return nil, ErrRecurrenceWindow
	}
	dayMask := make(map[int]bool, len(days))
	for _, d := range days {
		dayMask[d] = true
	}
	dur := end.Sub(start)
	var spans []Span
	for cur := start; !dateOnly(cur).After(untilDay); cur = nextStep(cur, r) {
		if len(spans) >= maxOccurrences {
			return nil, ErrTooManyOccurrences
		}
		if r == Custom {
			if !dayMask[isoWeekday(cur)] {
				continue
			}
		}
		spans = append(spans, Span{Start: cur, End: cur.Add(dur)})
	}
	return spans, nil
}

func nextStep(t time.Time, r Recurrence) time.Time {
	switch r {
	case Daily, Custom:
		return t.AddDate(0, 0, 1)
	case Weekly:
		return t.AddDate(0, 0, 7)
	case Monthly:
		// AddDate normalizes overflow (e.g. Jan 31 + 1mo → Mar 3), so a series
		// anchored past a short month's end shifts forward — accepted behavior.
		return t.AddDate(0, 1, 0)
	}
	return t.AddDate(0, 0, 1)
}

func isoWeekday(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
