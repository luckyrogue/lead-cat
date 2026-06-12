package meeting

import (
	"sort"
	"time"
)

func Overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

func FreeSlots(busy []Span, winStart, winEnd time.Time, minDur time.Duration) []Span {
	clipped := make([]Span, 0, len(busy))
	for _, b := range busy {
		s, e := b.Start, b.End
		if s.Before(winStart) {
			s = winStart
		}
		if e.After(winEnd) {
			e = winEnd
		}
		if s.Before(e) {
			clipped = append(clipped, Span{Start: s, End: e})
		}
	}
	sort.Slice(clipped, func(i, j int) bool { return clipped[i].Start.Before(clipped[j].Start) })

	merged := make([]Span, 0, len(clipped))
	for _, b := range clipped {
		if n := len(merged); n > 0 && !b.Start.After(merged[n-1].End) {
			if b.End.After(merged[n-1].End) {
				merged[n-1].End = b.End
			}
			continue
		}
		merged = append(merged, b)
	}

	var free []Span
	cursor := winStart
	add := func(s, e time.Time) {
		if e.Sub(s) >= minDur {
			free = append(free, Span{Start: s, End: e})
		}
	}
	for _, b := range merged {
		if cursor.Before(b.Start) {
			add(cursor, b.Start)
		}
		if b.End.After(cursor) {
			cursor = b.End
		}
	}
	if cursor.Before(winEnd) {
		add(cursor, winEnd)
	}
	return free
}
