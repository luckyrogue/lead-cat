package meeting_notifier

import (
	"fmt"
	"time"
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

func buildMessage(name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	return buildEventMessage("📅 Новая встреча", name, meetLink, startsAt, endsAt, loc)
}

func buildUpdatedMessage(name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	return buildEventMessage("✏️ Встреча изменена", name, meetLink, startsAt, endsAt, loc)
}

func buildRemovedMessage(name string, startsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	return fmt.Sprintf("➖ Вас удалили из встречи\n«%s»\n🗓 %s (%s)", name, s.Format("02.01.2006"), tzLabel(s))
}

func buildCancelledMessage(name string, startsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	return fmt.Sprintf("❌ Встреча отменена\n«%s»\n🗓 %s (%s)", name, s.Format("02.01.2006"), tzLabel(s))
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
