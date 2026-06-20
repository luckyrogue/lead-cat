package reminder_scheduler

import (
	"fmt"
	"time"

	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

func dueOffsets(now, startsAt time.Time, offsets []int) []int {
	var due []int
	for _, off := range offsets {
		threshold := startsAt.Add(-time.Duration(off) * time.Minute)
		if !now.Before(threshold) && now.Before(startsAt) {
			due = append(due, off)
		}
	}
	return due
}

func offsetLabel(min int, lang string) string {
	switch min {
	case 10:
		return boti18n.T(lang, "reminder.offset.10m")
	case 15:
		return boti18n.T(lang, "reminder.offset.15m")
	case 30:
		return boti18n.T(lang, "reminder.offset.30m")
	case 60:
		return boti18n.T(lang, "reminder.offset.1h")
	case 120:
		return boti18n.T(lang, "reminder.offset.2h")
	case 1440:
		return boti18n.T(lang, "reminder.offset.1d")
	default:
		return boti18n.T(lang, "reminder.offset.n_min", min)
	}
}

func message(name, meetLink string, offset int, lang string) string {
	m := fmt.Sprintf("%s\n«%s»", boti18n.T(lang, "reminder.telegram", offsetLabel(offset, lang)), name)
	if meetLink != "" {
		m += "\n🔗 " + meetLink
	}
	return m
}
