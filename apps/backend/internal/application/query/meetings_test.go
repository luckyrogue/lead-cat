package query

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

var (
	_ meetingListApp = (*fakeListApp)(nil)
	_ meetingStore   = (*fakeReadStore)(nil)
)

type fakeListApp struct {
	meetings []model.Meeting
}

func (f *fakeListApp) EmployeeSchedule(_ context.Context, _ string, _, _ time.Time) ([]model.Meeting, error) {
	return f.meetings, nil
}

func TestSchedule_Delegates(t *testing.T) {
	want := []model.Meeting{{Dept: "Eng"}, {Dept: "Sales"}}
	m := NewMeetings(&fakeListApp{meetings: want})
	got, err := m.Schedule(context.Background(), "a@x.io", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if len(got) != 2 || got[0].Dept != "Eng" || got[1].Dept != "Sales" {
		t.Fatalf("delegation wrong: %+v", got)
	}
}

type fakeReadStore struct {
	user  model.User
	parts []model.MeetingParticipant
}

func (f *fakeReadStore) GetUserByID(_ context.Context, _ uuid.UUID) (model.User, error) {
	return f.user, nil
}

func (f *fakeReadStore) ListParticipants(_ context.Context, _ uuid.UUID) ([]model.MeetingParticipant, error) {
	return f.parts, nil
}

func TestMeetingDTO_FormatsInLocation(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Almaty") // UTC+5
	if err != nil {
		t.Fatalf("load loc: %v", err)
	}
	organizer := uuid.New()
	store := &fakeReadStore{
		user:  model.User{Email: "org@x.io"},
		parts: []model.MeetingParticipant{{Email: "a@x.io"}, {Email: ""}, {Email: "b@x.io"}},
	}
	m := model.Meeting{
		ID: uuid.New(), Type: "Sync", Dept: "Eng", Host: "Mia",
		StartsAt:        time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC), // 10:00 Almaty
		EndsAt:          time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC),
		Recurrence:      "once",
		Status:          "scheduled",
		OrganizerUserID: &organizer,
	}
	dto := MeetingDTO(context.Background(), store, m, loc)
	if dto.Date != "2026-06-01" || dto.Start != "10:00" || dto.End != "10:30" {
		t.Fatalf("local formatting wrong: %+v", dto)
	}
	if dto.Organizer != "org@x.io" {
		t.Fatalf("organizer email not resolved: %q", dto.Organizer)
	}
	if len(dto.Participants) != 2 || dto.Participants[0] != "a@x.io" || dto.Participants[1] != "b@x.io" {
		t.Fatalf("participants wrong (empty email should be dropped): %+v", dto.Participants)
	}
}
