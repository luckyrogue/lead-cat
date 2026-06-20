package meeting_notifier

import (
	"fmt"
	"time"

	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

func buildEventMessage(header, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	e := endsAt.In(loc)
	msg := fmt.Sprintf("%s\n«%s»\n🗓 %s, %s–%s (%s)",
		header,
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

func buildMessage(lang, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	return buildEventMessage(boti18n.T(lang, "notif.created"), name, meetLink, startsAt, endsAt, loc)
}

func buildUpdatedMessage(lang, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	return buildEventMessage(boti18n.T(lang, "notif.updated"), name, meetLink, startsAt, endsAt, loc)
}

func buildRemovedMessage(lang, name string, startsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	return fmt.Sprintf("%s\n«%s»\n🗓 %s (%s)", boti18n.T(lang, "notif.removed"), name, s.Format("02.01.2006"), tzLabel(s))
}

func buildCancelledMessage(lang, name string, startsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	return fmt.Sprintf("%s\n«%s»\n🗓 %s (%s)", boti18n.T(lang, "notif.cancelled"), name, s.Format("02.01.2006"), tzLabel(s))
}

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
