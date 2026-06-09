package google

import (
	"testing"
	"time"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

func TestBuildPatch(t *testing.T) {
	e := docalendar.CalendarEvent{
		Title:          "Разработка | Планёрка",
		Description:    "desc",
		Start:          time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC),
		End:            time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC),
		AttendeeEmails: []string{"a@x"},
	}
	ev := buildPatch(e)
	if ev.Summary != "Разработка | Планёрка" || ev.Description != "desc" {
		t.Fatalf("summary/description not set: %+v", ev)
	}
	if ev.Start == nil || ev.End == nil {
		t.Fatal("start/end must be set")
	}
	if ev.Attendees != nil {
		t.Fatalf("patch must not set attendees, got %+v", ev.Attendees)
	}
	if ev.ConferenceData != nil {
		t.Fatal("patch must not set conference data")
	}
}

func TestAttendeeList(t *testing.T) {
	got := attendeeList([]string{"a@x", "b@y"})
	if len(got) != 2 || got[0].Email != "a@x" || got[1].Email != "b@y" {
		t.Fatalf("bad attendees: %+v", got)
	}
	if attendeeList(nil) != nil {
		t.Fatal("empty input must yield nil slice")
	}
}

func TestBuildEvent(t *testing.T) {
	start := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)
	ev := buildEvent(docalendar.CalendarEvent{
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
