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
	msg := fmt.Sprintf("📅 Новая встреча\n«%s»\n🗓 %s, %s–%s (Алматы)",
		name,
		s.Format("02.01.2006"),
		s.Format("15:04"),
		e.Format("15:04"))
	if meetLink != "" {
		msg += "\n🔗 " + meetLink
	}
	return msg
}
