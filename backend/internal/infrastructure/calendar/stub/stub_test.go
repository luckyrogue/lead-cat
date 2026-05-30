package stub

import (
	"context"
	"strings"
	"testing"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
)

func TestStubCreateEvent(t *testing.T) {
	res, err := New().CreateEvent(context.Background(), application.CalendarEvent{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.EventID == "" || !strings.HasPrefix(res.MeetLink, "https://meet.google.com/") {
		t.Fatalf("bad result: %+v", res)
	}
}
