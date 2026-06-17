package meeting

import (
	"testing"
	"time"
)

func TestOverlaps(t *testing.T) {
	a1, a2 := ts(2026, 6, 1, 10, 0), ts(2026, 6, 1, 11, 0)
	cases := []struct {
		name   string
		b1, b2 time.Time
		want   bool
	}{
		{"contained", ts(2026, 6, 1, 10, 15), ts(2026, 6, 1, 10, 45), true},
		{"partial-overlap", ts(2026, 6, 1, 10, 30), ts(2026, 6, 1, 11, 30), true},
		{"touching-edge", ts(2026, 6, 1, 11, 0), ts(2026, 6, 1, 12, 0), false},
		{"disjoint", ts(2026, 6, 1, 12, 0), ts(2026, 6, 1, 13, 0), false},
	}
	for _, c := range cases {
		if got := Overlaps(a1, a2, c.b1, c.b2); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestFreeSlots_FiltersShortGaps(t *testing.T) {
	win0, win1 := ts(2026, 6, 1, 9, 0), ts(2026, 6, 1, 17, 0)
	busy := []Span{
		{Start: ts(2026, 6, 1, 10, 0), End: ts(2026, 6, 1, 10, 20)},
		{Start: ts(2026, 6, 1, 10, 40), End: ts(2026, 6, 1, 12, 0)},
	}
	free := FreeSlots(busy, win0, win1, 30*time.Minute)
	for _, f := range free {
		if f.End.Sub(f.Start) < 30*time.Minute {
			t.Fatalf("slot shorter than minDur survived: %v", f)
		}
	}
	for _, f := range free {
		if f.Start.Equal(ts(2026, 6, 1, 10, 20)) {
			t.Fatal("short gap should have been filtered")
		}
	}
}

func TestFreeSlots_NoBusy_ReturnsWholeWindow(t *testing.T) {
	win0, win1 := ts(2026, 6, 1, 9, 0), ts(2026, 6, 1, 10, 0)
	free := FreeSlots(nil, win0, win1, 15*time.Minute)
	if len(free) != 1 || !free[0].Start.Equal(win0) || !free[0].End.Equal(win1) {
		t.Fatalf("got %+v", free)
	}
}
