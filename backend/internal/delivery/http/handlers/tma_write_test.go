package handlers

import "testing"

func TestToCreateMeetingInput(t *testing.T) {
	// Host falls back to the bot user's full name when empty; blank participant
	// emails are dropped; recurrence/desc pass through.
	in := toCreateMeetingInput(tmaCreateRequest{
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
	// Non-empty host is kept.
	if got := toCreateMeetingInput(tmaCreateRequest{Host: "Custom"}, "Real").Host; got != "Custom" {
		t.Fatalf("host kept: %q", got)
	}
}
