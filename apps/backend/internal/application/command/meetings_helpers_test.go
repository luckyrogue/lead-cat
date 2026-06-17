package command

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func curMeeting() model.Meeting {
	return model.Meeting{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		StartsAt:   time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC),
		EndsAt:     time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC),
		Recurrence: "once",
	}
}

func TestApplyMeetingUpdate_PartialField(t *testing.T) {
	loc := time.UTC
	out, err := ApplyMeetingUpdate(curMeeting(), UpdateInput{Dept: strPtr("Sales")}, loc)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if out.Dept != "Sales" || out.Type != "Sync" || out.Host != "Mia" {
		t.Fatalf("partial update wrong: %+v", out)
	}
	if out.Name == "" || out.Name == curMeeting().Name {
		t.Fatalf("name should be recomputed, got %q", out.Name)
	}
}

func TestApplyMeetingUpdate_DateMustComeTogether(t *testing.T) {
	_, err := ApplyMeetingUpdate(curMeeting(), UpdateInput{Date: strPtr("2026-06-02")}, time.UTC)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput when only Date supplied, got %v", err)
	}
}

func TestApplyMeetingUpdate_AllDateApplied(t *testing.T) {
	out, err := ApplyMeetingUpdate(curMeeting(), UpdateInput{
		Date: strPtr("2026-06-02"), Start: strPtr("09:00"), End: strPtr("09:45"),
	}, time.UTC)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !out.StartsAt.Equal(time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("start not applied: %v", out.StartsAt)
	}
	if !out.EndsAt.Equal(time.Date(2026, 6, 2, 9, 45, 0, 0, time.UTC)) {
		t.Fatalf("end not applied: %v", out.EndsAt)
	}
}

func TestApplyMeetingUpdate_EndBeforeStart(t *testing.T) {
	_, err := ApplyMeetingUpdate(curMeeting(), UpdateInput{
		Date: strPtr("2026-06-02"), Start: strPtr("10:00"), End: strPtr("09:00"),
	}, time.UTC)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for end<=start, got %v", err)
	}
}

func TestOwnerOrOrganizer(t *testing.T) {
	owner := uuid.New()
	organizer := uuid.New()
	stranger := uuid.New()
	org := model.Organization{OwnerUserID: &owner}

	if !ownerOrOrganizer(org, &organizer, owner) {
		t.Fatal("owner should be allowed")
	}
	if !ownerOrOrganizer(org, &organizer, organizer) {
		t.Fatal("organizer should be allowed")
	}
	if ownerOrOrganizer(org, &organizer, stranger) {
		t.Fatal("stranger should be denied")
	}
	if ownerOrOrganizer(model.Organization{}, nil, stranger) {
		t.Fatal("nil owner + nil organizer should deny")
	}
}
