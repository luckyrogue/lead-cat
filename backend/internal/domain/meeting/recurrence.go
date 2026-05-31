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

// Occurrences expands a recurring meeting into spans from start to until
// (inclusive by date). Once returns a single span and ignores until. Non-once
// requires a valid until (date >= start's date) and is capped at maxOccurrences.
func Occurrences(start, end time.Time, r Recurrence, until time.Time) ([]Span, error) {
	if r == Once {
		return []Span{{Start: start, End: end}}, nil
	}
	if until.IsZero() {
		return nil, ErrRecurrenceWindow
	}
	startDay := dateOnly(start)
	untilDay := dateOnly(until)
	if untilDay.Before(startDay) {
		return nil, ErrRecurrenceWindow
	}
	dur := end.Sub(start)
	var spans []Span
	for cur := start; !dateOnly(cur).After(untilDay); cur = next(cur, r) {
		if len(spans) >= maxOccurrences {
			return nil, ErrTooManyOccurrences
		}
		spans = append(spans, Span{Start: cur, End: cur.Add(dur)})
	}
	return spans, nil
}

func next(t time.Time, r Recurrence) time.Time {
	switch r {
	case Daily:
		return t.AddDate(0, 0, 1)
	case Weekly:
		return t.AddDate(0, 0, 7)
	case Biweekly:
		return t.AddDate(0, 0, 14)
	case Monthly:
		return t.AddDate(0, 1, 0)
	}
	return t.AddDate(0, 0, 1)
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
