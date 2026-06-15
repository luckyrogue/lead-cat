package telegram

import (
	"strings"
	"testing"
)

func TestPublicCommands(t *testing.T) {
	cmds := PublicCommands()
	if len(cmds) == 0 {
		t.Fatal("no commands")
	}
	want := map[string]bool{
		"menu": true, "new": true, "schedule": true,
		"checker": true, "settings": true, "edit": true, "help": true,
	}
	for _, c := range cmds {
		if strings.HasPrefix(c.Command, "/") {
			t.Fatalf("command %q must not include leading slash", c.Command)
		}
		if c.Command != strings.ToLower(c.Command) || c.Command == "" {
			t.Fatalf("command %q must be lowercase and non-empty", c.Command)
		}
		if c.Description == "" {
			t.Fatalf("command %q missing description", c.Command)
		}
		delete(want, c.Command)
	}
	if len(want) != 0 {
		t.Fatalf("missing commands: %v", want)
	}
}

func TestJoinURL(t *testing.T) {
	cases := []struct{ base, path, want string }{
		{"https://app.example.com", "meetings/create", "https://app.example.com/meetings/create"},
		{"https://app.example.com/", "meetings/create", "https://app.example.com/meetings/create"},
		{"https://app.example.com/", "/meetings/create", "https://app.example.com/meetings/create"},
		{"https://app.example.com", "", "https://app.example.com"},
	}
	for _, c := range cases {
		if got := joinURL(c.base, c.path); got != c.want {
			t.Fatalf("joinURL(%q,%q) = %q, want %q", c.base, c.path, got, c.want)
		}
	}
}

func TestWebAppMarkup(t *testing.T) {
	m := webAppMarkup("Открыть", "https://app.example.com")
	if len(m.InlineKeyboard) != 1 || len(m.InlineKeyboard[0]) != 1 {
		t.Fatalf("expected 1x1 keyboard, got %v", m.InlineKeyboard)
	}
	btn := m.InlineKeyboard[0][0]
	if btn.Text != "Открыть" {
		t.Fatalf("text = %q", btn.Text)
	}
	if btn.WebApp == nil || btn.WebApp.URL != "https://app.example.com" {
		t.Fatalf("web app url = %v", btn.WebApp)
	}
}

func TestHelpText(t *testing.T) {
	h := helpText()
	for _, sub := range []string{"/menu", "/new", "/schedule", "/checker", "/settings", "/edit"} {
		if !strings.Contains(h, sub) {
			t.Fatalf("help text missing %q", sub)
		}
	}
}
