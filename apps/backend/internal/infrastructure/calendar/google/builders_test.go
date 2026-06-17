package google

import (
	"testing"
	"time"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

func TestBuildEvent(t *testing.T) {
	e := docalendar.CalendarEvent{
		Title: "T", Description: "D",
		Start:          time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		End:            time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC),
		AttendeeEmails: []string{"a@x.io", "b@x.io"},
	}
	ev := buildEvent(e, "req-123")
	if ev.Summary != "T" || ev.Description != "D" {
		t.Fatalf("summary/desc: %+v", ev)
	}
	if ev.Start.DateTime != "2026-06-01T10:00:00Z" || ev.End.DateTime != "2026-06-01T11:00:00Z" {
		t.Fatalf("times: %q %q", ev.Start.DateTime, ev.End.DateTime)
	}
	if len(ev.Attendees) != 2 || ev.Attendees[0].Email != "a@x.io" {
		t.Fatalf("attendees: %+v", ev.Attendees)
	}
	if ev.ConferenceData == nil || ev.ConferenceData.CreateRequest == nil ||
		ev.ConferenceData.CreateRequest.RequestId != "req-123" ||
		ev.ConferenceData.CreateRequest.ConferenceSolutionKey.Type != "hangoutsMeet" {
		t.Fatalf("conference data: %+v", ev.ConferenceData)
	}
}

func TestBuildPatch_NoAttendeesOrConference(t *testing.T) {
	ev := buildPatch(docalendar.CalendarEvent{
		Title: "T", Description: "D",
		Start:          time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		End:            time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC),
		AttendeeEmails: []string{"ignored@x.io"},
	})
	if ev.Summary != "T" || ev.Start.DateTime == "" || ev.End.DateTime == "" {
		t.Fatalf("patch core fields: %+v", ev)
	}
	if len(ev.Attendees) != 0 || ev.ConferenceData != nil {
		t.Fatalf("patch must not carry attendees/conference: %+v", ev)
	}
}

func TestAttendeeList(t *testing.T) {
	if attendeeList(nil) != nil {
		t.Fatal("empty -> nil")
	}
	got := attendeeList([]string{"a@x.io", "b@x.io"})
	if len(got) != 2 || got[1].Email != "b@x.io" {
		t.Fatalf("attendeeList: %+v", got)
	}
}
