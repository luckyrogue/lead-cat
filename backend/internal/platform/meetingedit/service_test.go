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
	meetings       []postgres.MeetingWithTZ
	updateErr      error
	gotIn          application.UpdateMeetingInput
	gotWS          uuid.UUID
	gotUser        uuid.UUID
	gotMID         uuid.UUID
	applied        postgres.Meeting
	participants   []postgres.MeetingParticipant
	employees      []postgres.Employee
	addErr         error
	addedEmail     string
	removedEmail   string
	updatedSeries  int
	seriesIn       application.SeriesUpdateInput
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
func (f *fakeBackend) ListParticipants(_ context.Context, _ uuid.UUID) ([]postgres.MeetingParticipant, error) {
	return f.participants, nil
}
func (f *fakeBackend) SearchEmployees(_ context.Context, _ uuid.UUID, query string) ([]postgres.Employee, error) {
	var out []postgres.Employee
	for _, e := range f.employees {
		if strings.Contains(strings.ToLower(e.FullName), strings.ToLower(query)) || strings.Contains(strings.ToLower(e.Email), strings.ToLower(query)) {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeBackend) AddParticipant(_ context.Context, _, _, _ uuid.UUID, email string) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.addedEmail = email
	f.participants = append(f.participants, postgres.MeetingParticipant{Email: email})
	return nil
}
func (f *fakeBackend) UpdateSeries(_ context.Context, _, _, _ uuid.UUID, in application.SeriesUpdateInput) (int, error) {
	f.seriesIn = in
	f.updatedSeries++
	return 3, nil
}
func (f *fakeBackend) RemoveParticipant(_ context.Context, _, _, _ uuid.UUID, email string) error {
	f.removedEmail = email
	var kept []postgres.MeetingParticipant
	for _, p := range f.participants {
		if p.Email != email {
			kept = append(kept, p)
		}
	}
	f.participants = kept
	return nil
}

type memSessions struct{ m map[int64]*State }

func newMemSessions() *memSessions                                     { return &memSessions{m: map[int64]*State{}} }
func (s *memSessions) Get(_ context.Context, tg int64) (*State, error) { return s.m[tg], nil }
func (s *memSessions) Set(_ context.Context, tg int64, st State) error {
	c := st
	s.m[tg] = &c
	return nil
}
func (s *memSessions) Del(_ context.Context, tg int64) error { delete(s.m, tg); return nil }

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

func TestEditFlow_NotEditable(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	sess := newMemSessions()
	svc := New(&fakeBackend{meetings: []postgres.MeetingWithTZ{m}, updateErr: postgres.ErrMeetingNotEditable}, sess)
	const tg = int64(16)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:field:dept")
	svc.OnText(ctx, tg, "Маркетинг")
	if r, _ := svc.OnCallback(ctx, tg, "medit:apply"); !strings.Contains(r.Text, "больше недоступна") {
		t.Fatalf("not-editable mapping: %+v", r)
	}
	if st, _ := sess.Get(ctx, tg); st != nil {
		t.Fatal("session should be cleared when meeting not editable")
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

	// ErrMeetingNotEditable -> "больше недоступна", session cleared.
	sess3 := newMemSessions()
	svc3 := New(&fakeBackend{meetings: []postgres.MeetingWithTZ{m}, updateErr: postgres.ErrMeetingNotEditable}, sess3)
	const tg3 = int64(16)
	svc3.OnCallback(ctx, tg3, "medit:pick:"+m.ID.String())
	svc3.OnCallback(ctx, tg3, "medit:field:dept")
	svc3.OnText(ctx, tg3, "Маркетинг")
	if r, _ := svc3.OnCallback(ctx, tg3, "medit:apply"); !strings.Contains(r.Text, "больше недоступна") {
		t.Fatalf("not-editable mapping: %+v", r)
	}
	if st, _ := sess3.Get(ctx, tg3); st != nil {
		t.Fatal("session should be cleared when meeting not editable")
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

func TestParticipants_AddBySearch(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{
		meetings:  []postgres.MeetingWithTZ{m},
		applied:   m.Meeting,
		employees: []postgres.Employee{{FullName: "Иван Иванов", Email: "ivan@corp.kz"}},
	}
	svc := New(be, newMemSessions())
	const tg = int64(50)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:parts")
	svc.OnCallback(ctx, tg, "medit:padd")
	if r, ok := svc.OnText(ctx, tg, "иван"); !ok || len(r.Keyboard) == 0 {
		t.Fatalf("search reply: %+v ok=%v", r, ok)
	}
	svc.OnCallback(ctx, tg, "medit:padd:0")
	if be.addedEmail != "ivan@corp.kz" {
		t.Fatalf("added email = %q", be.addedEmail)
	}
}

func TestParticipants_AddByRawEmail(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(51)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:padd")
	svc.OnText(ctx, tg, "new@corp.kz")
	svc.OnCallback(ctx, tg, "medit:padd:0")
	if be.addedEmail != "new@corp.kz" {
		t.Fatalf("added email = %q", be.addedEmail)
	}
}

func TestParticipants_AddDuplicate(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting, addErr: application.ErrInvalidInput}
	svc := New(be, newMemSessions())
	const tg = int64(52)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:padd")
	svc.OnText(ctx, tg, "dup@corp.kz")
	if r, _ := svc.OnCallback(ctx, tg, "medit:padd:0"); !strings.Contains(r.Text, "Уже участник") {
		t.Fatalf("duplicate reply: %+v", r)
	}
}

func TestParticipants_Remove(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{
		meetings:     []postgres.MeetingWithTZ{m},
		applied:      m.Meeting,
		participants: []postgres.MeetingParticipant{{Email: "bye@corp.kz"}},
	}
	svc := New(be, newMemSessions())
	const tg = int64(53)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:parts")
	if r, _ := svc.OnCallback(ctx, tg, "medit:prem:0"); !strings.Contains(r.Text, "Удалить участника bye@corp.kz") {
		t.Fatalf("confirm reply: %+v", r)
	}
	svc.OnCallback(ctx, tg, "medit:premc")
	if be.removedEmail != "bye@corp.kz" {
		t.Fatalf("removed email = %q", be.removedEmail)
	}
}

func TestParticipants_SessionExpired(t *testing.T) {
	ctx := context.Background()
	svc := New(&fakeBackend{}, newMemSessions())
	const tg = int64(60)
	for _, data := range []string{"medit:parts", "medit:padd", "medit:premc"} {
		if r, ok := svc.OnCallback(ctx, tg, data); !ok || !strings.Contains(r.Text, "истекла") {
			t.Fatalf("%s: expected session-expired, got %+v ok=%v", data, r, ok)
		}
	}
}

func TestParticipants_AddBadIndex(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(61)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:padd")
	// no search performed yet → PartCands empty → index 0 is out of bounds
	if r, _ := svc.OnCallback(ctx, tg, "medit:padd:0"); !strings.Contains(r.Text, "не найден") {
		t.Fatalf("expected not-found for bad index, got %+v", r)
	}
	if be.addedEmail != "" {
		t.Fatalf("nothing should have been added, got %q", be.addedEmail)
	}
}

func TestParticipants_SearchNoResults(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting} // no employees
	svc := New(be, newMemSessions())
	const tg = int64(62)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:padd")
	// a non-email query with no directory matches
	if r, ok := svc.OnText(ctx, tg, "zzz"); !ok || !strings.Contains(r.Text, "Ничего не найдено") {
		t.Fatalf("expected no-results, got %+v ok=%v", r, ok)
	}
}

func seriesMeeting() postgres.MeetingWithTZ {
	m := sampleMeeting()
	sid := uuid.New()
	m.SeriesID = &sid
	m.Recurrence = "weekly"
	return m
}

func TestEditFlow_SeriesScopePrompt(t *testing.T) {
	ctx := context.Background()
	m := seriesMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(80)
	r, ok := svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	if !ok || !strings.Contains(r.Text, "Что редактируем") {
		t.Fatalf("series pick should show scope prompt, got %+v", r)
	}
}

func TestEditFlow_SeriesEdit(t *testing.T) {
	ctx := context.Background()
	m := seriesMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(81)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	r, _ := svc.OnCallback(ctx, tg, "medit:scope:series")
	for _, row := range r.Keyboard {
		for _, b := range row {
			if b.Data == "medit:field:rec" {
				t.Fatal("series menu must not offer recurrence")
			}
		}
	}
	svc.OnCallback(ctx, tg, "medit:field:datetime")
	if r, ok := svc.OnText(ctx, tg, "10:00-11:00"); !ok || !strings.Contains(r.Text, "★") {
		t.Fatalf("time input: %+v ok=%v", r, ok)
	}
	svc.OnCallback(ctx, tg, "medit:apply")
	if be.updatedSeries != 1 || be.seriesIn.Start == nil || *be.seriesIn.Start != "10:00" {
		t.Fatalf("UpdateSeries not called with time: %+v (called=%d)", be.seriesIn, be.updatedSeries)
	}
}

func TestEditFlow_NonSeriesNoPrompt(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(82)
	r, _ := svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	if !strings.Contains(r.Text, "Редактирование встречи") {
		t.Fatalf("non-series pick should go straight to field menu, got %+v", r)
	}
}
