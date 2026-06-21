package reminder_scheduler

import (
	"strings"
	"testing"
)

func TestOffsetLabel_Localized(t *testing.T) {
	if got := offsetLabel(60, "ru"); got != "1 час" {
		t.Errorf("ru 60 = %q", got)
	}
	if got := offsetLabel(60, "en"); got != "1 hour" {
		t.Errorf("en 60 = %q", got)
	}
	if got := offsetLabel(1440, "kk"); got != "1 күн" {
		t.Errorf("kk 1440 = %q", got)
	}
	if got := offsetLabel(7, "en"); got != "7 min" {
		t.Errorf("en 7 = %q", got)
	}
}

func TestMessage_Localized(t *testing.T) {
	en := message("Sync", "https://meet.google.com/abc", 60, "en")
	if !strings.Contains(en, "in 1 hour") || !strings.Contains(en, "«Sync»") || !strings.Contains(en, "🔗 https://meet.google.com/abc") {
		t.Fatalf("en message wrong:\n%s", en)
	}
	ru := message("Sync", "", 30, "ru")
	if !strings.Contains(ru, "через 30 минут") || strings.Contains(ru, "🔗") {
		t.Fatalf("ru message wrong:\n%s", ru)
	}
}
