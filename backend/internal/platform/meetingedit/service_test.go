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
	meetings  []postgres.MeetingWithTZ
	updateErr error
	gotIn     application.UpdateMeetingInput
	gotWS     uuid.UUID
	gotUser   uuid.UUID
	gotMID    uuid.UUID
	applied   postgres.Meeting
}

func (f *fakeBackend) ListEditableMeetings(_ context.Context, _ int64) ([]postgres.MeetingWithTZ, error) {
	return f.meetings, nil
}
func (f *fakeBackend) UpdateMeeting(_ context.Context, ws, user, mid uuid.UUID, in application.UpdateMeetingInput) (postgres.Meeting, error) {
	if f.updateErr != nil {
		return postgres.Meeting{}, f.updateErr
	}
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

func TestEditFlow_Recurrence(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(11)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	if r, ok := svc.OnCallback(ctx, tg, "medit:field:rec"); !ok || !strings.Contains(r.Text, "частоту") {
		t.Fatalf("rec menu: %+v", r)
	}
	svc.OnCallback(ctx, tg, "medit:set:rec:weekly")
	svc.OnCallback(ctx, tg, "medit:apply")
	if be.gotIn.Recurrence == nil || *be.gotIn.Recurrence != "weekly" {
		t.Fatalf("recurrence not passed: %+v", be.gotIn)
	}
}

func TestEditFlow_Cancel(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	sess := newMemSessions()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, sess)
	const tg = int64(12)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	r, ok := svc.OnCallback(ctx, tg, "medit:cancel")
	if !ok || !r.Edit || !strings.Contains(r.Text, "отменено") {
		t.Fatalf("cancel reply: %+v", r)
	}
	if st, _ := sess.Get(ctx, tg); st != nil {
		t.Fatal("session not cleared on cancel")
	}
}

func TestEditFlow_SessionExpired(t *testing.T) {
	ctx := context.Background()
	svc := New(&fakeBackend{}, newMemSessions())
	const tg = int64(13)
	if r, ok := svc.OnCallback(ctx, tg, "medit:field:dept"); !ok || !strings.Contains(r.Text, "истекла") {
		t.Fatalf("expected session-expired on field, got %+v ok=%v", r, ok)
	}
	if r, ok := svc.OnCallback(ctx, tg, "medit:apply"); !ok || !strings.Contains(r.Text, "истекла") {
		t.Fatalf("expected session-expired on apply, got %+v", r)
	}
}

func TestEditFlow_ApplyErrors(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()

	// ErrForbidden -> "Нет доступа", session cleared.
	sess := newMemSessions()
	svc := New(&fakeBackend{meetings: []postgres.MeetingWithTZ{m}, updateErr: application.ErrForbidden}, sess)
	const tg = int64(14)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:field:dept")
	svc.OnText(ctx, tg, "Маркетинг")
	if r, _ := svc.OnCallback(ctx, tg, "medit:apply"); !strings.Contains(r.Text, "Нет доступа") {
		t.Fatalf("forbidden mapping: %+v", r)
	}
	if st, _ := sess.Get(ctx, tg); st != nil {
		t.Fatal("session should be cleared on forbidden")
	}

	// ErrInvalidInput -> "Неверные данные", session kept.
	sess2 := newMemSessions()
	svc2 := New(&fakeBackend{meetings: []postgres.MeetingWithTZ{m}, updateErr: application.ErrInvalidInput}, sess2)
	const tg2 = int64(15)
	svc2.OnCallback(ctx, tg2, "medit:pick:"+m.ID.String())
	svc2.OnCallback(ctx, tg2, "medit:field:dept")
	svc2.OnText(ctx, tg2, "Маркетинг")
	if r, _ := svc2.OnCallback(ctx, tg2, "medit:apply"); !strings.Contains(r.Text, "Неверные данные") {
		t.Fatalf("invalid mapping: %+v", r)
	}
	if st, _ := sess2.Get(ctx, tg2); st == nil {
		t.Fatal("session should persist on invalid input")
	}
}
