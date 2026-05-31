package meeting_notifier

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMessage(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 5, 31, 14, 0, 0, 0, loc)
	end := time.Date(2026, 5, 31, 15, 0, 0, 0, loc)

	m := buildMessage("Разработка | Планёрка", "https://meet.google.com/abc", start, end, loc)
	for _, want := range []string{"Новая встреча", "Разработка | Планёрка", "31.05.2026", "14:00–15:00", "https://meet.google.com/abc", "UTC+5"} {
		if !strings.Contains(m, want) {
			t.Fatalf("message %q missing %q", m, want)
		}
	}

	if strings.Contains(buildMessage("X", "", start, end, loc), "🔗") {
		t.Fatal("no link line when meet link empty")
	}

	// stored times are UTC; rendering must convert to Almaty (+5).
	startUTC := time.Date(2026, 5, 31, 9, 0, 0, 0, time.UTC) // 14:00 Almaty
	m2 := buildMessage("X", "", startUTC, startUTC.Add(time.Hour), loc)
	if !strings.Contains(m2, "14:00–15:00") {
		t.Fatalf("UTC not converted to Almaty: %q", m2)
	}
}

func TestBuildUpdatedMessage(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 5, 31, 14, 0, 0, 0, loc)
	end := time.Date(2026, 5, 31, 15, 0, 0, 0, loc)

	m := buildUpdatedMessage("Разработка | Планёрка", "https://meet.google.com/abc", start, end, loc)
	for _, want := range []string{"изменена", "Разработка | Планёрка", "31.05.2026", "14:00–15:00", "UTC+5", "https://meet.google.com/abc"} {
		if !strings.Contains(m, want) {
			t.Fatalf("message %q missing %q", m, want)
		}
	}
	if strings.Contains(buildUpdatedMessage("X", "", start, end, loc), "🔗") {
		t.Fatal("no link line when meet link empty")
	}
}

func TestBuildRemovedMessage(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2026, 5, 31, 14, 0, 0, 0, loc)
	m := buildRemovedMessage("Разработка | Планёрка", start, loc)
	for _, want := range []string{"удалили", "Разработка | Планёрка", "31.05.2026", "UTC+5"} {
		if !strings.Contains(m, want) {
			t.Fatalf("message %q missing %q", m, want)
		}
	}
	if strings.Contains(m, "🔗") {
		t.Fatal("removed message has no meet link")
	}
}

func TestTZLabel(t *testing.T) {
	almaty, _ := time.LoadLocation("Asia/Almaty")
	cases := map[*time.Location]string{
		almaty:   "UTC+5",
		time.UTC: "UTC+0",
	}
	for loc, want := range cases {
		got := tzLabel(time.Date(2026, 5, 31, 12, 0, 0, 0, loc))
		if got != want {
			t.Fatalf("tzLabel(%v)=%q want %q", loc, got, want)
		}
	}
}
