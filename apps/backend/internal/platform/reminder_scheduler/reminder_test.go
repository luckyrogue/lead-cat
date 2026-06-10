package reminder_scheduler

import (
	"strings"
	"testing"
	"time"
)

func TestDueOffsets(t *testing.T) {
	start := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)

	now := start.Add(-16 * time.Minute)
	got := dueOffsets(now, start, []int{10, 15, 30})
	if len(got) != 1 || got[0] != 30 {
		t.Fatalf("at -16m want [30], got %v", got)
	}

	now = start.Add(-9 * time.Minute)
	got = dueOffsets(now, start, []int{10, 15, 30})
	if len(got) != 3 {
		t.Fatalf("at -9m want 3 offsets, got %v", got)
	}

	now = start.Add(-40 * time.Minute)
	if got := dueOffsets(now, start, []int{10, 15, 30}); len(got) != 0 {
		t.Fatalf("at -40m want none, got %v", got)
	}

	now = start.Add(1 * time.Minute)
	if got := dueOffsets(now, start, []int{10, 15, 30}); len(got) != 0 {
		t.Fatalf("after start want none, got %v", got)
	}
}

func TestOffsetLabel(t *testing.T) {
	cases := map[int]string{10: "10 минут", 60: "1 час", 120: "2 часа", 1440: "1 день", 45: "45 минут"}
	for min, want := range cases {
		if got := offsetLabel(min); got != want {
			t.Fatalf("offsetLabel(%d)=%q want %q", min, got, want)
		}
	}
}

func TestMessage(t *testing.T) {
	m := message("Разработка | Планёрка", "https://meet.google.com/abc", 15)
	if !strings.Contains(m, "через 15 минут") || !strings.Contains(m, "Разработка | Планёрка") || !strings.Contains(m, "https://meet.google.com/abc") {
		t.Fatalf("bad message: %q", m)
	}
	if strings.Contains(message("X", "", 15), "🔗") {
		t.Fatal("no link line when meet link empty")
	}
}
