// Package meetingedit drives the Telegram bot FSM for editing a meeting's fields.
package meetingedit

import (
	"fmt"
	"strings"
	"time"
)

// parseTimeRange parses "HH:MM–HH:MM" (en dash or hyphen) into start/end, requiring
// strict HH:MM and end > start.
func parseTimeRange(text string) (start, end string, err error) {
	rng := strings.NewReplacer("–", "-", "—", "-").Replace(strings.TrimSpace(text))
	parts := strings.SplitN(rng, "-", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("формат: ЧЧ:ММ–ЧЧ:ММ")
	}
	a := strings.TrimSpace(parts[0])
	b := strings.TrimSpace(parts[1])
	st, err := time.Parse("15:04", a)
	if err != nil || len(a) != 5 {
		return "", "", fmt.Errorf("неверное время начала (ЧЧ:ММ)")
	}
	en, err := time.Parse("15:04", b)
	if err != nil || len(b) != 5 {
		return "", "", fmt.Errorf("неверное время конца (ЧЧ:ММ)")
	}
	if !en.After(st) {
		return "", "", fmt.Errorf("конец должен быть позже начала")
	}
	return st.Format("15:04"), en.Format("15:04"), nil
}

// parseDateTime parses "YYYY-MM-DD HH:MM–HH:MM" (en dash or hyphen) into the
// override strings. It validates the formats and that end is after start.
func parseDateTime(text string) (date, start, end string, err error) {
	fields := strings.Fields(text)
	if len(fields) != 2 {
		return "", "", "", fmt.Errorf("формат: ГГГГ-ММ-ДД ЧЧ:ММ–ЧЧ:ММ")
	}
	d, err := time.Parse("2006-01-02", fields[0])
	if err != nil {
		return "", "", "", fmt.Errorf("неверная дата (нужно ГГГГ-ММ-ДД)")
	}
	rng := strings.NewReplacer("–", "-", "—", "-").Replace(fields[1])
	parts := strings.SplitN(rng, "-", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("нужно время начала и конца: ЧЧ:ММ–ЧЧ:ММ")
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])
	if len(startStr) != 5 || len(endStr) != 5 {
		return "", "", "", fmt.Errorf("время должно быть в формате ЧЧ:ММ")
	}
	st, err := time.Parse("15:04", startStr)
	if err != nil {
		return "", "", "", fmt.Errorf("неверное время начала (ЧЧ:ММ)")
	}
	en, err := time.Parse("15:04", endStr)
	if err != nil {
		return "", "", "", fmt.Errorf("неверное время конца (ЧЧ:ММ)")
	}
	if !en.After(st) {
		return "", "", "", fmt.Errorf("конец должен быть позже начала")
	}
	return d.Format("2006-01-02"), st.Format("15:04"), en.Format("15:04"), nil
}
