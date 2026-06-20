package emailtemplates

import (
	"html"
	"strings"
	"testing"
)

func TestRenderBookingConfirmation_Localized(t *testing.T) {
	for _, tc := range []struct{ lang, subjPart string }{
		{"ru", "подтвержден"},
		{"en", "confirmed"},
		{"kk", "расталды"},
	} {
		subject, text, out, err := RenderBookingConfirmation(BookingConfirmationData{
			Language: tc.lang, BookerName: "Visitor V", EventTitle: "Intro Call",
			Date: "Mon, 22 Jun 2026", Time: "10:00 – 10:30", Tz: "Asia/Almaty",
			MeetLink: "https://meet.google.com/abc-defg-hij",
		})
		if err != nil {
			t.Fatalf("[%s] render: %v", tc.lang, err)
		}
		if !strings.Contains(strings.ToLower(subject), tc.subjPart) {
			t.Fatalf("[%s] subject = %q, want contains %q", tc.lang, subject, tc.subjPart)
		}
		decoded := html.UnescapeString(out)
		for _, want := range []string{"Intro Call", "Mon, 22 Jun 2026", "10:00 – 10:30", "Asia/Almaty", "https://meet.google.com/abc-defg-hij"} {
			if !strings.Contains(decoded, want) {
				t.Fatalf("[%s] html missing %q", tc.lang, want)
			}
		}
		if !strings.Contains(text, "https://meet.google.com/abc-defg-hij") {
			t.Fatalf("[%s] text missing meet link", tc.lang)
		}
	}
}

func TestRenderBookingConfirmation_DefaultsToRu(t *testing.T) {
	subject, _, _, err := RenderBookingConfirmation(BookingConfirmationData{Language: "fr", EventTitle: "X"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(strings.ToLower(subject), "подтвержден") {
		t.Fatalf("garbage lang should fall back to ru, subject = %q", subject)
	}
}

func TestRenderBookingHostNotification_Localized(t *testing.T) {
	subject, text, out, err := RenderBookingHostNotification(BookingHostNotificationData{
		Language: "en", EventTitle: "Intro Call",
		BookerName: "Visitor V", BookerEmail: "visitor@example.com",
		Date: "Mon, 22 Jun 2026", Time: "10:00 – 10:30", Tz: "Asia/Almaty",
		MeetLink: "https://meet.google.com/abc-defg-hij",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(strings.ToLower(subject), "booking") {
		t.Fatalf("subject = %q", subject)
	}
	decoded := html.UnescapeString(out)
	for _, want := range []string{"Visitor V", "visitor@example.com", "Intro Call", "10:00 – 10:30", "https://meet.google.com/abc-defg-hij"} {
		if !strings.Contains(decoded, want) {
			t.Fatalf("html missing %q", want)
		}
	}
	if !strings.Contains(text, "visitor@example.com") {
		t.Fatalf("text missing booker email")
	}
}

func TestRenderBooking_EscapesUserContent(t *testing.T) {
	_, _, out, err := RenderBookingHostNotification(BookingHostNotificationData{
		Language: "ru", EventTitle: `<script>alert(1)</script>`, BookerName: "x", BookerEmail: "e@e.com",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Fatal("event title must be HTML-escaped")
	}
}
