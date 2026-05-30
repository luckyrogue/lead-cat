// Package botsettings renders and updates a bot user's reminder intervals.
// The pure helpers (parse/format/toggle/render) have no IO and are unit-tested.
package botsettings

import (
	"slices"
	"strconv"
	"strings"
)

// Button is a transport-agnostic inline-keyboard button (mapped to Telegram
// in the delivery layer).
type Button struct {
	Text string
	Data string
}

// Interval is a selectable reminder offset.
type Interval struct {
	Minutes int
	Label   string
}

// Intervals are the offered reminder offsets (ТЗ §5.2).
var Intervals = []Interval{
	{10, "10м"}, {15, "15м"}, {30, "30м"}, {60, "1ч"}, {120, "2ч"}, {1440, "1день"},
}

// Parse exposes the reminder-minutes CSV parser for other packages.
func Parse(csv string) []int { return parse(csv) }

func parse(csv string) []int {
	var out []int
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out
}

func format(mins []int) string {
	cp := append([]int(nil), mins...)
	slices.Sort(cp)
	cp = slices.Compact(cp)
	parts := make([]string, 0, len(cp))
	for _, m := range cp {
		parts = append(parts, strconv.Itoa(m))
	}
	return strings.Join(parts, ",")
}

func toggle(cur []int, v int) []int {
	if i := slices.Index(cur, v); i >= 0 {
		return slices.Delete(append([]int(nil), cur...), i, i+1)
	}
	return append(append([]int(nil), cur...), v)
}

func render(mins []int) (string, [][]Button) {
	text := "⏰ Напоминания о встречах. Выбери, за сколько предупреждать (можно несколько):"
	if len(mins) == 0 {
		text += "\nСейчас напоминания выключены."
	}
	var rows [][]Button
	var row []Button
	for _, iv := range Intervals {
		label := iv.Label
		if slices.Contains(mins, iv.Minutes) {
			label = "✓ " + label
		}
		row = append(row, Button{Text: label, Data: "rem:" + strconv.Itoa(iv.Minutes)})
		if len(row) == 3 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	return text, rows
}
