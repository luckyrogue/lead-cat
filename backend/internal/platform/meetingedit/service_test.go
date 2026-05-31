package meetingedit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

type fakeBackend struct {
	meetings []postgres.MeetingWithTZ
	gotIn    application.UpdateMeetingInput
	gotWS    uuid.UUID
	gotUser  uuid.UUID
	gotMID   uuid.UUID
	applied  postgres.Meeting
}

func (f *fakeBackend) ListEditableMeetings(_ context.Context, _ int64) ([]postgres.MeetingWithTZ, error) {
	return f.meetings, nil
}
func (f *fakeBackend) UpdateMeeting(_ context.Context, ws, user, mid uuid.UUID, in application.UpdateMeetingInput) (postgres.Meeting, error) {
	f.gotWS, f.gotUser, f.gotMID, f.gotIn = ws, user, mid, in
	return f.applied, nil
}

type memSessions struct{ m map[int64]*State }

func newMemSessions() *memSessions { return &memSessions{m: map[int64]*State{}} }
func (s *memSessions) Get(_ context.Context, tg int64) (*State, error) { return s.m[tg], nil }
func (s *memSessions) Set(_ context.Context, tg int64, st State) error { c := st; s.m[tg] = &c; return nil }
func (s *memSessions) Del(_ context.Context, tg int64) error          { delete(s.m, tg); return nil }

func sampleMeeting() postgres.MeetingWithTZ {
	loc, _ := time.LoadLocation("Asia/Almaty")
	org := uuid.New()
	return postgres.MeetingWithTZ{
		Meeting: postgres.Meeting{
			ID: uuid.New(), WorkspaceID: uuid.New(), OrganizerUserID: &org,
			Dept: "Разработка", Type: "Планёрка", Host: "Иванов",
			StartsAt:   time.Date(2026, 6, 1, 14, 0, 0, 0, loc).UTC(),
			EndsAt:     time.Date(2026, 6, 1, 15, 0, 0, 0, loc).UTC(),
			Recurrence: "once", Description: "d", Name: "Разработка | Планёрка | Иванов | 2026-06-01",
			MeetLink: "https://meet.google.com/x",
		},
		TZ: "Asia/Almaty",
	}
}

func TestEditFlow_TextField(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(42)

	start := svc.Start(ctx, tg)
	if len(start.Keyboard) != 1 || start.Keyboard[0][0].Data != "medit:pick:"+m.ID.String() {
		t.Fatalf("bad start keyboard: %+v", start.Keyboard)
	}
	if r, ok := svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String()); !ok || !strings.Contains(r.Text, "Редактирование") {
		t.Fatalf("pick reply: %+v ok=%v", r, ok)
	}
	if _, ok := svc.OnCallback(ctx, tg, "medit:field:dept"); !ok {
		t.Fatal("field:dept not handled")
	}
	if r, ok := svc.OnText(ctx, tg, "Маркетинг"); !ok || !strings.Contains(r.Text, "★") {
		t.Fatalf("ontext reply: %+v ok=%v", r, ok)
	}
	if _, ok := svc.OnCallback(ctx, tg, "medit:apply"); !ok {
		t.Fatal("apply not handled")
	}
	if be.gotIn.Dept == nil || *be.gotIn.Dept != "Маркетинг" {
		t.Fatalf("apply did not pass dept override: %+v", be.gotIn)
	}
	if be.gotWS != m.WorkspaceID || be.gotUser != *m.OrganizerUserID || be.gotMID != m.ID {
		t.Fatal("apply passed wrong ids")
	}
}

func TestEditFlow_NoChanges(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(7)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	r, ok := svc.OnCallback(ctx, tg, "medit:apply")
	if !ok || !strings.Contains(r.Text, "Нет изменений") {
		t.Fatalf("expected no-changes reply, got %+v", r)
	}
}

func TestEditFlow_DateTime(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(9)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:field:datetime")
	if r, ok := svc.OnText(ctx, tg, "bad"); !ok || !strings.Contains(r.Text, "формат") {
		t.Fatalf("expected format error, got %+v", r)
	}
	svc.OnText(ctx, tg, "2026-06-02 10:00-11:00")
	svc.OnCallback(ctx, tg, "medit:apply")
	if be.gotIn.Date == nil || *be.gotIn.Date != "2026-06-02" || be.gotIn.Start == nil || *be.gotIn.Start != "10:00" {
		t.Fatalf("datetime override not passed: %+v", be.gotIn)
	}
}
