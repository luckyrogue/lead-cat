// Package scheduleview drives the read-only /schedule bot flow (§4.6).
package scheduleview

import (
	"fmt"
	"strings"
	"time"
)

// parseDate parses "YYYY-MM-DD" as a start-of-day time in loc.
func parseDate(s string, loc *time.Location) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(s), loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("неверная дата (ГГГГ-ММ-ДД)")
	}
	return d, nil
}

// parseRange parses "YYYY-MM-DD..YYYY-MM-DD" into [from, to) where to is the day
// AFTER the end date (exclusive). End must not precede start.
func parseRange(s string, loc *time.Location) (from, to time.Time, err error) {
	parts := strings.SplitN(strings.TrimSpace(s), "..", 2)
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("формат: ГГГГ-ММ-ДД..ГГГГ-ММ-ДД")
	}
	d1, err := parseDate(parts[0], loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	d2, err := parseDate(parts[1], loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if d2.Before(d1) {
		return time.Time{}, time.Time{}, fmt.Errorf("конец раньше начала")
	}
	return d1, d2.AddDate(0, 0, 1), nil
}

// dayWindow returns the [from,to) window for a preset kind. ok is false for an
// unknown kind.
func dayWindow(now time.Time, kind string, loc *time.Location) (from, to time.Time, ok bool) {
	base := now.In(loc)
	sod := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, loc)
	switch kind {
	case "today":
		return sod, sod.AddDate(0, 0, 1), true
	case "tomorrow":
		return sod.AddDate(0, 0, 1), sod.AddDate(0, 0, 2), true
	case "upcoming":
		return now, now.AddDate(1, 0, 0), true
	}
	return time.Time{}, time.Time{}, false
}

// statusEmoji marks an upcoming (🔜) vs past (✅) meeting relative to now.
func statusEmoji(startsAt, now time.Time) string {
	if startsAt.After(now) {
		return "🔜"
	}
	return "✅"
}
