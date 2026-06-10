package handlers

import (
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application"
)

func TestToCreateMeetingInput(t *testing.T) {

	in := toCreateMeetingInput(miniappCreateRequest{
		Dept: "Eng", Type: "weekly", Host: "", Date: "2026-06-10",
		Start: "10:00", End: "10:30", Recurrence: "once", Desc: "sync",
		Participants: []string{"a@x.io", "", "  ", "b@x.io"},
	}, "Real Name")
	if in.Host != "Real Name" {
		t.Fatalf("host fallback: %q", in.Host)
	}
	if len(in.Participants) != 2 || in.Participants[0].Email != "a@x.io" || in.Participants[1].Email != "b@x.io" {
		t.Fatalf("participants: %+v", in.Participants)
	}
	if in.Date != "2026-06-10" || in.Start != "10:00" || in.End != "10:30" || in.Recurrence != "once" || in.Description != "sync" {
		t.Fatalf("passthrough: %+v", in)
	}

	if got := toCreateMeetingInput(miniappCreateRequest{Host: "Custom"}, "Real").Host; got != "Custom" {
		t.Fatalf("host kept: %q", got)
	}
}

func TestToConflictDTO(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")

	c := application.Conflict{
		Email:       "a@x.io",
		PersonName:  "Alice",
		MeetingName: "Weekly",
		Start:       time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	}
	d := toConflictDTO(c, loc)
	if d.Email != "a@x.io" || d.Name != "Alice" || d.Title != "Weekly" || d.Start != "14:00" || d.End != "15:00" {
		t.Fatalf("toConflictDTO got %+v", d)
	}
}
