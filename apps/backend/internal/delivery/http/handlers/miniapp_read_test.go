package handlers

import (
	"testing"
	"time"
)

func TestSplitMeetingTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	// 2026-06-01 14:00–15:00 Almaty == 09:00–10:00 UTC
	s := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	e := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	date, start, end := splitMeetingTime(s, e, loc)
	if date != "2026-06-01" || start != "14:00" || end != "15:00" {
		t.Fatalf("got %q %q %q", date, start, end)
	}
}

func TestTmaScopeWindow(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	if f, to, ok := miniappScopeWindow("upcoming", now); !ok || !f.Equal(now) || to.Before(now.AddDate(0, 0, 364)) {
		t.Fatalf("upcoming: %v %v %v", f, to, ok)
	}
	if f, to, ok := miniappScopeWindow("past", now); !ok || !to.Equal(now) || f.After(now.AddDate(0, 0, -364)) {
		t.Fatalf("past: %v %v %v", f, to, ok)
	}
	if f, to, ok := miniappScopeWindow("all", now); !ok || !f.Before(now) || !to.After(now) {
		t.Fatalf("all: %v %v %v", f, to, ok)
	}
	if _, _, ok := miniappScopeWindow("bogus", now); ok {
		t.Fatal("bogus scope should not be ok")
	}
}
