package meetingrecipients

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type fakeStore struct {
	parts     []postgres.MeetingParticipant
	partsErr  error
	byEmail   map[string]postgres.BotUser
	orgTG     int64
	orgLinked bool
}

func (f *fakeStore) ListParticipants(_ context.Context, _ uuid.UUID) ([]postgres.MeetingParticipant, error) {
	if f.partsErr != nil {
		return nil, f.partsErr
	}
	return f.parts, nil
}
func (f *fakeStore) GetBotUserByEmail(_ context.Context, email string) (postgres.BotUser, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return postgres.BotUser{}, errors.New("not found")
	}
	return u, nil
}
func (f *fakeStore) GetUserTelegramID(_ context.Context, _ uuid.UUID) (int64, bool, error) {
	return f.orgTG, f.orgLinked, nil
}

func TestResolve(t *testing.T) {
	org := uuid.New()

	f := &fakeStore{
		parts: []postgres.MeetingParticipant{{Email: "a@x"}, {Email: "b@x"}, {Email: ""}},
		byEmail: map[string]postgres.BotUser{
			"a@x": {TelegramID: 111, ReminderMinutes: "30"},
		},
		orgTG:     999,
		orgLinked: true,
	}
	m := postgres.Meeting{ID: uuid.New(), OrganizerUserID: &org}

	got, err := Resolve(context.Background(), f, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 recipients, got %d: %+v", len(got), got)
	}
	if got[0].TelegramID != 111 || got[0].ReminderMinutes != "30" || got[0].IsOrganizer {
		t.Fatalf("bad participant recipient: %+v", got[0])
	}
	if got[1].TelegramID != 999 || !got[1].IsOrganizer {
		t.Fatalf("bad organizer recipient: %+v", got[1])
	}
}

func TestResolveOrganizerAlsoParticipant(t *testing.T) {
	org := uuid.New()
	f := &fakeStore{
		parts:   []postgres.MeetingParticipant{{Email: "a@x"}},
		byEmail: map[string]postgres.BotUser{"a@x": {TelegramID: 999, ReminderMinutes: "15"}},
		orgTG:   999, orgLinked: true,
	}
	m := postgres.Meeting{ID: uuid.New(), OrganizerUserID: &org}

	got, err := Resolve(context.Background(), f, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("organizer who is a participant must not duplicate; got %d: %+v", len(got), got)
	}
	if got[0].IsOrganizer {
		t.Fatalf("existing participant entry should win (not organizer): %+v", got[0])
	}
}

func TestResolveListParticipantsError(t *testing.T) {
	f := &fakeStore{partsErr: errors.New("db down")}
	got, err := Resolve(context.Background(), f, postgres.Meeting{ID: uuid.New()})
	if err == nil {
		t.Fatal("want error when ListParticipants fails")
	}
	if got != nil {
		t.Fatalf("want nil recipients on error, got %+v", got)
	}
}
