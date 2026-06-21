package botsettings

import (
	"slices"
	"strconv"
	"strings"

	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

type Button struct {
	Text string
	Data string
}

type Interval struct {
	Minutes int
}

var Intervals = []Interval{
	{10}, {15}, {30}, {60}, {120}, {1440},
}

func Parse(csv string) []int { return parse(csv) }

func Format(mins []int) string { return format(mins) }

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

func intervalLabelKey(min int) string {
	switch min {
	case 10:
		return "botset.iv.10m"
	case 15:
		return "botset.iv.15m"
	case 30:
		return "botset.iv.30m"
	case 60:
		return "botset.iv.1h"
	case 120:
		return "botset.iv.2h"
	case 1440:
		return "botset.iv.1d"
	}
	return "botset.iv.10m"
}

func render(mins []int, lang string) (string, [][]Button) {
	text := boti18n.T(lang, "botset.title")
	if len(mins) == 0 {
		text += "\n" + boti18n.T(lang, "botset.off")
	}
	var rows [][]Button
	var row []Button
	for _, iv := range Intervals {
		label := boti18n.T(lang, intervalLabelKey(iv.Minutes))
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
