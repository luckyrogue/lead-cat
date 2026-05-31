package meeting

import (
	"testing"
	"time"
)

func tm(h, m int) time.Time { return time.Date(2026, 6, 1, h, m, 0, 0, time.UTC) }

func TestOverlaps(t *testing.T) {
	cases := []struct {
		name           string
		aS, aE, bS, bE time.Time
		want           bool
	}{
		{"disjoint before", tm(9, 0), tm(10, 0), tm(10, 0), tm(11, 0), false}, // touching edge = no overlap
		{"disjoint after", tm(11, 0), tm(12, 0), tm(9, 0), tm(10, 0), false},
		{"partial", tm(9, 0), tm(10, 0), tm(9, 30), tm(11, 0), true},
		{"full contain", tm(9, 0), tm(12, 0), tm(10, 0), tm(11, 0), true},
		{"identical", tm(9, 0), tm(10, 0), tm(9, 0), tm(10, 0), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Overlaps(c.aS, c.aE, c.bS, c.bE); got != c.want {
				t.Fatalf("Overlaps=%v want %v", got, c.want)
			}
		})
	}
}

func spansEqual(a, b []Span) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Start.Equal(b[i].Start) || !a[i].End.Equal(b[i].End) {
			return false
		}
	}
	return true
}

func TestFreeSlots(t *testing.T) {
	win, winEnd := tm(9, 0), tm(18, 0)
	min := 30 * time.Minute

	t.Run("no busy = whole window", func(t *testing.T) {
		if got := FreeSlots(nil, win, winEnd, min); !spansEqual(got, []Span{{win, winEnd}}) {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("busy spanning window = none", func(t *testing.T) {
		if got := FreeSlots([]Span{{tm(8, 0), tm(19, 0)}}, win, winEnd, min); len(got) != 0 {
			t.Fatalf("got %v want empty", got)
		}
	})
	t.Run("gaps around one meeting", func(t *testing.T) {
		got := FreeSlots([]Span{{tm(11, 0), tm(12, 30)}}, win, winEnd, min)
		want := []Span{{tm(9, 0), tm(11, 0)}, {tm(12, 30), tm(18, 0)}}
		if !spansEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})
	t.Run("merge overlapping busy", func(t *testing.T) {
		got := FreeSlots([]Span{{tm(11, 0), tm(12, 0)}, {tm(11, 30), tm(13, 0)}}, win, winEnd, min)
		want := []Span{{tm(9, 0), tm(11, 0)}, {tm(13, 0), tm(18, 0)}}
		if !spansEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})
	t.Run("min-duration filter drops short gaps", func(t *testing.T) {
		// leaves 9:00-9:15 and 17:45-18:00, both < 30m
		if got := FreeSlots([]Span{{tm(9, 15), tm(17, 45)}}, win, winEnd, min); len(got) != 0 {
			t.Fatalf("got %v want empty", got)
		}
	})
	t.Run("busy outside window clipped", func(t *testing.T) {
		got := FreeSlots([]Span{{tm(6, 0), tm(9, 30)}, {tm(18, 30), tm(20, 0)}}, win, winEnd, min)
		if !spansEqual(got, []Span{{tm(9, 30), tm(18, 0)}}) {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("unsorted busy input", func(t *testing.T) {
		got := FreeSlots([]Span{{tm(15, 0), tm(16, 0)}, {tm(10, 0), tm(11, 0)}}, win, winEnd, min)
		want := []Span{{tm(9, 0), tm(10, 0)}, {tm(11, 0), tm(15, 0)}, {tm(16, 0), tm(18, 0)}}
		if !spansEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})
}
