package meeting_notifier

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type sentMsg struct {
	ChatID int64
	Text   string
}

type fakeSender struct {
	sent []sentMsg
	err  error
}

func (f *fakeSender) SendMessage(_ context.Context, p *bot.SendMessageParams) (*models.Message, error) {
	cid, _ := p.ChatID.(int64)
	f.sent = append(f.sent, sentMsg{ChatID: cid, Text: p.Text})
	return &models.Message{}, f.err
}

var _ sender = (*fakeSender)(nil)

type fakeStore struct {
	meeting      postgres.Meeting
	getMeetErr   error
	org          postgres.Organization
	participants []postgres.MeetingParticipant
	byEmail      map[string]postgres.BotUser
	byTID        map[int64]postgres.BotUser
	orgTID       int64
	orgLinked    bool
	claimed      map[int64]bool
	claimErr     error
}

func (f *fakeStore) GetMeeting(_ context.Context, _, _ uuid.UUID) (postgres.Meeting, error) {
	return f.meeting, f.getMeetErr
}
func (f *fakeStore) GetOrganization(_ context.Context, _ uuid.UUID) (postgres.Organization, error) {
	return f.org, nil
}
func (f *fakeStore) ListParticipants(_ context.Context, _ uuid.UUID) ([]postgres.MeetingParticipant, error) {
	return f.participants, nil
}
func (f *fakeStore) GetBotUserByEmail(_ context.Context, email string) (postgres.BotUser, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return postgres.BotUser{}, sql.ErrNoRows
}
func (f *fakeStore) GetBotUserByTelegramID(_ context.Context, tid int64) (postgres.BotUser, error) {
	if u, ok := f.byTID[tid]; ok {
		return u, nil
	}
	return postgres.BotUser{}, sql.ErrNoRows
}
func (f *fakeStore) GetUserTelegramID(_ context.Context, _ uuid.UUID) (int64, bool, error) {
	return f.orgTID, f.orgLinked, nil
}
func (f *fakeStore) TryClaimReminder(_ context.Context, _ uuid.UUID, tid int64, _ int) (bool, error) {
	if f.claimErr != nil {
		return false, f.claimErr
	}
	return f.claimed[tid], nil
}

var _ store = (*fakeStore)(nil)

func baseStore() *fakeStore {
	org := uuid.New()
	return &fakeStore{
		meeting: postgres.Meeting{
			ID: uuid.New(), Name: "Sync", MeetLink: "https://meet.google.com/x",
			StartsAt:        time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC),
			EndsAt:          time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC),
			OrganizerUserID: &org,
		},
		org:          postgres.Organization{TZ: "Asia/Almaty"},
		participants: []postgres.MeetingParticipant{{Email: "p@x.io"}},
		byEmail:      map[string]postgres.BotUser{"p@x.io": {TelegramID: 600, Email: "p@x.io"}},
		byTID:        map[int64]postgres.BotUser{500: {TelegramID: 500, Email: "org@x.io"}},
		orgTID:       500,
		orgLinked:    true,
	}
}

func newNotifier(st store, snd sender) *Notifier { return New(st, snd, zap.NewNop()) }

func TestHandleCreated_SendsToClaimedOnly(t *testing.T) {
	fs := baseStore()
	fs.claimed = map[int64]bool{600: true, 500: false}
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleCreated(context.Background(), uuid.New(), fs.meeting.ID); err != nil {
		t.Fatalf("created: %v", err)
	}
	if len(snd.sent) != 1 || snd.sent[0].ChatID != 600 {
		t.Fatalf("want exactly one send to 600, got %+v", snd.sent)
	}
	if !strings.Contains(snd.sent[0].Text, "📅 Новая встреча") || !strings.Contains(snd.sent[0].Text, "Sync") {
		t.Fatalf("unexpected text: %q", snd.sent[0].Text)
	}
}

func TestHandleCreated_ClaimError_Aborts(t *testing.T) {
	fs := baseStore()
	fs.claimErr = errors.New("redis down")
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleCreated(context.Background(), uuid.New(), fs.meeting.ID); err == nil {
		t.Fatal("expected claim error to propagate")
	}
}

func TestHandleParticipantAdded_RoutesToUser(t *testing.T) {
	fs := baseStore()
	fs.byEmail["new@x.io"] = postgres.BotUser{TelegramID: 700, Email: "new@x.io"}
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleParticipantAdded(context.Background(), uuid.New(), fs.meeting.ID, "new@x.io"); err != nil {
		t.Fatalf("added: %v", err)
	}
	if len(snd.sent) != 1 || snd.sent[0].ChatID != 700 {
		t.Fatalf("want one send to 700, got %+v", snd.sent)
	}
	if !strings.Contains(snd.sent[0].Text, "➕ Вас добавили на встречу") {
		t.Fatalf("unexpected added text: %q", snd.sent[0].Text)
	}
}

func TestHandleParticipantRemoved_RoutesToUser(t *testing.T) {
	fs := baseStore()
	fs.byEmail["gone@x.io"] = postgres.BotUser{TelegramID: 800, Email: "gone@x.io"}
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleParticipantRemoved(context.Background(), uuid.New(), fs.meeting.ID, "gone@x.io"); err != nil {
		t.Fatalf("removed: %v", err)
	}
	if len(snd.sent) != 1 || snd.sent[0].ChatID != 800 || !strings.Contains(snd.sent[0].Text, "➖") {
		t.Fatalf("unexpected removed send: %+v", snd.sent)
	}
}

func TestHandleParticipant_NotFound_NoSend(t *testing.T) {
	fs := baseStore()
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleParticipantAdded(context.Background(), uuid.New(), fs.meeting.ID, "missing@x.io"); err != nil {
		t.Fatalf("want nil on not-found, got %v", err)
	}
	if len(snd.sent) != 0 {
		t.Fatalf("want zero sends, got %+v", snd.sent)
	}
}

func TestHandleCancelled_AllRecipients(t *testing.T) {
	fs := baseStore()
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleCancelled(context.Background(), uuid.New(), fs.meeting.ID); err != nil {
		t.Fatalf("cancelled: %v", err)
	}
	if len(snd.sent) != 2 {
		t.Fatalf("want 2 recipients, got %d: %+v", len(snd.sent), snd.sent)
	}
	for _, m := range snd.sent {
		if !strings.Contains(m.Text, "❌ Встреча отменена") {
			t.Fatalf("unexpected cancelled text: %q", m.Text)
		}
	}
}

func TestHandleUpdated_AllRecipients(t *testing.T) {
	fs := baseStore()
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleUpdated(context.Background(), uuid.New(), fs.meeting.ID); err != nil {
		t.Fatalf("updated: %v", err)
	}
	if len(snd.sent) != 2 {
		t.Fatalf("want 2 recipients, got %d", len(snd.sent))
	}
	if !strings.Contains(snd.sent[0].Text, "✏️ Встреча изменена") {
		t.Fatalf("unexpected updated text: %q", snd.sent[0].Text)
	}
}

func TestHandleUpdated_TZFallback(t *testing.T) {
	fs := baseStore()
	fs.org = postgres.Organization{TZ: ""}
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleUpdated(context.Background(), uuid.New(), fs.meeting.ID); err != nil {
		t.Fatalf("tz fallback should not error: %v", err)
	}
	if len(snd.sent) != 2 {
		t.Fatalf("want 2 sends despite blank tz, got %d", len(snd.sent))
	}
}
