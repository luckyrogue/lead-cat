package reminder_scheduler

import (
	"html"
	"strings"
	"testing"
)

func TestReminderEmailSubject(t *testing.T) {
	if got := reminderEmailSubject("en", "Sync", "10:00–10:30"); got != `Reminder: "Sync" at 10:00–10:30` {
		t.Fatalf("en subject = %q", got)
	}
	if got := reminderEmailSubject("ru", "Синк", "10:00"); !strings.Contains(got, "Напоминание") || !strings.Contains(got, "Синк") {
		t.Fatalf("ru subject = %q", got)
	}
	if got := reminderEmailSubject("kk", "Синк", "10:00"); !strings.Contains(got, "Еске салу") {
		t.Fatalf("kk subject = %q", got)
	}
}

func TestRenderReminderEmail_LocalizedAndConditional(t *testing.T) {
	out, err := renderReminderEmail(reminderEmailData{
		Lang: "ru", Name: "Mia", Title: "Sync",
		Date: "16.06.2026", Time: "10:00–10:30", Tz: "UTC+5",
		MeetLink: "https://meet.google.com/abc", UnsubscribeURL: "https://app/profile",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// html/template entity-escapes '+', '=', etc.; unescape so assertions read
	// the human-visible text rather than the wire form.
	decoded := html.UnescapeString(out)
	for _, want := range []string{"Привет, Mia", "«Sync»", "16.06.2026", "UTC+5", "Подключиться к Google Meet", "https://meet.google.com/abc"} {
		if !strings.Contains(decoded, want) {
			t.Fatalf("missing %q in render", want)
		}
	}
}

func TestRenderReminderEmail_NoLinkNoButton(t *testing.T) {
	out, err := renderReminderEmail(reminderEmailData{Lang: "en", Name: "Mia", Title: "Sync", Date: "16.06.2026", Time: "10:00"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "Join Google Meet") {
		t.Fatal("Join button should be absent without a meet link")
	}
}

func TestRenderReminderEmail_EscapesUserContent(t *testing.T) {
	out, err := renderReminderEmail(reminderEmailData{Lang: "ru", Title: `<script>alert(1)</script>`, Date: "x", Time: "y"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Fatal("user-supplied title must be HTML-escaped")
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatal("expected escaped title")
	}
}

func TestRenderReminderEmail_DefaultsName(t *testing.T) {
	out, _ := renderReminderEmail(reminderEmailData{Lang: "ru", Title: "Sync", Date: "x", Time: "y"})
	if !strings.Contains(out, "Привет, друг") {
		t.Fatal("expected default name fallback")
	}
}
