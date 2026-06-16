package meetingrecipients

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type fakeStore struct {
	parts      []postgres.MeetingParticipant
	byEmail    map[string]postgres.BotUser
	byTelegram map[int64]postgres.BotUser
	orgTg      int64
	orgLinked  bool
}

func (f *fakeStore) ListParticipants(context.Context, uuid.UUID) ([]postgres.MeetingParticipant, error) {
	return f.parts, nil
}
func (f *fakeStore) GetBotUserByEmail(_ context.Context, email string) (postgres.BotUser, error) {
	if bu, ok := f.byEmail[email]; ok {
		return bu, nil
	}
	return postgres.BotUser{}, errors.New("not found")
}
func (f *fakeStore) GetBotUserByTelegramID(_ context.Context, tg int64) (postgres.BotUser, error) {
	if bu, ok := f.byTelegram[tg]; ok {
		return bu, nil
	}
	return postgres.BotUser{}, errors.New("not found")
}
func (f *fakeStore) GetUserTelegramID(context.Context, uuid.UUID) (int64, bool, error) {
	return f.orgTg, f.orgLinked, nil
}

func TestResolve_PopulatesContactDetails(t *testing.T) {
	orgID := uuid.New()
	f := &fakeStore{
		parts: []postgres.MeetingParticipant{{Email: "mia@co.com"}},
		byEmail: map[string]postgres.BotUser{
			"mia@co.com": {TelegramID: 1, FullName: "Mia", Email: "mia@co.com", Language: "en", Timezone: "UTC", ReminderMinutes: "15"},
		},
		byTelegram: map[int64]postgres.BotUser{
			2: {TelegramID: 2, FullName: "Oli", Email: "oli@co.com", Language: "kk", Timezone: "Asia/Almaty"},
		},
		orgTg:     2,
		orgLinked: true,
	}
	m := postgres.Meeting{ID: uuid.New(), OrganizerUserID: &orgID}

	out, err := Resolve(context.Background(), f, m)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("recipients = %d, want 2", len(out))
	}

	var part, org *Recipient
	for i := range out {
		if out[i].IsOrganizer {
			org = &out[i]
		} else {
			part = &out[i]
		}
	}
	if part == nil || part.Email != "mia@co.com" || part.Language != "en" || part.Timezone != "UTC" || part.FullName != "Mia" {
		t.Fatalf("participant contact not populated: %+v", part)
	}
	if org == nil || org.Email != "oli@co.com" || org.Language != "kk" || org.Timezone != "Asia/Almaty" {
		t.Fatalf("organizer contact not populated from GetBotUserByTelegramID: %+v", org)
	}
}
