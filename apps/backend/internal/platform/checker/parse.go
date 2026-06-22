package checker

import (
	"fmt"
	"strings"
	"time"
)

func parseRange(s string, loc *time.Location) (from, to time.Time, err error) {
	parts := strings.SplitN(strings.TrimSpace(s), "..", 2)
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("формат: ГГГГ-ММ-ДД..ГГГГ-ММ-ДД")
	}
	d1, e1 := time.ParseInLocation("2006-01-02", strings.TrimSpace(parts[0]), loc)
	d2, e2 := time.ParseInLocation("2006-01-02", strings.TrimSpace(parts[1]), loc)
	if e1 != nil || e2 != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("неверная дата (ГГГГ-ММ-ДД)")
	}
	if d2.Before(d1) {
		return time.Time{}, time.Time{}, fmt.Errorf("конец раньше начала")
	}
	return d1, d2, nil
}

var ruWeekday = map[time.Weekday]string{
	time.Monday: "Пн", time.Tuesday: "Вт", time.Wednesday: "Ср",
	time.Thursday: "Чт", time.Friday: "Пт", time.Saturday: "Сб", time.Sunday: "Вс",
}

func dayLabel(day time.Time, loc *time.Location) string {
	d := day.In(loc)
	return fmt.Sprintf("%s, %02d.%02d", ruWeekday[d.Weekday()], d.Day(), int(d.Month()))
}
