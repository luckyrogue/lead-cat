package meeting_notifier

import (
	"fmt"
	"time"
)

// buildMessage renders the creation DM. Times are converted to loc (workspace
// timezone, Almaty by default). The link line is omitted when meetLink is empty.
func buildMessage(name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	e := endsAt.In(loc)
	msg := fmt.Sprintf("📅 Новая встреча\n«%s»\n🗓 %s, %s–%s (%s)",
		name,
		s.Format("02.01.2006"),
		s.Format("15:04"),
		e.Format("15:04"),
		tzLabel(s))
	if meetLink != "" {
		msg += "\n🔗 " + meetLink
	}
	return msg
}

// tzLabel renders the timezone of t as a UTC offset, e.g. "UTC+5" or "UTC+5:30".
func tzLabel(t time.Time) string {
	_, off := t.Zone()
	sign := "+"
	if off < 0 {
		sign, off = "-", -off
	}
	h, m := off/3600, (off%3600)/60
	if m == 0 {
		return fmt.Sprintf("UTC%s%d", sign, h)
	}
	return fmt.Sprintf("UTC%s%d:%02d", sign, h, m)
}
