package reminder_scheduler

import (
	"fmt"
	"strconv"
	"time"
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

func offsetLabel(min int) string {
	switch min {
	case 10:
		return "10 минут"
	case 15:
		return "15 минут"
	case 30:
		return "30 минут"
	case 60:
		return "1 час"
	case 120:
		return "2 часа"
	case 1440:
		return "1 день"
	default:
		return strconv.Itoa(min) + " минут"
	}
}

func message(name, meetLink string, offset int) string {
	m := fmt.Sprintf("⏰ Напоминание: встреча через %s!\n«%s»", offsetLabel(offset), name)
	if meetLink != "" {
		m += "\n🔗 " + meetLink
	}
	return m
}
