package google

import (
	"testing"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
)

func TestBuildEvent(t *testing.T) {
	start := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)
	ev := buildEvent(application.CalendarEvent{
		Title:          "Sync",
		Description:    "desc",
		Start:          start,
		End:            start.Add(time.Hour),
		AttendeeEmails: []string{"a@example.com", "b@example.com"},
	}, "req-123")

	if ev.Summary != "Sync" || ev.Description != "desc" {
		t.Fatalf("summary/desc wrong: %+v", ev)
	}
	if ev.Start.DateTime != "2025-06-02T10:00:00Z" || ev.End.DateTime != "2025-06-02T11:00:00Z" {
		t.Fatalf("times wrong: %s / %s", ev.Start.DateTime, ev.End.DateTime)
	}
	if len(ev.Attendees) != 2 || ev.Attendees[0].Email != "a@example.com" {
		t.Fatalf("attendees wrong: %+v", ev.Attendees)
	}
	if ev.ConferenceData == nil || ev.ConferenceData.CreateRequest == nil {
		t.Fatal("missing conference create request")
	}
	if ev.ConferenceData.CreateRequest.RequestId != "req-123" {
		t.Fatalf("request id wrong: %s", ev.ConferenceData.CreateRequest.RequestId)
	}
	if ev.ConferenceData.CreateRequest.ConferenceSolutionKey.Type != "hangoutsMeet" {
		t.Fatalf("solution key wrong: %+v", ev.ConferenceData.CreateRequest.ConferenceSolutionKey)
	}
}
