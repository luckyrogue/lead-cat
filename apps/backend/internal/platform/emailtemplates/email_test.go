package emailtemplates

import (
	"html"
	"strings"
	"testing"
)

func TestRenderReminder_SubjectAndLocalized(t *testing.T) {
	subject, text, out, err := RenderReminder(ReminderData{
		Language: "ru", Name: "Mia", Title: "Sync",
		Date: "16.06.2026", Time: "10:00–10:30", Tz: "UTC+5",
		MeetLink: "https://meet.google.com/abc", UnsubscribeURL: "https://app/profile",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(subject, "Напоминание") {
		t.Fatalf("subject = %q", subject)
	}
	decoded := html.UnescapeString(out)
	for _, want := range []string{"Привет, Mia", "«Sync»", "16.06.2026", "UTC+5", "Подключиться к Google Meet", "https://meet.google.com/abc"} {
		if !strings.Contains(decoded, want) {
			t.Fatalf("missing %q in html", want)
		}
	}
	for _, want := range []string{"Привет, Mia", "«Sync»", "https://meet.google.com/abc"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in text", want)
		}
	}
}

func TestRenderReminder_OutlookButton(t *testing.T) {
	_, _, out, err := RenderReminder(ReminderData{
		Language: "en", Name: "Mia", Title: "Sync", Date: "x", Time: "y",
		MeetLink: "https://meet.google.com/abc",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"[if mso]", "v:roundrect", "[if !mso]", "https://meet.google.com/abc"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestRenderReminder_NoLinkNoButton(t *testing.T) {
	_, _, out, err := RenderReminder(ReminderData{Language: "en", Name: "Mia", Title: "Sync", Date: "16.06.2026", Time: "10:00"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "Join Google Meet") {
		t.Fatal("button should be absent without meet link")
	}
}

func TestRenderReminder_EscapesUserContent(t *testing.T) {
	_, _, out, err := RenderReminder(ReminderData{Language: "ru", Title: `<script>alert(1)</script>`, Date: "x", Time: "y"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Fatal("title must be escaped")
	}
}

func TestRenderMagicLink_Ru(t *testing.T) {
	subject, text, out, err := RenderMagicLink(MagicLinkData{
		Language: "ru", SignInURL: "https://app.example/verify?token=abc", ExpiresMinutes: 15, FirstName: "Айгуль",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if subject != "Войти в Lead Cat" {
		t.Fatalf("subject = %q", subject)
	}
	decoded := html.UnescapeString(out)
	for _, want := range []string{"Привет, Айгуль", "Войти в Lead Cat", "https://app.example/verify?token=abc", "15 минут"} {
		if !strings.Contains(decoded, want) && !strings.Contains(text, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestRenderOrgInvite_En(t *testing.T) {
	subject, text, out, err := RenderOrgInvite(InviteData{
		Language: "en", OrgName: "Acme Corp", RoleLabel: "Admin",
		LoginURL: "https://app.example/login", InviterName: "Alice",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if subject != "Invitation to Acme Corp" {
		t.Fatalf("subject = %q", subject)
	}
	for _, want := range []string{"Alice invited you", "Acme Corp", "Admin", "https://app.example/login"} {
		if !strings.Contains(out, want) && !strings.Contains(text, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestRoleLabelLocal(t *testing.T) {
	if got := RoleLabelLocal("ru", "admin"); got != "Администратор" {
		t.Fatalf("got %q", got)
	}
	if got := RoleLabelLocal("en", "member"); got != "Member" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderWelcome_Kk(t *testing.T) {
	subject, text, out, err := RenderWelcome(WelcomeData{
		Language: "kk", FirstName: "Асан", AppURL: "https://t.me/bot/app", UnsubscribeURL: "https://app/profile",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(subject, "қош келдің") {
		t.Fatalf("subject = %q", subject)
	}
	for _, want := range []string{"Сәлем, Асан", "https://t.me/bot/app", "https://app/profile"} {
		if !strings.Contains(out, want) && !strings.Contains(text, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestFirstNameFromDisplay(t *testing.T) {
	if got := FirstNameFromDisplay("Alice Smith", ""); got != "Alice" {
		t.Fatalf("got %q", got)
	}
	if got := FirstNameFromDisplay("", "bob.jones@example.com"); got != "bob" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeLang(t *testing.T) {
	if NormalizeLang("") != "ru" || NormalizeLang("fr") != "ru" {
		t.Fatal("default ru")
	}
	if NormalizeLang("en") != "en" || NormalizeLang("kk") != "kk" {
		t.Fatal("en/kk")
	}
}
